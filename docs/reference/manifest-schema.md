---
title: Manifest Schema
description: Complete reference for all manifest schemas (Transform, DataSet, DataSetGroup, Connector, Store)
---

# Manifest Schema Reference

This document provides the complete schema reference for all manifest kinds in the Data Platform Kit.

The platform uses five manifest kinds, each with a clear ownership boundary:

| Kind | Owner | Purpose |
|------|-------|---------|
| **Connector** | Platform team | Versioned technology type with tools and connection schema |
| **Store** | Infra / SRE | Named instance of a Connector with connection details and secrets |
| **DataSet** | Data engineer | Data contract — table, prefix, or topic in a Store |
| **DataSetGroup** | Data engineer | Bundle of DataSets produced by a single materialisation |
| **Transform** | Data engineer | Unit of computation reading input DataSets, producing output DataSets |

---

## Transform Schema

The Transform manifest (`dk.yaml`) defines a unit of computation.
It is the **only manifest kind that runs** — Connectors, Stores, and DataSets are declarative metadata.

### Full Schema

```yaml
# dk.yaml
apiVersion: datakit.infoblox.dev/v1alpha1          # Required: API version
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

  inputs:                           # Required: Input DataSet references
    - dataset: string               # DataSet name (local or OCI ref)
      tags:                         # OR match DataSets by labels
        key: value
      version: string               # Optional: Semver range constraint
      cell: string                  # Optional: Cell/region constraint

  outputs:                          # Required: Output DataSet references
    - dataset: string               # DataSet name (local or OCI ref)
      tags:                         # OR match DataSets by labels
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
| Type | array of DataSetRef objects |
| Required | Yes |
| Description | Input/output DataSet references. Each entry uses either `dataset` (named reference) or `tags` (label matching) — not both. |

**DataSetRef fields:**

| Field | Type | Description |
|-------|------|-------------|
| `dataset` | string | Named DataSet reference (local name or OCI ref) |
| `tags` | map(string→string) | Match DataSets by labels (alternative to `dataset`) |
| `version` | string | Semver range constraint (e.g., `>=1.0.0`) |
| `cell` | string | Cell/region constraint |

At runtime, the runner resolves the chain: **Transform → DataSet → Store → Connector** to obtain connection details and plugin images.

#### spec.trigger

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Description | Reactive trigger policy. Defines when this transform executes. |

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
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  namespace: default
  version: 0.1.0
  labels:
    team: datakit
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - dataset: users
  outputs:
    - dataset: users-parquet
  trigger:
    policy: schedule
    schedule:
      cron: "0 */6 * * *"
  timeout: 30m
```

#### Generic Python Transform

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: enrich-users
  namespace: default
  version: 0.2.0
spec:
  runtime: generic-python
  mode: batch
  inputs:
    - dataset: users-parquet
  outputs:
    - dataset: users-enriched
  image: my-team/enrich-users:latest
  timeout: 15m
```

#### Streaming Transform

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: event-processor
  namespace: default
  version: 1.0.0
spec:
  runtime: generic-go
  mode: streaming
  inputs:
    - dataset: raw-events
  outputs:
    - dataset: processed-events
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
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: enrich
  version: 0.3.0
spec:
  runtime: generic-python
  inputs:
    - dataset: raw-events-parquet
  outputs:
    - dataset: enriched-events
  image: my-team/enrich:latest
  trigger:
    policy: on-change
```

#### Tag-based Input Resolution

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
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
    - dataset: analytics-summary
  image: my-team/dbt-runner:latest
  trigger:
    policy: schedule
    schedule:
      cron: "0 */6 * * *"
```

---

## DataSet Schema

A DataSet is a **named data contract**: a table, S3 prefix, or Kafka topic that lives in a Store.
It declares the schema (with optional column-level lineage via `from`) and data classification.

### Full Schema

```yaml
# dataset/<name>.yaml
apiVersion: datakit.infoblox.dev/v1alpha1          # Required: API version
kind: DataSet                                   # Required: Resource type

metadata:                           # Required: DataSet metadata
  name: string                      # Required: DataSet name (1-63 chars, lowercase, hyphenated)
  namespace: string                 # Optional: Team namespace
  version: string                   # Optional: Semantic version (e.g., "1.0.0")
  labels:                           # Optional: Key-value labels
    key: value

spec:                               # Required: DataSet specification
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
      from: string                  # Optional: Lineage source (e.g., "users.id") — links to another DataSet

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
| Description | DNS-safe DataSet identifier. |

#### spec.store

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Description | Name of the Store manifest this DataSet lives in. |

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
| Format | `<dataset-name>.<field-name>` |
| Description | Declares column-level lineage. Links this field to its source in another DataSet. |

### DataSet Validation Errors

| Code | Message | Resolution |
|------|---------|------------|
| E025 | `pii=true requires classification` | Add classification level |
| E026 | `confidential requires retention policy` | Add retention policy |

### Examples

#### Input DataSet (database table)

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
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

