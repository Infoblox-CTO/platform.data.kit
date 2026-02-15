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

func TestAssetShow(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setup      func(dir string)
		wantErr    bool
		errSubstr  string
		wantOutput string
	}{
		{
			name: "valid asset name - yaml output",
			args: []string{"aws-security"},
			setup: func(dir string) {
				writeShowAsset(t, dir, "sources", "aws-security", contracts.AssetTypeSource,
					"cloudquery.source.aws", "v24.0.2", "security-team")
			},
			wantOutput: "aws-security",
		},
		{
			name:      "non-existent asset name",
			args:      []string{"ghost"},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "json output",
			args: []string{"aws-security", "--output", "json"},
			setup: func(dir string) {
				writeShowAsset(t, dir, "sources", "aws-security", contracts.AssetTypeSource,
					"cloudquery.source.aws", "v24.0.2", "security-team")
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
			assetShowOutput = "yaml"

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"asset", "show"}, tt.args...))

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

func TestAssetShowYAMLOutput(t *testing.T) {
	tmpDir := t.TempDir()

	writeShowAsset(t, tmpDir, "sources", "aws-security", contracts.AssetTypeSource,
		"cloudquery.source.aws", "v24.0.2", "security-team")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	assetShowOutput = "yaml"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"asset", "show", "aws-security"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify YAML contains expected fields
	var parsed contracts.AssetManifest
	if err := yaml.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\nOutput: %s", err, output)
	}

	if parsed.Name != "aws-security" {
		t.Errorf("expected name 'aws-security', got %q", parsed.Name)
	}
	if parsed.Extension != "cloudquery.source.aws" {
		t.Errorf("expected extension 'cloudquery.source.aws', got %q", parsed.Extension)
	}
	if parsed.Version != "v24.0.2" {
		t.Errorf("expected version 'v24.0.2', got %q", parsed.Version)
	}
	if parsed.OwnerTeam != "security-team" {
		t.Errorf("expected ownerTeam 'security-team', got %q", parsed.OwnerTeam)
	}
}

func TestAssetShowJSONOutput(t *testing.T) {
	tmpDir := t.TempDir()

	writeShowAsset(t, tmpDir, "sources", "aws-security", contracts.AssetTypeSource,
		"cloudquery.source.aws", "v24.0.2", "security-team")

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	assetShowOutput = "yaml"

	buf := new(bytes.Buffer)
	cmd := rootCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"asset", "show", "aws-security", "--output", "json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the JSON output
	var parsed contracts.AssetManifest
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, buf.String())
	}

	if parsed.Name != "aws-security" {
		t.Errorf("expected name 'aws-security', got %q", parsed.Name)
	}
	if parsed.Extension != "cloudquery.source.aws" {
		t.Errorf("expected extension 'cloudquery.source.aws', got %q", parsed.Extension)
	}
	if parsed.Version != "v24.0.2" {
		t.Errorf("expected version 'v24.0.2', got %q", parsed.Version)
	}
}

// writeShowAsset creates an asset.yaml for use in show tests.
func writeShowAsset(t *testing.T, projectDir, typeDir, name string, assetType contracts.AssetType, ext, version, owner string) {
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
