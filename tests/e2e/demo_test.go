package e2e

import (
	"os"
	"strings"
	"testing"
)

// TestDemo_Quickstart runs the quickstart demo dialog file through the
// runner script and verifies all commands succeed. This demo requires
// Docker and k3d for the dev environment, build, and run steps.
//
// It is skipped in short mode and when DP_E2E_DEV is not set.
func TestDemo_Quickstart(t *testing.T) {
	skipIfShort(t)
	skipIfNoDocker(t)

	if os.Getenv("DP_E2E_DEV") == "" {
		t.Skip("set DP_E2E_DEV=1 to enable quickstart demo test (requires k3d)")
	}

	result := runDemo(t, "quickstart")

	if result.ExitCode != 0 {
		t.Fatalf("quickstart demo failed with exit code %d\nstdout:\n%s\nstderr:\n%s",
			result.ExitCode, result.Stdout, result.Stderr)
	}

	// Verify narration text appears in output
	expectedPhrases := []string{
		"Welcome to the Data Platform quickstart demo",
		"create a new data package",
		"validate the package configuration",
		"inspect the package metadata",
		"start the local development environment",
		"build the package",
		"run the pipeline",
		"tearing down the dev environment",
	}
	for _, phrase := range expectedPhrases {
		if !strings.Contains(result.Stdout, phrase) {
			t.Errorf("expected stdout to contain %q, but it didn't\nstdout:\n%s", phrase, result.Stdout)
		}
	}

	// Verify pipeline actually executed and produced output
	if !strings.Contains(result.Stdout, "Hello from quickstart-demo model!") {
		t.Errorf("expected pipeline output in stdout\nstdout:\n%s", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "Pipeline completed successfully") {
		t.Errorf("expected pipeline completion message in stdout\nstdout:\n%s", result.Stdout)
	}
}

// TestDemo_DevLifecycle runs the dev-lifecycle demo dialog file through
// the runner script. This demo requires a running k3d cluster and the
// DP_E2E_DEV environment variable to be set.
//
// It is skipped in short mode and when DP_E2E_DEV is not set.
func TestDemo_DevLifecycle(t *testing.T) {
	skipIfShort(t)

	if os.Getenv("DP_E2E_DEV") == "" {
		t.Skip("set DP_E2E_DEV=1 to enable dev lifecycle demo test (requires k3d cluster)")
	}

	result := runDemo(t, "dev-lifecycle")

	if result.ExitCode != 0 {
		t.Fatalf("dev-lifecycle demo failed with exit code %d\nstdout:\n%s\nstderr:\n%s",
			result.ExitCode, result.Stdout, result.Stderr)
	}
}
