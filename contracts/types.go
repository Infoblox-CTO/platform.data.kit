package contracts

import "time"

// PackageType represents the type of data package.
type PackageType string

const (
	// PackageTypePipeline is a data processing pipeline.
	PackageTypePipeline PackageType = "pipeline"

	// PackageTypeModel is a data model/schema package.
	PackageTypeModel PackageType = "model"

	// PackageTypeDataset is a static dataset package.
	PackageTypeDataset PackageType = "dataset"
)

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

// PipelineMode represents the execution mode for a pipeline.
type PipelineMode string

const (
	// PipelineModeBatch is a batch pipeline that runs to completion.
	PipelineModeBatch PipelineMode = "batch"

	// PipelineModeStreaming is a long-running streaming pipeline.
	PipelineModeStreaming PipelineMode = "streaming"
)

// IsValid checks if the pipeline mode is a valid value.
func (m PipelineMode) IsValid() bool {
	switch m {
	case PipelineModeBatch, PipelineModeStreaming, "":
		return true
	}
	return false
}

// Default returns the default pipeline mode if empty.
func (m PipelineMode) Default() PipelineMode {
	if m == "" {
		return PipelineModeBatch
	}
	return m
}
