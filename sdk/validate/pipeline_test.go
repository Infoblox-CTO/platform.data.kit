package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/data-platform/contracts"
)

func TestNewPipelineValidator(t *testing.T) {
	pipeline := &contracts.PipelineManifest{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "Pipeline",
		Metadata: contracts.PipelineMetadata{
			Name: "test-pipeline",
		},
		Spec: contracts.PipelineSpec{
			Image: "myorg/pipeline:latest",
		},
	}

	v := NewPipelineValidator(pipeline, "/path/to/pipeline")

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if v.Name() != "pipeline" {
		t.Errorf("Name() = %s, want pipeline", v.Name())
	}
}

func TestPipelineValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		pipeline  *contracts.PipelineManifest
		wantValid bool
		wantErrs  int
	}{
		{
			name:      "nil pipeline",
			pipeline:  nil,
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "valid pipeline",
			pipeline: &contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "valid-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Image: "docker.io/myorg/pipeline:v1.0.0",
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "missing apiVersion",
			pipeline: &contracts.PipelineManifest{
				Kind: "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "missing-api",
				},
				Spec: contracts.PipelineSpec{
					Image: "myorg/pipeline:latest",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing kind",
			pipeline: &contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Metadata: contracts.PipelineMetadata{
					Name: "missing-kind",
				},
				Spec: contracts.PipelineSpec{
					Image: "myorg/pipeline:latest",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "wrong kind",
			pipeline: &contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "WrongKind",
				Metadata: contracts.PipelineMetadata{
					Name: "wrong-kind",
				},
				Spec: contracts.PipelineSpec{
					Image: "myorg/pipeline:latest",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing image",
			pipeline: &contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "no-image",
				},
				Spec: contracts.PipelineSpec{},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "missing name",
			pipeline: &contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Spec: contracts.PipelineSpec{
					Image: "myorg/pipeline:latest",
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPipelineValidator(tt.pipeline, "/path")
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

func TestPipelineValidator_ValidateFromFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "file not found",
			path:    "testdata/nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPipelineValidatorFromFile(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
