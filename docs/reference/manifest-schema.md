---
title: Manifest Schema
description: Complete reference for all manifest schemas (Transform, Asset, AssetGroup, Connector, Store)
---

# Manifest Schema Reference

This document provides the complete schema reference for all manifest kinds in the Data Platform Kit.

The platform uses five manifest kinds, each with a clear ownership boundary:

| Kind | Owner | Purpose |
|------|-------|---------|
| **Connector** | Platform team | Technology type catalog (postgres, s3, kafka) |
| **Store** | Infra / SRE | Named instance of a Connector with connection details |
| **Asset** | Data engineer | Data contract — table, prefix, or topic in a Store |
| **AssetGroup** | Data engineer | Bundle of Assets produced by a single materialisation |
| **Transform** | Data engineer | Unit of computation reading input Assets, producing output Assets |

---

## Transform Schema

The Transform manifest (`dk.yaml`) defines a unit of computation.
It is the **only manifest kind that runs** — Connectors, Stores, and Assets are declarative metadata.

### Full Schema

```yaml
# dk.yaml
apiVersion: data.infoblox.com/v1alpha1          # Required: API version
kind: Transform                                 # Required: Resource type

metadata:                           # Required: Transform metadata
  name: string                      # Required: Transform name (1-63 chars, lowercase, hyphenated)
  namespace: string                 # Optional: Team namespace
  version: string                   # Optional: Semantic version (e.g., "0.1.0")
  labels:                           # Optional: Key-value labels
    key: value
  annotations:                      # Optional: Arbitrary metadata
    key: value

spec:                               # Required: Transform specification
  runtime: string                   # Required: cloudquery | generic-go | generic-python | dbt
  mode: string                      # Optional: batch | streaming (default: batch)

  inputs:                           # Required: Input Asset references
    - asset: string                 # Asset name (local or OCI ref)
      tags:                         # OR match assets by labels
        key: value
      version: string               # Optional: Semver range constraint
      cell: string                  # Optional: Cell/region constraint

  outputs:                          # Required: Output Asset references
    - asset: string                 # Asset name (local or OCI ref)
      tags:                         # OR match assets by labels
        key: value
      version: string               # Optional: Semver range constraint
      cell: string                  # Optional: Cell/region constraint

  image: string                     # Required for generic-go/generic-python/dbt runtimes
  command: [string]                 # Optional: Override container entrypoint

  env:                              # Optional: Environment variables
    - name: string
      value: string
    - name: string
      valueFrom:
        secretRef:
          name: string
          key: string

  trigger:                          # Optional: Reactive trigger policy
    policy: string                  # Required: schedule | on-change | manual | composite
    schedule:                       # Required when policy=schedule
      cron: string                  # 5-field cron expression
      timezone: string              # IANA timezone (default: UTC)
    policies:                       # Required when policy=composite
      - string                      # e.g., ["schedule", "on-change"]

  schedule:                         # Optional: Cron scheduling (shorthand for trigger.policy=schedule)
    cron: string                    # Cron expression (e.g., "0 */6 * * *")
    timezone: string                # IANA timezone (default: UTC)

  timeout: string                   # Optional: Max execution time (e.g., "30m", "1h")

  resources:                        # Optional: Kubernetes resource limits
    cpu: string                     # e.g., "500m", "1", "2"
    memory: string                  # e.g., "512Mi", "2Gi"

  replicas: integer                 # Optional: Parallel workers (streaming mode, default: 1)

  lineage:                          # Optional: Lineage configuration
    enabled: boolean                # Enable OpenLineage events (default: false)
    heartbeatInterval: string       # Heartbeat frequency (default: 30s)
```

### Field Reference

#### metadata.name

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Pattern | `^[a-z][a-z0-9-]{0,62}$` |
| Description | DNS-safe transform identifier. Must start with a lowercase letter. |

#### metadata.version

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Format | Semantic version (e.g., `0.1.0`) |
| Description | Version of the transform package. Used in OCI artifact tags. |

