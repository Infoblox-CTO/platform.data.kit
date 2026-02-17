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
					Schedule:    &contracts.ScheduleSpec{Cron: "0 * * * *"},
					Outputs: []contracts.ArtifactContract{
						{
							Name:           "output-data",
							Type:           contracts.ArtifactTypeS3Prefix,
							Binding:        "output-bucket",
							Classification: &contracts.Classification{},
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
					Description:  "A test source",
					Owner:        "data-team",
					Runtime:      contracts.RuntimeCloudQuery,
					Provides:     contracts.ArtifactContract{Name: "cloud-assets", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
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
					Description:  "A test destination",
					Owner:        "data-team",
					Runtime:      contracts.RuntimeCloudQuery,
					Accepts:      contracts.ArtifactContract{Name: "cloud-assets", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
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
					Schedule:    &contracts.ScheduleSpec{Cron: "0 * * * *"},
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
					Schedule:    &contracts.ScheduleSpec{Cron: "0 * * * *"},
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
						{Name: "output1", Type: contracts.ArtifactTypeS3Prefix, Binding: "output.data", Classification: &contracts.Classification{}},
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
					Description:  "A CloudQuery source",
					Owner:        "data-team",
					Runtime:      contracts.RuntimeCloudQuery,
					Provides:     contracts.ArtifactContract{Name: "cloud-assets", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
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

func TestManifestValidator_SourceProvidesRequired(t *testing.T) {
	tests := []struct {
		name        string
		source      *contracts.Source
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "source without provides triggers E102",
			source: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata:   contracts.ExtMetadata{Name: "my-source", Namespace: "data-team", Version: "1.0.0"},
				Spec: contracts.SourceSpec{
					Description:  "No provides",
					Owner:        "data-team",
					Runtime:      contracts.RuntimeCloudQuery,
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			wantErr:     true,
			wantErrCode: contracts.ErrCodeSourceProvidesRequired,
		},
		{
			name: "source with provides is valid",
			source: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata:   contracts.ExtMetadata{Name: "my-source", Namespace: "data-team", Version: "1.0.0"},
				Spec: contracts.SourceSpec{
					Description:  "Has provides",
					Owner:        "data-team",
					Runtime:      contracts.RuntimeCloudQuery,
					Provides:     contracts.ArtifactContract{Name: "cloud-assets", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.source, contracts.KindSource, "/path")
			errs := v.Validate(context.Background())
			found := hasErrorCode(errs, tt.wantErrCode)
			if tt.wantErr && !found {
				t.Errorf("expected error %s, got: %v", tt.wantErrCode, errs)
			}
			if !tt.wantErr && found {
				t.Errorf("did not expect error %s, got: %v", tt.wantErrCode, errs)
			}
		})
	}
}

func TestManifestValidator_DestAcceptsRequired(t *testing.T) {
	tests := []struct {
		name        string
		dest        *contracts.Destination
		wantErr     bool
		wantErrCode string
	}{
		{
			name: "destination without accepts triggers E103",
			dest: &contracts.Destination{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindDestination),
				Metadata:   contracts.ExtMetadata{Name: "my-dest", Namespace: "data-team", Version: "1.0.0"},
				Spec: contracts.DestinationSpec{
					Description:  "No accepts",
					Owner:        "data-team",
					Runtime:      contracts.RuntimeCloudQuery,
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			wantErr:     true,
			wantErrCode: contracts.ErrCodeDestAcceptsRequired,
		},
		{
			name: "destination with accepts is valid",
			dest: &contracts.Destination{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindDestination),
				Metadata:   contracts.ExtMetadata{Name: "my-dest", Namespace: "data-team", Version: "1.0.0"},
				Spec: contracts.DestinationSpec{
					Description:  "Has accepts",
					Owner:        "data-team",
					Runtime:      contracts.RuntimeCloudQuery,
					Accepts:      contracts.ArtifactContract{Name: "cloud-assets", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.dest, contracts.KindDestination, "/path")
			errs := v.Validate(context.Background())
			found := hasErrorCode(errs, tt.wantErrCode)
			if tt.wantErr && !found {
				t.Errorf("expected error %s, got: %v", tt.wantErrCode, errs)
			}
			if !tt.wantErr && found {
				t.Errorf("did not expect error %s, got: %v", tt.wantErrCode, errs)
			}
		})
	}
}

func TestManifestValidator_ImageRequiredForGeneric(t *testing.T) {
	tests := []struct {
		name     string
		manifest manifest.Manifest
		kind     contracts.Kind
		wantErr  bool
	}{
		{
			name: "generic-go source without image triggers E104",
			manifest: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata:   contracts.ExtMetadata{Name: "my-source", Namespace: "ns", Version: "1.0.0"},
				Spec: contracts.SourceSpec{
					Description:  "Generic source",
					Owner:        "team",
					Runtime:      contracts.RuntimeGenericGo,
					Provides:     contracts.ArtifactContract{Name: "data", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			kind:    contracts.KindSource,
			wantErr: true,
		},
		{
			name: "generic-go source with image is valid",
			manifest: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata:   contracts.ExtMetadata{Name: "my-source", Namespace: "ns", Version: "1.0.0"},
				Spec: contracts.SourceSpec{
					Description:  "Generic source",
					Owner:        "team",
					Runtime:      contracts.RuntimeGenericGo,
					Image:        "myimage:v1",
					Provides:     contracts.ArtifactContract{Name: "data", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			kind:    contracts.KindSource,
			wantErr: false,
		},
		{
			name: "generic-python destination without image triggers E104",
			manifest: &contracts.Destination{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindDestination),
				Metadata:   contracts.ExtMetadata{Name: "my-dest", Namespace: "ns", Version: "1.0.0"},
				Spec: contracts.DestinationSpec{
					Description:  "Generic dest",
					Owner:        "team",
					Runtime:      contracts.RuntimeGenericPython,
					Accepts:      contracts.ArtifactContract{Name: "data", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			kind:    contracts.KindDestination,
			wantErr: true,
		},
		{
			name: "cloudquery source without image is valid (not generic)",
			manifest: &contracts.Source{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindSource),
				Metadata:   contracts.ExtMetadata{Name: "my-source", Namespace: "ns", Version: "1.0.0"},
				Spec: contracts.SourceSpec{
					Description:  "CloudQuery source",
					Owner:        "team",
					Runtime:      contracts.RuntimeCloudQuery,
					Provides:     contracts.ArtifactContract{Name: "data", Type: contracts.ArtifactTypeS3Prefix},
					ConfigSchema: &contracts.ConfigSchema{},
				},
			},
			kind:    contracts.KindSource,
			wantErr: false,
		},
		{
			name: "generic-go model without image triggers E104",
			manifest: &contracts.Model{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindModel),
				Metadata:   contracts.ModelMetadata{Name: "my-model", Namespace: "ns", Version: "1.0.0"},
				Spec: contracts.ModelSpec{
					Description: "Generic model",
					Owner:       "team",
					Runtime:     contracts.RuntimeGenericGo,
					Mode:        contracts.ModeBatch,
					Schedule:    &contracts.ScheduleSpec{Cron: "0 * * * *"},
					Outputs: []contracts.ArtifactContract{
						{Name: "out", Type: contracts.ArtifactTypeS3Prefix, Binding: "b", Classification: &contracts.Classification{}},
					},
				},
			},
			kind:    contracts.KindModel,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.manifest, tt.kind, "/path")
			errs := v.Validate(context.Background())
			found := hasErrorCode(errs, contracts.ErrCodeImageRequiredGeneric)
			if tt.wantErr && !found {
				t.Errorf("expected E104, got: %v", errs)
			}
			if !tt.wantErr && found {
				t.Errorf("did not expect E104, got: %v", errs)
			}
		})
	}
}

func TestManifestValidator_ConfigSchemaWarning(t *testing.T) {
	// Source without configSchema should produce W104 warning.
	source := &contracts.Source{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       string(contracts.KindSource),
		Metadata:   contracts.ExtMetadata{Name: "my-source", Namespace: "ns", Version: "1.0.0"},
		Spec: contracts.SourceSpec{
			Description: "No config schema",
			Owner:       "team",
			Runtime:     contracts.RuntimeCloudQuery,
			Provides:    contracts.ArtifactContract{Name: "data", Type: contracts.ArtifactTypeS3Prefix},
		},
	}

	v := NewManifestValidator(source, contracts.KindSource, "/path")
	errs := v.Validate(context.Background())

	if errs.HasErrors() {
		t.Errorf("expected no errors, got: %v", errs)
	}
	if !errs.HasWarnings() {
		t.Error("expected W104 warning for missing configSchema")
	}
	if !hasErrorCode(errs, contracts.WarnCodeConfigSchemaMissing) {
		t.Errorf("expected W104 code, got: %v", errs)
	}
}

func TestManifestValidator_ClassificationRequired(t *testing.T) {
	// Model output without classification should produce E004.
	model := &contracts.Model{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       string(contracts.KindModel),
		Metadata:   contracts.ModelMetadata{Name: "my-model", Namespace: "ns", Version: "1.0.0"},
		Spec: contracts.ModelSpec{
			Description: "Missing classification",
			Owner:       "team",
			Runtime:     contracts.RuntimeGenericGo,
			Image:       "myimage:v1",
			Mode:        contracts.ModeBatch,
			Schedule:    &contracts.ScheduleSpec{Cron: "0 * * * *"},
			Outputs: []contracts.ArtifactContract{
				{Name: "out", Type: contracts.ArtifactTypeS3Prefix, Binding: "b"},
			},
		},
	}

	v := NewManifestValidator(model, contracts.KindModel, "/path")
	errs := v.Validate(context.Background())

	if !hasErrorCode(errs, contracts.ErrCodeClassificationRequired) {
		t.Errorf("expected E004 for missing classification on output, got: %v", errs)
	}
}

func TestManifestValidator_ScheduleBatchWarning(t *testing.T) {
	// Batch model without schedule should produce W209 warning.
	model := &contracts.Model{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       string(contracts.KindModel),
		Metadata:   contracts.ModelMetadata{Name: "my-model", Namespace: "ns", Version: "1.0.0"},
		Spec: contracts.ModelSpec{
			Description: "Batch no schedule",
			Owner:       "team",
			Runtime:     contracts.RuntimeGenericGo,
			Image:       "myimage:v1",
			Mode:        contracts.ModeBatch,
			Outputs: []contracts.ArtifactContract{
				{Name: "out", Type: contracts.ArtifactTypeS3Prefix, Binding: "b", Classification: &contracts.Classification{}},
			},
		},
	}

	v := NewManifestValidator(model, contracts.KindModel, "/path")
	errs := v.Validate(context.Background())

	if errs.HasErrors() {
		t.Errorf("expected no errors, got: %v", errs)
	}
	if !hasErrorCode(errs, contracts.WarnCodeScheduleBatchMode) {
		t.Errorf("expected W209 warning for batch without schedule, got: %v", errs)
	}

	// Streaming model without schedule should NOT produce W209.
	model.Spec.Mode = contracts.ModeStreaming
	v2 := NewManifestValidator(model, contracts.KindModel, "/path")
	errs2 := v2.Validate(context.Background())

	if hasErrorCode(errs2, contracts.WarnCodeScheduleBatchMode) {
		t.Errorf("streaming model should not get W209 warning, got: %v", errs2)
	}
}