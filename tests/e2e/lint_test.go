package e2e

import (
	"path/filepath"
	"testing"
)

func TestLint_ValidPackage(t *testing.T) {
	skipIfShort(t)

	validPkg := validPipelinePath(t)

	result, err := runDK(t, "lint", validPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}
}

func TestLint_InvalidPackage(t *testing.T) {
	skipIfShort(t)

	// Copy invalid-package to temp dir since tests might modify it
	tmpDir := createTempDir(t)
	invalidPkg := filepath.Join(tmpDir, "invalid-package")
	copyDir(t, invalidPackagePath(t), invalidPkg)

	result, err := runDK(t, "lint", invalidPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid package")
	}
}

func TestLint_NonExistentDirectory(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	result, err := runDK(t, "lint", nonExistent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for non-existent directory")
	}
}

func TestLint_StrictMode(t *testing.T) {
	skipIfShort(t)

	validPkg := validPipelinePath(t)

	result, err := runDK(t, "lint", "--strict", validPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Valid package should pass even in strict mode
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0 for valid package in strict mode, got %d\nstderr: %s",
			result.ExitCode, result.Stderr)
	}
}

func TestLint_SkipPII(t *testing.T) {
	skipIfShort(t)

	validPkg := validPipelinePath(t)

	result, err := runDK(t, "lint", "--skip-pii", validPkg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}
}

func TestLint_WithPathArgument(t *testing.T) {
	skipIfShort(t)

	t.Run("absolute path", func(t *testing.T) {
		validPkg := validPipelinePath(t)

		result, err := runDK(t, "lint", validPkg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})

	t.Run("relative path from package dir", func(t *testing.T) {
		validPkg := validPipelinePath(t)

		// Run lint in the package directory with current directory
		result, err := runDKInDir(t, validPkg, "lint", ".")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})

	t.Run("no path argument uses current directory", func(t *testing.T) {
		validPkg := validPipelinePath(t)

		result, err := runDKInDir(t, validPkg, "lint")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})
}
