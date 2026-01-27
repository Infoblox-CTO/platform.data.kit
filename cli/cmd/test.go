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
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
)

var (
	testData           string
	testTimeout        time.Duration
	testBindings       string
	testDuration       time.Duration // For streaming: how long to run the test
	testStartupTimeout time.Duration // For streaming: how long to wait for healthy
)

var testCmd = &cobra.Command{
	Use:   "test [package-dir]",
	Short: "Run tests for a DP package",
	Long: `Run tests for a DP data package in a local environment.

This command runs the package's pipeline with test data against
the local development environment (Docker Compose).`,
	Example: `  # Run tests in current package
  dp test
  
  # Run tests with specific test data
  dp test --data ./test/sample.json
  
  # Run with timeout
  dp test --timeout 5m`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().StringVar(&testData, "data", "", "Path to test data file")
	testCmd.Flags().DurationVar(&testTimeout, "timeout", 5*time.Minute, "Timeout for test execution (batch pipelines)")
	testCmd.Flags().StringVarP(&testBindings, "bindings", "b", "", "Path to test bindings file")
	testCmd.Flags().DurationVar(&testDuration, "duration", 30*time.Second, "How long to run streaming test before shutdown")
	testCmd.Flags().DurationVar(&testStartupTimeout, "startup-timeout", 60*time.Second, "How long to wait for streaming pipeline to become healthy")
}

// ensureNetworkExists creates the Docker network if it doesn't exist.
func ensureNetworkExists(networkName string) error {
	// Check if network exists
	checkCmd := exec.Command("docker", "network", "inspect", networkName)
	if err := checkCmd.Run(); err == nil {
		// Network exists
		return nil
	}

	// Create the network
	createCmd := exec.Command("docker", "network", "create", networkName)
	return createCmd.Run()
}