#### spec.runtime

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Enum | `cloudquery`, `generic-go`, `generic-python`, `dbt` |
| Description | Execution engine for the transform. |

**Runtime descriptions:**

| Runtime | Use Case | Requirements |
|---------|----------|-------------|
| `cloudquery` | CQ source-to-destination syncs | No image — plugin images come from Connector |
| `generic-go` | Custom Go containers | Requires `image` field |
| `generic-python` | Custom Python containers | Requires `image` field |
| `dbt` | dbt transformations | Requires `image` field |

#### spec.mode

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Enum | `batch`, `streaming` |
| Default | `batch` |
| Description | Execution mode. Batch creates k8s Jobs; streaming creates Deployments. |

#### spec.inputs / spec.outputs

| Property | Value |
|----------|-------|
| Type | array of AssetRef objects |
| Required | Yes |
| Description | Input/output Asset references. Each entry uses either `asset` (named reference) or `tags` (label matching) — not both. |

**AssetRef fields:**

| Field | Type | Description |
|-------|------|-------------|
| `asset` | string | Named Asset reference (local name or OCI ref) |
| `tags` | map(string→string) | Match Assets by labels (alternative to `asset`) |
| `version` | string | Semver range constraint (e.g., `>=1.0.0`) |
| `cell` | string | Cell/region constraint |

At runtime, the runner resolves the chain: **Transform → Asset → Store → Connector** to obtain connection details and plugin images.

#### spec.trigger

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Description | Reactive trigger policy. Replaces or extends `spec.schedule`. |

**Trigger fields:**

| Field | Type | Description |
|-------|------|-------------|
| `policy` | string (required) | `schedule`, `on-change`, `manual`, or `composite` |
| `schedule` | object | Required when policy=schedule. Has `cron` and `timezone` fields. |
| `policies` | array of strings | Required when policy=composite. Sub-policies to combine. |

#### spec.image

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes (for generic-go, generic-python, dbt) |
| Examples | `my-team/enrich-users:latest`, `python:3.11` |
| Description | Container image for non-CloudQuery runtimes. Not needed for `cloudquery` runtime. |

#### spec.timeout

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Default | `30m` |
| Pattern | Go duration format (e.g., `30m`, `1h30m`, `2h`) |
| Description | Maximum execution time before timeout. |

#### spec.schedule

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Fields | `cron` (string, 5-field cron), `timezone` (string, IANA) |
| Description | Cron schedule for batch transforms. |

#### spec.resources

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Fields | `cpu` (string), `memory` (string) |
| Description | Kubernetes-style resource requests/limits. |

#### spec.replicas

| Property | Value |
|----------|-------|
| Type | integer |
| Required | No |
| Default | 1 |
| Range | 1-100 |
| Description | Parallel workers for streaming mode. |

#### spec.lineage

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Fields | `enabled` (boolean), `heartbeatInterval` (string) |
| Description | OpenLineage event configuration. |

### Transform Validation Errors

| Code | Message | Resolution |
|------|---------|------------|
| E001 | `metadata.name is required` | Add name field |
| E002 | `spec.runtime is required` | Add runtime field |
| E004 | `invalid name format` | Use lowercase and hyphens only |
| E005 | `name too long` | Maximum 63 characters |

### Examples

#### CloudQuery Transform (no image needed)

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
  runtime: cloudquery
  mode: batch
  inputs:
    - asset: users
  outputs:
    - asset: users-parquet
  schedule:
    cron: "0 */6 * * *"
  timeout: 30m
```

#### Generic Python Transform

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

#### Streaming Transform

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: event-processor
  namespace: default
  version: 1.0.0
spec:
  runtime: generic-go
  mode: streaming
  inputs:
    - asset: raw-events
  outputs:
    - asset: processed-events
  image: my-team/event-processor:v1.0.0
  replicas: 3
  resources:
    cpu: "1"
    memory: 2Gi
  lineage:
    enabled: true
    heartbeatInterval: 30s
```

