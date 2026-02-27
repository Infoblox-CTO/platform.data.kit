---
title: Architecture
description: High-level architecture of the Data Platform
---

# DP Architecture

This document describes the high-level architecture of the Data Platform (DP).

## Overview

DP is a Kubernetes-native data pipeline platform that enables teams to contribute reusable, versioned "data packages" with a complete developer workflow.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Data Platform Architecture                          │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌──────────────────┐                                                            │
│  │    Developer     │                                                            │
│  └────────┬─────────┘                                                            │
│           │                                                                      │
│           ▼                                                                      │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                              DP CLI                                       │   │
│  │  ┌──────┐ ┌─────┐ ┌─────┐ ┌──────┐ ┌───────┐ ┌─────────┐ ┌─────────┐    │   │
│  │  │ init │ │ dev │ │ run │ │ lint │ │ build │ │ publish │ │ promote │    │   │
│  │  └──────┘ └─────┘ └─────┘ └──────┘ └───────┘ └─────────┘ └─────────┘    │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│           │                    │                       │                         │
│           │                    │                       │                         │
│           ▼                    ▼                       ▼                         │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐              │
│  │       SDK        │  │   OCI Registry   │  │     GitOps       │              │
│  │  • Validate      │  │  • Store Pkgs    │  │  • Kustomize     │              │
│  │  • Lineage       │  │  • Immutability  │  │  • ArgoCD        │              │
│  │  • Registry      │  │  • Versioning    │  │  • Environments  │              │
│  │  • Runner        │  │                  │  │                  │              │
│  │  • Catalog       │  │                  │  │                  │              │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘              │
│                                                         │                        │
│                                                         ▼                        │
│                              ┌──────────────────────────────────────────┐       │
│                              │         Kubernetes Cluster               │       │
│                              │  ┌────────────────────────────────────┐  │       │
│                              │  │     Platform Controller            │  │       │
│                              │  │  • PackageDeployment CRD           │  │       │
│                              │  │  • Pull OCI Artifacts              │  │       │
│                              │  │  • Create Jobs                     │  │       │
│                              │  │  • Emit Metrics                    │  │       │
│                              │  └────────────────────────────────────┘  │       │
│                              └──────────────────────────────────────────┘       │
│                                                                                  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. CLI (`cli/`)

The command-line interface for interacting with the platform.

**Responsibilities:**
- Package scaffolding (`dp init`)
- Local development (`dp dev`, `dp run`)
- Validation (`dp lint`, `dp test`)
- Publishing (`dp build`, `dp publish`)
- Promotion (`dp promote`)
- Observability (`dp status`, `dp logs`)

**Technology:** Go, Cobra

### 2. SDK (`sdk/`)

Core libraries used by the CLI and controller.

#### 2.1 Validate (`sdk/validate/`)
- Manifest validation (dp.yaml, connector, store, asset manifests)
- PII classification validation
- Schema validation

#### 2.2 Lineage (`sdk/lineage/`)
- OpenLineage event types
- Marquez emitter implementation
- Event builder pattern

#### 2.3 Registry (`sdk/registry/`)
- OCI artifact management using ORAS
- Bundler for creating artifacts
- Client for push/pull operations

#### 2.4 Runner (`sdk/runner/`)
- Local execution via Docker
- Lineage emission integration
- Run tracking

#### 2.5 Catalog (`sdk/catalog/`)
- Data catalog record types
- Marquez integration
- Metadata management

### 3. Contracts (`contracts/`)

Shared types and schemas for the five manifest kinds.

```go
// Transform is a unit of computation that reads/writes Assets.
type Transform struct {
    APIVersion string
    Kind       string
    Metadata   TransformMetadata
    Spec       TransformSpec
}

// AssetRef is a reference to a named Asset.
type AssetRef struct {
    Asset   string            // Asset name (mutually exclusive with Tags)
    Tags    map[string]string // Match assets by labels
    Version string            // Semver range constraint
    Cell    string            // Cell qualifier
}

// AssetManifest represents a data contract in a Store.
type AssetManifest struct {
    APIVersion string
    Kind       string
    Metadata   AssetMetadata
    Spec       AssetSpec
}
```

### 4. Platform Controller (`platform/controller/`)

Kubernetes controller for managing data packages.

**CRDs:**
- `PackageDeployment`: Represents a deployed data package

**Reconciliation Loop:**
1. Watch for PackageDeployment changes
2. Pull OCI artifact from registry
3. Extract pipeline configuration
4. Create/update Kubernetes Jobs
5. Monitor execution and emit metrics

