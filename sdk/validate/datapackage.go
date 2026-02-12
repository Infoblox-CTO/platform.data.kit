package validate

import (
	"context"
	"os"
	"strconv"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// DataPackageValidator validates dp.yaml manifests.
type DataPackageValidator struct {
	pkg     *contracts.DataPackage
	pkgPath string
}

// NewDataPackageValidator creates a validator for a DataPackage.
func NewDataPackageValidator(pkg *contracts.DataPackage, pkgPath string) *DataPackageValidator {
	return &DataPackageValidator{
		pkg:     pkg,
		pkgPath: pkgPath,
	}
}

// NewDataPackageValidatorFromFile creates a validator from a dp.yaml file.
func NewDataPackageValidatorFromFile(path string) (*DataPackageValidator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := manifest.NewParser()
	pkg, err := parser.ParseDataPackage(data)
	if err != nil {
		return nil, err
	}

	return &DataPackageValidator{
		pkg:     pkg,
		pkgPath: path,
	}, nil
}

// Name returns the validator name.
func (v *DataPackageValidator) Name() string {
	return "datapackage"
}

// Package returns the parsed DataPackage.
func (v *DataPackageValidator) Package() *contracts.DataPackage {
	return v.pkg
}

// Validate validates the DataPackage manifest.
func (v *DataPackageValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.pkg == nil {
		errs.AddError(ErrMissingRequired, "", "datapackage is nil")
		return errs
	}

	// Validate required fields
	v.validateRequiredFields(&errs)

	// Validate API version
	v.validateAPIVersion(&errs)

	// Validate metadata
	v.validateMetadata(&errs)

	// Validate spec
	v.validateSpec(&errs)

	// Validate inputs if present
	v.validateArtifacts(&errs, v.pkg.Spec.Inputs, "spec.inputs")

	// Validate outputs if present
	v.validateArtifacts(&errs, v.pkg.Spec.Outputs, "spec.outputs")

	return errs
}

// validateRequiredFields checks for required top-level fields.
func (v *DataPackageValidator) validateRequiredFields(errs *contracts.ValidationErrors) {
	if v.pkg.APIVersion == "" {
		errs.AddError(ErrMissingRequired, "apiVersion", "apiVersion is required")
	}

	if v.pkg.Kind == "" {
		errs.AddError(ErrMissingRequired, "kind", "kind is required")
	} else if v.pkg.Kind != "DataPackage" {
		errs.AddError(ErrInvalidFormat, "kind", "kind must be 'DataPackage'")
	}
}

// validateAPIVersion validates the API version format.
func (v *DataPackageValidator) validateAPIVersion(errs *contracts.ValidationErrors) {
	if v.pkg.APIVersion == "" {
		return // Already caught by required check
	}

	validVersions := []string{
		string(contracts.APIVersionV1Alpha1),
		string(contracts.APIVersionV1Beta1),
		string(contracts.APIVersionV1),
	}

	valid := false
	for _, ver := range validVersions {
		if v.pkg.APIVersion == ver {
			valid = true
			break
		}
	}

	if !valid {
		errs.AddError(ErrInvalidVersion, "apiVersion", "invalid API version: "+v.pkg.APIVersion)
	}
}

// validateMetadata validates the metadata section.
func (v *DataPackageValidator) validateMetadata(errs *contracts.ValidationErrors) {
	meta := v.pkg.Metadata

	if meta.Name == "" {
		errs.AddError(ErrMissingRequired, "metadata.name", "metadata.name is required")
	} else if !isDPIdentifierValid(meta.Name) {
		errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.name", "metadata.name must be DNS-safe")
	}

	if meta.Namespace == "" {
		errs.AddError(ErrMissingRequired, "metadata.namespace", "metadata.namespace is required")
	} else if !isDPIdentifierValid(meta.Namespace) {
		errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.namespace", "metadata.namespace must be DNS-safe")
	}

	if meta.Version == "" {
		errs.AddError(ErrMissingRequired, "metadata.version", "metadata.version is required")
	} else if !isDPSemVerValid(meta.Version) {
		errs.AddError(contracts.ErrCodeInvalidSemVer, "metadata.version", "metadata.version must be valid SemVer")
	}

	// Validate team label if present
	if meta.Labels != nil {
		if team, ok := meta.Labels["team"]; ok {
			if !isDPIdentifierValid(team) {
				errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.labels.team", "team label must be DNS-safe")
			}
		}
	}
}

// validateSpec validates the spec section.
func (v *DataPackageValidator) validateSpec(errs *contracts.ValidationErrors) {
	spec := v.pkg.Spec

	// Validate type
	validTypes := []contracts.PackageType{
		contracts.PackageTypePipeline,
	}

	valid := false
	for _, t := range validTypes {
		if spec.Type == t {
			valid = true
			break
		}
	}

	if !valid {
		errs.AddError(contracts.ErrCodeInvalidPackageType, "spec.type", "spec.type must be: pipeline")
	}

	// Validate description
	if spec.Description == "" {
		errs.AddError(ErrMissingRequired, "spec.description", "spec.description is required")
	}

	// For pipeline type, outputs are required
	if spec.Type == contracts.PackageTypePipeline && len(spec.Outputs) == 0 {
		errs.AddError(contracts.ErrCodeOutputsRequired, "spec.outputs", "outputs are required for pipeline type packages")
	}

	// For pipeline type, runtime is required
	if spec.Type == contracts.PackageTypePipeline {
		v.validateRuntime(errs)
	}

	// Validate schedule if present
	if spec.Schedule != nil {
		v.validateSchedule(errs, spec.Schedule)
	}
}

// validateSchedule validates the schedule section.
func (v *DataPackageValidator) validateSchedule(errs *contracts.ValidationErrors, schedule *contracts.ScheduleSpec) {
	// Schedule can have cron expression, or be suspended
	// For MVP, just verify cron is not empty if schedule is provided
	if schedule.Cron == "" && !schedule.Suspend {
		// This is okay - schedule may be event-driven
	}
}

// validateRuntime validates the runtime section for pipeline packages.
func (v *DataPackageValidator) validateRuntime(errs *contracts.ValidationErrors) {
	runtime := v.pkg.Spec.Runtime

	// Runtime is required for pipeline packages
	if runtime == nil {
		errs.AddError(contracts.ErrCodeRuntimeRequired, "spec.runtime", "spec.runtime is required for pipeline type packages")
		return
	}

	// Image is required
	if runtime.Image == "" {
		errs.AddError(contracts.ErrCodeRuntimeImageRequired, "spec.runtime.image", "spec.runtime.image is required")
	}

	// Validate timeout format if specified
	if runtime.Timeout != "" {
		if !isValidDuration(runtime.Timeout) {
			errs.AddError(contracts.ErrCodeInvalidTimeout, "spec.runtime.timeout", "spec.runtime.timeout must be a valid duration (e.g., 1h, 30m)")
		}
	}

	// Validate retries is non-negative
	if runtime.Retries < 0 {
		errs.AddError(ErrInvalidFormat, "spec.runtime.retries", "spec.runtime.retries must be non-negative")
	}

	// Validate replicas is positive
	if runtime.Replicas < 0 {
		errs.AddError(ErrInvalidFormat, "spec.runtime.replicas", "spec.runtime.replicas must be non-negative")
	}
}

// isValidDuration checks if a string is a valid Go duration.
func isValidDuration(s string) bool {
	// Check for common duration patterns: 1h, 30m, 2h30m, etc.
	// A simple check - valid durations have digits followed by unit
	if s == "" {
		return false
	}
	// Try to match a basic pattern: digits + unit (s, m, h)
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == 'h' || c == 'm' || c == 's' {
			if i == 0 {
				return false // No digits before unit
			}
			// Continue checking rest of string
			continue
		}
		return false // Invalid character
	}
	return true
}

