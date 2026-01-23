package validate

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/data.platform.kit/contracts"
)

// ArtifactValidator validates artifact contracts and their schema references.
type ArtifactValidator struct {
	artifact   *contracts.ArtifactContract
	packageDir string
	strictMode bool
}

// NewArtifactValidator creates a validator for an ArtifactContract.
func NewArtifactValidator(artifact *contracts.ArtifactContract, packageDir string) *ArtifactValidator {
	return &ArtifactValidator{
		artifact:   artifact,
		packageDir: packageDir,
		strictMode: false,
	}
}

// WithStrictMode enables strict validation.
func (v *ArtifactValidator) WithStrictMode(strict bool) *ArtifactValidator {
	v.strictMode = strict
	return v
}

// Name returns the validator name.
func (v *ArtifactValidator) Name() string {
	return "artifact"
}

// Validate validates the ArtifactContract.
func (v *ArtifactValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.artifact == nil {
		errs.AddError(ErrMissingRequired, "", "artifact is nil")
		return errs
	}

	// Validate required fields
	v.validateRequiredFields(&errs)

	// Validate schema
	v.validateSchema(&errs)

	// Validate classification
	v.validateClassification(&errs)

	return errs
}

// validateRequiredFields checks for required fields.
func (v *ArtifactValidator) validateRequiredFields(errs *contracts.ValidationErrors) {
	if v.artifact.Name == "" {
		errs.AddError(ErrMissingRequired, "name", "artifact name is required")
	} else if !isArtifactNameValid(v.artifact.Name) {
		errs.AddError(contracts.ErrCodeNameNotDNSSafe, "name", "artifact name must be DNS-safe")
	}

	// Validate type
	if !v.artifact.Type.IsValid() {
		errs.AddError(contracts.ErrCodeInvalidSchemaType, "type", "invalid artifact type")
	}

	// Validate binding reference
	if v.artifact.Binding == "" {
		errs.AddError(ErrMissingRequired, "binding", "artifact binding is required")
	}
}

// validateSchema validates schema configuration.
func (v *ArtifactValidator) validateSchema(errs *contracts.ValidationErrors) {
	if v.artifact.Schema == nil {
		// Schema is optional
		return
	}

	schema := v.artifact.Schema

	// Validate schema type if provided
	if schema.Type != "" && !schema.Type.IsValid() {
		errs.AddError(contracts.ErrCodeInvalidSchemaType, "schema.type", "invalid schema type")
	}

	// Validate schema file reference if provided
	if schema.SchemaRef != "" {
		schemaPath := filepath.Join(v.packageDir, schema.SchemaRef)
		if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
			errs.AddError(ErrFileNotFound, "schema.schemaRef", "schema file not found: "+schema.SchemaRef)
		} else if err == nil {
			// Validate schema content based on type
			v.validateSchemaContent(errs, schemaPath)
		}
	}
}

// validateSchemaContent validates the content of a schema file.
func (v *ArtifactValidator) validateSchemaContent(errs *contracts.ValidationErrors, schemaPath string) {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		errs.AddError(ErrFileNotFound, "schema.schemaRef", "failed to read schema file: "+err.Error())
		return
	}

	if v.artifact.Schema == nil {
		return
	}

	// Basic content validation based on type
	switch v.artifact.Schema.Type {
	case contracts.SchemaTypeAvro:
		if !artifactContainsBytes(data, []byte("\"type\"")) {
			errs.AddError(ErrSchemaError, "schema.schemaRef", "Avro schema must contain 'type' field")
		}
	case contracts.SchemaTypeJSON:
		if !artifactContainsBytes(data, []byte("\"$schema\"")) && !artifactContainsBytes(data, []byte("\"type\"")) {
			errs.AddError(ErrSchemaError, "schema.schemaRef", "JSON schema should contain '$schema' or 'type' field")
		}
	}
}

// validateClassification validates data classification settings.
func (v *ArtifactValidator) validateClassification(errs *contracts.ValidationErrors) {
	if v.artifact.Classification == nil {
		// Classification is optional
		return
	}

	class := v.artifact.Classification

	// Validate sensitivity level if provided
	if class.Sensitivity != "" && !class.Sensitivity.IsValid() {
		errs.AddError(ErrInvalidFormat, "classification.sensitivity", "invalid sensitivity level")
	}

	// Validate PII marker if set - in strict mode, PII requires high sensitivity
	if class.PII && v.strictMode {
		if class.Sensitivity != contracts.SensitivityConfidential && class.Sensitivity != contracts.SensitivityRestricted {
			errs.AddError(ErrInvalidFormat, "classification", "artifacts containing PII should have sensitivity of 'confidential' or 'restricted'")
		}
	}
}

// isArtifactNameValid checks if a string is a valid DNS-safe identifier.
func isArtifactNameValid(s string) bool {
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

// artifactContainsBytes checks if data contains the pattern.
func artifactContainsBytes(data, pattern []byte) bool {
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
