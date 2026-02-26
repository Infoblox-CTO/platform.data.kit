# Partitioning Model

This document defines how data platform packages are isolated, versioned, and deployed across multiple instances of pipeline infrastructure within a single Kubernetes cluster.

---

## Core Principle

Every deployment is the intersection of two independent dimensions:

```
WHAT runs  ×  WHERE it runs  =  Deployment
(Package)     (Cell)
```

- **Package** — a versioned, immutable artifact containing the Transform, Assets, and Connectors. It defines the computation, the data contracts, and the plugin versions.
- **Cell** — a named infrastructure context providing Stores (connection strings, credentials, namespace). It defines where data lives.

The same package can deploy to many cells. The same cell can host many packages.

---

## Discovery Model

Each concept lives where it naturally belongs:

| Concept | Artifact type | Discovery mechanism | Rationale |
|---|---|---|---|
| **Package** (Transform + Assets + Connectors) | OCI artifact | `dp search`, OCI registry tags | Static, versioned, immutable — fits image registries |
| **Cell** | k8s Custom Resource (cluster-scoped) | `kubectl get cells` | Cluster-level infrastructure — lives where infra lives |
| **Store** | k8s Custom Resource (namespaced to cell) | `kubectl get stores -n dp-canary` | Cell-scoped infrastructure — tied to the cell's namespace |
| **Connector** | Embedded in package (also publishable as OCI) | Part of package, or `dp search --kind connector` | Technology catalog, versioned with the package |

The governing principle: **if it's versioned and immutable, it goes in OCI. If it's infrastructure state, it goes in k8s.**

---

## Concepts

### Package (versioned, immutable)

A package is the unit of development, testing, and publishing. It contains everything about *what* runs:

| Manifest | Role | Example |
|---|---|---|
| **Transform** (`dp.yaml`) | Computation definition: runtime, mode, schedule, inputs/outputs | `runtime: cloudquery`, `inputs: [users]` |
| **Connector** (`connector/*.yaml`) | Technology catalog with plugin image refs | `plugin.source: cq-source-postgresql:v9.0.0` |
| **Asset** (`asset/*.yaml`) | Data contracts: schema, table/prefix/topic, classification | `table: public.users`, `format: parquet` |

A package is identified by name + version: `pg-to-s3:0.2.0`.

When you change a plugin version, add a table, alter a schema field, or modify custom code — that is a new package version. The package is what gets built, tested, published to OCI, and promoted through cells.

### Cell (k8s Custom Resource, cluster-scoped)

A cell is a cluster-scoped Kubernetes resource representing an isolated instance of pipeline infrastructure.

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Cell
metadata:
  name: canary
spec:
  namespace: dp-canary              # k8s namespace for Jobs/Deployments
  labels:
    tier: canary
    region: us-east-1
status:
  ready: true
  storeCount: 2
  packageCount: 3
```

Discovery:
```bash
kubectl get cells                    # list all cells in the cluster
kubectl describe cell canary         # show cell details + deployed packages
dp cell list                         # same, via CLI
```

A cell answers the question: "When an Asset says `store: warehouse`, what actual database does that mean?"

### Store (k8s Custom Resource, namespaced to cell)

Stores are namespaced resources living in their cell's namespace. The same logical Store name exists in every cell but points to different physical infrastructure:

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: dp-canary               # ← belongs to cell:canary
spec:
  connector: postgres
  connection:
    connection_string: "postgresql://canary-db:5432/dp_canary"
  secrets:
    password: ${PG_PASS}             # resolved from k8s Secret in this namespace
```

Discovery:
```bash
kubectl get stores -n dp-canary      # list stores in canary cell
kubectl get stores -n dp-stable      # list stores in stable cell
kubectl get stores --all-namespaces  # list stores across all cells
```

Credentials live only on Stores and are resolved from the cell's namespace (k8s Secrets, Vault, env vars).

### Store Resolution Across Cells

The same logical Store name points to different physical infrastructure per cell:

