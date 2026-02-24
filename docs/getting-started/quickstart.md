---
title: Quickstart
description: Create, run, and publish your first data package in 10 minutes
---

# Quickstart

This tutorial walks you through the complete DP workflow: from creating a new data package to promoting it to an environment.

**Time to complete**: ~10 minutes

## What You'll Build

A simple Kafka-to-S3 pipeline that:

1. Reads messages from a Kafka topic
2. Transforms the data
3. Writes to an S3 bucket

## Step 1: Create a New Package

Initialize a new data package:

```bash
dp init my-first-pipeline --kind model --runtime cloudquery
```

This creates the following structure:

```
my-first-pipeline/
├── dp.yaml           # Package manifest (includes runtime config)
├── bindings.yaml     # Infrastructure bindings
└── src/
    └── main.py       # Your pipeline code
```

Let's look at the generated manifest:

```yaml title="dp.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: my-first-pipeline
  namespace: default
  version: 1.0.0
spec:
  type: pipeline
  description: A sample data pipeline
  owner: my-team
  
  # Runtime configuration (all in one file!)
  runtime:
    image: myorg/my-first-pipeline:v1.0.0
    timeout: 30m
    retries: 3
  
  inputs:
    - name: events
      type: kafka-topic
      binding: input.events
      
  outputs:
    - name: processed
      type: s3-prefix
      binding: output.data
      classification:
        pii: false
        sensitivity: internal
```

## Step 2: Start Local Development

Start the local development stack. This deploys four services as Helm charts into a local k3d cluster:

- **Redpanda** — Kafka-compatible streaming (port 19092)
- **LocalStack** — AWS-compatible S3 (port 4566)
- **PostgreSQL** — Relational database (port 5432)
- **Marquez** — Data lineage tracking (ports 5000, 3000)

```bash
dp dev up
```

Each chart includes init jobs that automatically create topics, buckets, database schemas, and lineage namespaces — no manual setup required.

!!! tip "Alternative: Use Docker Compose"
    You can also run the local development stack using Docker Compose:
    
    ```bash
    dp dev up --runtime=compose
    ```

Check the status:

```bash
dp dev status
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
    ```bash
    dp config set dev.charts.redpanda.version 25.2.0
    dp config set dev.charts.postgres.values.primary.resources.limits.memory 1Gi
    ```

## Step 3: Validate Your Package

Check your package for errors:

```bash
dp lint ./my-first-pipeline
```

Expected output:

```
✓ dp.yaml: valid
✓ bindings.yaml: valid

All validations passed!
```

## Step 4: Run Locally

Execute your pipeline against the local stack:

```bash
dp run ./my-first-pipeline
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
    The `dp lineage` command is planned but not yet implemented. For now, use the Marquez UI directly.

Open the Marquez UI to view the lineage graph:

- **Marquez Web UI**: http://localhost:3000 — Visual lineage graph
- **Marquez API**: http://localhost:5000 — REST API for querying lineage

You can also check the logs from your run:

```bash
dp logs my-first-pipeline --follow
```

## Step 6: Build the Package

Create an OCI artifact from your package:

```bash
dp build ./my-first-pipeline
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
dp publish ./my-first-pipeline
```

!!! note "Authentication Required"
    You may need to authenticate with your registry first:
    ```bash
    docker login ghcr.io
    ```

## Step 8: Promote to Environment

Deploy to the development environment:

```bash
dp promote my-first-pipeline v0.1.0 --to dev
```

This creates a GitOps PR that will be reviewed and merged.

## Step 9: Check Status

Monitor your package across environments:

```bash
dp status my-first-pipeline
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
dp dev down
```

## Summary

You've completed the full DP workflow:

| Step | Command | What It Does |
|------|---------|--------------|
| 1 | `dp init` | Create a new data package |
| 2 | `dp dev up` | Start local infrastructure |
| 3 | `dp lint` | Validate manifests |
| 4 | `dp run` | Execute locally |
| 5 | ~~`dp lineage`~~ | View data lineage *(not yet implemented — use Marquez UI)* |
| 6 | `dp logs` | Stream logs from a run |
| 7 | `dp build` | Create OCI artifact |
| 8 | `dp publish` | Push to registry |
| 9 | `dp promote` | Deploy to environment |
| 10 | `dp status` | Check deployment status |
| 11 | `dp dev down` | Stop local stack |

## Next Steps

Now that you understand the basics:

- **[Concepts](../concepts/index.md)** - Deep dive into data packages, manifests, and lineage
- **[Tutorials](../tutorials/index.md)** - Build more complex pipelines
- **[CLI Reference](../reference/cli.md)** - Complete command documentation

!!! success "Congratulations!"
    You've successfully created, run, and published your first data package!

---

## CloudQuery Plugin Quickstart

This section walks you through creating and running a CloudQuery source plugin.

### Prerequisites

In addition to the standard prerequisites, you need:

- **CloudQuery CLI** — Install with `brew install cloudquery/tap/cloudquery` (macOS) or see [CloudQuery docs](https://docs.cloudquery.io/docs/quickstart)
- **Docker** — Required for building and running the plugin container

### Step 1: Scaffold a Plugin

```bash
# Create a CloudQuery source plugin (Python runtime)
dp init my-source --kind source --runtime cloudquery

# Or a Go plugin
dp init my-source --kind source --runtime generic-go
```

### Step 2: Explore the Generated Code

```bash
cd my-source
```

The scaffolded project includes:

- `dp.yaml` — Package manifest with CloudQuery configuration
- `main.py` — gRPC server entry point
- `plugin/` — Plugin implementation (tables, client, spec)
- `tests/` — Unit tests

### Step 3: Run Unit Tests

```bash
dp test
```

This runs `pytest` (Python) or `go test ./...` (Go) against the generated test suite.

### Step 4: Start Local Dev Stack

```bash
dp dev up
```

This starts PostgreSQL (and other services) for the sync destination.

### Step 5: Run the Plugin

```bash
dp run
```

This will:

1. Build the plugin Docker image
2. Start the container with gRPC port exposed
3. Wait for the gRPC server to be ready
4. Generate a CloudQuery sync configuration
5. Run `cloudquery sync` to fetch data into PostgreSQL
6. Display a sync summary

### Step 6: Validate and Publish

```bash
dp lint           # Validate manifest
dp build          # Build OCI artifact
dp publish        # Push to registry
```

### Step 7: Clean Up

```bash
dp dev down
```
