# Data Platform Kit — Core Model

> The four core concepts that define the data platform's manifest model, replacing the legacy Source / Model / Destination taxonomy.
>
> **Status: Implemented.** This document describes the current production model.

---

## Why Change?

The current model (Source, Destination, Model) is tightly coupled to CloudQuery.
A "Source" is really a CQ plugin — it doesn't generalise to `generic-go` or `generic-python` workloads.
Credentials scatter across Model `config`, Bindings, and extension `configSchema`.
There is no way to express column-level lineage, and `dp run` already ignores half the fields on a Model manifest.

The new model separates **technology**, **infrastructure**, **data contracts**, and **computation** into four distinct concepts with clear ownership boundaries.

---

## Core Concepts

### 1. Connector — *what* technology

A Connector describes a storage technology type: Postgres, S3, Kafka, Snowflake, etc.
It is a catalog entry maintained by the **platform team** and rarely changes.

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  protocol: postgresql
  capabilities: [source, destination]
  # Optional: CQ plugin image for CloudQuery runtime
  plugin:
    source: ghcr.io/infobloxopen/cq-source-postgres:0.1.0
    destination: ghcr.io/cloudquery/cq-destination-postgres:latest
```

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: s3
spec:
  type: s3
  protocol: s3
  capabilities: [destination]
  plugin:
    destination: ghcr.io/infobloxopen/cloudquery-plugin-s3:v7.10.2
```

**Who creates it:** Platform team.
**Lifecycle:** Versioned in the platform repo. Changes when a new technology is onboarded or a plugin is upgraded.

---

### 2. Store — *where* data lives

A Store is a **named instance** of a Connector: a specific database, bucket, or cluster with its connection details and credentials.
Secrets live **only** on the Store — never on Assets or Transforms.

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: default
  labels:
    team: data-platform
spec:
  connector: postgres
  connection:
    host: dp-postgres-postgresql.dp-local.svc.cluster.local
    port: 5432
    database: dataplatform
    schema: public
    sslmode: disable
  secrets:
    username: ${PG_USER}
    password: ${PG_PASSWORD}
```

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: lake-raw
  namespace: default
spec:
  connector: s3
  connection:
    bucket: cdpp-raw
    region: us-east-1
    endpoint: http://dp-localstack-localstack.dp-local.svc.cluster.local:4566
  secrets:
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
```

**Who creates it:** The team that owns the infrastructure (infra / team lead / SRE).
**Lifecycle:** One per logical data system. Updated when credentials rotate or endpoints change.

---

### 3. Asset / AssetGroup — *what* data exists

An Asset is a **named data contract**: a table, S3 prefix, or Kafka topic that lives in a Store.
It declares the schema (with optional column-level lineage via `from`) and data classification.

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
  namespace: default
spec:
  store: warehouse
  table: public.users
  classification: confidential
  schema:
    - name: id
      type: integer
    - name: email
      type: string
      pii: true
    - name: created_at
      type: timestamp
```

An output Asset uses `from` to declare where each column originates — this gives you **column-level lineage for free**:

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users-parquet
  namespace: default
spec:
  store: lake-raw
  prefix: data/users/
  format: parquet
  classification: confidential
  schema:
    - name: id
      type: integer
      from: users.id
    - name: email
      type: string
      pii: true
      from: users.email
    - name: created_at
      type: timestamp
      from: users.created_at
```

An **AssetGroup** bundles multiple Assets produced by a single materialisation (e.g. a CQ sync that snapshots several tables):

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: AssetGroup
metadata:
  name: pg-snapshot
  namespace: default
spec:
  store: lake-raw
  assets:
    - users-parquet
    - orders-parquet
    - products-parquet
```

**Who creates it:** Data engineer.
**Lifecycle:** One per logical dataset. Updated when schema evolves.

---

### 4. Transform — *how* data moves

A Transform is a unit of computation that reads input Assets and produces output Assets.
It carries the runtime, mode, schedule, and timeout — everything about *execution*.

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  namespace: default
  version: 0.1.0
  labels:
    team: data-platform
spec:
  runtime: cloudquery        # cloudquery | generic-go | generic-python | dbt
  mode: batch                # batch | streaming

  inputs:
    - asset: users           # → resolves to Store "warehouse" → Connector "postgres"

  outputs:
    - asset: users-parquet   # → resolves to Store "lake-raw" → Connector "s3"

  schedule:
    cron: "0 */6 * * *"

  timeout: 30m
```

For `generic-go` / `generic-python` runtimes, add image + env:

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: enrich-users
  namespace: default
  version: 0.2.0
