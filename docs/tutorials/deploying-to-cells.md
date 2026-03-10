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

Validate the package and produce a Helm chart:

```bash
dk build
```

```
Building package: .

Step 1/4: Validating manifests...
✓ Validation passed

Step 2/4: Gathering build info...
  Git commit: 29aef3c
  Git branch: main

Step 3/4: Creating OCI artifact bundle...

Step 4/4: Generating Helm chart...
✓ Helm chart: dist/pg-to-s3-1.2.4-g29aef3c.tgz (2.1 KB)
✓ Build complete!

Artifact Info:
  Name:      pg-to-s3
  Version:   1.2.4
  Chart:     dist/pg-to-s3-1.2.4-g29aef3c.tgz
```

The chart contains:

- `dk.yaml` (Transform manifest)
- `connector/*.yaml` (Connector definitions)
- `dataset/*.yaml` (DataSet contracts)
- `templates/packagedeployment.yaml` (Helm template)

The `store/` directory is **not** included — stores are cell-specific.

Publish to registry:

```bash
dk publish --registry ghcr.io/myorg
```

```
Step 1/4: Building artifact...
✓ Artifact built

Step 2/4: Preparing Helm chart...
✓ Using existing chart: dist/pg-to-s3-1.2.4-g29aef3c.tgz

Step 3/4: Checking tag availability...
✓ Tag is available

Step 4/4: Pushing to registry...
✓ OCI artifact pushed
✓ Helm chart pushed to oci://ghcr.io/myorg/data-team
```

## Step 6: Deploy to a Cell via GitOps

Add a deployment to your CM repo:

```
cm-repo/apps/pg-to-s3-canary/
├── version.txt
└── values.yaml
```

```title="version.txt"
1.2.4-g29aef3c
```

```yaml title="values.yaml"
cell: canary
```

ArgoCD pulls the Helm chart from the OCI registry, renders it with `cell: canary`, and creates a `PackageDeployment` in namespace `dk-canary`.

### Deploy to another cell

To deploy the same package to `stable`:

```
cm-repo/apps/pg-to-s3-stable/
├── version.txt    # 1.2.3-gabcdef (maybe an older, proven version)
└── values.yaml    # cell: stable
```

### Promote between cells

Update the version in the target cell's directory:

```bash
echo "1.2.4-g29aef3c" > cm-repo/apps/pg-to-s3-stable/version.txt
git commit -am "promote pg-to-s3 to stable" && git push
```

Or use the CLI:

```bash
dk promote pg-to-s3 1.2.4 --to stable
```

## Summary

| Step | Command | What happens |
|------|---------|-------------|
| Create | `dk init pg-to-s3 --runtime cloudquery` | Scaffold package with local stores |
| Dev | `dk run` | Run with `store/` directory |
| Test | `dk run --cell canary` | Run with cell stores |
| Build | `dk build` | Produce Helm chart (stores excluded) |
| Publish | `dk publish --registry ghcr.io/myorg` | Push chart to OCI registry |
| Deploy | Add `version.txt` + `values.yaml` to CM repo | ArgoCD deploys to cell |
| Promote | Update `version.txt` in target cell dir | ArgoCD updates deployment |

!!! tip "Key Insight"
    The package is always the same — it's **immutable**. What changes between environments is which Cell provides the Stores. This separation means your pipeline code never contains connection strings, credentials, or environment-specific configuration.
