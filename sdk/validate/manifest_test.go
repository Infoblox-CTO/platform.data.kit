package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

func TestNewManifestValidator(t *testing.T) {
	model := &contracts.Model{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       string(contracts.KindModel),
		Metadata: contracts.ModelMetadata{
			Name: "test-model",
		},
	}

	v := NewManifestValidator(model, contracts.KindModel, "/path/to/pkg")

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if v.Name() != "manifest" {
		t.Errorf("Name() = %s, want manifest", v.Name())
	}
	if v.Manifest() != model {
		t.Error("Manifest() should return the same manifest")
	}
	if v.Kind() != contracts.KindModel {
		t.Errorf("Kind() = %s, want Model", v.Kind())
	}
}

func TestManifestValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		manifest  manifest.Manifest
		kind      contracts.Kind
		wantValid bool
		wantErrs  int
	}{
		{
			name:      "nil manifest",
			manifest:  nil,
			kind:      contracts.KindModel,
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "valid model",
			manifest: &contracts.Model{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindModel),
				Metadata: contracts.ModelMetadata{
					Name:      "valid-model",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.ModelSpec{
					Description: "A test model",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeGenericGo,
					Image:       "myimage:v1",
					Mode:        contracts.ModeBatch,
					Outputs: []contracts.ArtifactContract{
						{
							Name:    "output-data",
							Type:    contracts.ArtifactTypeS3Prefix,
							Binding: "output-bucket",
						},
					},
				},
			},
			kind:      contracts.KindModel,
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "valid source",
			manifest: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata: contracts.ExtMetadata{
					Name:      "valid-source",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.SourceSpec{
					Description: "A test source",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeCloudQuery,
				},
			},
			kind:      contracts.KindSource,
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "valid destination",
			manifest: &contracts.Destination{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindDestination),
				Metadata: contracts.ExtMetadata{
					Name:      "valid-destination",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.DestinationSpec{
					Description: "A test destination",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeCloudQuery,
				},
			},
			kind:      contracts.KindDestination,
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "model missing name",
			manifest: &contracts.Model{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindModel),
				Metadata: contracts.ModelMetadata{
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.ModelSpec{
					Description: "Missing name",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeGenericGo,
				},
			},
			kind:      contracts.KindModel,
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "model missing description",
			manifest: &contracts.Model{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindModel),
				Metadata: contracts.ModelMetadata{
					Name:      "no-desc",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.ModelSpec{
					Owner:   "data-team",
					Runtime: contracts.RuntimeGenericGo,
				},
			},
			kind:      contracts.KindModel,
			wantValid: false,
			wantErrs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.manifest, tt.kind, "/path")
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

func TestManifestValidator_ValidateFromFile(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantErr   bool
		wantValid bool
	}{
		{
			name:      "valid model file",
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
			v, err := NewManifestValidatorFromFile(tt.path)

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

func TestManifestValidator_ModelValidation(t *testing.T) {
	tests := []struct {
		name         string
		model        *contracts.Model
		wantValid    bool
		wantErrCode  string
		wantErrField string
	}{
		{
			name: "model without outputs is invalid",
			model: &contracts.Model{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindModel),
				Metadata: contracts.ModelMetadata{
					Name:      "test-model",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.ModelSpec{
					Description: "A test model",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeGenericGo,
					Image:       "myimage:v1",
					Mode:        contracts.ModeBatch,
				},
			},
			wantValid:    false,
			wantErrCode:  contracts.ErrCodeOutputsRequired,
			wantErrField: "spec.outputs",
		},
		{
			name: "model with valid outputs is valid",
			model: &contracts.Model{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindModel),
				Metadata: contracts.ModelMetadata{
					Name:      "test-model",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.ModelSpec{
					Description: "A test model",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeGenericGo,
					Image:       "myimage:v1",
					Mode:        contracts.ModeBatch,
					Outputs: []contracts.ArtifactContract{
						{Name: "output1", Type: contracts.ArtifactTypeS3Prefix, Binding: "output.data", Classification: &contracts.Classification{}},
					},
				},
			},
			wantValid: true,
		},
		{
			name: "model with invalid mode",
			model: &contracts.Model{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindModel),
				Metadata: contracts.ModelMetadata{
					Name:      "test-model",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.ModelSpec{
					Description: "A test model",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeGenericGo,
					Image:       "myimage:v1",
					Mode:        contracts.Mode("invalid"),
					Outputs: []contracts.ArtifactContract{
						{Name: "output1", Type: contracts.ArtifactTypeS3Prefix, Binding: "output.data"},
					},
				},
			},
			wantValid:    false,
			wantErrCode:  ErrInvalidFormat,
			wantErrField: "spec.mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.model, contracts.KindModel, "/path/to/pkg")
			errs := v.Validate(context.Background())

			if tt.wantValid {
				if errs.HasErrors() {
					t.Errorf("expected valid, got errors: %v", errs)
				}
			} else {
				if !errs.HasErrors() {
					t.Error("expected errors, got valid")
				}
				found := false
				for _, e := range errs {
					if e.Code == tt.wantErrCode && e.Field == tt.wantErrField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error code %s at field %s, got: %v", tt.wantErrCode, tt.wantErrField, errs)
				}
			}
		})
	}
}

func TestManifestValidator_SourceValidation(t *testing.T) {
	tests := []struct {
		name      string
		source    *contracts.Source
		wantValid bool
	}{
		{
			name: "valid source with cloudquery runtime",
			source: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata: contracts.ExtMetadata{
					Name:      "my-source",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.SourceSpec{
					Description: "A CloudQuery source",
					Owner:       "data-team",
					Runtime:     contracts.RuntimeCloudQuery,
				},
			},
			wantValid: true,
		},
		{
			name: "source with invalid runtime",
			source: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata: contracts.ExtMetadata{
					Name:      "my-source",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.SourceSpec{
					Description: "Bad runtime",
					Owner:       "data-team",
					Runtime:     contracts.Runtime("nope"),
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.source, contracts.KindSource, "/path")
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
