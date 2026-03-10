---
title: Target State — Operating Model
description: Two-persona architecture for DataKit
---

# Target State — Operating Model

## Two personas, one platform

DataKit serves two distinct roles with clear ownership boundaries.

| Persona | Owns | Defines |
|---------|------|---------|
| **Platform engineer** | Connectors, environments, policies | *What is allowed* |
| **Data engineer** | DataSets, pipelines, models | *What runs and what it produces* |

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Platform Engineer                                 │
│                                                                             │
│   connector/                   environments/            policies/           │
│   ├── postgres.yaml            ├── dev.yaml              ├── quality.yaml   │
│   ├── s3.yaml                  ├── stage.yaml            └── versions.yaml  │
│   └── kafka.yaml               └── prod.yaml                               │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                            Data Engineer                                    │
│                                                                             │
│   store/                       dataset/                 transforms/         │
│   ├── warehouse.yaml           ├── users.yaml           └── aws_compliance/ │
│   └── lake-raw.yaml            ├── users-parquet.yaml       └── dk.yaml    │
│                                └── orders.yaml                              │
│                                                                             │
│   models/                                                                   │
│   └── dbt/                                                                  │
│       ├── staging/                                                          │
│       └── marts/                                                            │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Four layers

The platform separates concerns into four distinct layers. Each layer references the one above it by name — never by embedding its internals.

```
Connector ──▶ Store ──▶ DataSet ──▶ Transform ──▶ Binding
  (type)      (instance)  (contract)  (compute)    (infra)
```

### 1. Connectors — technology types

A Connector is a **technology type definition** maintained by the platform team. It declares what a storage technology *is* — Postgres, S3, Kafka, etc. — and which CloudQuery plugin images to use.

```yaml
# connector/postgres.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  protocol: postgresql
  capabilities: [source, destination]
```

Connectors rarely change. They define the technology catalog available to the platform.

### 2. Stores and DataSets — data contracts

A **Store** is a named instance of a Connector with connection details and credentials, managed by infra/SRE. A **DataSet** is a named data contract — a table, S3 prefix, or Kafka topic that lives in a Store — created by data engineers.

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

DataSets are declarative metadata: schema, classification, and lineage. Validation runs at `dk lint` time — errors surface before runtime.

### 3. Pipelines — reactive dependency graph

A pipeline is the dependency graph derived from Transform and DataSet manifests.
Each Transform declares its inputs and outputs; the graph is built automatically.

```bash
# View the full pipeline graph
dk pipeline show

# View the chain leading to a specific DataSet
dk pipeline show --destination event-summary
```

Transforms are independently deployable. Each declares a trigger (schedule,
on-change, manual) and references DataSets by name. There is no separate pipeline
manifest — the graph emerges from the individual dk.yaml files.

### 4. Bindings — per-environment infrastructure

Bindings map abstract DataSet references to concrete infrastructure, varying by environment. A pipeline's definition never changes between dev and prod — only its bindings do.

```yaml
# environments/prod.yaml (excerpt)
bindings:
  snowflake_raw:
    type: snowflake
    account: acme.us-east-1
    database: RAW
    schema: SECURITY
    role: LOADER
```

---

## Environments and policies

### Managed environments

Platform engineers define environments with allowed connectors, version constraints, approval workflows, and resource quotas.

| Environment | Version policy | Approval | Purpose |
|-------------|---------------|----------|---------|
| **dev** | Ranges allowed | Auto-merge | Rapid iteration |
| **stage** | Ranges allowed | Team approval | Integration testing |
| **prod** | Exact pins only | Multi-party | Production traffic |

### Policies

Declarative YAML policies enforce guardrails at `dk validate` time:

- **versions.yaml** — prod requires exact version pins; dev allows ranges
- **quality.yaml** — gold-tier outputs require tests and documentation
- **security.yaml** — all outputs require PII classification metadata

---

## Developer workflows

### Platform engineer

```
# Define connectors and stores
# Edit connector/ and store/ manifests
dk lint
```

### Data engineer

```
dk dataset create users --store warehouse --table public.users
dk dataset create users-parquet --store lake-raw --prefix data/users/
# Edit DataSet schemas and dbt models
dk lint
dk run
dk pipeline show
```

---

## RACI summary

| Area | Platform Eng | Data Eng |
|------|:------------:|:--------:|
| Define connectors and technology types | **R/A** | C |
| Store connection details and credentials | **R/A** | I |
| Managed environments (security, compliance, monitoring) | **R/A** | I |
| Dev environment blueprints and quotas | **R/A** | C |
| Domain dev environment declaration (capabilities needed) | C | **R/A** |
| Creating DataSets (data contracts) | C | **R/A** |
| Creating pipelines (sync → transform → test → publish) | C | **R/A** |
| dbt models, tests, docs | I | **R/A** |
| CI policy checks and enforcement | **R/A** | I |
| Incident response — platform runtime | **R/A** | C |
| Incident response — domain pipelines/models | C | **R/A** |

**Rule of thumb**: platform owns *how it runs safely*; data owns *what runs and what it produces*.

---

## Evolution path

The target state is reached incrementally through five features, each independently shippable:

| # | Feature | Delivers |
|---|---------|----------|
| 011 | Connector and Store system | Connector and Store manifests, technology type catalog |
| 012 | DataSet data contracts | `dk dataset create/validate`, DataSet manifests with schema and classification |
| 013 | Pipeline graph | Reactive dependency graph, `dk pipeline show`, trigger configuration |
| 014 | Environments and policies | `dk plan/apply`, declarative policies, version constraints |
| 015 | dbt model engine | dbt as a first-class runtime, sync → transform → test chains |

The existing `dk init` / `dk build` / `dk run` workflow continues to work throughout — it is progressively refactored under the hood as each feature lands.