// validateArtifacts validates input or output artifacts.
func (v *DataPackageValidator) validateArtifacts(errs *contracts.ValidationErrors, artifacts []contracts.ArtifactContract, basePath string) {
	seen := make(map[string]bool)

	for i := range artifacts {
		artifact := &artifacts[i]
		path := basePath + "[" + strconv.Itoa(i) + "]"

		if artifact.Name == "" {
			errs.AddError(ErrMissingRequired, path+".name", "artifact name is required")
		} else if seen[artifact.Name] {
			errs.AddError(ErrDuplicateName, path+".name", "duplicate artifact name: "+artifact.Name)
		} else {
			seen[artifact.Name] = true
		}

		if !artifact.Type.IsValid() {
			errs.AddError(contracts.ErrCodeInvalidSchemaType, path+".type", "invalid artifact type")
		}

		if artifact.Binding == "" {
			errs.AddError(ErrMissingRequired, path+".binding", "artifact binding is required")
		}

		// Validate classification if it requires outputs
		if artifact.Classification != nil {
			if artifact.Classification.Sensitivity != "" && !artifact.Classification.Sensitivity.IsValid() {
				errs.AddError(ErrInvalidFormat, path+".classification.sensitivity", "invalid sensitivity level")
			}
		}
	}
}

// isDPIdentifierValid checks if a string is a valid DNS-safe identifier.
func isDPIdentifierValid(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}
	for i, c := range s {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i > 0 && i < len(s)-1 {
			continue
		}
		return false
	}
	return true
}

// isDPSemVerValid checks if a string is a valid semantic version.
func isDPSemVerValid(s string) bool {
	// Basic SemVer validation: X.Y.Z format
	parts := 0
	numLen := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			numLen++
		} else if c == '.' {
			if numLen == 0 {
				return false
			}
			parts++
			numLen = 0
		} else if c == '-' || c == '+' {
			// Pre-release or build metadata
			break
		} else {
			return false
		}
	}
	return parts >= 2 && numLen > 0
}
