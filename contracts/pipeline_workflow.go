package contracts

// StepType enumerates the supported pipeline step types.
type StepType string

const (
	// StepTypeSync is an input-to-output data movement step.
	StepTypeSync StepType = "sync"

	// StepTypeTransform is a transform engine execution step (e.g., dbt run).
	StepTypeTransform StepType = "transform"

	// StepTypeTest is a validation/assertion step (e.g., dbt test).
	StepTypeTest StepType = "test"

	// StepTypePublish is a notification and optional promotion step.
	StepTypePublish StepType = "publish"

	// StepTypeCustom is a single container execution step (backward compat).
	StepTypeCustom StepType = "custom"
)

// ValidStepTypes returns all valid step type values.
func ValidStepTypes() []StepType {
	return []StepType{
		StepTypeSync, StepTypeTransform, StepTypeTest, StepTypePublish, StepTypeCustom,
	}
}

// IsValid reports whether s is a recognized step type.
func (s StepType) IsValid() bool {
	for _, v := range ValidStepTypes() {
		if s == v {
			return true
		}
	}
	return false
}

// PipelineWorkflow represents a multi-step pipeline workflow defined in pipeline.yaml.
type PipelineWorkflow struct {
	// APIVersion is the schema version (e.g., "data.infoblox.com/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "PipelineWorkflow".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata holds the pipeline's identity and description.
	Metadata PipelineWorkflowMetadata `json:"metadata" yaml:"metadata"`

	// Steps is the ordered sequence of pipeline steps.
	Steps []Step `json:"steps" yaml:"steps"`
}

// PipelineWorkflowMetadata holds the pipeline's identity and description.
type PipelineWorkflowMetadata struct {
	// Name is the pipeline name (DNS-safe, lowercase, 3–63 chars).
	Name string `json:"name" yaml:"name"`

	// Description is an optional human-readable description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Step represents a single unit of work within a pipeline workflow.
// Fields are conditionally required based on the step Type (see StepType).
type Step struct {
	// Common fields (all step types)

	// Name is the unique step identifier (DNS-safe, lowercase, 3–63 chars).
	Name string `json:"name" yaml:"name"`

	// Type is the step type discriminator.
	Type StepType `json:"type" yaml:"type"`

	// Description is an optional human-readable step description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Sync step fields

	// Input is the name of the input asset (sync step only).
	Input string `json:"input,omitempty" yaml:"input,omitempty"`

	// Output is the name of the output asset (sync step only).
	Output string `json:"output,omitempty" yaml:"output,omitempty"`

	// Transform / Test step fields

	// Asset is the name of the referenced asset (transform and test steps).
	Asset string `json:"asset,omitempty" yaml:"asset,omitempty"`

	// Command is the command to execute (test step only).
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`

	// Custom step fields

	// Image is the container image (custom step only).
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Args is the container arguments (custom step only).
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`

	// Shared optional fields

	// Env is additional environment variables for this step.
	Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

	// Publish step fields

	// Notify is the notification configuration (publish step only).
	Notify *NotifyConfig `json:"notify,omitempty" yaml:"notify,omitempty"`

	// Promote indicates whether to trigger environment promotion (publish step only).
	Promote bool `json:"promote,omitempty" yaml:"promote,omitempty"`
}

// NotifyConfig configures notifications for a publish step.
type NotifyConfig struct {
	// Channels is the list of notification channels (e.g., Slack channels).
	Channels []string `json:"channels,omitempty" yaml:"channels,omitempty"`

	// Recipients is the list of notification recipients (e.g., email addresses).
	Recipients []string `json:"recipients,omitempty" yaml:"recipients,omitempty"`
}

// StepStatus represents the execution status of a single pipeline step.
type StepStatus string

const (
	// StepStatusPending indicates the step has not started.
	StepStatusPending StepStatus = "pending"

	// StepStatusRunning indicates the step is currently executing.
	StepStatusRunning StepStatus = "running"

	// StepStatusCompleted indicates the step completed successfully.
	StepStatusCompleted StepStatus = "completed"

	// StepStatusFailed indicates the step failed.
	StepStatusFailed StepStatus = "failed"

	// StepStatusSkipped indicates the step was skipped (prior step failed or cancelled).
	StepStatusSkipped StepStatus = "skipped"
)

// StepResult captures the outcome of executing a single pipeline step.
type StepResult struct {
	// Name is the step name.
	Name string `json:"name"`

	// Type is the step type.
	Type StepType `json:"type"`

	// Status is the execution status.
	Status StepStatus `json:"status"`

	// Duration is the step execution duration (e.g., "2.5s").
	Duration string `json:"duration,omitempty"`

	// Error is the error message if the step failed.
	Error string `json:"error,omitempty"`
}

// PipelineRunResult captures the outcome of executing a full pipeline.
type PipelineRunResult struct {
	// PipelineName is the name of the pipeline that was executed.
	PipelineName string `json:"pipelineName"`

	// Status is the overall execution status (completed, failed, cancelled).
	Status StepStatus `json:"status"`

	// Steps is the list of step results.
	Steps []StepResult `json:"steps"`

	// Duration is the total pipeline execution duration.
	Duration string `json:"duration"`

	// FailedStep is the name of the step that caused failure (empty if no failure).
	FailedStep string `json:"failedStep,omitempty"`
}
