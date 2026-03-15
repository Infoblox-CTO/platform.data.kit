---
title: Target State — Operating Model
description: Two-persona architecture for DataKit
---

# Target State — Operating Model

## Two personas, one platform

DataKit serves two distinct roles with clear ownership boundaries.

| Persona | Owns | Defines |
|---------|------|---------|
| **Platform engineer** | Connectors, cells, stores, policies | *What infrastructure is available* |
| **Data engineer** | Transforms, DataSets, dbt models | *What runs and what it produces* |

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Platform Engineer                                 │
│                                                                             │
│   connector/                   gitops/envs/             policies/           │
│   ├── postgres.yaml            ├── dev/cells/c0/         (future)           │
│   ├── s3.yaml                  ├── int/cells/c0/                            │
│   └── kafka.yaml               └── prod/cells/{c0,canary}/                 │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                            Data Engineer                                    │
│                                                                             │
│   store/                       dataset/                 transforms/         │
│   ├── warehouse.yaml           ├── users.yaml           └── my-pipeline/    │
│   └── lake-raw.yaml            ├── users-parquet.yaml       ├── dk.yaml    │
│                                └── orders.yaml              └── models/    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Four manifest kinds

The platform separates concerns into four manifest kinds. Each references the one above it by name — never by embedding its internals.

```
Connector ──▶ Store ──▶ DataSet ──▶ Transform
  (type)      (instance)  (contract)  (compute)
```

### 1. Connectors — technology types

A Connector is a **technology type definition** maintained by the platform team. It declares what a storage technology *is* — Postgres, S3, Kafka, etc. — and which plugin images to use.

```yaml
# connector/postgres.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  provider: postgres
  capabilities: [source, destination]
```

Connectors rarely change. They define the technology catalog available to the platform.

### 2. Stores — infrastructure instances

A **Store** is a named instance of a Connector with connection details and credentials. Stores live in cells — the same logical Store name resolves to different physical infrastructure depending on where the transform runs.

```yaml
# store/warehouse.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: warehouse
spec:
  connector: postgres
  connection:
    connection_string: "postgresql://dkuser:dkpassword@localhost:5432/datakit"
```

At runtime, stores are presented to transforms as DSN environment variables:

```
DK_STORE_DSN_WAREHOUSE=postgresql://dkuser:dkpassword@localhost:5432/datakit
DK_STORE_TYPE_WAREHOUSE=postgres
```

### 3. DataSets — data contracts

A **DataSet** is a named data contract — a table, S3 prefix, or Kafka topic that lives in a Store — created by data engineers. DataSets declare schema, classification, and the Store they belong to.

```yaml
# dataset/users.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: users
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
```

### 4. Transforms — compute units

A **Transform** declares inputs, outputs, runtime, and mode. The dependency graph is built automatically from the input/output DataSet references — there is no separate pipeline manifest.

```yaml
# dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: my-pipeline
spec:
  runtime: cloudquery     # cloudquery | generic-go | generic-python | dbt
  mode: batch             # batch | streaming
  inputs:
    - dataset: users
  outputs:
    - dataset: users-parquet
```

```bash
# View the full pipeline graph
dk pipeline show --scan-dir .
```

---

## Store resolution — the connection bridge

Transforms never contain connection strings. Instead, the platform resolves Store manifests at runtime and presents connections as environment variables.

### Local development

`dk run` reads the package's `store/` directory and builds DSN env vars:

```bash
dk run ./my-pipeline
# Resolves: store/warehouse.yaml → DK_STORE_DSN_WAREHOUSE=postgresql://localhost:5432/datakit
```

### Cell-based resolution

`dk run --cell canary` fetches Store CRDs from the cell's Kubernetes namespace instead:

```bash
dk run --cell canary ./my-pipeline
# Resolves: kubectl get store warehouse -n dk-canary → DK_STORE_DSN_WAREHOUSE=postgresql://canary-db/dk_canary
```

### dbt integration

For dbt transforms, the Python SDK (`datakit-sdk`) reads the DSN env vars and generates `profiles.yml` automatically:

```bash
dk dbt run                    # resolves stores → generates profiles.yml → runs dbt
dk dbt test                   # same flow, runs dbt test
```

### generic-go transforms (future)

A Go SDK package (`sdk/stores`) will provide the same interface for Go-based transforms:

```go
import "github.com/Infoblox-CTO/platform.data.kit/sdk/stores"

warehouse := stores.Get("warehouse")  // reads DK_STORE_DSN_WAREHOUSE
fmt.Println(warehouse.DSN)            // "postgresql://..."
fmt.Println(warehouse.Type)           // "postgres"
```

### `dk test` dbt awareness (future)

`dk test` will detect `runtime: dbt` in dk.yaml and automatically run `dk dbt test` instead of the generic test runner. This completes the standard workflow for dbt transforms:

```bash
dk lint → dk run → dk test → dk build → dk publish → dk promote
```

### Production (Kubernetes)

