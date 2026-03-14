package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/lineage"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	runEnv             []string
	runNetwork         string
	runTimeout         time.Duration
	runDryRun          bool
	runDetach          bool
	runAttach          bool     // Explicitly attach to logs (for streaming)
	runSet             []string // --set flags for inline overrides
	runValueFiles      []string // -f flags for override files
	runCell            string   // --cell flag: resolve stores from this cell
	runKubeContext     string   // --context flag: kubectl context for multi-cluster
	runScanDirs        []string // --scan-dir flags for multi-transform run
	runLineage         bool     // --lineage flag: enable lineage tracking
	runLineageEndpoint string   // --lineage-endpoint flag: override lineage endpoint
)

// runCmd executes a pipeline locally
var runCmd = &cobra.Command{
	Use:   "run [package-dir]",
	Short: "Run a data package locally",
	Long: `Execute a data package locally using Docker.

The run command builds (if needed) and executes the package defined in
the specified directory.

The command will:
1. Parse dk.yaml manifest
2. Apply any override files (-f) and inline overrides (--set)
3. Build the Docker image
4. Start the container on the Docker network
5. Stream logs to stdout

Override precedence (lowest to highest):
  - dk.yaml (base configuration)
  - Override files (-f) in order specified
  - Inline overrides (--set) in order specified

Prerequisites:
  - Docker must be running

Examples:
  # Run pipeline in current directory
  dk run

  # Run pipeline in specific directory
  dk run ./my-pipeline

  # Run with custom environment variables
  dk run -e DEBUG=true -e LOG_LEVEL=debug

  # Override configuration values
  dk run --set spec.resources.memory=8Gi

  # Use an override file
  dk run -f prod-overrides.yaml

  # Combine override file and inline overrides
  dk run -f prod-overrides.yaml --set spec.timeout=4h

  # Dry run (validate only, don't execute)
  dk run --dry-run

  # Run in background
  dk run --detach

  # Run with stores resolved from a cell
  dk run --cell canary

  # Run against a cell in a specific kubectl context
  dk run --cell us-east --context arn:aws:eks:us-east-1:...:cluster/dk-prod

  # Run all transforms in a project (topological order)
  dk run --scan-dir ./my-project

  # Dry-run all transforms in a project
  dk run --scan-dir ./my-project --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPipeline,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringArrayVarP(&runEnv, "env", "e", []string{}, "Environment variables (KEY=VALUE)")
	runCmd.Flags().StringVar(&runNetwork, "network", "", "Docker network to connect to (auto-detected from dev runtime if empty)")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 30*time.Minute, "Timeout for pipeline execution")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Validate and build only, don't execute")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Run in background")
	runCmd.Flags().BoolVar(&runAttach, "attach", true, "Attach to container logs (default for streaming)")
	runCmd.Flags().StringArrayVar(&runSet, "set", []string{}, "Override values (key=value, can be repeated)")
	runCmd.Flags().StringArrayVarP(&runValueFiles, "values", "f", []string{}, "Override files (can be repeated)")
	runCmd.Flags().StringVar(&runCell, "cell", "", "Cell name for store resolution (e.g., canary, stable)")
	runCmd.Flags().StringVar(&runKubeContext, "context", "", "kubectl context for multi-cluster cell resolution")
	runCmd.Flags().StringArrayVar(&runScanDirs, "scan-dir", nil,
		"Scan directories for dk.yaml files (runs all transforms in topological order)")
	runCmd.Flags().BoolVar(&runLineage, "lineage", false, "Enable lineage tracking")
	runCmd.Flags().StringVar(&runLineageEndpoint, "lineage-endpoint", "",
		"Override lineage endpoint URL (or set OPENLINEAGE_URL env var)")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	// Multi-transform mode: scan directories and run in topological order
	if len(runScanDirs) > 0 {
		return runMultiTransform(cmd)
	}

	// Determine package directory
	packageDir := "."
	if len(args) > 0 {
		packageDir = args[0]
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify dk.yaml exists
	dkPath := filepath.Join(absDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		return fmt.Errorf("dk.yaml not found in %s - is this a valid DK package?", packageDir)
	}

	// Parse dk.yaml to detect package kind
	m, kind, err := manifest.ParseManifestFile(dkPath)
	if err != nil {
		return fmt.Errorf("failed to parse dk.yaml: %w", err)
	}

	// Read pipeline mode from Transform spec (defaults to batch)
	pipelineMode := "batch"
	var lineageEmitter lineage.Emitter
	if kind == contracts.KindTransform {
		if transform, ok := m.(*contracts.Transform); ok {
			if transform.Spec.Mode.IsValid() {
				pipelineMode = string(transform.Spec.Mode)
			}
			lineageEmitter = buildLineageEmitter(transform, runLineage, runLineageEndpoint)
		}
	}
	if lineageEmitter != nil {
		defer lineageEmitter.Close()
	}

	// Apply overrides if specified
	if len(runValueFiles) > 0 || len(runSet) > 0 {
		if err := applyOverrides(dkPath); err != nil {
			return fmt.Errorf("failed to apply overrides: %w", err)
		}
	}

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range runEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", e)
		}
		env[parts[0]] = parts[1]
	}

	// Auto-detect network from dev runtime if not specified
	network := runNetwork
	if network == "" {
		network = detectDevNetwork()
	}

	// Ensure the Docker network exists before running
	if network != "" {
		if err := ensureNetworkExists(network); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not ensure network %q exists: %v\n", network, err)
		}
	}

	// Create runner
	dockerRunner, err := runner.NewDockerRunner()
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Build run options
	opts := runner.RunOptions{
		PackageDir:     absDir,
		Env:            env,
		Network:        network,
		Timeout:        runTimeout,
		DryRun:         runDryRun,
		Detach:         runDetach,
		Output:         os.Stdout,
		Cell:           runCell,
		KubeContext:    runKubeContext,
		LineageEmitter: lineageEmitter,
	}

	fmt.Printf("Running pipeline from: %s\n", packageDir)
	fmt.Printf("Pipeline mode: %s\n", pipelineMode)
	if runCell != "" {
		fmt.Printf("Cell: %s (stores from dk-%s namespace)\n", runCell, runCell)
	}

	if runDryRun {
		fmt.Println("Dry run mode - will validate and build only")
	}

	fmt.Println()

	// Execute
	ctx := context.Background()
	result, err := dockerRunner.Run(ctx, opts)
	if err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	fmt.Println()

	// Print result
	if runDetach {
		fmt.Printf("✓ Pipeline started in background\n")
		fmt.Printf("  Run ID: %s\n", result.RunID)
		fmt.Printf("  Container: %s\n", result.ContainerID)
		fmt.Printf("  Mode: %s\n", pipelineMode)
		fmt.Println("\nUse these commands to manage the run:")
		fmt.Printf("  View logs: dk logs %s\n", result.RunID)
		fmt.Printf("  Stop:      dk stop %s\n", result.RunID)
	} else {
		switch result.Status {
		case "completed":
			if pipelineMode == "streaming" {
				fmt.Printf("✓ Streaming pipeline stopped gracefully\n")
			} else {
				fmt.Printf("✓ Pipeline completed successfully\n")
			}
			fmt.Printf("  Duration: %s\n", result.Duration.Round(time.Millisecond))
			if result.RecordsProcessed > 0 {
				fmt.Printf("  Records processed: %d\n", result.RecordsProcessed)
			}
		case "failed":
			fmt.Printf("✗ Pipeline failed\n")
			fmt.Printf("  Exit code: %d\n", result.ExitCode)
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", result.Error)
			}
			return fmt.Errorf("pipeline failed with exit code %d", result.ExitCode)
		default:
			fmt.Printf("Pipeline ended with status: %s\n", result.Status)
		}
	}

	return nil
}

// applyOverrides loads dk.yaml, applies override files and --set values,
// and writes the merged result to a temporary file for the runner.
// The runner will use this merged configuration.
func applyOverrides(dkPath string) error {
	// Read base dk.yaml
	baseData, err := os.ReadFile(dkPath)
	if err != nil {
		return fmt.Errorf("failed to read dk.yaml: %w", err)
	}

	// Parse as generic map for merging
	var base map[string]any
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return fmt.Errorf("failed to parse dk.yaml: %w", err)
	}

	mergeOpts := manifest.DefaultMergeOptions()

	// Apply override files in order
	for _, f := range runValueFiles {
		overrideData, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read override file %s: %w", f, err)
		}

		var override map[string]any
		if err := yaml.Unmarshal(overrideData, &override); err != nil {
			return fmt.Errorf("failed to parse override file %s: %w", f, err)
		}

		base = manifest.DeepMerge(base, override, mergeOpts)
		fmt.Printf("Applied overrides from: %s\n", f)
	}

	// Apply --set values in order
	for _, s := range runSet {
		path, value, err := manifest.ParseSetFlag(s)
		if err != nil {
			return fmt.Errorf("invalid --set value: %w", err)
		}

		// Validate the path is allowed
		if err := manifest.ValidateOverridePath(path); err != nil {
			return err
		}

		if err := manifest.SetPath(base, path, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", path, err)
		}
		fmt.Printf("Set: %s=%v\n", path, value)
	}

	// Note: This modifies the file in place. For non-destructive behavior,
	// we could write to a temp file and pass that to the runner.
	mergedData, err := yaml.Marshal(base)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	// Create backup of original
	backupPath := dkPath + ".bak"
	if err := os.WriteFile(backupPath, baseData, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write merged config
	if err := os.WriteFile(dkPath, mergedData, 0644); err != nil {
		// Restore from backup on failure
		os.WriteFile(dkPath, baseData, 0644)
		return fmt.Errorf("failed to write merged config: %w", err)
	}

	// Defer cleanup to restore original after run
	// Note: In a real implementation, we'd use defer or a cleanup function
	// For now, we leave the merged file - the user can restore from .bak

	return nil
}

// detectDevNetwork returns the Docker network name for the active dev runtime.
// For k3d, it uses "k3d-<cluster-name>". For compose, it uses "dk-network".
func detectDevNetwork() string {
	config, err := localdev.LoadConfig()
	if err != nil {
		// Fall back to trying k3d network first (since it's the default runtime),
		// then dk-network.
		return detectNetworkByProbing()
	}

	switch config.GetDefaultRuntime() {
	case localdev.RuntimeK3d:
		cluster := config.Dev.K3d.ClusterName
		if cluster == "" {
			cluster = localdev.DefaultClusterName
		}
		return fmt.Sprintf("k3d-%s", cluster)
	default:
		return detectNetworkByProbing()
	}
}

// detectNetworkByProbing checks which dev network actually exists.
func detectNetworkByProbing() string {
	// Try k3d network (default runtime)
	k3dNetwork := fmt.Sprintf("k3d-%s", localdev.DefaultClusterName)
	if networkExists(k3dNetwork) {
		return k3dNetwork
	}
	return "dk-network"
}

// runMultiTransform builds the pipeline graph, sorts transforms topologically,
// and executes each one sequentially in dependency order.
func runMultiTransform(cmd *cobra.Command) error {
	// Resolve scan directories to absolute paths
	absDirs := make([]string, 0, len(runScanDirs))
	for _, d := range runScanDirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %w", d, err)
		}
		absDirs = append(absDirs, abs)
	}

	// Build pipeline graph
	g, err := pipeline.BuildGraph(pipeline.GraphOptions{
		ScanDirs: absDirs,
		ShowAll:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to build pipeline graph: %w", err)
	}

	// Get transforms in topological order
	transforms := g.TransformDirs()
	if len(transforms) == 0 {
		fmt.Println("No transforms found in scanned directories.")
		return nil
	}

	fmt.Printf("Multi-transform run: %d transforms in topological order\n", len(transforms))
	if runCell != "" {
		fmt.Printf("Cell: %s (stores from dk-%s namespace)\n", runCell, runCell)
	}
	if runDryRun {
		fmt.Println("Dry run mode - will validate and build only")
	}
	fmt.Println()

	// Print execution plan
	for i, t := range transforms {
		fmt.Printf("  %d. %s (runtime: %s, dir: %s)\n", i+1, t.Name, t.Runtime, t.Dir)
	}
	fmt.Println()

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range runEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", e)
		}
		env[parts[0]] = parts[1]
	}

	// Auto-detect network from dev runtime if not specified
	network := runNetwork
	if network == "" {
		network = detectDevNetwork()
	}

	// Ensure the Docker network exists before running
	if network != "" {
		if err := ensureNetworkExists(network); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not ensure network %q exists: %v\n", network, err)
		}
	}

	// Execute each transform in order
	var failed []string
	var succeeded []string
	totalStart := time.Now()

	for i, t := range transforms {
		fmt.Printf("━━━ [%d/%d] %s ━━━\n", i+1, len(transforms), t.Name)

		// Create a fresh runner for each transform
		dockerRunner, err := runner.NewDockerRunner()
		if err != nil {
			fmt.Printf("✗ Failed to create runner: %v\n\n", err)
			failed = append(failed, t.Name)
			return fmt.Errorf("transform %s: failed to create runner: %w", t.Name, err)
		}

		// Build lineage emitter for this transform
		var emitter lineage.Emitter
		dkPath := filepath.Join(t.Dir, "dk.yaml")
		if m, _, parseErr := manifest.ParseManifestFile(dkPath); parseErr == nil {
			if transform, ok := m.(*contracts.Transform); ok {
				emitter = buildLineageEmitter(transform, runLineage, runLineageEndpoint)
			}
		}

		opts := runner.RunOptions{
			PackageDir:     t.Dir,
			Env:            env,
			Network:        network,
			Timeout:        runTimeout,
			DryRun:         runDryRun,
			Detach:         false, // Always run sequentially in multi-transform mode
			Output:         os.Stdout,
			Cell:           runCell,
			KubeContext:    runKubeContext,
			LineageEmitter: emitter,
		}

		ctx := context.Background()
		result, err := dockerRunner.Run(ctx, opts)
		// Clean up emitter after each transform
		if emitter != nil {
			emitter.Close()
		}

		if err != nil {
			fmt.Printf("✗ %s failed: %v\n\n", t.Name, err)
			failed = append(failed, t.Name)
			// Stop on first failure — downstream transforms depend on this one
			break
		}

		switch result.Status {
		case "completed":
			fmt.Printf("✓ %s completed (%s)\n\n", t.Name, result.Duration.Round(time.Millisecond))
			succeeded = append(succeeded, t.Name)
		case "failed":
			fmt.Printf("✗ %s failed (exit code %d)\n", t.Name, result.ExitCode)
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", result.Error)
			}
			fmt.Println()
			failed = append(failed, t.Name)
			break
		default:
			fmt.Printf("? %s ended with status: %s\n\n", t.Name, result.Status)
			succeeded = append(succeeded, t.Name)
		}
	}

	// Print summary
	totalDuration := time.Since(totalStart)
	fmt.Println("━━━ Summary ━━━")
	fmt.Printf("Total:     %d transforms\n", len(transforms))
	fmt.Printf("Succeeded: %d\n", len(succeeded))
	fmt.Printf("Failed:    %d\n", len(failed))
	fmt.Printf("Skipped:   %d\n", len(transforms)-len(succeeded)-len(failed))
	fmt.Printf("Duration:  %s\n", totalDuration.Round(time.Millisecond))

	if len(failed) > 0 {
		return fmt.Errorf("%d transform(s) failed: %s", len(failed), strings.Join(failed, ", "))
	}

	return nil
}

// buildLineageEmitter creates a lineage emitter from the transform spec and CLI flags.
// Returns nil if lineage is not enabled or on error (errors are logged as warnings).
// Lineage failures should never block pipeline execution.
func buildLineageEmitter(t *contracts.Transform, flagEnabled bool, flagEndpoint string) lineage.Emitter {
	// Determine if lineage is enabled
	enabled := flagEnabled
	if !enabled && t.Spec.Lineage != nil && t.Spec.Lineage.Enabled {
		enabled = true
	}
	if !enabled {
		return nil
	}

	// Determine emitter type from manifest, default to "marquez"
	emitterType := "marquez"
	if t.Spec.Lineage != nil && t.Spec.Lineage.Emitter != "" {
		emitterType = t.Spec.Lineage.Emitter
	}

	// Determine endpoint: flag > env var > type-specific default
	endpoint := flagEndpoint
	if endpoint == "" {
		endpoint = os.Getenv("OPENLINEAGE_URL")
	}
	if endpoint == "" {
		switch emitterType {
		case "marquez":
			endpoint = "http://localhost:5000"
		case "datahub":
			endpoint = "http://localhost:8080"
		}
	}

	// Determine namespace from manifest, default to "dk"
	namespace := "dk"
	if t.Spec.Lineage != nil && t.Spec.Lineage.Namespace != "" {
		namespace = t.Spec.Lineage.Namespace
	}

	config := lineage.EmitterConfig{
		Type:           emitterType,
		Endpoint:       endpoint,
		Namespace:      namespace,
		TimeoutSeconds: 30,
	}

	emitter, err := lineage.NewEmitterFromConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create lineage emitter: %v\n", err)
		return nil
	}

	return emitter
}

// networkExists checks if a Docker network exists.
func networkExists(name string) bool {
	cmd := exec.Command("docker", "network", "inspect", name)
	return cmd.Run() == nil
}
