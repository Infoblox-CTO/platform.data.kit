package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkflow_InitLintBuild(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Step 1: Initialize a new package
	t.Run("init", func(t *testing.T) {
		result, err := runDPInDir(t, tmpDir, "init", "--type", "pipeline", "test-workflow")
		if err != nil {
			t.Fatalf("init failed: %v", err)
		}

		if result.ExitCode != 0 {
			t.Fatalf("init returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
		}

		assertFileExists(t, filepath.Join(tmpDir, "test-workflow", "dp.yaml"))
	})

	pkgDir := filepath.Join(tmpDir, "test-workflow")

	// Step 2: Lint the package
	t.Run("lint", func(t *testing.T) {
		result, err := runDP(t, "lint", pkgDir)
		if err != nil {
			t.Fatalf("lint failed: %v", err)
		}

		if result.ExitCode != 0 {
			t.Fatalf("lint returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})

	// Step 3: Build the package (dry-run)
	t.Run("build", func(t *testing.T) {
		result, err := runDP(t, "build", "--dry-run", pkgDir)
		if err != nil {
			t.Fatalf("build failed: %v", err)
		}

		if result.ExitCode != 0 {
			t.Fatalf("build returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})
}

func TestWorkflow_AllPackageTypes(t *testing.T) {
	skipIfShort(t)

	packageTypes := []string{"pipeline", "dataset", "model"}

	for _, pkgType := range packageTypes {
		t.Run(pkgType, func(t *testing.T) {
			tmpDir := createTempDir(t)
			pkgName := "test-" + pkgType

			// Init
			result, err := runDPInDir(t, tmpDir, "init", "--type", pkgType, pkgName)
			if err != nil {
				t.Fatalf("init failed for type %s: %v", pkgType, err)
			}

			if result.ExitCode != 0 {
				t.Fatalf("init returned non-zero exit code for type %s: %d\nstderr: %s",
					pkgType, result.ExitCode, result.Stderr)
			}

			pkgDir := filepath.Join(tmpDir, pkgName)
			assertFileExists(t, filepath.Join(pkgDir, "dp.yaml"))
			assertFileContains(t, filepath.Join(pkgDir, "dp.yaml"), "type: "+pkgType)

			// Lint
			result, err = runDP(t, "lint", pkgDir)
			if err != nil {
				t.Fatalf("lint failed for type %s: %v", pkgType, err)
			}

			if result.ExitCode != 0 {
				t.Fatalf("lint returned non-zero exit code for type %s: %d\nstderr: %s",
					pkgType, result.ExitCode, result.Stderr)
			}

			// Build (dry-run)
			result, err = runDP(t, "build", "--dry-run", pkgDir)
			if err != nil {
				t.Fatalf("build failed for type %s: %v", pkgType, err)
			}

			if result.ExitCode != 0 {
				t.Fatalf("build returned non-zero exit code for type %s: %d\nstderr: %s",
					pkgType, result.ExitCode, result.Stderr)
			}
		})
	}
}

func TestWorkflow_Version(t *testing.T) {
	skipIfShort(t)

	result, err := runDP(t, "version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	// Version output should contain some version information
	output := result.Stdout + result.Stderr
	if output == "" {
		t.Error("expected version output, got empty string")
	}
}

func TestWorkflow_Help(t *testing.T) {
	skipIfShort(t)

	result, err := runDP(t, "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "Usage:") && !strings.Contains(output, "usage:") {
		t.Error("expected help output to contain 'Usage:'")
	}
}

func TestWorkflow_CommandHelp(t *testing.T) {
	skipIfShort(t)

	commands := []string{
		"init",
		"lint",
		"build",
		"run",
		"test",
		"logs",
		"status",
		"promote",
		"publish",
	}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			result, err := runDP(t, cmd, "--help")
			if err != nil {
				t.Fatalf("unexpected error for '%s --help': %v", cmd, err)
			}

			if result.ExitCode != 0 {
				t.Errorf("expected exit code 0 for '%s --help', got %d\nstderr: %s",
					cmd, result.ExitCode, result.Stderr)
			}

			output := result.Stdout + result.Stderr
			if output == "" {
				t.Errorf("expected help output for '%s', got empty string", cmd)
			}

			// Help output should contain the command name or usage information
			if !strings.Contains(strings.ToLower(output), cmd) &&
				!strings.Contains(strings.ToLower(output), "usage") {
				t.Errorf("expected help output for '%s' to contain command name or usage", cmd)
			}
		})
	}
}
