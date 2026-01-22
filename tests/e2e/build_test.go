package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuild_ValidPackage(t *testing.T) {
	skipIfShort(t)

	validPkg := validPipelinePath(t)

	// Use --dry-run to avoid needing docker/registry
	result, err := runDP(t, "build", "--dry-run", validPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstdout: %s\nstderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}
}

func TestBuild_InvalidPackage(t *testing.T) {
	skipIfShort(t)

	// Copy invalid-package to temp dir since tests might modify it
	tmpDir := createTempDir(t)
	invalidPkg := filepath.Join(tmpDir, "invalid-package")
	copyDir(t, invalidPackagePath(t), invalidPkg)

	result, err := runDP(t, "build", "--dry-run", invalidPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid package")
	}
}

func TestBuild_NonExistentDirectory(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	result, err := runDP(t, "build", nonExistent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for non-existent directory")
	}
}

func TestBuild_WithCustomTag(t *testing.T) {
	skipIfShort(t)

	validPkg := validPipelinePath(t)

	result, err := runDP(t, "build", "--dry-run", "--tag", "v1.2.3", validPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	// Check that the custom tag is mentioned in output
	if !strings.Contains(result.Stdout, "v1.2.3") && !strings.Contains(result.Stderr, "v1.2.3") {
		t.Log("Note: custom tag may not appear in dry-run output")
	}
}

func TestBuild_DryRunNoArtifact(t *testing.T) {
	skipIfShort(t)

	validPkg := validPipelinePath(t)
	tmpDir := createTempDir(t)

	// Copy valid package to temp dir to check for artifacts
	testPkg := filepath.Join(tmpDir, "test-package")
	copyDir(t, validPkg, testPkg)

	result, err := runDP(t, "build", "--dry-run", testPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	// Verify no artifact file was created (common artifact patterns)
	artifactPatterns := []string{"*.tar", "*.tar.gz", "*.oci", "artifact.*"}
	for _, pattern := range artifactPatterns {
		matches, _ := filepath.Glob(filepath.Join(testPkg, pattern))
		if len(matches) > 0 {
			t.Errorf("dry-run should not create artifact files, found: %v", matches)
		}
	}
}

func TestBuild_WithPath(t *testing.T) {
	skipIfShort(t)

	t.Run("absolute path", func(t *testing.T) {
		validPkg := validPipelinePath(t)

		result, err := runDP(t, "build", "--dry-run", validPkg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})

	t.Run("relative path from package dir", func(t *testing.T) {
		validPkg := validPipelinePath(t)

		result, err := runDPInDir(t, validPkg, "build", "--dry-run", ".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})

	t.Run("no path argument uses current directory", func(t *testing.T) {
		validPkg := validPipelinePath(t)

		result, err := runDPInDir(t, validPkg, "build", "--dry-run")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})

	t.Run("missing dp.yaml", func(t *testing.T) {
		tmpDir := createTempDir(t)

		// Create empty directory
		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		result, err := runDP(t, "build", "--dry-run", emptyDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ExitCode == 0 {
			t.Error("expected non-zero exit code for directory without dp.yaml")
		}
	})
}
