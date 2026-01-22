package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/data-platform/contracts"
)

func TestNewArtifactValidator(t *testing.T) {
	artifact := &contracts.ArtifactContract{
		Name:    "test-artifact",
		Type:    contracts.ArtifactTypeS3Prefix,
		Binding: "input-data",
	}

	v := NewArtifactValidator(artifact, "/path/to/pkg")

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if v.Name() != "artifact" {
		t.Errorf("Name() = %s, want artifact", v.Name())
	}
}

func TestArtifactValidator_WithStrictMode(t *testing.T) {
	artifact := &contracts.ArtifactContract{
		Name:    "test-artifact",
		Binding: "input-data",
	}

	v := NewArtifactValidator(artifact, "/path")
	v2 := v.WithStrictMode(true)

	if v2 != v {
		t.Error("WithStrictMode should return same validator")
	}
	if !v.strictMode {
		t.Error("strictMode should be true")
	}
}

func TestArtifactValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		artifact  *contracts.ArtifactContract
		wantValid bool
		wantErrs  int
	}{
		{
			name:      "nil artifact",
			artifact:  nil,
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "valid artifact",
			artifact: &contracts.ArtifactContract{
				Name:    "output-data",
				Type:    contracts.ArtifactTypeS3Prefix,
				Binding: "output-binding",
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "missing name",
			artifact: &contracts.ArtifactContract{
				Type:    contracts.ArtifactTypeS3Prefix,
				Binding: "binding",
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing binding",
			artifact: &contracts.ArtifactContract{
				Name: "artifact",
				Type: contracts.ArtifactTypeS3Prefix,
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "invalid artifact name",
			artifact: &contracts.ArtifactContract{
				Name:    "INVALID_NAME",
				Type:    contracts.ArtifactTypeS3Prefix,
				Binding: "binding",
			},
			wantValid: false,
			wantErrs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewArtifactValidator(tt.artifact, "/path")
			errs := v.Validate(context.Background())

			if tt.wantValid && errs.HasErrors() {
				t.Errorf("expected valid, got errors: %v", errs)
			}
			if !tt.wantValid && !errs.HasErrors() {
				t.Error("expected errors, got valid")
			}
			if tt.wantErrs > 0 && len(errs) < tt.wantErrs {
				t.Errorf("len(errs) = %d, want at least %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestArtifactValidator_ValidateSchema(t *testing.T) {
	tests := []struct {
		name      string
		artifact  *contracts.ArtifactContract
		wantValid bool
	}{
		{
			name: "no schema",
			artifact: &contracts.ArtifactContract{
				Name:    "artifact",
				Type:    contracts.ArtifactTypeS3Prefix,
				Binding: "binding",
				Schema:  nil,
			},
			wantValid: true,
		},
		{
			name: "valid schema type",
			artifact: &contracts.ArtifactContract{
				Name:    "artifact",
				Type:    contracts.ArtifactTypeS3Prefix,
				Binding: "binding",
				Schema: &contracts.SchemaSpec{
					Type: contracts.SchemaTypeParquet,
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewArtifactValidator(tt.artifact, "/path")
			errs := v.Validate(context.Background())

			if tt.wantValid && errs.HasErrors() {
				t.Errorf("expected valid, got errors: %v", errs)
			}
		})
	}
}

func TestArtifactValidator_ValidateClassification(t *testing.T) {
	tests := []struct {
		name      string
		artifact  *contracts.ArtifactContract
		wantValid bool
	}{
		{
			name: "no classification",
			artifact: &contracts.ArtifactContract{
				Name:           "artifact",
				Type:           contracts.ArtifactTypeS3Prefix,
				Binding:        "binding",
				Classification: nil,
			},
			wantValid: true,
		},
		{
			name: "with classification",
			artifact: &contracts.ArtifactContract{
				Name:    "artifact",
				Type:    contracts.ArtifactTypeS3Prefix,
				Binding: "binding",
				Classification: &contracts.Classification{
					Sensitivity: contracts.SensitivityConfidential,
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewArtifactValidator(tt.artifact, "/path")
			errs := v.Validate(context.Background())

			if tt.wantValid && errs.HasErrors() {
				t.Errorf("expected valid, got errors: %v", errs)
			}
		})
	}
}
