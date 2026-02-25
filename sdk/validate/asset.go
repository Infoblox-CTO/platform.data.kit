package validate

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
)

// Asset validation error codes.
const (
	ErrAssetRequired         = "E070"
	ErrAssetInvalidFQN       = "E071" // Deprecated: extensions removed in new model
	ErrAssetInvalidVersion   = "E072" // Deprecated: assets no longer individually versioned
	ErrAssetTypeMismatch     = "E073" // Deprecated: assets no longer have types
	ErrAssetSchemaValidation = "E074" // Deprecated: extension schema validation removed
	ErrAssetExtNotFound      = "E075" // Deprecated: extensions removed in new model
	ErrAssetRefNotFound      = "E076"
	ErrAssetBindingNotFound  = "E077"
)

var (
	assetNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,62}$`)
)

// AssetValidator validates asset.yaml files against structural rules.
type AssetValidator struct {
	resolver asset.SchemaResolver
	offline  bool
}

// NewAssetValidator creates a new AssetValidator.
// If resolver is nil, the default embedded resolver is used.
func NewAssetValidator(resolver asset.SchemaResolver) *AssetValidator {
	if resolver == nil {
		resolver = asset.DefaultResolver()
	}
	return &AssetValidator{resolver: resolver}
}

// NewOfflineAssetValidator creates a validator that only does structural validation.
func NewOfflineAssetValidator() *AssetValidator {
	return &AssetValidator{offline: true}
}

// Name returns the validator name.
func (v *AssetValidator) Name() string {
	return "asset"
}

// ValidateAsset validates a single asset manifest.
func (v *AssetValidator) ValidateAsset(_ context.Context, a *contracts.AssetManifest) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if a == nil {
		errs.AddError(ErrAssetRequired, "", "asset manifest is nil")
		return errs
	}

	// Structural validation
	v.validateStructure(&errs, a)

	return errs
}

// validateStructure performs structural validation of the asset manifest.
func (v *AssetValidator) validateStructure(errs *contracts.ValidationErrors, a *contracts.AssetManifest) {
	// Validate required fields
	if a.APIVersion == "" {
		errs.AddError(ErrAssetRequired, "apiVersion", "apiVersion is required")
	}
	if a.Kind == "" {
		errs.AddError(ErrAssetRequired, "kind", "kind is required")
	} else if a.Kind != "Asset" {
		errs.AddError(ErrInvalidFormat, "kind", "kind must be 'Asset'")
	}

	// Validate name
	if a.Metadata.Name == "" {
		errs.AddError(ErrAssetRequired, "metadata.name", "metadata.name is required")
	} else if !assetNamePattern.MatchString(a.Metadata.Name) {
		errs.AddErrorWithValue(ErrInvalidFormat, "metadata.name",
			"name must be DNS-safe (lowercase, 3-63 chars, start with letter)", a.Metadata.Name)
	}

	// Validate store reference
	if a.Spec.Store == "" {
		errs.AddError(ErrAssetRequired, "spec.store", "spec.store is required")
	}

	// Validate that at least one locator is provided
	if a.Spec.Table == "" && a.Spec.Prefix == "" && a.Spec.Topic == "" {
		errs.Add(&contracts.ValidationError{
			Code:     ErrAssetRequired,
			Field:    "spec",
			Message:  "at least one of spec.table, spec.prefix, or spec.topic is recommended",
			Severity: contracts.SeverityWarning,
		})
	}

	// Validate classification if set
	if a.Spec.Classification != "" {
		validClassifications := map[string]bool{
			"public": true, "internal": true, "confidential": true, "restricted": true,
		}
		if !validClassifications[a.Spec.Classification] {
			errs.AddErrorWithValue(ErrInvalidFormat, "spec.classification",
				"classification must be one of: public, internal, confidential, restricted",
				a.Spec.Classification)
		}
	}

	// Validate schema fields
	if len(a.Spec.Schema) > 0 {
		seen := make(map[string]bool)
		for i, f := range a.Spec.Schema {
			if f.Name == "" {
				errs.AddError(ErrAssetRequired, fmt.Sprintf("spec.schema[%d].name", i),
					"schema field name is required")
			} else if seen[f.Name] {
				errs.AddError(ErrInvalidFormat, fmt.Sprintf("spec.schema[%d].name", i),
					fmt.Sprintf("duplicate schema field name: %s", f.Name))
			} else {
				seen[f.Name] = true
			}
			if f.Type == "" {
				errs.AddError(ErrAssetRequired, fmt.Sprintf("spec.schema[%d].type", i),
					"schema field type is required")
			}
		}
	}
}
