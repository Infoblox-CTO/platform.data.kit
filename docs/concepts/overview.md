---
title: Architecture Overview
description: High-level architecture of DataKit
---

# Architecture Overview

DataKit provides a comprehensive system for building, publishing, and operating data pipelines with built-in governance, lineage tracking, and GitOps deployment.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Developer Workflow                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────────────────┐ │
│   │   dk     │───▶│   dk     │───▶│   dk     │───▶│       dk promote     │ │
│   │   init   │    │   dev    │    │  build   │    │  (GitOps PR/Deploy)  │ │
│   └──────────┘    └──────────┘    └──────────┘    └──────────────────────┘ │
│        │               │               │                     │              │
│        ▼               ▼               ▼                     ▼              │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────────────────┐ │
│   │ Package  │    │  Local   │    │   OCI    │    │    Kubernetes        │ │
│   │ Template │    │  Stack   │    │ Artifact │    │    Environment       │ │
│   └──────────┘    └──────────┘    └──────────┘    └──────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. DK CLI

The command-line interface is the primary interaction point for developers:

| Component | Purpose |
|-----------|---------|
| **Scaffolding** | Generate data package structures from templates |
| **Validation** | Verify manifests, schemas, and configurations |
| **Local Runtime** | Run pipelines against local k3d development stack |
| **Packaging** | Bundle packages as OCI artifacts |
| **Deployment** | Create GitOps PRs for environment promotion |

### 2. Data Package

A data package is a self-contained unit containing:

```
my-package/
├── dk.yaml          # Transform manifest (metadata, runtime, inputs, outputs)
└── src/             # Implementation code
```

The `dk.yaml` manifest consolidates all configuration in a single file, including runtime, inputs, and outputs.

!!! info "Learn More"
    See [Data Packages](data-packages.md) for detailed structure and fields.

### 3. Local Development Stack

Helm charts deployed to a local k3d cluster for development:

| Service | Purpose | Default Port |
|---------|---------|--------------|
| **Redpanda** | Kafka-compatible streaming | 19092 |
| **LocalStack** | S3-compatible storage | 4566 |
| **Marquez** | OpenLineage collection | 5000 |
| **PostgreSQL** | Marquez metadata store | 5432 |

### 4. OCI Registry

Data packages are published as OCI artifacts to container registries:

- **Versioning**: Semantic versioning with immutable tags
- **Layers**: Manifest, config, source, and dependencies as separate layers
- **Signing**: Optional artifact signing with Sigstore
- **Discovery**: Registry-based search and metadata

### 5. GitOps Pipeline

Environment promotion uses GitOps principles:

```
dk promote my-package v1.0.0 --to dev
           │
           ▼
┌──────────────────────────────────────────────────────┐
│ 1. Validate artifact exists in registry             │
│ 2. Generate environment-specific manifests          │
│ 3. Create PR to deployment repository               │
│ 4. Run validation checks (linting, policies)        │
│ 5. Await approval and merge                         │
│ 6. ArgoCD syncs to Kubernetes cluster               │
└──────────────────────────────────────────────────────┘
```

## Data Flow

### Development Flow

```
Developer → dk init → Local files → dk dev → Local stack → dk run → Results
                                      │
                                      ▼
                              Marquez (lineage)
```

### Production Flow

```
dk build → OCI Registry → dk promote → GitOps PR → ArgoCD → Kubernetes
                                          │
                                          ▼
                                  Marquez (production lineage)
```

## Integration Points

### OpenLineage

All pipeline runs emit OpenLineage events:

- **Marquez**: Default lineage backend
- **Custom backends**: Configurable OpenLineage endpoint
- **Events**: START, RUNNING, COMPLETE, FAIL, ABORT

### Infrastructure: Stores & Connectors

Data packages reference infrastructure through **Stores** (named instances of **Connectors**):

```yaml
# store/warehouse.yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
spec:
  connector: postgres
  connection:
    host: dk-postgres-postgresql.dk-local.svc.cluster.local
    port: 5432
    database: dataplatform
  secrets:
    username: ${PG_USER}
    password: ${PG_PASSWORD}
```

Stores are resolved per environment, allowing the same Transform to reference different infrastructure in dev vs. prod.

## Security Model

| Layer | Mechanism |
|-------|-----------|
| **Authentication** | OIDC/OAuth for CLI, service accounts for automation |
| **Authorization** | RBAC for environments and namespaces |
| **Artifact Integrity** | OCI signatures with Sigstore |
| **Data Classification** | Manifest-declared PII and sensitivity levels |
| **Audit** | OpenLineage events + Kubernetes audit logs |

## Next Steps

- [Data Packages](data-packages.md) - Deep dive into package structure
- [Manifests](manifests.md) - Manifest schema reference
- [Lineage](lineage.md) - Understanding data lineage
