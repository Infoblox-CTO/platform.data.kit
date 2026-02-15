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
				writeListAsset(t, dir, "sources", "aws-security", contracts.AssetTypeSource, "cloudquery.source.aws", "v24.0.2", "team-a")
				writeListAsset(t, dir, "sinks", "raw-output", contracts.AssetTypeSink, "cloudquery.sink.s3", "v1.2.0", "team-b")
				writeListAsset(t, dir, "sources", "gcp-infra", contracts.AssetTypeSource, "cloudquery.source.gcp", "v10.0.0", "team-c")
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
				writeListAsset(t, dir, "sources", "my-source", contracts.AssetTypeSource, "cloudquery.source.aws", "v1.0.0", "data-team")
			},
		},
		{
			name: "table includes header columns",
			args: []string{},
			setup: func(dir string) {
				writeListAsset(t, dir, "sources", "my-source", contracts.AssetTypeSource, "cloudquery.source.aws", "v1.0.0", "data-team")
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

	writeListAsset(t, tmpDir, "sources", "aws-security", contracts.AssetTypeSource, "cloudquery.source.aws", "v24.0.2", "team-a")
	writeListAsset(t, tmpDir, "sinks", "raw-output", contracts.AssetTypeSink, "cloudquery.sink.s3", "v1.2.0", "team-b")
	writeListAsset(t, tmpDir, "sources", "gcp-infra", contracts.AssetTypeSource, "cloudquery.source.gcp", "v10.0.0", "team-c")

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

	// Verify table header
	if !strings.Contains(output, "NAME") {
		t.Error("table should contain NAME header")
	}
	if !strings.Contains(output, "TYPE") {
		t.Error("table should contain TYPE header")
	}
	if !strings.Contains(output, "EXTENSION") {
		t.Error("table should contain EXTENSION header")
	}
	if !strings.Contains(output, "VERSION") {
		t.Error("table should contain VERSION header")
	}
	if !strings.Contains(output, "OWNER") {
		t.Error("table should contain OWNER header")
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

	writeListAsset(t, tmpDir, "sources", "my-source", contracts.AssetTypeSource, "cloudquery.source.aws", "v1.0.0", "data-team")

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
	if entry.Type != "source" {
		t.Errorf("expected type 'source', got %q", entry.Type)
	}
	if entry.Extension != "cloudquery.source.aws" {
		t.Errorf("expected extension 'cloudquery.source.aws', got %q", entry.Extension)
	}
	if entry.Version != "v1.0.0" {
		t.Errorf("expected version 'v1.0.0', got %q", entry.Version)
	}
	if entry.OwnerTeam != "data-team" {
		t.Errorf("expected ownerTeam 'data-team', got %q", entry.OwnerTeam)
	}
}

// writeListAsset creates an asset.yaml for use in list tests.
func writeListAsset(t *testing.T, projectDir, typeDir, name string, assetType contracts.AssetType, ext, version, owner string) {
	t.Helper()

	assetDir := filepath.Join(projectDir, "assets", typeDir, name)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	a := &contracts.AssetManifest{
		APIVersion: "cdpp.io/v1alpha1",
		Kind:       "Asset",
		Name:       name,
		Type:       assetType,
		Extension:  ext,
		Version:    version,
		OwnerTeam:  owner,
		Config: map[string]any{
			"accounts": []any{"123456789012"},
			"regions":  []any{"us-east-1"},
			"tables":   []any{"aws_s3_buckets"},
		},
	}
	data, _ := yaml.Marshal(a)
	if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
