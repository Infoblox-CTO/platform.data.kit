---
title: Target State — Operating Model
description: Two-persona architecture for DataKit
---

# Target State — Operating Model

## Two personas, one platform

DataKit serves two distinct roles with clear ownership boundaries.

| Persona | Owns | Defines |
|---------|------|---------|
| **Platform engineer** | Extensions, environments, policies | *What is allowed* |
| **Data engineer** | Assets, pipelines, models | *What runs and what it produces* |

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Platform Engineer                                 │
│                                                                             │
│   extensions/                  environments/            policies/           │
│   ├── cloudquery/              ├── dev.yaml              ├── quality.yaml   │
│   │   ├── sources/aws/         ├── stage.yaml            └── versions.yaml  │
│   │   │   ├── extension.yaml   └── prod.yaml                               │
│   │   │   ├── schema.json                                                   │
│   │   │   ├── versions.yaml                                                 │
│   │   │   └── templates/                                                    │
│   │   └── destinations/                                                     │
│   └── dbt/                                                                  │
│       └── model-engines/                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                            Data Engineer                                    │
│                                                                             │
│   assets/                      pipelines/               models/             │
│   ├── sources/                 └── aws_compliance/      └── dbt/            │
│   │   └── aws_security/           ├── pipeline.yaml         ├── staging/    │
│   │       └── asset.yaml          └── schedule.yaml         └── marts/     │
│   └── sinks/                                                                │
│       └── snowflake_raw/                                                    │
│           └── asset.yaml                                                    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Four layers

The platform separates concerns into four distinct layers. Each layer references the one above it by name — never by embedding its internals.

```
Extension ──▶ Asset ──▶ Pipeline ──▶ Binding
  (type)      (instance)  (wiring)    (infra)
```

### 1. Extensions — approved building blocks

An extension is a **type definition** published by a platform engineer. It declares what a source, sink, or model engine *is* — its configuration schema, version policy, templates, and documentation.

```yaml
# extensions/cloudquery/sources/aws/extension.yaml
name: cloudquery.source.aws
kind: source
engine: cloudquery
description: "CloudQuery AWS source — IAM, EC2, S3, and 300+ tables"
```

Each extension ships with a `schema.json` that validates consumer configuration, a `versions.yaml` for version policy, and optional templates and examples. Extensions are published to the OCI registry via `dk ext publish`.

### 2. Assets — configured instances

An asset is a **configured instance** of an extension, created by a data engineer. It contains no code — just configuration referencing an approved extension by FQN and version.

```yaml
# assets/sources/aws_security/asset.yaml
name: aws_security
type: source
extension: cloudquery.source.aws
version: v24.0.2
owner_team: security
config:
  accounts: ["prod", "security"]
  regions: ["us-east-1", "us-west-2"]
  tables: ["iam_roles", "iam_policies", "ec2_instances"]
```

The `config` block is validated against the extension's `schema.json` at `dk validate` time — errors surface before runtime.

### 3. Pipelines — multi-step wiring

A pipeline chains assets into a workflow: sync → transform → test → publish. Each step references assets by name.

```yaml
# pipelines/aws_compliance/pipeline.yaml
name: aws_security_compliance
steps:
  - name: sync
    source: aws_security
    sink: snowflake_raw
  - name: transform
    model: dbt_compliance
    select: "tag:compliance"
  - name: test
    model: dbt_compliance
    command: test
```

Pipelines can define schedules, backfill ranges, and notification targets. The existing single-container pipeline mode is preserved as a `custom` step type for backward compatibility.

### 4. Bindings — per-environment infrastructure

Bindings map abstract asset references to concrete infrastructure, varying by environment. A pipeline's definition never changes between dev and prod — only its bindings do.

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

Platform engineers define environments with allowed extensions, version constraints, approval workflows, and resource quotas.

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
dk ext create cloudquery.source.aws --kind source --engine cloudquery
# Edit schema.json, templates/, versions.yaml, examples
dk ext validate cloudquery.source.aws
dk ext publish
```

### Data engineer

```
dk asset create aws_security --ext cloudquery.source.aws --interactive
dk asset create snowflake_raw --ext cloudquery.dest.snowflake
dk pipeline create aws_compliance --template sync-transform-test
# Edit dbt models and tests
dk validate
dk plan --env dev
dk apply --env dev
dk pipeline run aws_compliance --env dev
dk pipeline backfill aws_compliance --from 2026-01-01 --to 2026-02-01
```

---

## RACI summary

| Area | Platform Eng | Data Eng |
|------|:------------:|:--------:|
| Define extension types (sources, sinks, model engines) | **R/A** | C |
| Extension schemas, templates, version policy | **R/A** | I |
| Managed environments (security, compliance, monitoring) | **R/A** | I |
| Dev environment blueprints and quotas | **R/A** | C |
| Domain dev environment declaration (capabilities needed) | C | **R/A** |
| Creating configured asset instances | C | **R/A** |
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
| 011 | Extension type system | `dk ext create/validate/publish`, extension.yaml + schema.json |
| 012 | Asset instances | `dk asset create/validate`, asset.yaml referencing extensions |
| 013 | Pipeline orchestration | Multi-step pipelines, `dk pipeline backfill`, schedule.yaml |
| 014 | Environments and policies | `dk plan/apply`, declarative policies, version constraints |
| 015 | dbt model engine | dbt as a first-class extension, sync → transform → test chains |

The existing `dk init` / `dk build` / `dk run` workflow continues to work throughout — it is progressively refactored under the hood as each feature lands. The current CloudQuery source plugin becomes the first built-in extension.