```
kubectl get store warehouse -n dp-local       → postgresql://localhost:5432/dp_local
kubectl get store warehouse -n dp-canary      → postgresql://canary-db:5432/dp_canary
kubectl get store warehouse -n dp-stable      → postgresql://prod-db:5432/dp_stable
kubectl get store warehouse -n dp-dev-dgarcia → postgresql://localhost:5432/dp_dev_dgarcia
```

---

## Resolution Order

When running a package in a cell, manifests are resolved by these rules:

| Manifest kind | Source | Rationale |
|---|---|---|
| **Transform** | Package | The computation is the thing being versioned |
| **Asset** | Package | Data contracts are part of the package's interface |
| **Connector** | Package | Plugin images are versioned with the code they support |
| **Store** | Cell (if `--cell` set), else package `store/` | Infrastructure varies per cell; package `store/` is the local dev fallback |

```
dp run                     →  Connectors: package    Stores: package      (local dev, self-contained)
dp run --cell canary       →  Connectors: package    Stores: cell:canary  (shared infra, package plugins)
dp run --cell stable       →  Connectors: package    Stores: cell:stable  (production)
```

The package always ships a `store/` directory with local dev defaults so `dp run` works out of the box without any cell configuration.

---

## Cell Lifecycle

### Creating cells

```bash
# Local development — dp dev up manages Cell + Store CRDs in k3d
dp dev up                          # creates k3d cluster + shared infra + cell:local
dp dev up --cell canary            # creates Cell CR + namespace + Stores in cluster
dp dev up --cell dev-dgarcia       # creates developer sandbox cell

# Production — GitOps applies Cell + Store manifests
kubectl apply -f cells/canary/     # Cell CR + Store CRs
```

### Cell + Store CRD layout for GitOps

```
platform-repo/
├── cells/
│   ├── canary/
│   │   ├── cell.yaml              # kind: Cell (cluster-scoped)
│   │   └── stores/
│   │       ├── warehouse.yaml     # kind: Store (namespaced: dp-canary)
│   │       └── lake-raw.yaml
│   ├── stable/
│   │   ├── cell.yaml
│   │   └── stores/
│   └── cell-us-east/
│       ├── cell.yaml
│       └── stores/
└── connectors/                    # optional: blessed connector catalog
    ├── postgres.yaml
    └── s3.yaml
```

### Local development: `dp dev up --cell`

When running locally, `dp dev up --cell canary` does:
1. Creates k8s namespace `dp-canary`
2. Creates database `dp_canary` in the shared PostgreSQL
3. Creates S3 bucket `dp-canary-raw` in LocalStack
4. Creates topic prefix `canary.*` in Redpanda
5. Applies Cell CR and Store CRs to the k3d cluster

After that, `dp run --cell canary` resolves Stores via `kubectl get stores -n dp-canary`.

### Local development without cells

`dp run` with no `--cell` flag still works — it uses the package-local `store/` directory as a fallback. This is the bootstrapping experience: `dp init` scaffolds self-contained files, `dp run` works immediately without any cell setup.

---

## Package Directory Structure

```
my-pipeline/
├── dp.yaml                        # Transform manifest (required)
├── connector/                     # Plugin images (versioned with the package)
│   ├── postgres.yaml
│   └── s3.yaml
├── asset/                         # Data contracts (versioned with the package)
│   ├── users.yaml
│   └── users-parquet.yaml
├── store/                         # Local dev stores (fallback when no --cell)
│   ├── source-db.yaml
│   └── dest-bucket.yaml
└── src/                           # Code (generic-go, generic-python only)
    └── main.py
```

For CloudQuery transforms, there is no `src/` — the computation is defined entirely by the Connector plugin images and the Asset table/prefix selections.

---

## Developer Journey: `dp init` → `dp run` → `dp publish`

### Step 1: `dp init` — scaffold a self-contained package

```bash
dp init pg-to-s3 --runtime cloudquery
```

Creates a directory with everything needed to run immediately — no cell, no cluster, no external config:

