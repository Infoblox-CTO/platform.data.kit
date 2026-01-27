# Tasks: Pipeline Execution Modes

## Phase 1: Contracts & Schema

### Setup

- [ ] 1.1 Create feature branch `006-pipeline-modes`

### Core Types

- [X] 1.2 Add `PipelineMode` type to `contracts/types.go`
  - Define `PipelineModeBatch` and `PipelineModeStreaming` constants
  - Add `IsValid()` method for validation

- [X] 1.3 Add `Probe` structs to `contracts/pipeline.go`
  - `Probe` with HTTPGet, Exec, TCPSocket options
  - `HTTPGetAction`, `ExecAction`, `TCPSocketAction`
  - Validation methods for each probe type

- [X] 1.4 Extend `PipelineSpec` in `contracts/pipeline.go`
  - Add `Mode PipelineMode` field with yaml tag
  - Add batch fields: `Timeout`, `Retries`, `BackoffLimit`
  - Add streaming fields: `Replicas`, `LivenessProbe`, `ReadinessProbe`, `TerminationGracePeriodSeconds`
  - Add `Lineage *PipelineLineage` field

- [X] 1.5 Add `PipelineLineage` struct to `contracts/pipeline.go`
  - `Enabled bool`
  - `HeartbeatInterval string` (streaming only)

### Validation

- [X] 1.6 Update `sdk/validate/` for mode-aware validation
  - Validate mode is valid enum value
  - Validate batch requires timeout
  - Validate streaming probe ports in range
  - Validate heartbeat interval is valid duration

### Tests [P]

- [X] 1.7 Add unit tests for `PipelineMode` in `contracts/types_test.go`
- [X] 1.8 Add unit tests for `Probe` validation in `contracts/pipeline_test.go`
- [X] 1.9 Add unit tests for mode-aware validation in `sdk/validate/`

---

## Phase 2: CLI Init

### Templates

- [X] 2.1 Create `cli/internal/templates/pipeline.batch.yaml.tmpl`
  - Include timeout, retries defaults
  - No probes section

- [X] 2.2 Create `cli/internal/templates/pipeline.streaming.yaml.tmpl`
  - Include replicas, livenessProbe, readinessProbe
  - Include terminationGracePeriodSeconds
  - No timeout section

### Init Command

- [X] 2.3 Add `--mode` / `-m` flag to `cli/cmd/init.go`
  - Validate mode value
  - Default to "batch"

- [X] 2.4 Update template selection in `cli/cmd/init.go`
  - Select batch or streaming template based on mode
  - Pass mode to template rendering

### Tests [P]

- [X] 2.5 Add tests for `dp init --mode=batch` in `cli/cmd/init_test.go`
- [X] 2.6 Add tests for `dp init --mode=streaming` in `cli/cmd/init_test.go`

---

## Phase 3: Local Execution (dp run)

### Runner Refactor

- [X] 3.1 Create `sdk/runner/batch.go`
  - Extract batch-specific logic from docker.go
  - `RunBatch()` method that waits for completion
  - Timeout handling with context

- [X] 3.2 Create `sdk/runner/streaming.go`
  - `RunStreaming()` method for detached execution
  - Log streaming goroutine
  - Graceful shutdown with SIGTERM

- [X] 3.3 Update `sdk/runner/docker.go` Run method
  - Detect mode from manifest or options
  - Dispatch to RunBatch or RunStreaming
  - Default to batch for backward compatibility

### CLI Run Command

- [X] 3.4 Update `cli/cmd/run.go` for mode awareness
  - Read mode from pipeline.yaml
  - Pass mode to runner
  - Handle Ctrl+C for streaming (SIGTERM)

- [X] 3.5 Add `--attach` flag to `cli/cmd/run.go`
  - For streaming: explicitly attach to logs
  - Default behavior: attach

- [X] 3.6 Update `--detach` flag behavior for streaming
  - Print container ID and exit
  - Don't stream logs

### Logs Command

- [X] 3.7 Create `cli/cmd/logs.go`
  - `dp logs` command implementation
  - `--follow` flag (default true)
  - `--tail` flag for last N lines
  - `--since` flag for time-based filtering

- [X] 3.8 Add `Logs()` method to `sdk/runner/docker.go`
  - Docker logs command wrapper
  - Follow mode with streaming

### Tests [P]

- [ ] 3.9 Add integration tests for batch run in `cli/cmd/run_test.go`
- [ ] 3.10 Add integration tests for streaming run in `cli/cmd/run_test.go`
- [X] 3.11 Add tests for `dp logs` in `cli/cmd/logs_test.go`

