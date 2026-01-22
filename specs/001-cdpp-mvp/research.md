# Research: CDPP MVP Technical Decisions

**Feature**: 001-cdpp-mvp  
**Date**: 2026-01-22  
**Status**: Complete

## Overview

This document consolidates research findings for the CDPP MVP technical implementation. All NEEDS CLARIFICATION markers from the Technical Context have been resolved.

---

## 1. CLI Framework

### Decision: **Cobra**

### Rationale
- Powers 173,000+ projects including kubectl, Docker CLI, GitHub CLI, Helm
- Excellent subcommand nesting for 10+ commands
- Built-in shell completion, help generation, command suggestions
- Seamless integration with Viper for configuration

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| urfave/cli | Less enterprise adoption, smaller ecosystem |
| kong | Smaller community, less battle-tested |

---

## 2. OCI Artifact Management

### Decision: **oras-go v2**

### Rationale
- CNCF sandbox project for non-container OCI artifacts
- Native Go library with stable v2 API
- Supports custom artifact types, versioning, and immutability

### Key Patterns

**Publishing Artifacts**:
```go
import "oras.land/oras-go/v2"

// Push with custom artifact type and annotations
oras.Push(ctx, target, artifact, opts)
```

**Ensuring Immutability**:
- Use digest-based references for guaranteed immutability
- Application-level check: resolve tag before push, fail if exists
- Registry-level: Enable tag immutability policies (ECR, Harbor)

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| go-containerregistry | Container-image focused, less artifact-centric |
| Custom HTTP client | Reinvents OCI distribution-spec compliance |

---

## 3. Structured CLI Output

### Decision: **`-o/--output` flag with table/json/yaml formats**

### Rationale
Industry standard pattern used by kubectl, gh, terraform:
- Default to human-readable table output
- JSON/YAML for machine consumption and automation
- Consistent flag across all commands

### Implementation Pattern
```go
rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table",
    "Output format: table, json, yaml")
```

---

## 4. Local Development Stack

### Decision: **Docker Compose with LocalStack + Redpanda + PostgreSQL**

### Rationale

| Component | Choice | Why |
|-----------|--------|-----|
| S3 | LocalStack | De facto standard for local AWS emulation |
| Kafka | Redpanda | Kafka API-compatible, single binary, lower resources |
| PostgreSQL | Official image | Simple, direct match to production |
| Orchestration | Docker Compose | Simpler than Kind, faster startup |

### When to Use Kind
- Testing Kubernetes Jobs/CronJobs
- Testing Helm charts
- Integration testing with K8s operators

---

## 5. DAG Orchestration

### Decision: **Dagster with Pipes for external workloads**

### Rationale
Dagster Pipes allows orchestrating external, non-Python workloads:
- Dagster schedules and observes; external code executes
- `PipesK8sClient` invokes Kubernetes Jobs
- Full observability: logs, metadata, asset lineage
- No Dagster dependency in pipeline code

### Key Pattern
```python
from dagster_k8s import PipesK8sClient

@dg.asset
def my_pipeline(context, k8s_client: PipesK8sClient):
    return k8s_client.run(
        context=context,
        image="my-pipeline:v1.0.0",
        command=["./run.sh"],
    ).get_results()
```

### Alternatives Considered

| Alternative | Why Rejected |
|-------------|--------------|
| Airflow | KubernetesOperator works but weaker asset model |
| Argo Workflows | K8s native but less integrated observability |
| Temporal | Workflow engine, not data pipeline orchestrator |

---

## 6. GitOps and Promotion

### Decision: **ArgoCD with Kustomize overlays (MVP: GitHub Actions for promotion)**

### Rationale

**ArgoCD over Flux**:
- ApplicationSets auto-generate apps from folder structure
- Built-in UI for sync status visibility
- Better promotion tool ecosystem (Kargo, Telefonistka)

