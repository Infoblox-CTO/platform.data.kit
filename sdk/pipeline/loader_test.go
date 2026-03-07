package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPipeline_ValidFile(t *testing.T) {
	dir := t.TempDir()
	content := "apiVersion: datakit.infoblox.dev/v1alpha1\nkind: PipelineWorkflow\nmetadata:\n  name: test-pipeline\n  description: A test pipeline\nsteps:\n  - name: sync-data\n    type: sync\n    input: my-source\n    output: my-sink\n  - name: transform-data\n    type: transform\n    asset: my-model\n"
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pw, err := LoadPipeline(dir)
	if err != nil {
		t.Fatalf("LoadPipeline() error = %v", err)
	}

	if pw.APIVersion != "datakit.infoblox.dev/v1alpha1" {
		t.Errorf("APIVersion = %q, want %q", pw.APIVersion, "datakit.infoblox.dev/v1alpha1")
	}
	if pw.Kind != "PipelineWorkflow" {
		t.Errorf("Kind = %q, want %q", pw.Kind, "PipelineWorkflow")
	}
	if pw.Metadata.Name != "test-pipeline" {
		t.Errorf("Metadata.Name = %q, want %q", pw.Metadata.Name, "test-pipeline")
	}
	if len(pw.Steps) != 2 {
		t.Fatalf("len(Steps) = %d, want 2", len(pw.Steps))
	}
	if pw.Steps[0].Name != "sync-data" {
		t.Errorf("Steps[0].Name = %q, want %q", pw.Steps[0].Name, "sync-data")
	}
	if pw.Steps[1].Name != "transform-data" {
		t.Errorf("Steps[1].Name = %q, want %q", pw.Steps[1].Name, "transform-data")
	}
}

func TestLoadPipeline_DirectFilePath(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "pipeline.yaml")
	content := "apiVersion: datakit.infoblox.dev/v1alpha1\nkind: PipelineWorkflow\nmetadata:\n  name: direct-path\nsteps:\n  - name: run-step\n    type: custom\n    image: my-image:latest\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pw, err := LoadPipeline(filePath)
	if err != nil {
		t.Fatalf("LoadPipeline() error = %v", err)
	}

	if pw.Metadata.Name != "direct-path" {
		t.Errorf("Metadata.Name = %q, want %q", pw.Metadata.Name, "direct-path")
	}
}

func TestLoadPipeline_MissingFile(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadPipeline(dir)
	if err == nil {
		t.Fatal("LoadPipeline() expected error for missing file, got nil")
	}
}

func TestLoadPipeline_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := "this is not: [valid yaml: because"
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPipeline(dir)
	if err == nil {
		t.Fatal("LoadPipeline() expected error for invalid YAML, got nil")
	}
}

func TestLoadPipeline_MalformedSteps(t *testing.T) {
	dir := t.TempDir()
	content := "apiVersion: datakit.infoblox.dev/v1alpha1\nkind: PipelineWorkflow\nmetadata:\n  name: bad-steps\nsteps: not-an-array\n"
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPipeline(dir)
	if err == nil {
		t.Fatal("LoadPipeline() expected error for malformed steps, got nil")
	}
}

func TestLoadPipeline_NonexistentPath(t *testing.T) {
	_, err := LoadPipeline("/nonexistent/path/to/pipeline")
	if err == nil {
		t.Fatal("LoadPipeline() expected error for nonexistent path, got nil")
	}
}

func TestFindPipeline_Found(t *testing.T) {
	dir := t.TempDir()
	content := "apiVersion: datakit.infoblox.dev/v1alpha1\nkind: PipelineWorkflow\nmetadata:\n  name: found-pipeline\nsteps:\n  - name: step-one\n    type: custom\n    image: test:latest\n"
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	path, err := FindPipeline(dir)
	if err != nil {
		t.Fatalf("FindPipeline() error = %v", err)
	}
	if path == "" {
		t.Fatal("FindPipeline() returned empty path, want non-empty")
	}
	if filepath.Base(path) != "pipeline.yaml" {
		t.Errorf("FindPipeline() = %q, want pipeline.yaml", filepath.Base(path))
	}
}

func TestFindPipeline_FoundInParent(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	content := "apiVersion: datakit.infoblox.dev/v1alpha1\nkind: PipelineWorkflow\nmetadata:\n  name: parent-pipeline\nsteps:\n  - name: step-one\n    type: custom\n    image: test:latest\n"
	if err := os.WriteFile(filepath.Join(parent, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	path, err := FindPipeline(child)
	if err != nil {
		t.Fatalf("FindPipeline() error = %v", err)
	}
	if path == "" {
		t.Fatal("FindPipeline() returned empty path, want non-empty")
	}
}

func TestFindPipeline_NotFound(t *testing.T) {
	dir := t.TempDir()

	path, err := FindPipeline(dir)
	if err != nil {
		t.Fatalf("FindPipeline() error = %v", err)
	}
	if path != "" {
		t.Errorf("FindPipeline() = %q, want empty string", path)
	}
}

func TestHasPipeline_Exists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if !HasPipeline(dir) {
		t.Error("HasPipeline() = false, want true")
	}
}

func TestHasPipeline_NotExists(t *testing.T) {
	dir := t.TempDir()

	if HasPipeline(dir) {
		t.Error("HasPipeline() = true, want false")
	}
}

// --- LoadSchedule Tests (T042) ---

func TestLoadSchedule_ValidFile(t *testing.T) {
	dir := t.TempDir()
	content := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Schedule
cron: "0 6 * * *"
timezone: America/New_York
`
	if err := os.WriteFile(filepath.Join(dir, "schedule.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	sm, err := LoadSchedule(dir)
	if err != nil {
		t.Fatalf("LoadSchedule() error = %v", err)
	}
	if sm == nil {
		t.Fatal("LoadSchedule() = nil, want schedule")
	}
	if sm.Cron != "0 6 * * *" {
		t.Errorf("Cron = %q, want %q", sm.Cron, "0 6 * * *")
	}
	if sm.Timezone != "America/New_York" {
		t.Errorf("Timezone = %q, want %q", sm.Timezone, "America/New_York")
	}
}

func TestLoadSchedule_MissingFileReturnsNil(t *testing.T) {
	dir := t.TempDir()

	sm, err := LoadSchedule(dir)
	if err != nil {
		t.Fatalf("LoadSchedule() error = %v", err)
	}
	if sm != nil {
		t.Errorf("LoadSchedule() = %v, want nil for missing file", sm)
	}
}

func TestLoadSchedule_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := `not: [valid: yaml: {{`
	if err := os.WriteFile(filepath.Join(dir, "schedule.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSchedule(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
