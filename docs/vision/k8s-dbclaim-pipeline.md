# Vision: K8s DatabaseClaim Analysis Pipeline

> Reference architecture showing how DataKit handles a real-world multi-stage
> data pipeline — from Kubernetes cluster extraction through dbt reporting.

## Problem

Teams deploy applications to Kubernetes. Some use the **DatabaseClaim** custom
resource to provision managed databases; others configure database connections
manually via environment variables, ConfigMaps, or Secrets. There is no single
view of database usage across the fleet.

**Goal:** Build a pipeline that extracts K8s metadata from multiple clusters,
normalizes it into PostgreSQL, and uses dbt to produce reporting tables that
answer:

- Which Helm releases use DatabaseClaim?
- Which do not, but still reference database URLs or credentials?
- What is the overall database adoption posture per cluster, namespace, team?

## Architecture

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  K8s Cluster │  │  K8s Cluster │  │  K8s Cluster │
│   us-east-1  │  │   us-west-2  │  │   eu-west-1  │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └────────┬────────┴────────┬────────┘
                │                 │
                ▼                 │
   ┌─────────────────────┐       │
   │   k8s-collector      │◄──────┘
   │   (CloudQuery)       │
   │   runtime: cloudquery│
   └──────────┬──────────┘
              │ writes Parquet
              ▼
   ┌─────────────────────┐
   │   S3 (s3-raw store) │
   │   helm-releases/     │
   │   k8s-deployments/   │
   │   k8s-configmaps/    │
   │   k8s-secrets/       │
   │   database-claims/   │
   └──────────┬──────────┘
              │ reads Parquet
              ▼
   ┌─────────────────────┐
   │   s3-to-postgres     │
   │   (CloudQuery)       │
   │   runtime: cloudquery│
   └──────────┬──────────┘
              │ writes rows
              ▼
   ┌─────────────────────┐
   │  PostgreSQL          │
   │  (pg-warehouse)      │
   │  raw.helm_releases   │
   │  raw.k8s_deployments │
   │  raw.k8s_configmaps  │
   │  raw.k8s_secrets     │
   │  raw.database_claims │
   └──────────┬──────────┘
              │ reads raw tables
              ▼
   ┌─────────────────────┐
   │   dbt-reporting      │
   │   (dbt)              │
   │   runtime: dbt       │
   └──────────┬──────────┘
              │ writes reporting tables
              ▼
   ┌─────────────────────┐
   │  PostgreSQL          │
   │  reporting.dbclaim_  │
   │    report            │
   └─────────────────────┘
```

## Project Directory Layout

```
k8s-dbclaim-pipeline/
├── connectors/
│   ├── k8s-cluster/
│   │   └── connector.yaml        # K8s source plugin definition
│   ├── s3-datalake/
│   │   └── connector.yaml        # S3 source+destination plugin
│   └── postgres-analytics/
│       └── connector.yaml        # PostgreSQL source+destination plugin
│
├── stores/
│   ├── s3-raw/
│   │   └── store.yaml            # S3 bucket for raw Parquet landing
│   └── pg-warehouse/
│       └── store.yaml            # PostgreSQL analytics database
│
├── datasets/
│   ├── helm-releases/
│   │   └── dk.yaml               # S3 Parquet: Helm release metadata
│   ├── k8s-deployments/
│   │   └── dk.yaml               # S3 Parquet: Deployment specs
│   ├── k8s-configmaps/
│   │   └── dk.yaml               # S3 Parquet: ConfigMap metadata
│   ├── k8s-secrets/
│   │   └── dk.yaml               # S3 Parquet: Secret metadata (no values)
│   ├── database-claims/
│   │   └── dk.yaml               # S3 Parquet: DatabaseClaim CRs
│   ├── db-usage-indicators/
│   │   └── dk.yaml               # PG table: parsed DB connection signals
│   ├── stg-helm-releases/
│   │   └── dk.yaml               # PG staging: cleaned Helm data
│   ├── stg-database-claims/
│   │   └── dk.yaml               # PG staging: cleaned DatabaseClaim data
│   ├── stg-db-indicators/
│   │   └── dk.yaml               # PG staging: cleaned indicator data
│   └── dbclaim-report/
│       └── dk.yaml               # PG reporting: final analysis table
│
├── transforms/
│   ├── k8s-collector/
│   │   └── dk.yaml               # CQ: K8s clusters → S3 Parquet
│   ├── s3-to-postgres/
│   │   └── dk.yaml               # CQ: S3 Parquet → PG raw tables
│   └── dbt-reporting/
│       ├── dk.yaml               # dbt: PG raw → PG reporting
│       └── models/
│           ├── staging/
│           │   ├── stg_helm_releases.sql
│           │   ├── stg_database_claims.sql
│           │   └── stg_db_indicators.sql
│           └── marts/
│               └── dbclaim_report.sql
│
├── dk.lock                        # Pinned schema versions (if using schemaRef)
└── README.md
```

## Key Manifests

### Connectors

```yaml
# connectors/k8s-cluster/connector.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: k8s-cluster
spec:
  provider: kubernetes
  type: kubernetes
  capabilities: [source]
  plugin:
    source: ghcr.io/cloudquery/cq-source-k8s:latest
