---
title: Quickstart
description: Create, run, and publish your first data package in 10 minutes
---
# Quickstart

This tutorial walks you through the complete Data Kit workflow: from creating a new data package to promoting it to an environment.

**Time to complete**: ~10 minutes

## What You'll Build

A simple data processing model that:

1. Processes input data
2. Transforms the data
3. Outputs results

## Step 1: Create a New Package

Initialize a new data package:

```bash
dk init my-first-pipeline --runtime generic-python
```

This creates the following structure:

```
my-first-pipeline/
├── dk.yaml           # Package manifest
├── main.py           # Your pipeline code
└── requirements.txt  # Python dependencies
```

Let's look at the generated manifest:

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: my-first-pipeline
  namespace: default
  version: 0.1.0
  labels:
    team: my-team
spec:
  runtime: generic-python
  mode: batch
  image: "${REGISTRY}/my-first-pipeline:${VERSION}"

  inputs:
    - dataset: my-first-pipeline-input

  outputs:
    - dataset: my-first-pipeline-output

  resources:
    cpu: "1"
    memory: "2Gi"
```

## Step 2: Start Local Development

Start the local development stack. This deploys four services as Helm charts into a local k3d cluster:

- **Redpanda** — Kafka-compatible streaming (port 19092)
- **LocalStack** — AWS-compatible S3 (port 4566)
- **PostgreSQL** — Relational database (port 5432)
- **Marquez** — Data lineage tracking (ports 5000, 3000)

```bash
dk dev up
```

Each chart includes init jobs that automatically create topics, buckets, database schemas, and lineage namespaces — no manual setup required.

!!! tip "Seed Data"
    If your input assets declare `dev.seed` data, you can load it now:
    ``bash     dk dev seed     ``
    This creates tables and inserts sample rows into the local PostgreSQL.
    Seed data is also loaded automatically before each `dk run`.

Check the status:

```bash
dk dev status
```

Expected output:

```
Local Development Stack
───────────────────────
Chart         Status    Ports
redpanda      healthy   19092, 18081
localstack    healthy   4566
postgres      healthy   5432
marquez       healthy   5000, 3000

Endpoints:
  Kafka:              localhost:19092
  Schema Registry:    http://localhost:18081
  S3 API:             http://localhost:4566
  PostgreSQL:         localhost:5432
  Marquez API:        http://localhost:5000
  Marquez Web:        http://localhost:3000
```

!!! tip "View Lineage"
    Open http://localhost:3000 in your browser to see the Marquez lineage UI.

!!! info "Chart Customization"
    You can override chart versions or Helm values via the config system:
    ``bash     dk config set dev.charts.redpanda.version 25.2.0     dk config set dev.charts.postgres.values.primary.resources.limits.memory 1Gi     ``

## Step 3: Validate Your Package

Check your package for errors:

```bash
dk lint ./my-first-pipeline
```

Expected output:

```
✓ dk.yaml: valid

All validations passed!
```

## Step 4: Run Locally

Execute your pipeline against the local stack:

```bash
dk run ./my-first-pipeline
```

You'll see output like:

```
▶ Starting pipeline: my-first-pipeline
  → Emitting START lineage event
  → Pulling container image...
  → Running pipeline...
  → Processing 100 messages
  → Writing to s3://local/output/
  → Emitting COMPLETE lineage event
✓ Pipeline completed successfully in 12.3s

Run ID: run-abc123
```

## Step 5: Check Lineage

View the lineage for your run:

!!! warning "Not Yet Implemented"
    The `dk lineage` command is planned but not yet implemented. For now, use the Marquez UI directly.

Open the Marquez UI to view the lineage graph:

- **Marquez Web UI**: http://localhost:3000 — Visual lineage graph
- **Marquez API**: http://localhost:5000 — REST API for querying lineage

You can also check the logs from your run:

```bash
dk logs my-first-pipeline --follow
```

## Step 6: Build the Package

Create an OCI artifact from your package:

```bash
dk build ./my-first-pipeline
```

Output:

```
▶ Building package: my-first-pipeline
  → Validating manifest...
  → Bundling files...
  → Creating OCI artifact...
✓ Built: my-first-pipeline:v0.1.0

