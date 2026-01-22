---
title: Data Packages
description: Understanding the fundamental unit of the Data Platform
---

# Data Packages

A **data package** is the fundamental unit of work in the Data Platform. It's a self-contained, versioned bundle that includes everything needed to run a data pipeline.

## What is a Data Package?

Think of a data package as a "container" for your data pipeline:

| Aspect | Description |
|--------|-------------|
| **Self-contained** | All configuration, code, and metadata in one place |
| **Versioned** | Immutable versions tracked in an OCI registry |
| **Portable** | Same package runs locally and in production |
| **Observable** | Built-in lineage tracking and metadata |

## Package Structure

Every data package follows this structure:

```
my-package/
├── dp.yaml           # Package manifest (required)
├── pipeline.yaml     # Pipeline configuration (optional)
├── bindings.yaml     # Infrastructure bindings (optional)
├── src/              # Source code
│   └── main.py
└── tests/            # Tests (optional)
    └── test_pipeline.py
```

### dp.yaml (Manifest)

The manifest is the heart of every package:

```yaml title="dp.yaml"
apiVersion: dp.io/v1alpha1
kind: DataPackage
metadata:
  name: my-kafka-pipeline
  namespace: analytics
  labels:
    team: data-engineering
    domain: events
spec:
  type: pipeline
  description: Processes event data from Kafka to S3
  owner: data-engineering@example.com
  
  # What this package consumes
  inputs:
    - name: events
      type: kafka-topic
      binding: input.events
      
  # What this package produces
  outputs:
    - name: processed-events
      type: s3-prefix
      binding: output.data
      classification:
        pii: false
        sensitivity: internal
```

!!! tip "See Also"
    Full manifest schema in [Manifests Reference](manifests.md).

### pipeline.yaml

Pipeline-specific configuration:

```yaml title="pipeline.yaml"
apiVersion: dp.io/v1alpha1
kind: PipelineConfig
spec:
  runtime: python:3.11
  
  schedule:
    cron: "0 */6 * * *"  # Every 6 hours
    
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "2"
      
  retries:
    maxAttempts: 3
    backoffMultiplier: 2
```

### bindings.yaml

References to infrastructure resources:

```yaml title="bindings.yaml"
apiVersion: dp.io/v1alpha1
kind: Bindings
spec:
  bindings:
    input.events:
      type: kafka-topic
      ref: production/user-events
      
    output.data:
      type: s3-prefix
      ref: analytics-bucket/processed/events/
```

## Package Types

The DP CLI supports multiple package types:

| Type | Purpose | Example |
|------|---------|---------|
| `pipeline` | Data transformation pipeline | Kafka → S3 ETL |
| `producer` | Data source/producer | Sensor data publisher |
| `consumer` | Data consumer | Dashboard data loader |
| `streaming` | Real-time streaming job | Kafka Streams app |

Specify the type when initializing:

```bash
dp init my-package --type pipeline
dp init my-producer --type producer
```

## Package Lifecycle

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Package Lifecycle                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌───────┐ │
│  │ Create  │ → │ Develop │ → │  Build  │ → │ Publish │ → │Promote│ │
│  │(dp init)│   │(dp dev) │   │(dp build│   │(dp push)│   │(dp ↑) │ │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘   └───────┘ │
│       │             │             │             │             │     │
│       ▼             ▼             ▼             ▼             ▼     │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌───────┐ │
│  │  Local  │   │  Local  │   │   OCI   │   │   OCI   │   │  K8s  │ │
│  │  Files  │   │  Stack  │   │Artifact │   │Registry │   │ Env   │ │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘   └───────┘ │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 1. Create

Initialize a new package with templates:

```bash
dp init analytics-pipeline --type pipeline
```

### 2. Develop

Iterate locally with the dev stack:

```bash
dp dev up          # Start local services
dp run ./package   # Test pipeline
dp dev down        # Stop services
```

### 3. Build

Package as an OCI artifact:

```bash
dp build ./package
# Output: analytics-pipeline:v1.0.0
```

### 4. Publish

Push to registry:

```bash
dp publish ./package
# Pushes to: ghcr.io/org/analytics-pipeline:v1.0.0
```

### 5. Promote

Deploy to an environment:

```bash
dp promote analytics-pipeline v1.0.0 --to dev
```

## Versioning

Packages use semantic versioning:

| Version Part | When to Increment |
|--------------|-------------------|
| **Major** (X.0.0) | Breaking changes to inputs/outputs |
| **Minor** (0.X.0) | New features, backward compatible |
| **Patch** (0.0.X) | Bug fixes, no behavior change |

Versions are immutable once published:

```bash
# Publish version 1.0.0
dp build --version v1.0.0
dp publish

# Cannot overwrite - must increment
dp build --version v1.0.1
dp publish
```

## Inputs and Outputs

### Declaring Inputs

Inputs describe what data the package consumes:

```yaml
inputs:
  - name: events           # Unique name within package
    type: kafka-topic      # Type of data source
    binding: input.events  # Reference to binding
    schema: events.avsc    # Optional schema file
```

### Declaring Outputs

Outputs describe what data the package produces:

```yaml
outputs:
  - name: processed
    type: s3-prefix
    binding: output.data
    classification:
      pii: false
      sensitivity: internal
    schema: output.avsc
```

### Supported Types

| Type | Description |
|------|-------------|
| `kafka-topic` | Kafka topic |
| `s3-prefix` | S3 bucket prefix |
| `database-table` | Database table |
| `http-endpoint` | HTTP API endpoint |

## Next Steps

- [Manifests](manifests.md) - Detailed manifest schema
- [Lineage](lineage.md) - How lineage is tracked
- [Environments](environments.md) - Deployment environments
