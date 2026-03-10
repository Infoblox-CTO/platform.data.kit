package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

func TestDataSetCreate(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(dir string)
		wantErr   bool
		errSubstr string
		wantFile  string // relative to project dir
	}{
		{
			name:     "success - create dataset",
			args:     []string{"aws-security", "--store", "my-s3"},
			wantFile: "datasets/aws-security/dataset.yaml",
		},
		{
			name:      "invalid name",
			args:      []string{"AB"},
			wantErr:   true,
			errSubstr: "invalid dataset name",
		},
		{
			name: "duplicate dataset",
			args: []string{"existing"},
			setup: func(dir string) {
				datasetDir := filepath.Join(dir, "datasets", "existing")
				if err := os.MkdirAll(datasetDir, 0755); err != nil {
					t.Fatal(err)
				}
				data, _ := yaml.Marshal(&contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
					Metadata: contracts.DataSetMetadata{Name: "existing"},
					Spec:     contracts.DataSetSpec{Store: "my-s3"},
				})
				if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "force overwrite existing",
			args: []string{"existing", "--force"},
			setup: func(dir string) {
				datasetDir := filepath.Join(dir, "datasets", "existing")
				if err := os.MkdirAll(datasetDir, 0755); err != nil {
					t.Fatal(err)
				}
				data, _ := yaml.Marshal(&contracts.DataSetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "DataSet",
					Metadata: contracts.DataSetMetadata{Name: "existing"},
					Spec:     contracts.DataSetSpec{Store: "old-store"},
				})
				if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantFile: "datasets/existing/dataset.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(tmpDir)
			}

			// Change to temp directory
			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)

			// Reset global flags to avoid state leaking between tests
			datasetCreateForce = false
			datasetCreateStore = ""

			// Execute command
			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"dataset", "create"}, tt.args...))

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errSubstr)) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file was created
			if tt.wantFile != "" {
				fullPath := filepath.Join(tmpDir, tt.wantFile)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("expected file %s not found", tt.wantFile)
				}
			}
		})
	}
}
