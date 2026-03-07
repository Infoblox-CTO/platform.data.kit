package contracts

// ScheduleManifest represents a pipeline schedule defined in schedule.yaml.
type ScheduleManifest struct {
	// APIVersion is the schema version (e.g., "datakit.infoblox.dev/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Schedule".
	Kind string `json:"kind" yaml:"kind"`

	// Cron is a standard 5-field cron expression (e.g., "0 */6 * * *").
	Cron string `json:"cron" yaml:"cron"`

	// Timezone is an IANA timezone identifier (e.g., "UTC", "America/New_York").
	// Defaults to UTC when empty.
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`

	// Suspend indicates whether the schedule is suspended.
	// When true, the schedule is registered but not active.
	Suspend bool `json:"suspend,omitempty" yaml:"suspend,omitempty"`
}
