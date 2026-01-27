# Pipeline Execution Modes

Data packages in DP support two execution modes that determine how your pipeline runs: **batch** and **streaming**. Understanding the difference is essential for designing pipelines that meet your data processing needs.

## Overview

| Aspect | Batch Mode | Streaming Mode |
|--------|------------|----------------|
| **Execution** | Runs to completion | Runs indefinitely |
| **Use Cases** | ETL jobs, reports, backfills | Real-time processing, event handling |
| **Scheduling** | Cron-based, on-demand | Always running |
| **Kubernetes Resource** | Job / CronJob | Deployment |
| **Scaling** | Sequential runs | Horizontal replicas |
| **Health Checks** | Exit code | Liveness/Readiness probes |

## Batch Mode

Batch pipelines process a finite dataset and exit when complete. This is the default mode for DP packages.

### When to Use Batch Mode

- **ETL Jobs**: Extract, transform, and load data between systems
- **Scheduled Reports**: Generate daily, weekly, or monthly reports
- **Data Backfills**: Reprocess historical data
- **One-time Migrations**: Move data from one format to another
- **Data Validation**: Check data quality on a schedule

### Batch Configuration

```yaml
# pipeline.yaml
apiVersion: dp.io/v1
kind: Pipeline
metadata:
  name: daily-etl
spec:
  mode: batch
  
  # How long before timeout (required for batch)
  timeout: 30m
  
  # Retry failed runs
  retries: 3
  backoffLimit: 3
  
  # Schedule with cron (optional)
  schedule:
    cron: "0 2 * * *"  # Run at 2 AM daily
    timezone: "America/New_York"
```

### Batch Lifecycle

1. **Start**: Pipeline begins processing
2. **Run**: Process input data
3. **Complete**: Exit successfully (code 0) or fail (non-zero)

### Local Development

```bash
# Run batch pipeline locally
dp run

# Run with timeout
dp run --timeout 10m

# Test with sample data
dp test
```

## Streaming Mode

Streaming pipelines run continuously, processing data as it arrives. They never exit under normal operation.

### When to Use Streaming Mode

- **Real-time Processing**: Handle events as they occur
- **Kafka Consumers**: Process messages from Kafka topics
- **API Endpoints**: Serve data via HTTP endpoints
- **Continuous Aggregation**: Maintain running statistics
- **Event-driven Workflows**: React to external triggers

### Streaming Configuration

```yaml
# pipeline.yaml
apiVersion: dp.io/v1
kind: Pipeline
metadata:
  name: kafka-processor
spec:
  mode: streaming
  
  # Number of replicas
  replicas: 3
  
  # Graceful shutdown period
  terminationGracePeriodSeconds: 30
  
  # Health checks
  livenessProbe:
    httpGet:
      path: /healthz
      port: 8080
    initialDelaySeconds: 10
    periodSeconds: 15
    
  readinessProbe:
    httpGet:
      path: /ready
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 10
    
  # Lineage heartbeats
  lineage:
    enabled: true
    heartbeatInterval: 30s
```

### Streaming Lifecycle

1. **Start**: Pipeline begins and initializes
2. **Ready**: Pipeline signals it's ready for traffic
3. **Running**: Continuously processes data, emits heartbeats
4. **Shutdown**: Receives SIGTERM, gracefully stops

### Health Checks

Streaming pipelines must implement health endpoints:

**Liveness Probe** - Is the process running correctly?
- Returns 2xx/3xx if healthy
- Pipeline is restarted if probe fails

**Readiness Probe** - Can the process handle traffic?
- Returns 2xx/3xx when ready
- Traffic is routed away if not ready

### Implementing Health Checks (Go)

```go
package main

import (
    "net/http"
    "sync/atomic"
)

var ready int32

func main() {
    // Health endpoints
    http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })
    
    http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
        if atomic.LoadInt32(&ready) == 1 {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("ready"))
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("not ready"))
        }
    })
    
    go http.ListenAndServe(":8080", nil)
    
    // Initialize and signal ready
    initialize()
    atomic.StoreInt32(&ready, 1)
    
    // Run forever
    runStreamProcessor()
}
```

### Local Development

```bash
# Run streaming pipeline (logs streamed)
dp run

# Run in background
dp run --detach

# Follow logs for detached container
dp logs --follow

# Test streaming behavior
dp test --duration 60s --startup-timeout 30s

# Stop a running pipeline
dp stop <run-id>
```

## Choosing Between Modes

### Use Batch When:

- ✅ Processing has a clear start and end
- ✅ Data volume is bounded
- ✅ Results are needed on a schedule
- ✅ No real-time requirements
- ✅ Failures should be retried automatically

### Use Streaming When:

- ✅ Processing is continuous
- ✅ Low latency is required
- ✅ Events arrive unpredictably
- ✅ Pipeline should always be available
- ✅ Horizontal scaling is needed

## Lineage Events

Both modes emit OpenLineage events for tracking:

| Event | Batch | Streaming |
|-------|-------|-----------|
| START | At run start | At deployment start |
| RUNNING | N/A | Heartbeat at interval |
| COMPLETE | On success | On graceful shutdown |
| FAIL | On error | On crash |

## Migration Between Modes

To change a pipeline's mode:

1. Update `spec.mode` in pipeline.yaml
2. Add/remove mode-specific fields (probes, timeout, etc.)
3. Run `dp build` to update the package
4. Deploy the new version

**Note**: Mode changes in production require redeployment. Streaming→Batch will stop the running Deployment and create a Job. Batch→Streaming will create a Deployment after the current Job completes.

## Best Practices

### Batch Best Practices

- Always set a `timeout` to prevent runaway jobs
- Use `retries` for transient failures
- Log progress for long-running jobs
- Exit with non-zero code on failure

### Streaming Best Practices

- Implement proper signal handling (SIGTERM)
- Use both liveness and readiness probes
- Set `terminationGracePeriodSeconds` appropriately
- Enable lineage heartbeats for monitoring
- Handle backpressure gracefully

## Related Topics

- [Manifests](manifests.md) - Manifest file structure
- [Lineage](lineage.md) - OpenLineage integration
- [Local Development](../tutorials/local-development.md) - Running pipelines locally
