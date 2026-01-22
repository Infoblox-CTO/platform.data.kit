# Observability and Lineage Patterns for Data Platforms

## Executive Summary

This document captures research on observability and lineage patterns for the CDPP data platform. It covers four key areas: OpenLineage for data lineage, Prometheus metrics, structured logging, and Grafana dashboards.

---

## 1. OpenLineage Standard

### Decision
**Adopt OpenLineage as the standard for data lineage tracking, with Marquez as the lineage backend.**

### How OpenLineage Works

OpenLineage defines a standard JSON schema for describing data pipeline runs, including:

#### Core Event Structure

```json
{
  "eventType": "START|RUNNING|COMPLETE|ABORT|FAIL|OTHER",
  "eventTime": "2024-01-22T10:00:00.000Z",
  "producer": "https://github.com/org/cdpp/v1.0.0",
  "schemaURL": "https://openlineage.io/spec/2-0-2/OpenLineage.json",
  "run": {
    "runId": "d46e465b-d358-4d32-83d4-df660ff614dd",
    "facets": { /* RunFacets */ }
  },
  "job": {
    "namespace": "cdpp-production",
    "name": "my-package.my-pipeline",
    "facets": { /* JobFacets */ }
  },
  "inputs": [{ /* InputDataset */ }],
  "outputs": [{ /* OutputDataset */ }]
}
```

#### Run Lifecycle States
| State | Description |
|-------|-------------|
| `START` | Beginning of a job run (required) |
| `RUNNING` | Additional metadata during execution |
| `COMPLETE` | Successful completion (terminal) |
| `FAIL` | Job failure (terminal) |
| `ABORT` | Job aborted abnormally (terminal) |
| `OTHER` | Additional metadata outside standard cycle |

#### Facet Types

**Run Facets** - Metadata about how the run executed:
- `parent`: Links to parent job/run
- `nominalTime`: Scheduled time vs actual time
- `errorMessage`: Error details on failure

**Job Facets** - Metadata about the job definition:
- `sql`: SQL query being executed
- `sourceCodeLocation`: Git repository/commit
- `documentation`: Job description

**Dataset Facets** - Metadata about inputs/outputs:
- `schema`: Column names and types
- `dataSource`: Connection information
- `dataQualityMetrics`: Row counts, null percentages

### Emitting Events from a Pipeline

```go
// Go example using HTTP transport
type OpenLineageClient struct {
    endpoint string
    producer string
}

func (c *OpenLineageClient) EmitRunEvent(ctx context.Context, event RunEvent) error {
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    req, _ := http.NewRequestWithContext(ctx, "POST", 
        c.endpoint+"/api/v1/lineage", bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := http.DefaultClient.Do(req)
    // handle response...
}
```

### Marquez Integration

**Why Marquez:**
- Reference implementation of OpenLineage backend
- LF AI & Data Foundation graduated project
- Provides Web UI for lineage visualization
- REST API for querying lineage
- PostgreSQL-based storage (operational simplicity)
- GraphQL API for flexible queries
- Compatible with OpenLineage spec 2-0-2

**Deployment:**
```yaml
# docker-compose.yml
services:
  marquez:
    image: marquezproject/marquez:0.50.0
    ports:
      - "5000:5000"  # API
      - "5001:5001"  # Admin
    environment:
      - POSTGRES_HOST=db
      - POSTGRES_DB=marquez
```

### Rationale
- **Industry standard**: OpenLineage is becoming the de-facto standard for data lineage
- **Vendor neutral**: Works with Airflow, Spark, dbt, Dagster integrations
- **Extensible**: Custom facets can capture CDPP-specific metadata
- **Queryable**: Marquez provides API for automating backfills, impact analysis

### Alternatives Considered

| Alternative | Reason Rejected |
|------------|-----------------|
| **Custom lineage schema** | Maintenance burden, no ecosystem support |
| **DataHub** | More complex, heavier infrastructure requirements |
| **Apache Atlas** | Enterprise-focused, complex setup |
| **Egeria** | Broader scope than needed, complex |

