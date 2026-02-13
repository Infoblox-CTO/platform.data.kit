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

func TestRenderDirectory_PythonGitignore(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:      "my-source",
		Namespace: "test",
		Language:  "python",
		Type:      "cloudquery",
		Role:      "source",
		GRPCPort:  7777,
		Version:   "v0.1.0",
	}

	if err := r.RenderDirectory(outputDir, "cloudquery/python", config); err != nil {
		t.Fatalf("RenderDirectory() error: %v", err)
	}

	// .gitignore should be created
	gitignorePath := filepath.Join(outputDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}
	content := string(data)
	for _, pattern := range []string{"__pycache__/", ".venv/", "*.py[cod]", ".pytest_cache/"} {
		if !strings.Contains(content, pattern) {
			t.Errorf(".gitignore should contain %q", pattern)
		}
	}
}

func TestRenderDirectory_GoGitignore(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	outputDir := t.TempDir()
	config := &PackageConfig{
		Name:      "my-source",
		Namespace: "test",
		Language:  "go",
		Type:      "cloudquery",
		Role:      "source",
		GRPCPort:  7777,
		Version:   "v0.1.0",
	}

	if err := r.RenderDirectory(outputDir, "cloudquery/go", config); err != nil {
		t.Fatalf("RenderDirectory() error: %v", err)
	}

	// .gitignore should be created
	gitignorePath := filepath.Join(outputDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("expected .gitignore to be created: %v", err)
	}
	content := string(data)
	for _, pattern := range []string{"vendor/", "*.test", "coverage.out"} {
		if !strings.Contains(content, pattern) {
			t.Errorf(".gitignore should contain %q", pattern)
		}
	}
}

func TestRenderDirectory_Makefile(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	for _, lang := range []string{"go", "python"} {
		t.Run(lang, func(t *testing.T) {
			outputDir := t.TempDir()
			config := &PackageConfig{
				Name:      "my-source",
				Namespace: "test",
				Language:  lang,
				Type:      "cloudquery",
				Role:      "source",
				GRPCPort:  7777,
				Version:   "v0.1.0",
			}

			if err := r.RenderDirectory(outputDir, "cloudquery/"+lang, config); err != nil {
				t.Fatalf("RenderDirectory() error: %v", err)
			}

			// Makefile should exist and include .datakit/Makefile.common
			makefilePath := filepath.Join(outputDir, "Makefile")
			data, err := os.ReadFile(makefilePath)
			if err != nil {
				t.Fatalf("expected Makefile to be created: %v", err)
			}
			if !strings.Contains(string(data), "include .datakit/Makefile.common") {
				t.Error("Makefile should include .datakit/Makefile.common")
			}
			if !strings.Contains(string(data), "my-source") {
				t.Error("Makefile should contain package name")
			}
		})
	}
}

func TestRenderDirectory_MakefileCommon(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	for _, lang := range []string{"go", "python"} {
		t.Run(lang, func(t *testing.T) {
			outputDir := t.TempDir()
			config := &PackageConfig{
				Name:      "my-source",
				Namespace: "test",
				Language:  lang,
				Type:      "cloudquery",
				Role:      "source",
				GRPCPort:  7777,
				Version:   "v0.1.0",
			}

			if err := r.RenderDirectory(outputDir, "cloudquery/"+lang, config); err != nil {
				t.Fatalf("RenderDirectory() error: %v", err)
			}

			// .datakit/Makefile.common should exist
			commonPath := filepath.Join(outputDir, ".datakit", "Makefile.common")
			data, err := os.ReadFile(commonPath)
			if err != nil {
				t.Fatalf("expected .datakit/Makefile.common to be created: %v", err)
			}
			content := string(data)

			// Should have the DO NOT EDIT warning
			if !strings.Contains(content, "DO NOT EDIT") {
				t.Error("Makefile.common should contain DO NOT EDIT warning")
			}
			// Should have version stamp
			if !strings.Contains(content, "v0.1.0") {
				t.Error("Makefile.common should contain dp CLI version stamp")
			}
			// Should have help target
			if !strings.Contains(content, "help:") {
				t.Error("Makefile.common should contain help target")
			}
			// Should have test target
			if !strings.Contains(content, "test:") {
				t.Error("Makefile.common should contain test target")
			}
		})
	}
}

func TestRenderMakefileCommon(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	tests := []struct {
		lang    string
		version string
		wantIn  []string
	}{
		{
			lang:    "go",
			version: "v1.2.3",
			wantIn:  []string{"DO NOT EDIT", "v1.2.3", "go test", "go fmt", "go vet", "go mod tidy"},
		},
		{
			lang:    "python",
			version: "v0.5.0",
			wantIn:  []string{"DO NOT EDIT", "v0.5.0", "pytest", "python3 -m venv", "$(VENV)/bin/pip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			content, hash, err := r.RenderMakefileCommon(tt.lang, tt.version)
			if err != nil {
				t.Fatalf("RenderMakefileCommon() error: %v", err)
			}
			if content == "" {
				t.Fatal("expected non-empty content")
			}
			if len(hash) != 64 {
				t.Errorf("expected 64-char SHA-256 hex hash, got %d chars", len(hash))
			}
			for _, want := range tt.wantIn {
				if !strings.Contains(content, want) {
					t.Errorf("content should contain %q", want)
				}
			}
		})
	}
}

func TestRenderMakefileCommon_DeterministicHash(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	_, hash1, _ := r.RenderMakefileCommon("go", "v1.0.0")
	_, hash2, _ := r.RenderMakefileCommon("go", "v1.0.0")

	if hash1 != hash2 {
		t.Errorf("same inputs should produce same hash: %s != %s", hash1, hash2)
	}

	_, hash3, _ := r.RenderMakefileCommon("go", "v2.0.0")
	if hash1 == hash3 {
		t.Error("different versions should produce different hashes")
	}
}

func TestRenderMakefileCommon_InvalidLanguage(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error: %v", err)
	}

	_, _, err = r.RenderMakefileCommon("rust", "v1.0.0")
	if err == nil {
		t.Error("expected error for invalid language")
	}
}