```

```yaml
# connectors/s3-datalake/connector.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: s3-datalake
spec:
  provider: s3
  type: s3
  capabilities: [source, destination]
  plugin:
    source: ghcr.io/cloudquery/cq-source-s3:latest
    destination: ghcr.io/cloudquery/cq-destination-s3:latest
```

```yaml
# connectors/postgres-analytics/connector.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: postgres-analytics
spec:
  provider: postgres
  type: postgres
  capabilities: [source, destination]
  plugin:
    source: ghcr.io/cloudquery/cq-source-postgres:latest
    destination: ghcr.io/cloudquery/cq-destination-postgres:latest
```

### Stores

```yaml
# stores/s3-raw/store.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: s3-raw
spec:
  connector: s3-datalake
  connection:
    bucket: dk-datalake-raw
    region: us-east-1
  secrets:
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
```

```yaml
# stores/pg-warehouse/store.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: pg-warehouse
spec:
  connector: postgres-analytics
  connection:
    host: analytics-pg.internal
    port: 5432
    database: warehouse
  secrets:
    username: ${PG_USER}
    password: ${PG_PASS}
```

### DataSets (representative samples)

```yaml
# datasets/helm-releases/dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: helm-releases
  version: "1.0.0"
  labels:
    domain: k8s-inventory
    tier: raw
spec:
  store: s3-raw
  prefix: data/helm-releases/
  format: parquet
  classification: internal
  schema:
    - name: cluster
      type: string
    - name: namespace
      type: string
    - name: release_name
      type: string
    - name: chart_name
      type: string
    - name: chart_version
      type: string
    - name: app_version
      type: string
    - name: status
      type: string
    - name: values_json
      type: string
    - name: collected_at
      type: timestamp
```

```yaml
# datasets/database-claims/dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: database-claims
  version: "1.0.0"
  labels:
    domain: k8s-inventory
    tier: raw
spec:
  store: s3-raw
  prefix: data/database-claims/
  format: parquet
  classification: internal
  schema:
    - name: cluster
      type: string
    - name: namespace
      type: string
    - name: name
      type: string
    - name: database_name
      type: string
    - name: instance_label
      type: string
    - name: db_type
      type: string
    - name: status
      type: string
    - name: collected_at
      type: timestamp
```

```yaml
# datasets/dbclaim-report/dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: dbclaim-report
  version: "1.0.0"
  labels:
    domain: k8s-inventory
    tier: reporting
spec:
  store: pg-warehouse
  table: reporting.dbclaim_report
  classification: internal
  schema:
    - name: cluster
      type: string
    - name: namespace
      type: string
    - name: release_name
      type: string
    - name: has_dbclaim
      type: boolean
    - name: has_db_indicators
      type: boolean
    - name: indicator_types
      type: string
    - name: db_type
      type: string
    - name: assessment
      type: string
    - name: updated_at
      type: timestamp
```

### Transforms

```yaml
# transforms/k8s-collector/dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: k8s-collector
  version: "0.1.0"