---

## 2. Prometheus Metrics for Pipelines

### Decision
**Use Prometheus metrics with carefully designed labels to balance observability with cardinality concerns.**

### Recommended Metrics

#### Pipeline Run Metrics

```go
// Counter for pipeline runs
var (
    pipelineRunsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cdpp_pipeline_runs_total",
            Help: "Total number of pipeline runs",
        },
        []string{"package", "status", "environment"},
    )
    
    // Histogram for duration
    pipelineRunDurationSeconds = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "cdpp_pipeline_run_duration_seconds",
            Help:    "Duration of pipeline runs in seconds",
            Buckets: prometheus.ExponentialBuckets(1, 2, 15), // 1s to ~9h
        },
        []string{"package", "environment"},
    )
    
    // Gauge for in-progress runs
    pipelineRunsInProgress = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "cdpp_pipeline_runs_in_progress",
            Help: "Number of pipeline runs currently in progress",
        },
        []string{"package", "environment"},
    )
    
    // Counter for records processed
    pipelineRecordsProcessedTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cdpp_pipeline_records_processed_total",
            Help: "Total number of records processed by pipelines",
        },
        []string{"package", "environment"},
    )
    
    // Gauge for last successful run timestamp
    pipelineLastSuccessTimestamp = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "cdpp_pipeline_last_success_timestamp_seconds",
            Help: "Unix timestamp of last successful pipeline run",
        },
        []string{"package", "environment"},
    )
)
```

### Naming Conventions

Following Prometheus best practices:

| Pattern | Example | Description |
|---------|---------|-------------|
| `<namespace>_<name>_<unit>` | `cdpp_pipeline_run_duration_seconds` | Include unit suffix |
| `_total` suffix | `cdpp_pipeline_runs_total` | For counters |
| `_info` suffix | `cdpp_build_info` | For pseudo-metrics with metadata |
| `_bytes` / `_seconds` | `cdpp_data_processed_bytes` | Use base units |

### Cardinality Management

**Safe Labels (low cardinality):**
- `package` - Package name (~100s of packages)
- `environment` - `dev`, `staging`, `production`
- `status` - `success`, `failure`, `timeout`
- `stage` - `extract`, `transform`, `load`

**Avoid in Labels:**
- `run_id` - Unbounded cardinality
- `version` - High cardinality over time; use separate info metric
- `user_id` - Unbounded
- `error_message` - Unbounded

**Version Handling Pattern:**
```go
// Use info metric pattern for version
var buildInfo = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "cdpp_package_info",
        Help: "Package metadata as labels",
    },
    []string{"package", "version", "git_sha"},
)

func init() {
    buildInfo.WithLabelValues("my-package", "1.2.3", "abc123").Set(1)
}
```

### Batch Job Considerations

For batch pipelines, use the PushGateway pattern or emit at job completion:

```go
// Record metrics at job end
func (r *Runner) Complete(status string, duration time.Duration) {
    pipelineRunsTotal.WithLabelValues(r.Package, status, r.Env).Inc()
    pipelineRunDurationSeconds.WithLabelValues(r.Package, r.Env).Observe(duration.Seconds())
    
    if status == "success" {
        pipelineLastSuccessTimestamp.WithLabelValues(r.Package, r.Env).SetToCurrentTime()
    }
}
```

### Rationale
- **Prometheus is standard**: Native Kubernetes integration, well-understood
- **Cardinality-conscious design**: Prevents metric explosion
- **Histogram over summary**: Allows aggregation across instances
- **Timestamp pattern**: "Last success" gauge enables freshness alerts

### Alternatives Considered

| Alternative | Reason Rejected |
|------------|-----------------|
| **StatsD** | Less structured, no native histogram support |
| **OpenTelemetry Metrics** | More complex, Prometheus still better for platform metrics |
| **Custom metrics system** | Reinventing the wheel, no ecosystem |

