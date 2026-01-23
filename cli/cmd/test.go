package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
)

var (
	testData     string
	testTimeout  time.Duration
	testBindings string
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
	testCmd.Flags().DurationVar(&testTimeout, "timeout", 5*time.Minute, "Timeout for test execution")
	testCmd.Flags().StringVarP(&testBindings, "bindings", "b", "", "Path to test bindings file")
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

	fmt.Printf("Running tests for: %s\n\n", packageDir)

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

	// Run the pipeline in test mode
	opts := runner.RunOptions{
		PackageDir:   absDir,
		Env:          env,
		BindingsFile: bindingsPath,
		Network:      "dp-network",
		Timeout:      testTimeout,
		DryRun:       false,
		Detach:       false,
		Output:       os.Stdout,
	}

	ctx := context.Background()
	result, err := dockerRunner.Run(ctx, opts)

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
