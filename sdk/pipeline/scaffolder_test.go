package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldPipeline_WithTemplate(t *testing.T) {
	dir := t.TempDir()

	path, err := ScaffoldPipeline(ScaffoldOpts{
		Name:       "my-pipeline",
		Template:   "sync-transform-test",
		ProjectDir: dir,
	})
	if err != nil {
		t.Fatalf("ScaffoldPipeline() error = %v", err)
	}

	if path != filepath.Join(dir, PipelineFileName) {
		t.Errorf("path = %q, want %q", path, filepath.Join(dir, PipelineFileName))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "name: my-pipeline") {
		t.Errorf("expected pipeline name in output, got:\n%s", content)
	}
	if !strings.Contains(content, "type: sync") {
		t.Errorf("expected sync step type in output, got:\n%s", content)
	}
}

func TestScaffoldPipeline_SyncOnly(t *testing.T) {
	dir := t.TempDir()

	path, err := ScaffoldPipeline(ScaffoldOpts{
		Name:       "simple-sync",
		Template:   "sync-only",
		ProjectDir: dir,
	})
	if err != nil {
		t.Fatalf("ScaffoldPipeline() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "name: simple-sync") {
		t.Errorf("expected pipeline name in output, got:\n%s", content)
	}
}

func TestScaffoldPipeline_Custom(t *testing.T) {
	dir := t.TempDir()

	_, err := ScaffoldPipeline(ScaffoldOpts{
		Name:       "my-custom",
		Template:   "custom",
		ProjectDir: dir,
	})
	if err != nil {
		t.Fatalf("ScaffoldPipeline() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, PipelineFileName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "type: custom") {
		t.Errorf("expected custom step type in output, got:\n%s", content)
	}
}

func TestScaffoldPipeline_ExistingFileError(t *testing.T) {
	dir := t.TempDir()

	// Create existing pipeline.yaml
	existing := filepath.Join(dir, PipelineFileName)
	if err := os.WriteFile(existing, []byte("existing"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := ScaffoldPipeline(ScaffoldOpts{
		Name:       "test",
		Template:   "sync-only",
		ProjectDir: dir,
	})
	if err == nil {
		t.Fatal("expected error for existing file, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v, want 'already exists'", err)
	}
}

func TestScaffoldPipeline_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()

	// Create existing pipeline.yaml
	existing := filepath.Join(dir, PipelineFileName)
	if err := os.WriteFile(existing, []byte("old-content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := ScaffoldPipeline(ScaffoldOpts{
		Name:       "overwritten",
		Template:   "sync-only",
		ProjectDir: dir,
		Force:      true,
	})
	if err != nil {
		t.Fatalf("ScaffoldPipeline() error = %v", err)
	}

	data, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(data), "old-content") {
		t.Error("file was not overwritten")
	}
	if !strings.Contains(string(data), "name: overwritten") {
		t.Errorf("expected new content, got:\n%s", string(data))
	}
}

func TestScaffoldPipeline_MissingName(t *testing.T) {
	_, err := ScaffoldPipeline(ScaffoldOpts{
		Template:   "sync-only",
		ProjectDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestScaffoldPipeline_MissingTemplate(t *testing.T) {
	_, err := ScaffoldPipeline(ScaffoldOpts{
		Name:       "test",
		ProjectDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for missing template, got nil")
	}
}

func TestScaffoldPipeline_MissingProjectDir(t *testing.T) {
	_, err := ScaffoldPipeline(ScaffoldOpts{
		Name:     "test",
		Template: "sync-only",
	})
	if err == nil {
		t.Fatal("expected error for missing project dir, got nil")
	}
}

func TestScaffoldPipeline_InvalidTemplate(t *testing.T) {
	_, err := ScaffoldPipeline(ScaffoldOpts{
		Name:       "test",
		Template:   "nonexistent-template",
		ProjectDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}