---

## 3. Structured Logging with Correlation IDs

### Decision
**Use Go's standard library `log/slog` with JSON output and consistent correlation ID propagation.**

### Why slog?

| Library | Performance | API Style | Stdlib | Recommendation |
|---------|-------------|-----------|--------|----------------|
| **slog** | Good | Modern, structured | Yes (Go 1.21+) | ✅ Recommended |
| **zerolog** | Excellent (fastest) | Chained | No | Alternative for perf-critical |
| **zap** | Excellent | Two APIs (Logger/Sugar) | No | Alternative for perf-critical |

**Key slog advantages:**
- Part of Go standard library (no dependency)
- Sufficient performance for most use cases
- Clean, idiomatic API
- Built-in JSON handler
- Context propagation support

### Implementation Pattern

```go
package logging

import (
    "context"
    "log/slog"
    "os"
)

type contextKey string
const (
    runIDKey     contextKey = "run_id"
    packageKey   contextKey = "package"
    versionKey   contextKey = "version"
)

// NewLogger creates a configured JSON logger
func NewLogger() *slog.Logger {
    return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level:     slog.LevelInfo,
        AddSource: true, // Include file:line
    }))
}

// WithRunContext adds correlation fields to context
func WithRunContext(ctx context.Context, runID, packageName, version string) context.Context {
    ctx = context.WithValue(ctx, runIDKey, runID)
    ctx = context.WithValue(ctx, packageKey, packageName)
    ctx = context.WithValue(ctx, versionKey, version)
    return ctx
}

// FromContext creates a logger with context fields
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
    logger := base
    
    if runID, ok := ctx.Value(runIDKey).(string); ok {
        logger = logger.With("run_id", runID)
    }
    if pkg, ok := ctx.Value(packageKey).(string); ok {
        logger = logger.With("package", pkg)
    }
    if ver, ok := ctx.Value(versionKey).(string); ok {
        logger = logger.With("version", ver)
    }
    
    return logger
}
```

### Usage Example

```go
func main() {
    logger := logging.NewLogger()
    
    ctx := logging.WithRunContext(context.Background(),
        "run-abc123",
        "my-package",
        "1.2.3",
    )
    
    log := logging.FromContext(ctx, logger)
    
    log.Info("pipeline started")
    // Output: {"time":"2024-01-22T10:00:00Z","level":"INFO","msg":"pipeline started",
    //          "run_id":"run-abc123","package":"my-package","version":"1.2.3"}
    
    log.With("stage", "extract").Info("extracting data", "records", 1000)
    // Output includes all context fields plus stage and records
}
```

### Standard Log Fields

| Field | Type | Description |
|-------|------|-------------|
| `time` | RFC3339 | Timestamp (automatic) |
| `level` | string | INFO, WARN, ERROR, DEBUG |
| `msg` | string | Log message |
| `run_id` | string | Unique run identifier |
| `package` | string | Package name |
| `version` | string | Package version |
| `stage` | string | Pipeline stage (optional) |
| `error` | string | Error message (on errors) |
| `source` | object | File and line (if enabled) |

### Log Aggregation in Kubernetes

**Architecture:**
```
┌─────────────┐     ┌──────────────┐     ┌───────────┐
│ Pod (stdout)│ ──► │ Fluent Bit   │ ──► │ Loki      │
│ JSON logs   │     │ (DaemonSet)  │     │           │
└─────────────┘     └──────────────┘     └───────────┘
                                               │
                                               ▼
                                         ┌───────────┐
                                         │ Grafana   │
                                         │           │
                                         └───────────┘
```

**Fluent Bit Configuration:**
```yaml
# fluent-bit configmap
[INPUT]
    Name              tail
    Path              /var/log/containers/*cdpp*.log
    Parser            cri
    Tag               kube.*

[FILTER]
    Name              kubernetes
    Match             kube.*
    K8S-Logging.Parser On

[OUTPUT]
    Name              loki
    Match             *
    Host              loki.monitoring.svc
    Port              3100
    Labels            job=cdpp, namespace=$kubernetes['namespace_name']
```

