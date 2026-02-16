package validate

import (
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestNewPIIValidator(t *testing.T) {
	v := NewPIIValidator()

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if !v.RequireClassification {
		t.Error("RequireClassification should be true by default")
	}
	if len(v.AllowedSensitivities) == 0 {
		t.Error("AllowedSensitivities should not be empty")
	}
}

func TestPIIValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		model     *contracts.Model
		wantValid bool
		wantErrs  int
	}{
		{
			name:      "nil model",
			model:     nil,
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "model with no outputs",
			model: &contracts.Model{
				Spec: contracts.ModelSpec{
					Outputs: nil,
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "model with classified output",
			model: &contracts.Model{
				Spec: contracts.ModelSpec{
					Outputs: []contracts.ArtifactContract{
						{
							Name: "output1",
							Classification: &contracts.Classification{
								Sensitivity: contracts.SensitivityPublic,
							},
						},
					},
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "model with unclassified output",
			model: &contracts.Model{
				Spec: contracts.ModelSpec{
					Outputs: []contracts.ArtifactContract{
						{
							Name:           "output1",
							Classification: nil,
						},
					},
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "mixed classified and unclassified",
			model: &contracts.Model{
				Spec: contracts.ModelSpec{
					Outputs: []contracts.ArtifactContract{
						{
							Name: "output1",
							Classification: &contracts.Classification{
								Sensitivity: contracts.SensitivityConfidential,
							},
						},
						{
							Name:           "output2",
							Classification: nil,
						},
					},
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPIIValidator()
			errs := v.Validate(tt.model)

			if tt.wantValid && errs.HasErrors() {
				t.Errorf("expected valid, got errors: %v", errs)
			}
			if !tt.wantValid && !errs.HasErrors() {
				t.Error("expected errors, got valid")
			}
			if tt.wantErrs > 0 && len(errs) != tt.wantErrs {
				t.Errorf("len(errs) = %d, want %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestPIIValidator_InvalidSensitivity(t *testing.T) {
	model := &contracts.Model{
		Spec: contracts.ModelSpec{
			Outputs: []contracts.ArtifactContract{
				{
					Name: "output1",
					Classification: &contracts.Classification{
						Sensitivity: contracts.Sensitivity("invalid-sensitivity"),
					},
				},
			},
		},
	}

	v := NewPIIValidator()
	errs := v.Validate(model)

	if !errs.HasErrors() {
		t.Error("expected error for invalid sensitivity")
	}
}

func TestPIIValidator_DisableClassificationRequired(t *testing.T) {
	model := &contracts.Model{
		Spec: contracts.ModelSpec{
			Outputs: []contracts.ArtifactContract{
				{
					Name:           "output1",
					Classification: nil,
				},
			},
		},
	}

	v := NewPIIValidator()
	v.RequireClassification = false

	errs := v.Validate(model)

	if errs.HasErrors() {
		t.Errorf("should not error when classification not required: %v", errs)
	}
}
