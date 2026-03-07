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

// repoRootDir returns the absolute path to the repository root.
// It navigates from the test file location (tests/e2e/) up two directories.
func repoRootDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller information")
	}

	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	absPath, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatalf("failed to get absolute path for repo root: %v", err)
	}

	return absPath
}

// dkBinaryPath finds the dk CLI binary path.
// It looks for the binary in the bin directory at the root of the repository.
func dkBinaryPath(t *testing.T) string {
	t.Helper()

	binaryPath := filepath.Join(repoRootDir(t), "bin", "dk")

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("dk binary not found at %s. Run 'make build' first.", binaryPath)
	}

	return binaryPath
}

// demoRunnerPath returns the absolute path to the demo runner script (demos/run_demo.sh).
// It verifies the script exists before returning.
func demoRunnerPath(t *testing.T) string {
	t.Helper()

	runnerPath := filepath.Join(repoRootDir(t), "demos", "run_demo.sh")

	if _, err := os.Stat(runnerPath); os.IsNotExist(err) {
		t.Fatalf("demo runner not found at %s", runnerPath)
	}

	return runnerPath
}

// runDemo executes a demo dialog file through the runner script.
// It sets the working directory to the repo root and prepends the bin/ directory
// to PATH so that dk commands work without absolute paths.
func runDemo(t *testing.T, demoName string) *CommandResult {
	t.Helper()

	root := repoRootDir(t)
	runner := demoRunnerPath(t)
	dialogFile := filepath.Join("demos", demoName, "demo.txt")

	cmd := exec.Command("bash", runner, dialogFile)
	cmd.Dir = root

	// Prepend bin/ to PATH so dk is available
	binDir := filepath.Join(root, "bin")
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))

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
			t.Fatalf("failed to run demo %q: %v", demoName, err)
		}
	}

	return result
}

// CommandResult holds the result of running a command.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// runDK executes the dk command with the given arguments and returns the result.
func runDK(t *testing.T, args ...string) (*CommandResult, error) {
	t.Helper()
	return runDKInDir(t, "", args...)
}

// runDKInDir executes the dk command in a specific directory with the given arguments.
func runDKInDir(t *testing.T, dir string, args ...string) (*CommandResult, error) {
	t.Helper()

	binaryPath := dkBinaryPath(t)

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
