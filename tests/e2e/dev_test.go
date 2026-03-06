package e2e

import (
	"os"
	"strings"
	"testing"
)

// TestDevUp_Status_Down exercises the full dk dev lifecycle:
//
//	dk dev up   → deploys all 4 Helm charts to k3d
//	dk dev status → verifies all charts report healthy
//	dk dev down → tears down all charts
//
// This test requires a running k3d cluster named "dk-local" and the
// dk binary built via `make build`. It is skipped in short mode and
// when the DK_E2E_DEV environment variable is not set.
func TestDevUp_Status_Down(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E dev test in short mode")
	}
	if os.Getenv("DK_E2E_DEV") == "" {
		t.Skip("set DK_E2E_DEV=1 to enable dev lifecycle E2E tests (requires k3d cluster)")
	}

	// --- dk dev up ---
	t.Run("dev_up", func(t *testing.T) {
		result, err := runDK(t, "dev", "up")
		if err != nil {
			t.Fatalf("dk dev up failed: %v\nstderr: %s", err, result.Stderr)
		}
		if result.ExitCode != 0 {
			t.Fatalf("dk dev up exited %d\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}
	})

	// --- dk dev status ---
	t.Run("dev_status", func(t *testing.T) {
		result, err := runDK(t, "dev", "status")
		if err != nil {
			t.Fatalf("dk dev status failed: %v\nstderr: %s", err, result.Stderr)
		}
		if result.ExitCode != 0 {
			t.Fatalf("dk dev status exited %d\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}

		// Verify all 4 charts appear in status output
		expectedCharts := []string{"redpanda", "localstack", "postgres", "marquez"}
		for _, chart := range expectedCharts {
			if !strings.Contains(strings.ToLower(result.Stdout), chart) {
				t.Errorf("dk dev status output missing chart %q:\n%s", chart, result.Stdout)
			}
		}
	})

	// --- dk dev down ---
	t.Run("dev_down", func(t *testing.T) {
		result, err := runDK(t, "dev", "down")
		if err != nil {
			t.Fatalf("dk dev down failed: %v\nstderr: %s", err, result.Stderr)
		}
		if result.ExitCode != 0 {
			t.Fatalf("dk dev down exited %d\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}
	})
}

// TestDevUp_Endpoints verifies that dk dev up prints the expected
// service endpoints. This is a lighter variant that only checks
// the output text — it does NOT require a running cluster.
func TestDevUp_Endpoints_Output(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E dev endpoint test in short mode")
	}
	if os.Getenv("DK_E2E_DEV") == "" {
		t.Skip("set DK_E2E_DEV=1 to enable dev lifecycle E2E tests (requires k3d cluster)")
	}

	result, err := runDK(t, "dev", "up")
	if err != nil {
		t.Fatalf("dk dev up failed: %v\nstderr: %s", err, result.Stderr)
	}

	// Check that key endpoints appear in the output
	expectedEndpoints := []string{
		"19092", // Kafka broker
		"4566",  // LocalStack S3
		"5432",  // PostgreSQL
		"5000",  // Marquez API
	}
	for _, ep := range expectedEndpoints {
		if !strings.Contains(result.Stdout, ep) {
			t.Errorf("dk dev up output missing endpoint port %q:\n%s", ep, result.Stdout)
		}
	}

	// Clean up
	_, _ = runDK(t, "dev", "down")
}
