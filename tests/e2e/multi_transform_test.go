package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// multiTransformPath returns the path to the multi-transform test fixture.
func multiTransformPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(testdataDir(t), "multi-transform")
	assertFileExists(t, path)
	return path
}

// TestMultiTransform_PipelineShow tests that dk pipeline show --scan-dir
// discovers transforms and datasets from a multi-transform project and
// renders the dependency graph correctly.
func TestMultiTransform_PipelineShow(t *testing.T) {
	skipIfShort(t)

	fixturePath := multiTransformPath(t)

	// Test text output (default)
	t.Run("text", func(t *testing.T) {
		result, err := runDK(t, "pipeline", "show", "--scan-dir", fixturePath)
		if err != nil {
			t.Fatalf("pipeline show failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("pipeline show exited %d:\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}

		output := result.Stdout + result.Stderr

		// Verify all transforms appear
		for _, name := range []string{"extract", "transform-etl", "load-report"} {
			if !strings.Contains(output, name) {
				t.Errorf("expected pipeline graph to contain transform %q, got:\n%s", name, output)
			}
		}

		// Verify all datasets appear
		for _, name := range []string{"raw-events", "processed-events", "final-report", "published-report"} {
			if !strings.Contains(output, name) {
				t.Errorf("expected pipeline graph to contain dataset %q, got:\n%s", name, output)
			}
		}

		// Verify runtime info
		if !strings.Contains(output, "runtime=go") {
			t.Errorf("expected pipeline graph to show runtime=go, got:\n%s", output)
		}

		// Verify trigger info
		if !strings.Contains(output, "schedule") {
			t.Errorf("expected pipeline graph to show schedule trigger, got:\n%s", output)
		}
		if !strings.Contains(output, "on-change") {
			t.Errorf("expected pipeline graph to show on-change trigger, got:\n%s", output)
		}
	})

	// Test JSON output
	t.Run("json", func(t *testing.T) {
		result, err := runDK(t, "pipeline", "show", "--scan-dir", fixturePath, "--output", "json")
		if err != nil {
			t.Fatalf("pipeline show --output json failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("pipeline show --output json exited %d:\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}

		output := result.Stdout
		if output == "" {
			output = result.Stderr
		}

		// Verify it's valid JSON with nodes and edges
		if !strings.Contains(output, `"nodes"`) {
			t.Errorf("expected JSON output to contain 'nodes', got:\n%s", output)
		}
		if !strings.Contains(output, `"edges"`) {
			t.Errorf("expected JSON output to contain 'edges', got:\n%s", output)
		}

		// Verify all transforms in JSON
		for _, name := range []string{"extract", "transform-etl", "load-report"} {
			if !strings.Contains(output, name) {
				t.Errorf("expected JSON output to contain transform %q", name)
			}
		}
	})

	// Test destination filtering
	t.Run("destination-filter", func(t *testing.T) {
		result, err := runDK(t, "pipeline", "show", "--scan-dir", fixturePath, "--destination", "final-report")
		if err != nil {
			t.Fatalf("pipeline show --destination failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("pipeline show --destination exited %d:\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}

		output := result.Stdout + result.Stderr

		// Should include the chain up to final-report
		if !strings.Contains(output, "extract") {
			t.Errorf("expected filtered graph to include 'extract'")
		}
		if !strings.Contains(output, "transform-etl") {
			t.Errorf("expected filtered graph to include 'transform-etl'")
		}
		if !strings.Contains(output, "final-report") {
			t.Errorf("expected filtered graph to include 'final-report'")
		}

		// Should NOT include load-report (downstream of final-report)
		if strings.Contains(output, "load-report") {
			t.Errorf("expected filtered graph to exclude 'load-report' (downstream of destination)")
		}
	})
}

// TestMultiTransform_Lint tests that dk lint --scan-dir validates all
// transforms in a multi-transform project.
func TestMultiTransform_Lint(t *testing.T) {
	skipIfShort(t)

	fixturePath := multiTransformPath(t)

	result, err := runDK(t, "lint", "--scan-dir", fixturePath)
	if err != nil {
		t.Fatalf("lint --scan-dir failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("lint --scan-dir exited %d:\nstdout: %s\nstderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}

	output := result.Stdout + result.Stderr

	// All three transforms should pass
	if !strings.Contains(output, "3 passed") {
		t.Errorf("expected all 3 transforms to pass, got:\n%s", output)
	}

	// Should report dataset validation
	if !strings.Contains(output, "dataset(s) validated") {
		t.Errorf("expected dataset validation report, got:\n%s", output)
	}

	// Should report overall success
	if !strings.Contains(output, "All validations passed") {
		t.Errorf("expected 'All validations passed', got:\n%s", output)
	}
}

// TestMultiTransform_Status tests that dk status --scan-dir shows a correct
// project summary for a multi-transform project.
func TestMultiTransform_Status(t *testing.T) {
	skipIfShort(t)

	fixturePath := multiTransformPath(t)

	result, err := runDK(t, "status", "--scan-dir", fixturePath)
	if err != nil {
		t.Fatalf("status --scan-dir failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("status --scan-dir exited %d:\nstdout: %s\nstderr: %s",
			result.ExitCode, result.Stdout, result.Stderr)
	}

	output := result.Stdout + result.Stderr

	// Check summary counts
	if !strings.Contains(output, "Transforms: 3") {
		t.Errorf("expected 'Transforms: 3', got:\n%s", output)
	}
	if !strings.Contains(output, "DataSets:   4") {
		t.Errorf("expected 'DataSets:   4', got:\n%s", output)
	}
	if !strings.Contains(output, "Edges:      6") {
		t.Errorf("expected 'Edges:      6', got:\n%s", output)
	}

	// Check transform table entries
	for _, name := range []string{"extract", "transform-etl", "load-report"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected status table to contain transform %q, got:\n%s", name, output)
		}
	}

	// Check runtimes are listed
	if !strings.Contains(output, "generic-go") {
		t.Errorf("expected status table to show 'generic-go' runtime, got:\n%s", output)
	}

	// Check trigger types
	if !strings.Contains(output, "schedule") {
		t.Errorf("expected status to show schedule trigger, got:\n%s", output)
	}
	if !strings.Contains(output, "on-change") {
		t.Errorf("expected status to show on-change trigger, got:\n%s", output)
	}
}

// TestMultiTransform_RunScanDir tests that dk run --scan-dir discovers
// transforms in the correct topological order and attempts to execute them
// sequentially.
func TestMultiTransform_RunScanDir(t *testing.T) {
	skipIfShort(t)
	skipIfNoDocker(t)

	fixturePath := multiTransformPath(t)

	// Run with --dry-run; the build will fail because the fixture has no
	// source code, but we can verify the discovery and ordering output.
	result, _ := runDK(t, "run", "--scan-dir", fixturePath, "--dry-run")

	// Combine output for assertions (some output goes to stderr on failure)
	output := result.Stdout + result.Stderr

	// Verify multi-transform mode header
	if !strings.Contains(output, "Multi-transform run: 3 transforms in topological order") {
		t.Errorf("expected multi-transform header, got:\n%s", output)
	}

	// Verify topological ordering: extract must come before transform-etl,
	// which must come before load-report
	extractIdx := strings.Index(output, "1. extract")
	etlIdx := strings.Index(output, "2. transform-etl")
	loadIdx := strings.Index(output, "3. load-report")

	if extractIdx == -1 {
		t.Errorf("expected extract as step 1, got:\n%s", output)
	}
	if etlIdx == -1 {
		t.Errorf("expected transform-etl as step 2, got:\n%s", output)
	}
	if loadIdx == -1 {
		t.Errorf("expected load-report as step 3, got:\n%s", output)
	}

	if extractIdx != -1 && etlIdx != -1 && loadIdx != -1 {
		if extractIdx >= etlIdx || etlIdx >= loadIdx {
			t.Errorf("expected topological order: extract < transform-etl < load-report, but got indices %d, %d, %d",
				extractIdx, etlIdx, loadIdx)
		}
	}

	// Verify dry run mode indicator
	if !strings.Contains(output, "Dry run mode") {
		t.Errorf("expected 'Dry run mode' indicator, got:\n%s", output)
	}

	// Verify execution attempt: it should try to run extract first
	if !strings.Contains(output, "[1/3] extract") {
		t.Errorf("expected execution attempt for extract, got:\n%s", output)
	}

	// Verify summary section is printed
	if !strings.Contains(output, "Summary") {
		t.Errorf("expected summary section, got:\n%s", output)
	}
	if !strings.Contains(output, "Total:     3") {
		t.Errorf("expected 'Total: 3' in summary, got:\n%s", output)
	}
}

// TestMultiTransform_RunScanDirEmpty tests that dk run --scan-dir handles
// an empty directory gracefully.
func TestMultiTransform_RunScanDirEmpty(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, err := runDK(t, "run", "--scan-dir", tmpDir)
	if err != nil {
		t.Fatalf("run --scan-dir on empty dir failed: %v", err)
	}

	// Should exit cleanly with a message about no transforms found
	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "No transforms found") {
		t.Errorf("expected 'No transforms found' message, got:\n%s", output)
	}
}

// TestMultiTransform_PipelineShowEmpty tests that dk pipeline show --scan-dir
// handles an empty directory gracefully.
func TestMultiTransform_PipelineShowEmpty(t *testing.T) {
	skipIfShort(t)

	tmpDir := createTempDir(t)

	result, err := runDK(t, "pipeline", "show", "--scan-dir", tmpDir)
	if err != nil {
		t.Fatalf("pipeline show on empty dir failed: %v", err)
	}

	output := result.Stdout + result.Stderr
	if !strings.Contains(output, "No transforms or datasets found") {
		t.Errorf("expected empty graph message, got:\n%s", output)
	}
}

// TestMultiTransform_EndToEnd exercises the full multi-transform workflow
// using dk init to scaffold real transforms, then validates the pipeline
// with lint, pipeline show, status, and run.
func TestMultiTransform_EndToEnd(t *testing.T) {
	skipIfShort(t)
	skipIfNoDocker(t)

	projectDir := createTempDir(t)

	// Step 1: Scaffold two transforms
	t.Run("scaffold", func(t *testing.T) {
		// Create transform A
		result, err := runDKInDir(t, projectDir, "init", "--runtime", "generic-go", "transform-a")
		if err != nil {
			t.Fatalf("init transform-a failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("init transform-a exited %d: %s", result.ExitCode, result.Stderr)
		}

		// Create transform B
		result, err = runDKInDir(t, projectDir, "init", "--runtime", "generic-go", "transform-b")
		if err != nil {
			t.Fatalf("init transform-b failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("init transform-b exited %d: %s", result.ExitCode, result.Stderr)
		}

		assertFileExists(t, filepath.Join(projectDir, "transform-a", "dk.yaml"))
		assertFileExists(t, filepath.Join(projectDir, "transform-b", "dk.yaml"))
	})

	// Step 2: Wire transforms together via shared dataset.
	// transform-a outputs "shared-data", transform-b reads "shared-data".
	// This creates a dependency: transform-a must run before transform-b.
	t.Run("wire-dependencies", func(t *testing.T) {
		dkA := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: transform-a
  namespace: default
  version: 0.1.0
spec:
  runtime: generic-go
  mode: batch
  description: "First transform"
  owner: test-team
  image: "transform-a:test"
  inputs:
    - dataset: source-data
  outputs:
    - dataset: shared-data
  resources:
    cpu: "1"
    memory: "2Gi"
`
		dkB := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: transform-b
  namespace: default
  version: 0.1.0
spec:
  runtime: generic-go
  mode: batch
  description: "Second transform"
  owner: test-team
  image: "transform-b:test"
  inputs:
    - dataset: shared-data
  outputs:
    - dataset: final-data
  resources:
    cpu: "1"
    memory: "2Gi"
`
		writeTestFile(t, filepath.Join(projectDir, "transform-a", "dk.yaml"), dkA)
		writeTestFile(t, filepath.Join(projectDir, "transform-b", "dk.yaml"), dkB)
	})

	// Step 3: Pipeline show — verify dependency graph
	t.Run("pipeline-show", func(t *testing.T) {
		result, err := runDK(t, "pipeline", "show", "--scan-dir", projectDir)
		if err != nil {
			t.Fatalf("pipeline show failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("pipeline show exited %d:\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}

		output := result.Stdout + result.Stderr

		// Both transforms should be in the graph
		if !strings.Contains(output, "transform-a") {
			t.Errorf("expected graph to contain transform-a")
		}
		if !strings.Contains(output, "transform-b") {
			t.Errorf("expected graph to contain transform-b")
		}

		// The shared dataset should be visible
		if !strings.Contains(output, "shared-data") {
			t.Errorf("expected graph to contain shared-data dataset")
		}
	})

	// Step 4: Lint both transforms
	t.Run("lint", func(t *testing.T) {
		result, err := runDK(t, "lint", "--scan-dir", projectDir)
		if err != nil {
			t.Fatalf("lint --scan-dir failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("lint --scan-dir exited %d:\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}

		output := result.Stdout + result.Stderr
		if !strings.Contains(output, "2 passed") {
			t.Errorf("expected 2 transforms to pass lint, got:\n%s", output)
		}
	})

	// Step 5: Status
	t.Run("status", func(t *testing.T) {
		result, err := runDK(t, "status", "--scan-dir", projectDir)
		if err != nil {
			t.Fatalf("status --scan-dir failed: %v", err)
		}
		if result.ExitCode != 0 {
			t.Fatalf("status --scan-dir exited %d:\nstdout: %s\nstderr: %s",
				result.ExitCode, result.Stdout, result.Stderr)
		}

		output := result.Stdout + result.Stderr
		if !strings.Contains(output, "Transforms: 2") {
			t.Errorf("expected 'Transforms: 2', got:\n%s", output)
		}
	})

	// Step 6: Run with --scan-dir --dry-run — verify topological ordering
	t.Run("run-ordering", func(t *testing.T) {
		result, _ := runDK(t, "run", "--scan-dir", projectDir, "--dry-run")
		output := result.Stdout + result.Stderr

		// transform-a must be scheduled before transform-b
		if !strings.Contains(output, "Multi-transform run: 2 transforms") {
			t.Errorf("expected 2-transform run header, got:\n%s", output)
		}

		aIdx := strings.Index(output, "1. transform-a")
		bIdx := strings.Index(output, "2. transform-b")

		if aIdx == -1 {
			t.Errorf("expected transform-a as step 1, got:\n%s", output)
		}
		if bIdx == -1 {
			t.Errorf("expected transform-b as step 2, got:\n%s", output)
		}
		if aIdx != -1 && bIdx != -1 && aIdx >= bIdx {
			t.Errorf("expected transform-a before transform-b (topological order)")
		}
	})
}

// writeTestFile writes content to a file, failing the test on error.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := writeFile(path, content); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// writeFile writes content to a file.
func writeFile(path, content string) error {
	return writeFileBytes(path, []byte(content))
}

// writeFileBytes writes bytes to a file.
func writeFileBytes(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}
