// Package metrics provides observability metrics for DP.
package metrics

import (
	"context"
	"time"
)

// Metrics defines the interface for recording pipeline metrics.
type Metrics interface {
	// RecordRunStart records the start of a pipeline run.
	RecordRunStart(ctx context.Context, run *RunInfo)
	// RecordRunComplete records the completion of a pipeline run.
	RecordRunComplete(ctx context.Context, run *RunInfo, result *RunResult)
	// RecordRunError records an error during a pipeline run.
	RecordRunError(ctx context.Context, run *RunInfo, err error)
	// Close closes the metrics reporter.
	Close() error
}

// RunInfo contains information about a pipeline run.
type RunInfo struct {
	// RunID is the unique identifier for this run.
	RunID string
	// Package is the package name.
	Package string
	// Namespace is the team/namespace.
	Namespace string
	// Version is the package version.
	Version string
	// Environment is the deployment environment.
	Environment string
	// StartTime is when the run started.
	StartTime time.Time
}

// RunResult contains the result of a pipeline run.
type RunResult struct {
	// Status is the run status (success, failure, cancelled).
	Status RunStatus
	// EndTime is when the run ended.
	EndTime time.Time
	// Duration is the run duration.
	Duration time.Duration
	// RecordsProcessed is the number of records processed.
	RecordsProcessed int64
	// BytesProcessed is the number of bytes processed.
	BytesProcessed int64
	// ErrorMessage is the error message if failed.
	ErrorMessage string
}

// RunStatus represents the status of a run.
type RunStatus string

const (
	// StatusSuccess means the run completed successfully.
	StatusSuccess RunStatus = "success"
	// StatusFailure means the run failed.
	StatusFailure RunStatus = "failure"
	// StatusCancelled means the run was cancelled.
	StatusCancelled RunStatus = "cancelled"
)

// Labels contains metric labels.
type Labels struct {
	Package     string
	Namespace   string
	Version     string
	Environment string
	Status      string
}

// NoopMetrics is a no-op implementation of Metrics.
type NoopMetrics struct{}

// NewNoopMetrics creates a new NoopMetrics.
func NewNoopMetrics() *NoopMetrics {
	return &NoopMetrics{}
}

// RecordRunStart implements Metrics.
func (n *NoopMetrics) RecordRunStart(ctx context.Context, run *RunInfo) {}

// RecordRunComplete implements Metrics.
func (n *NoopMetrics) RecordRunComplete(ctx context.Context, run *RunInfo, result *RunResult) {}

// RecordRunError implements Metrics.
func (n *NoopMetrics) RecordRunError(ctx context.Context, run *RunInfo, err error) {}

// Close implements Metrics.
func (n *NoopMetrics) Close() error { return nil }

// Ensure NoopMetrics implements Metrics.
var _ Metrics = (*NoopMetrics)(nil)
