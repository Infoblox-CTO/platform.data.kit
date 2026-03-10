# Building a Streaming Pipeline

This tutorial walks you through building a streaming data pipeline that continuously processes events from Kafka and writes aggregated data to S3.

## Prerequisites

- [dk CLI installed](../getting-started/installation.md)
- [Local development stack running](../getting-started/quickstart.md)
- Basic understanding of [pipeline modes](../concepts/pipeline-modes.md)

## Overview

We'll build a real-time event aggregator that:

1. Consumes user events from Kafka
2. Aggregates events by user ID (1-minute windows)
3. Writes aggregations to S3 as Parquet files
4. Exposes health endpoints for monitoring

## Step 1: Initialize the Project

Create a new streaming Transform:

```bash
dk init user-aggregator --runtime generic-go
cd user-aggregator
```

This creates a project structure:

```
user-aggregator/
├── dk.yaml                 # Transform manifest
├── connector/              # Connector definitions
├── store/                  # Store definitions
├── asset/                  # DataSet definitions
├── src/
│   └── main.go            # Pipeline code
└── schemas/
    └── event.avsc         # Input schema
```

## Step 2: Configure the Transform

Edit `dk.yaml` to configure streaming behavior:

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: user-aggregator
  namespace: analytics
  version: 1.0.0
spec:
  runtime: generic-go
  mode: streaming
  image: myorg/user-aggregator:v1.0.0

  inputs:
    - dataset: user-events

  outputs:
    - dataset: user-aggregations

  # Horizontal scaling
  replicas: 3
  resources:
    cpu: "500m"
    memory: "1Gi"

  # Lineage tracking
  lineage:
    enabled: true
    heartbeatInterval: 30s
```

## Step 3: Define DataSets and Stores

Create the input DataSet:

```yaml title="asset/user-events.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: user-events
  namespace: analytics
spec:
  store: events-kafka
  topic: user-activity
  format: avro
```

Create the output DataSet:

```yaml title="asset/user-aggregations.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: user-aggregations
  namespace: analytics
spec:
  store: analytics-s3
  prefix: aggregations/users/
  format: parquet
  classification: internal
```

Create the Stores:

```yaml title="store/events-kafka.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: events-kafka
spec:
  connector: kafka
  connection:
    bootstrap-servers: kafka:9092
    consumer-group: user-aggregator
    auto-offset-reset: earliest
