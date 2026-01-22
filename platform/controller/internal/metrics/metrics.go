// Package metrics provides Prometheus metrics for the DP controller.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// PackageDeploymentsTotal is the total number of package deployments.
	PackageDeploymentsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "dp_package_deployments_total",
			Help: "Total number of PackageDeployment resources",
		},
		[]string{"namespace", "phase"},
	)

	// PackageRunsTotal is the total number of package runs.
	PackageRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dp_package_runs_total",
			Help: "Total number of package runs",
		},
		[]string{"namespace", "package", "status"},
	)

	// PackageRunDuration is the duration of package runs.
	PackageRunDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dp_package_run_duration_seconds",
			Help:    "Duration of package runs in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to 512s
		},
		[]string{"namespace", "package"},
	)

	// RecordsProcessed is the number of records processed.
	RecordsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dp_records_processed_total",
			Help: "Total number of records processed",
		},
		[]string{"namespace", "package"},
	)

	// ReconciliationsTotal is the total number of reconciliations.
	ReconciliationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dp_reconciliations_total",
			Help: "Total number of reconciliation loops",
		},
		[]string{"result"},
	)

	// ReconciliationDuration is the duration of reconciliations.
	ReconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dp_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation loops in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
		},
		[]string{},
	)

	// PackagePullDuration is the duration of package pulls.
	PackagePullDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dp_package_pull_duration_seconds",
			Help:    "Duration of package pulls from registry in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 100ms to 51s
		},
		[]string{"registry"},
	)

	// ErrorsTotal is the total number of errors.
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dp_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type"},
	)
)

func init() {
	// Register metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		PackageDeploymentsTotal,
		PackageRunsTotal,
		PackageRunDuration,
		RecordsProcessed,
		ReconciliationsTotal,
		ReconciliationDuration,
		PackagePullDuration,
		ErrorsTotal,
	)
}

// RecordReconciliation records a reconciliation result.
func RecordReconciliation(result string, durationSeconds float64) {
	ReconciliationsTotal.WithLabelValues(result).Inc()
	ReconciliationDuration.WithLabelValues().Observe(durationSeconds)
}

// RecordPackageRun records a package run.
func RecordPackageRun(namespace, pkg, status string, durationSeconds float64, records int64) {
	PackageRunsTotal.WithLabelValues(namespace, pkg, status).Inc()
	PackageRunDuration.WithLabelValues(namespace, pkg).Observe(durationSeconds)
	if records > 0 {
		RecordsProcessed.WithLabelValues(namespace, pkg).Add(float64(records))
	}
}

// RecordPackagePull records a package pull.
func RecordPackagePull(registry string, durationSeconds float64) {
	PackagePullDuration.WithLabelValues(registry).Observe(durationSeconds)
}

// RecordError records an error.
func RecordError(errorType string) {
	ErrorsTotal.WithLabelValues(errorType).Inc()
}

// UpdateDeploymentCount updates the deployment count for a namespace/phase.
func UpdateDeploymentCount(namespace, phase string, count float64) {
	PackageDeploymentsTotal.WithLabelValues(namespace, phase).Set(count)
}
