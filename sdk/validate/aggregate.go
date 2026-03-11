package validate

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/schema"
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

	dkPath := filepath.Join(v.packageDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		result.AddError(ErrFileNotFound, "dk.yaml", "dk.yaml not found - this is required for a valid package")
	} else {
		dpResult := v.validateManifest(ctx, dkPath)
		result.Merge(dpResult)
	}

	schemasDir := filepath.Join(v.packageDir, "schemas")
	if _, err := os.Stat(schemasDir); err == nil {
		schemasResult := v.validateSchemas(ctx, schemasDir)
		result.Merge(schemasResult)
	}

	// Validate datasets if datasets/ directory exists
	datasetsResult := v.validateDataSets(ctx)
	result.Merge(datasetsResult)

	// Validate schema lock if dk.lock exists and not skipped
	if !v.vctx.SkipSchemaLock {
		lockResult := v.validateSchemaLock(ctx)
		result.Merge(lockResult)
	}

	return result
}

// validateManifest validates the dk.yaml file using the kind-aware ManifestValidator.
func (v *AggregateValidator) validateManifest(ctx context.Context, path string) *ValidationResult {
	result := NewValidationResult()

	validator, err := NewManifestValidatorFromFile(path)
	if err != nil {
		result.AddError(ErrParseError, "dk.yaml", "failed to parse dk.yaml: "+err.Error())
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

// validateDataSets validates all dataset.yaml files in the datasets/ directory.
func (v *AggregateValidator) validateDataSets(ctx context.Context) *ValidationResult {
	result := NewValidationResult()

	datasets, err := dataset.LoadAllDataSets(v.packageDir)
	if err != nil {
		result.AddWarning("failed to load datasets: " + err.Error())
		return result
	}

	if len(datasets) == 0 {
		return result
	}

	validator := NewDataSetValidator()
	for _, ds := range datasets {
		errs := validator.ValidateDataSet(ctx, ds)
		if errs.HasErrors() {
			result.Valid = false
		}
		for _, e := range errs {
			result.Errors.Add(e)
		}
	}

	return result
}

// validateSchemaLock checks that every schemaRef in datasets has a corresponding
// entry in dk.lock.
func (v *AggregateValidator) validateSchemaLock(_ context.Context) *ValidationResult {
	result := NewValidationResult()

	// Load lock file — if it doesn't exist, nothing to validate.
	lock, err := schema.ReadLockFile(v.packageDir)
	if err != nil {
		result.AddWarning("failed to read dk.lock: " + err.Error())
		return result
	}
	if lock == nil {
		// No lock file — check if any datasets use schemaRef.
		datasets, err := dataset.LoadAllDataSets(v.packageDir)
		if err != nil {
			return result
		}
		hasSchemaRef := false
		for _, ds := range datasets {
			if ds.Spec.SchemaRef != "" {
				hasSchemaRef = true
				break
			}
		}
		if hasSchemaRef {
			result.AddWarning("datasets use schemaRef but dk.lock not found — run 'dk lock' to generate")
		}
		return result
	}

	// Verify each dataset schemaRef has a lock entry.
	datasets, err := dataset.LoadAllDataSets(v.packageDir)
	if err != nil {
		return result
	}

	for _, ds := range datasets {
		if ds.Spec.SchemaRef == "" {
			continue
		}
		module, _ := schema.ParseSchemaRef(ds.Spec.SchemaRef)
		if schema.FindLockedSchema(lock, module) == nil {
			result.Errors.AddError(
				contracts.ErrCodeSchemaLockMissing,
				"datasets/"+ds.Metadata.Name+"/dataset.yaml",
				"schema lock entry missing for schemaRef \""+ds.Spec.SchemaRef+"\" — run 'dk lock'",
			)
			result.Valid = false
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