#### Reactive Transform with Trigger

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: enrich
  version: 0.3.0
spec:
  runtime: generic-python
  inputs:
    - asset: raw-events-parquet
  outputs:
    - asset: enriched-events
  image: my-team/enrich:latest
  trigger:
    policy: on-change
```

#### Tag-based Input Resolution

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: aggregate-all
  version: 1.0.0
spec:
  runtime: dbt
  inputs:
    - tags:
        domain: analytics
        tier: curated
      version: ">=1.0.0"
  outputs:
    - asset: analytics-summary
  image: my-team/dbt-runner:latest
  trigger:
    policy: schedule
    schedule:
      cron: "0 */6 * * *"
```

---

## Asset Schema

An Asset is a **named data contract**: a table, S3 prefix, or Kafka topic that lives in a Store.
It declares the schema (with optional column-level lineage via `from`) and data classification.

### Full Schema

```yaml
# asset/<name>.yaml
apiVersion: data.infoblox.com/v1alpha1          # Required: API version
kind: Asset                                     # Required: Resource type

metadata:                           # Required: Asset metadata
  name: string                      # Required: Asset name (1-63 chars, lowercase, hyphenated)
  namespace: string                 # Optional: Team namespace
  version: string                   # Optional: Semantic version (e.g., "1.0.0")
  labels:                           # Optional: Key-value labels
    key: value

spec:                               # Required: Asset specification
  store: string                     # Required: Name of the Store where this data lives
  table: string                     # Optional: Fully-qualified table name (relational stores)
  prefix: string                    # Optional: Object prefix (object stores like S3)
  topic: string                     # Optional: Topic name (streaming stores like Kafka)
  format: string                    # Optional: Data format (parquet, json, csv, avro)
  classification: string            # Optional: public | internal | confidential | restricted

  schema:                           # Optional: Field/column definitions
    - name: string                  # Required: Field name
      type: string                  # Required: Data type (integer, string, timestamp, boolean, float)
      pii: boolean                  # Optional: Contains PII? (default: false)
      from: string                  # Optional: Lineage source (e.g., "users.id")

  dev:                              # Optional: Development-only configuration
    seed:                           # Optional: Sample data for local development
      inline:                       # Option A: Rows defined directly in YAML
        - { col: value, ... }
      file: string                  # Option B: Path to a CSV or JSON seed file
      profiles:                     # Optional: Named alternative data sets
        <profile-name>:
          inline:                   # Inline rows for this profile
            - { col: value, ... }
          file: string              # OR file path for this profile
```

!!! info "Dev-only section"
    The `dev` block is ignored in production deployments. It exists solely to
    support the local development workflow (`dk dev seed` and auto-seeding
    during `dk run`).

### Field Reference

#### metadata.name

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Pattern | `^[a-z][a-z0-9-]{0,62}$` |
| Description | DNS-safe asset identifier. |

#### spec.store

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Description | Name of the Store manifest this asset lives in. |

#### spec.table / spec.prefix / spec.topic

| Property | Value |
|----------|-------|
| Type | string |
| Required | At least one of `table`, `prefix`, or `topic` |
| Description | Location within the Store. Use `table` for databases, `prefix` for S3, `topic` for Kafka. |

#### spec.classification

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Enum | `public`, `internal`, `confidential`, `restricted` |
| Description | Data sensitivity level. |

#### spec.schema[].from

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Format | `<asset-name>.<field-name>` |
| Description | Declares column-level lineage. Links this field to its source in another Asset. |

### Asset Validation Errors

| Code | Message | Resolution |
|------|---------|------------|
| E025 | `pii=true requires classification` | Add classification level |
| E026 | `confidential requires retention policy` | Add retention policy |

### Examples

#### Input Asset (database table)

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

#### Output Asset with column-level lineage

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

#### Input Asset with seed data and profiles

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
  namespace: default
