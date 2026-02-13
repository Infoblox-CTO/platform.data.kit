package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderDirectory_Python(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:        "my-source",
		Namespace:   "data-team",
		Description: "Test CloudQuery plugin",
		Owner:       "data-team",
		Language:    "python",
		Type:        "cloudquery",
		Role:        "source",
		GRPCPort:    7777,
		Concurrency: 10000,
	}

	if err := r.RenderDirectory(outputDir, "cloudquery/python", config); err != nil {
		t.Fatalf("RenderDirectory() error: %v", err)
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
	for _, want := range []string{"my-source", "data-team", "cloudquery", "source"} {
		if !strings.Contains(content, want) {
			t.Errorf("dp.yaml should contain %q", want)
		}
	}
}

func TestRenderDirectory_Go(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:        "go-source",
		Namespace:   "analytics",
		Description: "Go CloudQuery plugin",
		Owner:       "analytics",
		Language:    "go",
		Type:        "cloudquery",
		Role:        "source",
		GRPCPort:    8888,
		Concurrency: 5000,
	}

	if err := r.RenderDirectory(outputDir, "cloudquery/go", config); err != nil {
		t.Fatalf("RenderDirectory() error: %v", err)
	}

	dpPath := filepath.Join(outputDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Fatal("expected dp.yaml to be created for Go template")
	}

	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read dp.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "go-source") {
		t.Error("dp.yaml should contain package name 'go-source'")
	}
	if !strings.Contains(content, "8888") {
		t.Error("dp.yaml should contain custom grpcPort 8888")
	}
}

func TestRenderDirectory_InvalidSubDir(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{Name: "test"}

	err = r.RenderDirectory(outputDir, "cloudquery/nonexistent", config)
	if err == nil {
		t.Error("expected error for invalid template subdirectory")
	}
}
