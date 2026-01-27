# Technical Plan: Pipeline Execution Modes

## Tech Stack

- **Language**: Go 1.25
- **CLI Framework**: Cobra
- **Container Runtime**: Docker
- **Kubernetes Resources**: Jobs, CronJobs, Deployments
- **Testing**: Go testing + testify

## Architecture

### Component Changes

```
┌─────────────────────────────────────────────────────────────────┐
│                           CLI Layer                              │
├─────────────────────────────────────────────────────────────────┤
│  init.go     │  run.go      │  test.go     │  logs.go (new)    │
│  --mode flag │  mode-aware  │  mode-aware  │  stream logs      │
│              │  execution   │  testing     │                    │
└──────────────┴──────────────┴──────────────┴────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         SDK Layer                                │
├─────────────────────────────────────────────────────────────────┤
│  contracts/pipeline.go     │  runner/docker.go                  │
│  - PipelineMode enum       │  - RunBatch()                      │
│  - Probe structs           │  - RunStreaming()                  │
│  - Mode-aware validation   │  - mode-aware lifecycle            │
├────────────────────────────┼────────────────────────────────────┤
│  lineage/emitter.go        │  runner/health.go (new)            │
│  - Heartbeat support       │  - Health check polling            │
│  - Streaming events        │  - Probe execution                 │
└────────────────────────────┴────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Controller Layer                            │
├─────────────────────────────────────────────────────────────────┤
│  controller/reconciler.go                                        │
│  - Batch → Job/CronJob                                          │
│  - Streaming → Deployment                                        │
└─────────────────────────────────────────────────────────────────┘
```

## File Structure Changes

```
contracts/
├── pipeline.go          # Add Mode, Probe, PipelineLineage
├── types.go             # Add PipelineMode enum

cli/cmd/
├── init.go              # Add --mode flag
├── run.go               # Mode-aware execution
├── test.go              # Mode-aware testing
├── logs.go              # NEW: Stream container logs

cli/internal/templates/
├── pipeline.batch.yaml.tmpl      # NEW: Batch template
├── pipeline.streaming.yaml.tmpl  # NEW: Streaming template

sdk/runner/
├── docker.go            # Mode-aware Run method
├── health.go            # NEW: Health check implementation
├── batch.go             # NEW: Batch-specific logic
├── streaming.go         # NEW: Streaming-specific logic

sdk/lineage/
├── emitter.go           # Add heartbeat support
├── heartbeat.go         # NEW: Heartbeat goroutine

platform/controller/
├── reconciler.go        # Mode-aware resource generation
├── job.go               # NEW: Job/CronJob generation
├── deployment.go        # NEW: Deployment generation
```

## Implementation Phases

### Phase 1: Contract & Schema (P1)
- Add `PipelineMode` enum to contracts
- Add `Probe` struct for health checks
- Add mode-aware fields to `PipelineSpec`
- Update validation logic
- Backward compatibility: default to batch

### Phase 2: CLI Init (P1)
- Add `--mode` flag to `dp init`
- Create mode-specific templates
- Generate appropriate defaults per mode

### Phase 3: Local Execution (P1)
- Mode detection in runner
- Batch: wait for exit, stream output
- Streaming: detach, stream logs, handle Ctrl+C
- Add `dp logs` command for streaming

### Phase 4: Testing (P1)
- Batch test: run with data, check exit
- Streaming test: start, health check, send events, validate, shutdown
- Add `--duration` flag for streaming tests

### Phase 5: Health Checks (P2)
- Implement probe execution (HTTP, exec, TCP)
- Health polling for streaming startup
- Integrate with test command

### Phase 6: Lineage Heartbeats (P3)
- Add heartbeat goroutine for streaming
- Configure heartbeat interval
- Emit RUNNING events periodically

### Phase 7: Controller (P2)
- Generate Job for batch pipelines
- Generate CronJob for scheduled batch
- Generate Deployment for streaming
- Configure probes from spec

## API Changes

### dp init
```bash
dp init my-pipeline --mode=batch     # Default
dp init my-pipeline --mode=streaming
dp init my-pipeline -m streaming     # Short flag
```

### dp run
```bash
# Batch (unchanged behavior)
dp run                    # Waits for exit

# Streaming
dp run                    # Runs detached, streams logs
dp run --detach           # Runs detached, no logs
dp run --attach           # Explicitly attach to logs
```

### dp test
```bash
# Batch (unchanged behavior)
dp test                   # Run with test data

# Streaming  
dp test                   # Start, health check, send events, shutdown
dp test --duration=60s    # Run for 60s before shutdown
dp test --startup-timeout=30s  # Wait 30s for healthy
```

### dp logs (new)
```bash
dp logs                   # Stream logs from running container
dp logs --follow          # Follow logs (default)
dp logs --tail=100        # Show last 100 lines
dp logs --since=5m        # Logs from last 5 minutes
```

## Testing Strategy

### Unit Tests
- `TestPipelineModeValidation` - enum values
- `TestProbeValidation` - probe struct validation
- `TestBatchDefaults` - timeout, retries defaults
- `TestStreamingDefaults` - replicas, probes defaults
- `TestModeDetection` - backward compatibility

### Integration Tests
- `TestBatchRun` - batch pipeline runs to completion
- `TestBatchTimeout` - timeout kills container
- `TestStreamingRun` - streaming runs in background
- `TestStreamingGracefulShutdown` - SIGTERM handling
- `TestStreamingHealthCheck` - health check polling
- `TestBatchTest` - batch test with data
- `TestStreamingTest` - streaming test lifecycle

### E2E Tests
- Full batch workflow: init → run → test → build
- Full streaming workflow with health checks
- Controller Job generation
- Controller Deployment generation

## Rollout Plan

1. **Phase 1-4**: Core functionality, local development
2. **Phase 5-6**: Enhanced features (probes, heartbeats)
3. **Phase 7**: Production deployment (controller)

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Breaking existing pipelines | Default mode=batch, backward compatible |
| Streaming logs buffer overflow | Use ring buffer, configurable size |
| Health check false positives | Configurable thresholds, sensible defaults |
| Graceful shutdown timeout | Configurable, default 30s, force kill after |
