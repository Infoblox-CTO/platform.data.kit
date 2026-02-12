package contracts

// DataPackage represents the root manifest (dp.yaml) declaring package identity, type, and contracts.
type DataPackage struct {
	// APIVersion is the schema version (e.g., "dp.io/v1alpha1")
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "DataPackage"
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains package identification information
	Metadata PackageMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the package specification
	Spec DataPackageSpec `json:"spec" yaml:"spec"`
}

// PackageMetadata contains identification information for a package.
type PackageMetadata struct {
	// Name is the package name (DNS-safe, lowercase)
	Name string `json:"name" yaml:"name"`

	// Namespace is the team/org namespace
	Namespace string `json:"namespace" yaml:"namespace"`

	// Version is the package version (semantic versioning)
	Version string `json:"version" yaml:"version"`

	// Labels are key-value labels for filtering
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// DataPackageSpec contains the package specification details.
type DataPackageSpec struct {
	// Type is the package type (currently only "pipeline" is supported).
	Type PackageType `json:"type" yaml:"type"`

	// Description is a human-readable purpose description
	Description string `json:"description" yaml:"description"`

	// Owner is the team or individual owner
	Owner string `json:"owner" yaml:"owner"`

	// Inputs are declared input dependencies
	Inputs []ArtifactContract `json:"inputs,omitempty" yaml:"inputs,omitempty"`

	// Outputs are declared output artifacts (required for pipeline type)
	Outputs []ArtifactContract `json:"outputs,omitempty" yaml:"outputs,omitempty"`

	// Schedule is the scheduling configuration
	Schedule *ScheduleSpec `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Resources specifies CPU/memory requirements
	Resources *ResourceSpec `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Lineage is the lineage tracking configuration
	Lineage *LineageSpec `json:"lineage,omitempty" yaml:"lineage,omitempty"`

	// Runtime is the container runtime configuration (previously in pipeline.yaml)
	Runtime *RuntimeSpec `json:"runtime,omitempty" yaml:"runtime,omitempty"`
}

// RuntimeSpec defines the container runtime configuration for executing a pipeline.
// This section consolidates what was previously defined in pipeline.yaml.
type RuntimeSpec struct {
	// Image is the container image to run (required).
	// Supports ${VAR} substitution from environment variables.
	Image string `json:"image" yaml:"image"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`

	// Args are arguments to pass to the entrypoint.
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`

	// Env contains custom environment variable definitions.
	// Note: Bindings are automatically mapped to env vars.
	Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

	// EnvFrom sources environment variables from secrets or configmaps.
	EnvFrom []EnvFromSource `json:"envFrom,omitempty" yaml:"envFrom,omitempty"`

	// Timeout is the maximum execution time (e.g., "1h", "30m").
	// Defaults to "1h" if not specified.
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Retries is the number of retry attempts on failure.
	// Defaults to 3 if not specified.
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Replicas is the number of parallel instances.
	// Defaults to 1 if not specified.
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// ServiceAccountName is the Kubernetes service account to use.
	ServiceAccountName string `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`

	// SuccessfulJobsHistoryLimit is the number of successful jobs to retain.
	// Defaults to 3 if not specified.
	SuccessfulJobsHistoryLimit int `json:"successfulJobsHistoryLimit,omitempty" yaml:"successfulJobsHistoryLimit,omitempty"`

	// FailedJobsHistoryLimit is the number of failed jobs to retain.
	// Defaults to 5 if not specified.
	FailedJobsHistoryLimit int `json:"failedJobsHistoryLimit,omitempty" yaml:"failedJobsHistoryLimit,omitempty"`
}

// ScheduleSpec defines the scheduling configuration for a package.
type ScheduleSpec struct {
	// Cron is a cron expression for scheduling (e.g., "0 */6 * * *")
	Cron string `json:"cron,omitempty" yaml:"cron,omitempty"`

	// Timezone is the timezone for the cron schedule
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`

	// Suspend indicates if scheduling is suspended
	Suspend bool `json:"suspend,omitempty" yaml:"suspend,omitempty"`
}

// ResourceSpec defines resource requirements for a package.
type ResourceSpec struct {
	// CPU is the CPU request/limit (e.g., "2", "500m")
	CPU string `json:"cpu,omitempty" yaml:"cpu,omitempty"`

	// Memory is the memory request/limit (e.g., "4Gi", "512Mi")
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`

	// EphemeralStorage is the ephemeral storage request/limit
	EphemeralStorage string `json:"ephemeralStorage,omitempty" yaml:"ephemeralStorage,omitempty"`
}

// LineageSpec defines the lineage tracking configuration for a package.
type LineageSpec struct {
	// Enabled indicates if lineage tracking is enabled
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// Emitter is the lineage emitter type (e.g., "marquez", "openlineage")
	Emitter string `json:"emitter,omitempty" yaml:"emitter,omitempty"`

	// Namespace is the lineage namespace
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}
