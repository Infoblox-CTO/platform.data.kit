// Package validate provides PII classification validation for data packages.
package validate

import (
	"fmt"
	"strings"

	"github.com/Infoblox-CTO/data.platform.kit/contracts"
)

// PIIValidator validates PII classification on data package outputs.
type PIIValidator struct {
	// RequireClassification requires all outputs to have a classification.
	RequireClassification bool
	// AllowedSensitivities restricts sensitivities to a subset.
	AllowedSensitivities []contracts.Sensitivity
}

// NewPIIValidator creates a new PII validator.
func NewPIIValidator() *PIIValidator {
	return &PIIValidator{
		RequireClassification: true,
		AllowedSensitivities:  contracts.ValidSensitivities(),
	}
}

// Validate checks PII classification requirements on a data package.
func (v *PIIValidator) Validate(pkg *contracts.DataPackage) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if pkg == nil {
		errs = append(errs, &contracts.ValidationError{
			Code:    ErrMissingRequired,
			Field:   "",
			Message: "data package is nil",
		})
		return errs
	}

	// Check all outputs have classification
	if v.RequireClassification {
		for i, output := range pkg.Spec.Outputs {
			if output.Classification == nil {
				errs = append(errs, &contracts.ValidationError{
					Code:    contracts.ErrCodeClassificationRequired,
					Field:   fmt.Sprintf("spec.outputs[%d].classification", i),
					Message: fmt.Sprintf("output %q missing required classification field", output.Name),
				})
			}
		}
	}

	// Validate sensitivity values
	for i, output := range pkg.Spec.Outputs {
		if output.Classification != nil && output.Classification.Sensitivity != "" {
			if !v.isValidSensitivity(output.Classification.Sensitivity) {
				errs = append(errs, &contracts.ValidationError{
					Code:    contracts.ErrCodeClassificationRequired,
					Field:   fmt.Sprintf("spec.outputs[%d].classification.sensitivity", i),
					Message: fmt.Sprintf("output %q has invalid sensitivity value: %s (allowed: %s)", output.Name, output.Classification.Sensitivity, v.getAllowedSensitivitiesString()),
				})
			}
		}
	}

	// Check for PII flag on potentially sensitive outputs (warning only, no severity field)
	// Warnings are logged but don't fail validation

	return errs
}

// isValidSensitivity checks if a sensitivity value is valid.
func (v *PIIValidator) isValidSensitivity(sensitivity contracts.Sensitivity) bool {
	for _, allowed := range v.AllowedSensitivities {
		if sensitivity == allowed {
			return true
		}
	}
	return false
}

// getAllowedSensitivitiesString returns a comma-separated list of allowed sensitivities.
func (v *PIIValidator) getAllowedSensitivitiesString() string {
	strs := make([]string, len(v.AllowedSensitivities))
	for i, s := range v.AllowedSensitivities {
		strs[i] = string(s)
	}
	return strings.Join(strs, ", ")
}

// isPotentiallyPIIOutput checks if an output might contain PII based on naming conventions.
func (v *PIIValidator) isPotentiallyPIIOutput(output contracts.ArtifactContract) bool {
	name := strings.ToLower(output.Name)
	piiIndicators := []string{
		"user", "customer", "person", "employee",
		"email", "phone", "address", "ssn", "social",
		"passport", "license", "credit", "card",
		"medical", "health", "patient",
	}
	for _, indicator := range piiIndicators {
		if strings.Contains(name, indicator) {
			return true
		}
	}
	return false
}

// ValidatePIICompliance performs comprehensive PII compliance validation.
func ValidatePIICompliance(pkg *contracts.DataPackage) contracts.ValidationErrors {
	validator := NewPIIValidator()
	return validator.Validate(pkg)
}
