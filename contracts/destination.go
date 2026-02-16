package contracts

// Destination is a platform extension that writes data.
// Created by infra engineers. Published to extension registry.
// Referenced by data engineers in Model manifests.
type Destination struct {
	// APIVersion is the schema version (e.g., "data.infoblox.com/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Destination".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains extension identification information.
	Metadata ExtMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the destination specification.
	Spec DestinationSpec `json:"spec" yaml:"spec"`
}

// DestinationSpec defines what a destination extension accepts and how it runs.
type DestinationSpec struct {
	// Runtime identifies how this extension executes.
	Runtime Runtime `json:"runtime" yaml:"runtime"`

	// Description is a human-readable description.
	Description string `json:"description" yaml:"description"`

	// Owner is the team or individual owner.
	Owner string `json:"owner" yaml:"owner"`

	// Accepts describes the input contract — what this destination accepts.
	Accepts ArtifactContract `json:"accepts" yaml:"accepts"`

	// ConfigSchema defines what configuration knobs data engineers can set.
	ConfigSchema *ConfigSchema `json:"configSchema,omitempty" yaml:"configSchema,omitempty"`

	// Image is the container image for generic runtimes.
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`
}
