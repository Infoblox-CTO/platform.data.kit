package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

func TestAssetValidate(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		setup      func(dir string)
		wantErr    bool
		wantOutput string // substring expected in output
	}{
		{
			name: "valid single asset by path",
			args: []string{"assets/sources/aws-security"},
			setup: func(dir string) {
				writeValidAsset(t, dir, "sources", "aws-security")
			},
		},
		{
			name: "valid all assets in project",
			args: []string{"--offline"},
			setup: func(dir string) {
				writeValidAsset(t, dir, "sources", "aws-security")
				writeValidAsset(t, dir, "sinks", "raw-output")
			},
			wantOutput: "All 2 assets are valid",
		},
		{
			name: "invalid asset - wrong config type",
			args: []string{"assets/sources/bad-asset"},
			setup: func(dir string) {
				assetDir := filepath.Join(dir, "assets", "sources", "bad-asset")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				a := &contracts.AssetManifest{
					APIVersion: "cdpp.io/v1alpha1",
					Kind:       "Asset",
					Name:       "bad-asset",
					Type:       contracts.AssetTypeSource,
					Extension:  "cloudquery.source.aws",
					Version:    "v1.0.0",
					OwnerTeam:  "team",
					Config: map[string]any{
						"accounts": "not-an-array", // wrong type
						"regions":  []any{"us-east-1"},
						"tables":   []any{"t"},
					},
				}
				data, _ := yaml.Marshal(a)
				if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
		{
			name:    "non-existent path",
			args:    []string{"assets/sources/ghost"},
			wantErr: true,
		},
		{
			name: "offline mode skips schema validation",
			args: []string{"--offline", "assets/sources/aws-security"},
			setup: func(dir string) {
				// Asset with incorrect config but offline mode won't catch it
				assetDir := filepath.Join(dir, "assets", "sources", "aws-security")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				a := &contracts.AssetManifest{
					APIVersion: "cdpp.io/v1alpha1",
					Kind:       "Asset",
					Name:       "aws-security",
					Type:       contracts.AssetTypeSource,
					Extension:  "cloudquery.source.aws",
					Version:    "v1.0.0",
					OwnerTeam:  "team",
					Config:     map[string]any{"wrong": "config"},
				}
				data, _ := yaml.Marshal(a)
				if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name:       "no assets directory",
			args:       []string{},
			wantOutput: "No assets found",
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
			assetValidateOffline = false

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"asset", "validate"}, tt.args...))

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

// writeValidAsset creates a valid asset.yaml in the appropriate type directory.
func writeValidAsset(t *testing.T, projectDir, typeDir, name string) {
	t.Helper()

	assetDir := filepath.Join(projectDir, "assets", typeDir, name)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	assetType := contracts.AssetTypeSource
	ext := "cloudquery.source.aws"
	switch typeDir {
	case "sinks":
		assetType = contracts.AssetTypeSink
		ext = "cloudquery.sink.s3"
	case "model-engines":
		assetType = contracts.AssetTypeModelEngine
		ext = "dp.model-engine.dbt"
	}

	a := &contracts.AssetManifest{
		APIVersion: "cdpp.io/v1alpha1",
		Kind:       "Asset",
		Name:       name,
		Type:       assetType,
		Extension:  ext,
		Version:    "v1.0.0",
		OwnerTeam:  "data-team",
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
