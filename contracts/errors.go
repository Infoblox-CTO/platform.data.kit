package contracts

import (
	"fmt"
	"strings"
)

// ValidationError represents a manifest validation error.
type ValidationError struct {
	// Code is the error code (e.g., "E001")
	Code string `json:"code"`

	// Field is the field path that caused the error
	Field string `json:"field"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// Value is the invalid value (if applicable)
	Value any `json:"value,omitempty"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Field, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors []*ValidationError

// Error implements the error interface.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("%d validation errors:\n  - %s", len(e), strings.Join(msgs, "\n  - "))
}

// HasErrors returns true if there are any validation errors.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Add adds a validation error to the collection.
func (e *ValidationErrors) Add(err *ValidationError) {
	*e = append(*e, err)
}

// AddError creates and adds a validation error.
func (e *ValidationErrors) AddError(code, field, message string) {
	e.Add(&ValidationError{
		Code:    code,
		Field:   field,
		Message: message,
	})
}

// AddErrorWithValue creates and adds a validation error with the invalid value.
func (e *ValidationErrors) AddErrorWithValue(code, field, message string, value any) {
	e.Add(&ValidationError{
		Code:    code,
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// Error code constants per data-model.md validation rules.
const (
	// DataPackage validation errors (E001-E003)
	ErrCodeNameNotDNSSafe     = "E001"
	ErrCodeInvalidPackageType = "E002"
	ErrCodeOutputsRequired    = "E003"

	// ArtifactContract validation errors (E004-E005)
	ErrCodeClassificationRequired = "E004"
	ErrCodeInvalidSchemaType      = "E005"

	// Binding validation errors (E010-E011)
	ErrCodeBindingNotFound     = "E010"
	ErrCodeBindingTypeMismatch = "E011"

	// PackageVersion validation errors (E020-E021)
	ErrCodeInvalidSemVer        = "E020"
	ErrCodeVersionAlreadyExists = "E021"

	// PipelineManifest validation errors (E030-E031)
	ErrCodeInvalidImageRef = "E030"
	ErrCodeInvalidTimeout  = "E031"

	// RuntimeSpec validation errors (E040-E041)
	ErrCodeRuntimeRequired      = "E040"
	ErrCodeRuntimeImageRequired = "E041"
)

// Error message templates.
var errorMessages = map[string]string{
	ErrCodeNameNotDNSSafe:         "name must be DNS-safe (lowercase, alphanumeric, hyphens)",
	ErrCodeInvalidPackageType:     "type must be one of: pipeline, model, dataset",
	ErrCodeOutputsRequired:        "outputs are required for pipeline type packages",
	ErrCodeClassificationRequired: "classification is required for output artifacts",
	ErrCodeInvalidSchemaType:      "schema type must be one of: parquet, avro, json, csv",
	ErrCodeBindingNotFound:        "binding reference not found in environment",
	ErrCodeBindingTypeMismatch:    "binding type does not match artifact type",
	ErrCodeInvalidSemVer:          "version must be a valid SemVer string",
	ErrCodeVersionAlreadyExists:   "version already exists and cannot be overwritten",
	ErrCodeInvalidImageRef:        "image must be a valid container image reference",
	ErrCodeInvalidTimeout:         "timeout must be a positive duration",
	ErrCodeRuntimeRequired:        "spec.runtime is required for pipeline type packages",
	ErrCodeRuntimeImageRequired:   "spec.runtime.image is required",
}

// NewValidationError creates a new validation error with the standard message.
func NewValidationError(code, field string) *ValidationError {
	return &ValidationError{
		Code:    code,
		Field:   field,
		Message: errorMessages[code],
	}
}

// NewValidationErrorWithValue creates a new validation error with the standard message and value.
func NewValidationErrorWithValue(code, field string, value any) *ValidationError {
	return &ValidationError{
		Code:    code,
		Field:   field,
		Message: errorMessages[code],
		Value:   value,
	}
}

// NewValidationErrorCustom creates a new validation error with a custom message.
func NewValidationErrorCustom(code, field, message string) *ValidationError {
	return &ValidationError{
		Code:    code,
		Field:   field,
		Message: message,
	}
}
