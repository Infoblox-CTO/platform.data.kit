package dataset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

func TestLoadDataSet(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string) string // returns path to pass to LoadDataSet
		wantName  string
		wantStore string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid dataset from directory",
			setup: func(dir string) string {
				datasetDir := filepath.Join(dir, "datasets", "my-source")
				writeDataSetYAML(t, datasetDir, &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1",
					Kind:       "DataSet",
					Metadata:   contracts.DataSetMetadata{Name: "my-source"},
					Spec:       contracts.DataSetSpec{Store: "my-s3", Prefix: "raw/"},
				})
				return datasetDir
			},
			wantName:  "my-source",
			wantStore: "my-s3",
		},
		{
			name: "valid dataset from file path",
			setup: func(dir string) string {
				datasetDir := filepath.Join(dir, "datasets", "my-sink")
				writeDataSetYAML(t, datasetDir, &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1",
					Kind:       "DataSet",
					Metadata:   contracts.DataSetMetadata{Name: "my-sink"},
					Spec:       contracts.DataSetSpec{Store: "my-pg", Table: "public.events"},
				})
				return filepath.Join(datasetDir, "dataset.yaml")
			},
			wantName:  "my-sink",
			wantStore: "my-pg",
		},
		{
			name: "missing file",
			setup: func(dir string) string {
				return filepath.Join(dir, "nonexistent")
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "malformed YAML",
			setup: func(dir string) string {
				datasetDir := filepath.Join(dir, "bad")
				if err := os.MkdirAll(datasetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), []byte("not: [valid: yaml: :::"), 0644); err != nil {
					t.Fatal(err)
				}
				return datasetDir
			},
			wantErr:   true,
			errSubstr: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)

			dataset, err := LoadDataSet(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if dataset.Metadata.Name != tt.wantName {
				t.Errorf("Metadata.Name = %q, want %q", dataset.Metadata.Name, tt.wantName)
			}
			if dataset.Spec.Store != tt.wantStore {
				t.Errorf("Spec.Store = %q, want %q", dataset.Spec.Store, tt.wantStore)
			}
		})
	}
}

func TestLoadAllDataSets(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string)
		wantCount int
		wantNames []string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "three datasets in flat layout",
			setup: func(dir string) {
				writeDataSetYAML(t, filepath.Join(dir, "datasets", "src-a"), &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
					Metadata: contracts.DataSetMetadata{Name: "src-a"},
					Spec:     contracts.DataSetSpec{Store: "my-s3", Prefix: "raw/"},
				})
				writeDataSetYAML(t, filepath.Join(dir, "datasets", "sink-b"), &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
					Metadata: contracts.DataSetMetadata{Name: "sink-b"},
					Spec:     contracts.DataSetSpec{Store: "my-pg", Table: "public.events"},
				})
				writeDataSetYAML(t, filepath.Join(dir, "datasets", "model-c"), &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
					Metadata: contracts.DataSetMetadata{Name: "model-c"},
					Spec:     contracts.DataSetSpec{Store: "my-pg", Table: "public.enriched"},
				})
			},
			wantCount: 3,
			wantNames: []string{"src-a", "sink-b", "model-c"},
		},
		{
			name:      "no datasets directory",
			setup:     func(dir string) {},
			wantCount: 0,
		},
		{
			name: "empty datasets directory",
			setup: func(dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "datasets"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)

			datasets, err := LoadAllDataSets(tmpDir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(datasets) != tt.wantCount {
				t.Fatalf("got %d datasets, want %d", len(datasets), tt.wantCount)
			}

			if tt.wantNames != nil {
				names := make(map[string]bool)
				for _, a := range datasets {
					names[a.Metadata.Name] = true
				}
				for _, n := range tt.wantNames {
					if !names[n] {
						t.Errorf("expected dataset %q not found", n)
					}
				}
			}
		})
	}
}

func TestFindDataSetByName(t *testing.T) {
	tmpDir := t.TempDir()
	writeDataSetYAML(t, filepath.Join(tmpDir, "datasets", "my-source"), &contracts.DataSetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
		Metadata: contracts.DataSetMetadata{Name: "my-source"},
		Spec:     contracts.DataSetSpec{Store: "my-s3", Prefix: "raw/"},
	})

	// Found
	dataset, err := FindDataSetByName(tmpDir, "my-source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dataset.Metadata.Name != "my-source" {
		t.Errorf("Metadata.Name = %q, want %q", dataset.Metadata.Name, "my-source")
	}

	// Not found
	_, err = FindDataSetByName(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent dataset")
	}
}

func TestDataSetPath(t *testing.T) {
	path := DataSetPath("/project", "my-source")
	want := filepath.Join("/project", "datasets", "my-source", "dataset.yaml")
	if path != want {
		t.Errorf("DataSetPath = %q, want %q", path, want)
	}
}

func TestDataSetDir(t *testing.T) {
	dir := DataSetDir("/project", "my-source")
	want := filepath.Join("/project", "datasets", "my-source")
	if dir != want {
		t.Errorf("DataSetDir = %q, want %q", dir, want)
	}
}

// writeDataSetYAML is a test helper that creates a dataset.yaml file in the given directory.
func writeDataSetYAML(t *testing.T, dir string, dataset *contracts.DataSetManifest) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(dataset)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dataset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// containsString checks if s contains the substring sub.
func containsString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
