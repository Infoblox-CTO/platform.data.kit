package contracts

import "time"

// Kind identifies the manifest kind.
type Kind string

const (
	// KindConnector is a storage technology type (platform team).
	KindConnector Kind = "Connector"

	// KindStore is a named infrastructure instance with secrets (infra owner).
	KindStore Kind = "Store"

	// KindDataSet is a named data contract with schema and lineage (data engineer).
	KindDataSet Kind = "DataSet"

	// KindDataSetGroup bundles multiple datasets from a single materialisation.
	KindDataSetGroup Kind = "DataSetGroup"

	// KindTransform is a unit of computation that reads/writes datasets (data engineer).
	KindTransform Kind = "Transform"
)

// AllKinds returns all current (non-deprecated) kind values.
func AllKinds() []Kind {
	return []Kind{KindConnector, KindStore, KindDataSet, KindDataSetGroup, KindTransform}
}

// IsValid checks if the kind is a recognized value.
func (k Kind) IsValid() bool {
	switch k {
	case KindConnector, KindStore, KindDataSet, KindDataSetGroup, KindTransform:
		return true
	}
	return false
}

// Runtime identifies how the extension or workload executes.
type Runtime string

const (
	// RuntimeCloudQuery uses the CloudQuery SDK.
	RuntimeCloudQuery Runtime = "cloudquery"

	// RuntimeGenericGo is a generic Go container.
	RuntimeGenericGo Runtime = "generic-go"

	// RuntimeGenericPython is a generic Python container.
	RuntimeGenericPython Runtime = "generic-python"

	// RuntimeDBT uses the dbt transformation engine.
	RuntimeDBT Runtime = "dbt"
)

// IsValid checks if the runtime is a recognized value.
func (r Runtime) IsValid() bool {
	switch r {
	case RuntimeCloudQuery, RuntimeGenericGo, RuntimeGenericPython, RuntimeDBT:
		return true
	}
	return false
}

// IsGeneric returns true for generic-go and generic-python runtimes
// that require a user-provided container image.
func (r Runtime) IsGeneric() bool {
	return r == RuntimeGenericGo || r == RuntimeGenericPython
}

// Mode identifies the execution pattern.
type Mode string

const (
	// ModeBatch is a finite execution that processes data and exits.
	ModeBatch Mode = "batch"

	// ModeStreaming is a long-running process that continuously processes data.
	ModeStreaming Mode = "streaming"
)

// IsValid checks if the mode is a valid value.
func (m Mode) IsValid() bool {
	switch m {
	case ModeBatch, ModeStreaming:
		return true
	}
	return false
}

// Default returns the default mode if empty.
func (m Mode) Default() Mode {
	if m == "" {
		return ModeBatch
	}
	return m
}

// RunStatus represents the status of a pipeline run.
type RunStatus string

const (
	// RunStatusPending means the run is pending.
	RunStatusPending RunStatus = "pending"

	// RunStatusRunning means the run is currently executing.
	RunStatusRunning RunStatus = "running"

	// RunStatusCompleted means the run finished successfully.
	RunStatusCompleted RunStatus = "completed"

	// RunStatusFailed means the run failed.
	RunStatusFailed RunStatus = "failed"

	// RunStatusCancelled means the run was cancelled.
	RunStatusCancelled RunStatus = "cancelled"
)

// RunTrigger represents what triggered a pipeline run.
type RunTrigger string

const (
	// RunTriggerSchedule means the run was triggered by a schedule.
	RunTriggerSchedule RunTrigger = "schedule"

	// RunTriggerEvent means the run was triggered by an event.
	RunTriggerEvent RunTrigger = "event"

	// RunTriggerManual means the run was triggered manually.
	RunTriggerManual RunTrigger = "manual"

	// RunTriggerPromotion means the run was triggered by a promotion.
	RunTriggerPromotion RunTrigger = "promotion"
)

// RunRecord represents a record of a pipeline execution.
type RunRecord struct {
	// ID is the unique identifier for this run.
	ID string `json:"id" yaml:"id"`

	// PackageRef references the package that was run.
	PackageRef ArtifactRef `json:"packageRef" yaml:"packageRef"`

	// Environment is the environment where the run occurred.
	Environment string `json:"environment" yaml:"environment"`

	// Status is the current status of the run.
	Status RunStatus `json:"status" yaml:"status"`

	// Trigger indicates what triggered this run.
	Trigger RunTrigger `json:"trigger" yaml:"trigger"`

	// StartTime is when the run started.
	StartTime time.Time `json:"startTime" yaml:"startTime"`

	// EndTime is when the run completed (if finished).
	EndTime *time.Time `json:"endTime,omitempty" yaml:"endTime,omitempty"`

	// RecordsProcessed is the number of records processed.
	RecordsProcessed int64 `json:"recordsProcessed,omitempty" yaml:"recordsProcessed,omitempty"`

	// ErrorMessage contains error details if the run failed.
	ErrorMessage string `json:"errorMessage,omitempty" yaml:"errorMessage,omitempty"`

	// Metadata contains additional run metadata.
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
