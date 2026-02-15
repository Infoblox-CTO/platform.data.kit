package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// Asset validation error codes.
const (
	ErrAssetRequired         = "E070"
	ErrAssetInvalidFQN       = "E071"
	ErrAssetInvalidVersion   = "E072"
	ErrAssetTypeMismatch     = "E073"
	ErrAssetSchemaValidation = "E074"
	ErrAssetExtNotFound      = "E075"
	ErrAssetRefNotFound      = "E076"
	ErrAssetBindingNotFound  = "E077"
)

var (
	assetNamePattern    = regexp.MustCompile(`^[a-z][a-z0-9-]{2,62}$`)
	assetVersionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$`)
)

// AssetValidator validates asset.yaml files against structural rules and extension schemas.
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
func (v *AssetValidator) ValidateAsset(ctx context.Context, a *contracts.AssetManifest) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if a == nil {
		errs.AddError(ErrAssetRequired, "", "asset manifest is nil")
		return errs
	}

	// Structural validation
	v.validateStructure(&errs, a)

	// Schema validation (if not offline)
	if !v.offline && v.resolver != nil {
		v.validateConfigSchema(ctx, &errs, a)
	}

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
	if a.Name == "" {
		errs.AddError(ErrAssetRequired, "name", "name is required")
	} else if !assetNamePattern.MatchString(a.Name) {
		errs.AddErrorWithValue(ErrInvalidFormat, "name",
			"name must be DNS-safe (lowercase, 3-63 chars, start with letter)", a.Name)
	}

	// Validate type
	if !a.Type.IsValid() {
		errs.AddErrorWithValue(ErrInvalidFormat, "type",
			fmt.Sprintf("type must be one of: %v", contracts.ValidAssetTypes()), string(a.Type))
	}

	// Validate extension FQN
	if a.Extension == "" {
		errs.AddError(ErrAssetRequired, "extension", "extension FQN is required")
	} else {
		_, kind, _, err := contracts.ParseExtensionFQN(a.Extension)
		if err != nil {
			errs.AddErrorWithValue(ErrAssetInvalidFQN, "extension", err.Error(), a.Extension)
		} else {
			// Validate type matches FQN kind
			if a.Type.IsValid() && contracts.AssetType(kind) != a.Type {
				errs.AddError(ErrAssetTypeMismatch, "type",
					fmt.Sprintf("asset type %q does not match extension kind %q (from FQN %q)",
						a.Type, kind, a.Extension))
			}
		}
	}

	// Validate version
	if a.Version == "" {
		errs.AddError(ErrAssetRequired, "version", "version is required")
	} else if !assetVersionPattern.MatchString(a.Version) {
		errs.AddErrorWithValue(ErrAssetInvalidVersion, "version",
			"version must be a valid semver (e.g., v1.0.0)", a.Version)
	}

	// Validate ownerTeam
	if a.OwnerTeam == "" {
		errs.AddError(ErrAssetRequired, "ownerTeam", "ownerTeam is required")
	}

	// Validate config is present
	if a.Config == nil {
		errs.AddError(ErrAssetRequired, "config", "config is required")
	}
}

// validateConfigSchema validates the asset's config block against the extension's JSON Schema.
func (v *AssetValidator) validateConfigSchema(ctx context.Context, errs *contracts.ValidationErrors, a *contracts.AssetManifest) {
	if a.Extension == "" || a.Config == nil {
		return
	}

	schemaBytes, err := v.resolver.ResolveSchema(ctx, a.Extension, a.Version)
	if err != nil {
		errs.Add(&contracts.ValidationError{
			Code:     ErrAssetExtNotFound,
			Field:    "extension",
			Message:  fmt.Sprintf("could not resolve extension schema for %s@%s: %v", a.Extension, a.Version, err),
			Severity: contracts.SeverityWarning,
		})
		return
	}

	// Parse the schema
	var schemaDoc any
	if err := json.Unmarshal(schemaBytes, &schemaDoc); err != nil {
		errs.Add(&contracts.ValidationError{
			Code:    ErrAssetSchemaValidation,
			Field:   "config",
			Message: fmt.Sprintf("failed to parse extension schema: %v", err),
		})
		return
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		errs.Add(&contracts.ValidationError{
			Code:    ErrAssetSchemaValidation,
			Field:   "config",
			Message: fmt.Sprintf("failed to compile extension schema: %v", err),
		})
		return
	}

	schema, err := c.Compile("schema.json")
	if err != nil {
		errs.Add(&contracts.ValidationError{
			Code:    ErrAssetSchemaValidation,
			Field:   "config",
			Message: fmt.Sprintf("failed to compile extension schema: %v", err),
		})
		return
	}

	// Validate config against schema
	if err := schema.Validate(a.Config); err != nil {
		ve, ok := err.(*jsonschema.ValidationError)
		if ok {
			// Convert jsonschema errors to our ValidationError format
			addSchemaErrors(errs, ve, a.Extension)
		} else {
			errs.Add(&contracts.ValidationError{
				Code:    ErrAssetSchemaValidation,
				Field:   "config",
				Message: fmt.Sprintf("schema validation failed: %v", err),
			})
		}
	}
}

// addSchemaErrors recursively converts JSON Schema validation errors.
func addSchemaErrors(errs *contracts.ValidationErrors, ve *jsonschema.ValidationError, extension string) {
	if len(ve.Causes) > 0 {
		for _, cause := range ve.Causes {
			addSchemaErrors(errs, cause, extension)
		}
		return
	}

	field := "config"
	if len(ve.InstanceLocation) > 0 {
		field = "config/" + joinPath(ve.InstanceLocation)
	}

	message := ve.Error()
	if ve.SchemaURL != "" {
		message = fmt.Sprintf("%s (extension: %s)", simplifySchemaError(ve), extension)
	}

	errs.Add(&contracts.ValidationError{
		Code:    ErrAssetSchemaValidation,
		Field:   field,
		Message: message,
	})
}

// simplifySchemaError extracts a human-readable message from a jsonschema error.
func simplifySchemaError(ve *jsonschema.ValidationError) string {
	msg := ve.Error()
	// Try to make the error more user-friendly
	if len(ve.InstanceLocation) > 0 {
		return fmt.Sprintf("%s: %s", joinPath(ve.InstanceLocation), msg)
	}
	return msg
}

// joinPath joins a string slice into a JSON Pointer-like path.
func joinPath(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "/"
		}
		result += p
	}
	return result
}