---

## Phase 4: Testing (dp test)

### Batch Testing

- [X] 4.1 Update batch test flow in `cli/cmd/test.go`
  - Ensure test data mounted
  - Wait for completion
  - Report exit code and duration

### Streaming Testing

- [X] 4.2 Implement streaming test flow in `cli/cmd/test.go`
  - Start container
  - Wait for health check (new)
  - Send test events (if configured)
  - Validate outputs (if configured)
  - Graceful shutdown

- [X] 4.3 Add `--duration` flag to test command
  - How long to run streaming test
  - Default 30s

- [X] 4.4 Add `--startup-timeout` flag to test command
  - How long to wait for healthy
  - Default 60s

### Test Runner

- [X] 4.5 Create `sdk/runner/test.go`
  - `TestBatch()` method
  - `TestStreaming()` method with health check loop

### Tests [P]

- [ ] 4.6 Add tests for batch test behavior in `cli/cmd/test_test.go`
- [ ] 4.7 Add tests for streaming test behavior in `cli/cmd/test_test.go`

---

## Phase 5: Health Checks

### Health Check Implementation

- [X] 5.1 Create `sdk/runner/health.go`
  - `CheckHealth()` function
  - HTTP probe implementation
  - Exec probe implementation
  - TCP probe implementation

- [X] 5.2 Implement HTTP health check
  - Make HTTP request to container
  - Check status code (200-399 = healthy)
  - Respect timeout settings

- [X] 5.3 Implement exec health check
  - Docker exec into container
  - Run command, check exit code
  - 0 = healthy

- [X] 5.4 Implement TCP health check
  - TCP connect to container port
  - Connection success = healthy

### Health Polling

- [X] 5.5 Implement health poll loop
  - Poll at `periodSeconds` interval
  - Track consecutive successes/failures
  - Report state changes

### Tests [P]

- [X] 5.6 Add unit tests for HTTP probe in `sdk/runner/health_test.go`
- [X] 5.7 Add unit tests for exec probe in `sdk/runner/health_test.go`
- [X] 5.8 Add unit tests for TCP probe in `sdk/runner/health_test.go`
- [ ] 5.9 Add integration test for health polling

---

## Phase 6: Lineage Heartbeats

### Heartbeat Implementation

- [X] 6.1 Create `sdk/lineage/heartbeat.go`
  - `StartHeartbeat()` function
  - Background goroutine
  - Emit RUNNING events at interval

- [X] 6.2 Add heartbeat integration to streaming runner
  - Start heartbeat on container start
  - Stop heartbeat on shutdown
  - Configure interval from spec

- [X] 6.3 Add RUNNING event type to lineage
  - New event type for heartbeat
  - Include uptime, records processed (if available)

### Tests [P]

- [X] 6.4 Add unit tests for heartbeat in `sdk/lineage/heartbeat_test.go`
- [ ] 6.5 Add integration test for streaming lineage events

---

## Phase 7: Controller

### Job Generation

- [X] 7.1 Create `platform/controller/job.go`
  - Generate Kubernetes Job spec from batch pipeline
  - Configure backoff, retries
  - Set timeout via activeDeadlineSeconds

- [X] 7.2 Create CronJob generation for scheduled batch
  - Generate CronJob from pipeline with schedule
  - Configure schedule from spec

### Deployment Generation

- [X] 7.3 Create `platform/controller/deployment.go`
  - Generate Kubernetes Deployment from streaming pipeline
  - Configure replicas
  - Configure liveness/readiness probes
  - Set terminationGracePeriodSeconds

### Reconciler Updates

- [X] 7.4 Update reconciler for mode-aware generation
  - Detect mode from DataPackage
  - Dispatch to Job or Deployment generator
  - Handle mode changes (delete old, create new)

### Tests [P]

- [X] 7.5 Add tests for Job generation in `platform/controller/job_test.go`
- [X] 7.6 Add tests for Deployment generation in `platform/controller/deployment_test.go`
- [ ] 7.7 Add reconciler tests for mode switching

---

## Documentation

- [X] 8.1 Update `docs/reference/manifest-schema.md` with mode fields
- [X] 8.2 Update `docs/reference/cli.md` with new flags and commands
- [X] 8.3 Create `docs/concepts/pipeline-modes.md` explaining batch vs streaming
- [X] 8.4 Update `docs/tutorials/` with streaming example