```
pg-to-s3/
├── dp.yaml                        # Transform: runtime, mode, inputs/outputs
├── connector/
│   ├── postgres.yaml              # Connector: cq-source-postgresql image ref
│   └── s3.yaml                    # Connector: cq-destination-s3 image ref
├── asset/
│   ├── pg-to-s3-source-table.yaml # Asset: store: source-db, table: public.example_table
│   └── pg-to-s3-dest-table.yaml   # Asset: store: dest-bucket, prefix: pg-to-s3/, format: parquet
└── store/                         # ← Local dev stores (self-contained, no cell needed)
    ├── source-db.yaml             # Store: connector: postgres, connection_string: localhost PG
    └── dest-bucket.yaml           # Store: connector: s3, bucket: dp-raw, endpoint: localstack
```

Key: Assets reference Stores by name (`store: source-db`). The `store/` directory provides those names with local dev connection strings. This is the package-local defaults — no cell required.

### Step 2: `dp run` — run locally with package stores

```bash
cd pg-to-s3
dp run
```

Resolution: no `--cell` → use `store/` from the package directory. The runner loads `dp.yaml` → follows `inputs[].asset` → reads `asset/*.yaml` → each asset has `store: <name>` → loads `store/<name>.yaml` → gets connection strings → generates runtime config → runs.

At this point you have a working package with zero infrastructure setup beyond what `dp dev up` provides.

### Step 3: `dp run --cell canary` — run against cell stores

```bash
dp run --cell canary
```

Same package, different stores. The `store/` directory is ignored. Stores are resolved from the cell:

```
Asset "pg-to-s3-source-table" → store: source-db → kubectl get store source-db -n dp-canary
Asset "pg-to-s3-dest-table"   → store: dest-bucket → kubectl get store dest-bucket -n dp-canary
```

The package doesn't change. Only the store resolution changes.

### Step 4: `dp build` + `dp publish` — ship the package

```bash
dp build                           # validate + produce Helm chart tarball
dp publish                         # helm push → OCI registry
```

The `store/` directory is **not** included in the published chart. Only Transform, Connectors, and Assets ship. The chart is cell-independent.

### Step 5: Deploy into a cell — what files to add in the CM repo

To deploy `pg-to-s3:1.2.4-g29aef` into cell `canary`, add one directory to the CM repo:

```
cm-repo/apps/pg-to-s3-canary/
├── version.txt                    # 1.2.4-g29aef
└── values.yaml                    # cell: canary
```

```
# version.txt
1.2.4-g29aef
```

```yaml
# values.yaml
cell: canary
# Optional overrides:
# resources:
#   requests:
#     cpu: 200m
#     memory: 512Mi
# schedule: "0 */6 * * *"
```

That's it. ArgoCD sees the new directory, pulls `oci://ghcr.io/infoblox-cto/dp/pg-to-s3:1.2.4-g29aef`, renders the Helm chart with `cell: canary`, and applies the `PackageDeployment` CR to `dp-canary` namespace.

To deploy the same package to `stable`, add another directory:

```
cm-repo/apps/pg-to-s3-stable/
├── version.txt                    # 1.2.3-gabcdef  (maybe an older, proven version)
└── values.yaml                    # cell: stable
```

To promote canary → stable: `echo "1.2.4-g29aef" > apps/pg-to-s3-stable/version.txt && git push`.

---

## Cells Across Clusters (kubectl contexts)

Cells are k8s CRDs — they live in a specific cluster. When you have multiple clusters (k3d local, EKS staging, EKS prod), each cluster has its own set of cells:

```
k3d-dp-local (context: k3d-dp-local)
├── cell:local
├── cell:canary
└── cell:dev-dgarcia

eks-staging (context: arn:aws:eks:us-east-1:...:cluster/dp-staging)
├── cell:canary
├── cell:stable

eks-prod (context: arn:aws:eks:us-east-1:...:cluster/dp-prod)
├── cell:us-east
├── cell:eu-west
```

### Cell CRD with context

The Cell CRD itself doesn't need to know its own context — it lives in one cluster. But the **dp CLI** and **CM repo** need to know which cluster a cell belongs to. This is handled by a context field:

