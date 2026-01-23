package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/data.platform.kit/contracts"
)

func TestNewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Error("NewParser() returned nil")
	}

	_, ok := p.(*DefaultParser)
	if !ok {
		t.Error("NewParser() did not return *DefaultParser")
	}
}

func TestDefaultParser_ParseDataPackage(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		wantName string
	}{
		{
			name: "valid datapackage",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: dataset
  description: Test package
  owner: data-team
`),
			wantErr:  false,
			wantName: "test-pkg",
		},
		{
			name: "wrong kind",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: test
`),
			wantErr: true,
		},
		{
			name:    "malformed YAML",
			data:    []byte("this is not valid yaml: [broken"),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte(""),
			wantErr: true,
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := p.ParseDataPackage(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDataPackage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pkg.Metadata.Name != tt.wantName {
				t.Errorf("ParseDataPackage() name = %v, want %v", pkg.Metadata.Name, tt.wantName)
			}
		})
	}
}

func TestDefaultParser_ParsePipeline(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		wantName  string
		wantImage string
	}{
		{
			name: "valid pipeline",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  image: myorg/pipeline:latest
`),
			wantErr:   false,
			wantName:  "test-pipeline",
			wantImage: "myorg/pipeline:latest",
		},
		{
			name: "wrong kind",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test
`),
			wantErr: true,
		},
		{
			name:    "malformed YAML",
			data:    []byte("not: valid: yaml:"),
			wantErr: true,
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline, err := p.ParsePipeline(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePipeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if pipeline.Metadata.Name != tt.wantName {
					t.Errorf("ParsePipeline() name = %v, want %v", pipeline.Metadata.Name, tt.wantName)
				}
				if pipeline.Spec.Image != tt.wantImage {
					t.Errorf("ParsePipeline() image = %v, want %v", pipeline.Spec.Image, tt.wantImage)
				}
			}
		})
	}
}

func TestDefaultParser_ParseBindings(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		wantCount int
	}{
		{
			name: "valid bindings",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings:
  - name: data-bucket
    type: s3-prefix
    s3:
      bucket: my-bucket
      prefix: data/
  - name: events-topic
    type: kafka-topic
    kafka:
      topic: events
`),
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "empty bindings",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings: []
`),
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:    "malformed YAML",
			data:    []byte("not valid yaml ["),
			wantErr: true,
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindings, err := p.ParseBindings(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBindings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(bindings) != tt.wantCount {
				t.Errorf("ParseBindings() count = %v, want %v", len(bindings), tt.wantCount)
			}
		})
	}
}

func TestParseDataPackageFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string) string
		wantErr  bool
		wantName string
	}{
		{
			name: "valid file",
			setup: func(t *testing.T, dir string) string {
				content := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: file-test
  namespace: test
  version: 1.0.0
spec:
  type: dataset
  owner: test
`
				path := filepath.Join(dir, "dp.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path
			},
			wantErr:  false,
			wantName: "file-test",
		},
		{
			name: "file not found",
			setup: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "nonexistent.yaml")
			},
			wantErr: true,
		},
		{
			name: "malformed file",
			setup: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "bad.yaml")
				if err := os.WriteFile(path, []byte("not valid yaml ["), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(t, dir)

			pkg, err := ParseDataPackageFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDataPackageFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pkg.Metadata.Name != tt.wantName {
				t.Errorf("ParseDataPackageFile() name = %v, want %v", pkg.Metadata.Name, tt.wantName)
			}
		})
	}
}

func TestParsePipelineFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string) string
		wantErr  bool
		wantName string
	}{
		{
			name: "valid file",
			setup: func(t *testing.T, dir string) string {
				content := `apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: pipeline-file-test
spec:
  image: myorg/app:v1
`
				path := filepath.Join(dir, "pipeline.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path
			},
			wantErr:  false,
			wantName: "pipeline-file-test",
		},
		{
			name: "file not found",
			setup: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "missing.yaml")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(t, dir)

			pipeline, err := ParsePipelineFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePipelineFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pipeline.Metadata.Name != tt.wantName {
				t.Errorf("ParsePipelineFile() name = %v, want %v", pipeline.Metadata.Name, tt.wantName)
			}
		})
	}
}

func TestParseBindingsFile(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, dir string) string
		wantErr   bool
		wantCount int
	}{
		{
			name: "valid file",
			setup: func(t *testing.T, dir string) string {
				content := `apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings:
  - name: test-bucket
    type: s3-prefix
    s3:
      bucket: test
`
				path := filepath.Join(dir, "bindings.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "file not found",
			setup: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "missing.yaml")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(t, dir)

			bindings, err := ParseBindingsFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBindingsFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(bindings) != tt.wantCount {
				t.Errorf("ParseBindingsFile() count = %v, want %v", len(bindings), tt.wantCount)
			}
		})
	}
}

func TestParser_ParseFromTestdata(t *testing.T) {
	// Test parsing from actual testdata files
	t.Run("valid datapackage", func(t *testing.T) {
		pkg, err := ParseDataPackageFile("testdata/valid/datapackage.yaml")
		if err != nil {
			t.Fatalf("ParseDataPackageFile() error = %v", err)
		}
		if pkg.Metadata.Name != "valid-manifest" {
			t.Errorf("name = %v, want valid-manifest", pkg.Metadata.Name)
		}
		if pkg.Spec.Type != contracts.PackageTypePipeline {
			t.Errorf("type = %v, want %v", pkg.Spec.Type, contracts.PackageTypePipeline)
		}
	})

	t.Run("valid pipeline", func(t *testing.T) {
		pipeline, err := ParsePipelineFile("testdata/valid/pipeline.yaml")
		if err != nil {
			t.Fatalf("ParsePipelineFile() error = %v", err)
		}
		if pipeline.Metadata.Name != "sample-pipeline" {
			t.Errorf("name = %v, want sample-pipeline", pipeline.Metadata.Name)
		}
		if pipeline.Spec.Image != "docker.io/myorg/pipeline:latest" {
			t.Errorf("image = %v, want docker.io/myorg/pipeline:latest", pipeline.Spec.Image)
		}
	})

	t.Run("malformed yaml", func(t *testing.T) {
		_, err := ParseDataPackageFile("testdata/invalid/malformed.yaml")
		if err == nil {
			t.Error("expected error for malformed YAML")
		}
	})

	t.Run("missing metadata", func(t *testing.T) {
		pkg, err := ParseDataPackageFile("testdata/invalid/missing-metadata.yaml")
		// This should parse successfully but have empty metadata
		if err != nil {
			t.Fatalf("ParseDataPackageFile() unexpected error = %v", err)
		}
		if pkg.Metadata.Name != "" {
			t.Errorf("expected empty name for missing metadata, got %v", pkg.Metadata.Name)
		}
	})
}

// Edge case tests for malformed YAML (T030)
func TestParser_MalformedYAML(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "unclosed bracket",
			data: []byte("key: [value"),
		},
		{
			name: "unclosed brace",
			data: []byte("key: {nested: value"),
		},
		{
			name: "bad indentation",
			data: []byte("key:\n value\n  nested: bad"),
		},
		{
			name: "duplicate keys",
			data: []byte("key: value1\nkey: value2"),
		},
		{
			name: "tabs instead of spaces",
			data: []byte("key:\n\t- value"),
		},
		{
			name: "invalid unicode",
			data: []byte("key: \xff\xfe"),
		},
		{
			name: "null bytes",
			data: []byte("key: \x00value"),
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ParseDataPackage(tt.data)
			// Some malformed YAML might parse, we're just checking no panic
			_ = err
		})
	}
}

// Edge case tests for missing required fields (T031)
func TestParser_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		checkErr bool
	}{
		{
			name: "missing apiVersion",
			data: []byte(`kind: DataPackage
metadata:
  name: test
`),
			checkErr: false, // Parses but validation should catch
		},
		{
			name: "missing kind",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
metadata:
  name: test
`),
			checkErr: true, // Should error on wrong/empty kind
		},
		{
			name: "missing metadata",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
spec:
  type: dataset
`),
			checkErr: false, // Parses but has empty metadata
		},
		{
			name: "missing spec",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test
`),
			checkErr: false, // Parses but has empty spec
		},
		{
			name: "empty metadata name",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: ""
`),
			checkErr: false, // Parses but validation should catch empty name
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := p.ParseDataPackage(tt.data)
			if tt.checkErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Package should exist even with missing fields
				if pkg == nil {
					t.Error("expected package to be returned")
				}
			}
		})
	}
}
