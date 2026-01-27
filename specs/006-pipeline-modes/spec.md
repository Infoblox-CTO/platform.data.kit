# Feature Specification: Pipeline Execution Modes (Batch vs Streaming)

**Feature Branch**: `006-pipeline-modes`  
**Created**: January 26, 2026  
**Status**: Draft  
**Input**: Analysis of batch vs long-running pipeline requirements for dp tooling

## Overview

This feature introduces explicit pipeline execution modes (`batch` and `streaming`) to properly handle the fundamental differences in lifecycle, health monitoring, deployment strategy, and developer workflow between short-lived batch jobs and long-running streaming services.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Declare Pipeline Mode (Priority: P1)

As a developer, I want to declare whether my pipeline is a batch job or a streaming service so that dp can apply the appropriate execution strategy.

**Why this priority**: This is the foundational capability - all other features depend on knowing the pipeline mode.

**Independent Test**: Run `dp init my-batch --mode=batch` and verify `pipeline.yaml` contains `mode: batch`. Run `dp init my-stream --mode=streaming` and verify `mode: streaming`.

**Acceptance Scenarios**:

1. **Given** I run `dp init my-pipeline --mode=batch`, **When** the command completes, **Then** `pipeline.yaml` contains `spec.mode: batch` with batch-specific defaults (timeout, retries)
2. **Given** I run `dp init my-pipeline --mode=streaming`, **When** the command completes, **Then** `pipeline.yaml` contains `spec.mode: streaming` with streaming-specific defaults (replicas, probes)
3. **Given** I run `dp init my-pipeline` without `--mode`, **When** the command completes, **Then** `pipeline.yaml` defaults to `spec.mode: batch`
4. **Given** a `pipeline.yaml` without mode specified, **When** dp validates or runs it, **Then** mode defaults to `batch` for backward compatibility

---

### User Story 2 - Run Batch Pipeline Locally (Priority: P1)

As a developer, I want `dp run` to wait for completion and report exit status when running a batch pipeline so that I know if the job succeeded or failed.

**Why this priority**: Core execution behavior - batch pipelines must run to completion with clear success/failure indication.

**Independent Test**: Create a batch pipeline that exits after processing, run `dp run`, verify command waits and exits with same code as container.

**Acceptance Scenarios**:

1. **Given** a batch pipeline that exits with code 0, **When** I run `dp run`, **Then** dp waits for completion and exits with code 0
2. **Given** a batch pipeline that exits with code 1, **When** I run `dp run`, **Then** dp waits for completion and exits with code 1
3. **Given** a batch pipeline with timeout of 30s, **When** the pipeline runs longer than 30s, **Then** dp terminates the container and reports timeout error
4. **Given** a batch pipeline, **When** I run `dp run`, **Then** stdout/stderr are streamed to terminal in real-time

---

### User Story 3 - Run Streaming Pipeline Locally (Priority: P1)

As a developer, I want `dp run` to start my streaming pipeline in the background and stream logs so that I can develop and observe its behavior.

**Why this priority**: Core execution behavior - streaming pipelines run indefinitely and need background execution with log visibility.

**Independent Test**: Create a streaming pipeline, run `dp run`, verify container runs in background, logs stream to terminal, Ctrl+C stops gracefully.

**Acceptance Scenarios**:

1. **Given** a streaming pipeline, **When** I run `dp run`, **Then** the container starts in background and logs stream to terminal
2. **Given** a running streaming pipeline, **When** I press Ctrl+C, **Then** the container receives SIGTERM and shuts down gracefully
3. **Given** a streaming pipeline, **When** I run `dp run --detach`, **Then** container starts and dp exits immediately, printing container ID
4. **Given** a detached streaming pipeline, **When** I run `dp logs`, **Then** logs from the running container are displayed

---

### User Story 4 - Test Batch Pipeline (Priority: P1)

As a developer, I want `dp test` to run my batch pipeline with test data and verify it completes successfully so that I can validate my pipeline logic.

**Why this priority**: Essential for CI/CD and local development validation of batch jobs.

**Independent Test**: Create batch pipeline with test data, run `dp test`, verify pipeline processes data and exits with expected results.

**Acceptance Scenarios**:

1. **Given** a batch pipeline with test data in `testdata/`, **When** I run `dp test`, **Then** pipeline runs with test data mounted and exit code determines test result
2. **Given** a batch pipeline that processes 100 records, **When** I run `dp test`, **Then** summary shows records processed and duration
3. **Given** a batch pipeline, **When** test exceeds timeout, **Then** test fails with timeout error

---

### User Story 5 - Test Streaming Pipeline (Priority: P1)

As a developer, I want `dp test` to start my streaming pipeline, verify it becomes healthy, send test events, and validate processing so that I can validate my streaming logic.

**Why this priority**: Streaming pipelines need different test semantics - startup health, continuous processing, graceful shutdown.

**Independent Test**: Create streaming pipeline, run `dp test`, verify startup health check, test events processed, graceful shutdown.

**Acceptance Scenarios**:

1. **Given** a streaming pipeline, **When** I run `dp test`, **Then** dp starts container, waits for health check, sends test events, validates output, shuts down
2. **Given** a streaming pipeline with liveness probe, **When** liveness check fails during test, **Then** test fails with health check error
3. **Given** a streaming pipeline, **When** container fails to become healthy within startup timeout, **Then** test fails with startup timeout error
4. **Given** a streaming pipeline, **When** I run `dp test --duration=60s`, **Then** streaming test runs for 60 seconds before shutdown

