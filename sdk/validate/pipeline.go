package validate

import (
	"context"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// Error codes for pipeline validation.
const (
	ErrInvalidPipelineMode  = "E050"
	ErrMissingBatchTimeout  = "E051"
	ErrInvalidProbePort     = "E052"
	ErrInvalidHeartbeat     = "E053"
	ErrStreamingWithTimeout = "E054"
	ErrBatchWithProbes      = "E055"
	ErrInvalidProbeConfig   = "E056"
)

// PipelineValidator validates pipeline.yaml manifests with mode-aware rules.
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
	pipeline, err := manifest.ParsePipelineFile(path)
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

// Pipeline returns the parsed PipelineManifest.
func (v *PipelineValidator) Pipeline() *contracts.PipelineManifest {
	return v.pipeline
}

// Validate validates the PipelineManifest with mode-aware rules.
func (v *PipelineValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.pipeline == nil {
		errs.AddError(ErrMissingRequired, "", "pipeline manifest is nil")
		return errs
	}

	// Validate mode
	v.validateMode(&errs)

	// Get effective mode (default to batch)
	mode := v.pipeline.Spec.Mode.Default()

	// Mode-specific validation
	switch mode {
	case contracts.PipelineModeBatch:
		v.validateBatchMode(&errs)
	case contracts.PipelineModeStreaming:
		v.validateStreamingMode(&errs)
	}

	// Validate probes if present
	if v.pipeline.Spec.LivenessProbe != nil {
		v.validateProbe(&errs, v.pipeline.Spec.LivenessProbe, "spec.livenessProbe")
	}
	if v.pipeline.Spec.ReadinessProbe != nil {
		v.validateProbe(&errs, v.pipeline.Spec.ReadinessProbe, "spec.readinessProbe")
	}

	// Validate lineage configuration
	if v.pipeline.Spec.Lineage != nil {
		v.validateLineage(&errs, mode)
	}

	return errs
}

// validateMode validates the pipeline mode field.
func (v *PipelineValidator) validateMode(errs *contracts.ValidationErrors) {
	if !v.pipeline.Spec.Mode.IsValid() {
		errs.AddError(ErrInvalidPipelineMode, "spec.mode",
			"spec.mode must be 'batch' or 'streaming' (or empty for batch)")
	}
}

// validateBatchMode validates batch-specific configuration.
func (v *PipelineValidator) validateBatchMode(errs *contracts.ValidationErrors) {
	// Batch pipelines should have a timeout (warning if missing, not error)
	// We make timeout optional but recommended
	if v.pipeline.Spec.Timeout == "" {
		errs.AddWarning(ErrMissingBatchTimeout, "spec.timeout",
			"batch pipelines should specify a timeout to prevent indefinite execution")
	} else if !isValidDuration(v.pipeline.Spec.Timeout) {
		errs.AddError(contracts.ErrCodeInvalidTimeout, "spec.timeout",
			"spec.timeout must be a valid duration (e.g., 30m, 1h)")
	}

	// Warn if probes are set for batch (they're ignored)
	if v.pipeline.Spec.LivenessProbe != nil || v.pipeline.Spec.ReadinessProbe != nil {
		errs.AddWarning(ErrBatchWithProbes, "spec",
			"livenessProbe and readinessProbe are ignored for batch pipelines")
	}

	// Validate retries is non-negative
	if v.pipeline.Spec.Retries < 0 {
		errs.AddError(ErrInvalidFormat, "spec.retries", "spec.retries must be non-negative")
	}

	// Validate backoffLimit is non-negative
	if v.pipeline.Spec.BackoffLimit < 0 {
		errs.AddError(ErrInvalidFormat, "spec.backoffLimit", "spec.backoffLimit must be non-negative")
	}
}

// validateStreamingMode validates streaming-specific configuration.
func (v *PipelineValidator) validateStreamingMode(errs *contracts.ValidationErrors) {
	// Streaming pipelines should not have timeout (it doesn't apply)
	if v.pipeline.Spec.Timeout != "" {
		errs.AddWarning(ErrStreamingWithTimeout, "spec.timeout",
			"timeout is ignored for streaming pipelines (they run indefinitely)")
	}

	// Validate replicas is positive (if specified)
	if v.pipeline.Spec.Replicas < 0 {
		errs.AddError(ErrInvalidFormat, "spec.replicas", "spec.replicas must be non-negative")
	}

	// Validate termination grace period
	if v.pipeline.Spec.TerminationGracePeriodSeconds < 0 {
		errs.AddError(ErrInvalidFormat, "spec.terminationGracePeriodSeconds",
			"spec.terminationGracePeriodSeconds must be non-negative")
	}

	// Recommend probes for streaming (warning if missing)
	if v.pipeline.Spec.LivenessProbe == nil && v.pipeline.Spec.ReadinessProbe == nil {
		errs.AddWarning(ErrMissingRequired, "spec",
			"streaming pipelines should define livenessProbe and/or readinessProbe for health monitoring")
	}
}

// validateProbe validates a health check probe configuration.
func (v *PipelineValidator) validateProbe(errs *contracts.ValidationErrors, probe *contracts.Probe, path string) {
	// Must have exactly one probe type
	count := 0
	if probe.HTTPGet != nil {
		count++
	}
	if probe.Exec != nil {
		count++
	}
	if probe.TCPSocket != nil {
		count++
	}

	if count == 0 {
		errs.AddError(ErrInvalidProbeConfig, path,
			"probe must specify one of: httpGet, exec, or tcpSocket")
		return
	}
	if count > 1 {
		errs.AddError(ErrInvalidProbeConfig, path,
			"probe must specify only one of: httpGet, exec, or tcpSocket")
		return
	}

	// Validate HTTP probe
	if probe.HTTPGet != nil {
		if probe.HTTPGet.Port <= 0 || probe.HTTPGet.Port > 65535 {
			errs.AddError(ErrInvalidProbePort, path+".httpGet.port",
				"port must be between 1 and 65535")
		}
		if probe.HTTPGet.Path == "" {
			errs.AddError(ErrMissingRequired, path+".httpGet.path",
				"httpGet.path is required")
		}
	}

	// Validate exec probe
	if probe.Exec != nil {
		if len(probe.Exec.Command) == 0 {
			errs.AddError(ErrMissingRequired, path+".exec.command",
				"exec.command is required and must not be empty")
		}
	}

	// Validate TCP socket probe
	if probe.TCPSocket != nil {
		if probe.TCPSocket.Port <= 0 || probe.TCPSocket.Port > 65535 {
			errs.AddError(ErrInvalidProbePort, path+".tcpSocket.port",
				"port must be between 1 and 65535")
		}
	}

	// Validate timing parameters
	if probe.InitialDelaySeconds < 0 {
		errs.AddError(ErrInvalidFormat, path+".initialDelaySeconds",
			"initialDelaySeconds must be non-negative")
	}
	if probe.PeriodSeconds < 0 {
		errs.AddError(ErrInvalidFormat, path+".periodSeconds",
			"periodSeconds must be non-negative")
	}
	if probe.TimeoutSeconds < 0 {
		errs.AddError(ErrInvalidFormat, path+".timeoutSeconds",
			"timeoutSeconds must be non-negative")
	}
	if probe.SuccessThreshold < 0 {
		errs.AddError(ErrInvalidFormat, path+".successThreshold",
			"successThreshold must be non-negative")
	}
	if probe.FailureThreshold < 0 {
		errs.AddError(ErrInvalidFormat, path+".failureThreshold",
			"failureThreshold must be non-negative")
	}
}

// validateLineage validates lineage configuration with mode awareness.
func (v *PipelineValidator) validateLineage(errs *contracts.ValidationErrors, mode contracts.PipelineMode) {
	lineage := v.pipeline.Spec.Lineage

	// Validate heartbeat interval if specified
	if lineage.HeartbeatInterval != "" {
		if !isValidDuration(lineage.HeartbeatInterval) {
			errs.AddError(ErrInvalidHeartbeat, "spec.lineage.heartbeatInterval",
				"heartbeatInterval must be a valid duration (e.g., 5m, 1h)")
		}

		// Heartbeat only makes sense for streaming pipelines
		if mode == contracts.PipelineModeBatch {
			errs.AddWarning(ErrInvalidHeartbeat, "spec.lineage.heartbeatInterval",
				"heartbeatInterval is ignored for batch pipelines")
		}
	}
}