**Loki Query Examples:**
```logql
# All logs for a specific run
{job="cdpp"} |= `run-abc123`

# Errors for a package
{job="cdpp"} | json | package="my-package" | level="ERROR"

# Extract duration from logs
{job="cdpp"} | json | line_format "{{.package}}: {{.duration_ms}}ms"
```

### Rationale
- **slog is sufficient**: Performance is good enough, stdlib reduces dependencies
- **JSON format**: Machine-parseable, works with all aggregators
- **Context propagation**: Consistent correlation across async operations
- **Kubernetes-native**: stdout logging works with DaemonSet collectors

### Alternatives Considered

| Alternative | Reason Rejected |
|------------|-----------------|
| **zerolog** | Faster but external dependency; slog is fast enough |
| **zap** | More complex dual-API; slog is simpler |
| **logrus** | Deprecated feel, slower, no compelling advantage |
| **Text logs** | Not machine-parseable, harder to query |

---

## 4. Grafana Dashboards for Data Platforms

### Decision
**Create two dashboard tiers: Platform Dashboard (operators) and Package Dashboard (authors).**

### Platform Dashboard (Operators)

**Purpose:** Overall health of the data platform

**Key Panels:**

| Panel | Visualization | Query | Purpose |
|-------|--------------|-------|---------|
| Total Runs (24h) | Stat | `sum(increase(cdpp_pipeline_runs_total[24h]))` | Volume indicator |
| Success Rate | Gauge | `sum(rate(cdpp_pipeline_runs_total{status="success"}[1h])) / sum(rate(cdpp_pipeline_runs_total[1h]))` | Overall health |
| Active Runs | Stat | `sum(cdpp_pipeline_runs_in_progress)` | Current load |
| Runs by Status | Time series | `sum by (status)(rate(cdpp_pipeline_runs_total[5m]))` | Trend analysis |
| P99 Duration | Time series | `histogram_quantile(0.99, sum by (le)(rate(cdpp_pipeline_run_duration_seconds_bucket[1h])))` | Performance |
| Failed Runs Table | Table | `cdpp_pipeline_runs_total{status="failure"} offset 5m` | Recent failures |
| Top 10 Slowest | Bar gauge | `topk(10, avg by (package)(cdpp_pipeline_run_duration_seconds))` | Bottlenecks |
| Resource Usage | Time series | CPU/memory of runner pods | Capacity planning |

**Dashboard Variables:**
- `environment`: Filter by dev/staging/production
- `time_range`: Standard Grafana time picker

### Package Dashboard (Authors)

**Purpose:** Deep-dive into a specific package's performance

**Key Panels:**

| Panel | Visualization | Query | Purpose |
|-------|--------------|-------|---------|
| Package Info | Stat | `cdpp_package_info{package="$package"}` | Current version |
| Run History | Time series | `cdpp_pipeline_runs_total{package="$package"}` | Run trends |
| Duration Distribution | Heatmap | `cdpp_pipeline_run_duration_seconds_bucket{package="$package"}` | Duration patterns |
| Error Rate | Time series | `rate(cdpp_pipeline_runs_total{package="$package",status="failure"}[5m])` | Stability |
| Records Processed | Time series | `rate(cdpp_pipeline_records_processed_total{package="$package"}[5m])` | Throughput |
| Recent Runs | Table | Logs query for run_id + status + duration | Run details |
| Upstream/Downstream | Node graph | Marquez API or static | Lineage context |
| Last 10 Errors | Logs | `{package="$package"} | json | level="ERROR"` | Debugging |

**Dashboard Variables:**
- `package`: Dropdown from `label_values(cdpp_pipeline_runs_total, package)`
- `environment`: Filter by environment

### Alerting Rules

