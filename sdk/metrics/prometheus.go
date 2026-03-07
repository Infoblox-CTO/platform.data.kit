// Package metrics provides observability metrics for DataKit.
package metrics

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics implements Metrics using Prometheus.
type PrometheusMetrics struct {
	registry *prometheus.Registry

	// Counters
	runsTotal    *prometheus.CounterVec
	recordsTotal *prometheus.CounterVec
	bytesTotal   *prometheus.CounterVec
	errorsTotal  *prometheus.CounterVec

	// Histograms
	runDuration *prometheus.HistogramVec

	// Gauges
	runsInProgress *prometheus.GaugeVec
}

// NewPrometheusMetrics creates a new PrometheusMetrics.
func NewPrometheusMetrics() *PrometheusMetrics {
	registry := prometheus.NewRegistry()

	m := &PrometheusMetrics{
		registry: registry,
		runsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dp_runs_total",
				Help: "Total number of pipeline runs",
			},
			[]string{"package", "namespace", "environment", "status"},
		),
		recordsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dp_records_processed_total",
				Help: "Total number of records processed",
			},
			[]string{"package", "namespace", "environment"},
		),
		bytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dp_bytes_processed_total",
				Help: "Total number of bytes processed",
			},
			[]string{"package", "namespace", "environment"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dp_errors_total",
				Help: "Total number of errors",
			},
			[]string{"package", "namespace", "environment", "type"},
		),
		runDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "dp_run_duration_seconds",
				Help:    "Duration of pipeline runs in seconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s to 68m
			},
			[]string{"package", "namespace", "environment", "status"},
		),
		runsInProgress: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "dp_runs_in_progress",
				Help: "Number of runs currently in progress",
			},
			[]string{"package", "namespace", "environment"},
		),
	}

	// Register all metrics
	registry.MustRegister(
		m.runsTotal,
		m.recordsTotal,
		m.bytesTotal,
		m.errorsTotal,
		m.runDuration,
		m.runsInProgress,
	)

	return m
}

// RecordRunStart implements Metrics.
func (m *PrometheusMetrics) RecordRunStart(ctx context.Context, run *RunInfo) {
	m.runsInProgress.WithLabelValues(
		run.Package,
		run.Namespace,
		run.Environment,
	).Inc()
}

// RecordRunComplete implements Metrics.
func (m *PrometheusMetrics) RecordRunComplete(ctx context.Context, run *RunInfo, result *RunResult) {
	labels := []string{
		run.Package,
		run.Namespace,
		run.Environment,
		string(result.Status),
	}

	m.runsTotal.WithLabelValues(labels...).Inc()
	m.runDuration.WithLabelValues(labels...).Observe(result.Duration.Seconds())

	if result.RecordsProcessed > 0 {
		m.recordsTotal.WithLabelValues(
			run.Package,
			run.Namespace,
			run.Environment,
		).Add(float64(result.RecordsProcessed))
	}

	if result.BytesProcessed > 0 {
		m.bytesTotal.WithLabelValues(
			run.Package,
			run.Namespace,
			run.Environment,
		).Add(float64(result.BytesProcessed))
	}

	m.runsInProgress.WithLabelValues(
		run.Package,
		run.Namespace,
		run.Environment,
	).Dec()
}

// RecordRunError implements Metrics.
func (m *PrometheusMetrics) RecordRunError(ctx context.Context, run *RunInfo, err error) {
	m.errorsTotal.WithLabelValues(
		run.Package,
		run.Namespace,
		run.Environment,
		"runtime",
	).Inc()
}

// Close implements Metrics.
func (m *PrometheusMetrics) Close() error {
	return nil
}

// Handler returns an HTTP handler for the metrics endpoint.
func (m *PrometheusMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Registry returns the Prometheus registry.
func (m *PrometheusMetrics) Registry() *prometheus.Registry {
	return m.registry
}

// Ensure PrometheusMetrics implements Metrics.
var _ Metrics = (*PrometheusMetrics)(nil)
