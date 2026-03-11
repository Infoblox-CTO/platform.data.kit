# K8s DatabaseClaim Analysis Pipeline

This example demonstrates a three-stage data pipeline that extracts Kubernetes
metadata from multiple clusters and produces a reporting table showing which
Helm releases use the DatabaseClaim custom resource versus those that configure
databases manually.

## Pipeline Stages

```
K8s Clusters → [k8s-collector] → S3 Parquet → [s3-to-postgres] → PostgreSQL → [dbt-reporting] → Report
```

| Transform | Runtime | Trigger | Description |
|-----------|---------|---------|-------------|
| **k8s-collector** | cloudquery | schedule (4h) | Extract Helm releases, Deployments, ConfigMaps, Secrets, and DatabaseClaims from K8s clusters to S3 as Parquet |
| **s3-to-postgres** | cloudquery | on-change | Load Parquet files from S3 into PostgreSQL staging tables, parse database connection indicators |
| **dbt-reporting** | dbt | on-change | Transform staging tables into a reporting view classifying each release |

## DataSets

### Raw (S3 Parquet)

| Dataset | Description |
|---------|-------------|
| `helm-releases` | Helm release metadata per cluster/namespace |
| `k8s-deployments` | Deployment specs with image and env vars |
| `k8s-configmaps` | ConfigMap key listings with DB config detection |
| `k8s-secrets` | Secret type and key listings (no values) |
| `database-claims` | DatabaseClaim custom resource status |

### Staging (PostgreSQL)

| Dataset | Description |
|---------|-------------|
| `stg-helm-releases` | Cleaned and deduplicated Helm data |
| `stg-database-claims` | Normalized DatabaseClaim data |
| `stg-db-indicators` | Parsed DB connection signals per release |
| `db-usage-indicators` | Raw indicator rows from ConfigMaps/Secrets/env vars |

### Reporting (PostgreSQL)

| Dataset | Description |
|---------|-------------|
| `dbclaim-report` | Final analysis: each release classified as managed/unmanaged/no-db |

## View the Pipeline Graph

```bash
# Text tree
dk pipeline show --scan-dir ./examples/k8s-dbclaim-pipeline

# Mermaid diagram
dk pipeline show --scan-dir ./examples/k8s-dbclaim-pipeline --output mermaid

# JSON format
dk pipeline show --scan-dir ./examples/k8s-dbclaim-pipeline --output json
```

## Assessment Categories

The `dbclaim-report` classifies each Helm release:

- **managed** — Has a DatabaseClaim CR provisioning its database
- **unmanaged** — Shows database connection indicators (env vars, ConfigMap keys,
  Secret keys matching DB patterns) but no DatabaseClaim
- **no-db** — No database usage signals detected

## See Also

- [Vision Document](../../docs/vision/k8s-dbclaim-pipeline.md) — Full reference architecture
- [Reactive Pipeline Example](../reactive-pipeline/) — Simpler three-transform pipeline