```yaml
# prometheus-rules.yaml
groups:
  - name: cdpp-alerts
    rules:
      # No successful runs in 2x expected interval
      - alert: CDPPPipelineStale
        expr: |
          time() - cdpp_pipeline_last_success_timestamp_seconds > 7200
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Pipeline {{ $labels.package }} has not succeeded in 2+ hours"
          
      # High failure rate
      - alert: CDPPHighFailureRate
        expr: |
          sum by (package, environment) (
            rate(cdpp_pipeline_runs_total{status="failure"}[15m])
          ) / sum by (package, environment) (
            rate(cdpp_pipeline_runs_total[15m])
          ) > 0.1
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Package {{ $labels.package }} has >10% failure rate"
          
      # Long-running pipeline
      - alert: CDPPLongRunningPipeline
        expr: |
          cdpp_pipeline_runs_in_progress > 0
          and on (package)
          (time() - cdpp_pipeline_run_start_timestamp_seconds) > 3600
        labels:
          severity: warning
        annotations:
          summary: "Pipeline {{ $labels.package }} running for >1 hour"
          
      # No runs at all (pipeline may be broken)
      - alert: CDPPNoRuns
        expr: |
          absent(increase(cdpp_pipeline_runs_total[6h])) == 1
        for: 30m
        labels:
          severity: warning
        annotations:
          summary: "No pipeline runs observed in 6 hours"
```

### Dashboard as Code

```json
{
  "dashboard": {
    "title": "CDPP Platform Overview",
    "uid": "cdpp-platform",
    "tags": ["cdpp", "data-platform"],
    "templating": {
      "list": [
        {
          "name": "environment",
          "type": "custom",
          "options": [
            {"text": "All", "value": ".*"},
            {"text": "Production", "value": "production"},
            {"text": "Staging", "value": "staging"}
          ]
        }
      ]
    }
  }
}
```

### Rationale
- **Two-tier approach**: Different audiences need different views
- **Prometheus + Loki**: Unified stack, correlation between metrics and logs
- **Alert on symptoms**: Focus on freshness and error rate, not internal metrics
- **Variables**: Allow drill-down without multiple dashboards

### Alternatives Considered

| Alternative | Reason Rejected |
|------------|-----------------|
| **Single dashboard** | Too cluttered, different user needs |
| **Per-package dashboards** | Doesn't scale, use variables instead |
| **Datadog/New Relic** | Vendor lock-in, cost, Grafana is sufficient |
| **Custom UI** | Reinventing the wheel |

---

## Summary of Decisions

| Area | Decision | Key Technology |
|------|----------|----------------|
| **Lineage** | OpenLineage + Marquez | OpenLineage spec 2.0, Marquez 0.50+ |
| **Metrics** | Prometheus with cardinality-conscious labels | prometheus/client_golang |
| **Logging** | slog with JSON output, correlation IDs | log/slog (stdlib) |
| **Dashboards** | Two-tier Grafana (platform + package) | Grafana, Loki, Prometheus |

## Implementation Priority

1. **Phase 1**: Structured logging with slog (foundation for debugging)
2. **Phase 2**: Prometheus metrics (enables alerting)
3. **Phase 3**: Platform Grafana dashboard (operational visibility)
4. **Phase 4**: OpenLineage integration (lineage tracking)
5. **Phase 5**: Package dashboards (self-service for authors)

---

## References

- [OpenLineage Spec](https://openlineage.io/docs/spec/facets/)
- [Marquez Project](https://marquezproject.ai/)
- [Prometheus Naming Best Practices](https://prometheus.io/docs/practices/naming/)
- [Prometheus Instrumentation](https://prometheus.io/docs/practices/instrumentation/)
- [Go slog Package](https://pkg.go.dev/log/slog)
- [Grafana Loki](https://grafana.com/docs/loki/latest/)
- [Kubernetes Logging Architecture](https://kubernetes.io/docs/concepts/cluster-administration/logging/)
