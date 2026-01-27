# Data Model: Pipeline Execution Modes

## Entities

### PipelineMode (Enum)

Defines the execution mode for a pipeline.

```go
type PipelineMode string

const (
    PipelineModeBatch     PipelineMode = "batch"
    PipelineModeStreaming PipelineMode = "streaming"
)
```

### PipelineSpec (Extended)

Extended pipeline specification with mode-aware fields.

```go
type PipelineSpec struct {
    // Existing fields...
    Image              string            `yaml:"image"`
    Command            []string          `yaml:"command,omitempty"`
    Args               []string          `yaml:"args,omitempty"`
    Env                []EnvVar          `yaml:"env,omitempty"`
    
    // New: Execution mode
    Mode               PipelineMode      `yaml:"mode,omitempty"`  // defaults to "batch"
    
    // Batch-specific fields
    Timeout            string            `yaml:"timeout,omitempty"`      // e.g., "30m", "1h"
    Retries            int               `yaml:"retries,omitempty"`      // max retry attempts
    BackoffLimit       int               `yaml:"backoffLimit,omitempty"` // k8s Job backoff
    
    // Streaming-specific fields
    Replicas           int               `yaml:"replicas,omitempty"`     // deployment replicas
    LivenessProbe      *Probe            `yaml:"livenessProbe,omitempty"`
    ReadinessProbe     *Probe            `yaml:"readinessProbe,omitempty"`
    TerminationGracePeriodSeconds int    `yaml:"terminationGracePeriodSeconds,omitempty"`
    
    // Lineage configuration
    Lineage            *PipelineLineage  `yaml:"lineage,omitempty"`
}
```

### Probe

Health check probe configuration (Kubernetes-compatible).

```go
type Probe struct {
    HTTPGet             *HTTPGetAction `yaml:"httpGet,omitempty"`
    Exec                *ExecAction    `yaml:"exec,omitempty"`
    TCPSocket           *TCPSocketAction `yaml:"tcpSocket,omitempty"`
    InitialDelaySeconds int            `yaml:"initialDelaySeconds,omitempty"`
    PeriodSeconds       int            `yaml:"periodSeconds,omitempty"`
    TimeoutSeconds      int            `yaml:"timeoutSeconds,omitempty"`
    SuccessThreshold    int            `yaml:"successThreshold,omitempty"`
    FailureThreshold    int            `yaml:"failureThreshold,omitempty"`
}

type HTTPGetAction struct {
    Path   string `yaml:"path"`
    Port   int    `yaml:"port"`
    Scheme string `yaml:"scheme,omitempty"` // HTTP or HTTPS
}

type ExecAction struct {
    Command []string `yaml:"command"`
}

type TCPSocketAction struct {
    Port int `yaml:"port"`
}
```

### PipelineLineage

Lineage configuration for pipelines.

```go
type PipelineLineage struct {
    Enabled           bool   `yaml:"enabled,omitempty"`
    HeartbeatInterval string `yaml:"heartbeatInterval,omitempty"` // streaming only, e.g., "5m"
}
```

### RunOptions (Extended)

Extended runner options for mode-aware execution.

```go
type RunOptions struct {
    // Existing fields...
    PackageDir   string
    Env          map[string]string
    Network      string
    Timeout      time.Duration
    DryRun       bool
    Detach       bool
    Output       io.Writer
    
    // New: Mode-aware fields
    Mode                  PipelineMode
    StreamLogs            bool          // for streaming: attach to logs
    GracefulShutdownWait  time.Duration // for streaming: SIGTERM wait
}
```

### TestOptions (Extended)

Extended test options for mode-aware testing.

```go
type TestOptions struct {
    // Existing fields...
    PackageDir   string
    TestData     string
    Timeout      time.Duration
    
    // New: Streaming test fields
    Mode              PipelineMode
    StartupTimeout    time.Duration // time to wait for healthy
    TestDuration      time.Duration // how long to run streaming test
    TestEvents        []TestEvent   // events to send during test
    ExpectedOutputs   []string      // expected output patterns
}

type TestEvent struct {
    Topic   string
    Key     string
    Payload []byte
    Delay   time.Duration // delay before sending
}
```

## Relationships

```
PipelineManifest
    └── PipelineSpec
            ├── Mode: PipelineMode (batch | streaming)
            ├── [batch] Timeout, Retries, BackoffLimit
            ├── [streaming] Replicas, LivenessProbe, ReadinessProbe
            └── Lineage: PipelineLineage
                    └── [streaming] HeartbeatInterval
```

## State Transitions

### Batch Pipeline States

```
PENDING → RUNNING → COMPLETED
                  → FAILED (exit != 0)
                  → TIMEOUT (exceeded timeout)
```

### Streaming Pipeline States

```
PENDING → STARTING → HEALTHY → RUNNING ←→ UNHEALTHY
                                      → TERMINATING → COMPLETED
                                      → FAILED (crash)
```

## Validation Rules

1. **Mode**: Must be "batch" or "streaming", defaults to "batch"
2. **Batch timeout**: Required for batch, must be parseable duration
3. **Streaming replicas**: Must be >= 1, defaults to 1
4. **Probes**: If specified, must have valid port (1-65535)
5. **Heartbeat interval**: Only valid for streaming mode
6. **Backward compatibility**: Missing mode field = batch
