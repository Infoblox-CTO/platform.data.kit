// Package runs provides run tracking and history for DP.
package runs

import (
	"context"
	"time"
)

// StatusAggregator aggregates run status across environments.
type StatusAggregator struct {
	service Service
}

// NewStatusAggregator creates a new StatusAggregator.
func NewStatusAggregator(service Service) *StatusAggregator {
	return &StatusAggregator{
		service: service,
	}
}

// PackageStatus represents the status of a package across environments.
type PackageStatus struct {
	// Package is the package name.
	Package string `json:"package"`
	// Namespace is the team/namespace.
	Namespace string `json:"namespace"`
	// Environments contains status per environment.
	Environments map[string]*EnvironmentStatus `json:"environments"`
	// OverallHealth is the overall health status.
	OverallHealth HealthStatus `json:"overall_health"`
}

// EnvironmentStatus represents the status in a specific environment.
type EnvironmentStatus struct {
	// Environment name.
	Environment string `json:"environment"`
	// CurrentVersion is the deployed version.
	CurrentVersion string `json:"current_version"`
	// LastRun is the most recent run.
	LastRun *RunSummary `json:"last_run,omitempty"`
	// RecentRuns are the most recent runs.
	RecentRuns []*RunSummary `json:"recent_runs,omitempty"`
	// Health is the health status.
	Health HealthStatus `json:"health"`
	// Stats contains run statistics.
	Stats *RunStats `json:"stats,omitempty"`
}

// RunSummary is a summary of a run.
type RunSummary struct {
	// ID is the run ID.
	ID string `json:"id"`
	// Status is the run status.
	Status RunStatus `json:"status"`
	// StartTime is when the run started.
	StartTime time.Time `json:"start_time"`
	// DurationMs is the run duration.
	DurationMs int64 `json:"duration_ms,omitempty"`
	// RecordsProcessed is the number of records processed.
	RecordsProcessed int64 `json:"records_processed,omitempty"`
}

// RunStats contains run statistics.
type RunStats struct {
	// TotalRuns is the total number of runs.
	TotalRuns int `json:"total_runs"`
	// SuccessRuns is the number of successful runs.
	SuccessRuns int `json:"success_runs"`
	// FailureRuns is the number of failed runs.
	FailureRuns int `json:"failure_runs"`
	// SuccessRate is the success rate (0-100).
	SuccessRate float64 `json:"success_rate"`
	// AvgDurationMs is the average duration in milliseconds.
	AvgDurationMs float64 `json:"avg_duration_ms"`
	// TotalRecordsProcessed is the total records processed.
	TotalRecordsProcessed int64 `json:"total_records_processed"`
}

// HealthStatus represents the health of a package or environment.
type HealthStatus string

const (
	// HealthHealthy means everything is working well.
	HealthHealthy HealthStatus = "healthy"
	// HealthDegraded means some issues but mostly working.
	HealthDegraded HealthStatus = "degraded"
	// HealthUnhealthy means there are failures.
	HealthUnhealthy HealthStatus = "unhealthy"
	// HealthUnknown means no runs have been recorded.
	HealthUnknown HealthStatus = "unknown"
)

// GetPackageStatus returns the status of a package across all environments.
func (a *StatusAggregator) GetPackageStatus(ctx context.Context, pkg, namespace string) (*PackageStatus, error) {
	environments := []string{"dev", "int", "prod"}

	status := &PackageStatus{
		Package:      pkg,
		Namespace:    namespace,
		Environments: make(map[string]*EnvironmentStatus),
	}

	healthyCount := 0
	unhealthyCount := 0
	unknownCount := 0

	for _, env := range environments {
		envStatus, err := a.getEnvironmentStatus(ctx, pkg, env)
		if err != nil {
			return nil, err
		}
		status.Environments[env] = envStatus

		switch envStatus.Health {
		case HealthHealthy:
			healthyCount++
		case HealthUnhealthy:
			unhealthyCount++
		case HealthUnknown:
			unknownCount++
		}
	}

	// Calculate overall health
	if unhealthyCount > 0 {
		status.OverallHealth = HealthUnhealthy
	} else if unknownCount == len(environments) {
		status.OverallHealth = HealthUnknown
	} else if healthyCount == len(environments)-unknownCount {
		status.OverallHealth = HealthHealthy
	} else {
		status.OverallHealth = HealthDegraded
	}

	return status, nil
}

