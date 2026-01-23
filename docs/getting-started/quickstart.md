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
