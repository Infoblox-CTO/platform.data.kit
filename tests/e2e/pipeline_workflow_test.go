package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineWorkflow_CreateAndShow(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Step 1: Initialize a package
	t.Run("init", func(t *testing.T) {
		result, err := runDKInDir(t, tmpDir, "init", "--runtime", "generic-go", "test-pipeline-wf")
		if err != nil {
			t.Fatalf("init failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("init returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
	})

	pkgDir := filepath.Join(tmpDir, "test-pipeline-wf")

	// Step 2: Create a pipeline workflow (use --force since dk init already creates pipeline.yaml)
	t.Run("pipeline_create", func(t *testing.T) {
		result, err := runDKInDir(t, pkgDir, "pipeline", "create", "my-workflow", "--force")
		if err != nil {
			t.Fatalf("pipeline create failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("pipeline create returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
		assertFileExists(t, filepath.Join(pkgDir, "pipeline.yaml"))
		assertFileContains(t, filepath.Join(pkgDir, "pipeline.yaml"), "name: my-workflow")
	})

	// Step 3: Show the pipeline
	t.Run("pipeline_show", func(t *testing.T) {
		result, err := runDKInDir(t, pkgDir, "pipeline", "show")
		if err != nil {
			t.Fatalf("pipeline show failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("pipeline show returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
		output := result.Stdout + result.Stderr
		if !strings.Contains(output, "my-workflow") {
			t.Errorf("pipeline show output should contain pipeline name, got: %s", output)
		}
	})

	// Step 4: Show as JSON
	t.Run("pipeline_show_json", func(t *testing.T) {
		result, err := runDKInDir(t, pkgDir, "pipeline", "show", "--output", "json")
		if err != nil {
			t.Fatalf("pipeline show --output json failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("pipeline show --output json returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
		}
		output := result.Stdout + result.Stderr
		if !strings.Contains(output, "\"name\"") {
			t.Errorf("pipeline show JSON output should contain name field, got: %s", output)
		}
	})
}

func TestPipelineWorkflow_ListTemplates(t *testing.T) {
	skipIfShort(t)

	result, err := runDK(t, "pipeline", "create", "--list-templates")
	if err != nil {
		t.Fatalf("pipeline create --list-templates failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
	}
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "sync-transform-test") {
		t.Errorf("expected template list to contain sync-transform-test, got: %s", output)
	}
}

func TestPipelineWorkflow_CreateWithTemplates(t *testing.T) {
	skipIfShort(t)

	templates := []string{"sync-transform-test", "sync-only", "custom"}

	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			tmpDir := createTempDir(t)

			result, err := runDKInDir(t, tmpDir, "pipeline", "create", "test-"+tmpl, "--template", tmpl)
			if err != nil {
				t.Fatalf("pipeline create --template %s failed: %v", tmpl, err)
			}
			if result.ExitCode != 0 {
				t.Fatalf("returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
			}

			pipelinePath := filepath.Join(tmpDir, "pipeline.yaml")
			assertFileExists(t, pipelinePath)
			assertFileContains(t, pipelinePath, "name: test-"+tmpl)
		})
	}
}

func TestPipelineWorkflow_CreateForceOverwrite(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Create first
	result, err := runDKInDir(t, tmpDir, "pipeline", "create", "original")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("first create returned non-zero exit code: %d", result.ExitCode)
	}

	// Create again without --force should fail
	result, err = runDKInDir(t, tmpDir, "pipeline", "create", "overwritten")
	if err == nil && result.ExitCode == 0 {
		t.Fatal("expected error when creating pipeline without --force, but succeeded")
	}

	// Create again with --force should succeed
	result, err = runDKInDir(t, tmpDir, "pipeline", "create", "overwritten", "--force")
	if err != nil {
		t.Fatalf("create --force failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("create --force returned non-zero exit code: %d\nstderr: %s", result.ExitCode, result.Stderr)
	}

	assertFileContains(t, filepath.Join(tmpDir, "pipeline.yaml"), "name: overwritten")
}

func TestPipelineWorkflow_BackfillMissingFlags(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Backfill without --from and --to should error
	result, _ := runDKInDir(t, tmpDir, "pipeline", "backfill")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code when backfill called without required flags")
	}
}

func TestPipelineWorkflow_RunNoPipeline(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Run in a directory with no pipeline.yaml
	result, _ := runDKInDir(t, tmpDir, "pipeline", "run")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code when running in directory without pipeline.yaml")
	}
}

func TestPipelineWorkflow_ShowNoPipeline(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	// Show in a directory with no pipeline.yaml
	result, _ := runDKInDir(t, tmpDir, "pipeline", "show")
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code when showing in directory without pipeline.yaml")
	}
}
