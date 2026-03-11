package contracts

// Transform is a unit of computation that reads input DataSets and produces output DataSets.
// It carries the runtime, mode, trigger, and timeout — everything about execution.
// Created by the data engineer.
type Transform struct {
	// APIVersion is the schema version (e.g., "datakit.infoblox.dev/v1alpha1").
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

// TransformSpec defines the computation that reads/writes DataSets.
type TransformSpec struct {
	// Runtime is the execution engine: cloudquery, generic-go, generic-python, dbt.
	Runtime Runtime `json:"runtime" yaml:"runtime"`

	// Mode is the execution mode: batch or streaming.
	Mode Mode `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Inputs lists the input DataSet references (data sources for this transform).
	Inputs []DataSetRef `json:"inputs" yaml:"inputs"`

	// Outputs lists the output DataSet references (data produced by this transform).
	Outputs []DataSetRef `json:"outputs" yaml:"outputs"`

	// Image is the container image for generic-go/generic-python/dbt runtimes.
	Image string `json:"image,omitempty" yaml:"image,omitempty"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`

	// Env is a list of environment variables to set.
	Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

	// Trigger defines when this transform executes.
	Trigger *TriggerSpec `json:"trigger,omitempty" yaml:"trigger,omitempty"`

	// Timeout is the maximum execution duration (e.g., "30m", "1h").
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Resources defines CPU/memory requests and limits.
	Resources *ResourceSpec `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Replicas is the number of parallel workers (for streaming mode).
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// Lineage configures lineage event emission.
	Lineage *LineageSpec `json:"lineage,omitempty" yaml:"lineage,omitempty"`

	// ServiceAccountName is the Kubernetes ServiceAccount to use for the
	// Job pod when running as a CloudQuery transform in a k8s cluster.
	// If empty, the namespace's default ServiceAccount is used.
	ServiceAccountName string `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`
}

// DataSetRef is a reference to a named DataSet.
// Exactly one of DataSet (exact name) or Tags (label selector) must be set.
// The DataSet name is resolved at runtime to find the Store and Connector.
type DataSetRef struct {
	// DataSet is the name of the DataSet manifest (local name or OCI ref).
	// Mutually exclusive with Tags.
	DataSet string `json:"dataset,omitempty" yaml:"dataset,omitempty"`

	// Tags matches datasets by their metadata labels.
	// Mutually exclusive with DataSet.
	Tags map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Version is a semver range constraint (e.g., ">=1.0.0 <2.0.0", "^1.2.0").
	// Used with Tags to resolve the best-matching dataset version.
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Cell optionally qualifies which cell's Stores to resolve for this DataSet.
	// When empty, the deployment cell (or package store/ fallback) is used.
	// When set, the Store is resolved from the named cell's namespace.
	// This enables cross-cell transforms (fan-out, fan-in, routing).
	Cell string `json:"cell,omitempty" yaml:"cell,omitempty"`

	// Schema is an APX module ID the transform expects this ref to conform to.
	// Used for consumer-side schema validation without owning the DataSet.
	Schema string `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// TriggerPolicy identifies when a transform should execute.
type TriggerPolicy string

const (
	// TriggerPolicySchedule runs the transform on a cron schedule.
	TriggerPolicySchedule TriggerPolicy = "schedule"

	// TriggerPolicyOnChange runs the transform when any input dataset's data is updated.
	TriggerPolicyOnChange TriggerPolicy = "on-change"

	// TriggerPolicyManual runs the transform only on explicit invocation.
	TriggerPolicyManual TriggerPolicy = "manual"

	// TriggerPolicyComposite combines multiple trigger policies.
	TriggerPolicyComposite TriggerPolicy = "composite"
)

// IsValid checks if the trigger policy is a recognized value.
func (tp TriggerPolicy) IsValid() bool {
	switch tp {
	case TriggerPolicySchedule, TriggerPolicyOnChange, TriggerPolicyManual, TriggerPolicyComposite:
		return true
	}
	return false
}

// TriggerSpec defines when a transform should execute.
type TriggerSpec struct {
	// Policy is the trigger policy: schedule, on-change, manual, or composite.
	Policy TriggerPolicy `json:"policy" yaml:"policy"`

	// Schedule is the cron configuration (required when policy is "schedule").
	Schedule *ScheduleSpec `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Policies lists the sub-policies for composite triggers.
	Policies []TriggerPolicy `json:"policies,omitempty" yaml:"policies,omitempty"`
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