#### Output DataSet with column-level lineage

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
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

#### Input DataSet with seed data and profiles

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
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

## DataSetGroup Schema

A DataSetGroup bundles multiple DataSets produced by a single materialisation
(e.g., a CloudQuery sync that snapshots several tables at once).

### Full Schema

```yaml
# dataset-group/<name>.yaml
apiVersion: datakit.infoblox.dev/v1alpha1          # Required: API version
kind: DataSetGroup                              # Required: Resource type

metadata:                           # Required: DataSetGroup metadata
  name: string                      # Required: Group name (1-63 chars, lowercase, hyphenated)
  namespace: string                 # Optional: Team namespace
  labels:                           # Optional: Key-value labels
    key: value

spec:                               # Required: DataSetGroup specification
  store: string                     # Required: Common Store for all DataSets in the group
  datasets:                         # Required: List of DataSet names
    - string
```

### Example

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSetGroup
metadata:
  name: pg-snapshot
  namespace: default
spec:
  store: lake-raw
  datasets:
    - users-parquet
    - orders-parquet
    - products-parquet
```

---

## Connector Schema

A Connector describes a storage technology type (e.g., Postgres, S3, Kafka).
Maintained by the **platform team**. Multiple versions of the same connector
(identified by `spec.provider`) can coexist — each as a separate CR with a
unique `metadata.name` (e.g., `postgres-1-2-0`, `postgres-1-3-0`).

### Full Schema

```yaml
# connector/<name>.yaml
apiVersion: datakit.infoblox.dev/v1alpha1          # Required: API version
kind: Connector                                 # Required: Resource type

metadata:                           # Required: Connector metadata
  name: string                      # Required: CR instance name (e.g., "postgres-1-2-0")
  labels:                           # Optional: Key-value labels (convention: datakit.infoblox.dev/provider)
    key: value
  annotations:                      # Optional: Arbitrary annotations
    key: value

spec:                               # Required: Connector specification
  provider: string                  # Optional: Logical identity stores reference (defaults to type)
  version: string                   # Optional: Semantic version of this release (e.g., "1.2.0")
  type: string                      # Required: Technology identifier (postgres, s3, kafka, snowflake)
  protocol: string                  # Optional: Wire protocol (postgresql, s3, kafka)
  capabilities:                     # Required: Supported roles
    - string                        # "source", "destination", or both

  plugin:                           # Optional: CloudQuery plugin images
    source: string                  # CQ source plugin image
    destination: string             # CQ destination plugin image

  tools:                            # Optional: Technology-specific actions
    - name: string                  # Required: Tool identifier (e.g., "psql", "dsn")
      description: string           # Optional: Human-readable summary
      type: string                  # Required: "exec" (run command) or "config" (generate output)
      requires: [string]            # Optional: Binaries that must be on $PATH
      command: string               # Optional: Go template for shell command (type=exec)
      format: string                # Optional: Output format — "text", "file", or "env" (type=config)
      path: string                  # Optional: File path to write to (format=file)
      mode: string                  # Optional: "append" or "overwrite" (format=file)
      template: string              # Optional: Go template for output content (type=config)
      postMessage: string           # Optional: Go template displayed after execution
      default: boolean              # Optional: Default tool when none is specified

  connectionSchema:                 # Optional: Structured connection field declarations
    <logical-name>:
      field: string                 # Required: Key in Store.spec.connection
      description: string           # Optional: Human-readable explanation
      default: string               # Optional: Fallback value when absent
      secret: boolean               # Optional: May also be fulfilled from Store.spec.secrets
      optional: boolean             # Optional: Field is not required
