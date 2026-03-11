# K8s Report Demo

Collects Kubernetes metadata from the local k3d cluster and loads it into
PostgreSQL for reporting. Runs entirely against the `dk dev` infrastructure.

## Pipeline

```
k3d cluster (k8s API)
    │
    ▼  [k8s-collector]  CloudQuery kubernetes → S3 (parquet)
LocalStack S3 (dk-k8s-report)
    │
    ▼  [s3-to-postgres]  CloudQuery S3 → PostgreSQL
PostgreSQL (reporting.k8s_*)
```

**Tables synced**: `k8s_core_namespaces`, `k8s_apps_deployments`, `k8s_core_services`

## Prerequisites

- `dk dev up` (k3d cluster running with LocalStack + PostgreSQL)
- AWS CLI (for bucket creation in setup)

## Setup

```bash
cd demos/k8s-report
bash setup.sh
```

This applies the `dk-k8s-reader` ServiceAccount + RBAC, creates the S3 bucket,
and adds the `reporting` schema to PostgreSQL.

## Run

```bash
# Stage 1: Collect k8s objects → S3 parquet
dk run demos/k8s-report/transforms/k8s-collector

# Stage 2: Load S3 parquet → PostgreSQL
dk run demos/k8s-report/transforms/s3-to-postgres
```

## Verify

```bash
# Check parquet files in S3
aws --endpoint-url http://localhost:4566 s3 ls s3://dk-k8s-report/ --recursive

# Query reporting tables
psql -h localhost -U dkuser -d datakit \
  -c 'SELECT * FROM reporting.k8s_namespaces LIMIT 10'

psql -h localhost -U dkuser -d datakit \
  -c 'SELECT * FROM reporting.k8s_deployments LIMIT 10'

# Visualize the pipeline graph
dk pipeline show --scan-dir demos/k8s-report
```

## RBAC

The `k8s-collector` transform runs as a Kubernetes Job with the
`dk-k8s-reader` ServiceAccount, which has cluster-wide read access to
core resources (namespaces, pods, services, configmaps, secrets, deployments).

See `k8s/rbac.yaml` for the full ClusterRole definition.