spec:
  runtime: generic-python
  mode: batch

  inputs:
    - asset: users-parquet

  outputs:
    - asset: users-enriched

  image: my-team/enrich-users:latest
  timeout: 15m
```

**Who creates it:** Data engineer.
**Lifecycle:** One per pipeline step. Updated when logic, schedule, or dependencies change.

---

## How It All Fits Together

```
Platform Team          Infra Owner           Data Engineer
─────────────         ───────────           ──────────────
Connector             Store                 Asset
  postgres  ◄────────  warehouse            users (in warehouse)
  s3        ◄────────  lake-raw             users-parquet (in lake-raw)
                                            │
                                            Transform
                                              pg-to-s3
                                              inputs:  [users]
                                              outputs: [users-parquet]
```

At execution time the runner resolves the chain:

```
Transform "pg-to-s3"
  → input Asset "users"        → Store "warehouse"  → Connector "postgres"  → plugin image + connection
  → output Asset "users-parquet" → Store "lake-raw"   → Connector "s3"       → plugin image + connection
```

For the **CloudQuery runtime**, the runner auto-generates `config.yaml` from this graph — no hand-authored CQ config needed.
For **generic-go / generic-python**, the runner injects Store connection details as environment variables.

---

## Comparison With Current Model

| Concern | Current (v1alpha1) | Proposed |
|---|---|---|
| Technology type | Implicit in Source/Destination `runtime` + `image` | Explicit **Connector** |
| Infrastructure instance | Scattered: Model `config`, Bindings YAML | Single **Store** with secrets |
| Data contract | `ArtifactContract` on Model (type + format) | **Asset** with full schema + lineage |
| Column lineage | Not supported | `from` field on Asset schema |
| Computation | **Model** (mixes data refs, config, runtime, schedule) | **Transform** (pure computation) |
| CQ config | Hand-authored `config.yaml` | Auto-generated from Asset → Store → Connector graph |
| Secrets | In Model `config`, Bindings, ConfigSchema | **Only** on Store |

---

## Dagster Equivalence

The model maps 1:1 to Dagster concepts, enabling a future `dp generate dagster` command:

| Platform Kit | Dagster | Notes |
|---|---|---|
| Connector | I/O Manager type | e.g. `S3IOManager`, `PostgresIOManager` |
| Store | Resource instance | `PostgresResource(host=..., password=...)` |
| Asset | `@asset` | With `ins={}` for dependencies |
| AssetGroup | `@multi_asset` | Multiple `AssetOut` from one function |
| Transform | Function body + `@asset` deps | The computation that runs inside an asset |

---

## Example: Full Pipeline Package

A complete pipeline that syncs three Postgres tables to S3 as Parquet:

```
my-pipeline/
├── connector/
│   ├── postgres.yaml
│   └── s3.yaml
├── store/
│   ├── warehouse.yaml
│   └── lake-raw.yaml
├── asset/
│   ├── users.yaml
│   ├── orders.yaml
│   ├── products.yaml
│   ├── users-parquet.yaml
│   ├── orders-parquet.yaml
│   └── products-parquet.yaml
├── asset-group/
│   └── pg-snapshot.yaml
└── transform/
    └── pg-to-s3.yaml          # runtime: cloudquery, inputs: [users, orders, products]
                                # outputs: [users-parquet, orders-parquet, products-parquet]
```

Running `dp run` in this directory:

1. Parses the Transform and resolves its input/output Assets
2. Looks up each Asset's Store and its Store's Connector
3. For CloudQuery runtime: auto-generates `config.yaml` with source + destination plugin configs, injects Store credentials
4. Builds a k8s Job with native sidecar containers (source plugin, destination plugin, CQ orchestrator)
5. Polls for completion and reports sync results

No Bindings file. No hand-authored `config.yaml`. No Source/Destination extension registry lookups.

---

## Packaging & Deployment

### What is the deployable artifact?

A **Transform** is the only concept that *runs*. Connectors, Stores, and Assets are declarative metadata.
The deployable unit is a **Transform package** — the Transform manifest plus its declared output Assets.

### Where things live

```
PLATFORM REPO (shared, maintained by platform team)
├── connectors/
│   ├── postgres.yaml              ← technology catalog
│   └── s3.yaml
└── stores/                        ← can also be per-team repos
    ├── warehouse.yaml
    └── lake-raw.yaml

