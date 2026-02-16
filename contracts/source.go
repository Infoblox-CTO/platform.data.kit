package contracts

// Source is a platform extension that ingests data.
// Created by infra engineers. Published to extension registry.
// Referenced by data engineers in Model manifests.
type Source struct {
	// APIVersion is the schema version (e.g., "data.infoblox.com/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Source".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains extension identification information.
	Metadata ExtMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the source specification.
	Spec SourceSpec `json:"spec" yaml:"spec"`
}

// ExtMetadata contains identification information for platform extensions (Source, Destination).
type ExtMetadata struct {
	// Name is the extension name (DNS-safe, lowercase).
	Name string `json:"name" yaml:"name"`

	// Namespace is the platform team namespace.
	Namespace string `json:"namespace" yaml:"namespace"`

	// Version is the extension version (semantic versioning).
	Version string `json:"version" yaml:"version"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// SourceSpec defines what a source extension provides and how it runs.
type SourceSpec struct {
	// Runtime identifies how this extension executes.
	Runtime Runtime `json:"runtime" yaml:"runtime"`

	// Description is a human-readable description.
	Description string `json:"description" yaml:"description"`

	// Owner is the team or individual owner.
	Owner string `json:"owner" yaml:"owner"`

	// Provides describes the output contract — what this source produces.
	Provides ArtifactContract `json:"provides" yaml:"provides"`

	// ConfigSchema defines what configuration knobs data engineers can set.
	ConfigSchema *ConfigSchema `json:"configSchema,omitempty" yaml:"configSchema,omitempty"`

	// Image is the container image for generic runtimes.
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`
}

// ConfigSchema describes what config keys an extension accepts.
// Used by `dp lint` to validate Model.spec.config against extensions.
type ConfigSchema struct {
	// Properties defines the available configuration properties.
	Properties map[string]ConfigProperty `json:"properties" yaml:"properties"`

	// Required lists the required property names.
	Required []string `json:"required,omitempty" yaml:"required,omitempty"`
}

// ConfigProperty describes a single configuration property.
type ConfigProperty struct {
	// Type is the JSON Schema type (string, number, boolean, array, object).
	Type string `json:"type" yaml:"type"`

	// Description is a human-readable description.
	Description string `json:"description" yaml:"description"`

	// Default is the default value.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`

	// Enum restricts the value to a set of allowed values.
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`
}
