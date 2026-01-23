package validate

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// AggregateValidator validates all manifests in a package directory.
type AggregateValidator struct {
	packageDir string
	vctx       *ValidationContext
}

// NewAggregateValidator creates an aggregate validator for a package directory.
func NewAggregateValidator(packageDir string) *AggregateValidator {
	return &AggregateValidator{
		packageDir: packageDir,
		vctx:       DefaultValidationContext(packageDir),
	}
}

// WithContext sets a custom validation context.
func (v *AggregateValidator) WithContext(ctx *ValidationContext) *AggregateValidator {
	v.vctx = ctx
	return v
}

// Name returns the validator name.
func (v *AggregateValidator) Name() string {
	return "aggregate"
}

// Validate validates all manifests in the package directory.
func (v *AggregateValidator) Validate(ctx context.Context) *ValidationResult {
	result := NewValidationResult()

	if _, err := os.Stat(v.packageDir); os.IsNotExist(err) {
		result.AddError(ErrFileNotFound, "", "package directory not found: "+v.packageDir)
		return result
	}

	dpPath := filepath.Join(v.packageDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		result.AddError(ErrFileNotFound, "dp.yaml", "dp.yaml not found - this is required for a valid package")
	} else {
		dpResult := v.validateDataPackage(ctx, dpPath)
		result.Merge(dpResult)
	}

	bindingsPath := filepath.Join(v.packageDir, "bindings.yaml")
	if _, err := os.Stat(bindingsPath); err == nil {
		bindingsResult := v.validateBindings(ctx, bindingsPath)
		result.Merge(bindingsResult)
	}

	schemasDir := filepath.Join(v.packageDir, "schemas")
	if _, err := os.Stat(schemasDir); err == nil {
		schemasResult := v.validateSchemas(ctx, schemasDir)
		result.Merge(schemasResult)
	}

	return result
}

// validateDataPackage validates the dp.yaml file.
func (v *AggregateValidator) validateDataPackage(ctx context.Context, path string) *ValidationResult {
	result := NewValidationResult()

	validator, err := NewDataPackageValidatorFromFile(path)
	if err != nil {
		result.AddError(ErrParseError, "dp.yaml", "failed to parse dp.yaml: "+err.Error())
		return result
	}

	errs := validator.Validate(ctx)
	if errs.HasErrors() {
		result.Valid = false
		for _, e := range errs {
			result.Errors.Add(e)
		}
	}

	// Run PII validation if configured
	if v.vctx != nil && v.vctx.ValidatePII {
		piiResult := v.validatePII(ctx, validator.Package())
		result.Merge(piiResult)
	}

	return result
}

// validateBindings validates the bindings.yaml file.
func (v *AggregateValidator) validateBindings(ctx context.Context, path string) *ValidationResult {
	result := NewValidationResult()

	validator, err := NewBindingsValidatorFromFile(path)
	if err != nil {
		result.AddError(ErrParseError, "bindings.yaml", "failed to parse bindings.yaml: "+err.Error())
		return result
	}

	errs := validator.Validate(ctx)
	if errs.HasErrors() {
		result.Valid = false
		for _, e := range errs {
			result.Errors.Add(e)
		}
	}

	return result
}

// validateSchemas validates schema files in the schemas directory.
func (v *AggregateValidator) validateSchemas(ctx context.Context, schemasDir string) *ValidationResult {
	result := NewValidationResult()

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		result.AddWarning("failed to read schemas directory: " + err.Error())
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		schemaPath := filepath.Join(schemasDir, entry.Name())
		ext := filepath.Ext(entry.Name())

		switch ext {
		case ".avsc":
			v.validateAvroSchema(result, schemaPath)
		case ".json":
			v.validateJSONSchema(result, schemaPath)
		case ".proto":
			v.validateProtoSchema(result, schemaPath)
		}
	}

	return result
}

// validateAvroSchema validates an Avro schema file.
func (v *AggregateValidator) validateAvroSchema(result *ValidationResult, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		result.AddWarning("failed to read Avro schema: " + path)
		return
	}

	if !bytesContain(data, []byte("\"type\"")) {
		result.AddError(ErrSchemaError, "schemas/"+filepath.Base(path), "Avro schema missing 'type' field")
	}
}

// validateJSONSchema validates a JSON Schema file.
func (v *AggregateValidator) validateJSONSchema(result *ValidationResult, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		result.AddWarning("failed to read JSON schema: " + path)
		return
	}

	if !bytesContain(data, []byte("{")) {
		result.AddError(ErrSchemaError, "schemas/"+filepath.Base(path), "JSON schema must be valid JSON")
	}
}

// validateProtoSchema validates a Protobuf schema file.
func (v *AggregateValidator) validateProtoSchema(result *ValidationResult, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		result.AddWarning("failed to read Protobuf schema: " + path)
		return
	}

	if !bytesContain(data, []byte("syntax")) && !bytesContain(data, []byte("message")) {
		result.AddError(ErrSchemaError, "schemas/"+filepath.Base(path), "Protobuf schema should contain syntax or message declaration")
	}
}

// bytesContain checks if data contains the pattern.
func bytesContain(data, pattern []byte) bool {
	for i := 0; i <= len(data)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// validatePII runs PII classification validation on a data package.
func (v *AggregateValidator) validatePII(ctx context.Context, pkg *contracts.DataPackage) *ValidationResult {
	result := NewValidationResult()

	if pkg == nil {
		return result
	}

	piiValidator := NewPIIValidator()
	errs := piiValidator.Validate(pkg)
	if errs.HasErrors() {
		result.Valid = false
		for _, e := range errs {
			result.Errors.Add(e)
		}
	}

	return result
}

// ValidatePackage is a convenience function to validate a package directory.
func ValidatePackage(ctx context.Context, packageDir string) *ValidationResult {
	validator := NewAggregateValidator(packageDir)
	return validator.Validate(ctx)
}

// ValidatePackageStrict validates with strict mode enabled.
func ValidatePackageStrict(ctx context.Context, packageDir string) *ValidationResult {
	validator := NewAggregateValidator(packageDir)
	validator.vctx.StrictMode = true
	return validator.Validate(ctx)
}
