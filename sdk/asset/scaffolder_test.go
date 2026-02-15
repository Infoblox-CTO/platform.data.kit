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
		wantType  contracts.AssetType
		wantDir   string // relative to project dir
		wantErr   bool
		errSubstr string
	}{
		{
			name: "source asset from embedded schema",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:         "aws-security",
					ExtensionFQN: "cloudquery.source.aws",
					ProjectDir:   dir,
					Version:      "v24.0.2",
				}
			},
			wantType: contracts.AssetTypeSource,
			wantDir:  "assets/sources/aws-security",
		},
		{
			name: "sink asset placement",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:         "my-sink",
					ExtensionFQN: "infoblox.sink.snowflake",
					ProjectDir:   dir,
					Version:      "v1.0.0",
				}
			},
			wantType: contracts.AssetTypeSink,
			wantDir:  "assets/sinks/my-sink",
		},
		{
			name: "model-engine asset placement",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:         "dbt-transform",
					ExtensionFQN: "dbt.model-engine.core",
					ProjectDir:   dir,
					Version:      "v2.0.0",
				}
			},
			wantType: contracts.AssetTypeModelEngine,
			wantDir:  "assets/models/dbt-transform",
		},
		{
			name: "invalid name - too short",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:         "ab",
					ExtensionFQN: "cloudquery.source.aws",
					ProjectDir:   dir,
				}
			},
			wantErr:   true,
			errSubstr: "invalid asset name",
		},
		{
			name: "invalid name - uppercase",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:         "MyAsset",
					ExtensionFQN: "cloudquery.source.aws",
					ProjectDir:   dir,
				}
			},
			wantErr:   true,
			errSubstr: "invalid asset name",
		},
		{
			name: "invalid FQN",
			opts: func(dir string) ScaffoldOpts {
				return ScaffoldOpts{
					Name:         "my-asset",
					ExtensionFQN: "bad-fqn",
					ProjectDir:   dir,
				}
			},
			wantErr:   true,
			errSubstr: "invalid extension FQN",
		},
		{
			name: "duplicate name detection",
			opts: func(dir string) ScaffoldOpts {
				// Pre-create an asset
				writeAssetYAML(t, filepath.Join(dir, "assets", "sources", "existing"), &contracts.AssetManifest{
					APIVersion: "cdpp.io/v1alpha1", Kind: "Asset", Name: "existing",
					Type: contracts.AssetTypeSource, Extension: "cloudquery.source.aws",
					Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"tables": []any{"t1"}},
				})
				return ScaffoldOpts{
					Name:         "existing",
					ExtensionFQN: "cloudquery.source.aws",
					ProjectDir:   dir,
				}
			},
			wantErr:   true,
			errSubstr: "already exists",
		},
		{
			name: "force overwrite",
			opts: func(dir string) ScaffoldOpts {
				// Pre-create an asset
				writeAssetYAML(t, filepath.Join(dir, "assets", "sources", "existing"), &contracts.AssetManifest{
					APIVersion: "cdpp.io/v1alpha1", Kind: "Asset", Name: "existing",
					Type: contracts.AssetTypeSource, Extension: "cloudquery.source.aws",
					Version: "v1.0.0", OwnerTeam: "team", Config: map[string]any{"tables": []any{"t1"}},
				})
				return ScaffoldOpts{
					Name:         "existing",
					ExtensionFQN: "cloudquery.source.aws",
					ProjectDir:   dir,
					Force:        true,
					Version:      "v24.0.2",
				}
			},
			wantType: contracts.AssetTypeSource,
			wantDir:  "assets/sources/existing",
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

			if asset.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", asset.Type, tt.wantType)
			}
			if asset.Name != opts.Name {
				t.Errorf("Name = %q, want %q", asset.Name, opts.Name)
			}
			if asset.Extension != opts.ExtensionFQN {
				t.Errorf("Extension = %q, want %q", asset.Extension, opts.ExtensionFQN)
			}
		})
	}
}

func TestScaffold_PlaceholderConfig(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := Scaffold(ScaffoldOpts{
		Name:         "aws-security",
		ExtensionFQN: "cloudquery.source.aws",
		ProjectDir:   tmpDir,
		Version:      "v24.0.2",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The embedded cloudquery.source schema has required: [accounts, regions, tables]
	// These should appear as placeholder config entries
	asset := result.Asset
	if _, ok := asset.Config["accounts"]; !ok {
		t.Error("config should contain 'accounts' placeholder from schema")
	}
	if _, ok := asset.Config["regions"]; !ok {
		t.Error("config should contain 'regions' placeholder from schema")
	}
	if _, ok := asset.Config["tables"]; !ok {
		t.Error("config should contain 'tables' placeholder from schema")
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
