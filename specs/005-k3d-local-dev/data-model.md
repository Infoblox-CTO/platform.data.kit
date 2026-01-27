# Data Model: k3d Local Development Environment

**Feature**: 005-k3d-local-dev  
**Date**: January 25, 2026

## Entities

### Cluster

Represents a k3d Kubernetes cluster for local development.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Cluster identifier (default: "dp-local") |
| State | enum | running, stopped, creating, deleting, unknown |
| CreatedAt | timestamp | When the cluster was created |
| KubeContext | string | kubectl context name (k3d-{name}) |
| ServerURL | string | Kubernetes API server URL |

**Constraints**:
- Name must be unique per workstation
- Name follows DNS label format (lowercase, alphanumeric, hyphens)

### Service

Represents a deployed workload within the k3d cluster.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Service identifier (redpanda, localstack, postgres) |
| Status | enum | pending, running, failed, unknown |
| Health | enum | healthy, unhealthy, unknown |
| PodCount | int | Number of pods for this service |
| Ports | []PortMapping | Port mappings for the service |

**Constraints**:
- Exactly 3 services for MVP: redpanda, localstack, postgres

### PortMapping

Represents a port mapping between the cluster and localhost.

| Field | Type | Description |
|-------|------|-------------|
| LocalPort | int | Port on localhost |
| ClusterPort | int | Port inside the cluster |
| ServiceName | string | Target service name |
| Protocol | string | tcp or udp (default: tcp) |

**Standard Mappings**:
| Service | LocalPort | ClusterPort |
|---------|-----------|-------------|
| redpanda | 19092 | 9092 |
| localstack | 4566 | 4566 |
| postgres | 5432 | 5432 |

### PortForward

Represents an active port-forward process.

| Field | Type | Description |
|-------|------|-------------|
| PID | int | Process ID of kubectl port-forward |
| LocalPort | int | Local port being forwarded |
| TargetService | string | Kubernetes service being targeted |
| StartedAt | timestamp | When the port forward started |
| Status | enum | active, terminated, failed |

**Constraints**:
- One port forward per local port
- Port forwards are transient (not persisted across CLI invocations)

### Configuration

User preferences for the dp dev commands.

| Field | Type | Description |
|-------|------|-------------|
| Runtime | enum | compose, k3d (default: compose) |
| WorkspacePath | string | Path to DP workspace (for compose mode) |
| ClusterName | string | k3d cluster name (default: dp-local) |

**Storage**: `~/.config/dp/config.yaml`

**Schema**:
```yaml
dev:
  runtime: k3d
  workspace: ""
  k3d:
    clusterName: dp-local
```

### StackStatus

Aggregate status of the entire local development stack.

| Field | Type | Description |
|-------|------|-------------|
| Running | bool | Whether the stack is operational |
| Runtime | string | Current runtime (compose or k3d) |
| Services | []ServiceStatus | Status of each service |
| PortForwards | []PortForward | Active port forwards (k3d only) |
| Errors | []string | Any error messages |

## State Transitions

### Cluster Lifecycle

```
[not exists] --create--> [creating] --success--> [running]
                                    --failure--> [not exists]

[running] --stop--> [stopped]
[stopped] --start--> [running]
[running/stopped] --delete--> [deleting] --success--> [not exists]
```

### Service Lifecycle

```
[pending] --deployed--> [running] --healthy--> [running, healthy]
                                 --unhealthy--> [running, unhealthy]
[running] --crashed--> [failed]
[failed] --restart--> [pending]
```

## Relationships

```
Cluster 1──* Service
Service 1──* PortMapping
Cluster 1──* PortForward
Configuration 1──1 Cluster (by name reference)
```

## Validation Rules

1. **Cluster Name**: Must match pattern `^[a-z0-9][a-z0-9-]*[a-z0-9]$`
2. **Port Range**: LocalPort must be 1024-65535 (unprivileged)
3. **Runtime**: Must be one of: "compose", "k3d"
4. **WorkspacePath**: If specified, must be an absolute path
