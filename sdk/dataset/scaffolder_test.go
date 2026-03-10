package dataset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestScaffold(t *testing.T) {
	tests := []struct {
		name      string
		opts      func(dir string) ScaffoldOpts
		wantDir   string // relative to project dir
		wantErr   bool
		errSubstr string
	}{
		{
			name: "basic dataset",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:       "aws-security",
					ProjectDir: dir,
					Store:      "my-s3",
				}
			},
			wantDir: "datasets/aws-security",
		},
		{
			name: "invalid name - too short",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:       "ab",
					ProjectDir: dir,
				}
			},
			wantErr:   true,
			errSubstr: "invalid dataset name",
		},
		{
			name: "invalid name - uppercase",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:       "MyDataSet",
					ProjectDir: dir,
				}
			},
			wantErr:   true,
			errSubstr: "invalid dataset name",
		},
		{
			name: "duplicate name detection",
			opts: func(dir string) ScaffoldOpts {
				// Pre-create a dataset in flat layout
				writeDataSetYAML(t, filepath.Join(dir, "datasets", "existing"), &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
					Metadata: contracts.DataSetMetadata{Name: "existing"},
					Spec:     contracts.DataSetSpec{Store: "my-s3"},
				})
				return ScaffoldOpts{
					Name:       "existing",
					ProjectDir: dir,
				}
			},
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "force overwrite",
			opts: func(dir string) ScaffoldOpts {
				// Pre-create a dataset in flat layout
				writeDataSetYAML(t, filepath.Join(dir, "datasets", "existing"), &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
					Metadata: contracts.DataSetMetadata{Name: "existing"},
					Spec:     contracts.DataSetSpec{Store: "old-store"},
				})
				return ScaffoldOpts{
					Name:       "existing",
					ProjectDir: dir,
					Store:      "new-store",
					Force:      true,
				}
			},
			wantDir: "datasets/existing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			opts := tt.opts(tmpDir)

			result, err := Scaffold(opts)
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

			// Verify the file was created
			if _, err := os.Stat(result.DataSetPath); os.IsNotExist(err) {
				t.Fatalf("dataset.yaml not created at %s", result.DataSetPath)
			}

			// Verify the directory placement
			expectedDir := filepath.Join(tmpDir, tt.wantDir)
			if result.DataSetDir != expectedDir {
				t.Errorf("DataSetDir = %q, want %q", result.DataSetDir, expectedDir)
			}

			// Load and verify the dataset
			dataset, err := LoadDataSet(result.DataSetDir)
			if err != nil {
				t.Fatalf("failed to reload dataset: %v", err)
			}

			if dataset.Metadata.Name != opts.Name {
				t.Errorf("Metadata.Name = %q, want %q", dataset.Metadata.Name, opts.Name)
			}
			if dataset.Kind != "DataSet" {
				t.Errorf("Kind = %q, want %q", dataset.Kind, "DataSet")
			}
		})
	}
}

func TestValidateDataSetName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "my-source", wantErr: false},
		{name: "valid with numbers", input: "source-123", wantErr: false},
		{name: "valid minimum length", input: "abc", wantErr: false},
		{name: "too short", input: "ab", wantErr: true},
		{name: "uppercase", input: "MySource", wantErr: true},
		{name: "starts with number", input: "1source", wantErr: true},
		{name: "has underscore", input: "my_source", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataSetName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDataSetName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