```yaml
# cells/canary/cell.yaml — for local k3d
apiVersion: data.infoblox.com/v1alpha1
kind: Cell
metadata:
  name: canary
spec:
  namespace: dp-canary
  labels:
    tier: canary
```

```yaml
# cells/us-east/cell.yaml — for EKS prod
apiVersion: data.infoblox.com/v1alpha1
kind: Cell
metadata:
  name: us-east
spec:
  namespace: dp-us-east
  labels:
    tier: production
    region: us-east-1
```

The Cell CRD is applied to whichever cluster it belongs to. The `kubectl apply` target determines the cluster:

```bash
# Local
kubectl --context k3d-dp-local apply -f cells/canary/

# EKS
kubectl --context arn:aws:eks:...:dp-staging apply -f cells/canary/
```

### CLI: `--context` flag

The dp CLI respects kubectl context to target a specific cluster:

```bash
dp run --cell canary                                    # uses current context
dp run --cell canary --context k3d-dp-local             # explicit: local k3d
dp run --cell us-east --context arn:aws:eks:...:dp-prod # explicit: EKS prod
dp cell list                                            # cells in current context
dp cell list --context arn:aws:eks:...:dp-prod          # cells in EKS prod
```

### CM repo: one app per (package × cell × cluster)

When the same cell name exists in multiple clusters (e.g., `canary` in both k3d and EKS), the CM repo disambiguates by cluster:

```
cm-repo/
├── clusters/
│   ├── dp-staging/                # EKS staging cluster
│   │   └── apps/
│   │       ├── pg-to-s3-canary/
│   │       │   ├── version.txt    # 1.2.4-g29aef
│   │       │   └── values.yaml    # cell: canary
│   │       └── pg-to-s3-stable/
│   │           ├── version.txt
│   │           └── values.yaml
│   └── dp-prod/                   # EKS prod cluster
│       └── apps/
│           ├── pg-to-s3-us-east/
│           │   ├── version.txt
│           │   └── values.yaml    # cell: us-east
│           └── pg-to-s3-eu-west/
│               ├── version.txt
│               └── values.yaml    # cell: eu-west
```

Each cluster gets its own ArgoCD ApplicationSet pointing at its `clusters/<name>/apps/` path. ArgoCD itself handles multi-cluster via its cluster registration:

```yaml
# ArgoCD ApplicationSet — per cluster
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: dp-apps-staging
spec:
  generators:
    - git:
        repoURL: https://github.com/Infoblox-CTO/platform.data.cm.git
        revision: main
        directories:
          - path: clusters/dp-staging/apps/*
  template:
    spec:
      destination:
        server: https://dp-staging.eks.amazonaws.com   # ← cluster endpoint
```

### Single-cluster simplification

If you only have one cluster (common for local dev and small teams), the CM repo is flat — no `clusters/` nesting:

```
cm-repo/
├── apps/
│   ├── pg-to-s3-canary/
│   ├── pg-to-s3-stable/
│   └── enrich-canary/
```

The multi-cluster layout is additive. You grow into it when you add a second cluster.

### `dp dev up` — always local

`dp dev up` always targets the local k3d cluster (creating it if needed). It never touches remote EKS clusters. Remote cell management is done via `kubectl apply` and GitOps:

```bash
dp dev up                          # k3d only — creates cluster + cell:local
dp dev up --cell canary            # k3d only — creates cell:canary in k3d
# Remote cells: managed by platform team via GitOps, never by dp dev up
```

---

## Use Cases

### Testing a new plugin version

Package v0.1.0 uses `cq-source-postgresql:v8`. You want to test v9.

```bash
# Edit connector/postgres.yaml → change image tag to v9.0.0
# Package is now v0.2.0-dev

dp run                            # test locally with package stores
dp run --cell canary              # test against shared infra canary data
# Verify results, then:
dp publish                        # push v0.2.0 to OCI registry
# GitOps promotes v0.2.0 to cell:stable
```

### Testing new custom code

Same flow. Your package `src/main.py` has new logic, or the container image is rebuilt. The package version changes, you test through cells:

```bash
dp run                            # local
dp run --cell canary              # shared infra
dp publish                        # ship
```

