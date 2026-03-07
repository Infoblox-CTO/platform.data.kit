// Package validate provides manifest validation for DataKit data packages.
package validate

import (
	"context"
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// Error codes used by validators.
const (
	ErrMissingRequired = "E001"
	ErrInvalidFormat   = "E002"
	ErrInvalidVersion  = "E020"
	ErrFileNotFound    = "E040"
	ErrParseError      = "E041"
	ErrSchemaError     = "E042"
	ErrDuplicateName   = "E043"
)

// Validator defines the interface for manifest validators.
type Validator interface {
	Validate(ctx context.Context) contracts.ValidationErrors
	Name() string
}

// ValidationResult represents the result of validating multiple manifests.
type ValidationResult struct {
	Valid    bool
	Errors   contracts.ValidationErrors
	Warnings []string
}

// NewValidationResult creates a new empty validation result.
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:    true,
		Errors:   contracts.ValidationErrors{},
		Warnings: []string{},
	}
}

// AddError adds a validation error to the result.
func (r *ValidationResult) AddError(code, field, message string) {
	r.Errors.AddError(code, field, message)
	r.Valid = false
}

// AddWarning adds a warning message to the result.
func (r *ValidationResult) AddWarning(msg string) {
	r.Warnings = append(r.Warnings, msg)
}

// Merge combines another validation result into this one.
func (r *ValidationResult) Merge(other *ValidationResult) {
	if other == nil {
		return
	}

	if !other.Valid {
		r.Valid = false
	}

	for _, err := range other.Errors {
		r.Errors.Add(err)
	}

	r.Warnings = append(r.Warnings, other.Warnings...)
}

// ValidationContext holds context for validation operations.
type ValidationContext struct {
	PackageDir           string
	StrictMode           bool
	SkipSchemaValidation bool
	ValidatePII          bool
}

// DefaultValidationContext returns a validation context with default settings.
func DefaultValidationContext(packageDir string) *ValidationContext {
	return &ValidationContext{
		PackageDir:           packageDir,
		StrictMode:           false,
		SkipSchemaValidation: false,
		ValidatePII:          true, // PII validation enabled by default
	}
}

// validateRequired checks if a required field is present.
func validateRequired(field, value string) *contracts.ValidationError {
	if value == "" {
		return &contracts.ValidationError{
			Code:    ErrMissingRequired,
			Field:   field,
			Message: fmt.Sprintf("%s is required", field),
		}
	}
	return nil
}

// validateEnum checks if a value is in a set of allowed values.
func validateEnum[T comparable](field string, value T, allowed []T) *contracts.ValidationError {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return &contracts.ValidationError{
		Code:    ErrInvalidFormat,
		Field:   field,
		Message: fmt.Sprintf("invalid %s: must be one of %v", field, allowed),
	}
}

// validateSemVer checks if a string is a valid semantic version.
func validateSemVer(field, version string) *contracts.ValidationError {
	if version == "" {
		return nil
	}

	if len(version) < 5 {
		return &contracts.ValidationError{
			Code:    ErrInvalidVersion,
			Field:   field,
			Message: "version must be a valid semantic version (e.g., 1.0.0)",
		}
	}

	return nil
}

// validateIdentifier checks if a string is a valid Kubernetes-style identifier.
func validateIdentifier(field, value string) *contracts.ValidationError {
	if value == "" {
		return nil
	}

	if len(value) > 63 {
		return &contracts.ValidationError{
			Code:    ErrInvalidFormat,
			Field:   field,
			Message: fmt.Sprintf("%s must be 63 characters or less", field),
		}
	}

	for i, c := range value {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '-' && i > 0 && i < len(value)-1)) {
			return &contracts.ValidationError{
				Code:    ErrInvalidFormat,
				Field:   field,
				Message: fmt.Sprintf("%s must be lowercase alphanumeric with optional hyphens (not at start/end)", field),
			}
		}
	}

	return nil
}
