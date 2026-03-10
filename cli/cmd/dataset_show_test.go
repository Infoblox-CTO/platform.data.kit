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

func TestDataSetShow(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setup      func(dir string)
		wantErr    bool
		errSubstr  string
		wantOutput string
	}{
		{
			name: "valid dataset name - yaml output",
			args: []string{"aws-security"},
			setup: func(dir string) {
				writeShowDataSet(t, dir, "aws-security", "my-s3")
			},
			wantOutput: "aws-security",
		},
		{
			name:      "non-existent dataset name",
			args:      []string{"ghost"},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "json output",
			args: []string{"aws-security", "--output", "json"},
			setup: func(dir string) {
				writeShowDataSet(t, dir, "aws-security", "my-s3")
			},
			wantOutput: "aws-security",
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
			datasetShowOutput = "yaml"

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"dataset", "show"}, tt.args...))

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
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

func TestDataSetShowYAMLOutput(t *testing.T) {
	tmpDir := t.TempDir()

	writeShowDataSet(t, tmpDir, "aws-security", "my-s3")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	datasetShowOutput = "yaml"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"dataset", "show", "aws-security"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify YAML contains expected fields
	var parsed contracts.DataSetManifest
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\nOutput: %s", err, output)
	}

	if parsed.Metadata.Name != "aws-security" {
		t.Errorf("expected name 'aws-security', got %q", parsed.Metadata.Name)
	}
	if parsed.Spec.Store != "my-s3" {
		t.Errorf("expected store 'my-s3', got %q", parsed.Spec.Store)
	}
}

func TestDataSetShowJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	writeShowDataSet(t, tmpDir, "aws-security", "my-s3")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	datasetShowOutput = "yaml"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"dataset", "show", "aws-security", "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the JSON output
	var parsed contracts.DataSetManifest
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if parsed.Metadata.Name != "aws-security" {
		t.Errorf("expected name 'aws-security', got %q", parsed.Metadata.Name)
	}
	if parsed.Spec.Store != "my-s3" {
		t.Errorf("expected store 'my-s3', got %q", parsed.Spec.Store)
	}
}

// writeShowDataSet creates a dataset.yaml for use in show tests.
func writeShowDataSet(t *testing.T, projectDir, name, store string) {
	t.Helper()

	datasetDir := filepath.Join(projectDir, "datasets", name)
	if err := os.MkdirAll(datasetDir, 0755); err != nil {
		t.Fatal(err)
	}

	a := &contracts.DataSetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "DataSet",
		Metadata:   contracts.DataSetMetadata{Name: name},
		Spec:       contracts.DataSetSpec{Store: store, Table: "public.events"},
	}
	data, _ := yaml.Marshal(a)
	if err := os.WriteFile(filepath.Join(datasetDir, "dataset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
