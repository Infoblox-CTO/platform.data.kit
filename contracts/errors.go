package contracts

import (
	"fmt"
	"strings"
)

// ValidationSeverity represents the severity of a validation issue.
type ValidationSeverity string

const (
	// SeverityError indicates a validation error that must be fixed.
	SeverityError ValidationSeverity = "error"
	// SeverityWarning indicates a validation warning that should be reviewed.
	SeverityWarning ValidationSeverity = "warning"
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

	// Severity indicates if this is an error or warning
	Severity ValidationSeverity `json:"severity,omitempty"`
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

// HasErrors returns true if there are any validation errors (not warnings).
func (e ValidationErrors) HasErrors() bool {
	return len(e.Errors()) > 0
}

// HasWarnings returns true if there are any validation warnings.
func (e ValidationErrors) HasWarnings() bool {
	return len(e.Warnings()) > 0
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
		Code:     code,
		Field:    field,
		Message:  message,
		Value:    value,
		Severity: SeverityError,
	})
}

// AddWarning creates and adds a validation warning.
func (e *ValidationErrors) AddWarning(code, field, message string) {
	e.Add(&ValidationError{
		Code:     code,
		Field:    field,
		Message:  message,
		Severity: SeverityWarning,
	})
}

// Errors returns only the validation errors (not warnings).
func (e ValidationErrors) Errors() ValidationErrors {
	var errs ValidationErrors
	for _, err := range e {
		if err.Severity != SeverityWarning {
			errs = append(errs, err)
		}
	}
	return errs
}

// Warnings returns only the validation warnings.
func (e ValidationErrors) Warnings() ValidationErrors {
	var warnings ValidationErrors
	for _, err := range e {
		if err.Severity == SeverityWarning {
			warnings = append(warnings, err)
		}
	}
	return warnings
}

// Error code constants per data-model.md validation rules.
const (
	// Manifest validation errors (E001-E003)
	ErrCodeNameNotDNSSafe     = "E001"
	ErrCodeInvalidPackageType = "E002"
	ErrCodeOutputsRequired    = "E003"

	// DataSet/classification validation errors (E004-E005)
	ErrCodeClassificationRequired = "E004"
	ErrCodeInvalidSchemaType      = "E005"

	// PackageVersion validation errors (E020-E021)
	ErrCodeInvalidSemVer        = "E020"
	ErrCodeVersionAlreadyExists = "E021"

	// Image and timeout validation errors (E030-E031)
	ErrCodeInvalidImageRef = "E030"
	ErrCodeInvalidTimeout  = "E031"

	// Runtime validation errors (E040-E041)
	ErrCodeRuntimeRequired      = "E040"
	ErrCodeRuntimeImageRequired = "E041"

	// Model validation warnings (W200-W209)
	WarnCodeTriggerBatchMode = "W209"

	// --- New kind validation errors (E200+) ---

	// Connector validation errors (E200-E209)
	ErrCodeConnectorTypeRequired         = "E200" // spec.type is required
	ErrCodeConnectorCapabilitiesRequired = "E201" // spec.capabilities must be non-empty

	// Store validation errors (E210-E219)
	ErrCodeStoreConnectorRequired  = "E210" // spec.connector is required
	ErrCodeStoreConnectionRequired = "E211" // spec.connection must be non-empty
	ErrCodeStoreSecretsInvalid     = "E212" // spec.secrets contains invalid interpolation syntax

	// DataSet validation errors (E220-E229)
	ErrCodeDataSetStoreRequired    = "E220" // spec.store is required
	ErrCodeDataSetLocationRequired = "E221" // at least one of table/prefix/topic is required
	ErrCodeDataSetSchemaInvalid    = "E222" // spec.schema contains invalid field definitions

	// Transform validation errors (E230-E239)
	ErrCodeTransformInputsRequired  = "E230" // spec.inputs must be non-empty
	ErrCodeTransformOutputsRequired = "E231" // spec.outputs must be non-empty
	ErrCodeTransformImageRequired   = "E232" // spec.image required for generic-* runtimes

	// DataSetGroup validation errors (E240-E249)
	ErrCodeDataSetGroupStoreRequired    = "E240" // spec.store is required
	ErrCodeDataSetGroupDataSetsRequired = "E241" // spec.datasets must be non-empty

	// Schema lock validation errors (E310-E319)
	ErrCodeSchemaRefMutualExclusive = "E310" // schemaRef and inline schema are mutually exclusive
	ErrCodeSchemaRefInvalidFormat   = "E311" // schemaRef format invalid (expected "module@constraint")
	ErrCodeSchemaLockMissing        = "E312" // schema lock entry missing for schemaRef
	ErrCodeSchemaLockChecksumFail   = "E313" // schema lock checksum mismatch
	ErrCodeSchemaBreakingChange     = "E314" // breaking schema change detected
)