**Single repo, folder-per-environment**:
```
packages/
  kafka-s3-pipeline/
    overlays/
      dev/
        version.yaml      ← Promotion changes only this file
        bindings.yaml
      integration/
      prod/
```

**Kustomize over Helm for packages**:
- Simpler promotion: copy version.yaml between folders
- Transparent diffing: reviewers see exact YAML
- Helm reserved for third-party charts

**Promotion Workflow**:
- MVP: GitHub Actions workflow creates PR
- Post-MVP: Kargo for automated promotion with gates

**Rollback = promoting previous version** (same mechanism, no special tooling)

---

## 7. Environment Bindings

### Decision: **ConfigMaps + External Secrets Operator**

### Rationale

| Config Type | Solution |
|-------------|----------|
| Non-sensitive (buckets, brokers) | ConfigMap in overlay folder |
| Sensitive (passwords, keys) | ExternalSecret from Vault/AWS |

Packages declare required bindings in `dp.yaml`; validated at deploy time.

---

## 8. Observability Stack

### Decision: **Prometheus + Grafana + OpenLineage/Marquez + slog**

### Metrics (Prometheus)

**Key Metrics**:
- `cdpp_pipeline_runs_total{package, environment, status}`
- `cdpp_pipeline_run_duration_seconds{package, environment}` (histogram)
- `cdpp_pipeline_last_success_timestamp_seconds{package, environment}`

**Cardinality Control**:
- Safe labels: package, environment, status, stage
- Avoid in labels: run_id, version (use info metric pattern)

### Logging (slog)

**Decision**: Go stdlib `log/slog` with JSON output

**Rationale**:
- Part of stdlib (Go 1.21+), no external dependency
- Clean API with context propagation
- Sufficient performance for this use case

**Correlation IDs**: run_id, package, version, environment

**Aggregation**: Fluent Bit DaemonSet → Loki → Grafana

### Lineage (OpenLineage + Marquez)

**Decision**: Adopt OpenLineage standard with Marquez backend

**Event Structure**:
- RunEvent with START/RUNNING/COMPLETE/FAIL states
- Job, Run, and Dataset facets for metadata
- OpenLineage-compatible JSON events

**Marquez**:
- LF AI & Data graduated project
- Web UI, REST + GraphQL APIs
- PostgreSQL storage

### Dashboards

**Platform Dashboard** (operators):
- Success rate, active runs, P99 duration
- Failed runs table with drill-down

**Package Dashboard** (authors):
- Run history, duration heatmap
- Error rate, recent logs

**Alerting Rules**:
- Stale pipeline (no success in 2h)
- High failure rate (>10%)
- Long-running jobs

---

## 9. Technology Stack Summary

| Area | Decision | Key Dependency |
|------|----------|----------------|
| CLI Framework | Cobra | github.com/spf13/cobra |
| OCI Artifacts | oras-go v2 | oras.land/oras-go/v2 |
| Local Stack | Docker Compose | LocalStack, Redpanda, PostgreSQL |
| Orchestration | Dagster Pipes | dagster-k8s |
| GitOps | ArgoCD + Kustomize | argoproj/argo-cd |
| Promotion (MVP) | GitHub Actions | Manual PR workflow |
| Bindings | ConfigMap + ESO | external-secrets/external-secrets |
| Metrics | Prometheus | client_golang |
| Logging | slog (stdlib) | log/slog |
| Lineage | OpenLineage + Marquez | OpenLineage API |
| Dashboards | Grafana | grafana/grafana |

---

## Open Items Resolved

All NEEDS CLARIFICATION markers from Technical Context have been resolved:

| Item | Resolution |
|------|------------|
| Orchestrator choice | Dagster with Pipes |
| GitOps tool | ArgoCD (not Flux) |
| Promotion mechanism | GitHub Actions PR (MVP); Kargo (post-MVP) |
| Lineage format | OpenLineage standard |
| Logging library | Go stdlib slog |
