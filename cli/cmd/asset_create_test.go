package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

func TestAssetCreate(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(dir string)
		wantErr   bool
		errSubstr string
		wantFile  string // relative to project dir
	}{
		{
			name:     "success - create source asset",
			args:     []string{"aws-security", "--ext", "cloudquery.source.aws", "--version", "v24.0.2"},
			wantFile: "assets/sources/aws-security/asset.yaml",
		},
		{
			name:    "missing --ext flag",
			args:    []string{"my-asset"},
			wantErr: true,
		},
		{
			name:      "invalid FQN",
			args:      []string{"my-asset", "--ext", "bad-fqn"},
			wantErr:   true,
			errSubstr: "invalid extension FQN",
		},
		{
			name:      "invalid name",
			args:      []string{"AB", "--ext", "cloudquery.source.aws"},
			wantErr:   true,
			errSubstr: "invalid asset name",
		},
		{
			name: "duplicate asset",
			args: []string{"existing", "--ext", "cloudquery.source.aws"},
			setup: func(dir string) {
				assetDir := filepath.Join(dir, "assets", "sources", "existing")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				data, _ := yaml.Marshal(&contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset", Name: "existing",
					Type: contracts.AssetTypeSource, Extension: "cloudquery.source.aws",
					Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"tables": []any{"t1"}},
				})
				if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "force overwrite existing",
			args: []string{"existing", "--ext", "cloudquery.source.aws", "--force"},
			setup: func(dir string) {
				assetDir := filepath.Join(dir, "assets", "sources", "existing")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				data, _ := yaml.Marshal(&contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset", Name: "existing",
					Type: contracts.AssetTypeSource, Extension: "cloudquery.source.aws",
					Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"tables": []any{"t1"}},
				})
				if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantFile: "assets/sources/existing/asset.yaml",
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
			assetCreateExt = ""
			assetCreateForce = false
			assetCreateInteractive = false
			assetCreateVersion = ""

			// Execute command
			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"asset", "create"}, tt.args...))

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
