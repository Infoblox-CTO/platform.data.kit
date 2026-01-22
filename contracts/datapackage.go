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
	// Type is the package type: "pipeline", "model", or "dataset"
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