### Canary / stable promotion

Two cells in the same cluster, same infrastructure (or separate schemas/databases):

```
cell:canary   namespace: dp-canary    Store "warehouse" → dp_canary DB
cell:stable   namespace: dp-stable    Store "warehouse" → dp_stable DB
```

Deploy package v0.2.0 to canary. If metrics look good, deploy to stable. Both cells run in the same k8s cluster. The controller manages which package version is deployed to which cell.

### Developer sandbox

```bash
dp dev up --cell dev-dgarcia      # creates namespace dp-dev-dgarcia + stores
dp run --cell dev-dgarcia         # runs your package version in isolation
```

Developer cells can use a shared database instance with an isolated schema, or a per-developer database. The Store manifest in the cell config controls this.

### Tenant cell groups

```
cell:us-east    Store "warehouse" → us-east PG (tenants A, B, C)
cell:eu-west    Store "warehouse" → eu-west PG (tenants D, E)
```

Same package deployed to both cells. Each cell's Stores point to the regional database containing that cell's tenants' data. Tenant routing is achieved by which data the cell's Stores expose, not by Transform logic.

---

## Cross-Cell Transforms (Fan-Out / Routing)

Most transforms operate within a single cell — they read and write using that cell's Stores. But some transforms span cells: they consume from a shared source and route data to multiple target cells.

### The problem

A Kafka topic has incoming multi-tenant data (random hash key, all tenants mixed). A routing Transform needs to:
1. Read from the shared topic (one input, in an `ingress` cell)
2. Partition and write to 3 different cells (each cell has its own output stores)

The standard model — "Stores come from the cell" — doesn't work here because the Transform touches multiple cells.

### Solution: cell-qualified Asset references

Assets gain an optional `cell` field on their output references. When set, the Store for that Asset is resolved from the named cell instead of the deployment cell:

```yaml
# dp.yaml — Router Transform, deployed to cell:ingress
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: tenant-router
spec:
  runtime: generic-go
  mode: streaming

  inputs:
    - asset: raw-events              # no cell → resolved from deployment cell (ingress)

  outputs:
    - asset: tenant-a-events
      cell: cell-us-east             # ← resolve Store from cell-us-east
    - asset: tenant-b-events
      cell: cell-us-east             # ← same cell, different asset
    - asset: tenant-c-events
      cell: cell-eu-west             # ← resolve Store from cell-eu-west
```

```yaml
# asset/raw-events.yaml — input from shared Kafka
kind: Asset
metadata:
  name: raw-events
spec:
  store: shared-kafka               # resolved from cell:ingress (the deployment cell)
  topic: incoming.events

# asset/tenant-a-events.yaml — output to cell-us-east
kind: Asset
metadata:
  name: tenant-a-events
spec:
  store: event-lake                  # resolved from cell:cell-us-east (cell-qualified)
  prefix: "tenant-a/"
  format: parquet
```

### Resolution with cell-qualified refs

```
Transform "tenant-router" deployed to cell:ingress
  → input "raw-events"
      Asset.store = "shared-kafka"   → no cell override → resolve from cell:ingress
      kubectl get store shared-kafka -n dp-ingress

  → output "tenant-a-events"
      Asset.store = "event-lake"     → cell: cell-us-east
      kubectl get store event-lake -n dp-cell-us-east

  → output "tenant-c-events"
      Asset.store = "event-lake"     → cell: cell-eu-west
      kubectl get store event-lake -n dp-cell-eu-west
```

The Store name `event-lake` exists in both `cell-us-east` and `cell-eu-west` but points to different S3 buckets / databases. The routing Transform's code decides which records go to which output Asset based on tenant ID.

### AssetRef with optional cell

The `AssetRef` struct gains an optional `Cell` field:

```yaml
# Current — single-cell (common case)
inputs:
  - asset: raw-events
outputs:
  - asset: users-parquet

# Extended — cell qualifier on outputs (cross-cell routing)
outputs:
  - asset: tenant-a-events
    cell: cell-us-east
  - asset: tenant-c-events
    cell: cell-eu-west
```

