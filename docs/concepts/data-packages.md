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
├── dp.yaml           # Package manifest with runtime config (required)
├── bindings.yaml     # Infrastructure bindings (optional)
├── src/              # Source code
│   └── main.py
└── tests/            # Tests (optional)
    └── test_pipeline.py
```

### dp.yaml (Manifest)

The manifest is the heart of every package. It includes all configuration in a single file:

```yaml title="dp.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: my-kafka-pipeline
  namespace: analytics
  version: 1.0.0
  labels:
    team: data-engineering
    domain: events
spec:
  type: pipeline
  description: Processes event data from Kafka to S3
  owner: data-engineering@example.com
  
  # Runtime configuration (required for pipeline type)
  runtime:
    image: myorg/my-pipeline:v1.0.0
    timeout: 30m
    retries: 3
    env:
      - name: LOG_LEVEL
        value: info
    resources:
      cpu: "500m"
      memory: "1Gi"
  
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

### Runtime Configuration

For pipeline packages, the `spec.runtime` section defines how the container runs:

```yaml title="dp.yaml (runtime section)"
spec:
  runtime:
    image: myorg/my-pipeline:v1.0.0     # Required: container image
    timeout: 30m                         # Max execution time
    retries: 3                           # Max retry attempts
    env:                                 # Environment variables
      - name: LOG_LEVEL
        value: info
    envFrom:                             # Environment from secrets/configmaps
      - secretRef:
          name: db-credentials
    resources:                           # Resource limits
      cpu: "1"
      memory: "2Gi"
```

#### Overriding at Runtime

You can override configuration values without modifying dp.yaml:

```bash
# Override image for local testing
dp run ./my-pipeline --set spec.runtime.image=local:dev

# Apply environment-specific overrides
dp run ./my-pipeline -f production.yaml

# Combine both (--set takes precedence)
dp run ./my-pipeline -f production.yaml --set spec.runtime.timeout=1h

# Preview merged configuration
dp show ./my-pipeline -f production.yaml --set spec.runtime.image=new:v2
```

### bindings.yaml

References to infrastructure resources:

```yaml title="bindings.yaml"
apiVersion: data.infoblox.com/v1alpha1
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

The DP CLI supports the following package types:

### Pipeline

A data processing pipeline (e.g., Kafka → S3 ETL). This is the default package type.

```bash
dp init my-pipeline
```

### CloudQuery

A [CloudQuery](https://docs.cloudquery.io/) source plugin that extracts data from external systems using the CloudQuery SDK and gRPC protocol. CloudQuery plugins run as gRPC servers inside containers, and `dp run` orchestrates the full sync lifecycle.

```bash
# Create a Python CloudQuery source plugin (default)
dp init my-source --type cloudquery

# Create a Go CloudQuery source plugin
dp init my-source --type cloudquery --lang go
```

This creates a complete, immediately-runnable plugin project:

```
my-source/
├── dp.yaml                     # Package manifest with cloudquery config
├── main.py                     # gRPC server entry point
├── pyproject.toml              # Python project config
├── requirements.txt            # Python dependencies
├── plugin/
│   ├── __init__.py
│   ├── plugin.py               # Plugin class (get_tables, sync)
│   ├── client.py               # Client for API connections
│   ├── spec.py                 # Plugin configuration spec
│   └── tables/
│       ├── __init__.py
│       └── example_resource.py # Example table definition
└── tests/
    └── test_example_resource.py
```

The generated `dp.yaml` includes CloudQuery-specific configuration:

```yaml title="dp.yaml (cloudquery type)"
apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: my-source
  namespace: default
  version: 0.1.0
spec:
  type: cloudquery
  description: A CloudQuery source plugin
  owner: my-team
  cloudquery:
    role: source           # Plugin role (source only, for now)
    tables:                # Tables this plugin provides
      - example_resource
    grpcPort: 7777         # gRPC server port
    concurrency: 10000     # Max concurrent table resolvers
  runtime:
    image: my-source:latest
```

#### CloudQuery Workflow

```bash
dp init my-source --type cloudquery   # Scaffold plugin
dp test                                # Run unit tests (pytest/go test)
dp dev up                              # Start local dev stack (PostgreSQL)
dp run                                 # Build container → start gRPC → sync
dp test --integration                  # Full sync integration test
dp lint                                # Validate manifest
dp build                               # Build OCI artifact
dp publish                             # Publish to registry
dp promote my-source 0.1.0 --to dev   # Deploy
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
