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
			args: []string{"assets/aws-security"},
			setup: func(dir string) {
				writeValidateAsset(t, dir, "aws-security", "s3://bucket")
			},
		},
		{
			name: "valid all assets in project",
			args: []string{"--offline"},
			setup: func(dir string) {
				writeValidateAsset(t, dir, "aws-security", "s3://bucket")
				writeValidateAsset(t, dir, "raw-output", "pg://db")
			},
			wantOutput: "All 2 assets are valid",
		},
		{
			name: "invalid asset - missing store",
			args: []string{"assets/bad-asset"},
			setup: func(dir string) {
				assetDir := filepath.Join(dir, "assets", "bad-asset")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				a := &contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1",
					Kind:       "Asset",
					Metadata: contracts.AssetMetadata{
						Name: "bad-asset",
					},
					Spec: contracts.AssetSpec{
						// Store intentionally omitted
						Table: "some_table",
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
			args:    []string{"assets/ghost"},
			wantErr: true,
		},
		{
			name: "offline mode skips schema validation",
			args: []string{"--offline", "assets/aws-security"},
			setup: func(dir string) {
				writeValidateAsset(t, dir, "aws-security", "s3://bucket")
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

// writeValidateAsset creates a valid asset.yaml in the flat assets/<name>/ layout.
func writeValidateAsset(t *testing.T, projectDir, name, store string) {
	t.Helper()

	assetDir := filepath.Join(projectDir, "assets", name)
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}

	a := &contracts.AssetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "Asset",
		Metadata: contracts.AssetMetadata{
			Name: name,
		},
		Spec: contracts.AssetSpec{
			Store:          store,
			Table:          "default_table",
			Classification: "internal",
		},
	}
	data, _ := yaml.Marshal(a)
	if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
