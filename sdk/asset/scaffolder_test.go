package asset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestScaffold(t *testing.T) {
	tests := []struct {
		name      string
		opts      func(dir string) ScaffoldOpts
		wantDir   string // relative to project dir
		wantErr   bool
		errSubstr string
	}{
		{
			name: "basic asset",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:       "aws-security",
					ProjectDir: dir,
					Store:      "my-s3",
				}
			},
			wantDir: "assets/aws-security",
		},
		{
			name: "invalid name - too short",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:       "ab",
					ProjectDir: dir,
				}
			},
			wantErr:   true,
			errSubstr: "invalid asset name",
		},
		{
			name: "invalid name - uppercase",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:       "MyAsset",
					ProjectDir: dir,
				}
			},
			wantErr:   true,
			errSubstr: "invalid asset name",
		},
		{
			name: "duplicate name detection",
			opts: func(dir string) ScaffoldOpts {
				// Pre-create an asset in flat layout
				writeAssetYAML(t, filepath.Join(dir, "assets", "existing"), &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset",
					Metadata: contracts.AssetMetadata{Name: "existing"},
					Spec:     contracts.AssetSpec{Store: "my-s3"},
				})
				return ScaffoldOpts{
					Name:       "existing",
					ProjectDir: dir,
				}
			},
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "force overwrite",
			opts: func(dir string) ScaffoldOpts {
				// Pre-create an asset in flat layout
				writeAssetYAML(t, filepath.Join(dir, "assets", "existing"), &contracts.AssetManifest{
					APIVersion: "data.infoblox.com/v1alpha1", Kind: "Asset",
					Metadata: contracts.AssetMetadata{Name: "existing"},
					Spec:     contracts.AssetSpec{Store: "old-store"},
				})
				return ScaffoldOpts{
					Name:       "existing",
					ProjectDir: dir,
					Store:      "new-store",
					Force:      true,
				}
			},
			wantDir: "assets/existing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			opts := tt.opts(tmpDir)

			result, err := Scaffold(opts)
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

			// Verify the file was created
			if _, err := os.Stat(result.AssetPath); os.IsNotExist(err) {
				t.Fatalf("asset.yaml not created at %s", result.AssetPath)
			}

			// Verify the directory placement
			expectedDir := filepath.Join(tmpDir, tt.wantDir)
			if result.AssetDir != expectedDir {
				t.Errorf("AssetDir = %q, want %q", result.AssetDir, expectedDir)
			}

			// Load and verify the asset
			asset, err := LoadAsset(result.AssetDir)
			if err != nil {
				t.Fatalf("failed to reload asset: %v", err)
			}

			if asset.Metadata.Name != opts.Name {
				t.Errorf("Metadata.Name = %q, want %q", asset.Metadata.Name, opts.Name)
			}
			if asset.Kind != "Asset" {
				t.Errorf("Kind = %q, want %q", asset.Kind, "Asset")
			}
		})
	}
}

func TestValidateAssetName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "my-source", wantErr: false},
		{name: "valid with numbers", input: "source-123", wantErr: false},
		{name: "valid minimum length", input: "abc", wantErr: false},
		{name: "too short", input: "ab", wantErr: true},
		{name: "uppercase", input: "MySource", wantErr: true},
		{name: "starts with number", input: "1source", wantErr: true},
		{name: "has underscore", input: "my_source", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssetName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssetName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestExtractSchemaFields(t *testing.T) {
	schema := `{
		"type": "object",
		"required": ["accounts", "tables"],
		"properties": {
			"accounts": {
				"type": "array",
				"description": "List of AWS account IDs"
			},
			"tables": {
				"type": "array",
				"description": "Tables to sync"
			},
			"concurrency": {
				"type": "integer",
				"default": 10000,
				"description": "Max concurrent resolvers"
			}
		}
	}`

	fields, err := ExtractSchemaFields([]byte(schema))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(fields))
	}

	// Fields should be sorted alphabetically
	if fields[0].Name != "accounts" {
		t.Errorf("fields[0].Name = %q, want %q", fields[0].Name, "accounts")
	}
	if !fields[0].Required {
		t.Error("accounts should be required")
	}
	if fields[0].Description != "List of AWS account IDs" {
		t.Errorf("accounts description = %q", fields[0].Description)
	}

	// concurrency should not be required
	found := false
	for _, f := range fields {
		if f.Name == "concurrency" {
			found = true
			if f.Required {
				t.Error("concurrency should not be required")
			}
			if f.Default != float64(10000) {
				t.Errorf("concurrency default = %v", f.Default)
			}
		}
	}
	if !found {
		t.Error("concurrency field not found")
	}
}
