package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	runEnv         []string
	runBindings    string
	runNetwork     string
	runTimeout     time.Duration
	runDryRun      bool
	runDetach      bool
	runAttach      bool     // Explicitly attach to logs (for streaming)
	runSet         []string // --set flags for inline overrides
	runValueFiles  []string // -f flags for override files
	runSync        bool     // --sync runs a full source→destination sync
	runDestination string   // --destination selects the destination plugin
	runRegistry    string   // --registry overrides the plugin registry for this invocation
)

// runCmd executes a pipeline locally
var runCmd = &cobra.Command{
	Use:   "run [package-dir]",
	Short: "Run a data package locally",
	Long: `Execute a data package locally using the k3d development cluster.

Supported package types: pipeline, cloudquery

The run command builds (if needed) and executes the package defined in
the specified directory.

For pipeline packages the command will:
1. Parse dp.yaml manifest
2. Apply any override files (-f) and inline overrides (--set)
3. Build the Docker image
4. Start the container on the k3d Docker network
5. Stream logs to stdout

For cloudquery packages the command will:
1. Parse dp.yaml manifest and validate the cloudquery section
2. Build a distroless container image for the plugin
3. Import the image into the k3d cluster
4. Deploy the plugin as a Kubernetes Pod (develop like you deploy)
5. Port-forward the gRPC port to localhost
6. Discover and display plugin tables
7. Clean up the pod

Override precedence (lowest to highest):
  - dp.yaml (base configuration)
  - Override files (-f) in order specified
  - Inline overrides (--set) in order specified

Prerequisites:
  - Docker must be running
  - Local dev stack must be running (dp dev up)
  - For CloudQuery: cloudquery CLI, kubectl, and k3d must be installed

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
  dp run -f prod-overrides.yaml --set spec.runtime.timeout=4h

  # Dry run (validate only, don't execute)
  dp run --dry-run

  # Run in background
  dp run --detach

  # Run a CloudQuery plugin
  dp run ./my-source

  # Run a full sync (source → local files)
  dp run ./my-source --sync

  # Sync to PostgreSQL
  dp run ./my-source --sync --destination postgresql`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPipeline,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringArrayVarP(&runEnv, "env", "e", []string{}, "Environment variables (KEY=VALUE)")
	runCmd.Flags().StringVarP(&runBindings, "bindings", "b", "", "Path to bindings file")
	runCmd.Flags().StringVar(&runNetwork, "network", "", "Docker network to connect to (auto-detected from dev runtime if empty)")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 30*time.Minute, "Timeout for pipeline execution")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Validate and build only, don't execute")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Run in background")
	runCmd.Flags().BoolVar(&runAttach, "attach", true, "Attach to container logs (default for streaming)")
	runCmd.Flags().StringArrayVar(&runSet, "set", []string{}, "Override values (key=value, can be repeated)")
	runCmd.Flags().StringArrayVarP(&runValueFiles, "values", "f", []string{}, "Override files (can be repeated)")
	runCmd.Flags().BoolVar(&runSync, "sync", false, "Run a full CloudQuery sync (source → destination)")
	runCmd.Flags().StringVar(&runDestination, "destination", "file", "Destination plugin for sync (file, postgresql, s3)")
	runCmd.Flags().StringVar(&runRegistry, "registry", "", "Override plugin registry for this invocation")
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

	// Parse dp.yaml to detect package type
	dp, err := manifest.ParseDataPackageFile(dpPath)
	if err != nil {
		return fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	// Route to CloudQuery run path if type is cloudquery
	if dp.Spec.Type == contracts.PackageTypeCloudQuery {
		return runCloudQuery(cmd, absDir, dp)
	}

	// Read pipeline mode from pipeline.yaml (if exists)
	pipelineMode := "batch" // Default
	pipelinePath := filepath.Join(absDir, "pipeline.yaml")
	if pipeline, err := manifest.ParsePipelineFile(pipelinePath); err == nil {
		if pipeline.Spec.Mode != "" {
			pipelineMode = string(pipeline.Spec.Mode)
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
		PackageDir:   absDir,
		Env:          env,
		BindingsFile: runBindings,
		Network:      network,
		Timeout:      runTimeout,
		DryRun:       runDryRun,
		Detach:       runDetach,
		Output:       os.Stdout,
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

// --- CloudQuery Run Path ---

// runCloudQuery orchestrates the CloudQuery plugin workflow using the k3d cluster:
// 1. Check for cloudquery CLI and kubectl
// 2. Build plugin container image
// 3. Import image into k3d cluster
// 4. Deploy as a Kubernetes Pod
// 5. Port-forward gRPC port to localhost
// 6. Discover tables (validates gRPC + plugin logic)
// 7. Display summary and cleanup
func runCloudQuery(cmd *cobra.Command, absDir string, dp *contracts.DataPackage) error {
	// Check for required CLIs
	if err := checkCloudQueryBinary(); err != nil {
		return err
	}
	if err := checkBinary("kubectl", "kubectl is required for k3d deployment.\nInstall: https://kubernetes.io/docs/tasks/tools/"); err != nil {
		return err
	}
	if err := checkBinary("k3d", "k3d is required for local development.\nInstall: https://k3d.io/#installation"); err != nil {
		return err
	}

	// Resolve k3d cluster config
	config, err := localdev.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	clusterName := config.Dev.K3d.ClusterName
	if clusterName == "" {
		clusterName = localdev.DefaultClusterName
	}
	kubeContext := fmt.Sprintf("k3d-%s", clusterName)
	namespace := localdev.DefaultNamespace

	// Verify the k3d cluster is running
	if err := verifyClusterRunning(kubeContext, namespace); err != nil {
		return err
	}

	// Determine gRPC port
	grpcPort := 7777
	if dp.Spec.CloudQuery != nil && dp.Spec.CloudQuery.GRPCPort > 0 {
		grpcPort = dp.Spec.CloudQuery.GRPCPort
	}

	podName := fmt.Sprintf("cq-%s", dp.Metadata.Name)
	imageName := fmt.Sprintf("%s/%s:latest", dp.Metadata.Namespace, dp.Metadata.Name)

	// Set up signal handler for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	cleanup := func() {
		fmt.Println("\nCleaning up...")
		deletePod(kubeContext, namespace, podName)
	}

	go func() {
		<-sigCh
		cleanup()
		os.Exit(1)
	}()

	// Build plugin container image
	lang := detectCloudQueryLanguage(absDir)
	fmt.Printf("Building CloudQuery plugin image: %s (lang=%s)\n", imageName, lang)
	if err := buildDockerImage(absDir, imageName, lang, grpcPort, false); err != nil {
		return fmt.Errorf("failed to build plugin image: %w", err)
	}

	// Import image into k3d so the cluster can use it
	fmt.Printf("Importing image into k3d cluster %q...\n", clusterName)
	if err := importImageToK3d(imageName, clusterName); err != nil {
		return fmt.Errorf("failed to import image into k3d: %w", err)
	}

	// Delete any previous pod with the same name
	deletePod(kubeContext, namespace, podName)

	// Deploy as a Kubernetes Pod
	fmt.Printf("Deploying pod %q in namespace %q...\n", podName, namespace)
	if err := createPluginPod(kubeContext, namespace, podName, imageName, grpcPort); err != nil {
		return fmt.Errorf("failed to create plugin pod: %w", err)
	}
	defer cleanup()

	// Wait for pod to be ready
	fmt.Println("Waiting for pod to be ready...")
	if err := waitForPodReady(kubeContext, namespace, podName, 60*time.Second); err != nil {
		// Show pod logs for debugging
		showPodLogs(kubeContext, namespace, podName)
		return fmt.Errorf("pod not ready: %w", err)
	}
	fmt.Println("✓ Pod is running")

	// Start port-forward
	fmt.Printf("Port-forwarding localhost:%d → pod/%s:%d\n", grpcPort, podName, grpcPort)
	pfCmd, err := startPortForward(kubeContext, namespace, podName, grpcPort)
	if err != nil {
		return fmt.Errorf("failed to start port-forward: %w", err)
	}
	defer func() {
		if pfCmd.Process != nil {
			pfCmd.Process.Kill()
			pfCmd.Wait()
		}
	}()

	// Wait for gRPC to be reachable through the port-forward
	fmt.Printf("Waiting for gRPC server on port %d...\n", grpcPort)
	if err := waitForGRPC(grpcPort, 30*time.Second); err != nil {
		return fmt.Errorf("gRPC server health check failed: %w", err)
	}
	fmt.Println("✓ gRPC server is ready")

	// Generate source-only spec for table discovery
	sourceConfigPath, err := generateSourceConfig(dp, grpcPort)
	if err != nil {
		return fmt.Errorf("failed to generate source config: %w", err)
	}
	defer os.Remove(sourceConfigPath)

	// Discover and display tables (validates gRPC + plugin logic, no login required)
	fmt.Println("\nDiscovering tables...")
	fmt.Println()
	if err := runCloudQueryTables(sourceConfigPath); err != nil {
		return fmt.Errorf("cloudquery table discovery failed: %w", err)
	}

	// If --sync is enabled, run a full sync with the selected destination
	if runSync {
		fmt.Printf("\nPreparing sync: %s → %s\n", dp.Metadata.Name, runDestination)

		// Load hierarchical config and resolve plugin image
		cfg, err := localdev.LoadHierarchicalConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// T035: Observability — log which config source is active
		fmt.Printf("  Config: registry=%s", cfg.Plugins.Registry)

		// --registry flag overrides config
		if runRegistry != "" {
			cfg.Plugins.Registry = runRegistry
			fmt.Printf(" (overridden by --registry flag: %s)", runRegistry)
		}
		fmt.Println()

		imageRef := resolvePluginImage(runDestination, cfg)
		fmt.Printf("  Plugin image: %s\n", imageRef)

		// Pull the destination plugin image (with mirror fallback)
		if err := pullWithMirrorFallback(imageRef, cfg.Plugins.Mirrors); err != nil {
			return fmt.Errorf("failed to pull destination image: %w", err)
		}

		// Deploy destination plugin:
		// - file: local Docker container with bind mount (output lands on host)
		// - others: k3d pod with port-forward
		var destPort int
		var cleanupFn func()

		if runDestination == "file" {
			containerName, port, cleanup, err := deployDestinationContainer(imageRef)
			if err != nil {
				return fmt.Errorf("failed to deploy destination container: %w", err)
			}
			destPort = port
			cleanupFn = cleanup
			fmt.Printf("  Destination container: %s (port %d)\n", containerName, destPort)
		} else {
			clusterName := cfg.Dev.K3d.ClusterName
			namespace := "dp-local"

			podName, port, cleanup, err := deployDestinationPod(imageRef, clusterName, namespace)
			if err != nil {
				return fmt.Errorf("failed to deploy destination pod: %w", err)
			}
			destPort = port
			cleanupFn = cleanup
			fmt.Printf("  Destination pod: %s (port %d)\n", podName, destPort)
		}
		defer cleanupFn()

		syncConfigPath, err := generateSyncConfig(dp, grpcPort, runDestination, destPort, cfg, "dp-local")
		if err != nil {
			return fmt.Errorf("failed to generate sync config: %w", err)
		}
		defer os.Remove(syncConfigPath)

		fmt.Printf("\nSyncing: %s → %s\n\n", dp.Metadata.Name, runDestination)
		if err := runCloudQuerySync(syncConfigPath); err != nil {
			return fmt.Errorf("cloudquery sync failed: %w", err)
		}

		fmt.Println()
		fmt.Printf("✓ Sync completed: %s → %s\n", dp.Metadata.Name, runDestination)
		if runDestination == "file" {
			fmt.Println("  Output directory: ./cq-sync-output/")
		}
		return nil
	}

	fmt.Println()
	fmt.Println("✓ CloudQuery plugin is working correctly")
	fmt.Printf("\nTo run a full sync, add the --sync flag:\n")
	fmt.Printf("  dp run %s --sync\n", dp.Metadata.Name)
	fmt.Printf("  dp run %s --sync --destination postgresql\n", dp.Metadata.Name)

	return nil
}

// checkCloudQueryBinary verifies the cloudquery CLI is installed.
func checkCloudQueryBinary() error {
	return checkBinary("cloudquery",
		"cloudquery CLI not found in PATH.\n\n"+
			"Install it with one of:\n"+
			"  brew install cloudquery/tap/cloudquery   # macOS\n"+
			"  curl -L https://github.com/cloudquery/cloudquery/releases/latest/download/cloudquery_linux_amd64 -o /usr/local/bin/cloudquery && chmod +x /usr/local/bin/cloudquery   # Linux\n"+
			"\nSee https://docs.cloudquery.io/docs/quickstart for more options")
}

// checkBinary verifies a CLI binary is installed.
func checkBinary(name, errMsg string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

// verifyClusterRunning checks that the k3d cluster is up and reachable.
func verifyClusterRunning(kubeContext, namespace string) error {
	cmd := exec.Command("kubectl", "--context", kubeContext, "get", "namespace", namespace)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("k3d cluster not reachable (context=%s, namespace=%s).\n"+
			"Make sure the dev environment is running:\n"+
			"  dp dev up\n\n%s", kubeContext, namespace, stderr.String())
	}
	return nil
}

// importImageToK3d loads a locally-built Docker image into the k3d cluster.
func importImageToK3d(imageName, clusterName string) error {
	cmd := exec.Command("k3d", "image", "import", imageName, "--cluster", clusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// createPluginPod creates a Kubernetes Pod running the CloudQuery plugin.
func createPluginPod(kubeContext, namespace, podName, imageName string, grpcPort int) error {
	cmd := exec.Command("kubectl", "--context", kubeContext,
		"run", podName,
		"--namespace", namespace,
		"--image", imageName,
		"--image-pull-policy", "Never", // Use the imported image, don't pull from registry
		"--port", fmt.Sprintf("%d", grpcPort),
		"--labels", "app=cloudquery-plugin",
		"--restart", "Never",
	)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// waitForPodReady waits for a pod to reach the Running phase.
func waitForPodReady(kubeContext, namespace, podName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd := exec.Command("kubectl", "--context", kubeContext,
			"get", "pod", podName,
			"--namespace", namespace,
			"-o", "jsonpath={.status.phase}",
		)
		out, err := cmd.Output()
		if err == nil {
			phase := string(out)
			if phase == "Running" {
				return nil
			}
			if phase == "Failed" || phase == "Error" {
				return fmt.Errorf("pod entered %s phase", phase)
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timeout waiting for pod %s to be running after %v", podName, timeout)
}

// startPortForward starts a kubectl port-forward process and returns the command.
func startPortForward(kubeContext, namespace, podName string, port int) (*exec.Cmd, error) {
	cmd := exec.Command("kubectl", "--context", kubeContext,
		"port-forward",
		fmt.Sprintf("pod/%s", podName),
		fmt.Sprintf("%d:%d", port, port),
		"--namespace", namespace,
	)
	// Discard port-forward output (it's noisy)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start port-forward: %w", err)
	}
	// Give port-forward a moment to establish
	time.Sleep(2 * time.Second)
	return cmd, nil
}

// showPodLogs prints recent pod logs for debugging.
func showPodLogs(kubeContext, namespace, podName string) {
	cmd := exec.Command("kubectl", "--context", kubeContext,
		"logs", podName,
		"--namespace", namespace,
		"--tail", "20",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Run() // Best effort
}

// deletePod deletes a Kubernetes Pod (best effort).
func deletePod(kubeContext, namespace, podName string) {
	cmd := exec.Command("kubectl", "--context", kubeContext,
		"delete", "pod", podName,
		"--namespace", namespace,
		"--grace-period", "0",
		"--force",
		"--ignore-not-found",
	)
	cmd.Run() // Best effort
}

// detectCloudQueryLanguage determines whether a CloudQuery plugin project is
// Python or Go by checking for language-specific files in the project directory.
func detectCloudQueryLanguage(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(dir, "main.py")); err == nil {
		return "python"
	}
	return "python"
}

// cloudQueryDockerfile returns a generated Dockerfile for a CloudQuery plugin.
// The CLI owns the build process — users never see or edit a Dockerfile.
// Both Go and Python use distroless runtime images for minimal attack surface.
// Dockerfiles use buildx cache mounts for fast rebuilds (Go module cache, build cache,
// pip cache). Module/dependency files are copied first to maximize layer caching.
func cloudQueryDockerfile(lang string, grpcPort int) string {
	switch lang {
	case "go":
		return fmt.Sprintf(`# syntax=docker/dockerfile:1
# Build stage — cached Go modules and build artifacts via buildx cache mounts
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /plugin .

# Runtime stage — distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /plugin /usr/local/bin/plugin
EXPOSE %d
ENTRYPOINT ["/usr/local/bin/plugin", "serve", "--address", "0.0.0.0:%d"]
`, grpcPort, grpcPort)

	default: // python
		return fmt.Sprintf(`# syntax=docker/dockerfile:1
# Build stage — cached pip installs via buildx cache mount
FROM python:3.13-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --target=/deps -r requirements.txt
COPY . .

# Runtime stage — distroless for minimal attack surface
FROM gcr.io/distroless/python3-debian12:nonroot
WORKDIR /app
COPY --from=builder /deps /usr/local/lib/python3.13/site-packages
COPY --from=builder /app /app
ENV PYTHONPATH=/usr/local/lib/python3.13/site-packages
EXPOSE %d
ENTRYPOINT ["python3", "main.py", "serve", "--address", "[::]:%d"]
`, grpcPort, grpcPort)
	}
}

// buildDockerImage generates a Dockerfile on the fly and builds the image using
// docker buildx. Buildx is required for --mount=type=cache directives in the
// Dockerfile that cache Go modules, Go build artifacts, and pip packages.
// A .dockerignore is ensured in the build context to exclude sync output
// and other runtime artifacts that would invalidate the Docker layer cache.
func buildDockerImage(dir, imageName string, lang string, grpcPort int, noCache bool) error {
	dockerfile := cloudQueryDockerfile(lang, grpcPort)

	// Ensure .dockerignore exists in the build context so runtime artifacts
	// (like cq-sync-output/) don't bust the Docker layer cache on rebuilds.
	if err := ensureDockerignore(dir); err != nil {
		return fmt.Errorf("failed to ensure .dockerignore: %w", err)
	}

	// Use buildx for cache mount support. --load ensures the image is
	// available in the local docker image store (needed for k3d import).
	args := []string{"buildx", "build", "--load", "-t", imageName, "-f", "-"}
	if noCache {
		args = append(args, "--no-cache")
	}
	args = append(args, dir)

	cmd := exec.Command("docker", args...)
	cmd.Stdin = strings.NewReader(dockerfile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// dockerignorePatterns are patterns that should always be excluded from the
// Docker build context for data package builds. These are runtime artifacts
// that change between runs and would otherwise invalidate layer caches.
var dockerignorePatterns = []string{
	"cq-sync-output/",
	"*.log",
	".env",
	".env.*",
}

// ensureDockerignore makes sure a .dockerignore file in dir contains all
// required exclusion patterns. If the file doesn't exist it is created.
// If it already exists, any missing patterns are appended.
func ensureDockerignore(dir string) error {
	ignorePath := filepath.Join(dir, ".dockerignore")

	existing := ""
	if data, err := os.ReadFile(ignorePath); err == nil {
		existing = string(data)
	}

	var missing []string
	for _, p := range dockerignorePatterns {
		if !strings.Contains(existing, p) {
			missing = append(missing, p)
		}
	}

	if len(missing) == 0 {
		return nil // all patterns already present
	}

	// Append missing patterns (with a leading newline separator if file existed)
	f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if existing != "" && !strings.HasSuffix(existing, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	header := "# Auto-generated by dp — exclude runtime artifacts from build context\n"
	if !strings.Contains(existing, header) {
		if _, err := f.WriteString(header); err != nil {
			return err
		}
	}

	for _, p := range missing {
		if _, err := f.WriteString(p + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// waitForGRPC waits for the gRPC server to become reachable via TCP.
func waitForGRPC(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("localhost:%d", port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for gRPC server on %s after %v", addr, timeout)
}

// generateSourceConfig creates a temporary CloudQuery source-only configuration file.
// No destination is needed — used with `cloudquery tables` for zero-auth validation.
func generateSourceConfig(dp *contracts.DataPackage, grpcPort int) (string, error) {
	tables := `["*"]`
	if dp.Spec.CloudQuery != nil && len(dp.Spec.CloudQuery.Tables) > 0 {
		tableList := make([]string, len(dp.Spec.CloudQuery.Tables))
		for i, t := range dp.Spec.CloudQuery.Tables {
			tableList[i] = fmt.Sprintf("%q", t)
		}
		tables = "[" + strings.Join(tableList, ", ") + "]"
	}

	config := fmt.Sprintf(`kind: source
spec:
  name: "%s"
  registry: grpc
  path: "localhost:%d"
  tables: %s
  spec: {}
`, dp.Metadata.Name, grpcPort, tables)

	tmpFile, err := os.CreateTemp("", "cq-source-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp source config: %w", err)
	}

	if _, err := tmpFile.WriteString(config); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write source config: %w", err)
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// cqTable represents a table discovered by cloudquery tables.
type cqTable struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Columns     []cqColumn `json:"columns"`
	Relations   []cqTable  `json:"relations"`
}

// cqColumn represents a column in a CloudQuery table.
type cqColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// runCloudQueryTables runs `cloudquery tables` to discover and display plugin tables.
// This validates the gRPC connection and plugin logic without requiring any destination or login.
func runCloudQueryTables(configPath string) error {
	outputDir, err := os.MkdirTemp("", "cq-tables-*")
	if err != nil {
		return fmt.Errorf("failed to create temp output dir: %w", err)
	}
	defer os.RemoveAll(outputDir)

	cmd := exec.Command("cloudquery", "tables", configPath,
		"--format", "json",
		"--output-dir", outputDir,
		"--log-console",
		"--log-level", "warn",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("cloudquery tables exited with code %d", exitErr.ExitCode())
		}
		return err
	}

	// Collect all tables from JSON files
	var allTables []cqTable
	filepath.WalkDir(outputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Ext(d.Name()) == ".json" {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			var tables []cqTable
			if jsonErr := json.Unmarshal(data, &tables); jsonErr != nil {
				return nil
			}
			allTables = append(allTables, tables...)
		}
		return nil
	})

	if len(allTables) == 0 {
		fmt.Println("No tables discovered.")
		return nil
	}

	fmt.Printf("Discovered %d table(s):\n\n", len(allTables))
	printTables(allTables, "")

	return nil
}

// printTables displays tables and their columns with optional indentation for relations.
func printTables(tables []cqTable, indent string) {
	for _, t := range tables {
		fmt.Printf("%s  %s\n", indent, t.Name)
		if t.Description != "" {
			fmt.Printf("%s    %s\n", indent, t.Description)
		}
		fmt.Printf("%s    Columns:\n", indent)
		// Compute max column name width for alignment
		maxWidth := 0
		for _, c := range t.Columns {
			if len(c.Name) > maxWidth {
				maxWidth = len(c.Name)
			}
		}
		for _, c := range t.Columns {
			fmt.Printf("%s      %-*s  %s\n", indent, maxWidth, c.Name, c.Type)
		}
		if len(t.Relations) > 0 {
			fmt.Printf("%s    Relations:\n", indent)
			printTables(t.Relations, indent+"    ")
		}
		fmt.Println()
	}
}

// --- Destination Plugin Management ---

// destinationPluginInfo holds metadata for a supported CloudQuery destination plugin.
type destinationPluginInfo struct {
	// defaultVersion is the built-in default version for this plugin.
	defaultVersion string
}

// supportedDestinations maps destination names to their plugin info.
var supportedDestinations = map[string]destinationPluginInfo{
	"file": {
		defaultVersion: "v5.5.1",
	},
	"postgresql": {
		defaultVersion: "v8.14.1",
	},
	"s3": {
		defaultVersion: "v7.10.1",
	},
}

// resolvePluginImage builds the full container image reference for a destination plugin,
// applying the image resolution state machine:
//  1. If override.image is set → use as-is
//  2. If override.version is set → {registry}/cloudquery-plugin-{name}:{override.version}
//  3. Otherwise → {registry}/cloudquery-plugin-{name}:{built-in-default-version}
func resolvePluginImage(name string, cfg *localdev.Config) string {
	// Check for image override
	if cfg != nil && cfg.Plugins.Overrides != nil {
		if override, ok := cfg.Plugins.Overrides[name]; ok {
			if override.Image != "" {
				return override.Image
			}
			if override.Version != "" {
				registry := localdev.DefaultPluginRegistry
				if cfg.Plugins.Registry != "" {
					registry = cfg.Plugins.Registry
				}
				return fmt.Sprintf("%s/cloudquery-plugin-%s:%s", registry, name, override.Version)
			}
		}
	}

	// Use default registry and version
	registry := localdev.DefaultPluginRegistry
	if cfg != nil && cfg.Plugins.Registry != "" {
		registry = cfg.Plugins.Registry
	}

	version := localdev.DefaultPluginVersions[name]
	if version == "" {
		// Fallback for unknown plugins
		version = "latest"
	}

	return fmt.Sprintf("%s/cloudquery-plugin-%s:%s", registry, name, version)
}

// pullDestinationImage pulls a container image using docker.
func pullDestinationImage(imageRef string) error {
	fmt.Printf("  Pulling image: %s\n", imageRef)

	if err := checkBinary("docker", "Docker is required to pull destination plugin images.\nInstall: https://docs.docker.com/get-docker/"); err != nil {
		return err
	}

	cmd := exec.Command("docker", "pull", imageRef)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("docker pull failed (exit %d): image %q may not exist or registry is unreachable", exitErr.ExitCode(), imageRef)
		}
		return fmt.Errorf("failed to pull image %q: %w", imageRef, err)
	}

	fmt.Println("  ✓ Image pulled successfully")
	return nil
}

// pullWithMirrorFallback tries to pull the image from the primary registry,
// and if that fails, tries each mirror in order by replacing the registry prefix.
// Returns nil on success, or an error listing all attempted registries if all fail.
func pullWithMirrorFallback(imageRef string, mirrors []string) error {
	// Try primary first
	err := pullDestinationImage(imageRef)
	if err == nil {
		return nil
	}

	if len(mirrors) == 0 {
		return err
	}

	// Extract the image name+tag (everything after the first slash-delimited registry)
	// e.g. "ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1" → image suffix is "/cloudquery-plugin-file:v5.5.1"
	// Registry prefix is "ghcr.io/infobloxopen"
	primaryRegistry := extractRegistryPrefix(imageRef)
	imageSuffix := strings.TrimPrefix(imageRef, primaryRegistry)

	attempted := []string{imageRef}
	for _, mirror := range mirrors {
		mirrorRef := mirror + imageSuffix
		fmt.Printf("  Trying mirror: %s\n", mirrorRef)
		if mirrorErr := pullDestinationImage(mirrorRef); mirrorErr == nil {
			fmt.Printf("  ✓ Pulled from mirror: %s\n", mirror)
			return nil
		}
		attempted = append(attempted, mirrorRef)
	}

	return fmt.Errorf("failed to pull image from all registries: %s", strings.Join(attempted, ", "))
}

// extractRegistryPrefix extracts the registry/org prefix from an image reference.
// e.g. "ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1" → "ghcr.io/infobloxopen"
// e.g. "registry.io/cloudquery-plugin-file:v5.5.1" → "registry.io"
func extractRegistryPrefix(imageRef string) string {
	// Find the last component that looks like "cloudquery-plugin-*" or "*:<tag>"
	// Split by "/" and take everything before the image name
	parts := strings.Split(imageRef, "/")
	if len(parts) <= 1 {
		return ""
	}
	// The image name is the last part (before :tag)
	return strings.Join(parts[:len(parts)-1], "/")
}

// deployDestinationPod imports an image into k3d, deploys it as a pod, and starts
// port-forwarding. Returns the pod name, local forwarded port, cleanup function, and error.
func deployDestinationPod(imageRef, clusterName, namespace string) (string, int, func(), error) {
	podName := fmt.Sprintf("dp-dest-%s-%d", strings.ReplaceAll(
		strings.Split(filepath.Base(imageRef), ":")[0], "cloudquery-plugin-", ""),
		time.Now().Unix())

	cleanup := func() {
		cleanupDestinationPod(podName, namespace)
	}

	// Import image into k3d cluster
	fmt.Println("  Importing image to k3d cluster...")
	if err := importImageToK3d(imageRef, clusterName); err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to import image to k3d: %w", err)
	}

	// Delete any pre-existing pod with the same name (from a previous failed run)
	deletePod(fmt.Sprintf("k3d-%s", clusterName), namespace, podName)

	// Deploy as a pod
	fmt.Printf("  Deploying destination pod: %s\n", podName)
	runPodCmd := exec.Command("kubectl", "run", podName,
		"--image", imageRef,
		"--image-pull-policy=Never",
		"--port=7777",
		"--restart=Never",
		"-n", namespace,
	)
	runPodCmd.Stdout = os.Stdout
	runPodCmd.Stderr = os.Stderr
	if err := runPodCmd.Run(); err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to create destination pod: %w", err)
	}

	// Wait for pod to be ready
	fmt.Println("  Waiting for destination pod to be ready...")
	waitCmd := exec.Command("kubectl", "wait", "--for=condition=Ready",
		fmt.Sprintf("pod/%s", podName),
		"-n", namespace,
		"--timeout=120s",
	)
	waitCmd.Stdout = os.Stdout
	waitCmd.Stderr = os.Stderr
	if err := waitCmd.Run(); err != nil {
		return "", 0, cleanup, fmt.Errorf("destination pod did not become ready: %w", err)
	}

	// Find a free local port for port-forwarding
	destPort, err := findFreePort()
	if err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to find free port: %w", err)
	}

	// Start port-forwarding in background
	pfCmd := exec.Command("kubectl", "port-forward",
		fmt.Sprintf("pod/%s", podName),
		fmt.Sprintf("%d:7777", destPort),
		"-n", namespace,
	)
	pfCmd.Stdout = os.Stdout
	pfCmd.Stderr = os.Stderr
	if err := pfCmd.Start(); err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to start port-forward: %w", err)
	}

	// Enhance cleanup to also kill port-forward
	originalCleanup := cleanup
	cleanup = func() {
		if pfCmd.Process != nil {
			pfCmd.Process.Kill()
		}
		originalCleanup()
	}

	// Wait briefly for port-forward to establish
	time.Sleep(2 * time.Second)

	fmt.Printf("  ✓ Destination pod ready (forwarding localhost:%d → pod:7777)\n", destPort)
	return podName, destPort, cleanup, nil
}

// cleanupDestinationPod force-deletes a destination pod (best effort).
func cleanupDestinationPod(podName, namespace string) {
	cmd := exec.Command("kubectl", "delete", "pod", podName,
		"-n", namespace,
		"--force",
		"--grace-period=0",
	)
	cmd.Run() // best effort
}

// deployDestinationContainer runs a destination plugin as a local Docker container
// with a bind mount for file output. Returns container name, local port, cleanup function, and error.
// This is used for the file destination so output files appear directly on the host filesystem
// rather than being trapped inside a k3d pod (which uses a distroless image with no cp/tar).
func deployDestinationContainer(imageRef string) (string, int, func(), error) {
	containerName := fmt.Sprintf("dp-dest-%s-%d", strings.ReplaceAll(
		strings.Split(filepath.Base(imageRef), ":")[0], "cloudquery-plugin-", ""),
		time.Now().Unix())

	cleanup := func() {
		cleanupDestinationContainer(containerName)
	}

	// Find a free local port for the gRPC endpoint
	destPort, err := findFreePort()
	if err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to find free port: %w", err)
	}

	// Create output directory on host
	outputDir := filepath.Join(".", "cq-sync-output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get absolute path for bind mount (required by Docker)
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to resolve output directory path: %w", err)
	}

	// Run container with port mapping and bind mount.
	// The file plugin's working dir is /home/nonroot, so spec path "./cq-sync-output"
	// resolves to /home/nonroot/cq-sync-output inside the container.
	fmt.Printf("  Starting destination container: %s\n", containerName)
	runCmd := exec.Command("docker", "run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("%d:7777", destPort),
		"-v", fmt.Sprintf("%s:/home/nonroot/cq-sync-output", absOutputDir),
		imageRef,
	)
	runCmd.Stderr = os.Stderr
	if err := runCmd.Run(); err != nil {
		return "", 0, cleanup, fmt.Errorf("failed to start destination container: %w", err)
	}

	// Wait briefly for gRPC server to be ready
	time.Sleep(2 * time.Second)

	fmt.Printf("  ✓ Destination container ready (localhost:%d, output → %s)\n", destPort, absOutputDir)
	return containerName, destPort, cleanup, nil
}

// cleanupDestinationContainer stops and removes a destination container (best effort).
func cleanupDestinationContainer(containerName string) {
	cmd := exec.Command("docker", "rm", "-f", containerName)
	cmd.Run() // best effort
}

// findFreePort returns an available TCP port.
func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}

// generateSyncConfig creates a temporary CloudQuery sync configuration with
// both a source (gRPC, pointing to the running source plugin pod) and a destination
// (gRPC, pointing to the running destination plugin pod/container).
// The destination spec is resolved from: config overrides → in-cluster auto-detect → hardcoded defaults.
func generateSyncConfig(dp *contracts.DataPackage, sourcePort int, destName string, destPort int, cfg *localdev.Config, namespace string) (string, error) {
	tables := `["*"]`
	if dp.Spec.CloudQuery != nil && len(dp.Spec.CloudQuery.Tables) > 0 {
		tableList := make([]string, len(dp.Spec.CloudQuery.Tables))
		for i, t := range dp.Spec.CloudQuery.Tables {
			tableList[i] = fmt.Sprintf("%q", t)
		}
		tables = "[" + strings.Join(tableList, ", ") + "]"
	}

	destSpec := resolveDestinationSpec(destName, cfg, namespace)
	destOpts := defaultDestinationOpts(destName)

	config := fmt.Sprintf(`kind: source
spec:
  name: "%s"
  registry: grpc
  path: "localhost:%d"
  tables: %s
  destinations: ["%s"]
  spec: {}
---
kind: destination
spec:
  name: "%s"
  registry: grpc
  path: "localhost:%d"
%s  spec:
%s
`, dp.Metadata.Name, sourcePort, tables, destName,
		destName, destPort, destOpts, destSpec)

	tmpFile, err := os.CreateTemp("", "cq-sync-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create temp sync config: %w", err)
	}

	if _, err := tmpFile.WriteString(config); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write sync config: %w", err)
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// resolveDestinationSpec returns the YAML spec block for a destination plugin.
// Resolution order:
//  1. Config override (plugins.destinations.<name>.*) — if user has explicitly configured values
//  2. In-cluster auto-detect — discovers services in the k3d namespace (e.g., dp-postgres-postgres)
//  3. Hardcoded defaults — fallback values for local development
//
// For PostgreSQL, the connection string uses the in-cluster service DNS name since the
// destination plugin runs as a pod inside k3d and can reach the service directly.
// For S3, auto-detects LocalStack endpoint from the cluster.
func resolveDestinationSpec(name string, cfg *localdev.Config, namespace string) string {
	switch name {
	case "file":
		return resolveFileSpec(cfg)
	case "postgresql":
		return resolvePostgresqlSpec(cfg, namespace)
	case "s3":
		return resolveS3Spec(cfg, namespace)
	default:
		return "    {}"
	}
}

// resolveFileSpec returns the file destination spec.
// Config overrides: plugins.destinations.file.path
func resolveFileSpec(cfg *localdev.Config) string {
	path := "./cq-sync-output"
	if cfg != nil && cfg.Plugins.Destinations != nil {
		if dc, ok := cfg.Plugins.Destinations["file"]; ok && dc.Path != "" {
			path = dc.Path
		}
	}
	return fmt.Sprintf(`    path: "%s"
    format: "json"
    no_rotate: true`, path)
}

// resolvePostgresqlSpec returns the PostgreSQL destination spec.
// Resolution: config override → in-cluster auto-detect → hardcoded default.
func resolvePostgresqlSpec(cfg *localdev.Config, namespace string) string {
	// 1. Config override
	if cfg != nil && cfg.Plugins.Destinations != nil {
		if dc, ok := cfg.Plugins.Destinations["postgresql"]; ok && dc.ConnectionString != "" {
			return fmt.Sprintf(`    connection_string: "%s"`, dc.ConnectionString)
		}
	}

	// 2. Auto-detect from k3d cluster: discover postgresql service in namespace
	if connStr := detectPostgresqlService(namespace); connStr != "" {
		return fmt.Sprintf(`    connection_string: "%s"`, connStr)
	}

	// 3. Hardcoded fallback (host-accessible default)
	return `    connection_string: "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable"`
}

// resolveS3Spec returns the S3 destination spec.
// Resolution: config override → in-cluster auto-detect (LocalStack) → hardcoded default.
func resolveS3Spec(cfg *localdev.Config, namespace string) string {
	bucket := "dp-data"
	region := "us-east-1"
	path := "cq-sync/{{TABLE}}/{{UUID}}.json"
	endpoint := ""

	// 1. Config overrides
	if cfg != nil && cfg.Plugins.Destinations != nil {
		if dc, ok := cfg.Plugins.Destinations["s3"]; ok {
			if dc.Bucket != "" {
				bucket = dc.Bucket
			}
			if dc.Region != "" {
				region = dc.Region
			}
			if dc.Path != "" {
				path = dc.Path
			}
			if dc.Endpoint != "" {
				endpoint = dc.Endpoint
			}
		}
	}

	// 2. Auto-detect LocalStack endpoint if not configured
	if endpoint == "" {
		if ep := detectLocalStackService(namespace); ep != "" {
			endpoint = ep
		}
	}

	spec := fmt.Sprintf(`    bucket: "%s"
    region: "%s"
    path: "%s"
    format: "json"
    no_rotate: true`, bucket, region, path)

	if endpoint != "" {
		spec += fmt.Sprintf(`
    endpoint: "%s"
    force_path_style: true`, endpoint)
	}

	return spec
}

// detectPostgresqlService discovers a PostgreSQL service in the given k3d namespace.
// Returns a connection string using in-cluster DNS, or empty string if not found.
// The destination plugin runs as a pod inside k3d, so it can reach services via DNS.
func detectPostgresqlService(namespace string) string {
	// Look for a service with "postgres" in the name
	cmd := exec.Command("kubectl", "get", "svc", "-n", namespace,
		"-o", "jsonpath={.items[?(@.metadata.name=='dp-postgres-postgres')].metadata.name}")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	svcName := strings.TrimSpace(string(out))
	if svcName == "" {
		return ""
	}

	// Get credentials from the pod's environment (best effort)
	user := "postgres"
	password := "postgres"
	db := "postgres"

	// Build in-cluster connection string using service DNS
	// Format: <svc>.<namespace>.svc.cluster.local
	return fmt.Sprintf("postgresql://%s:%s@%s.%s.svc.cluster.local:5432/%s?sslmode=disable",
		user, password, svcName, namespace, db)
}

// detectLocalStackService discovers a LocalStack service in the given k3d namespace.
// Returns the in-cluster endpoint URL, or empty string if not found.
func detectLocalStackService(namespace string) string {
	cmd := exec.Command("kubectl", "get", "svc", "-n", namespace,
		"-o", "jsonpath={.items[?(@.metadata.name=='dp-localstack-localstack')].metadata.name}")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	svcName := strings.TrimSpace(string(out))
	if svcName == "" {
		return ""
	}
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:4566", svcName, namespace)
}

// defaultDestinationOpts returns top-level destination config options (siblings of
// name/registry/path) as indented YAML lines. These are CloudQuery destination-level
// settings, NOT plugin-specific spec fields.
func defaultDestinationOpts(name string) string {
	switch name {
	case "file":
		// File plugin doesn't implement DeleteStale, so append mode is required.
		return "  write_mode: \"append\"\n"
	default:
		return ""
	}
}

// defaultDestinationSpec returns the default YAML spec block for a destination.
// Each line is indented with 4 spaces to nest under the spec: key in the config.
func defaultDestinationSpec(name string) string {
	switch name {
	case "file":
		return `    path: "./cq-sync-output"
    format: "json"
    no_rotate: true`
	case "postgresql":
		return `    connection_string: "postgresql://postgres:postgres@localhost:5432/dp?sslmode=disable"`
	case "s3":
		return `    bucket: "dp-data"
    region: "us-east-1"
    path: "cq-sync/{{TABLE}}/{{UUID}}.json"
    format: "json"
    no_rotate: true`
	default:
		return "    {}"
	}
}

// runCloudQuerySync runs `cloudquery sync` with the given config file.
func runCloudQuerySync(configPath string) error {
	cmd := exec.Command("cloudquery", "sync", configPath,
		"--log-console",
		"--log-level", "info",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("cloudquery sync exited with code %d", exitErr.ExitCode())
		}
		return err
	}

	return nil
}