When `cell` is omitted (the common case), the Store is resolved from the deployment cell. When `cell` is set, the Store is resolved from that named cell. This keeps single-cell transforms unchanged while enabling cross-cell routing.

### Patterns this enables

| Pattern | Input | Output | Example |
|---|---|---|---|
| **Fan-out** | Shared topic → 1 cell | Multiple assets → N cells | Tenant router: Kafka → per-cell S3 |
| **Fan-in** | Assets from N cells | Single asset → 1 cell | Aggregator: per-cell PG → central warehouse |
| **Cross-cell ETL** | Asset from cell A | Asset in cell B | Migration: old-cell PG → new-cell PG |
| **Broadcast** | Asset from 1 cell | Same asset replicated to N cells | Replication: primary → regional caches |

All of these are expressed with the same mechanism: `cell` on `AssetRef`.

---

## Multi-Tenancy Within Cells

Cells handle **infrastructure isolation** (different namespace, credentials, physical endpoints). Tenancy within a cell is handled at the **data level**:

| Pattern | When to use | How |
|---|---|---|
| **One cell per tenant** | Strong isolation required, regulatory, separate infra | Each tenant's cell has its own Stores pointing to dedicated infra |
| **Cell per tenant group** | Cost-efficient, regional grouping | Cell Stores point to shared infra; multiple tenants coexist in same DB/bucket |
| **Single cell, tenant-filtered** | All tenants in one place | Assets define which tables/topics to read; tenant ID in table names or topic partitions |

These compose: a cell can serve multiple tenants, and the package's Assets specify which tables to read. The same Transform definition deploys to multiple cells with no changes — only the Stores differ.

---

## Infrastructure Sharing Within a Cluster

`dp dev up` installs shared infrastructure (PostgreSQL, S3/LocalStack, Kafka/Redpanda, Marquez) once. Cells share these instances but use separate databases, buckets, or topic prefixes:

```
Shared infrastructure (namespace: dp-infra)
├── PostgreSQL
│   ├── database: dp_local          ← cell:local
│   ├── database: dp_canary         ← cell:canary
│   ├── database: dp_stable         ← cell:stable
│   └── database: dp_dev_dgarcia    ← cell:dev-dgarcia
├── LocalStack S3
│   ├── bucket: dp-local-raw
│   ├── bucket: dp-canary-raw
│   ├── bucket: dp-stable-raw
│   └── bucket: dp-dev-dgarcia-raw
├── Redpanda
│   ├── topics: local.*
│   ├── topics: canary.*
│   └── topics: stable.*
└── Marquez (shared, observes all cells)
```

`dp dev up` creates the shared infra. `dp dev up --cell <name>` creates the cell's namespace, database/schema, bucket, and topic prefix within the shared infra, then writes the cell's Store manifests.

In production, cells may use fully separate infrastructure (dedicated RDS, MSK, etc.) — the model is the same, only the Store connection strings change.

---

## Packaging: Helm Chart as OCI Artifact

### The artifact

`dp publish` produces a **Helm chart** pushed to an OCI registry. The chart is the cell-independent, versioned release unit:

```
ghcr.io/infoblox-cto/dp/pg-to-s3:1.2.4-g29aef     ← Helm chart OCI artifact
```

The chart bundles everything in the package — Transform manifest, Connectors, Assets, and (for generic runtimes) the container image reference. It is cell-independent: no Store references, no namespace, no credentials.

### What's inside the chart

```
pg-to-s3/
├── Chart.yaml                     # name: pg-to-s3, version: 1.2.4-g29aef
├── values.yaml                    # cell: "", overrides (resources, replicas, schedule)
├── templates/
│   └── packagedeployment.yaml     # renders → PackageDeployment CR
└── manifests/                     # package manifests embedded in chart
    ├── dp.yaml
    ├── connector/
    │   ├── postgres.yaml
    │   └── s3.yaml
    └── asset/
        ├── users.yaml
        └── users-parquet.yaml
```

