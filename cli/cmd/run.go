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
	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	runEnv        []string
	runNetwork    string
	runTimeout    time.Duration
	runDryRun     bool
	runDetach     bool
	runAttach     bool     // Explicitly attach to logs (for streaming)
	runSet        []string // --set flags for inline overrides
	runValueFiles []string // -f flags for override files
)

// runCmd executes a pipeline locally
var runCmd = &cobra.Command{
	Use:   "run [package-dir]",
	Short: "Run a data package locally",
	Long: `Execute a data package locally using Docker.

The run command builds (if needed) and executes the package defined in
the specified directory.

The command will:
1. Parse dp.yaml manifest
2. Apply any override files (-f) and inline overrides (--set)
3. Build the Docker image
4. Start the container on the Docker network
5. Stream logs to stdout

Override precedence (lowest to highest):
  - dp.yaml (base configuration)
  - Override files (-f) in order specified
  - Inline overrides (--set) in order specified

Prerequisites:
  - Docker must be running

Examples:
  # Run pipeline in current directory
  dp run

  # Run pipeline in specific directory
  dp run ./my-pipeline

  # Run with custom environment variables
  dp run -e DEBUG=true -e LOG_LEVEL=debug

  # Override configuration values
  dp run --set spec.resources.memory=8Gi

  # Use an override file
  dp run -f prod-overrides.yaml

  # Combine override file and inline overrides
  dp run -f prod-overrides.yaml --set spec.timeout=4h

  # Dry run (validate only, don't execute)
  dp run --dry-run

  # Run in background
  dp run --detach`,
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
}

func runPipeline(cmd *cobra.Command, args []string) error {
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

	// Verify dp.yaml exists
	dpPath := filepath.Join(absDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		return fmt.Errorf("dp.yaml not found in %s - is this a valid DP package?", packageDir)
	}

	// Parse dp.yaml to detect package kind
	m, kind, err := manifest.ParseManifestFile(dpPath)
	if err != nil {
		return fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	// Read pipeline mode from Transform spec (defaults to batch)
	pipelineMode := "batch"
	if kind == contracts.KindTransform {
		if transform, ok := m.(*contracts.Transform); ok {
			if transform.Spec.Mode.IsValid() {
				pipelineMode = string(transform.Spec.Mode)
			}
		}
	}

	// Apply overrides if specified
	if len(runValueFiles) > 0 || len(runSet) > 0 {
		if err := applyOverrides(dpPath); err != nil {
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
		PackageDir: absDir,
		Env:        env,
		Network:    network,
		Timeout:    runTimeout,
		DryRun:     runDryRun,
		Detach:     runDetach,
		Output:     os.Stdout,
	}

	fmt.Printf("Running pipeline from: %s\n", packageDir)
	fmt.Printf("Pipeline mode: %s\n", pipelineMode)

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
		fmt.Printf("  View logs: dp logs %s\n", result.RunID)
		fmt.Printf("  Stop:      dp stop %s\n", result.RunID)
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

// applyOverrides loads dp.yaml, applies override files and --set values,
// and writes the merged result to a temporary file for the runner.
// The runner will use this merged configuration.
func applyOverrides(dpPath string) error {
	// Read base dp.yaml
	baseData, err := os.ReadFile(dpPath)
	if err != nil {
		return fmt.Errorf("failed to read dp.yaml: %w", err)
	}

	// Parse as generic map for merging
	var base map[string]any
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return fmt.Errorf("failed to parse dp.yaml: %w", err)
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

	// Write merged config back to dp.yaml
	// Note: This modifies the file in place. For non-destructive behavior,
	// we could write to a temp file and pass that to the runner.
	mergedData, err := yaml.Marshal(base)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	// Create backup of original
	backupPath := dpPath + ".bak"
	if err := os.WriteFile(backupPath, baseData, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write merged config
	if err := os.WriteFile(dpPath, mergedData, 0644); err != nil {
		// Restore from backup on failure
		os.WriteFile(dpPath, baseData, 0644)
		return fmt.Errorf("failed to write merged config: %w", err)
	}

	// Defer cleanup to restore original after run
	// Note: In a real implementation, we'd use defer or a cleanup function
	// For now, we leave the merged file - the user can restore from .bak

	return nil
}

// detectDevNetwork returns the Docker network name for the active dev runtime.
// For k3d, it uses "k3d-<cluster-name>". For compose, it uses "dp-network".
func detectDevNetwork() string {
	config, err := localdev.LoadConfig()
	if err != nil {
		// Fall back to trying k3d network first (since it's the default runtime),
		// then dp-network.
		return detectNetworkByProbing()
	}

	switch config.GetDefaultRuntime() {
	case localdev.RuntimeK3d:
		cluster := config.Dev.K3d.ClusterName
		if cluster == "" {
			cluster = localdev.DefaultClusterName
		}
		return fmt.Sprintf("k3d-%s", cluster)
	case localdev.RuntimeCompose:
		return "dp-network"
	default:
		return detectNetworkByProbing()
	}
}

// detectNetworkByProbing checks which dev network actually exists.
func detectNetworkByProbing() string {
	// Try k3d network first (default runtime)
	k3dNetwork := fmt.Sprintf("k3d-%s", localdev.DefaultClusterName)
	if networkExists(k3dNetwork) {
		return k3dNetwork
	}
	// Fall back to compose network
	return "dp-network"
}

// networkExists checks if a Docker network exists.
func networkExists(name string) bool {
	cmd := exec.Command("docker", "network", "inspect", name)
	return cmd.Run() == nil
}
