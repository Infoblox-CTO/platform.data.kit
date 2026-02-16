// Package validate provides PII classification validation for manifests.
package validate

import (
	"fmt"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// PIIValidator validates PII classification on Model outputs.
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

// Validate checks PII classification requirements on a Model manifest.
func (v *PIIValidator) Validate(model *contracts.Model) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if model == nil {
		errs = append(errs, &contracts.ValidationError{
			Code:    ErrMissingRequired,
			Field:   "",
			Message: "model is nil",
		})
		return errs
	}

	// Check all outputs have classification
	if v.RequireClassification {
		for i, output := range model.Spec.Outputs {
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
	for i, output := range model.Spec.Outputs {
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

// ValidatePIICompliance performs comprehensive PII compliance validation on a Model.
func ValidatePIICompliance(model *contracts.Model) contracts.ValidationErrors {
	validator := NewPIIValidator()
	return validator.Validate(model)
}