Artifact: ghcr.io/my-org/my-first-pipeline:v0.1.0
Size: 2.3 MB
```

## Step 7: Publish to Registry

Push the artifact to your OCI registry:

```bash
dk publish ./my-first-pipeline
```

!!! note "Authentication Required"
    You may need to authenticate with your registry first:
    ``bash     docker login ghcr.io     ``

## Step 8: Promote to Environment

Deploy to the development environment:

```bash
dk promote my-first-pipeline v0.1.0 --to dev
```

This creates a GitOps PR that will be reviewed and merged.

## Step 9: Check Status

Monitor your package across environments:

```bash
dk status my-first-pipeline
```

Output:

```
Package: my-first-pipeline

Environment  Version   Status    Last Run
───────────  ───────   ──────    ────────
dev          v0.1.0    running   2 min ago
int          -         -         -
prod         -         -         -
```

## Step 10: Clean Up

When you're done, stop the local stack:

```bash
dk dev down
```

## Summary

You've completed the full DK workflow:

| Step | Command             | What It Does                                                 |
| ---- | ------------------- | ------------------------------------------------------------ |
| 1    | `dk init`         | Create a new data package                                    |
| 2    | `dk dev up`       | Start local infrastructure                                   |
| 2b   | `dk dev seed`     | Load sample data into local stores                           |
| 3    | `dk lint`         | Validate manifests                                           |
| 4    | `dk run`          | Execute locally (auto-seeds if needed)                       |
| 5    | ~~`dk lineage`~~ | View data lineage*(not yet implemented — use Marquez UI)* |
| 6    | `dk logs`         | Stream logs from a run                                       |
| 7    | `dk build`        | Create OCI artifact                                          |
| 8    | `dk publish`      | Push to registry                                             |
| 9    | `dk promote`      | Deploy to environment                                        |
| 10   | `dk status`       | Check deployment status                                      |
| 11   | `dk dev down`     | Stop local stack                                             |

## Next Steps

Now that you understand the basics:

- **[Concepts](../concepts/index.md)** - Deep dive into data packages, manifests, and lineage
- **[Tutorials](../tutorials/index.md)** - Build more complex pipelines
- **[CLI Reference](../reference/cli.md)** - Complete command documentation

!!! success "Congratulations!"
    You've successfully created, run, and published your first data package!

---

## CloudQuery Model Quickstart

This section walks you through creating and running a CloudQuery sync model.

!!! info "What is a CloudQuery Model?"
    CloudQuery models use the [CloudQuery CLI](https://cloudquery.io) to sync data between sources and destinations. The `dk init --runtime cloudquery` command generates a `config.yaml` file that you run with `cloudquery sync`.

### Prerequisites

In addition to the standard prerequisites, you need:

- **CloudQuery CLI** — Install with `brew install cloudquery/tap/cloudquery` (macOS) or see [CloudQuery docs](https://docs.cloudquery.io/docs/quickstart)

### Step 1: Create a CloudQuery Model

```bash
dk init my-sync --runtime cloudquery
```

This creates:

- `dk.yaml` — Package manifest with CloudQuery configuration
- `config.yaml` — CloudQuery sync configuration

### Step 2: Configure the Sync

Edit `config.yaml` to configure your source and destination:

```yaml
kind: source
spec:
  name: my-source
  registry: cloudquery
  path: cloudquery/postgresql
  tables: ["public.my_table"]
  destinations: ["my-destination"]
  spec:
    connection_string: "${CONNECTION_STRING}"

---
kind: destination
spec:
  name: my-destination
  registry: cloudquery
  path: cloudquery/s3
  spec:
    bucket: "my-data-lake"
    path: "raw/my-sync/{{TABLE}}/{{UUID}}.parquet"
```

### Step 3: Start Local Dev Stack

```bash
dk dev up
```

This starts PostgreSQL and LocalStack for local testing.

### Step 4: Run the Sync

CloudQuery models are run directly with the CloudQuery CLI:

```bash
export CONNECTION_STRING="postgres://postgres:postgres@localhost:5432/postgres"
cloudquery sync config.yaml
```

!!! note "Why not `dk run`?"
    CloudQuery models use configuration files rather than application code.
    The CloudQuery CLI handles the actual sync execution.

### Step 5: Validate and Publish

```bash
dk lint           # Validate manifest
dk build          # Build OCI artifact  
dk publish        # Push to registry
```

### Step 6: Clean Up

```bash
dk dev down
```
