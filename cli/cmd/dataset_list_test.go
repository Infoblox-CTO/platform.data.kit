package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

func TestDataSetList(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setup      func(dir string)
		wantErr    bool
		wantOutput string // substring expected in output
	}{
		{
			name: "table output with 3 datasets",
			args: []string{},
			setup: func(dir string) {
				writeListDataSet(t, dir, "aws-security", "my-s3")
				writeListDataSet(t, dir, "raw-output", "my-pg")
				writeListDataSet(t, dir, "gcp-infra", "my-s3")
			},
			wantOutput: "aws-security",
		},
		{
			name:       "empty project",
			args:       []string{},
			wantOutput: "No datasets found",
		},
		{
			name: "json output with datasets",
			args: []string{"--output", "json"},
			setup: func(dir string) {
				writeListDataSet(t, dir, "my-source", "my-s3")
			},
		},
		{
			name: "table includes header columns",
			args: []string{},
			setup: func(dir string) {
				writeListDataSet(t, dir, "my-source", "my-s3")
			},
			wantOutput: "NAME",
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

			// Reset global flags
			datasetListOutput = "table"

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"dataset", "list"}, tt.args...))

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

			output := buf.String()
			if tt.wantOutput != "" {
				if !strings.Contains(output, tt.wantOutput) {
					t.Errorf("output %q should contain %q", output, tt.wantOutput)
				}
			}
		})
	}
}

func TestDataSetListTableFormat(t *testing.T) {
	tmpDir := t.TempDir()

	writeListDataSet(t, tmpDir, "aws-security", "my-s3")
	writeListDataSet(t, tmpDir, "raw-output", "my-pg")
	writeListDataSet(t, tmpDir, "gcp-infra", "my-s3")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	datasetListOutput = "table"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"dataset", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify table headers
	if !strings.Contains(output, "NAME") {
		t.Error("table should contain NAME header")
	}
	if !strings.Contains(output, "STORE") {
		t.Error("table should contain STORE header")
	}

	// Verify each dataset appears
	if !strings.Contains(output, "aws-security") {
		t.Error("table should contain aws-security")
	}
	if !strings.Contains(output, "raw-output") {
		t.Error("table should contain raw-output")
	}
	if !strings.Contains(output, "gcp-infra") {
		t.Error("table should contain gcp-infra")
	}
}

func TestDataSetListJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()

	writeListDataSet(t, tmpDir, "my-source", "my-s3")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	datasetListOutput = "table"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"dataset", "list", "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the JSON output
	var entries []datasetListEntry
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, buf.String())
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Name != "my-source" {
		t.Errorf("expected name 'my-source', got %q", entry.Name)
	}
	if entry.Store != "my-s3" {
		t.Errorf("expected store 'my-s3', got %q", entry.Store)
	}
}

// writeListDataSet creates a dataset.yaml for use in list tests.
func writeListDataSet(t *testing.T, projectDir, name, store string) {
	t.Helper()

	datasetDir := filepath.Join(projectDir, "datasets", name)
	if err := os.MkdirAll(datasetDir, 0755); err != nil {
		t.Fatal(err)
	}

	a := &contracts.DataSetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "DataSet",
		Metadata:   contracts.DataSetMetadata{Name: name},
		Spec:       contracts.DataSetSpec{Store: store},
	}
	data, _ := yaml.Marshal(a)
	if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
