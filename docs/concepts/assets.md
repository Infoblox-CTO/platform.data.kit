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
| **Scoped** | Each asset belongs to a data package via `dp.yaml` |

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
├── dp.yaml
├── bindings.yaml
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
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
name: aws-security                    # DNS-safe, 3-63 characters
type: source                          # source | sink | model-engine
extension: cloudquery.source.aws      # Extension FQN (vendor.kind.name)
version: v24.0.2                      # Extension version (semver)
ownerTeam: security-data              # Team that owns this asset
description: "AWS security data"      # Optional description
binding: aws-raw-output               # Optional binding reference
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

Each extension publishes a JSON Schema that defines the allowed configuration fields. When you run `dp asset validate`, the asset's `config` block is validated against this schema:

```bash
# Validate a single asset
dp asset validate assets/sources/aws-security/

# Validate all assets in the project
dp asset validate
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
| E076 | Asset referenced in dp.yaml not found on disk |
| E077 | Asset binding references non-existent binding |

## Lifecycle

### 1. Create

Scaffold an asset from an extension:

```bash
dp asset create aws-security --ext cloudquery.source.aws
```

The scaffolder resolves the extension's JSON Schema and generates placeholder config with required fields.

### 2. Configure

Edit the generated `asset.yaml` to fill in your configuration values. Set `ownerTeam` to your team name.

### 3. Validate

Validate the config against the extension schema:

```bash
dp asset validate
```

### 4. Reference

Add the asset to your `dp.yaml`:

```yaml
spec:
  assets:
    - aws-security
```

### 5. Build & Deploy

Assets are included in the normal `dp build → dp publish → dp promote` workflow.

## Bindings

Assets can optionally reference a named binding from `bindings.yaml`:

```yaml
# asset.yaml
binding: aws-raw-output
```

```yaml
# bindings.yaml
bindings:
  - name: aws-raw-output
    asset: aws-security
    type: s3-prefix
    properties:
      bucket: my-bucket
      prefix: raw/security/
```

The `dp validate` command cross-validates that each asset's `binding` field resolves to an existing entry in `bindings.yaml`.

## CLI Commands

| Command | Description |
|---------|-------------|
| `dp asset create <name> --ext <fqn>` | Scaffold a new asset |
| `dp asset validate [path]` | Validate asset configuration |
| `dp asset list` | List all assets in the project |
| `dp asset show <name>` | Show full asset details |

## Relationship to Other Concepts

```
Extension Schema (JSON Schema)
       │
       ▼
  Asset (asset.yaml)  ←──→  Binding (bindings.yaml)
       │
       ▼
  Data Package (dp.yaml)
       │
       ▼
  Build → Publish → Promote
```

- **Extensions** provide the runtime and schema; assets provide the configuration
- **Data Packages** reference assets by name in their `spec.assets` list
- **Bindings** provide infrastructure-specific configuration (buckets, topics, etc.)

## See Also

- [Data Packages](data-packages.md) — The container for assets
- [Manifests](manifests.md) — dp.yaml and bindings.yaml reference
- [CLI Reference](../reference/cli.md) — Complete CLI documentation
- [Manifest Schema](../reference/manifest-schema.md) — Schema reference
