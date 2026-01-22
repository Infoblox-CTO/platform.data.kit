package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/data-platform/contracts"
)

func TestNewDataPackageValidator(t *testing.T) {
	pkg := &contracts.DataPackage{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "DataPackage",
		Metadata: contracts.PackageMetadata{
			Name: "test-pkg",
		},
	}

	v := NewDataPackageValidator(pkg, "/path/to/pkg")

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if v.Name() != "datapackage" {
		t.Errorf("Name() = %s, want datapackage", v.Name())
	}
	if v.Package() != pkg {
		t.Error("Package() should return the same package")
	}
}

func TestDataPackageValidator_Validate(t *testing.T) {
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
			name: "valid package",
			pkg: &contracts.DataPackage{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "DataPackage",
				Metadata: contracts.PackageMetadata{
					Name:      "valid-pkg",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.DataPackageSpec{
					Type:        contracts.PackageTypeDataset,
					Description: "A test package",
					Owner:       "data-team",
					Outputs: []contracts.ArtifactContract{
						{
							Name:    "output-data",
							Type:    contracts.ArtifactTypeS3Prefix,
							Binding: "output-bucket",
						},
					},
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "missing apiVersion",
			pkg: &contracts.DataPackage{
				Kind: "DataPackage",
				Metadata: contracts.PackageMetadata{
					Name: "missing-api",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing kind",
			pkg: &contracts.DataPackage{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Metadata: contracts.PackageMetadata{
					Name: "missing-kind",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "wrong kind",
			pkg: &contracts.DataPackage{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "WrongKind",
				Metadata: contracts.PackageMetadata{
					Name: "wrong-kind",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "invalid apiVersion",
			pkg: &contracts.DataPackage{
				APIVersion: "invalid/version",
				Kind:       "DataPackage",
				Metadata: contracts.PackageMetadata{
					Name: "invalid-api",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDataPackageValidator(tt.pkg, "/path")
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

func TestDataPackageValidator_ValidateFromFile(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		wantValid bool
	}{
		{
			name:      "valid file",
			path:      "testdata/valid/datapackage-basic.yaml",
			wantErr:   false,
			wantValid: true,
		},
		{
			name:    "file not found",
			path:    "testdata/nonexistent.yaml",
			wantErr: true,
		},
		{
			name:      "missing name",
			path:      "testdata/invalid/missing-name.yaml",
			wantErr:   false,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewDataPackageValidatorFromFile(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			errs := v.Validate(context.Background())
			if tt.wantValid && errs.HasErrors() {
				t.Errorf("expected valid, got errors: %v", errs)
			}
			if !tt.wantValid && !errs.HasErrors() {
				t.Error("expected errors, got valid")
			}
		})
	}
}
