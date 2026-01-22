// Package runs provides run tracking and history for DP.
package runs

import (
	"context"
	"time"
)

// RunRecord represents a record of a pipeline run.
type RunRecord struct {
	// ID is the unique identifier for this run.
	ID string `json:"id"`
	// Package is the package name.
	Package string `json:"package"`
	// Namespace is the team/namespace.
	Namespace string `json:"namespace"`
	// Version is the package version.
	Version string `json:"version"`
	// Environment is the deployment environment.
	Environment string `json:"environment"`
	// Status is the run status.
	Status RunStatus `json:"status"`
	// StartTime is when the run started.
	StartTime time.Time `json:"start_time"`
	// EndTime is when the run ended.
	EndTime *time.Time `json:"end_time,omitempty"`
	// Duration is the run duration in milliseconds.
	DurationMs int64 `json:"duration_ms,omitempty"`
	// RecordsProcessed is the number of records processed.
	RecordsProcessed int64 `json:"records_processed,omitempty"`
	// BytesProcessed is the number of bytes processed.
	BytesProcessed int64 `json:"bytes_processed,omitempty"`
	// ErrorMessage is the error message if failed.
	ErrorMessage string `json:"error_message,omitempty"`
	// Metadata contains additional run metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RunStatus represents the status of a run.
type RunStatus string

const (
	// StatusPending means the run is pending.
	StatusPending RunStatus = "pending"
	// StatusRunning means the run is in progress.
	StatusRunning RunStatus = "running"
	// StatusSuccess means the run completed successfully.
	StatusSuccess RunStatus = "success"
	// StatusFailure means the run failed.
	StatusFailure RunStatus = "failure"
	// StatusCancelled means the run was cancelled.
	StatusCancelled RunStatus = "cancelled"
)

// Service provides run tracking operations.
type Service interface {
	// StartRun creates a new run record and returns the run ID.
	StartRun(ctx context.Context, run *RunRecord) (string, error)
	// UpdateRun updates an existing run record.
	UpdateRun(ctx context.Context, run *RunRecord) error
	// CompleteRun marks a run as complete.
	CompleteRun(ctx context.Context, id string, status RunStatus, endTime time.Time, records, bytes int64, errMsg string) error
	// GetRun returns a run by ID.
	GetRun(ctx context.Context, id string) (*RunRecord, error)
	// ListRuns lists runs with optional filters.
	ListRuns(ctx context.Context, filter *RunFilter, limit, offset int) ([]*RunRecord, int, error)
	// GetLatestRun returns the latest run for a package/environment.
	GetLatestRun(ctx context.Context, pkg, env string) (*RunRecord, error)
}

// RunFilter defines filters for listing runs.
type RunFilter struct {
	// Package filters by package name.
	Package string
	// Namespace filters by namespace.
	Namespace string
	// Environment filters by environment.
	Environment string
	// Status filters by status.
	Status RunStatus
	// Since filters runs after this time.
	Since *time.Time
	// Until filters runs before this time.
	Until *time.Time
}

// DefaultService is the default implementation of Service.
type DefaultService struct {
	store Store
}

// NewService creates a new DefaultService.
func NewService(store Store) *DefaultService {
	return &DefaultService{
		store: store,
	}
}

// StartRun implements Service.
func (s *DefaultService) StartRun(ctx context.Context, run *RunRecord) (string, error) {
	if run.ID == "" {
		run.ID = generateRunID()
	}
	run.Status = StatusRunning
	run.StartTime = time.Now().UTC()

	if err := s.store.Create(ctx, run); err != nil {
		return "", err
	}

	return run.ID, nil
}

// UpdateRun implements Service.
func (s *DefaultService) UpdateRun(ctx context.Context, run *RunRecord) error {
	return s.store.Update(ctx, run)
}

// CompleteRun implements Service.
func (s *DefaultService) CompleteRun(ctx context.Context, id string, status RunStatus, endTime time.Time, records, bytes int64, errMsg string) error {
	run, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}

	run.Status = status
	run.EndTime = &endTime
	run.DurationMs = endTime.Sub(run.StartTime).Milliseconds()
	run.RecordsProcessed = records
	run.BytesProcessed = bytes
	run.ErrorMessage = errMsg

	return s.store.Update(ctx, run)
}

// GetRun implements Service.
func (s *DefaultService) GetRun(ctx context.Context, id string) (*RunRecord, error) {
	return s.store.Get(ctx, id)
}

// ListRuns implements Service.
func (s *DefaultService) ListRuns(ctx context.Context, filter *RunFilter, limit, offset int) ([]*RunRecord, int, error) {
	return s.store.List(ctx, filter, limit, offset)
}

// GetLatestRun implements Service.
func (s *DefaultService) GetLatestRun(ctx context.Context, pkg, env string) (*RunRecord, error) {
	filter := &RunFilter{
		Package:     pkg,
		Environment: env,
	}
	runs, _, err := s.store.List(ctx, filter, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	return runs[0], nil
}

// generateRunID generates a unique run ID.
func generateRunID() string {
	return time.Now().UTC().Format("20060102-150405") + "-" + randomSuffix(6)
}

// randomSuffix generates a random alphanumeric suffix.
func randomSuffix(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