func runTest(cmd *cobra.Command, args []string) error {
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

	// Detect pipeline mode from pipeline.yaml
	pipelineMode := contracts.PipelineModeBatch // default
	pipelinePath := filepath.Join(absDir, "pipeline.yaml")
	if _, err := os.Stat(pipelinePath); err == nil {
		pipeline, err := manifest.ParsePipelineFile(pipelinePath)
		if err == nil && pipeline.Spec.Mode != "" {
			pipelineMode = pipeline.Spec.Mode
		}
	}

	fmt.Printf("Running tests for: %s\n", packageDir)
	fmt.Printf("Pipeline mode: %s\n\n", pipelineMode.Default())

	// Find test data
	testDataPath := testData
	if testDataPath == "" {
		// Look for test data in standard locations
		testPaths := []string{
			filepath.Join(absDir, "test", "data"),
			filepath.Join(absDir, "test", "input"),
			filepath.Join(absDir, "testdata"),
		}
		for _, p := range testPaths {
			if _, err := os.Stat(p); err == nil {
				testDataPath = p
				break
			}
		}
	}

	if testDataPath != "" {
		fmt.Printf("Using test data from: %s\n", testDataPath)
	} else {
		fmt.Println("No test data found. Running pipeline with test topics.")
	}

	// Create runner
	dockerRunner, err := runner.NewDockerRunner()
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Set up test environment
	env := map[string]string{
		"DP_TEST_MODE":  "true",
		"DP_INPUT_TYPE": "test",
	}

	// Use test bindings if available
	bindingsPath := testBindings
	if bindingsPath == "" {
		// Look for test bindings
		testBindingsPaths := []string{
			filepath.Join(absDir, "test", "bindings.yaml"),
			filepath.Join(absDir, "bindings.test.yaml"),
		}
		for _, p := range testBindingsPaths {
			if _, err := os.Stat(p); err == nil {
				bindingsPath = p
				break
			}
		}
	}

	// Look for local dev bindings as fallback
	if bindingsPath == "" {
		localBindings, _ := findComposeFile()
		if localBindings != "" {
			localBindingsPath := filepath.Join(filepath.Dir(localBindings), "bindings.local.yaml")
			if _, err := os.Stat(localBindingsPath); err == nil {
				bindingsPath = localBindingsPath
			}
		}
	}

	if bindingsPath != "" {
		fmt.Printf("Using bindings from: %s\n", bindingsPath)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("Test Execution")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	// Ensure the Docker network exists
	if err := ensureNetworkExists("dp-network"); err != nil {
		fmt.Printf("Warning: Could not create network: %v\n", err)
	}

	// Run the pipeline in test mode - behavior depends on pipeline mode
	timeout := testTimeout
	if pipelineMode == contracts.PipelineModeStreaming {
		// For streaming, use the duration instead of timeout
		timeout = testDuration
		fmt.Printf("Streaming test: will run for %s\n", testDuration)
		if testStartupTimeout > 0 {
			fmt.Printf("Startup timeout: %s\n", testStartupTimeout)
		}
		fmt.Println()
	}

	opts := runner.RunOptions{
		PackageDir:   absDir,
		Env:          env,
		BindingsFile: bindingsPath,
		Network:      "dp-network",
		Timeout:      timeout,
		DryRun:       false,
		Detach:       false,
		Output:       os.Stdout,
		Mode:         pipelineMode,
	}

	ctx := context.Background()

	var result *runner.RunResult
	if pipelineMode == contracts.PipelineModeStreaming {
		// For streaming: start, wait for duration, then stop
		result, err = runStreamingTest(ctx, dockerRunner, opts)
	} else {
		// For batch: run to completion
		result, err = dockerRunner.Run(ctx, opts)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("Test Results")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	if err != nil {
		fmt.Println("✗ Tests FAILED")
		fmt.Printf("  Error: %v\n", err)
		return fmt.Errorf("tests failed")
	}

	switch result.Status {
	case "completed":
		fmt.Println("✓ Tests PASSED")
		fmt.Printf("  Duration: %s\n", result.Duration.Round(time.Millisecond))
		if result.RecordsProcessed > 0 {
			fmt.Printf("  Records processed: %d\n", result.RecordsProcessed)
		}
	case "failed":
		fmt.Println("✗ Tests FAILED")
		fmt.Printf("  Exit code: %d\n", result.ExitCode)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		return fmt.Errorf("tests failed with exit code %d", result.ExitCode)
	default:
		fmt.Printf("Tests ended with status: %s\n", result.Status)
	}

	return nil
}

// runStreamingTest runs a streaming pipeline test.
// It starts the pipeline, waits for the specified duration, then stops it gracefully.
func runStreamingTest(ctx context.Context, r runner.Runner, opts runner.RunOptions) (*runner.RunResult, error) {
	// For streaming tests, we run detached and then stop after duration
	streamingOpts := opts
	streamingOpts.Detach = true

	// Start the pipeline
	result, err := r.Run(ctx, streamingOpts)
	if err != nil {
		return result, err
	}

	fmt.Printf("Started streaming pipeline (container: %s)\n", result.ContainerID)
	fmt.Printf("Running for %s...\n\n", opts.Timeout)

	// Stream logs in background while waiting
	logCtx, logCancel := context.WithCancel(ctx)
	defer logCancel()

	go func() {
		if err := r.Logs(logCtx, result.RunID, true, opts.Output); err != nil {
			if logCtx.Err() == nil {
				fmt.Fprintf(opts.Output, "Log streaming error: %v\n", err)
			}
		}
	}()

	// Wait for the specified duration
	select {
	case <-time.After(opts.Timeout):
		fmt.Println("\nTest duration reached, stopping pipeline...")
	case <-ctx.Done():
		fmt.Println("\nTest cancelled, stopping pipeline...")
	}

	// Stop the pipeline gracefully
	logCancel() // Stop log streaming
	if err := r.Stop(ctx, result.RunID); err != nil {
		fmt.Printf("Warning: failed to stop pipeline: %v\n", err)
	}

	// Update result
	result.Status = contracts.RunStatusCompleted
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(result.StartTime)

	return result, nil
}
