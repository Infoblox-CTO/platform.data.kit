package validate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
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

	var dpPkg *contracts.DataPackage

	dpPath := filepath.Join(v.packageDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		result.AddError(ErrFileNotFound, "dp.yaml", "dp.yaml not found - this is required for a valid package")
	} else {
		dpResult, pkg := v.validateDataPackage(ctx, dpPath)
		result.Merge(dpResult)
		dpPkg = pkg
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

	// Validate assets if assets/ directory exists
	assetsResult := v.validateAssets(ctx)
	result.Merge(assetsResult)

	// Check for orphan assets (assets on disk not referenced in dp.yaml)
	v.checkOrphanAssets(result, dpPkg)

	// Validate pipeline workflow if pipeline.yaml exists and is PipelineWorkflow kind.
	pipelinePath := filepath.Join(v.packageDir, pipeline.PipelineFileName)
	if _, err := os.Stat(pipelinePath); err == nil {
		pw, loadErr := pipeline.LoadPipeline(pipelinePath)
		if loadErr == nil && pw.Kind == "PipelineWorkflow" {
			pwResult := v.validatePipelineWorkflow(ctx, pipelinePath)
			result.Merge(pwResult)
		}
	}

	// Validate schedule if schedule.yaml exists
	schedulePath := filepath.Join(v.packageDir, ScheduleFileName)
	if _, err := os.Stat(schedulePath); err == nil {
		schedResult := v.validateSchedule(ctx, schedulePath)
		result.Merge(schedResult)
	}

	return result
}

// validateDataPackage validates the dp.yaml file.
func (v *AggregateValidator) validateDataPackage(ctx context.Context, path string) (*ValidationResult, *contracts.DataPackage) {
	result := NewValidationResult()

	validator, err := NewDataPackageValidatorFromFile(path)
	if err != nil {
		result.AddError(ErrParseError, "dp.yaml", "failed to parse dp.yaml: "+err.Error())
		return result, nil
	}

	errs := validator.Validate(ctx)
	if errs.HasErrors() {
		result.Valid = false
		for _, e := range errs {
			result.Errors.Add(e)
		}
	}

	pkg := validator.Package()

	// Run CloudQuery-specific validation if the package type is cloudquery
	if pkg != nil && pkg.Spec.Type == contracts.PackageTypeCloudQuery {
		cqValidator := NewCloudQueryValidator(pkg)
		cqErrs := cqValidator.Validate(ctx)
		for _, e := range cqErrs {
			if e.Severity == contracts.SeverityWarning {
				result.AddWarning(e.Error())
			} else {
				result.AddError(e.Code, e.Field, e.Message)
			}
		}
	}

	// Run PII validation if configured
	if v.vctx != nil && v.vctx.ValidatePII {
		piiResult := v.validatePII(ctx, validator.Package())
		result.Merge(piiResult)
	}

	return result, pkg
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

	// Cross-validate asset-binding references
	assets, loadErr := asset.LoadAllAssets(v.packageDir)
	if loadErr == nil && len(assets) > 0 {
		assetErrs := validator.ValidateAssetBindings(assets)
		if assetErrs.HasErrors() {
			result.Valid = false
		}
		for _, e := range assetErrs {
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

// checkOrphanAssets warns about assets on disk that are not referenced in dp.yaml.
func (v *AggregateValidator) checkOrphanAssets(result *ValidationResult, dpPkg *contracts.DataPackage) {
	assets, err := asset.LoadAllAssets(v.packageDir)
	if err != nil || len(assets) == 0 {
		return
	}

	// Build set of referenced asset names from dp.yaml
	referenced := make(map[string]bool)
	if dpPkg != nil {
		for _, name := range dpPkg.Spec.Assets {
			referenced[name] = true
		}
	}

	// If dp.yaml has no assets section at all, don't warn — the user may not have adopted assets yet
	if dpPkg == nil || len(dpPkg.Spec.Assets) == 0 {
		return
	}

	for _, a := range assets {
		if !referenced[a.Name] {
			result.AddWarning(fmt.Sprintf("asset %q exists in assets/ but is not referenced in dp.yaml spec.assets (consider adding it or removing the asset)", a.Name))
		}
	}
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

// validateAssets validates all asset.yaml files in the assets/ directory.
func (v *AggregateValidator) validateAssets(ctx context.Context) *ValidationResult {
	result := NewValidationResult()

	assets, err := asset.LoadAllAssets(v.packageDir)
	if err != nil {
		result.AddWarning("failed to load assets: " + err.Error())
		return result
	}

	if len(assets) == 0 {
		return result
	}

	validator := NewAssetValidator(asset.DefaultResolver())
	for _, a := range assets {
		errs := validator.ValidateAsset(ctx, a)
		if errs.HasErrors() {
			result.Valid = false
		}
		for _, e := range errs {
			result.Errors.Add(e)
		}
	}

	return result
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

// validatePipelineWorkflow validates a pipeline.yaml file.
func (v *AggregateValidator) validatePipelineWorkflow(ctx context.Context, path string) *ValidationResult {
	result := NewValidationResult()

	pwValidator, err := NewPipelineWorkflowValidatorFromFile(path)
	if err != nil {
		result.AddError(ErrParseError, "pipeline.yaml", "failed to parse pipeline.yaml: "+err.Error())
		return result
	}

	errs := pwValidator.Validate(ctx)
	if errs.HasErrors() {
		result.Valid = false
		for _, e := range errs {
			result.Errors.Add(e)
		}
	}

	// Cross-validate asset references if assets exist
	assets, loadErr := asset.LoadAllAssets(v.packageDir)
	if loadErr == nil && len(assets) > 0 {
		assetErrs := ValidateAssetReferences(pwValidator.Workflow(), assets)
		if assetErrs.HasErrors() {
			result.Valid = false
			for _, e := range assetErrs {
				result.Errors.Add(e)
			}
		}
	}

	return result
}

// validateSchedule validates a schedule.yaml file.
func (v *AggregateValidator) validateSchedule(ctx context.Context, path string) *ValidationResult {
	result := NewValidationResult()

	schedValidator, err := NewScheduleValidatorFromFile(path)
	if err != nil {
		result.AddError(ErrParseError, "schedule.yaml", "failed to parse schedule.yaml: "+err.Error())
		return result
	}

	errs := schedValidator.Validate(ctx)
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
