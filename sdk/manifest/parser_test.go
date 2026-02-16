package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
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

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		wantKind contracts.Kind
		wantName string
	}{
		{
			name: "valid source",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Source
metadata:
  name: my-source
  namespace: data-team
  version: 1.0.0
spec:
  description: A test source
  owner: data-team
  runtime: cloudquery
`),
			wantErr:  false,
			wantKind: contracts.KindSource,
			wantName: "my-source",
		},
		{
			name: "valid destination",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Destination
metadata:
  name: my-dest
  namespace: data-team
  version: 1.0.0
spec:
  description: A test destination
  owner: data-team
  runtime: cloudquery
`),
			wantErr:  false,
			wantKind: contracts.KindDestination,
			wantName: "my-dest",
		},
		{
			name: "valid model",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: my-model
  namespace: data-team
  version: 1.0.0
spec:
  description: A test model
  owner: data-team
  runtime: generic-go
  image: myimage:v1
  mode: batch
`),
			wantErr:  false,
			wantKind: contracts.KindModel,
			wantName: "my-model",
		},
		{
			name: "unsupported kind returns error",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test
`),
			wantErr: true,
		},
		{
			name:    "empty bytes returns error",
			data:    []byte(""),
			wantErr: true,
		},
		{
			name:    "invalid YAML returns error",
			data:    []byte("invalid: yaml: content: ["),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, kind, err := ParseManifest(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if kind != tt.wantKind {
					t.Errorf("kind = %v, want %v", kind, tt.wantKind)
				}
				if m.GetName() != tt.wantName {
					t.Errorf("name = %v, want %v", m.GetName(), tt.wantName)
				}
			}
		})
	}
}

func TestParseManifestFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string) string
		wantErr  bool
		wantName string
		wantKind contracts.Kind
	}{
		{
			name: "valid model file",
			setup: func(t *testing.T, dir string) string {
				content := `apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: file-test
  namespace: test
  version: 1.0.0
spec:
  description: Test
  owner: test
  runtime: generic-go
`
				path := filepath.Join(dir, "dp.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path
			},
			wantErr:  false,
			wantName: "file-test",
			wantKind: contracts.KindModel,
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

			m, kind, err := ParseManifestFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseManifestFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if m.GetName() != tt.wantName {
					t.Errorf("ParseManifestFile() name = %v, want %v", m.GetName(), tt.wantName)
				}
				if kind != tt.wantKind {
					t.Errorf("ParseManifestFile() kind = %v, want %v", kind, tt.wantKind)
				}
			}
		})
	}
}

func TestDefaultParser_ParseSource(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Source
metadata:
  name: aws-source
  namespace: data-team
  version: 1.0.0
spec:
  description: AWS source
  owner: data-team
  runtime: cloudquery
  image: myorg/aws-source:v1
`)

	p := NewParser()
	src, err := p.ParseSource(data)
	if err != nil {
		t.Fatalf("ParseSource() error = %v", err)
	}
	if src.Metadata.Name != "aws-source" {
		t.Errorf("name = %v, want aws-source", src.Metadata.Name)
	}
	if src.Spec.Runtime != contracts.RuntimeCloudQuery {
		t.Errorf("runtime = %v, want cloudquery", src.Spec.Runtime)
	}
}

func TestDefaultParser_ParseDestination(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Destination
metadata:
  name: pg-dest
  namespace: data-team
  version: 1.0.0
spec:
  description: Postgres destination
  owner: data-team
  runtime: cloudquery
  image: myorg/pg-dest:v1
`)

	p := NewParser()
	dest, err := p.ParseDestination(data)
	if err != nil {
		t.Fatalf("ParseDestination() error = %v", err)
	}
	if dest.Metadata.Name != "pg-dest" {
		t.Errorf("name = %v, want pg-dest", dest.Metadata.Name)
	}
}

