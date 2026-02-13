# Data Model: CloudQuery Plugin Package Type

**Feature**: 008-cloudquery-plugins  
**Date**: 2026-02-12

## Entity Diagram

```
┌──────────────────────────────────────────────────────┐
│                  DataPackage (dp.yaml)                │
│  apiVersion, kind                                    │
│  ┌─────────────────────┐  ┌────────────────────────┐ │
│  │  PackageMetadata     │  │  DataPackageSpec        │ │
│  │  name, namespace,    │  │  type: "cloudquery"     │ │
│  │  version, labels,    │  │  description, owner     │ │
│  │  annotations         │  │  runtime: RuntimeSpec   │ │
│  └─────────────────────┘  │  cloudquery: CQSpec ──┐ │ │
│                           └──────────────────────│─┘ │
│                                                  │   │
│  ┌───────────────────────────────────────────────▼─┐ │
│  │  CloudQuerySpec                                  │ │
│  │  role: CQRole ("source" | "destination")         │ │
│  │  tables: []string                                │ │
│  │  grpcPort: int (default: 7777)                   │ │
│  │  concurrency: int (default: 10000)               │ │
│  └──────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────┘
```

## Entities

### PackageType (modified)

Existing enum extended with a new value.

| Value | Description | Status |
|-------|-------------|--------|
| `pipeline` | Existing — batch/streaming data pipeline | Active |
| `cloudquery` | New — CloudQuery source/destination plugin | **Added** |

**File**: `contracts/types.go`

### CloudQuerySpec (new)

Configuration specific to CloudQuery packages. Stored in `dp.yaml` under `spec.cloudquery`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `role` | `CloudQueryRole` | Yes | — | Plugin role: `source` or `destination` |
| `tables` | `[]string` | No | `[]` | Table names this plugin provides |
| `grpcPort` | `int` | No | `7777` | Port the gRPC server listens on |
| `concurrency` | `int` | No | `10000` | Max concurrent table resolvers |

**File**: `contracts/cloudquery.go` (new)

**Validation rules**:
- `role` is required, must be `source` or `destination`
- `role: destination` generates a warning (not yet supported)
- `grpcPort` must be 1–65535 if provided
- `concurrency` must be > 0 if provided

### CloudQueryRole (new)

Enum for CloudQuery plugin role.

| Value | Description | Status |
|-------|-------------|--------|
| `source` | Reads data from external systems | Active |
| `destination` | Writes data to external systems | Reserved, not yet supported |

**File**: `contracts/cloudquery.go` (new)

### DataPackageSpec (modified)

Existing struct gains a new optional field.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `type` | `PackageType` | Yes | Now accepts `"cloudquery"` in addition to `"pipeline"` |
| `description` | `string` | No | Unchanged |
| `owner` | `string` | No | Unchanged |
| `inputs` | `[]ArtifactContract` | No | Not used for cloudquery type |
| `outputs` | `[]ArtifactContract` | No | Not required for cloudquery type (unlike pipeline) |
| `schedule` | `*ScheduleSpec` | No | Not used for cloudquery type |
| `resources` | `*ResourceSpec` | No | Unchanged |
| `lineage` | `*LineageSpec` | No | Unchanged |
| `runtime` | `*RuntimeSpec` | Yes (for cloudquery) | Container image config |
| `cloudquery` | `*CloudQuerySpec` | Yes (when type=cloudquery) | **New field** |

**File**: `contracts/datapackage.go`

### PackageConfig (modified)

Template rendering configuration. Gains new fields for cloudquery scaffolding.

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `Name` | `string` | Yes | Unchanged |
| `Namespace` | `string` | Yes | Unchanged |
| `Team` | `string` | No | Unchanged |
| `Description` | `string` | No | Unchanged |
| `Owner` | `string` | No | Unchanged |
| `Language` | `string` | Yes | `"go"` or `"python"` |
| `Mode` | `string` | No | Pipeline mode — not used for cloudquery |
| `Type` | `string` | No | **New** — `"pipeline"` or `"cloudquery"` |
| `Role` | `string` | No | **New** — `"source"` or `"destination"` |
| `GRPCPort` | `int` | No | **New** — default 7777 |
| `Concurrency` | `int` | No | **New** — default 10000 |

**File**: `cli/internal/templates/renderer.go`

## Relationships

```
DataPackageSpec.Type = "cloudquery"  →  DataPackageSpec.CloudQuery is required
DataPackageSpec.Type = "cloudquery"  →  DataPackageSpec.Runtime is required
DataPackageSpec.Type = "cloudquery"  →  DataPackageSpec.Outputs is NOT required
DataPackageSpec.Type = "pipeline"    →  DataPackageSpec.CloudQuery is ignored/nil
CloudQuerySpec.Role = "destination"  →  Warning: not yet supported
```

## State Transitions

CloudQuery plugins do not have complex state transitions. The relevant states are from the existing `RunStatus` enum and apply uniformly:

```
pending → running → completed
                  → failed
```

The `dp run` command for cloudquery packages has an internal multi-step flow:

```
idle → building → starting_grpc → syncing → summarizing → done
                                           → failed (at any step)
```

This is an internal flow within the `run` command, not a persisted state model.
