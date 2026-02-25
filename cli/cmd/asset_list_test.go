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

func TestAssetList(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setup      func(dir string)
		wantErr    bool
		wantOutput string // substring expected in output
	}{
		{
			name: "table output with 3 assets",
			args: []string{},
			setup: func(dir string) {
				writeListAsset(t, dir, "aws-security", "my-s3")
				writeListAsset(t, dir, "raw-output", "my-pg")
				writeListAsset(t, dir, "gcp-infra", "my-s3")
			},
			wantOutput: "aws-security",
		},
		{
			name:       "empty project",
			args:       []string{},
			wantOutput: "No assets found",
		},
		{
			name: "json output with assets",
			args: []string{"--output", "json"},
			setup: func(dir string) {
				writeListAsset(t, dir, "my-source", "my-s3")
			},
		},
		{
			name: "table includes header columns",
			args: []string{},
			setup: func(dir string) {
				writeListAsset(t, dir, "my-source", "my-s3")
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
			assetListOutput = "table"

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"asset", "list"}, tt.args...))

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

func TestAssetListTableFormat(t *testing.T) {
	tmpDir := t.TempDir()

	writeListAsset(t, tmpDir, "aws-security", "my-s3")
	writeListAsset(t, tmpDir, "raw-output", "my-pg")
	writeListAsset(t, tmpDir, "gcp-infra", "my-s3")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	assetListOutput = "table"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"asset", "list"})

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

	// Verify each asset appears
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

func TestAssetListJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()

	writeListAsset(t, tmpDir, "my-source", "my-s3")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	assetListOutput = "table"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"asset", "list", "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the JSON output
	var entries []assetListEntry
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

// writeListAsset creates an asset.yaml for use in list tests.
func writeListAsset(t *testing.T, projectDir, name, store string) {
	t.Helper()

	assetDir := filepath.Join(projectDir, "assets", name)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	a := &contracts.AssetManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Asset",
		Metadata:   contracts.AssetMetadata{Name: name},
		Spec:       contracts.AssetSpec{Store: store},
	}
	data, _ := yaml.Marshal(a)
	if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
