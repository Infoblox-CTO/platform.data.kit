package contracts

// Model is a data workload that moves and/or transforms data
// using platform-provided sources and destinations.
// Created by data engineers.
type Model struct {
	// APIVersion is the schema version (e.g., "data.infoblox.com/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Model".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains model identification information.
	Metadata ModelMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the model specification.
	Spec ModelSpec `json:"spec" yaml:"spec"`
}

// ModelMetadata contains identification information for a Model.
type ModelMetadata struct {
	// Name is the model name (DNS-safe, lowercase).
	Name string `json:"name" yaml:"name"`

	// Namespace is the team namespace.
	Namespace string `json:"namespace" yaml:"namespace"`

	// Version is the model version (semantic versioning).
	Version string `json:"version" yaml:"version"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// ModelSpec contains the model specification details.
type ModelSpec struct {
	// Runtime identifies how this workload executes.
	Runtime Runtime `json:"runtime" yaml:"runtime"`

	// Mode is the execution pattern: batch or streaming.
	Mode Mode `json:"mode" yaml:"mode"`

	// Description is a human-readable description.
	Description string `json:"description" yaml:"description"`

	// Owner is the team or individual owner.
	Owner string `json:"owner" yaml:"owner"`

	// Source references a published Source extension (optional for generic runtimes).
	Source *ExtensionRef `json:"source,omitempty" yaml:"source,omitempty"`

	// Destination references a published Destination extension (optional for generic runtimes).
	Destination *ExtensionRef `json:"destination,omitempty" yaml:"destination,omitempty"`

	// Inputs are declared input dependencies for lineage and governance.
	Inputs []ArtifactContract `json:"inputs,omitempty" yaml:"inputs,omitempty"`

	// Outputs are declared output artifacts (required).
	Outputs []ArtifactContract `json:"outputs,omitempty" yaml:"outputs,omitempty"`

	// Config is runtime-specific configuration validated against extension configSchemas.
	Config map[string]any `json:"config,omitempty" yaml:"config,omitempty"`

	// Schedule is the scheduling configuration.
	Schedule *ScheduleSpec `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Resources specifies CPU/memory requirements.
	Resources *ResourceSpec `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Timeout is the maximum execution time (e.g., "1h", "30m").
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Retries is the number of retry attempts on failure.
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Replicas is the number of parallel instances.
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// Image is the container image (for generic runtimes or advanced use).
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`

	// Env contains custom environment variable definitions.
	Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

	// Lineage is the lineage tracking configuration.
	Lineage *LineageSpec `json:"lineage,omitempty" yaml:"lineage,omitempty"`
}

// ExtensionRef points to a published Source or Destination extension.
type ExtensionRef struct {
	// Name is the extension name (e.g., "postgres-cdc").
	Name string `json:"name" yaml:"name"`

	// Namespace is the extension namespace (e.g., "platform").
	Namespace string `json:"namespace" yaml:"namespace"`

	// Version is the extension version or constraint (e.g., "1.2.0", ">=1.0.0").
	Version string `json:"version" yaml:"version"`
}
