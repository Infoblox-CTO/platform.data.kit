package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestPipelineWorkflowValidator_ValidPipeline(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "PipelineWorkflow",
		Metadata:   contracts.PipelineWorkflowMetadata{Name: "my-pipeline"},
		Steps: []contracts.Step{
			{Name: "sync-data", Type: contracts.StepTypeSync, Input: "my-source", Output: "my-sink"},
			{Name: "transform-data", Type: contracts.StepTypeTransform, Asset: "my-model"},
			{Name: "test-data", Type: contracts.StepTypeTest, Asset: "my-model", Command: []string{"dbt", "test"}},
			{Name: "publish-results", Type: contracts.StepTypePublish},
			{Name: "custom-step", Type: contracts.StepTypeCustom, Image: "my-image:latest"},
		},
	}
	v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
	errs := v.Validate(context.Background())
	if errs.HasErrors() {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestPipelineWorkflowValidator_NilWorkflow(t *testing.T) {
	v := NewPipelineWorkflowValidator(nil, "pipeline.yaml")
	errs := v.Validate(context.Background())
	if !errs.HasErrors() {
		t.Error("expected error for nil workflow")
	}
}

func TestPipelineWorkflowValidator_MissingRequired(t *testing.T) {
	tests := []struct {
		name     string
		pw       contracts.PipelineWorkflow
		wantCode string
	}{
		{
			name:     "missing apiVersion",
			pw:       contracts.PipelineWorkflow{Kind: "PipelineWorkflow", Metadata: contracts.PipelineWorkflowMetadata{Name: "test-pipe"}, Steps: []contracts.Step{{Name: "step-one", Type: contracts.StepTypePublish}}},
			wantCode: ErrPipelineMissingRequired,
		},
		{
			name:     "missing kind",
			pw:       contracts.PipelineWorkflow{APIVersion: "data.infoblox.com/v1alpha1", Metadata: contracts.PipelineWorkflowMetadata{Name: "test-pipe"}, Steps: []contracts.Step{{Name: "step-one", Type: contracts.StepTypePublish}}},
			wantCode: ErrPipelineMissingRequired,
		},
		{
			name:     "missing metadata.name",
			pw:       contracts.PipelineWorkflow{APIVersion: "data.infoblox.com/v1alpha1", Kind: "PipelineWorkflow", Steps: []contracts.Step{{Name: "step-one", Type: contracts.StepTypePublish}}},
			wantCode: ErrPipelineMissingRequired,
		},
		{
			name:     "empty steps",
			pw:       contracts.PipelineWorkflow{APIVersion: "data.infoblox.com/v1alpha1", Kind: "PipelineWorkflow", Metadata: contracts.PipelineWorkflowMetadata{Name: "test-pipe"}},
			wantCode: ErrPipelineEmptySteps,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPipelineWorkflowValidator(&tt.pw, "pipeline.yaml")
			errs := v.Validate(context.Background())
			if !hasErrorCode(errs, tt.wantCode) {
				t.Errorf("expected error code %s, got %v", tt.wantCode, errs)
			}
		})
	}
}

func TestPipelineWorkflowValidator_InvalidAPIVersion(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		APIVersion: "wrong/version",
		Kind:       "PipelineWorkflow",
		Metadata:   contracts.PipelineWorkflowMetadata{Name: "test-pipe"},
		Steps:      []contracts.Step{{Name: "step-one", Type: contracts.StepTypePublish}},
	}
	v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrPipelineInvalidAPIVersion) {
		t.Errorf("expected error code %s, got %v", ErrPipelineInvalidAPIVersion, errs)
	}
}

func TestPipelineWorkflowValidator_InvalidKind(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "WrongKind",
		Metadata:   contracts.PipelineWorkflowMetadata{Name: "test-pipe"},
		Steps:      []contracts.Step{{Name: "step-one", Type: contracts.StepTypePublish}},
	}
	v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrPipelineInvalidKind) {
		t.Errorf("expected error code %s, got %v", ErrPipelineInvalidKind, errs)
	}
}

func TestPipelineWorkflowValidator_InvalidPipelineName(t *testing.T) {
	tests := []struct {
		name     string
		pipeName string
	}{
		{name: "uppercase", pipeName: "MyPipeline"},
		{name: "starts with digit", pipeName: "1pipeline"},
		{name: "too short", pipeName: "ab"},
		{name: "special chars", pipeName: "my_pipeline!"},
		{name: "spaces", pipeName: "my pipeline"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw := &contracts.PipelineWorkflow{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "PipelineWorkflow",
				Metadata:   contracts.PipelineWorkflowMetadata{Name: tt.pipeName},
				Steps:      []contracts.Step{{Name: "step-one", Type: contracts.StepTypePublish}},
			}
			v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
			errs := v.Validate(context.Background())
			if !hasErrorCode(errs, ErrPipelineInvalidName) {
				t.Errorf("expected error code %s for name %q, got %v", ErrPipelineInvalidName, tt.pipeName, errs)
			}
		})
	}
}