```

```yaml title="store/analytics-s3.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: analytics-s3
spec:
  connector: s3
  connection:
    endpoint: http://minio:9000
    region: us-east-1
    partition-by: date
  secrets:
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
```

## Step 4: Implement the Pipeline

Edit `src/main.go`:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "sync/atomic"
    "syscall"
    "time"
)

// Health state
var (
    ready   int32
    healthy int32 = 1
)

// Aggregation state
var (
    recordsProcessed int64
    mu               sync.Mutex
    aggregations     = make(map[string]int64)
)

func main() {
    // Setup health endpoints
    http.HandleFunc("/healthz", healthzHandler)
    http.HandleFunc("/ready", readyHandler)
    go http.ListenAndServe(":8080", nil)

    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

    go func() {
        <-sigCh
        log.Println("Shutdown signal received, draining...")
        atomic.StoreInt32(&ready, 0) // Stop accepting traffic
        cancel()
    }()

    // Initialize Kafka consumer
    consumer, err := initConsumer()
    if err != nil {
        log.Fatalf("Failed to init consumer: %v", err)
    }
    defer consumer.Close()

    // Initialize S3 writer
    writer, err := initS3Writer()
    if err != nil {
        log.Fatalf("Failed to init S3 writer: %v", err)
    }
    defer writer.Close()

    // Start aggregation flusher (every minute)
    go flushAggregations(ctx, writer)

    // Signal ready
    atomic.StoreInt32(&ready, 1)
    log.Println("Pipeline ready, consuming events...")

    // Process events until shutdown
    for {
        select {
        case <-ctx.Done():
            log.Println("Flushing final aggregations...")
            flushNow(writer)
            log.Println("Shutdown complete")
            return
        default:
            msg, err := consumer.ReadMessage(time.Second)
            if err != nil {
                continue // Timeout, try again
            }
            processEvent(msg)
        }
    }
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
    if atomic.LoadInt32(&healthy) == 1 {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("unhealthy"))
    }
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
    if atomic.LoadInt32(&ready) == 1 {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ready"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("not ready"))
    }
}

func processEvent(msg []byte) {
    // Parse event and aggregate
    userID := extractUserID(msg)
    
    mu.Lock()
    aggregations[userID]++
    mu.Unlock()
    
    atomic.AddInt64(&recordsProcessed, 1)
}

func flushAggregations(ctx context.Context, writer *S3Writer) {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            flushNow(writer)
        case <-ctx.Done():
            return
        }
    }
}

func flushNow(writer *S3Writer) {
    mu.Lock()
    toFlush := aggregations
    aggregations = make(map[string]int64)
    mu.Unlock()

    if len(toFlush) > 0 {
        if err := writer.WriteAggregations(toFlush); err != nil {
            log.Printf("Failed to write aggregations: %v", err)
            atomic.StoreInt32(&healthy, 0)
        }
    }
}

// Placeholder implementations - replace with real code
func initConsumer() (*KafkaConsumer, error) { return &KafkaConsumer{}, nil }
func initS3Writer() (*S3Writer, error)      { return &S3Writer{}, nil }
func extractUserID(msg []byte) string       { return "user-1" }

type KafkaConsumer struct{}
func (c *KafkaConsumer) ReadMessage(timeout time.Duration) ([]byte, error) { 
    time.Sleep(timeout)
    return nil, nil 
}
func (c *KafkaConsumer) Close() error { return nil }

type S3Writer struct{}
func (w *S3Writer) WriteAggregations(data map[string]int64) error { return nil }
func (w *S3Writer) Close() error { return nil }
```

## Step 5: Test Locally

Start the local development stack:

```bash
dk dev up
```

Run the streaming pipeline test:

```bash
# Test for 60 seconds
dk test --duration 60s
```

The test will:
1. Build and start the container
2. Wait for the health check to pass
3. Run for 60 seconds
4. Send SIGTERM for graceful shutdown
5. Report results

For interactive development:

```bash
# Run attached (see logs, Ctrl+C to stop)
dk run

# Or run detached
dk run --detach
dk logs --follow
```

## Step 6: Monitor Health

While the pipeline is running, you can check health:

```bash
# From another terminal
curl http://localhost:8080/healthz
# Response: ok

curl http://localhost:8080/ready  
# Response: ready
```

## Step 7: Build and Deploy

When ready for deployment:

```bash
# Build the OCI artifact
dk build

# Publish to registry
dk publish

# Promote to development
dk promote --to dev
```

In Kubernetes, the controller will create a Deployment with:
- 3 replicas (as configured)
- Liveness and readiness probes
- Rolling update strategy
- Graceful termination

## Best Practices for Streaming Pipelines

### 1. Handle Signals Properly

Always handle SIGTERM for graceful shutdown:

```go
signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
```

### 2. Separate Liveness and Readiness

- **Liveness**: "Is the process running?" (restart if fails)
- **Readiness**: "Can I handle traffic?" (route traffic away if fails)

### 3. Set Appropriate Timeouts

```yaml
terminationGracePeriodSeconds: 30  # Time to drain
livenessProbe:
  initialDelaySeconds: 10          # Time to start
  periodSeconds: 15                # Check interval
```

### 4. Enable Lineage Heartbeats

Track uptime and processing metrics:

```yaml
lineage:
  enabled: true
  heartbeatInterval: 30s
```

### 5. Handle Backpressure

If you can't keep up with input:
- Buffer in memory (with limits)
- Set readiness to false
- Scale horizontally

## Next Steps

- [Pipeline Modes Concept](../concepts/pipeline-modes.md) - Deep dive on batch vs streaming
- [Health Checks](../concepts/governance.md) - Health check best practices
- [Lineage](../concepts/lineage.md) - OpenLineage integration
- [Local Development](local-development.md) - Development workflow
