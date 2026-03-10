package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

func TestNewManifestValidator(t *testing.T) {
	tr := &contracts.Transform{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       string(contracts.KindTransform),
		Metadata: contracts.TransformMetadata{
			Name: "test-transform",
		},
		Spec: contracts.TransformSpec{
			Runtime: contracts.RuntimeGenericGo,
			Image:   "myimage:v1",
			Inputs:  []contracts.DataSetRef{{DataSet: "in"}},
			Outputs: []contracts.DataSetRef{{DataSet: "out"}},
		},
	}

	v := NewManifestValidator(tr, contracts.KindTransform, "/path/to/pkg")

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if v.Name() != "manifest" {
		t.Errorf("Name() = %s, want manifest", v.Name())
	}
	if v.Manifest() != tr {
		t.Error("Manifest() should return the same manifest")
	}
	if v.Kind() != contracts.KindTransform {
		t.Errorf("Kind() = %s, want Transform", v.Kind())
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
			kind:      contracts.KindTransform,
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "valid connector",
			manifest: &contracts.Connector{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindConnector),
				Metadata:   contracts.ConnectorMetadata{Name: "postgres"},
				Spec: contracts.ConnectorSpec{
					Type:         "postgres",
					Capabilities: []string{"source", "destination"},
				},
			},
			kind:      contracts.KindConnector,
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "valid store",
			manifest: &contracts.Store{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindStore),
				Metadata:   contracts.StoreMetadata{Name: "warehouse"},
				Spec: contracts.StoreSpec{
					Connector:  "postgres",
					Connection: map[string]any{"host": "localhost"},
				},
			},
			kind:      contracts.KindStore,
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "valid transform",
			manifest: &contracts.Transform{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindTransform),
				Metadata: contracts.TransformMetadata{
					Name: "valid-transform",
				},
				Spec: contracts.TransformSpec{
					Runtime: contracts.RuntimeGenericGo,
					Image:   "myimage:v1",
					Mode:    contracts.ModeBatch,
					Trigger: &contracts.TriggerSpec{Policy: contracts.TriggerPolicySchedule, Schedule: &contracts.ScheduleSpec{Cron: "0 * * * *"}},
					Inputs:  []contracts.DataSetRef{{DataSet: "in"}},
					Outputs: []contracts.DataSetRef{{DataSet: "out"}},
				},
			},
			kind:      contracts.KindTransform,
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "transform missing name",
			manifest: &contracts.Transform{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindTransform),
				Metadata:   contracts.TransformMetadata{},
				Spec: contracts.TransformSpec{
					Runtime: contracts.RuntimeGenericGo,
					Image:   "myimage:v1",
					Inputs:  []contracts.DataSetRef{{DataSet: "in"}},
					Outputs: []contracts.DataSetRef{{DataSet: "out"}},
				},
			},
			kind:      contracts.KindTransform,
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
			name:      "valid transform file",
			path:      "testdata/valid/transform-basic.yaml",
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