The controller injects `DK_STORE_DSN_*` / `DK_STORE_TYPE_*` env vars from Store CRDs into the Job/Deployment spec. For dbt containers, the Docker entrypoint calls `dk-profiles generate` to create `profiles.yml` before running dbt. The Python SDK (`datakit-sdk`) will be published to PyPI for use in Docker images.

---

## GitOps layout

Deployments are organized by environment and cell:

```
gitops/
├── charts/
│   └── dk-app/                     # Shared Helm chart (all packages)
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│           └── packagedeployment.yaml
├── envs/
│   ├── dev/
│   │   └── cells/
│   │       └── c0/
│   │           ├── stores/          # Cell-specific Store CRDs
│   │           └── apps/            # Per-package values.yaml
│   │               └── my-pipeline/
│   │                   └── values.yaml   # appVersion: v1.0.0
│   ├── int/
│   │   └── cells/c0/{stores,apps}
│   └── prod/
│       └── cells/
│           ├── c0/{stores,apps}
│           └── canary/{stores,apps}
├── crds/
└── argocd/
    └── applicationset.yaml          # Git generator on envs/*/cells/*/apps/*
```

- **One shared chart** (`dk-app`) for all packages — no per-package Helm chart generation
- **`dk promote`** creates a PR that writes `appVersion` to `gitops/envs/{env}/cells/{cell}/apps/{pkg}/values.yaml`
- **ArgoCD** discovers apps via git generator and renders the shared chart with each app's values

### Promotion

```bash
dk promote my-pipeline v1.0.0 --to dev              # default cell c0
dk promote my-pipeline v1.0.0 --to prod --cell canary  # specific cell
dk rollback my-pipeline --to prod --to-version v0.9.0   # rollback
```

---

## Environments and cells

### Environments

| Environment | Approval | Purpose |
|-------------|----------|---------|
| **dev** | Auto-merge | Rapid iteration |
| **int** | Team approval | Integration testing |
| **prod** | Multi-party | Production workloads |

### Cells

A cell is a named deployment target within an environment. The default cell is `c0`. Additional cells (e.g., `canary`, `group1`) enable progressive rollout or workload isolation within the same environment.

Each cell has its own Kubernetes namespace (`dk-{env}-{cell}`), Stores, and deployed packages.

---

## Policies (future)

Declarative YAML policies will enforce guardrails at `dk lint` time:

- **versions.yaml** — prod requires exact version pins; dev allows ranges
- **quality.yaml** — gold-tier outputs require tests and documentation
- **security.yaml** — all outputs require PII classification metadata

---

## Developer workflows

### Platform engineer

```bash
# Define connectors and create cell stores
# Edit connector/ and store/ manifests
dk lint
```

### Data engineer

```bash
dk dataset create users --store warehouse --table public.users
dk init my-pipeline --runtime cloudquery
dk dbt run                    # for dbt transforms
dk lint
dk run
dk pipeline show
dk build
dk publish
dk promote my-pipeline v1.0.0 --to dev
```

---

## RACI summary

| Area | Platform Eng | Data Eng |
|------|:------------:|:--------:|
| Define connectors and technology types | **R/A** | C |
| Store connection details and credentials | **R/A** | I |
| Cell infrastructure and store CRDs | **R/A** | I |
| Dev environment (dk dev up) | **R/A** | C |
| Creating DataSets (data contracts) | C | **R/A** |
| Creating transforms and pipelines | C | **R/A** |
| dbt models, tests, docs | I | **R/A** |
| Promotion workflow and approvals | **R/A** | I |
| Incident response — platform runtime | **R/A** | C |
| Incident response — domain pipelines/models | C | **R/A** |

**Rule of thumb**: platform owns *how it runs safely*; data owns *what runs and what it produces*.

---

## Extension model (future)

The four manifest kinds (Transform, DataSet, Connector, Store) can be extended through a registry in the future:

- **Connectors** can be published as reusable technology type definitions, versioned and shared across teams
- **Stores** can be templated per connector type, with standardized connection schemas
- **DataSets** can reference published schemas via `schemaRef` (already implemented via APX catalog)
- **Transforms** can reference published Connector/Store bundles rather than defining them inline

This would enable an internal marketplace where platform teams publish approved connectors and store templates, and data teams compose them into transforms without needing to understand the underlying infrastructure.

No extension registry exists today. The current model (all manifests co-located in the package) works well for small-to-medium teams. Extension publishing becomes valuable when connector/store definitions need to be shared across many independent teams.

---

## Evolution path

| # | Feature | Status |
|---|---------|--------|
| 011 | Connector and Store system | Done |
| 012 | DataSet data contracts | Done |
| 013 | Pipeline graph | Done |
| 014 | Cell-based promotion (replaces env/binding model) | Done |
| 015 | dbt runtime with Python SDK | Done |
| 016 | Publish Python SDK to PyPI (`pip install datakit-sdk`) | Future |
| 017 | Go SDK for store resolution (`sdk/stores` package for generic-go transforms) | Future |
| 018 | `dk test` dbt awareness (detect `runtime: dbt`, run `dk dbt test`) | Future |
| 019 | Declarative policies | Future |
| 020 | Extension registry | Future |
| 021 | Multi-transform project orchestration | Future |
