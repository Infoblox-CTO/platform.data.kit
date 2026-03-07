package asset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

func TestLoadAsset(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string) string // returns path to pass to LoadAsset
		wantName  string
		wantStore string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid asset from directory",
			setup: func(dir string) string {
				assetDir := filepath.Join(dir, "assets", "my-source")
				writeAssetYAML(t, assetDir, &contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1",
					Kind:       "Asset",
					Metadata:   contracts.AssetMetadata{Name: "my-source"},
					Spec:       contracts.AssetSpec{Store: "my-s3", Prefix: "raw/"},
				})
				return assetDir
			},
			wantName:  "my-source",
			wantStore: "my-s3",
		},
		{
			name: "valid asset from file path",
			setup: func(dir string) string {
				assetDir := filepath.Join(dir, "assets", "my-sink")
				writeAssetYAML(t, assetDir, &contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1",
					Kind:       "Asset",
					Metadata:   contracts.AssetMetadata{Name: "my-sink"},
					Spec:       contracts.AssetSpec{Store: "my-pg", Table: "public.events"},
				})
				return filepath.Join(assetDir, "asset.yaml")
			},
			wantName:  "my-sink",
			wantStore: "my-pg",
		},
		{
			name: "missing file",
			setup: func(dir string) string {
				return filepath.Join(dir, "nonexistent")
			},
			wantErr:   true,
			errSubstr: "not found",
		},
		{
			name: "malformed YAML",
			setup: func(dir string) string {
				assetDir := filepath.Join(dir, "bad")
				if err := os.MkdirAll(assetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(assetDir, "asset.yaml"), []byte("not: [valid: yaml: :::"), 0644); err != nil {
					t.Fatal(err)
				}
				return assetDir
			},
			wantErr:   true,
			errSubstr: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)

			asset, err := LoadAsset(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if asset.Metadata.Name != tt.wantName {
				t.Errorf("Metadata.Name = %q, want %q", asset.Metadata.Name, tt.wantName)
			}
			if asset.Spec.Store != tt.wantStore {
				t.Errorf("Spec.Store = %q, want %q", asset.Spec.Store, tt.wantStore)
			}
		})
	}
}

func TestLoadAllAssets(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string)
		wantCount int
		wantNames []string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "three assets in flat layout",
			setup: func(dir string) {
				writeAssetYAML(t, filepath.Join(dir, "assets", "src-a"), &contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "Asset",
					Metadata: contracts.AssetMetadata{Name: "src-a"},
					Spec:     contracts.AssetSpec{Store: "my-s3", Prefix: "raw/"},
				})
				writeAssetYAML(t, filepath.Join(dir, "assets", "sink-b"), &contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "Asset",
					Metadata: contracts.AssetMetadata{Name: "sink-b"},
					Spec:     contracts.AssetSpec{Store: "my-pg", Table: "public.events"},
				})
				writeAssetYAML(t, filepath.Join(dir, "assets", "model-c"), &contracts.AssetManifest{
					APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "Asset",
					Metadata: contracts.AssetMetadata{Name: "model-c"},
					Spec:     contracts.AssetSpec{Store: "my-pg", Table: "public.enriched"},
				})
			},
			wantCount: 3,
			wantNames: []string{"src-a", "sink-b", "model-c"},
		},
		{
			name:      "no assets directory",
			setup:     func(dir string) {},
			wantCount: 0,
		},
		{
			name: "empty assets directory",
			setup: func(dir string) {
				if err := os.MkdirAll(filepath.Join(dir, "assets"), 0755); err != nil {
					t.Fatal(err)
				}
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setup(tmpDir)

			assets, err := LoadAllAssets(tmpDir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !containsString(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(assets) != tt.wantCount {
				t.Fatalf("got %d assets, want %d", len(assets), tt.wantCount)
			}

			if tt.wantNames != nil {
				names := make(map[string]bool)
				for _, a := range assets {
					names[a.Metadata.Name] = true
				}
				for _, n := range tt.wantNames {
					if !names[n] {
						t.Errorf("expected asset %q not found", n)
					}
				}
			}
		})
	}
}

func TestFindAssetByName(t *testing.T) {
	tmpDir := t.TempDir()
	writeAssetYAML(t, filepath.Join(tmpDir, "assets", "my-source"), &contracts.AssetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1", Kind: "Asset",
		Metadata: contracts.AssetMetadata{Name: "my-source"},
		Spec:     contracts.AssetSpec{Store: "my-s3", Prefix: "raw/"},
	})

	// Found
	asset, err := FindAssetByName(tmpDir, "my-source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset.Metadata.Name != "my-source" {
		t.Errorf("Metadata.Name = %q, want %q", asset.Metadata.Name, "my-source")
	}

	// Not found
	_, err = FindAssetByName(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent asset")
	}
}

func TestAssetPath(t *testing.T) {
	path := AssetPath("/project", "my-source")
	want := filepath.Join("/project", "assets", "my-source", "asset.yaml")
	if path != want {
		t.Errorf("AssetPath = %q, want %q", path, want)
	}
}

func TestAssetDir(t *testing.T) {
	dir := AssetDir("/project", "my-source")
	want := filepath.Join("/project", "assets", "my-source")
	if dir != want {
		t.Errorf("AssetDir = %q, want %q", dir, want)
	}
}

// writeAssetYAML is a test helper that creates an asset.yaml file in the given directory.
func writeAssetYAML(t *testing.T, dir string, asset *contracts.AssetManifest) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(asset)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "asset.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

// containsString checks if s contains the substring sub.
func containsString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