// getEnvironmentStatus returns the status for a specific environment.
func (a *StatusAggregator) getEnvironmentStatus(ctx context.Context, pkg, env string) (*EnvironmentStatus, error) {
	filter := &RunFilter{
		Package:     pkg,
		Environment: env,
	}

	// Get recent runs
	runs, _, err := a.service.ListRuns(ctx, filter, 10, 0)
	if err != nil {
		return nil, err
	}

	envStatus := &EnvironmentStatus{
		Environment: env,
		Health:      HealthUnknown,
	}

	if len(runs) == 0 {
		return envStatus, nil
	}

	// Get last run
	lastRun := runs[0]
	envStatus.LastRun = &RunSummary{
		ID:               lastRun.ID,
		Status:           lastRun.Status,
		StartTime:        lastRun.StartTime,
		DurationMs:       lastRun.DurationMs,
		RecordsProcessed: lastRun.RecordsProcessed,
	}
	envStatus.CurrentVersion = lastRun.Version

	// Build recent runs summary
	envStatus.RecentRuns = make([]*RunSummary, 0, len(runs))
	for _, run := range runs {
		envStatus.RecentRuns = append(envStatus.RecentRuns, &RunSummary{
			ID:               run.ID,
			Status:           run.Status,
			StartTime:        run.StartTime,
			DurationMs:       run.DurationMs,
			RecordsProcessed: run.RecordsProcessed,
		})
	}

	// Calculate stats
	envStatus.Stats = a.calculateStats(runs)

	// Determine health based on recent runs
	envStatus.Health = a.calculateHealth(runs)

	return envStatus, nil
}

// calculateStats calculates run statistics.
func (a *StatusAggregator) calculateStats(runs []*RunRecord) *RunStats {
	if len(runs) == 0 {
		return nil
	}

	stats := &RunStats{}
	var totalDuration int64

	for _, run := range runs {
		stats.TotalRuns++
		switch run.Status {
		case StatusSuccess:
			stats.SuccessRuns++
		case StatusFailure:
			stats.FailureRuns++
		}
		totalDuration += run.DurationMs
		stats.TotalRecordsProcessed += run.RecordsProcessed
	}

	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(stats.SuccessRuns) / float64(stats.TotalRuns) * 100
		stats.AvgDurationMs = float64(totalDuration) / float64(stats.TotalRuns)
	}

	return stats
}

// calculateHealth determines health based on recent runs.
func (a *StatusAggregator) calculateHealth(runs []*RunRecord) HealthStatus {
	if len(runs) == 0 {
		return HealthUnknown
	}

	// Check last 5 runs
	checkRuns := runs
	if len(checkRuns) > 5 {
		checkRuns = checkRuns[:5]
	}

	failures := 0
	for _, run := range checkRuns {
		if run.Status == StatusFailure {
			failures++
		}
	}

	// If last run failed, at minimum degraded
	if checkRuns[0].Status == StatusFailure {
		if failures >= 3 {
			return HealthUnhealthy
		}
		return HealthDegraded
	}

	// If more than half failed, degraded
	if failures > len(checkRuns)/2 {
		return HealthDegraded
	}

	return HealthHealthy
}

// GetAllPackagesStatus returns status for all packages.
func (a *StatusAggregator) GetAllPackagesStatus(ctx context.Context) ([]*PackageStatus, error) {
	// Get all unique packages from runs
	// For MVP, we'd need to query distinct packages
	// This is a simplified implementation
	return nil, nil
}
