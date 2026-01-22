package validate

import (
	"context"
	"os"
	"strconv"

	"github.com/Infoblox-CTO/data-platform/contracts"
	"github.com/Infoblox-CTO/data-platform/sdk/manifest"
)

// PipelineValidator validates pipeline.yaml manifests.
type PipelineValidator struct {
	pipeline     *contracts.PipelineManifest
	pipelinePath string
}

// NewPipelineValidator creates a validator for a PipelineManifest.
func NewPipelineValidator(pipeline *contracts.PipelineManifest, path string) *PipelineValidator {
	return &PipelineValidator{
		pipeline:     pipeline,
		pipelinePath: path,
	}
}

// NewPipelineValidatorFromFile creates a validator from a pipeline.yaml file.
func NewPipelineValidatorFromFile(path string) (*PipelineValidator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := manifest.NewParser()
	pipeline, err := parser.ParsePipeline(data)
	if err != nil {
		return nil, err
	}

	return &PipelineValidator{
		pipeline:     pipeline,
		pipelinePath: path,
	}, nil
}

// Name returns the validator name.
func (v *PipelineValidator) Name() string {
	return "pipeline"
}

// Validate validates the PipelineManifest.
func (v *PipelineValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.pipeline == nil {
		errs.AddError(ErrMissingRequired, "", "pipeline manifest is nil")
		return errs
	}

	// Validate required fields
	v.validateRequiredFields(&errs)

	// Validate runtime configuration
	v.validateRuntime(&errs)

	// Validate environment variables
	v.validateEnv(&errs)

	// Validate binding references
	v.validateBindings(&errs)

	return errs
}

// validateRequiredFields checks for required fields.
func (v *PipelineValidator) validateRequiredFields(errs *contracts.ValidationErrors) {
	if v.pipeline.APIVersion == "" {
		errs.AddError(ErrMissingRequired, "apiVersion", "apiVersion is required")
	}

	if v.pipeline.Kind == "" {
		errs.AddError(ErrMissingRequired, "kind", "kind is required")
	} else if v.pipeline.Kind != "Pipeline" {
		errs.AddError(ErrInvalidFormat, "kind", "kind must be 'Pipeline'")
	}

	if v.pipeline.Metadata.Name == "" {
		errs.AddError(ErrMissingRequired, "metadata.name", "metadata.name is required")
	}
}

// validateRuntime validates the runtime configuration.
func (v *PipelineValidator) validateRuntime(errs *contracts.ValidationErrors) {
	spec := v.pipeline.Spec

	if spec.Image == "" {
		errs.AddError(ErrMissingRequired, "spec.image", "spec.image is required")
	} else if !isPipelineImageRefValid(spec.Image) {
		errs.AddError(contracts.ErrCodeInvalidImageRef, "spec.image", "invalid container image reference format")
	}

	// Validate replicas if specified
	if spec.Replicas < 0 {
		errs.AddError(ErrInvalidFormat, "spec.replicas", "replicas must be non-negative")
	}
}

// validateEnv validates environment variable configurations.
func (v *PipelineValidator) validateEnv(errs *contracts.ValidationErrors) {
	for i, env := range v.pipeline.Spec.Env {
		idx := strconv.Itoa(i)

		if env.Name == "" {
			errs.AddError(ErrMissingRequired, "spec.env["+idx+"].name", "env var name is required")
		}

		// Check that exactly one source is specified
		sources := 0
		if env.Value != "" {
			sources++
		}
		if env.ValueFrom != nil {
			sources++
		}

		if sources == 0 {
			errs.AddError(ErrMissingRequired, "spec.env["+idx+"]", "env var must have value or valueFrom")
		}
		if sources > 1 {
			errs.AddError(ErrInvalidFormat, "spec.env["+idx+"]", "env var cannot have both value and valueFrom")
		}
	}
}

// validateBindings validates binding references.
func (v *PipelineValidator) validateBindings(errs *contracts.ValidationErrors) {
	for i, binding := range v.pipeline.Spec.Bindings {
		idx := strconv.Itoa(i)

		if binding.Name == "" {
			errs.AddError(ErrMissingRequired, "spec.bindings["+idx+"].name", "binding name is required")
		}
		if binding.Ref == "" {
			errs.AddError(ErrMissingRequired, "spec.bindings["+idx+"].ref", "binding ref is required")
		}
	}
}

// isPipelineImageRefValid performs basic validation of container image references.
func isPipelineImageRefValid(image string) bool {
	if image == "" {
		return false
	}

	// Image can be:
	// - name
	// - name:tag
	// - registry/name
	// - registry/name:tag
	// - registry:port/name:tag
	for _, c := range image {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '/' || c == ':' || c == '@') {
			return false
		}
	}

	return true
}