TEAM A's PACKAGE (deployable unit — "pg-to-s3")
├── transform.yaml                 ← runtime: cloudquery, schedule, timeout
├── assets/
│   ├── users.yaml                 ← input (declares "I read this")
│   └── users-parquet.yaml         ← output (my contract to consumers)
└── (no code — CQ plugins are image refs on Connectors)

TEAM B's PACKAGE (depends on Team A's output — "enrich-users")
├── transform.yaml                 ← runtime: generic-python
├── assets/
│   └── users-enriched.yaml        ← output
├── src/
│   └── main.py
└── Dockerfile
```

Team B references `users-parquet` as an input. That Asset was *declared* by Team A and published to the registry.
Team B doesn't deploy Team A's code — they declare a dependency on Team A's **output Asset** (the data contract).

---

## OCI Registry

Transform packages and Asset declarations are distributed as **OCI artifacts** — the same registries that host container images (GHCR, ECR, ACR, Artifactory).

### Artifact layout

```
ghcr.io/my-org/transforms/pg-to-s3:0.1.0         ← OCI artifact
├── layer 0: transform.yaml                        mediaType: application/vnd.dp.transform.v1+yaml
├── layer 1: assets/users.yaml                     mediaType: application/vnd.dp.asset.v1+yaml
├── layer 2: assets/users-parquet.yaml             mediaType: application/vnd.dp.asset.v1+yaml
└── (for generic-go/python only)
    └── layer 3: image ref                         mediaType: application/vnd.oci.image.manifest.v1+json
```

- **CloudQuery transforms** are manifest-only — no code, just YAML layers. Tiny artifacts.
- **generic-go/python transforms** include a reference to (or embed) the built container image.
- **Output Assets** are also published as standalone artifacts for cross-team discovery:

```
ghcr.io/my-org/assets/users-parquet:0.1.0         ← just the asset declaration
└── layer 0: asset.yaml                            mediaType: application/vnd.dp.asset.v1+yaml
```

### CLI workflow

```bash
# Team A publishes their transform + output assets
dp build                        # generic-go/python: builds container image
                                # cloudquery: validates manifests (no build needed)

dp publish                      # pushes OCI artifact to registry
                                # → ghcr.io/my-org/transforms/pg-to-s3:0.1.0
                                # → ghcr.io/my-org/assets/users-parquet:0.1.0  (auto-published)

# Team B depends on Team A's output asset
# In their transform.yaml:
#   inputs:
#     - asset: ghcr.io/my-org/assets/users-parquet:0.1.0
dp run                          # pulls asset manifest from OCI, resolves Store → Connector

# Production deployment
# PackageDeployment CRD references the OCI artifact:
#   spec.package.image: ghcr.io/my-org/transforms/pg-to-s3:0.1.0
```

### Why OCI

1. **Already have one.** Every org running containers has an OCI registry. No new infra.
2. **Auth is solved.** Image pull secrets, OIDC, robot accounts — all existing patterns.
3. **Versioning is native.** Tags, SemVer, digest pinning — `pg-to-s3:0.1.0`, `pg-to-s3@sha256:abc...`.
4. **Signing is free.** Sigstore/cosign works on OCI artifacts for supply chain security.
5. **Go libraries exist.** [oras-project/oras-go](https://github.com/oras-project/oras-go) and [google/go-containerregistry](https://github.com/google/go-containerregistry) are production-grade.
6. **Precedent.** Helm charts, Flux sources, Crossplane packages, and Carvel bundles all use OCI registries for non-image YAML distribution.

### Asset discovery

Two complementary mechanisms:

| Approach | How it works | Best for |
|---|---|---|
| **Convention-based paths** | Assets live at `<registry>/<org>/assets/<name>:<version>`. `dp search` lists them. | Simple orgs, small teams |
| **Catalog index** | A lightweight manifest listing all published assets, updated on each `dp publish`. Stored as `ghcr.io/<org>/dp-catalog:latest`. | Large orgs, many teams |

The catalog index is analogous to a Helm repo `index.yaml`, but stored in OCI instead of behind an HTTP server.

### Production deployment flow

```
dp publish
    │
    ▼
OCI Registry (ghcr.io/my-org/transforms/pg-to-s3:0.1.0)
    │
    ▼
PackageDeployment CRD applied (GitOps / ArgoCD)
    │
    ▼
Controller reconciles:
    1. Pulls Transform manifest from OCI
    2. Resolves input/output Assets → Stores → Connectors
    3. Injects Store secrets from cluster secret store (Vault, k8s Secrets)
    4. Batch → CronJob  |  Streaming → Deployment
    5. Reports status on PackageDeployment .status
```
