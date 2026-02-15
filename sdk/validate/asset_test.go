package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
)

func TestAssetValidator_ValidateAsset(t *testing.T) {
	tests := []struct {
		name       string
		asset      *contracts.AssetManifest
		offline    bool
		wantErrors int
		wantCodes  []string
	}{
		{
			name: "valid asset",
			asset: &contracts.AssetManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Asset",
				Name:       "aws-security",
				Type:       contracts.AssetTypeSource,
				Extension:  "cloudquery.source.aws",
				Version:    "v24.0.2",
				OwnerTeam:  "security-data",
				Config: map[string]any{
					"accounts": []any{"123456789012"},
					"regions":  []any{"us-east-1"},
					"tables":   []any{"aws_s3_buckets"},
				},
			},
			wantErrors: 0,
		},
		{
			name:       "nil asset",
			asset:      nil,
			wantErrors: 1,
			wantCodes:  []string{ErrAssetRequired},
		},
		{
			name: "missing required config field",
			asset: &contracts.AssetManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Asset",
				Name:       "aws-missing",
				Type:       contracts.AssetTypeSource,
				Extension:  "cloudquery.source.aws",
				Version:    "v24.0.2",
				OwnerTeam:  "team",
				Config:     map[string]any{"accounts": []any{"123456789012"}},
				// Missing "regions" and "tables" required fields
			},
			wantErrors: 1, // Schema validation catches multiple missing fields as one error tree
			wantCodes:  []string{ErrAssetSchemaValidation},
		},
		{
			name: "wrong config type",
			asset: &contracts.AssetManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Asset",
				Name:       "aws-wrong-type",
				Type:       contracts.AssetTypeSource,
				Extension:  "cloudquery.source.aws",
				Version:    "v24.0.2",
				OwnerTeam:  "team",
				Config: map[string]any{
					"accounts": "not-an-array", // Should be array
					"regions":  []any{"us-east-1"},
					"tables":   []any{"aws_s3_buckets"},
				},
			},
			wantErrors: 1,
			wantCodes:  []string{ErrAssetSchemaValidation},
		},
		{
			name: "invalid FQN",
			asset: &contracts.AssetManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Asset",
				Name:       "bad-fqn",
				Type:       contracts.AssetTypeSource,
				Extension:  "bad-fqn",
				Version:    "v1.0.0",
				OwnerTeam:  "team",
				Config:     map[string]any{"k": "v"},
			},
			offline:    true,
			wantErrors: 1,
			wantCodes:  []string{ErrAssetInvalidFQN},
		},
		{
			name: "invalid version",
			asset: &contracts.AssetManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Asset",
				Name:       "bad-version",
				Type:       contracts.AssetTypeSource,
				Extension:  "cloudquery.source.aws",
				Version:    "not-semver",
				OwnerTeam:  "team",
				Config:     map[string]any{"k": "v"},
			},
			offline:    true,
			wantErrors: 1,
			wantCodes:  []string{ErrAssetInvalidVersion},
		},
		{
			name: "type mismatch with FQN kind",
			asset: &contracts.AssetManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Asset",
				Name:       "type-mismatch",
				Type:       contracts.AssetTypeSink, // FQN says "source"
				Extension:  "cloudquery.source.aws",
				Version:    "v1.0.0",
				OwnerTeam:  "team",
				Config:     map[string]any{"k": "v"},
			},
			offline:    true,
			wantErrors: 1,
			wantCodes:  []string{ErrAssetTypeMismatch},
		},
		{
			name: "empty config with no required fields (offline)",
			asset: &contracts.AssetManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Asset",
				Name:       "empty-config",
				Type:       contracts.AssetTypeSource,
				Extension:  "cloudquery.source.aws",
				Version:    "v1.0.0",
				OwnerTeam:  "team",
				Config:     map[string]any{},
			},
			offline:    true,
			wantErrors: 0,
		},
		{
			name:       "all required fields missing",
			asset:      &contracts.AssetManifest{},
			offline:    true,
			wantErrors: 6, // apiVersion, kind (invalid), name, extension, version, ownerTeam, config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v *AssetValidator
			if tt.offline {
				v = NewOfflineAssetValidator()
			} else {
				v = NewAssetValidator(asset.NewEmbeddedResolver())
			}

			errs := v.ValidateAsset(context.Background(), tt.asset)

			if tt.wantErrors > 0 && len(errs) == 0 {
				t.Fatalf("expected %d+ errors, got 0", tt.wantErrors)
			}

			if tt.wantErrors == 0 && len(errs) > 0 {
				t.Fatalf("expected no errors, got %d: %v", len(errs), errs)
			}

			if tt.wantCodes != nil {
				for _, code := range tt.wantCodes {
					found := false
					for _, e := range errs {
						if e.Code == code {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error code %s not found in errors: %v", code, errs)
					}
				}
			}
		})
	}
}

func TestAssetValidator_Name(t *testing.T) {
	v := NewAssetValidator(nil)
	if v.Name() != "asset" {
		t.Errorf("Name() = %q, want %q", v.Name(), "asset")
	}
}
