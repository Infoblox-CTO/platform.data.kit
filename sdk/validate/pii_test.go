package validate

import (
	"testing"

	"github.com/Infoblox-CTO/data-platform/contracts"
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
		pkg       *contracts.DataPackage
		wantValid bool
		wantErrs  int
	}{
		{
			name:      "nil package",
			pkg:       nil,
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "package with no outputs",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Outputs: nil,
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "package with classified output",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
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
			name: "package with unclassified output",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
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
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
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
			errs := v.Validate(tt.pkg)

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
	pkg := &contracts.DataPackage{
		Spec: contracts.DataPackageSpec{
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
	errs := v.Validate(pkg)

	if !errs.HasErrors() {
		t.Error("expected error for invalid sensitivity")
	}
}

func TestPIIValidator_DisableClassificationRequired(t *testing.T) {
	pkg := &contracts.DataPackage{
		Spec: contracts.DataPackageSpec{
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

	errs := v.Validate(pkg)

	if errs.HasErrors() {
		t.Errorf("should not error when classification not required: %v", errs)
	}
}