func TestPipelineWorkflowValidator_InvalidStepName(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "PipelineWorkflow",
		Metadata:   contracts.PipelineWorkflowMetadata{Name: "test-pipe"},
		Steps:      []contracts.Step{{Name: "AB", Type: contracts.StepTypePublish}},
	}
	v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrPipelineInvalidStepName) {
		t.Errorf("expected error code %s, got %v", ErrPipelineInvalidStepName, errs)
	}
}

func TestPipelineWorkflowValidator_DuplicateStepName(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "PipelineWorkflow",
		Metadata:   contracts.PipelineWorkflowMetadata{Name: "test-pipe"},
		Steps: []contracts.Step{
			{Name: "step-one", Type: contracts.StepTypePublish},
			{Name: "step-one", Type: contracts.StepTypePublish},
		},
	}
	v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrPipelineDuplicateStepName) {
		t.Errorf("expected error code %s, got %v", ErrPipelineDuplicateStepName, errs)
	}
}

func TestPipelineWorkflowValidator_InvalidStepType(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "PipelineWorkflow",
		Metadata:   contracts.PipelineWorkflowMetadata{Name: "test-pipe"},
		Steps:      []contracts.Step{{Name: "step-one", Type: contracts.StepType("invalid")}},
	}
	v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrPipelineInvalidStepType) {
		t.Errorf("expected error code %s, got %v", ErrPipelineInvalidStepType, errs)
	}
}

func TestPipelineWorkflowValidator_MissingStepFields(t *testing.T) {
	tests := []struct {
		name     string
		step     contracts.Step
		wantCode string
	}{
		{
			name:     "sync missing input",
			step:     contracts.Step{Name: "sync-step", Type: contracts.StepTypeSync, Output: "my-sink"},
			wantCode: ErrPipelineMissingStepField,
		},
		{
			name:     "sync missing output",
			step:     contracts.Step{Name: "sync-step", Type: contracts.StepTypeSync, Input: "my-source"},
			wantCode: ErrPipelineMissingStepField,
		},
		{
			name:     "transform missing asset",
			step:     contracts.Step{Name: "transform-step", Type: contracts.StepTypeTransform},
			wantCode: ErrPipelineMissingStepField,
		},
		{
			name:     "test missing asset",
			step:     contracts.Step{Name: "test-step", Type: contracts.StepTypeTest, Command: []string{"dbt", "test"}},
			wantCode: ErrPipelineMissingStepField,
		},
		{
			name:     "test missing command",
			step:     contracts.Step{Name: "test-step", Type: contracts.StepTypeTest, Asset: "my-model"},
			wantCode: ErrPipelineMissingStepField,
		},
		{
			name:     "custom missing image",
			step:     contracts.Step{Name: "custom-step", Type: contracts.StepTypeCustom},
			wantCode: ErrPipelineCustomMissingImg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw := &contracts.PipelineWorkflow{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "PipelineWorkflow",
				Metadata:   contracts.PipelineWorkflowMetadata{Name: "test-pipe"},
				Steps:      []contracts.Step{tt.step},
			}
			v := NewPipelineWorkflowValidator(pw, "pipeline.yaml")
			errs := v.Validate(context.Background())
			if !hasErrorCode(errs, tt.wantCode) {
				t.Errorf("expected error code %s, got %v", tt.wantCode, errs)
			}
		})
	}
}

func TestPipelineWorkflowValidator_Name(t *testing.T) {
	v := NewPipelineWorkflowValidator(nil, "pipeline.yaml")
	if v.Name() != "pipeline-workflow" {
		t.Errorf("Name() = %q, want %q", v.Name(), "pipeline-workflow")
	}
}

func TestValidateAssetReferences_Valid(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		Steps: []contracts.Step{
			{Name: "sync-step", Type: contracts.StepTypeSync, Input: "my-source", Output: "my-sink"},
			{Name: "transform-step", Type: contracts.StepTypeTransform, Asset: "my-model"},
			{Name: "test-step", Type: contracts.StepTypeTest, Asset: "my-source"},
		},
	}
	assets := []*contracts.AssetManifest{
		{Metadata: contracts.AssetMetadata{Name: "my-source"}},
		{Metadata: contracts.AssetMetadata{Name: "my-sink"}},
		{Metadata: contracts.AssetMetadata{Name: "my-model"}},
	}
	errs := ValidateAssetReferences(pw, assets)
	if errs.HasErrors() {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidateAssetReferences_NotFound(t *testing.T) {
	pw := &contracts.PipelineWorkflow{
		Steps: []contracts.Step{
			{Name: "sync-step", Type: contracts.StepTypeSync, Input: "missing-source", Output: "my-sink"},
		},
	}
	assets := []*contracts.AssetManifest{
		{Metadata: contracts.AssetMetadata{Name: "my-sink"}},
	}
	errs := ValidateAssetReferences(pw, assets)
	if !hasErrorCode(errs, ErrPipelineAssetNotFound) {
		t.Errorf("expected error code %s, got %v", ErrPipelineAssetNotFound, errs)
	}
}
