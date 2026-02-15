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
		wantType  contracts.AssetType
		wantExt   string
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid asset from directory",
			setup: func(dir string) string {
				assetDir := filepath.Join(dir, "assets", "sources", "my-source")
				writeAssetYAML(t, assetDir, &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1",
					Kind:       "Asset",
					Name:       "my-source",
					Type:       contracts.AssetTypeSource,
					Extension:  "cloudquery.source.aws",
					Version:    "v24.0.2",
					OwnerTeam:  "team-a",
					Config:     map[string]any{"tables": []any{"t1"}},
				})
				return assetDir
			},
			wantName: "my-source",
			wantType: contracts.AssetTypeSource,
			wantExt:  "cloudquery.source.aws",
		},
		{
			name: "valid asset from file path",
			setup: func(dir string) string {
				assetDir := filepath.Join(dir, "assets", "sinks", "my-sink")
				writeAssetYAML(t, assetDir, &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1",
					Kind:       "Asset",
					Name:       "my-sink",
					Type:       contracts.AssetTypeSink,
					Extension:  "infoblox.sink.snowflake",
					Version:    "v1.0.0",
					OwnerTeam:  "team-b",
					Config:     map[string]any{"database": "mydb"},
				})
				return filepath.Join(assetDir, "asset.yaml")
			},
			wantName: "my-sink",
			wantType: contracts.AssetTypeSink,
			wantExt:  "infoblox.sink.snowflake",
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

			if asset.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", asset.Name, tt.wantName)
			}
			if asset.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", asset.Type, tt.wantType)
			}
			if asset.Extension != tt.wantExt {
				t.Errorf("Extension = %q, want %q", asset.Extension, tt.wantExt)
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
			name: "three assets in different type directories",
			setup: func(dir string) {
				writeAssetYAML(t, filepath.Join(dir, "assets", "sources", "src-a"), &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset", Name: "src-a",
					Type: contracts.AssetTypeSource, Extension: "cloudquery.source.aws",
					Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"tables": []any{"t1"}},
				})
				writeAssetYAML(t, filepath.Join(dir, "assets", "sinks", "sink-b"), &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset", Name: "sink-b",
					Type: contracts.AssetTypeSink, Extension: "infoblox.sink.snowflake",
					Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"db": "mydb"},
				})
				writeAssetYAML(t, filepath.Join(dir, "assets", "models", "model-c"), &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset", Name: "model-c",
					Type: contracts.AssetTypeModelEngine, Extension: "dbt.model-engine.core",
					Version: "v2.0.0", OwnerTeam: "team", Config: map[string]any{"target": "prod"},
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
		{
			name: "misplaced asset type",
			setup: func(dir string) {
				// Source asset placed in sinks/ directory
				writeAssetYAML(t, filepath.Join(dir, "assets", "sinks", "wrong"), &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset", Name: "wrong",
					Type: contracts.AssetTypeSource, Extension: "cloudquery.source.aws",
					Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"tables": []any{"t1"}},
				})
			},
			wantErr:   true,
			errSubstr: "expected",
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
					names[a.Name] = true
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
	writeAssetYAML(t, filepath.Join(tmpDir, "assets", "sources", "my-source"), &contracts.AssetManifest{
		APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset", Name: "my-source",
		Type: contracts.AssetTypeSource, Extension: "cloudquery.source.aws",
		Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"tables": []any{"t1"}},
	})

	// Found
	asset, err := FindAssetByName(tmpDir, "my-source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if asset.Name != "my-source" {
		t.Errorf("Name = %q, want %q", asset.Name, "my-source")
	}

	// Not found
	_, err = FindAssetByName(tmpDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent asset")
	}
}

func TestAssetPath(t *testing.T) {
	path := AssetPath("/project", contracts.AssetTypeSource, "my-source")
	want := filepath.Join("/project", "assets", "sources", "my-source", "asset.yaml")
	if path != want {
		t.Errorf("AssetPath = %q, want %q", path, want)
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
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
