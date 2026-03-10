package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

func TestDataSetValidate(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setup      func(dir string)
		wantErr    bool
		wantOutput string // substring expected in output
	}{
		{
			name: "valid single dataset by path",
			args: []string{"datasets/aws-security"},
			setup: func(dir string) {
				writeValidateDataSet(t, dir, "aws-security", "s3://bucket")
			},
		},
		{
			name: "valid all datasets in project",
			args: []string{"--offline"},
			setup: func(dir string) {
				writeValidateDataSet(t, dir, "aws-security", "s3://bucket")
				writeValidateDataSet(t, dir, "raw-output", "pg://db")
			},
			wantOutput: "All 2 datasets are valid",
		},
		{
			name: "invalid dataset - missing store",
			args: []string{"datasets/bad-dataset"},
			setup: func(dir string) {
				datasetDir := filepath.Join(dir, "datasets", "bad-dataset")
				if err := os.MkdirAll(datasetDir, 0755); err != nil {
					t.Fatal(err)
				}
				a := &contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1",
					Kind:       "DataSet",
					Metadata: contracts.DataSetMetadata{
						Name: "bad-dataset",
					},
					Spec: contracts.DataSetSpec{
						// Store intentionally omitted
						Table: "some_table",
					},
				}
				data, _ := yaml.Marshal(a)
				if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
		{
			name:    "non-existent path",
			args:    []string{"datasets/ghost"},
			wantErr: true,
		},
		{
			name: "offline mode skips schema validation",
			args: []string{"--offline", "datasets/aws-security"},
			setup: func(dir string) {
				writeValidateDataSet(t, dir, "aws-security", "s3://bucket")
			},
		},
		{
			name:       "no datasets directory",
			args:       []string{},
			wantOutput: "No datasets found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(tmpDir)
			}

			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)

			// Reset global flags to avoid state leaking between tests
			datasetValidateOffline = false

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"dataset", "validate"}, tt.args...))

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v\nOutput: %s", err, buf.String())
			}

			if tt.wantOutput != "" {
				output := buf.String()
				if !bytes.Contains([]byte(output), []byte(tt.wantOutput)) {
					t.Errorf("output %q should contain %q", output, tt.wantOutput)
				}
			}
		})
	}
}

// writeValidateDataSet creates a valid dataset.yaml in the flat datasets/<name>/ layout.
func writeValidateDataSet(t *testing.T, projectDir, name, store string) {
	t.Helper()

	datasetDir := filepath.Join(projectDir, "datasets", name)
	if err := os.MkdirAll(datasetDir, 0755); err != nil {
		t.Fatal(err)
	}

	a := &contracts.DataSetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "DataSet",
		Metadata: contracts.DataSetMetadata{
			Name: name,
		},
		Spec: contracts.DataSetSpec{
			Store:          store,
			Table:          "default_table",
			Classification: "internal",
		},
	}
	data, _ := yaml.Marshal(a)
	if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
