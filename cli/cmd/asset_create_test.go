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
			name:     "success - create asset",
			args:     []string{"aws-security", "--store", "my-s3"},
			wantFile: "assets/aws-security/asset.yaml",
		},
		{
			name:      "invalid name",
			args:      []string{"AB"},
			wantErr:   true,
			errSubstr: "invalid asset name",
		},
		{
			name: "duplicate asset",
			args: []string{"existing"},
			setup: func(dir string) {
				assetDir := filepath.Join(dir, "assets", "existing")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				data, _ := yaml.Marshal(&contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "Asset",
					Metadata: contracts.AssetMetadata{Name: "existing"},
					Spec:     contracts.AssetSpec{Store: "my-s3"},
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
			args: []string{"existing", "--force"},
			setup: func(dir string) {
				assetDir := filepath.Join(dir, "assets", "existing")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				data, _ := yaml.Marshal(&contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "Asset",
					Metadata: contracts.AssetMetadata{Name: "existing"},
					Spec:     contracts.AssetSpec{Store: "old-store"},
				})
				if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), data, 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantFile: "assets/existing/asset.yaml",
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
			assetCreateForce = false
			assetCreateStore = ""

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