spec:
  runtime: cloudquery
  mode: batch
  inputs: []
  outputs:
    - dataset: helm-releases
    - dataset: k8s-deployments
    - dataset: k8s-configmaps
    - dataset: k8s-secrets
    - dataset: database-claims
  trigger:
    policy: schedule
    schedule:
      cron: "0 */4 * * *"
  timeout: 30m
  resources:
    cpu: "1"
    memory: 2Gi
```

```yaml
# transforms/s3-to-postgres/dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: s3-to-postgres
  version: "0.1.0"
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - dataset: helm-releases
    - dataset: k8s-deployments
    - dataset: k8s-configmaps
    - dataset: k8s-secrets
    - dataset: database-claims
  outputs:
    - dataset: db-usage-indicators
    - dataset: stg-helm-releases
    - dataset: stg-database-claims
    - dataset: stg-db-indicators
  trigger:
    policy: on-change
  timeout: 30m
```

```yaml
# transforms/dbt-reporting/dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: dbt-reporting
  version: "0.1.0"
spec:
  runtime: dbt
  mode: batch
  inputs:
    - dataset: stg-helm-releases
    - dataset: stg-database-claims
    - dataset: stg-db-indicators
    - dataset: db-usage-indicators
  outputs:
    - dataset: dbclaim-report
  image: ghcr.io/infoblox-cto/dk-dbt-reporting:latest
  trigger:
    policy: on-change
  timeout: 1h
```

## Pipeline Graph

Running `dk pipeline show --scan-dir ./examples/k8s-dbclaim-pipeline`:

```
Pipeline: k8s-dbclaim-pipeline

  k8s-collector (cloudquery, schedule: 0 */4 * * *)
    → helm-releases (s3-raw, parquet)
    → k8s-deployments (s3-raw, parquet)
    → k8s-configmaps (s3-raw, parquet)
    → k8s-secrets (s3-raw, parquet)
    → database-claims (s3-raw, parquet)

  s3-to-postgres (cloudquery, trigger: on-change)
    ← helm-releases
    ← k8s-deployments
    ← k8s-configmaps
    ← k8s-secrets
    ← database-claims
    → db-usage-indicators (pg-warehouse)
    → stg-helm-releases (pg-warehouse)
    → stg-database-claims (pg-warehouse)
    → stg-db-indicators (pg-warehouse)

  dbt-reporting (dbt, trigger: on-change)
    ← stg-helm-releases
    ← stg-database-claims
    ← stg-db-indicators
    ← db-usage-indicators
    → dbclaim-report (pg-warehouse, reporting.dbclaim_report)
```

## Developer Lifecycle

### 1. Local Development

```bash
# Scaffold (future: dk project init)
mkdir k8s-dbclaim-pipeline && cd k8s-dbclaim-pipeline

# Bring up local infrastructure
dk dev up                  # k3d + LocalStack S3 + PostgreSQL + Marquez

# Seed test data
dk dev seed                # Loads fixture data into local stores

# Iterate on transforms
dk lint ./transforms/k8s-collector
dk run  ./transforms/k8s-collector

# View the full pipeline
dk pipeline show --scan-dir .

# Run all transforms in order (future: dk run --project .)
dk run ./transforms/k8s-collector
dk run ./transforms/s3-to-postgres
dk dbt run --dir ./transforms/dbt-reporting
```

### 2. CI Pipeline

```bash
# Validate everything
dk lint --scan-dir .            # Future: project-wide lint
dk lock --verify                # Ensure dk.lock is up-to-date
dk test ./transforms/k8s-collector
dk test ./transforms/s3-to-postgres

# Build artifacts
dk build ./transforms/k8s-collector
dk build ./transforms/s3-to-postgres
dk build ./transforms/dbt-reporting

# Publish to registry
dk publish ./transforms/k8s-collector
dk publish ./transforms/s3-to-postgres
dk publish ./transforms/dbt-reporting
```

### 3. Promotion (Dev → Int → Prod)

```bash
# Promote all transforms to dev
dk promote k8s-collector v0.1.0 --to dev
dk promote s3-to-postgres v0.1.0 --to dev
dk promote dbt-reporting v0.1.0 --to dev

