---
title: Cells & Stores
description: Understanding the Package × Cell deployment model
---

# Cells & Stores

DataKit uses a **Package × Cell** model to separate what runs from where it runs. This page explains how Cells and Stores provide infrastructure context for data packages.

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
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Cell
metadata:
  name: canary
spec:
  namespace: dk-canary              # k8s namespace for Jobs/Deployments
  labels:
    tier: canary
    region: us-east-1
status:
  ready: true
  storeCount: 2
  packageCount: 3
```

Each Cell owns a dedicated Kubernetes namespace (`dk-<cell>`) where its Stores, Secrets, and workloads live.

### Store (namespaced CRD)

A Store is a namespaced Kubernetes Custom Resource that provides connection details for a specific infrastructure instance. Stores live in their Cell's namespace.

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: dk-canary               # belongs to cell:canary
spec:
  connector: postgres
  connection:
    connection_string: "postgresql://canary-db:5432/dk_canary"
  secrets:
    password: ${PG_PASS}             # resolved from k8s Secret
```

The same logical Store name can exist in every cell, pointing to different physical infrastructure:

| Cell | Store name | Physical target |
|------|-----------|-----------------|
| `local` | `warehouse` | `postgresql://localhost:5432/dp_local` |
| `canary` | `warehouse` | `postgresql://canary-db:5432/dk_canary` |
| `stable` | `warehouse` | `postgresql://prod-db:5432/dp_stable` |

## Discovery

| Concept | Discovery mechanism |
|---------|---------------------|
| **Cell** | `dk cell list` or `kubectl get cells` |
| **Store** | `dk cell stores <name>` or `kubectl get stores -n dk-<cell>` |
| **Package** | `dk search` or OCI registry tags |

```bash
# List all cells in current cluster
dk cell list

# Show details of a specific cell
dk cell show canary

# List stores in a cell
dk cell stores canary

# Target a different cluster
dk cell list --context arn:aws:eks:us-east-1:...:cluster/dk-prod
```

## Resolution Order

When running a package, manifests are resolved by these rules:

| Manifest kind | Source | Rationale |
|---|---|---|
| **Transform** | Package | The computation is the thing being versioned |
| **DataSet** | Package | Data contracts are part of the package interface |
| **Connector** | Package | Plugin images are versioned with the code |
| **Store** | Cell (if `--cell` set), else package `store/` | Infrastructure varies per cell |

```bash
dk run                     # Stores from package store/ directory (local dev)
dk run --cell canary       # Stores from cell:canary via kubectl
dk run --cell stable       # Stores from cell:stable via kubectl
```

The package always ships a `store/` directory with local dev defaults so `dk run` works out of the box without any cell configuration.

## Cell Lifecycle

### Creating cells locally

```bash
# dk dev up creates k3d cluster + shared infra + cell:local
dk dev up

# Create additional cells for testing
dk dev up --cell canary
dk dev up --cell dev-dgarcia
```

### Creating cells in production (GitOps)

Cells are laid out under `gitops/envs/{env}/cells/` in the GitOps repository. Each cell has `stores/` (infrastructure) and `apps/` (deployed packages):

```
gitops/envs/
├── dev/
│   └── cells/
│       └── c0/
│           ├── stores/
│           │   ├── warehouse.yaml         # kind: Store (namespace: dk-c0)
│           │   └── lake-raw.yaml
│           └── apps/                      # Per-package values.yaml (managed by dk promote)
│               └── my-pipeline/
│                   └── values.yaml        # appVersion: v1.0.0
├── int/
│   └── cells/
│       └── c0/
│           ├── stores/
│           └── apps/
└── prod/
    └── cells/
        └── c0/
            ├── stores/
            └── apps/
```

ArgoCD discovers apps via a git generator on `envs/*/cells/*/apps/*` and renders the shared `dk-app` chart with each app's `values.yaml`.

Promote a package to an environment (default cell `c0`):
```bash
dk promote my-pipeline v1.0.0 --to dev
```

### Cells across clusters

Cells are k8s CRDs — they live in a specific cluster. When you have multiple clusters, each has its own set of cells:

```
k3d-dk-local
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
dk run --cell canary --context k3d-dk-local
dk cell list --context arn:aws:eks:us-east-1:...:cluster/dk-prod
```

## Package × Cell Deployment

### What ships in the package

When you `dk build` and `dk publish`, the package becomes an immutable OCI artifact in the registry. The artifact contains:

| Included | Excluded |
|----------|----------|
| `dk.yaml` (Transform) | `store/` directory |
| `connector/*.yaml` | `src/` (baked into image) |
| `dataset/*.yaml` | `tests/` |

The `store/` directory is intentionally excluded — stores are cell-specific and resolved at deploy time.

A shared `dk-app` Helm chart (in `gitops/charts/dk-app/`) is used for all deployments. There is no per-package chart.

### Deploying to a cell

To deploy `pg-to-s3:1.2.4` to the `canary` cell in dev:

```bash
dk promote pg-to-s3 1.2.4 --to dev --cell canary
```

This creates a PR that writes `envs/dev/cells/canary/apps/pg-to-s3/values.yaml`:

```yaml title="envs/dev/cells/canary/apps/pg-to-s3/values.yaml"
appVersion: "1.2.4"
# Optional overrides:
# resources:
#   requests:
#     cpu: 200m
#     memory: 512Mi
# schedule: "0 */6 * * *"
```

ArgoCD discovers the app via git generator, renders the shared `dk-app` chart with the cell and appVersion, and applies the `PackageDeployment` CR to `dk-canary` namespace.

### Cross-cell transforms

When a Transform reads from one cell and writes to another, use the `cell` field on DataSet references:

```yaml title="dk.yaml"
spec:
  inputs:
    - dataset: raw-events
      cell: cell-us-east          # read from us-east cell
  outputs:
    - dataset: processed-events   # write to deployment cell (default)
```

!!! tip "See Also"
    - [Data Packages](data-packages.md) — package structure and manifests
    - [Environments](environments.md) — promotion workflow
    - [Deploying to Cells tutorial](../tutorials/deploying-to-cells.md) — step-by-step guide
