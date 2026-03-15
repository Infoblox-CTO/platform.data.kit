---
title: Deploying to Cells
description: End-to-end guide for building and deploying data packages to cells
---

# Deploying to Cells

This tutorial walks through the complete developer journey: from creating a data package to deploying it across multiple cells.

## Prerequisites

- `dk` CLI installed ([Installation](../getting-started/installation.md))
- Local dev environment running (`dk dev up`)
- Basic understanding of [Data Packages](../concepts/data-packages.md) and [Cells](../concepts/cells.md)

## Step 1: Create a Package

Scaffold a new CloudQuery-based pipeline:

```bash
dk init pg-to-s3 --runtime cloudquery
cd pg-to-s3
```

This creates a self-contained package:

```
pg-to-s3/
├── dk.yaml                        # Transform manifest
├── connector/
│   ├── postgres.yaml              # Source connector
│   └── s3.yaml                    # Destination connector
├── dataset/
│   ├── pg-to-s3-source-table.yaml # Source DataSet (store: source-db)
│   └── pg-to-s3-dest-table.yaml   # Dest DataSet (store: dest-bucket)
└── store/                         # Local dev stores (fallback)
    ├── source-db.yaml             # localhost PostgreSQL
    └── dest-bucket.yaml           # LocalStack S3
```

Key: Assets reference Stores by name (`store: source-db`). The `store/` directory provides those names with local dev connection strings. No cell needed yet.

## Step 2: Run Locally with Package Stores

```bash
dk run
```

With no `--cell` flag, the runner uses `store/` from the package directory:

```
✓ Loading dk.yaml
✓ Resolving stores from package store/ directory
  - source-db: postgres → localhost:5432
  - dest-bucket: s3 → localhost:4566 (LocalStack)
✓ Generating CloudQuery config
✓ Running pipeline...
```

This works out of the box — no cluster or cell configuration needed.

## Step 3: Create a Cell

Set up a cell in your local k3d cluster:

```bash
dk dev up --cell canary
```

This creates:

1. Kubernetes namespace `dk-canary`
2. Cell CR (cluster-scoped) named `canary`
3. Store CRs with connection details for the cell's infrastructure

Verify the cell exists:

```bash
dk cell list
```

```
NAME      NAMESPACE    READY   STORES   PACKAGES   AGE
local     dk-local     true    2        0          1h
canary    dk-canary    true    2        0          5s
```

Check the cell's stores:

```bash
dk cell stores canary
```

```
NAME           CONNECTOR   READY   AGE
source-db      postgres    true    5s
dest-bucket    s3          true    5s
```

## Step 4: Run Against a Cell

Now run the same package against the canary cell:

```bash
dk run --cell canary
```

The `store/` directory is skipped. Stores are resolved from the cell:

```
✓ Loading dk.yaml
✓ Resolving stores from cell: canary
  - source-db: kubectl get store source-db -n dk-canary → postgres://canary-db:5432/dk_canary
  - dest-bucket: kubectl get store dest-bucket -n dk-canary → s3://dk-canary-raw
✓ Generating CloudQuery config
✓ Running pipeline...
```

The package doesn't change — only the store resolution changes.

### Targeting a different cluster

If you have multiple clusters, use `--context` to specify which one:

```bash
dk run --cell canary --context k3d-dk-local
dk run --cell us-east --context arn:aws:eks:us-east-1:...:cluster/dk-prod
```

## Step 5: Build and Publish

Validate the package and create an OCI artifact:

```bash
dk build
```

```
Building package: .

Step 1/3: Validating manifests...
✓ Validation passed

Step 2/3: Gathering build info...
  Git commit: 29aef3c
  Git branch: main

Step 3/3: Creating OCI artifact bundle...
✓ Build complete!

Artifact Info:
  Name:      pg-to-s3
  Version:   1.2.4
  Layers:    2
  OCI Size:  2.1 KB
```

The OCI artifact contains your Transform manifest, Connectors, and DataSets. The `store/` directory is **not** included — stores are cell-specific.

Publish to registry:

```bash
dk publish --registry ghcr.io/myorg
```

```
Step 1/3: Building artifact...
✓ Artifact built

Step 2/3: Checking tag availability...
✓ Tag is available

Step 3/3: Pushing to registry...
✓ OCI artifact pushed
```

## Step 6: Deploy to a Cell via GitOps

Promote the package to a cell using the CLI:

```bash
dk promote pg-to-s3 1.2.4 --to dev --cell canary
```

The `--to` flag specifies the environment (always required). The `--cell` flag is optional and defaults to `c0`.

This creates a PR that writes a `values.yaml` to the environment + cell layout:

```
envs/dev/cells/canary/apps/pg-to-s3/values.yaml
```

```yaml title="envs/dev/cells/canary/apps/pg-to-s3/values.yaml"
appVersion: "1.2.4"
```

ArgoCD uses a git generator to discover `envs/*/cells/*/apps/*` and renders the shared `dk-app` chart with the per-app `values.yaml`, creating a `PackageDeployment` in namespace `dk-canary`.

### Deploy to another environment

```bash
dk promote pg-to-s3 1.2.3 --to prod --cell stable
```

This creates `envs/prod/cells/stable/apps/pg-to-s3/values.yaml` with the specified version.

### Promote using the default cell

When `--cell` is omitted, it defaults to `c0`:

```bash
dk promote pg-to-s3 1.2.4 --to prod
```

This creates `envs/prod/cells/c0/apps/pg-to-s3/values.yaml`.

## Summary

| Step | Command | What happens |
|------|---------|-------------|
| Create | `dk init pg-to-s3 --runtime cloudquery` | Scaffold package with local stores |
| Dev | `dk run` | Run with `store/` directory |
| Test | `dk run --cell canary` | Run with cell stores |
| Build | `dk build` | Create OCI artifact (stores excluded) |
| Publish | `dk publish --registry ghcr.io/myorg` | Push OCI artifact to registry |
| Deploy | `dk promote pg-to-s3 1.2.4 --to dev --cell canary` | PR updates cell values.yaml, ArgoCD deploys |
| Promote | `dk promote pg-to-s3 1.2.4 --to prod` | Same — update version in target env (default cell c0) |

!!! tip "Key Insight"
    The package is always the same — it's **immutable**. What changes between environments is which Cell provides the Stores. This separation means your pipeline code never contains connection strings, credentials, or environment-specific configuration.
