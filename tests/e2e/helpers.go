package e2e

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// dpBinaryPath finds the dp CLI binary path.
// It looks for the binary in the bin directory at the root of the repository.
func dpBinaryPath(t *testing.T) string {
	t.Helper()

	// Get the path to the repository root
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller information")
	}

	// Navigate from tests/e2e to repository root
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	binaryPath := filepath.Join(repoRoot, "bin", "dp")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("dp binary not found at %s. Run 'make build' first.", binaryPath)
	}

	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	return absPath
}

// CommandResult holds the result of running a command.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runDP executes the dp command with the given arguments and returns the result.
func runDP(t *testing.T, args ...string) (*CommandResult, error) {
	t.Helper()
	return runDPInDir(t, "", args...)
}

// runDPInDir executes the dp command in a specific directory with the given arguments.
func runDPInDir(t *testing.T, dir string, args ...string) (*CommandResult, error) {
	t.Helper()

	binaryPath := dpBinaryPath(t)

	cmd := exec.Command(binaryPath, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, err
		}
	}

	return result, nil
}

// createTempDir creates a temporary directory for testing.
func createTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// copyDir recursively copies a directory from src to dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	srcInfo, err := os.Stat(src)
	if err != nil {
		t.Fatalf("failed to stat source directory: %v", err)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		t.Fatalf("failed to create destination directory: %v", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("failed to read source directory: %v", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
		} else {
			copyFile(t, srcPath, dstPath)
		}
	}
}

// copyFile copies a single file from src to dst.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()

	srcFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("failed to open source file: %v", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		t.Fatalf("failed to stat source file: %v", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		t.Fatalf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}
}

// assertFileExists checks that a file exists at the given path.
func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}

// assertFileContains checks that a file contains the given substring.
func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}

	if !strings.Contains(string(content), substr) {
		t.Errorf("file %s does not contain expected substring: %q", path, substr)
	}
}

// skipIfShort skips the test if the -short flag is provided.
func skipIfShort(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
}

// skipIfNoDocker skips the test if docker is not available.
func skipIfNoDocker(t *testing.T) {
	t.Helper()

	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("skipping test: docker is not available")
	}
}
