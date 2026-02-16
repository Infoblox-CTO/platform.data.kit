package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderKindDirectory_SourceCloudQuery(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:        "my-source",
		Namespace:   "data-team",
		Description: "CloudQuery source",
		Owner:       "data-team",
		Kind:        "source",
		Runtime:     "cloudquery",
		GRPCPort:    7777,
		Concurrency: 10000,
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	dpPath := filepath.Join(outputDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Fatal("expected dp.yaml to be created")
	}

	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read dp.yaml: %v", err)
	}

	content := string(data)
	for _, want := range []string{"my-source", "data-team", "cloudquery", "Source"} {
		if !strings.Contains(content, want) {
			t.Errorf("dp.yaml should contain %q", want)
		}
	}
}

func TestRenderKindDirectory_SourceGenericGo(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:        "go-source",
		Namespace:   "analytics",
		Description: "Go source extension",
		Owner:       "analytics",
		Kind:        "source",
		Runtime:     "generic-go",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dp.yaml", "main.go", "Dockerfile"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected %s to be created", f)
		}
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "dp.yaml"))
	if err != nil {
		t.Fatalf("failed to read dp.yaml: %v", err)
	}
	if !strings.Contains(string(data), "go-source") {
		t.Error("dp.yaml should contain package name 'go-source'")
	}
}

func TestRenderKindDirectory_DestinationCloudQuery(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:        "pg-dest",
		Namespace:   "data-team",
		Description: "PostgreSQL destination",
		Owner:       "data-team",
		Kind:        "destination",
		Runtime:     "cloudquery",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	dpPath := filepath.Join(outputDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Fatal("expected dp.yaml to be created")
	}

	data, _ := os.ReadFile(dpPath)
	content := string(data)
	if !strings.Contains(content, "Destination") {
		t.Error("dp.yaml should contain 'Destination'")
	}
}

func TestRenderKindDirectory_DestinationGenericGo(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "s3-writer",
		Kind:    "destination",
		Runtime: "generic-go",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dp.yaml", "main.go", "Dockerfile"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected %s to be created", f)
		}
	}
}

func TestRenderKindDirectory_ModelCloudQuery(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "my-model",
		Kind:    "model",
		Runtime: "cloudquery",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	dpPath := filepath.Join(outputDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Fatal("expected dp.yaml to be created")
	}

	configPath := filepath.Join(outputDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config.yaml to be created for model/cloudquery")
	}
}

func TestRenderKindDirectory_ModelDBT(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "user-agg",
		Kind:    "model",
		Runtime: "dbt",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dp.yaml", "dbt_project.yml", "profiles.yml", "models/example.sql"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected dbt file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_ModelGenericGo(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "data-worker",
		Kind:    "model",
		Runtime: "generic-go",
		Mode:    "batch",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dp.yaml", "main.go", "go.mod", "Dockerfile"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestRenderKindDirectory_ModelGenericPython(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:    "fraud-scorer",
		Kind:    "model",
		Runtime: "generic-python",
		Mode:    "streaming",
	}

	if err := r.RenderKindDirectory(outputDir, config); err != nil {
		t.Fatalf("RenderKindDirectory() error: %v", err)
	}

	for _, f := range []string{"dp.yaml", "main.py", "requirements.txt", "Dockerfile"} {
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
		Kind:    "source",
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