spec:
  store: warehouse
  table: example_table
  classification: internal
  schema:
    - name: id
      type: integer
    - name: name
      type: string
    - name: created_at
      type: timestamp
  dev:
    seed:
      inline:
        - { id: 1, name: "alice",   created_at: "2026-01-01T00:00:00Z" }
        - { id: 2, name: "bob",     created_at: "2026-01-15T00:00:00Z" }
        - { id: 3, name: "charlie", created_at: "2026-02-01T00:00:00Z" }
      profiles:
        large-dataset:
          file: testdata/large-users.csv
        edge-cases:
          inline:
            - { id: -1, name: "",        created_at: "1970-01-01T00:00:00Z" }
            - { id: 999, name: "O'Reilly", created_at: "2099-12-31T23:59:59Z" }
        empty: {}
```

#### spec.dev.seed

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Description | Sample data to load into the backing store for local development. |

| Sub-field | Type | Description |
|-----------|------|-------------|
| `inline` | array of maps | Rows defined directly in YAML (default profile). |
| `file` | string | Path (relative to package root) to a CSV or JSON seed file (default profile). |
| `profiles` | map of objects | Named alternative data sets. Each profile has its own `inline` or `file`. |

Seed data is loaded by `dk dev seed` or automatically before `dk run`. A SHA-256
checksum is tracked in a `_dp_seed_meta` table so that unchanged data is skipped
on subsequent runs.

---

## AssetGroup Schema

An AssetGroup bundles multiple Assets produced by a single materialisation
(e.g., a CloudQuery sync that snapshots several tables at once).

### Full Schema

```yaml
# asset-group/<name>.yaml
apiVersion: data.infoblox.com/v1alpha1          # Required: API version
kind: AssetGroup                                # Required: Resource type

metadata:                           # Required: AssetGroup metadata
  name: string                      # Required: Group name (1-63 chars, lowercase, hyphenated)
  namespace: string                 # Optional: Team namespace
  labels:                           # Optional: Key-value labels
    key: value

spec:                               # Required: AssetGroup specification
  store: string                     # Required: Common Store for all assets in the group
  assets:                           # Required: List of Asset names
    - string
```

### Example

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

---

## Connector Schema

A Connector describes a storage technology type (e.g., Postgres, S3, Kafka).
Maintained by the **platform team** and rarely changes.

### Full Schema

```yaml
# connector/<name>.yaml
apiVersion: data.infoblox.com/v1alpha1          # Required: API version
kind: Connector                                 # Required: Resource type

metadata:                           # Required: Connector metadata
  name: string                      # Required: Connector name (e.g., "postgres", "s3", "kafka")
  labels:                           # Optional: Key-value labels
    key: value

spec:                               # Required: Connector specification
  type: string                      # Required: Technology identifier (postgres, s3, kafka, snowflake)
  protocol: string                  # Optional: Wire protocol (postgresql, s3, kafka)
  capabilities:                     # Required: Supported roles
    - string                        # "source", "destination", or both

  plugin:                           # Optional: CloudQuery plugin images
    source: string                  # CQ source plugin image
    destination: string             # CQ destination plugin image
```

### Field Reference

#### spec.type

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Examples | `postgres`, `s3`, `kafka`, `snowflake` |
| Description | Technology identifier. |

#### spec.capabilities

| Property | Value |
|----------|-------|
| Type | array of strings |
| Required | Yes |
| Values | `source`, `destination` |
| Description | What roles this connector can serve. |

#### spec.plugin

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Fields | `source` (string), `destination` (string) |
| Description | CloudQuery plugin images for the `cloudquery` runtime. |

### Examples

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  protocol: postgresql
  capabilities: [source, destination]
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

---

## Store Schema

A Store is a **named instance** of a Connector: a specific database, bucket, or cluster
with its connection details and credentials. Secrets live **only** on the Store.

### Full Schema

```yaml
# store/<name>.yaml
apiVersion: data.infoblox.com/v1alpha1          # Required: API version
kind: Store                                     # Required: Resource type

metadata:                           # Required: Store metadata
  name: string                      # Required: Logical store name (e.g., "warehouse", "lake-raw")
  namespace: string                 # Optional: Team namespace
  labels:                           # Optional: Key-value labels
    key: value

