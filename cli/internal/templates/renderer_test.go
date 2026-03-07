package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderKindDirectory_TransformCloudQuery(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:        "my-transform",
		Namespace:   "data-team",
		Description: "CloudQuery transform",
		Owner:       "data-team",
		Kind:        "transform",
		Runtime:     "cloudquery",
		GRPCPort:    7777,
		Concurrency: 10000,
		Mode:        "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	dkPath := filepath.Join(outputDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		t.Fatal("expected dk.yaml to be created")
	}

	data, err := os.ReadFile(dkPath)
	if err != nil {
		t.Fatalf("failed to read dk.yaml: %v", err)
	}

	content := string(data)
	for _, want := range []string{"my-transform", "data-team", "cloudquery", "Transform"} {
		if !strings.Contains(content, want) {
			t.Errorf("dk.yaml should contain %q", want)
		}
	}

	// config.yaml should NOT be scaffolded — it is auto-generated at runtime by dk run.
	configPath := filepath.Join(outputDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("config.yaml should not be scaffolded; it is auto-generated at runtime")
	}

	// Verify connector/, store/, and asset/ subdirectories are scaffolded.
	for _, sub := range []string{
		"connector/postgres.yaml",
		"connector/s3.yaml",
		"store/source-db.yaml",
		"store/dest-bucket.yaml",
		"asset/source.yaml",
		"asset/destination.yaml",
	} {
		p := filepath.Join(outputDir, sub)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("expected %s to be created", sub)
		}
	}
}

func TestRenderKindDirectory_TransformGenericGo(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "data-worker",
		Kind:    "transform",
		Runtime: "generic-go",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dk.yaml", "go.mod", "main.go", "cmd/root.go", ".gitignore", ".dockerignore"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_TransformDBT(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "user-agg",
		Kind:    "transform",
		Runtime: "dbt",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dk.yaml", "dbt_project.yml", "profiles.yml", "models/example.sql"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected dbt file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_TransformGenericPython(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "fraud-scorer",
		Kind:    "transform",
		Runtime: "generic-python",
		Mode:    "streaming",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dk.yaml", "main.py", "requirements.txt"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_TransformCloudQuery_Legacy(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "my-transform",
		Kind:    "transform",
		Runtime: "cloudquery",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	dkPath := filepath.Join(outputDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		t.Fatal("expected dk.yaml to be created")
	}

	// config.yaml should NOT be scaffolded — it is auto-generated at runtime by dk run.
	configPath := filepath.Join(outputDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("config.yaml should not be scaffolded; it is auto-generated at runtime")
	}
}

func TestRenderKindDirectory_TransformDBT_Legacy(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "user-agg",
		Kind:    "transform",
		Runtime: "dbt",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dk.yaml", "dbt_project.yml", "profiles.yml", "models/example.sql"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected dbt file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_TransformGenericGo_Legacy(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "data-worker",
		Kind:    "transform",
		Runtime: "generic-go",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dk.yaml", "go.mod", "main.go", "cmd/root.go", ".gitignore", ".dockerignore"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_TransformGenericPython_Legacy(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "fraud-scorer",
		Kind:    "transform",
		Runtime: "generic-python",
		Mode:    "streaming",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dk.yaml", "main.py", "requirements.txt"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_InvalidKind(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "test",
		Kind:    "widget",
		Runtime: "cloudquery",
	}

	err = r.RenderKindDirectory(outputDir, config)
	if err == nil {
		t.Error("expected error for invalid kind")
	}
}

func TestRenderKindDirectory_InvalidRuntime(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "test",
		Kind:    "transform",
		Runtime: "nonexistent",
	}

	err = r.RenderKindDirectory(outputDir, config)
	if err == nil {
		t.Error("expected error for invalid runtime subdirectory")
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"my-app", "my_app"},
		{"simple", "simple"},
		{"a-b-c", "a_b_c"},
		{"already_snake", "already_snake"},
		{"camelCase", "camel_case"},
		{"PascalCase", "pascal_case"},
		{"my-cool-plugin", "my_cool_plugin"},
		{"x", "x"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := snakeCase(tt.in)
			if got != tt.want {
				t.Errorf("snakeCase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPascalCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"my-app", "MyApp"},
		{"simple", "Simple"},
		{"a-b-c", "ABC"},
		{"already_snake", "AlreadySnake"},
		{"my-cool-plugin", "MyCoolPlugin"},
		{"x", "X"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := pascalCase(tt.in)
			if got != tt.want {
				t.Errorf("pascalCase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