// Error message templates.
var errorMessages = map[string]string{
	ErrCodeNameNotDNSSafe:         "name must be DNS-safe (lowercase, alphanumeric, hyphens)",
	ErrCodeInvalidPackageType:     "kind must be one of: Connector, Store, DataSet, DataSetGroup, Transform",
	ErrCodeOutputsRequired:        "outputs are required for Transform kind packages",
	ErrCodeClassificationRequired: "classification is required for output artifacts",
	ErrCodeInvalidSchemaType:      "schema type must be one of: parquet, avro, json, csv",
	ErrCodeInvalidSemVer:          "version must be a valid SemVer string",
	ErrCodeVersionAlreadyExists:   "version already exists and cannot be overwritten",
	ErrCodeInvalidImageRef:        "image must be a valid container image reference",
	ErrCodeInvalidTimeout:         "timeout must be a positive duration",
	ErrCodeRuntimeRequired:        "spec.runtime is required",
	ErrCodeRuntimeImageRequired:   "spec.image is required for generic-* runtimes",
	WarnCodeTriggerBatchMode:      "trigger is recommended for batch mode transforms",

	// --- New kind error messages ---
	ErrCodeConnectorTypeRequired:         "spec.type is required for Connector",
	ErrCodeConnectorCapabilitiesRequired: "spec.capabilities must list at least one capability (source, destination)",
	ErrCodeStoreConnectorRequired:        "spec.connector is required for Store",
	ErrCodeStoreConnectionRequired:       "spec.connection must contain at least one connection parameter",
	ErrCodeStoreSecretsInvalid:           "spec.secrets values must use ${VAR} interpolation syntax",
	ErrCodeDataSetStoreRequired:          "spec.store is required for DataSet",
	ErrCodeDataSetLocationRequired:       "at least one of spec.table, spec.prefix, or spec.topic is required",
	ErrCodeDataSetSchemaInvalid:          "spec.schema contains invalid field definitions (name and type are required)",
	ErrCodeTransformInputsRequired:       "spec.inputs must contain at least one dataset reference",
	ErrCodeTransformOutputsRequired:      "spec.outputs must contain at least one dataset reference",
	ErrCodeTransformImageRequired:        "spec.image is required for generic-go, generic-python, and dbt runtimes",
	ErrCodeDataSetGroupStoreRequired:     "spec.store is required for DataSetGroup",
	ErrCodeDataSetGroupDataSetsRequired:  "spec.datasets must contain at least one dataset name",

	// --- Schema lock error messages ---
	ErrCodeSchemaRefMutualExclusive: "spec.schemaRef and spec.schema are mutually exclusive — use one or the other",
	ErrCodeSchemaRefInvalidFormat:   "spec.schemaRef must be in the format \"module@constraint\" (e.g., \"users@^1.0.0\")",
	ErrCodeSchemaLockMissing:        "dk.lock is missing an entry for this schemaRef — run 'dk lock' to resolve",
	ErrCodeSchemaLockChecksumFail:   "dk.lock checksum does not match resolved schema — run 'dk lock --upgrade'",
	ErrCodeSchemaBreakingChange:     "breaking schema change detected between locked and current version",
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