func TestManifestValidator_ConnectorValidation(t *testing.T) {
	tests := []struct {
		name      string
		connector *contracts.Connector
		wantValid bool
	}{
		{
			name: "valid connector",
			connector: &contracts.Connector{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindConnector),
				Metadata:   contracts.ConnectorMetadata{Name: "postgres"},
				Spec: contracts.ConnectorSpec{
					Type:         "postgres",
					Capabilities: []string{"source", "destination"},
				},
			},
			wantValid: true,
		},
		{
			name: "connector missing type",
			connector: &contracts.Connector{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindConnector),
				Metadata:   contracts.ConnectorMetadata{Name: "postgres"},
				Spec: contracts.ConnectorSpec{
					Capabilities: []string{"source"},
				},
			},
			wantValid: false,
		},
		{
			name: "connector missing capabilities",
			connector: &contracts.Connector{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindConnector),
				Metadata:   contracts.ConnectorMetadata{Name: "postgres"},
				Spec: contracts.ConnectorSpec{
					Type: "postgres",
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.connector, contracts.KindConnector, "/path")
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

func TestManifestValidator_StoreValidation(t *testing.T) {
	tests := []struct {
		name      string
		store     *contracts.Store
		wantValid bool
	}{
		{
			name: "valid store",
			store: &contracts.Store{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindStore),
				Metadata:   contracts.StoreMetadata{Name: "warehouse"},
				Spec: contracts.StoreSpec{
					Connector:  "postgres",
					Connection: map[string]any{"host": "localhost"},
				},
			},
			wantValid: true,
		},
		{
			name: "store missing connector",
			store: &contracts.Store{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindStore),
				Metadata:   contracts.StoreMetadata{Name: "warehouse"},
				Spec: contracts.StoreSpec{
					Connection: map[string]any{"host": "localhost"},
				},
			},
			wantValid: false,
		},
		{
			name: "store missing connection",
			store: &contracts.Store{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindStore),
				Metadata:   contracts.StoreMetadata{Name: "warehouse"},
				Spec: contracts.StoreSpec{
					Connector: "postgres",
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.store, contracts.KindStore, "/path")
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

func TestManifestValidator_TransformValidation(t *testing.T) {
	tests := []struct {
		name         string
		transform    *contracts.Transform
		wantValid    bool
		wantErrCode  string
		wantErrField string
	}{
		{
			name: "transform without inputs is invalid",
			transform: &contracts.Transform{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindTransform),
				Metadata:   contracts.TransformMetadata{Name: "test-transform"},
				Spec: contracts.TransformSpec{
					Runtime: contracts.RuntimeGenericGo,
					Image:   "myimage:v1",
					Mode:    contracts.ModeBatch,
					Trigger: &contracts.TriggerSpec{Policy: contracts.TriggerPolicySchedule, Schedule: &contracts.ScheduleSpec{Cron: "0 * * * *"}},
					Outputs: []contracts.DataSetRef{{DataSet: "out"}},
				},
			},
			wantValid:    false,
			wantErrCode:  contracts.ErrCodeTransformInputsRequired,
			wantErrField: "spec.inputs",
		},
		{
			name: "transform with valid inputs/outputs is valid",
			transform: &contracts.Transform{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindTransform),
				Metadata:   contracts.TransformMetadata{Name: "test-transform"},
				Spec: contracts.TransformSpec{
					Runtime: contracts.RuntimeGenericGo,
					Image:   "myimage:v1",
					Mode:    contracts.ModeBatch,
					Trigger: &contracts.TriggerSpec{Policy: contracts.TriggerPolicySchedule, Schedule: &contracts.ScheduleSpec{Cron: "0 * * * *"}},
					Inputs:  []contracts.DataSetRef{{DataSet: "in"}},
					Outputs: []contracts.DataSetRef{{DataSet: "out"}},
				},
			},
			wantValid: true,
		},
		{
			name: "transform with invalid mode",
			transform: &contracts.Transform{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       string(contracts.KindTransform),
				Metadata:   contracts.TransformMetadata{Name: "test-transform"},
				Spec: contracts.TransformSpec{
					Runtime: contracts.RuntimeGenericGo,
					Image:   "myimage:v1",
					Mode:    contracts.Mode("invalid"),
					Inputs:  []contracts.DataSetRef{{DataSet: "in"}},
					Outputs: []contracts.DataSetRef{{DataSet: "out"}},
				},
			},
			wantValid:    false,
			wantErrCode:  ErrInvalidFormat,
			wantErrField: "spec.mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewManifestValidator(tt.transform, contracts.KindTransform, "/path/to/pkg")
			errs := v.Validate(context.Background())

			if tt.wantValid {
				if errs.HasErrors() {
					t.Errorf("expected valid, got errors: %v", errs)
				}
			} else {
				if !errs.HasErrors() {
					t.Error("expected errors, got valid")
				}
				if tt.wantErrCode != "" {
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
			}
		})
	}
}

func TestManifestValidator_ScheduleBatchWarning(t *testing.T) {
	// Batch transform without schedule should produce W209 warning.
	tr := &contracts.Transform{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       string(contracts.KindTransform),
		Metadata:   contracts.TransformMetadata{Name: "my-transform"},
		Spec: contracts.TransformSpec{
			Runtime: contracts.RuntimeGenericGo,
			Image:   "myimage:v1",
			Mode:    contracts.ModeBatch,
			Inputs:  []contracts.DataSetRef{{DataSet: "in"}},
			Outputs: []contracts.DataSetRef{{DataSet: "out"}},
		},
	}

	v := NewManifestValidator(tr, contracts.KindTransform, "/path")
	errs := v.Validate(context.Background())

	if errs.HasErrors() {
		t.Errorf("expected no errors, got: %v", errs)
	}
	if !hasErrorCode(errs, contracts.WarnCodeTriggerBatchMode) {
		t.Errorf("expected W209 warning for batch without trigger, got: %v", errs)
	}

	// Streaming transform without schedule should NOT produce W209.
	tr.Spec.Mode = contracts.ModeStreaming
	v2 := NewManifestValidator(tr, contracts.KindTransform, "/path")
	errs2 := v2.Validate(context.Background())

	if hasErrorCode(errs2, contracts.WarnCodeTriggerBatchMode) {
		t.Errorf("streaming transform should not get W209 warning, got: %v", errs2)
	}
}

// hasErrorCode checks if the errors contain a specific code.
func hasErrorCode(errs contracts.ValidationErrors, code string) bool {
	for _, e := range errs {
		if e.Code == code {
			return true
		}
	}
	return false
}
