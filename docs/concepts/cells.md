---
title: Cells & Stores
description: Understanding the Package × Cell deployment model
---

# Cells & Stores

The Data Platform uses a **Package × Cell** model to separate what runs from where it runs. This page explains how Cells and Stores provide infrastructure context for data packages.

## Core Principle

Every deployment is the intersection of two independent dimensions:

```
WHAT runs  ×  WHERE it runs  =  Deployment
(Package)     (Cell)
```

- **Package** — a versioned, immutable Helm chart containing the Transform, Assets, and Connectors. It defines the computation, the data contracts, and the plugin versions.
- **Cell** — a named infrastructure context providing Stores (connection strings, credentials, namespace). It defines where data lives.

The same package can deploy to many cells. The same cell can host many packages.

## Concepts

### Cell (cluster-scoped CRD)

A Cell is a cluster-scoped Kubernetes Custom Resource representing an isolated instance of pipeline infrastructure.

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Cell
metadata:
  name: canary
spec:
  namespace: dp-canary              # k8s namespace for Jobs/Deployments
  labels:
    tier: canary
    region: us-east-1
status:
  ready: true
  storeCount: 2
  packageCount: 3
```

Each Cell owns a dedicated Kubernetes namespace (`dp-<cell>`) where its Stores, Secrets, and workloads live.

### Store (namespaced CRD)

A Store is a namespaced Kubernetes Custom Resource that provides connection details for a specific infrastructure instance. Stores live in their Cell's namespace.

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: dp-canary               # belongs to cell:canary
spec:
  connector: postgres
  connection:
    connection_string: "postgresql://canary-db:5432/dp_canary"
  secrets:
    password: ${PG_PASS}             # resolved from k8s Secret
```

The same logical Store name can exist in every cell, pointing to different physical infrastructure:

| Cell | Store name | Physical target |
|------|-----------|-----------------|
| `local` | `warehouse` | `postgresql://localhost:5432/dp_local` |
| `canary` | `warehouse` | `postgresql://canary-db:5432/dp_canary` |
| `stable` | `warehouse` | `postgresql://prod-db:5432/dp_stable` |

## Discovery

| Concept | Discovery mechanism |
|---------|---------------------|
| **Cell** | `dp cell list` or `kubectl get cells` |
| **Store** | `dp cell stores <name>` or `kubectl get stores -n dp-<cell>` |
| **Package** | `dp search` or OCI registry tags |

```bash
# List all cells in current cluster
dp cell list

# Show details of a specific cell
dp cell show canary

# List stores in a cell
dp cell stores canary

# Target a different cluster
dp cell list --context arn:aws:eks:us-east-1:...:cluster/dp-prod
```

## Resolution Order

When running a package, manifests are resolved by these rules:

| Manifest kind | Source | Rationale |
|---|---|---|
| **Transform** | Package | The computation is the thing being versioned |
| **Asset** | Package | Data contracts are part of the package interface |
| **Connector** | Package | Plugin images are versioned with the code |
| **Store** | Cell (if `--cell` set), else package `store/` | Infrastructure varies per cell |

```bash
dp run                     # Stores from package store/ directory (local dev)
dp run --cell canary       # Stores from cell:canary via kubectl
dp run --cell stable       # Stores from cell:stable via kubectl
```

The package always ships a `store/` directory with local dev defaults so `dp run` works out of the box without any cell configuration.

## Cell Lifecycle

### Creating cells locally

```bash
# dp dev up creates k3d cluster + shared infra + cell:local
dp dev up

# Create additional cells for testing
dp dev up --cell canary
dp dev up --cell dev-dgarcia
```

### Creating cells in production (GitOps)

```
platform-repo/
├── cells/
│   ├── canary/
│   │   ├── cell.yaml              # kind: Cell (cluster-scoped)
│   │   └── stores/
│   │       ├── warehouse.yaml     # kind: Store (namespace: dp-canary)
│   │       └── lake-raw.yaml
│   ├── stable/
│   │   ├── cell.yaml
│   │   └── stores/
│   └── us-east/
│       ├── cell.yaml
│       └── stores/
```

Apply to cluster:
```bash
kubectl apply -f cells/canary/
```

### Cells across clusters

Cells are k8s CRDs — they live in a specific cluster. When you have multiple clusters, each has its own set of cells:

```
k3d-dp-local
├── cell:local
├── cell:canary
└── cell:dev-dgarcia

eks-staging
├── cell:canary
├── cell:stable

eks-prod
├── cell:us-east
├── cell:eu-west
```

Use `--context` to target a specific cluster:
```bash
dp run --cell canary --context k3d-dp-local
dp cell list --context arn:aws:eks:us-east-1:...:cluster/dp-prod
```

## Package × Cell Deployment

### What ships in the package

When you `dp build` and `dp publish`, the package becomes an immutable Helm chart in an OCI registry. The chart contains:

| Included | Excluded |
|----------|----------|
| `dp.yaml` (Transform) | `store/` directory |
| `connector/*.yaml` | `src/` (baked into image) |
| `asset/*.yaml` | `tests/` |
| `templates/packagedeployment.yaml` | |

The `store/` directory is intentionally excluded — stores are cell-specific and resolved at deploy time.

### Deploying to a cell

To deploy `pg-to-s3:1.2.4` to cell `canary`, add to your CM repo:

```
cm-repo/apps/pg-to-s3-canary/
├── version.txt    # 1.2.4-g29aef
└── values.yaml    # cell: canary
```

```yaml title="values.yaml"
cell: canary
# Optional overrides:
# resources:
#   requests:
#     cpu: 200m
#     memory: 512Mi
# schedule: "0 */6 * * *"
```

ArgoCD pulls the chart from the OCI registry, renders it with `cell: canary`, and applies the `PackageDeployment` CR to `dp-canary` namespace.

### Cross-cell transforms

When a Transform reads from one cell and writes to another, use the `cell` field on asset references:

```yaml title="dp.yaml"
spec:
  inputs:
    - asset: raw-events
      cell: cell-us-east          # read from us-east cell
  outputs:
    - asset: processed-events     # write to deployment cell (default)
```

!!! tip "See Also"
    - [Data Packages](data-packages.md) — package structure and manifests
    - [Environments](environments.md) — promotion workflow
    - [Deploying to Cells tutorial](../tutorials/deploying-to-cells.md) — step-by-step guide
