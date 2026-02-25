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

## Core Concepts

The data platform uses four core concepts to separate concerns:

| Concept | What it represents | Who creates it |
|---------|--------------------|----------------|
| **Connector** | A technology type (Postgres, S3, Kafka) | Platform team |
| **Store** | A named instance of a Connector with connection details | Infra / SRE |
| **Asset** | A data contract (table, S3 prefix, topic) in a Store | Data engineer |
| **Transform** | A unit of computation that reads/writes Assets | Data engineer |

## Package Structure

A Transform package — the deployable unit — follows this structure:

```
my-pipeline/
├── dp.yaml           # Transform manifest (required)
├── src/              # Source code (generic-go / generic-python)
│   └── main.py
└── tests/            # Tests (optional)
    └── test_pipeline.py
```

For CloudQuery runtimes, no source code is needed — the Connector's plugin images handle execution.

### dp.yaml (Manifest)

The manifest is the heart of every package:

```yaml title="dp.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: my-kafka-pipeline
  namespace: analytics
  version: 1.0.0
  labels:
    team: data-engineering
    domain: events
spec:
  runtime: generic-python       # cloudquery | generic-go | generic-python | dbt
  mode: batch                   # batch | streaming
  description: Processes event data from Kafka to S3
  owner: data-engineering@example.com
  image: myorg/my-pipeline:v1.0.0
  timeout: 30m

  inputs:
    - asset: raw-events         # references an Asset by name

  outputs:
    - asset: processed-events   # references an Asset by name
      classification:
        pii: false
        sensitivity: internal
```

!!! tip "See Also"
    Full manifest schema in [Manifests Reference](manifests.md).

### Runtime Configuration

The `spec` section of a Transform defines how the container runs:

```yaml title="dp.yaml (spec section)"
spec:
  runtime: generic-python
  image: myorg/my-pipeline:v1.0.0       # Required: container image
  timeout: 30m                           # Max execution time
  env:                                   # Environment variables
    - name: LOG_LEVEL
      value: info
  resources:                             # Resource limits
    cpu: "1"
    memory: "2Gi"
```

#### Overriding at Runtime

You can override configuration values without modifying dp.yaml:

```bash
# Override image for local testing
dp run ./my-pipeline --set spec.image=local:dev

# Apply environment-specific overrides
dp run ./my-pipeline -f production.yaml

# Combine both (--set takes precedence)
dp run ./my-pipeline -f production.yaml --set spec.timeout=1h

# Preview merged configuration
dp show ./my-pipeline -f production.yaml --set spec.image=new:v2
```

## Runtimes

The DP CLI supports the following runtimes:

### Generic Python

A containerised Python pipeline. This is the default runtime.

```bash
dp init my-pipeline --runtime generic-python
```

### Generic Go

A containerised Go pipeline.

```bash
dp init my-pipeline --runtime generic-go
```

### CloudQuery

A [CloudQuery](https://docs.cloudquery.io/) sync that uses Connector plugin images to move data between Stores. No application code required.

```bash
dp init my-sync --runtime cloudquery
```

### dbt

A [dbt](https://www.getdbt.com/) transformation project.

```bash
dp init my-transforms --runtime dbt
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
dp init analytics-pipeline --runtime generic-python
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

Inputs declare which Assets a Transform reads:

```yaml
inputs:
  - asset: users              # references an Asset by name
```

### Declaring Outputs

Outputs declare which Assets a Transform produces:

```yaml
outputs:
  - asset: users-parquet      # references an Asset by name
    classification:
      pii: false
      sensitivity: internal
```

At execution time, the runner resolves each Asset → Store → Connector to obtain connection details and credentials.

### Supported Runtimes

| Runtime | Description |
|---------|-------------|
| `cloudquery` | CloudQuery SDK sync |
| `generic-go` | Go container |
| `generic-python` | Python container |
| `dbt` | dbt transformations |

## Next Steps

- [Manifests](manifests.md) - Detailed manifest schema
- [Lineage](lineage.md) - How lineage is tracked
- [Environments](environments.md) - Deployment environments