spec:                               # Required: Store specification
  connector: string                 # Required: Name of the Connector this store is an instance of
  connection:                       # Required: Technology-specific connection parameters
    key: value                      # e.g., host, port, bucket, region, endpoint
  secrets:                          # Optional: Credential references using ${VAR} interpolation
    key: string                     # e.g., username: ${PG_USER}
```

### Field Reference

#### spec.connector

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Description | References a Connector by name (e.g., `postgres`, `s3`). |

#### spec.connection

| Property | Value |
|----------|-------|
| Type | map (string → any) |
| Required | Yes |
| Description | Technology-specific connection parameters. Structure depends on the Connector type. |

#### spec.secrets

| Property | Value |
|----------|-------|
| Type | map (string → string) |
| Required | No |
| Description | Credential references using `${VAR}` interpolation. Resolved from environment variables or a secret store at runtime. |

### Examples

#### Postgres Store

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
    host: dk-postgres-postgresql.dk-local.svc.cluster.local
    port: 5432
    database: dataplatform
    schema: public
    sslmode: disable
  secrets:
    username: ${PG_USER}
    password: ${PG_PASSWORD}
```

#### S3 Store

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
    endpoint: http://dk-localstack-localstack.dk-local.svc.cluster.local:4566
  secrets:
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
```

---

## pipeline.yaml Schema

The pipeline workflow manifest defines ordered execution steps for multi-step data pipelines.
This is a separate concept from Transform — it orchestrates multiple steps.

### Full Schema

```yaml
# pipeline.yaml
apiVersion: data.infoblox.com/v1alpha1  # Required: API version
kind: PipelineWorkflow                  # Required: Resource type

metadata:                               # Required: Pipeline metadata
  name: string                          # Required: Pipeline name (3-63 chars, lowercase)
  description: string                   # Optional: Human-readable description

steps:                                  # Required: Ordered list of steps
  - name: string                        # Required: Step name (3-63 chars, DNS-safe)
    type: string                        # Required: sync | transform | test | publish | custom
    description: string                 # Optional: Step description

    # Sync step fields
    source: string                      # Required for sync: Source asset name
    sink: string                        # Required for sync: Sink asset name

    # Transform step fields
    asset: string                       # Required for transform: Asset name

    # Test step fields
    asset: string                       # Required for test: Asset to test
    command: [string]                   # Required for test: Command and args

    # Publish step fields
    promote: boolean                    # Optional: Trigger promotion
    notify:                             # Optional: Notification config
      channels: [string]
      recipients: [string]

    # Custom step fields
    image: string                       # Required for custom: Container image
    command: [string]                   # Optional: Override entrypoint
    args: [string]                      # Optional: Container arguments

    # Common optional fields
    env:                                # Optional: Environment variables
      - name: string
        value: string
      - name: string
        valueFrom:
          secretRef:
            name: string
            key: string
```

### Step Type Requirements

| Type | Required Fields | Description |
|------|----------------|-------------|
| `sync` | `source`, `sink` | Data ingestion from source to sink |
| `transform` | `asset` | Transformation engine execution |
| `test` | `asset`, `command` | Validation and assertions |
| `publish` | — | Notification and promotion |
| `custom` | `image` | Arbitrary container execution |

### Pipeline Validation Errors

| Code | Message | Resolution |
|------|---------|------------|
| E080 | `metadata.name is required` | Add name to metadata |
| E081 | `steps list is required` | Add at least one step |
| E082 | `step name is required` | Give each step a name |
| E083 | `step type is required` | Set type: sync/transform/test/publish/custom |
| E084 | `invalid step name format` | Use 3-63 lowercase chars, hyphens allowed |
| E085 | `duplicate step name` | Make each step name unique |
| E086 | `invalid step type` | Use a valid step type |
| E087 | `sync step requires source` | Add source field to sync step |
| E088 | `sync step requires sink` | Add sink field to sync step |
| E089 | `transform step requires asset` | Add asset field to transform step |
| E090 | `test step requires asset` | Add asset field to test step |
| E091 | `custom step requires image` | Add image field to custom step |

---

## schedule.yaml Schema

Optional cron-based schedule for pipeline execution.

### Full Schema

```yaml
# schedule.yaml
apiVersion: data.infoblox.com/v1alpha1  # Required: API version
kind: Schedule                          # Required: Resource type