func TestDefaultParser_ParseModel(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: etl-model
  namespace: data-team
  version: 2.0.0
spec:
  description: ETL model
  owner: data-team
  runtime: generic-go
  image: myorg/etl:v2
  mode: batch
  env:
    - name: LOG_LEVEL
      value: debug
  outputs:
    - name: processed-data
      type: s3-prefix
      binding: output-bucket
`)

	p := NewParser()
	model, err := p.ParseModel(data)
	if err != nil {
		t.Fatalf("ParseModel() error = %v", err)
	}
	if model.Metadata.Name != "etl-model" {
		t.Errorf("name = %v, want etl-model", model.Metadata.Name)
	}
	if model.Spec.Mode != contracts.ModeBatch {
		t.Errorf("mode = %v, want batch", model.Spec.Mode)
	}
	if len(model.Spec.Env) != 1 {
		t.Errorf("env count = %v, want 1", len(model.Spec.Env))
	}
	if len(model.Spec.Outputs) != 1 {
		t.Errorf("outputs count = %v, want 1", len(model.Spec.Outputs))
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
	t.Run("valid model", func(t *testing.T) {
		m, kind, err := ParseManifestFile("testdata/valid/datapackage.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() error = %v", err)
		}
		if m.GetName() != "valid-manifest" {
			t.Errorf("name = %v, want valid-manifest", m.GetName())
		}
		if kind != contracts.KindModel {
			t.Errorf("kind = %v, want Model", kind)
		}
	})

	t.Run("valid source", func(t *testing.T) {
		m, kind, err := ParseManifestFile("testdata/valid/pipeline.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() error = %v", err)
		}
		if m.GetName() != "sample-source" {
			t.Errorf("name = %v, want sample-source", m.GetName())
		}
		if kind != contracts.KindSource {
			t.Errorf("kind = %v, want Source", kind)
		}
	})

	t.Run("malformed yaml", func(t *testing.T) {
		_, _, err := ParseManifestFile("testdata/invalid/malformed.yaml")
		if err == nil {
			t.Error("expected error for malformed YAML")
		}
	})

	t.Run("missing metadata", func(t *testing.T) {
		m, _, err := ParseManifestFile("testdata/invalid/missing-metadata.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() unexpected error = %v", err)
		}
		if m.GetName() != "" {
			t.Errorf("expected empty name for missing metadata, got %v", m.GetName())
		}
	})
}

// Edge case tests for malformed YAML (T030)
func TestParser_MalformedYAML(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "unclosed bracket", data: []byte("key: [value")},
		{name: "unclosed brace", data: []byte("key: {nested: value")},
		{name: "bad indentation", data: []byte("key:\n value\n  nested: bad")},
		{name: "duplicate keys", data: []byte("key: value1\nkey: value2")},
		{name: "tabs instead of spaces", data: []byte("key:\n\t- value")},
		{name: "invalid unicode", data: []byte("key: \xff\xfe")},
		{name: "null bytes", data: []byte("key: \x00value")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseManifest(tt.data)
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
			data: []byte(`kind: Model
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
			checkErr: true, // Should error on empty kind
		},
		{
			name: "missing metadata",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Model
spec:
  runtime: generic-go
`),
			checkErr: false, // Parses but has empty metadata
		},
		{
			name: "missing spec",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: test
`),
			checkErr: false, // Parses but has empty spec
		},
		{
			name: "empty metadata name",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: ""
`),
			checkErr: false, // Parses but validation should catch empty name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, err := ParseManifest(tt.data)
			if tt.checkErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if m == nil {
					t.Error("expected manifest to be returned")
				}
			}
		})
	}
}

func TestManifestInterface(t *testing.T) {
	src := &contracts.Source{
		Kind: string(contracts.KindSource),
		Metadata: contracts.ExtMetadata{
			Name:      "test-src",
			Namespace: "ns",
			Version:   "1.0.0",
		},
		Spec: contracts.SourceSpec{
			Description: "Test source",
			Owner:       "team",
		},
	}

	// Verify it satisfies the Manifest interface
	var m Manifest = src
	if m.GetKind() != contracts.KindSource {
		t.Errorf("GetKind() = %v, want Source", m.GetKind())
	}
	if m.GetName() != "test-src" {
		t.Errorf("GetName() = %v, want test-src", m.GetName())
	}
	if m.GetNamespace() != "ns" {
		t.Errorf("GetNamespace() = %v, want ns", m.GetNamespace())
	}
	if m.GetVersion() != "1.0.0" {
		t.Errorf("GetVersion() = %v, want 1.0.0", m.GetVersion())
	}
	if m.GetDescription() != "Test source" {
		t.Errorf("GetDescription() = %v, want Test source", m.GetDescription())
	}
	if m.GetOwner() != "team" {
		t.Errorf("GetOwner() = %v, want team", m.GetOwner())
	}
}
