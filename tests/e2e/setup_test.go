package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain is the global test entry point for E2E tests.
func TestMain(m *testing.M) {
	// Verify the dp binary exists before running tests
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		os.Exit(1)
	}

	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	binaryPath := filepath.Join(repoRoot, "bin", "dp")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Print a helpful message but don't fail - individual tests will skip
		println("Warning: dp binary not found at", binaryPath)
		println("Run 'make build' to build the CLI before running E2E tests")
	}

	os.Exit(m.Run())
}

// testdataDir returns the absolute path to the testdata directory.
func testdataDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller information")
	}

	testdataPath := filepath.Join(filepath.Dir(filename), "testdata")
	absPath, err := filepath.Abs(testdataPath)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Fatalf("testdata directory not found: %s", absPath)
	}

	return absPath
}

// validPipelinePath returns the path to the valid-pipeline fixture.
func validPipelinePath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdataDir(t), "valid-pipeline")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("valid-pipeline fixture not found: %s", path)
	}

	return path
}

// invalidPackagePath returns the path to the invalid-package fixture.
func invalidPackagePath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(testdataDir(t), "invalid-package")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("invalid-package fixture not found: %s", path)
	}

	return path
}
