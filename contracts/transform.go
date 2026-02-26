package contracts

// Transform is a unit of computation that reads input Assets and produces output Assets.
// It carries the runtime, mode, schedule, and timeout — everything about execution.
// Created by the data engineer.
type Transform struct {
	// APIVersion is the schema version (e.g., "data.infoblox.com/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Transform".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains transform identification information.
	Metadata TransformMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the transform specification.
	Spec TransformSpec `json:"spec" yaml:"spec"`
}

// TransformMetadata contains identification information for a Transform.
type TransformMetadata struct {
	// Name is the transform name (e.g., "pg-to-s3", "enrich-users").
	Name string `json:"name" yaml:"name"`

	// Namespace is the team namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Version is the semantic version (e.g., "0.1.0").
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// TransformSpec defines the computation that reads/writes Assets.
type TransformSpec struct {
	// Runtime is the execution engine: cloudquery, generic-go, generic-python, dbt.
	Runtime Runtime `json:"runtime" yaml:"runtime"`

	// Mode is the execution mode: batch or streaming.
	Mode Mode `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Inputs lists the input Asset references (data sources for this transform).
	Inputs []AssetRef `json:"inputs" yaml:"inputs"`

	// Outputs lists the output Asset references (data produced by this transform).
	Outputs []AssetRef `json:"outputs" yaml:"outputs"`

	// Image is the container image for generic-go/generic-python/dbt runtimes.
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`

	// Env is a list of environment variables to set.
	Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

	// Schedule defines optional cron scheduling for batch transforms.
	Schedule *ScheduleSpec `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Timeout is the maximum execution duration (e.g., "30m", "1h").
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Resources defines CPU/memory requests and limits.
	Resources *ResourceSpec `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Replicas is the number of parallel workers (for streaming mode).
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// Lineage configures lineage event emission.
	Lineage *LineageSpec `json:"lineage,omitempty" yaml:"lineage,omitempty"`
}

// AssetRef is a reference to a named Asset.
// The Asset name is resolved at runtime to find the Store and Connector.
type AssetRef struct {
	// Asset is the name of the Asset manifest (local name or OCI ref).
	Asset string `json:"asset" yaml:"asset"`

	// Cell optionally qualifies which cell's Stores to resolve for this Asset.
	// When empty, the deployment cell (or package store/ fallback) is used.
	// When set, the Store is resolved from the named cell's namespace.
	// This enables cross-cell transforms (fan-out, fan-in, routing).
	Cell string `json:"cell,omitempty" yaml:"cell,omitempty"`
}

// --- Manifest interface implementation for Transform ---

// GetKind returns the manifest kind.
func (t *Transform) GetKind() Kind { return KindTransform }

// GetName returns the transform name.
func (t *Transform) GetName() string { return t.Metadata.Name }

// GetNamespace returns the transform namespace.
func (t *Transform) GetNamespace() string { return t.Metadata.Namespace }

// GetVersion returns the transform version.
func (t *Transform) GetVersion() string { return t.Metadata.Version }

// GetDescription returns an empty string.
func (t *Transform) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (t *Transform) GetOwner() string { return "" }
