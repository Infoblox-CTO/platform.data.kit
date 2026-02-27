---
title: Manifests
description: Complete reference for data package manifest files
---

# Manifests

The manifest (`dp.yaml`) is the central configuration file for every data package. It defines metadata, runtime, inputs, outputs, and governance requirements.

## Manifest Kinds

The platform defines five manifest kinds:

| Kind | Purpose | Who creates |
|------|---------|-------------|
| **Transform** | Computation — reads inputs, produces outputs | Data engineer |
| **Asset** | Data contract — schema, classification, lineage | Data engineer |
| **AssetGroup** | Bundle of Assets produced by one Transform | Data engineer |
| **Connector** | Technology type — Postgres, S3, Kafka, etc. | Platform team |
| **Store** | Named instance of a Connector with credentials | Infra / SRE |

## Transform Manifest

The Transform is the primary manifest kind for data packages:

```yaml title="dp.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: my-package
  namespace: default
  version: 0.1.0
spec:
  runtime: generic-python       # cloudquery | generic-go | generic-python | dbt
  mode: batch                   # batch | streaming
  image: myorg/my-package:v0.1.0
  timeout: 30m

  inputs:
    - asset: source-data

  outputs:
    - asset: output-data
```

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `apiVersion` | string | Always `data.infoblox.com/v1alpha1` |
| `kind` | string | `Transform`, `Asset`, `AssetGroup`, `Connector`, or `Store` |
| `metadata.name` | string | Package name (lowercase, hyphenated) |
| `spec.runtime` | string | One of: `cloudquery`, `generic-go`, `generic-python`, `dbt` |

### metadata

```yaml
metadata:
  name: my-kafka-pipeline          # Required: unique package name
  namespace: analytics             # Optional: logical grouping
  version: 1.0.0                   # Semantic version
  labels:                          # Optional: key-value labels
    team: data-engineering
    domain: events
    cost-center: analytics
  annotations:                     # Optional: arbitrary metadata
    dp.io/documentation: https://wiki.example.com/my-pipeline
```

#### Naming Rules

- **name**: 1-63 characters, lowercase alphanumeric and hyphens
- **namespace**: 1-63 characters, lowercase alphanumeric and hyphens
- **labels**: Keys up to 63 chars, values up to 253 chars

### spec.inputs

Inputs declare which Assets a Transform reads:

```yaml
spec:
  inputs:
    - asset: raw-events            # Reference asset by name
    - asset: user-metadata         # Multiple inputs supported
    - tags:                        # Or match by labels
        domain: analytics
        tier: raw
      version: ">=1.0.0"           # Optional semver constraint
```

Each input (and output) uses either `asset` (exact name) or `tags` (label selector) — not both.

### spec.outputs

Outputs declare which Assets a Transform produces:

```yaml
spec:
  outputs:
    - asset: enriched-events       # Asset name to write to
```

!!! note
    Data classification (`pii`, `sensitivity`) is declared on the **Asset** manifest, not on the Transform's AssetRef.

## Asset Manifest

An Asset declares a data contract — a table, S3 prefix, or topic that lives in a Store:

```yaml title="asset/users.yaml"
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

Output Assets can use `from` for column-level lineage:

```yaml title="asset/users-parquet.yaml"
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
```

### Classification Fields

| Field | Type | Description |
|-------|------|-------------|
| `pii` | boolean | Contains personally identifiable information |
| `sensitivity` | string | `internal`, `confidential`, or `restricted` |
| `retention.days` | integer | Retention period in days |
| `retention.deletionPolicy` | string | `delete` or `archive` |
| `tags` | array | Custom classification tags |

## Connector Manifest

A Connector describes a storage technology type:

```yaml title="connector/postgres.yaml"
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

## Store Manifest

A Store is a named instance of a Connector with connection details and credentials:

```yaml title="store/warehouse.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: default
spec:
  connector: postgres
  connection:
    host: dp-postgres-postgresql.dp-local.svc.cluster.local
    port: 5432
    database: dataplatform
    schema: public
  secrets:
    username: ${PG_USER}
    password: ${PG_PASSWORD}
```

!!! note "Secrets on Stores only"
    Credentials live **only** on Store manifests — never on Assets or Transforms.

## Validation

Validate your manifest:

```bash
dp lint ./my-package
```

The linter checks:

- ✓ Required fields present
- ✓ Valid names (lowercase, hyphenated)
- ✓ Valid kind and runtime values
- ✓ Schema files exist if specified
- ✓ Classification is valid for outputs

### Common Validation Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid name` | Uppercase or special chars | Use lowercase and hyphens only |
| `unsupported kind` | Kind not one of the five valid kinds | Use Transform, Asset, AssetGroup, Connector, or Store |
| `schema not found` | Schema file doesn't exist | Create file or remove reference |
| `pii without sensitivity` | PII true but no sensitivity level | Add sensitivity classification |

## Next Steps

- [Data Packages](data-packages.md) - Package structure overview
- [Lineage](lineage.md) - How manifests enable lineage
- [Configuration Reference](../reference/configuration.md) - Full config options
