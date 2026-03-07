package validate

import (
	"context"
	"regexp"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
)

// Error codes for pipeline workflow validation.
const (
	ErrPipelineMissingRequired   = "E080"
	ErrPipelineInvalidAPIVersion = "E081"
	ErrPipelineInvalidKind       = "E082"
	ErrPipelineEmptySteps        = "E083"
	ErrPipelineInvalidStepName   = "E084"
	ErrPipelineDuplicateStepName = "E085"
	ErrPipelineInvalidStepType   = "E086"
	ErrPipelineMissingStepField  = "E087"
	ErrPipelineAssetNotFound     = "E088"
	ErrPipelineCustomMissingImg  = "E090"
	ErrPipelineInvalidName       = "E091"
)

const (
	pipelineWorkflowAPIVersion = "datakit.infoblox.dev/v1alpha1"
	pipelineWorkflowKind       = "PipelineWorkflow"
)

// dnsNameRegexp matches DNS-safe names: lowercase, starts with letter, 3-63 chars.
var dnsNameRegexp = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]$`)

// PipelineWorkflowValidator validates PipelineWorkflow manifests.
type PipelineWorkflowValidator struct {
	workflow     *contracts.PipelineWorkflow
	workflowPath string
}

// NewPipelineWorkflowValidator creates a validator for a PipelineWorkflow.
func NewPipelineWorkflowValidator(pw *contracts.PipelineWorkflow, path string) *PipelineWorkflowValidator {
	return &PipelineWorkflowValidator{
		workflow:     pw,
		workflowPath: path,
	}
}

// NewPipelineWorkflowValidatorFromFile creates a validator by loading from a file.
func NewPipelineWorkflowValidatorFromFile(path string) (*PipelineWorkflowValidator, error) {
	pw, err := pipeline.LoadPipeline(path)
	if err != nil {
		return nil, err
	}
	return &PipelineWorkflowValidator{
		workflow:     pw,
		workflowPath: path,
	}, nil
}

// Name returns the validator name.
func (v *PipelineWorkflowValidator) Name() string {
	return "pipeline-workflow"
}

// Workflow returns the parsed PipelineWorkflow.
func (v *PipelineWorkflowValidator) Workflow() *contracts.PipelineWorkflow {
	return v.workflow
}

// Validate validates the PipelineWorkflow manifest.
func (v *PipelineWorkflowValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.workflow == nil {
		errs.AddError(ErrPipelineMissingRequired, "", "pipeline workflow manifest is nil")
		return errs
	}

	v.validateRequired(&errs)
	v.validateAPIVersion(&errs)
	v.validateKind(&errs)
	v.validatePipelineName(&errs)
	v.validateSteps(&errs)

	return errs
}

// validateRequired checks for required top-level fields.
func (v *PipelineWorkflowValidator) validateRequired(errs *contracts.ValidationErrors) {
	if v.workflow.APIVersion == "" {
		errs.AddError(ErrPipelineMissingRequired, "apiVersion", "apiVersion is required")
	}
	if v.workflow.Kind == "" {
		errs.AddError(ErrPipelineMissingRequired, "kind", "kind is required")
	}
	if v.workflow.Metadata.Name == "" {
		errs.AddError(ErrPipelineMissingRequired, "metadata.name", "metadata.name is required")
	}
	if len(v.workflow.Steps) == 0 {
		errs.AddError(ErrPipelineEmptySteps, "steps", "at least one step is required")
	}
}

// validateAPIVersion checks the apiVersion value.
func (v *PipelineWorkflowValidator) validateAPIVersion(errs *contracts.ValidationErrors) {
	if v.workflow.APIVersion != "" && v.workflow.APIVersion != pipelineWorkflowAPIVersion {
		errs.AddError(ErrPipelineInvalidAPIVersion, "apiVersion", "apiVersion must be "+pipelineWorkflowAPIVersion)
	}
}

// validateKind checks the kind value.
func (v *PipelineWorkflowValidator) validateKind(errs *contracts.ValidationErrors) {
	if v.workflow.Kind != "" && v.workflow.Kind != pipelineWorkflowKind {
		errs.AddError(ErrPipelineInvalidKind, "kind", "kind must be "+pipelineWorkflowKind)
	}
}

// validatePipelineName checks that the pipeline name is DNS-safe.
func (v *PipelineWorkflowValidator) validatePipelineName(errs *contracts.ValidationErrors) {
	name := v.workflow.Metadata.Name
	if name != "" && !dnsNameRegexp.MatchString(name) {
		errs.AddError(ErrPipelineInvalidName, "metadata.name", "pipeline name must be DNS-safe: lowercase letters, digits, and hyphens, 3-63 characters, starting with a letter")
	}
}

// validateSteps validates all steps in the pipeline.
func (v *PipelineWorkflowValidator) validateSteps(errs *contracts.ValidationErrors) {
	seen := make(map[string]bool)

	for i, step := range v.workflow.Steps {
		v.validateStepName(errs, step, i)
		v.validateStepType(errs, step, i)
		v.validateStepTypeFields(errs, step, i)

		if step.Name != "" {
			if seen[step.Name] {
				errs.AddError(ErrPipelineDuplicateStepName, "steps["+itoa(i)+"].name", "duplicate step name: "+step.Name)
			}
			seen[step.Name] = true
		}
	}
}

// validateStepName checks that a step name is DNS-safe.
func (v *PipelineWorkflowValidator) validateStepName(errs *contracts.ValidationErrors, step contracts.Step, idx int) {
	field := "steps[" + itoa(idx) + "].name"
	if step.Name == "" {
		errs.AddError(ErrPipelineMissingRequired, field, "step name is required")
		return
	}
	if !dnsNameRegexp.MatchString(step.Name) {
		errs.AddError(ErrPipelineInvalidStepName, field, "step name must be DNS-safe: lowercase letters, digits, and hyphens, 3-63 characters, starting with a letter")
	}
}

// validateStepType checks that the step type is valid.
func (v *PipelineWorkflowValidator) validateStepType(errs *contracts.ValidationErrors, step contracts.Step, idx int) {
	field := "steps[" + itoa(idx) + "].type"
	if step.Type == "" {
		errs.AddError(ErrPipelineMissingRequired, field, "step type is required")
		return
	}
	if !step.Type.IsValid() {
		errs.AddError(ErrPipelineInvalidStepType, field, "invalid step type: "+string(step.Type)+"; must be one of: sync, transform, test, publish, custom")
	}
}

// validateStepTypeFields checks type-specific required fields.
func (v *PipelineWorkflowValidator) validateStepTypeFields(errs *contracts.ValidationErrors, step contracts.Step, idx int) {
	prefix := "steps[" + itoa(idx) + "]"
	switch step.Type {
	case contracts.StepTypeSync:
		if step.Input == "" {
			errs.AddError(ErrPipelineMissingStepField, prefix+".input", "input is required for sync step")
		}
		if step.Output == "" {
			errs.AddError(ErrPipelineMissingStepField, prefix+".output", "output is required for sync step")
		}
	case contracts.StepTypeTransform:
		if step.Asset == "" {
			errs.AddError(ErrPipelineMissingStepField, prefix+".asset", "asset is required for transform step")
		}
	case contracts.StepTypeTest:
		if step.Asset == "" {
			errs.AddError(ErrPipelineMissingStepField, prefix+".asset", "asset is required for test step")
		}
		if len(step.Command) == 0 {
			errs.AddError(ErrPipelineMissingStepField, prefix+".command", "command is required for test step")
		}
	case contracts.StepTypeCustom:
		if step.Image == "" {
			errs.AddError(ErrPipelineCustomMissingImg, prefix+".image", "image is required for custom step")
		}
	case contracts.StepTypePublish:
		// No required fields for publish step
	}
}

// ValidateAssetReferences checks that all step asset references resolve to existing assets.
func ValidateAssetReferences(pw *contracts.PipelineWorkflow, assets []*contracts.AssetManifest) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	// Build asset lookup map by metadata.name
	assetMap := make(map[string]*contracts.AssetManifest)
	for _, a := range assets {
		assetMap[a.Metadata.Name] = a
	}

	for i, step := range pw.Steps {
		prefix := "steps[" + itoa(i) + "]"

		switch step.Type {
		case contracts.StepTypeSync:
			if step.Input != "" {
				if _, ok := assetMap[step.Input]; !ok {
					errs.AddError(ErrPipelineAssetNotFound, prefix+".input", "asset not found: "+step.Input)
				}
			}
			if step.Output != "" {
				if _, ok := assetMap[step.Output]; !ok {
					errs.AddError(ErrPipelineAssetNotFound, prefix+".output", "asset not found: "+step.Output)
				}
			}

		case contracts.StepTypeTransform:
			if step.Asset != "" {
				if _, ok := assetMap[step.Asset]; !ok {
					errs.AddError(ErrPipelineAssetNotFound, prefix+".asset", "asset not found: "+step.Asset)
				}
			}

		case contracts.StepTypeTest:
			if step.Asset != "" {
				if _, ok := assetMap[step.Asset]; !ok {
					errs.AddError(ErrPipelineAssetNotFound, prefix+".asset", "asset not found: "+step.Asset)
				}
			}
		}
	}

	return errs
}

// itoa converts an int to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}