---

### User Story 6 - Deploy Batch Pipeline to Kubernetes (Priority: P2)

As a platform operator, I want batch pipelines to be deployed as Kubernetes Jobs/CronJobs so that they run to completion with proper retry handling.

**Why this priority**: Production deployment - batch jobs need Job semantics for completion tracking and retries.

**Independent Test**: Deploy batch pipeline, verify Job resource created, verify completion status tracked, verify retries on failure.

**Acceptance Scenarios**:

1. **Given** a batch pipeline with schedule, **When** deployed to cluster, **Then** controller creates CronJob with specified schedule
2. **Given** a batch pipeline without schedule, **When** triggered via API, **Then** controller creates Job and tracks completion
3. **Given** a batch Job that fails, **When** retries configured, **Then** Job restarts up to retry limit
4. **Given** a batch Job completes, **When** lineage is enabled, **Then** COMPLETE event is emitted with duration and record count

---

### User Story 7 - Deploy Streaming Pipeline to Kubernetes (Priority: P2)

As a platform operator, I want streaming pipelines to be deployed as Kubernetes Deployments so that they run continuously with health monitoring.

**Why this priority**: Production deployment - streaming services need Deployment semantics for continuous availability.

**Independent Test**: Deploy streaming pipeline, verify Deployment created, verify pods restart on failure, verify probes configured.

**Acceptance Scenarios**:

1. **Given** a streaming pipeline, **When** deployed to cluster, **Then** controller creates Deployment with specified replicas
2. **Given** a streaming Deployment, **When** pod crashes, **Then** Kubernetes restarts it automatically
3. **Given** a streaming pipeline with liveness probe, **When** probe fails, **Then** pod is restarted
4. **Given** a streaming pipeline, **When** lineage is enabled, **Then** periodic heartbeat events are emitted

---

### User Story 8 - Lineage Events for Streaming Pipelines (Priority: P3)

As a data governance user, I want streaming pipelines to emit periodic lineage heartbeats so that I can track their continuous operation.

**Why this priority**: Important for governance but not blocking for basic functionality.

**Independent Test**: Run streaming pipeline with lineage enabled, verify heartbeat events emitted at configured interval.

**Acceptance Scenarios**:

1. **Given** a streaming pipeline with lineage enabled, **When** running, **Then** RUNNING heartbeat events are emitted every `heartbeatInterval`
2. **Given** a streaming pipeline, **When** it shuts down gracefully, **Then** COMPLETE event is emitted
3. **Given** a streaming pipeline, **When** it crashes, **Then** FAIL event is emitted with error details

---

## Non-Functional Requirements

### Backward Compatibility

- Pipelines without explicit `mode` default to `batch`
- Existing `dp run` and `dp test` behavior unchanged for batch-mode pipelines
- No breaking changes to `dp.yaml` or `pipeline.yaml` schema

### Performance

- Streaming pipeline health checks complete within 5 seconds
- Lineage heartbeat overhead < 1% CPU for streaming pipelines

---

## Out of Scope

- Auto-scaling for streaming pipelines (use HPA separately)
- Complex DAG orchestration (use Argo Workflows)
- Exactly-once semantics (application responsibility)
- Multi-container sidecar patterns

---

## Technical Notes

### Pipeline Modes

| Aspect | `batch` | `streaming` |
|--------|---------|-------------|
| Lifecycle | Runs to completion | Runs indefinitely |
| K8s Resource | Job / CronJob | Deployment |
| Success Criteria | Exit code 0 | Stays healthy |
| Timeout | Required (default 30m) | Not applicable |
| Restart Policy | OnFailure | Always |
| Health Checks | None | Liveness + Readiness |
| Lineage Events | START → COMPLETE/FAIL | START → RUNNING... → COMPLETE/FAIL |
| `dp run` behavior | Wait for exit | Run detached, stream logs |
| `dp test` behavior | Run with test data, check exit | Start, health check, send events, shutdown |

### Schema Changes

**pipeline.yaml additions**:
```yaml
spec:
  mode: batch | streaming    # New field, defaults to "batch"
  
  # Batch-specific (ignored for streaming)
  timeout: 30m
  retries: 3
  backoffLimit: 6
  
  # Streaming-specific (ignored for batch)  
  replicas: 1
  livenessProbe:
    httpGet:
      path: /healthz
      port: 8080
    initialDelaySeconds: 10
    periodSeconds: 30
  readinessProbe:
    httpGet:
      path: /ready
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 10
  terminationGracePeriodSeconds: 30
  
  # Lineage (both modes)
  lineage:
    enabled: true
    heartbeatInterval: 5m   # Streaming only
```

### Affected Components

1. **contracts/pipeline.go** - Add Mode field, probe structs
2. **cli/cmd/init.go** - Add `--mode` flag
3. **cli/cmd/run.go** - Mode-aware execution
4. **cli/cmd/test.go** - Mode-aware test behavior  
5. **cli/cmd/logs.go** - New command for streaming logs
6. **sdk/runner/docker.go** - Mode-aware container lifecycle
7. **platform/controller/** - Generate Job vs Deployment
8. **sdk/lineage/** - Heartbeat support for streaming
9. **cli/internal/templates/** - Mode-specific templates
