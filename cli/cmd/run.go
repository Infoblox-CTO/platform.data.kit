package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Infoblox-CTO/data-platform/sdk/runner"
	"github.com/spf13/cobra"
)

var (
	runEnv      []string
	runBindings string
	runNetwork  string
	runTimeout  time.Duration
	runDryRun   bool
	runDetach   bool
)

// runCmd executes a pipeline locally
var runCmd = &cobra.Command{
	Use:   "run [package-dir]",
	Short: "Run a pipeline locally",
	Long: `Execute a data pipeline locally using Docker.

The run command builds (if needed) and executes the pipeline defined in
the specified package directory. It uses the Docker runtime to execute
the pipeline container.

The command will:
1. Parse dp.yaml and pipeline.yaml manifests
2. Build the Docker image if a Dockerfile exists
3. Start the container with configured environment and bindings
4. Stream logs to stdout

Prerequisites:
  - Docker must be running
  - Local dev stack should be running (dp dev up) for bindings

Examples:
  # Run pipeline in current directory
  dp run

  # Run pipeline in specific directory
  dp run ./my-pipeline

  # Run with custom environment variables
  dp run -e DEBUG=true -e LOG_LEVEL=debug

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
	runCmd.Flags().StringVarP(&runBindings, "bindings", "b", "", "Path to bindings file")
	runCmd.Flags().StringVar(&runNetwork, "network", "dp-network", "Docker network to connect to")
	runCmd.Flags().DurationVar(&runTimeout, "timeout", 30*time.Minute, "Timeout for pipeline execution")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "Validate and build only, don't execute")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Run in background")
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

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range runEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", e)
		}
		env[parts[0]] = parts[1]
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
		Network:      runNetwork,
		Timeout:      runTimeout,
		DryRun:       runDryRun,
		Detach:       runDetach,
		Output:       os.Stdout,
	}

	fmt.Printf("Running pipeline from: %s\n", packageDir)

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
		fmt.Println("\nUse these commands to manage the run:")
		fmt.Printf("  View logs: docker logs -f %s\n", result.RunID)
		fmt.Printf("  Stop:      docker stop %s\n", result.RunID)
	} else {
		switch result.Status {
		case "completed":
			fmt.Printf("✓ Pipeline completed successfully\n")
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
