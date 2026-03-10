---
title: DataSets
description: Understanding DataSets — named data contracts for data pipelines
---

# DataSets

A **DataSet** is a named data contract that describes a table, S3 prefix, or Kafka topic living in a Store. DataSets are **declarative metadata** — they define *what data exists*, *where it lives*, *its schema*, and *how it is classified*, without containing any runtime logic.

## What is a DataSet?

| Aspect | Description |
|--------|-------------|
| **Declarative** | No code — just a YAML file describing the data contract |
| **Store-bound** | Every DataSet references a Store that provides connection details |
| **Schema-aware** | Declares column names, types, PII flags, and lineage links |
| **Classified** | Carries data sensitivity classification (public, internal, confidential, restricted) |
| **Versioned** | Semantic version tracks contract evolution |
| **Scoped** | Each DataSet belongs to a namespace and is referenced by Transforms |

## DataSet Structure

DataSets live in the `dataset/` directory (or any directory you choose) alongside
other manifests:

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

### DataSet manifest

Every DataSet is defined by a YAML file with `kind: DataSet`:

```yaml title="dataset/users.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: users                         # DNS-safe, 1-63 characters
  namespace: default
  version: 1.0.0                      # Semantic version
  labels:
    team: data-engineering
    domain: identity
spec:
  store: warehouse                    # References a Store by name
  table: public.users                 # Table name (for relational stores)
  classification: confidential        # Data sensitivity level
  schema:
    - name: id
      type: integer
    - name: email
      type: string
      pii: true
    - name: created_at
      type: timestamp
```

## Location Types

A DataSet must specify at least one of `table`, `prefix`, or `topic` to identify
where the data lives within the Store:

| Field | Store type | Example |
|-------|-----------|---------|
| `table` | Relational databases (Postgres, Snowflake) | `public.users` |
| `prefix` | Object stores (S3) | `data/users/` |
| `topic` | Streaming platforms (Kafka) | `user-events` |

## Schema and Lineage

DataSets declare their columns in the `spec.schema` array. Each column can
optionally carry a `from` field that links it to a column in another DataSet,
establishing **column-level lineage**:

```yaml title="dataset/users-parquet.yaml"
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
      from: users.id                  # Lineage: came from users.id
    - name: email
      type: string
      pii: true
      from: users.email               # Lineage: came from users.email
    - name: created_at
      type: timestamp
      from: users.created_at
```

## Classification

Every DataSet should declare a `classification` level:

| Level | Description |
|-------|-------------|
| `public` | Non-sensitive, publicly shareable data |
| `internal` | Internal-only, no PII |
| `confidential` | Contains PII or sensitive business data |
| `restricted` | Highly regulated data (financial, health) |

When any column has `pii: true`, classification is **required** (enforced by `dk lint`).

## Seed Data for Local Development

DataSets can declare sample data in a `dev.seed` section. This data is loaded
into the backing database during local development so that your pipeline has
real rows to process without manual SQL or external fixtures.

```yaml title="dataset/source.yaml"
spec:
  store: warehouse
  table: example_table
  schema:
    - name: id
      type: integer
    - name: name
      type: string
  dev:
    seed:
      inline:
        - { id: 1, name: "alice" }
        - { id: 2, name: "bob" }
```

### How It Works

1. `dk dev seed` (or the auto-seed that runs before `dk run`) reads every
   input DataSet in the package.
2. For each DataSet with a `dev.seed` section, it generates `CREATE TABLE IF
   NOT EXISTS` + `INSERT` statements and executes them against the local
   PostgreSQL instance via `kubectl exec`.
3. A SHA-256 checksum of the seed data is stored in a `_dk_seed_meta` table.
   On subsequent runs, unchanged data is skipped automatically.
4. When data *does* change, the table is `TRUNCATE`d before inserting so the
   contents always match the seed spec — no duplicates, no stale rows.

### Seed Profiles

You can define **named profiles** for different test scenarios under
`dev.seed.profiles`. Each profile has its own `inline` rows or seed `file`:

```yaml title="dataset/source.yaml"
dev:
  seed:
    inline:
      - { id: 1, name: "alice" }      # default profile
    profiles:
      large-dataset:
        file: testdata/large.csv       # file-based profile
      edge-cases:
        inline:
          - { id: -1, name: "" }
          - { id: 999, name: "O'Reilly" }
      empty: {}                        # empty table for testing
```

Activate a profile with `dk dev seed --profile <name>`:

```bash
# Default seed data
dk dev seed

# Switch to the edge-cases profile
dk dev seed --profile edge-cases

# Force re-seed even if unchanged
dk dev seed --force
```

!!! tip "Seed files"
    Seed files can be CSV or JSON. Place them in your package directory and
    reference them with a relative path (e.g., `testdata/data.csv`).

## CLI Commands

| Command | Description |
|---------|-------------|
| `dk dataset list` | List all DataSets in the project |
| `dk dataset show <name>` | Show full DataSet details |
| `dk dataset validate [path]` | Validate DataSet configuration |
| `dk dev seed` | Load seed data into local dev stores |
| `dk dev seed --profile <name>` | Use a named seed profile |

## Relationship to Other Concepts

```
Connector (technology type)
       │
       ▼
  Store (named instance with credentials)
       │
       ▼
  DataSet (data contract: table/prefix/topic + schema)
       │
       ▼
  Transform (dk.yaml — reads input DataSets, writes output DataSets)
       │
       ▼
  Build → Publish → Promote
```

- **Stores** provide infrastructure-specific connection details (buckets, topics, databases)
- **DataSets** declare what data lives in a Store — the schema, classification, and lineage
- **Transforms** reference DataSets by name in their `spec.inputs` and `spec.outputs`
- **DataSetGroups** bundle multiple DataSets produced by a single materialisation

## See Also

- [Data Packages](data-packages.md) — The container for DataSets
- [Manifests](manifests.md) — dk.yaml and manifest reference
- [CLI Reference](../reference/cli.md) — Complete CLI documentation
- [Manifest Schema](../reference/manifest-schema.md) — Schema reference
