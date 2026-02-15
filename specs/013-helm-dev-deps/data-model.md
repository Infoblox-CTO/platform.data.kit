# Data Model: Helm-Based Dev Dependencies

**Feature**: 013-helm-dev-deps | **Date**: 2026-02-15

## Entities

### ChartDefinition

Represents a single dev dependency Helm chart with all metadata needed for uniform deployment, health checking, port forwarding, and status reporting.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Unique identifier (e.g., `redpanda`, `localstack`, `postgres`, `marquez`) |
| ReleaseName | string | Helm release name (e.g., `dp-redpanda`) |
| Namespace | string | Kubernetes namespace (default `dp-local`) |
| PortForwards | []PortForward | Port-forwarding rules for this chart's services |
| HealthLabels | map[string]string | Pod label selector for health checking |
| HealthTimeout | duration | Max time to wait for healthy pods |
| DisplayEndpoints | []DisplayEndpoint | Human-readable endpoint info for `dp dev status` output |

### PortForward

Defines a single port-forwarding rule from localhost to a Kubernetes service.

| Field | Type | Description |
|-------|------|-------------|
| ServiceName | string | Kubernetes service name to forward to |
| LocalPort | int | Port on localhost |
| RemotePort | int | Port on the Kubernetes service |
| Protocol | string | `TCP` (default) |

### DisplayEndpoint

Human-readable endpoint information shown in `dp dev status` and after `dp dev up`.

| Field | Type | Description |
|-------|------|-------------|
| Label | string | Display name (e.g., `Kafka`, `S3 API`, `Marquez API`) |
| URL | string | Localhost URL (e.g., `localhost:19092`, `http://localhost:4566`) |

### ChartOverride (config)

User-configurable overrides for a specific chart, stored in the hierarchical config system.

| Field | Type | Description |
|-------|------|-------------|
| Version | string | Chart version override (empty = use embedded default) |
| Values | map[string]any | Additional Helm `--set` values merged at deploy time |

### ChartRegistry

The embedded collection of all chart definitions. Defined as a Go slice of `ChartDefinition` — not a separate data structure.

| Property | Description |
|----------|-------------|
| Source | `sdk/localdev/charts/embed.go` |
| Population | Statically defined in Go code |
| Extensibility | Add a new entry + chart directory to register a new dependency |

## Relationships

```
Config (dev.charts.<name>)
  └── ChartOverride (0..1 per ChartDefinition)

ChartRegistry ([]ChartDefinition)
  ├── ChartDefinition: redpanda
  │     ├── PortForward: 19092→9092 (kafka)
  │     ├── PortForward: 8080→8080 (console)
  │     ├── PortForward: 18081→8081 (schema-registry)
  │     └── DisplayEndpoint: Kafka, Console, Schema Registry
  ├── ChartDefinition: localstack
  │     ├── PortForward: 4566→4566 (gateway)
  │     └── DisplayEndpoint: S3 API
  ├── ChartDefinition: postgres
  │     ├── PortForward: 5432→5432 (postgres)
  │     └── DisplayEndpoint: PostgreSQL
  └── ChartDefinition: marquez
        ├── PortForward: 5000→5000 (api)
        ├── PortForward: 5001→5001 (admin)
        ├── PortForward: 3000→3000 (web)
        └── DisplayEndpoint: Marquez API, Marquez Admin, Marquez Web

embed.FS
  └── Contains chart directories (Chart.yaml, values.yaml, templates/, charts/*.tgz)
```

## State Transitions

### Dev Stack Lifecycle

```
[Not Running] ──dp dev up──▶ [Deploying] ──all healthy──▶ [Running]
                                  │                            │
                                  │ partial failure            │
                                  ▼                            │
                            [Partial]                          │
                              │ dp dev up (retry)              │
                              └──────────────────▶ [Running]   │
                                                               │
[Running] ──dp dev down──▶ [Stopping] ──────────▶ [Not Running]
```

### Per-Chart States (during deployment)

```
[Pending] ──helm upgrade──▶ [Installing] ──success──▶ [Deployed]
                                  │
                                  │ failure
                                  ▼
                            [Failed] ──dp dev up (retry)──▶ [Installing]
```

## Validation Rules

- `ChartDefinition.Name` must be unique across the registry
- `PortForward.LocalPort` must be unique across all chart definitions (no port conflicts)
- `ChartOverride.Version` when set must be a valid SemVer string
- `ChartOverride.Values` keys must be valid Helm `--set` paths (dot-separated)

## Port Allocation

| Local Port | Service | Chart | Remote Port |
|------------|---------|-------|-------------|
| 19092 | Kafka broker | redpanda | 9092 |
| 18081 | Schema Registry | redpanda | 8081 |
| 8080 | Redpanda Console | redpanda | 8080 |
| 4566 | S3 API | localstack | 4566 |
| 5432 | PostgreSQL | postgres | 5432 |
| 5000 | Marquez API | marquez | 5000 |
| 5001 | Marquez Admin | marquez | 5001 |
| 3000 | Marquez Web | marquez | 3000 |
