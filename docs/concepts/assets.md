---
title: Assets
description: Understanding asset instances — configured extensions for data pipelines
---

# Assets

An **asset** is a configured instance of an approved extension. Assets are **config-only** — you declare *what to run* and *how to configure it*, while the extension provides the runtime implementation.

## What is an Asset?

| Aspect | Description |
|--------|-------------|
| **Config-only** | No code — just a YAML file with configuration |
| **Schema-validated** | Config is validated against the extension's JSON Schema |
| **Typed** | Source, sink, or model-engine — derived from the extension |
| **Versioned** | Pinned to a specific extension version (semver) |
| **Scoped** | Each asset belongs to a data package via `dk.yaml` |

## Asset Types

Assets come in three types, determined by the extension's kind:

| Type | Directory | Description | Example |
|------|-----------|-------------|---------|
| **source** | `assets/sources/` | Pulls data into the platform | CloudQuery AWS source |
| **sink** | `assets/sinks/` | Pushes data to external destinations | S3 output, PostgreSQL sink |
| **model-engine** | `assets/models/` | Transforms data in-place | dbt model runner |

## Asset Structure

Each asset lives in a type-based directory under `assets/`:

```
my-package/
├── dk.yaml
└── assets/
    ├── sources/
    │   ├── aws-security/
    │   │   └── asset.yaml
    │   └── gcp-infra/
    │       └── asset.yaml
    └── sinks/
        └── raw-output/
            └── asset.yaml
```

### asset.yaml

Every asset is defined by a single `asset.yaml` file:

```yaml title="asset.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
name: aws-security                    # DNS-safe, 3-63 characters
type: source                          # source | sink | model-engine
extension: cloudquery.source.aws      # Extension FQN (vendor.kind.name)
version: v24.0.2                      # Extension version (semver)
ownerTeam: security-data              # Team that owns this asset
description: "AWS security data"      # Optional description
config:                               # Validated against extension schema
  accounts:
    - "123456789012"
  regions:
    - us-east-1
    - us-west-2
  tables:
    - aws_s3_buckets
    - aws_iam_roles
```

## Extensions and FQNs

Assets reference extensions using a **fully-qualified name (FQN)** with three segments:

```
<vendor>.<kind>.<name>
```

| Segment | Description | Example |
|---------|-------------|---------|
| **vendor** | Organization that provides the extension | `cloudquery` |
| **kind** | Asset type (`source`, `sink`, `model-engine`) | `source` |
| **name** | Specific extension name | `aws` |

Examples:

- `cloudquery.source.aws` — CloudQuery AWS source plugin
- `cloudquery.source.gcp` — CloudQuery GCP source plugin
- `cloudquery.sink.s3` — CloudQuery S3 sink plugin

## Schema Validation

Each extension publishes a JSON Schema that defines the allowed configuration fields. When you run `dk asset validate`, the asset's `config` block is validated against this schema:

```bash
# Validate a single asset
dk asset validate assets/sources/aws-security/

# Validate all assets in the project
dk asset validate
```

### Error Codes

| Code | Description |
|------|-------------|
| E070 | Required field missing (name, type, extension, etc.) |
| E071 | Invalid extension FQN format |
| E072 | Invalid version format (must be semver) |
| E073 | Asset type does not match extension kind |
| E074 | Config block fails schema validation |
| E075 | Extension schema not found |
| E076 | Asset referenced in dk.yaml not found on disk |

## Lifecycle

### 1. Create

Scaffold an asset from an extension:

```bash
dk asset create aws-security --ext cloudquery.source.aws
```

The scaffolder resolves the extension's JSON Schema and generates placeholder config with required fields.

### 2. Configure

Edit the generated `asset.yaml` to fill in your configuration values. Set `ownerTeam` to your team name.

### 3. Validate

Validate the config against the extension schema:

```bash
dk asset validate
```

### 4. Reference

Add the asset to your `dk.yaml`:

```yaml
spec:
  assets:
    - aws-security
```

### 5. Build & Deploy

Assets are included in the normal `dk build → dk publish → dk promote` workflow.

## Store References

Assets reference a Store by name via the `spec.store` field in the data package manifest. Store manifests define connection details for infrastructure:

```yaml
# store.yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: aws-raw-output
spec:
  type: s3-prefix
  connection:
    bucket: my-bucket
    prefix: raw/security/
```

The `dk validate` command cross-validates that each asset's store reference resolves to an existing Store manifest.

## Seed Data for Local Development

Assets can declare sample data in a `dev.seed` section. This data is loaded
into the backing database during local development so that your pipeline has
real rows to process without manual SQL or external fixtures.

```yaml title="asset/source.yaml"
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
   input asset in the package.
2. For each asset with a `dev.seed` section, it generates `CREATE TABLE IF
   NOT EXISTS` + `INSERT` statements and executes them against the local
   PostgreSQL instance via `kubectl exec`.
3. A SHA-256 checksum of the seed data is stored in a `_dp_seed_meta` table.
   On subsequent runs, unchanged data is skipped automatically.
4. When data *does* change, the table is `TRUNCATE`d before inserting so the
   contents always match the seed spec — no duplicates, no stale rows.

### Seed Profiles

You can define **named profiles** for different test scenarios under
`dev.seed.profiles`. Each profile has its own `inline` rows or seed `file`:

```yaml title="asset/source.yaml"
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
| `dk asset create <name> --ext <fqn>` | Scaffold a new asset |
| `dk asset validate [path]` | Validate asset configuration |
| `dk asset list` | List all assets in the project |
| `dk asset show <name>` | Show full asset details |
| `dk dev seed` | Load seed data into local dev stores |
| `dk dev seed --profile <name>` | Use a named seed profile |

## Relationship to Other Concepts

```
Extension Schema (JSON Schema)
       │
       ▼
  Asset (asset.yaml)  ←──→  Store (store.yaml)
       │
       ▼
  Data Package (dk.yaml)
       │
       ▼
  Build → Publish → Promote
```

- **Extensions** provide the runtime and schema; assets provide the configuration
- **Data Packages** reference assets by name in their `spec.assets` list
- **Stores** provide infrastructure-specific connection details (buckets, topics, etc.)

## See Also

- [Data Packages](data-packages.md) — The container for assets
- [Manifests](manifests.md) — dk.yaml and store manifest reference
- [CLI Reference](../reference/cli.md) — Complete CLI documentation
- [Manifest Schema](../reference/manifest-schema.md) — Schema reference
