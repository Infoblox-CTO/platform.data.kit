package validate

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// DataSet validation error codes.
const (
	ErrDataSetRequired    = "E070"
	ErrDataSetRefNotFound = "E076"
)

var (
	datasetNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,62}$`)
)

// DataSetValidator validates dataset manifests against structural rules.
type DataSetValidator struct {
	offline bool
}

// NewDataSetValidator creates a new DataSetValidator.
func NewDataSetValidator() *DataSetValidator {
	return &DataSetValidator{}
}

// NewOfflineDataSetValidator creates a validator that only does structural validation.
func NewOfflineDataSetValidator() *DataSetValidator {
	return &DataSetValidator{offline: true}
}

// Name returns the validator name.
func (v *DataSetValidator) Name() string {
	return "dataset"
}

// ValidateDataSet validates a single dataset manifest.
func (v *DataSetValidator) ValidateDataSet(_ context.Context, a *contracts.DataSetManifest) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if a == nil {
		errs.AddError(ErrDataSetRequired, "", "dataset manifest is nil")
		return errs
	}

	// Structural validation
	v.validateStructure(&errs, a)

	return errs
}

// validateStructure performs structural validation of the dataset manifest.
func (v *DataSetValidator) validateStructure(errs *contracts.ValidationErrors, a *contracts.DataSetManifest) {
	// Validate required fields
	if a.APIVersion == "" {
		errs.AddError(ErrDataSetRequired, "apiVersion", "apiVersion is required")
	}
	if a.Kind == "" {
		errs.AddError(ErrDataSetRequired, "kind", "kind is required")
	} else if a.Kind != "DataSet" {
		errs.AddError(ErrInvalidFormat, "kind", "kind must be 'DataSet'")
	}

	// Validate name
	if a.Metadata.Name == "" {
		errs.AddError(ErrDataSetRequired, "metadata.name", "metadata.name is required")
	} else if !datasetNamePattern.MatchString(a.Metadata.Name) {
		errs.AddErrorWithValue(ErrInvalidFormat, "metadata.name",
			"name must be DNS-safe (lowercase, 3-63 chars, start with letter)", a.Metadata.Name)
	}

	// Validate store reference
	if a.Spec.Store == "" {
		errs.AddError(ErrDataSetRequired, "spec.store", "spec.store is required")
	}

	// Validate that at least one locator is provided
	if a.Spec.Table == "" && a.Spec.Prefix == "" && a.Spec.Topic == "" {
		errs.Add(&contracts.ValidationError{
			Code:     ErrDataSetRequired,
			Field:    "spec",
			Message:  "at least one of spec.table, spec.prefix, or spec.topic is recommended",
			Severity: contracts.SeverityWarning,
		})
	}

	// Validate classification if set
	if a.Spec.Classification != "" {
		validClassifications := map[string]bool{
			"public": true, "internal": true, "confidential": true, "restricted": true,
		}
		if !validClassifications[a.Spec.Classification] {
			errs.AddErrorWithValue(ErrInvalidFormat, "spec.classification",
				"classification must be one of: public, internal, confidential, restricted",
				a.Spec.Classification)
		}
	}

	// Validate schema fields
	if len(a.Spec.Schema) > 0 {
		seen := make(map[string]bool)
		for i, f := range a.Spec.Schema {
			if f.Name == "" {
				errs.AddError(ErrDataSetRequired, fmt.Sprintf("spec.schema[%d].name", i),
					"schema field name is required")
			} else if seen[f.Name] {
				errs.AddError(ErrInvalidFormat, fmt.Sprintf("spec.schema[%d].name", i),
					fmt.Sprintf("duplicate schema field name: %s", f.Name))
			} else {
				seen[f.Name] = true
			}
			if f.Type == "" {
				errs.AddError(ErrDataSetRequired, fmt.Sprintf("spec.schema[%d].type", i),
					"schema field type is required")
			}
		}
	}
}