cron: string                            # Required: 5-field cron expression
timezone: string                        # Optional: IANA timezone (default: UTC)
suspend: boolean                        # Optional: Pause execution (default: false)
```

### Cron Expression Format

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sun=0)
│ │ │ │ │
* * * * *
```

### Schedule Examples

| Expression | Description |
|-----------|-------------|
| `0 6 * * *` | Every day at 6:00 AM |
| `*/15 * * * *` | Every 15 minutes |
| `0 0 * * 1` | Every Monday at midnight |
| `0 8 1 * *` | First day of each month at 8:00 AM |

### Schedule Validation Errors

| Code | Message | Resolution |
|------|---------|------------|
| E100 | `cron expression is required` | Add a cron field |
| E101 | `invalid cron expression` | Use valid 5-field cron syntax |
| E102 | `invalid timezone` | Use a valid IANA timezone (e.g., America/New_York) |

---

## Validation Rules Summary

### Common Errors

| Code | Message | Resolution |
|------|---------|------------|
| E001 | `metadata.name is required` | Add name field |
| E002 | `spec.runtime is required` | Add runtime field (Transform) |
| E004 | `invalid name format` | Use lowercase and hyphens only |
| E005 | `name too long` | Maximum 63 characters |
| E011 | `schema file not found: <path>` | Create the referenced schema file |

### Classification Errors

| Code | Message | Resolution |
|------|---------|------------|
| E025 | `pii=true requires classification` | Add classification level on Asset |
| E026 | `confidential requires retention` | Add retention policy |

### CloudQuery Errors

| Code | Message | Resolution |
|------|---------|------------|
| E060 | `cloudquery runtime requires Connector with plugin` | Ensure referenced Connector has plugin images |
| E061 | `Connector role is required and must be valid` | Set valid capability on Connector |
| E062 | `grpcPort must be between 1024 and 65535` | Use a valid port number |
| E063 | `concurrency must be greater than 0` | Set concurrency ≥ 1 |

---

## Example: Complete Pipeline Package

A complete pipeline that syncs Postgres tables to S3:

### Directory structure

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
│   ├── users-parquet.yaml
│   ├── orders.yaml
│   └── orders-parquet.yaml
├── asset-group/
│   └── pg-snapshot.yaml
└── dk.yaml                  # Transform manifest
```

### dk.yaml (Transform)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  namespace: analytics
  version: 0.1.0
  labels:
    team: data-engineering
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - asset: users
    - asset: orders
  outputs:
    - asset: users-parquet
    - asset: orders-parquet
  schedule:
    cron: "0 2 * * *"
  timeout: 30m
```

### connector/postgres.yaml

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  protocol: postgresql
  capabilities: [source, destination]
  plugin:
    source: ghcr.io/infobloxopen/cq-source-postgres:0.1.0
```

### store/warehouse.yaml

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: analytics
spec:
  connector: postgres
  connection:
    host: dk-postgres.svc.cluster.local
    port: 5432
    database: analytics
  secrets:
    username: ${PG_USER}
    password: ${PG_PASSWORD}
```

### asset/users.yaml (input)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
  namespace: analytics
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

### asset/users-parquet.yaml (output with lineage)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users-parquet
  namespace: analytics
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
```

---

## See Also

- [CLI Reference](cli.md) — `dk lint` and `dk init` commands
- [Concepts: Manifests](../concepts/manifests.md) — Conceptual overview of all manifest kinds
- [Concepts: Data Packages](../concepts/data-packages.md) — Package structure and runtimes