```yaml
# Chart.yaml
apiVersion: v2
name: pg-to-s3
version: 1.2.4-g29aef
appVersion: "1.2.4"
description: PostgreSQL to S3 data package
type: application
annotations:
  io.infoblox.dp/kind: package
  io.infoblox.dp/runtime: cloudquery
```

```yaml
# values.yaml — cell-independent defaults, overridable per deployment
cell: ""                            # REQUIRED at deploy time
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 1Gi
replicas: 1
schedule: ""                        # batch mode: cron expression
```

```yaml
# templates/packagedeployment.yaml
apiVersion: data.infoblox.com/v1alpha1
kind: PackageDeployment
metadata:
  name: {{ .Chart.Name }}
  namespace: dp-{{ .Values.cell }}
spec:
  package:
    name: {{ .Chart.Name }}
    version: {{ .Chart.Version }}
    registry: {{ .Values.registry | default "ghcr.io/infoblox-cto/dp" }}
  cell: {{ .Values.cell }}
  mode: {{ .Values.mode | default "batch" }}
  {{- if .Values.schedule }}
  schedule:
    cron: {{ .Values.schedule | quote }}
  {{- end }}
  resources:
    {{- toYaml .Values.resources | nindent 4 }}
```

### Build → Publish flow

```bash
dp build                           # validates manifests, builds Helm chart locally
dp publish                         # helm push → OCI registry
dp publish --registry ghcr.io/myorg/dp   # push to specific registry
```

`dp build` creates `dist/pg-to-s3-1.2.4-g29aef.tgz` — a standard Helm chart tarball. `dp publish` is equivalent to:

```bash
helm push dist/pg-to-s3-1.2.4-g29aef.tgz oci://ghcr.io/infoblox-cto/dp
```

The chart version is derived from git: `<semver>-g<short-sha>`. Tag immutability is enforced — you cannot overwrite a published version.


---

## CLI Commands

```bash
# Cell management
dp dev up                          # shared infra + default "local" cell
dp dev up --cell canary            # create canary cell in cluster
dp dev up --cell dev-$(whoami)     # create developer sandbox cell
dp cell list                       # list available cells (kubectl get cells)
dp cell show canary                # show cell details + stores
dp cell stores canary              # list stores in canary cell

# Package development
dp init foo --runtime cloudquery   # scaffold package with local dev stores
dp run                             # run with package stores (no cell, local dev)
dp run --cell canary               # run with canary cell stores
dp run --cell dev-dgarcia          # run in sandbox cell

# Build + publish (package → Helm chart → OCI)
dp build                           # validate + produce dist/foo-1.2.4-g29aef.tgz
dp publish                         # helm push → OCI registry
dp publish --registry ghcr.io/myorg/dp

# Promotion (update version.txt in CM repo)
dp promote pg-to-s3 1.2.4-g29aef --to stable   # PR against CM repo
dp promote pg-to-s3 1.2.4-g29aef --to canary --auto-merge

# Discovery
kubectl get cells                  # cluster-scoped: all cells
kubectl get stores -n dp-canary    # namespaced: stores in canary cell
kubectl get stores -A              # all stores across all cells
dp search pg-to-s3                 # OCI: find packages in registry
helm pull oci://ghcr.io/infoblox-cto/dp/pg-to-s3 --version 1.2.4-g29aef
```

---

## Summary Table

| Concept | Scope | k8s kind | Discovery | Who manages | Versioned? |
|---|---|---|---|---|---|
| **Package** | Registry | — (OCI artifact) | `dp search`, registry tags | Data engineer | Yes |
| **Transform** | Package | — (in OCI) | Part of package | Data engineer | Yes (package version) |
| **Asset** | Package | — (in OCI) | Part of package | Data engineer | Yes (package version) |
| **Connector** | Package | — (in OCI) | Part of package | Data engineer | Yes (package version) |
| **Cell** | Cluster | `Cell` (cluster-scoped CRD) | `kubectl get cells` | Platform team | No (infra config) |
| **Store** | Cell namespace | `Store` (namespaced CRD) | `kubectl get stores -n <ns>` | Platform team / `dp dev up` | No (infra config) |