### 5. GitOps (`gitops/`)

Environment definitions using Kustomize.

```
gitops/
├── base/
│   ├── kustomization.yaml
│   └── crds/
├── environments/
│   ├── dev/
│   │   └── kustomization.yaml
│   ├── int/
│   │   └── kustomization.yaml
│   └── prod/
│       └── kustomization.yaml
└── argocd/
    └── applicationset.yaml
```

## Data Flow

### Local Development

```
Developer → dp init → Creates dp.yaml
         → dp dev up → Deploys embedded Helm charts to k3d
                        (Redpanda, LocalStack, PostgreSQL, Marquez)
                        Init jobs seed topics, buckets, schemas, namespaces
         → dp run → Builds container, runs locally
         → Lineage events → Marquez
```

#### Helm Chart Deployment Mechanism

The `dp dev up` command uses a uniform Helm chart deployment mechanism:

1. **Embedded Charts**: All dev dependency charts are embedded in the CLI binary via Go's `embed.FS` (`sdk/localdev/charts/`)
2. **Chart Registry**: A `DefaultCharts` registry defines each chart's port-forwarding rules, health labels, display endpoints, and timeouts
3. **Uniform Deployment**: `charts.DeployCharts()` extracts charts to a temp directory and runs `helm upgrade --install` in parallel
4. **Init Jobs**: Each chart includes Helm hook jobs (post-install/post-upgrade) that automatically create required resources (Kafka topics, S3 buckets, DB schemas, Marquez namespaces)
5. **Config Overrides**: Users can override chart versions (`dev.charts.<name>.version`) or Helm values (`dev.charts.<name>.values.<path>`) via the hierarchical config system
6. **Upstream Subcharts**: Redpanda and PostgreSQL wrap upstream Helm charts as subcharts, inheriting production-quality templates while providing dev-appropriate value overrides

Adding a new dev dependency requires only:

- Creating a chart directory under `sdk/localdev/charts/<name>/`
- Registering a `ChartDefinition` in the `DefaultCharts` slice in `embed.go`
- No changes to deployment, health-checking, port-forwarding, or CLI code

### Publish & Promote

```
Developer → dp build → Validates & bundles artifact
         → dp publish → Pushes to OCI registry (digest-based)
         → dp promote → Creates PR to gitops repo
         → PR merged → ArgoCD syncs
         → Controller → Pulls artifact, creates Job
```

### Lineage Tracking

```
Runner emits OpenLineage events:
  START → Job begins execution
  COMPLETE → Job finished successfully  
  FAIL → Job failed with error

Events sent to:
  Marquez (local) → http://localhost:5000/api/v1/lineage
  Marquez (prod) → Configured via environment
```

## Key Design Decisions

### 1. OCI for Package Storage

**Rationale:** 
- Immutable by design (content-addressable)
- Existing tooling (Docker registries, Harbor)
- Standard format with ecosystem support

### 2. GitOps for Promotion

**Rationale:**
- Auditable change history
- Declarative desired state
- Rollback = git revert
- No direct cluster access needed

### 3. OpenLineage for Lineage

**Rationale:**
- Industry standard
- Marquez integration
- Vendor neutral

### 4. Go Monorepo

**Rationale:**
- Independent versioning per module
- Shared contracts
- Single CI pipeline
- Clear dependency direction

## Dependency Graph

```
                   contracts
                      │
           ┌─────────┴─────────┐
           ▼                   ▼
          sdk          platform/controller
           │
           ▼
          cli
```

## Scaling Considerations

| Aspect | MVP | Scale Target |
|--------|-----|--------------|
| Packages | 10-50 | 500+ |
| Environments | 3 | 10+ |
| Concurrent runs | 10 | 100+ |
| OCI artifact size | <500MB | <1GB |

## Security Model

1. **No Secrets in Code**: All secrets via Kubernetes/external-secrets
2. **PII Metadata**: Required classification on outputs
3. **Immutable Artifacts**: No modification after publish
4. **PR-based Promotion**: Audit trail for all changes
5. **RBAC**: Kubernetes RBAC for controller

## Observability

### Metrics
- `dp_run_total{status,package,namespace}`
- `dp_run_duration_seconds{package,namespace}`
- `dp_controller_reconcile_total{result}`

### Logging
- Structured JSON with slog
- Correlation IDs for tracing
- Levels: DEBUG, INFO, WARN, ERROR

### Dashboards
- Pipeline health: `dashboards/pipeline-health.json`
- Controller metrics: `dashboards/controller.json`
