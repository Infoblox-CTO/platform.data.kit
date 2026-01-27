# Research: k3d Local Development Environment

**Feature**: 005-k3d-local-dev  
**Date**: January 25, 2026  
**Status**: Complete

## Research Tasks

### 1. k3d CLI Integration Patterns

**Decision**: Use `os/exec` to invoke k3d CLI commands directly

**Rationale**: 
- k3d has no official Go SDK; CLI is the stable interface
- Matches existing pattern in `ComposeManager` which uses `docker compose` CLI
- k3d CLI is well-documented with predictable JSON output (`--output json`)

**Alternatives Considered**:
- k3d Go library (internal): Unstable, not intended for external use
- Docker API directly: Would bypass k3d's cluster management benefits

**Implementation Notes**:
```go
// k3d cluster create dp-local --config <embedded-config>
// k3d cluster start dp-local
// k3d cluster stop dp-local
// k3d cluster delete dp-local
// k3d cluster list --output json
```

### 2. Kubernetes Manifest Embedding

**Decision**: Use Go 1.16+ `embed` package to bundle manifests in the binary

**Rationale**:
- Eliminates external file dependencies (FR-011)
- Works from any directory (User Story 5)
- Manifests version-controlled with CLI

**Alternatives Considered**:
- External manifest files: Requires workspace path resolution
- ConfigMaps fetched at runtime: Adds network dependency

**Implementation Pattern**:
```go
//go:embed manifests/*.yaml
var manifestFS embed.FS
```

### 3. Port Forwarding Strategy

**Decision**: Use `kubectl port-forward` as background processes, managed by the CLI

**Rationale**:
- Standard Kubernetes tooling
- Survives service restarts (reconnects automatically with `--pod-running-timeout`)
- Simple process management with PID tracking

**Alternatives Considered**:
- NodePort services: Requires k3d port mapping configuration, less flexible
- LoadBalancer with MetalLB: Overkill for local dev
- k3d's built-in port mapping: Only works at cluster creation time

**Port Mapping**:
| Service | Cluster Port | Local Port | kubectl Command |
|---------|-------------|------------|-----------------|
| Redpanda Kafka | 9092 | 19092 | `kubectl port-forward svc/redpanda 19092:9092` |
| LocalStack | 4566 | 4566 | `kubectl port-forward svc/localstack 4566:4566` |
| PostgreSQL | 5432 | 5432 | `kubectl port-forward svc/postgres 5432:5432` |

### 4. Health Check Strategy

**Decision**: Use `kubectl wait` for pod readiness, then verify port connectivity

**Rationale**:
- Native Kubernetes approach
- Consistent with k8s deployment patterns
- Timeout configurable

**Implementation**:
```bash
kubectl wait --for=condition=ready pod -l app=redpanda --timeout=120s
kubectl wait --for=condition=ready pod -l app=localstack --timeout=120s
kubectl wait --for=condition=ready pod -l app=postgres --timeout=120s
```

### 5. RuntimeManager Interface Design

**Decision**: Define interface in `sdk/localdev/runtime.go` that both `ComposeManager` and `K3dManager` implement

**Rationale**:
- Clean abstraction for runtime selection
- Existing `ComposeManager` already has the right method signatures
- Enables future runtimes (kind, minikube) with same interface

**Interface**:
```go
type RuntimeManager interface {
    Up(ctx context.Context, detach bool, output io.Writer) error
    Down(ctx context.Context, removeVolumes bool, output io.Writer) error
    Status(ctx context.Context) (*StackStatus, error)
    WaitForHealthy(ctx context.Context, timeout time.Duration) error
}
```

### 6. Configuration File Format

**Decision**: YAML configuration at `~/.config/dp/config.yaml`

**Rationale**:
- Standard XDG config location
- YAML consistent with dp.yaml manifests
- Simple key-value for initial implementation

**Schema**:
```yaml
dev:
  runtime: k3d  # or compose (default)
  workspace: /path/to/dp/workspace  # optional, for compose mode
  k3d:
    clusterName: dp-local
    kubeContext: k3d-dp-local
```

### 7. Prerequisite Checking

**Decision**: Check for k3d, kubectl, and Docker before cluster operations

**Rationale**:
- Fail fast with actionable error messages
- Check version compatibility where needed

**Checks**:
```go
// Check binaries exist and are executable
exec.LookPath("k3d")
exec.LookPath("kubectl") 
exec.LookPath("docker")

// Check Docker daemon running
docker info

// Check k3d version (optional, for compatibility)
k3d version --output json
```

### 8. Kubernetes Manifest Design for Services

**Decision**: Use Deployments with Services for all three components

**Rationale**:
- StatefulSets unnecessary for ephemeral local dev
- Deployments simpler to manage and restart
- PersistentVolumeClaims for data persistence within cluster lifetime

**Service Configurations**:

**Redpanda** (single-node dev mode):
- Image: `docker.redpanda.com/redpandadata/redpanda:v24.2.4`
- Ports: 9092 (Kafka), 8081 (Schema Registry), 8082 (HTTP Proxy)
- Mode: `--mode dev-container --smp 1`

**LocalStack**:
- Image: `localstack/localstack:3.0`
- Port: 4566 (unified gateway)
- Services: S3 enabled by default

**PostgreSQL**:
- Image: `postgres:16-alpine`
- Port: 5432
- Default credentials: postgres/postgres (local dev only)

## Resolved Clarifications

All technical context items resolved. No NEEDS CLARIFICATION markers remain.

## Dependencies Identified

| Dependency | Version | Purpose |
|------------|---------|---------|
| k3d | >= 5.0 | Kubernetes cluster management |
| kubectl | >= 1.28 | Kubernetes API access |
| Docker | >= 24.0 | Container runtime for k3d |

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Port conflicts with existing services | Medium | High | Check port availability before starting; clear error messages |
| k3d version incompatibility | Low | Medium | Document minimum version; version check at startup |
| Port forward instability | Low | Medium | Implement reconnection logic; provide status command |
| Resource exhaustion on low-spec machines | Medium | Medium | Document minimum requirements (4GB RAM) |
