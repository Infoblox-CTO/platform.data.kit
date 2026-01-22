// Package testutil provides shared test utilities for CLI testing.
package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory for testing and returns a cleanup function.
// The directory is automatically removed when cleanup is called.
func TempDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	return dir, func() {
		// t.TempDir() handles cleanup automatically
	}
}

// CaptureOutput captures stdout and stderr during test execution.
type CaptureOutput struct {
	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}

// NewCaptureOutput creates a new output capture.
func NewCaptureOutput() *CaptureOutput {
	return &CaptureOutput{
		Stdout: new(bytes.Buffer),
		Stderr: new(bytes.Buffer),
	}
}

// WriteFile creates a file with the given content in the specified directory.
func WriteFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}

// ReadFile reads a file and returns its content.
func ReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	return string(data)
}

// FileExists checks if a file exists.
func FileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

// AssertFileExists asserts that a file exists.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if !FileExists(t, path) {
		t.Errorf("expected file to exist: %s", path)
	}
}

// AssertFileNotExists asserts that a file does not exist.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if FileExists(t, path) {
		t.Errorf("expected file to not exist: %s", path)
	}
}

// AssertContains asserts that a string contains a substring.
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !bytes.Contains([]byte(s), []byte(substr)) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}

// LoadTestdata loads a file from the testdata directory.
func LoadTestdata(t *testing.T, filename string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to load testdata %s: %v", filename, err)
	}
	return data
}