# After dev validation, promote to int
dk promote k8s-collector v0.1.0 --to int

# Promote to prod (specific cell)
dk promote k8s-collector v0.1.0 --to prod --cell canary
```

### 4. Environment Resolution

Stores resolve differently per environment via cell-based resolution:

| Store | Dev (local) | Int | Prod |
|-------|-------------|-----|------|
| `s3-raw` | LocalStack `localhost:4566` | `s3://int-datalake/` | `s3://prod-datalake/` |
| `pg-warehouse` | Local PG `localhost:5432` | `int-pg.internal:5432` | `analytics-pg.internal:5432` |

The pipeline definition never changes between environments. Only Store connection
details and secrets differ, resolved at deploy time from the target cell's
Kubernetes namespace (`dk-<cellName>`).

## Status

| Capability | Status | Notes |
|-----------|--------|-------|
| Transform manifests (CloudQuery) | Done | `dk init`, `dk run`, `dk build` |
| DataSet manifests | Done | `dk dataset create`, inline schema |
| Schema references (APX) | Done | `schemaRef`, `dk lock`, `dk schema` |
| Pipeline graph visualization | Done | `dk pipeline show --scan-dir` |
| Local dev environment | Done | `dk dev up/seed` |
| Cell-based promotion | Done | `dk promote --to {env} --cell {cell}`, shared dk-app chart, ArgoCD git generator |
| Cell-based Store resolution | Done | `dk run --cell canary`, DSN env vars |
| dbt runtime | Done | `dk dbt run/test/debug`, Python SDK auto-generates profiles.yml from Store DSN |
| Store → runtime bridge | Done | `DK_STORE_DSN_*` / `DK_STORE_TYPE_*` env vars for all runtimes |
| Python SDK | Done | `datakit-sdk` package: `datakit.stores`, `datakit.profiles`, `dk-profiles` CLI |
| **Python SDK publishing** | **Future** | `datakit-sdk` needs to be published to PyPI so dbt Docker images can `pip install datakit-sdk` |
| **Go SDK for stores** | **Future** | Go equivalent of Python SDK's `stores.get()` — reads `DK_STORE_DSN_*` env vars for generic-go transforms |
| **`dk test` dbt awareness** | **Future** | `dk test` should detect `runtime: dbt` and run `dk dbt test` automatically |
| **Project scaffolding** | **Future** | No `dk project init` for multi-transform projects |
| **Store/Connector scaffolding** | **Future** | No `dk store create` or `dk connector create` |
| **Project-wide lint** | **Partial** | `dk lint --scan-dir .` exists but doesn't cross-validate across transforms |
| **Multi-transform run** | **Future** | `dk run` executes one transform at a time |
| **Real dk status** | **Future** | Returns hardcoded data |
| **On-change triggers** | **Future** | Types defined, no event system |
| **Declarative policies** | **Future** | No policy engine yet |

## dbt Model Examples

The dbt transform would contain SQL models that clean, join, and aggregate the
raw data into the reporting shape.

**Staging models** clean and normalize raw data:

```sql
-- models/staging/stg_helm_releases.sql
-- Deduplicate and clean Helm release data
-- Extracts chart metadata and normalizes status values
```

```sql
-- models/staging/stg_database_claims.sql
-- Clean DatabaseClaim CR data
-- Normalize status and extract instance labels
```

```sql
-- models/staging/stg_db_indicators.sql
-- Parse ConfigMaps and Secrets for database connection signals
-- Look for patterns: DATABASE_URL, PGHOST, DB_HOST, jdbc:, etc.
```

**Mart models** produce the final reporting table:

```sql
-- models/marts/dbclaim_report.sql
-- Join Helm releases with DatabaseClaims and DB indicators
-- Classify each release: has_dbclaim, has_db_indicators, assessment
-- Assessment categories: "managed" (has DBClaim), "unmanaged" (has DB
-- indicators but no DBClaim), "no-db" (no database signals detected)
```
