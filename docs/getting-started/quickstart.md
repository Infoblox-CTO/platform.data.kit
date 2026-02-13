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
dp init my-first-pipeline --type pipeline
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

Start the local development stack (Kafka, S3, Marquez):

```bash
dp dev up
```

!!! tip "Alternative: Use k3d Runtime"
    You can also run the local development stack using k3d (Kubernetes):
    
    ```bash
    dp dev up --runtime=k3d
    ```
    
    This creates a k3d cluster with the same services. Useful when:
    
    - You want to test Kubernetes-native deployments
    - You're running from a directory without docker-compose.yaml
    - You need to simulate production K8s environments locally

Check the status:

```bash
dp dev status
```

Expected output:

```
Local Development Stack
───────────────────────
Service     Status    Port
kafka       running   9092
minio       running   9000
marquez     running   5000
postgres    running   5432

Marquez UI: http://localhost:5000
```

!!! tip "View Lineage"
    Open http://localhost:5000 in your browser to see the Marquez lineage UI.

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

```bash
dp lineage my-first-pipeline
```

Or open the Marquez UI at http://localhost:5000 to see a visual graph.

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
| 5 | `dp lineage` | View data lineage |
| 6 | `dp build` | Create OCI artifact |
| 7 | `dp publish` | Push to registry |
| 8 | `dp promote` | Deploy to environment |
| 9 | `dp status` | Check deployment status |
| 10 | `dp dev down` | Stop local stack |

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
# Create a Python CloudQuery source plugin
dp init my-source --type cloudquery

# Or a Go plugin
dp init my-source --type cloudquery --lang go
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