```

### Field Reference

#### spec.provider

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Default | Falls back to `spec.type` if omitted |
| Examples | `postgres`, `s3`, `kafka` |
| Description | Logical connector identity that Stores reference via `spec.connector`. Multiple CRs can share the same provider — one per version. |

#### spec.version

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Format | Semantic version (e.g., `1.2.0`) |
| Description | Version of this connector release. Used with `connectorVersion` on Stores for version-constrained resolution. |

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

#### spec.tools

| Property | Value |
|----------|-------|
| Type | array of ConnectorTool objects |
| Required | No |
| Description | Technology-specific actions this connector exposes (e.g., launch psql, generate DSN, mount S3 bucket). |

**ConnectorTool fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Tool identifier (e.g., `psql`, `dsn`, `mount`) |
| `description` | string | No | Human-readable summary |
| `type` | string | Yes | `exec` (run a command) or `config` (generate output) |
| `requires` | array of strings | No | Binary names that must be on `$PATH` |
| `command` | string | No | Go template for shell command (`type=exec`) |
| `format` | string | No | Output format: `text`, `file`, or `env` (`type=config`) |
| `template` | string | No | Go template for output content (`type=config`) |
| `default` | boolean | No | Mark as default tool when none is specified |

#### spec.connectionSchema

| Property | Value |
|----------|-------|
| Type | map of ConnectionSchemaField objects |
| Required | No |
| Description | Declares the structured connection fields this connector expects. Maps logical field names to `Store.spec.connection` keys. |

**ConnectionSchemaField fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `field` | string | Yes | Key name in `Store.spec.connection` |
| `description` | string | No | Human-readable explanation |
| `default` | string | No | Fallback value when absent from the store |
| `secret` | boolean | No | May also be fulfilled from `Store.spec.secrets` |
| `optional` | boolean | No | Field is not required |

### Examples

#### Basic Connector (single version)

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
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

#### Versioned Connector with tools and connectionSchema

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: postgres-1-3-0
  labels:
    datakit.infoblox.dev/provider: postgres
spec:
  provider: postgres
  version: 1.3.0
  type: postgres
  protocol: postgresql
  capabilities: [source, destination]
  plugin:
    source: ghcr.io/infobloxopen/cq-source-postgres:v1.3.0
    destination: ghcr.io/infobloxopen/cq-destination-postgresql:v8.14.1
  tools:
    - name: psql
      description: Launch interactive psql session
      type: exec
      requires: [psql]
      command: "psql {{.DSN}}"
      default: true
    - name: dsn
      description: Print connection string
      type: config
      format: text
      template: "postgresql://{{.Connection.host}}:{{.Connection.port}}/{{.Connection.database}}"
  connectionSchema:
    host:
      field: host
      description: Database hostname
      default: localhost
    port:
      field: port
      description: Database port
      default: "5432"
    database:
      field: database
      description: Database name
    username:
      field: username
      description: Database user
      secret: true
    password:
      field: password
      description: Database password
      secret: true
```

#### Destination-only Connector

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
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

The `spec.connector` field references a Connector by its **provider** identity
(i.e., `spec.provider` on the Connector, NOT `metadata.name`). This allows
multiple connector versions to coexist while stores reference the logical type.

### Full Schema

```yaml
# store/<name>.yaml
apiVersion: datakit.infoblox.dev/v1alpha1          # Required: API version
kind: Store                                     # Required: Resource type

metadata:                           # Required: Store metadata
  name: string                      # Required: Logical store name (e.g., "warehouse", "lake-raw")
  namespace: string                 # Optional: Team namespace
  labels:                           # Optional: Key-value labels
    key: value
  annotations:                      # Optional: Arbitrary annotations
    key: value

spec:                               # Required: Store specification
  connector: string                 # Required: Provider name of the Connector (references spec.provider)
  connectorVersion: string          # Optional: Semver range constraining compatible versions
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
| Description | Provider name of the Connector this store is an instance of. References `spec.provider` on the Connector (e.g., `postgres`, `s3`), **not** the CR `metadata.name`. |

#### spec.connectorVersion

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Format | Semver range (e.g., `^1.0.0`, `>=1.2.0 <2.0.0`) |
| Default | Highest available version of the named provider |
| Description | Constrains which connector versions this store is compatible with. When omitted, the platform selects the highest available version. |

#### spec.connection

| Property | Value |
|----------|-------|
| Type | map (string → any) |
| Required | Yes |
| Description | Technology-specific connection parameters. Keys should match the Connector's `connectionSchema` field definitions. |

#### spec.secrets

| Property | Value |
|----------|-------|
| Type | map (string → string) |
| Required | No |
| Description | Credential references using `${VAR}` interpolation. Resolved from environment variables or a secret store at runtime. |

### Examples

#### Postgres Store

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: default
  labels:
    team: datakit
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
apiVersion: datakit.infoblox.dev/v1alpha1
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

#### Store with version-constrained Connector

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: analytics-db
  namespace: analytics
spec:
  connector: postgres
  connectorVersion: ">=1.2.0 <2.0.0"
  connection:
    host: analytics-primary.internal
    port: 5432
    database: analytics
  secrets:
    username: ${ANALYTICS_PG_USER}
    password: ${ANALYTICS_PG_PASSWORD}
```

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
| E025 | `pii=true requires classification` | Add classification level on DataSet |
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
├── dataset/
│   ├── users.yaml
│   ├── users-parquet.yaml
│   ├── orders.yaml
│   └── orders-parquet.yaml
├── dataset-group/
│   └── pg-snapshot.yaml
└── dk.yaml                  # Transform manifest
```

### dk.yaml (Transform)

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
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
    - dataset: users
    - dataset: orders
  outputs:
    - dataset: users-parquet
    - dataset: orders-parquet
  trigger:
    policy: schedule
    schedule:
      cron: "0 2 * * *"
  timeout: 30m
```

### connector/postgres.yaml

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
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
apiVersion: datakit.infoblox.dev/v1alpha1
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

### dataset/users.yaml (input)

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
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

### dataset/users-parquet.yaml (output with lineage)

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
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
- [Concepts: DataSets](../concepts/datasets.md) — DataSet data contracts
- [Concepts: Data Packages](../concepts/data-packages.md) — Package structure and runtimes
