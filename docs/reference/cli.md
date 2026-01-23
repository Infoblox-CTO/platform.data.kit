---
title: CLI Reference
description: Complete reference for all dp CLI commands
---

# CLI Reference

Complete reference for all `dp` CLI commands with examples and flags.

## Global Flags

These flags apply to all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--output` | `-o` | Output format (table, json, yaml) | table |
| `--help` | `-h` | Show help | - |
| `--version` | `-v` | Show version | - |

---

## Commands Overview

| Command | Description |
|---------|-------------|
| [`dp init`](#dp-init) | Create a new data package |
| [`dp dev`](#dp-dev) | Manage local development stack |
| [`dp lint`](#dp-lint) | Validate package manifests |
| [`dp run`](#dp-run) | Execute pipeline locally |
| [`dp show`](#dp-show) | Show effective manifest |
| [`dp test`](#dp-test) | Run pipeline tests |
| [`dp build`](#dp-build) | Build OCI artifact |
| [`dp publish`](#dp-publish) | Publish to registry |
| [`dp promote`](#dp-promote) | Promote to environment |
| [`dp status`](#dp-status) | Show package status |
| [`dp logs`](#dp-logs) | Stream logs |
| [`dp rollback`](#dp-rollback) | Rollback to previous version |
| [`dp lineage`](#dp-lineage) | View data lineage |

---

## dp init

Create a new data package.

```bash
dp init <package-name> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--type` | Package type (pipeline, producer, consumer, streaming) | pipeline |
| `--dir` | Directory to create package in | . |

### Examples

```bash
# Create a pipeline package
dp init my-pipeline --type pipeline
```

```bash
# Create in specific directory
dp init kafka-processor --dir ./packages
```

### Output

Creates the following directory structure:

```
my-pipeline/
├── dp.yaml
├── bindings.yaml
└── src/
    └── main.py
```

The generated `dp.yaml` includes a `spec.runtime` section for pipeline packages.

---

## dp dev

Manage the local development stack.

### dp dev up

Start the local development stack.

```bash
dp dev up [flags]
```

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--detach` | Run in background | false |
| `--timeout` | Startup timeout | 60s |

#### Examples

```bash
# Start local stack
dp dev up
```

```bash
# Start in background
dp dev up --detach
```

### dp dev down

Stop the local development stack.

```bash
dp dev down [flags]
```

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--volumes` | Remove volumes | false |

#### Examples

```bash
# Stop stack
dp dev down
```

```bash
# Stop and remove volumes
dp dev down --volumes
```

### dp dev status

Show status of local development stack.

```bash
dp dev status
```

#### Output Example

```
Service         Status    Ports
━━━━━━━━━━━━━━  ━━━━━━━   ━━━━━━━━━
kafka           running   9092
minio           running   9000
marquez         running   5000
postgres        running   5432
```

---

## dp lint

Validate package manifests.

```bash
dp lint [package-dir] [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--strict` | - | Treat warnings as errors | false |
| `--skip-pii` | - | Skip PII classification validation | false |
| `--set` | - | Override values (key=value, repeatable) | - |
| `--values` | `-f` | Override files (repeatable) | - |

### Validated Files

| File | Description |
|------|-------------|
| `dp.yaml` | Package manifest (includes runtime config) |
| `bindings.yaml` | Binding configuration |
| `schemas/` | Schema files |

### Validation Rules

| Code | Description |
|------|-------------|
| E001-E003 | Required fields |
| E004-E005 | Schema references |
| E010-E011 | Binding configuration |
| E025 | PII classification required |
| E030-E031 | Runtime configuration |
| E040-E041 | Runtime required for pipeline type |

### Examples

```bash
# Lint current directory
dp lint
```

```bash
# Lint specific package
dp lint ./my-pipeline
```

```bash
# Lint with overrides applied
dp lint ./my-pipeline -f production.yaml

# Lint with inline override
dp lint ./my-pipeline --set spec.runtime.image=myimage:v2
```

```bash
# Strict mode (warnings become errors)
dp lint --strict
```

---

## dp run

Execute pipeline locally.

```bash
dp run [package-dir] [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--network` | - | Docker network | dp-network |
| `--env` | - | Environment variables (KEY=VALUE) | - |
| `--bindings` | - | Bindings file path | bindings.yaml |
| `--dry-run` | - | Print what would run | false |
| `--detach` | - | Run in background | false |
| `--timeout` | - | Execution timeout | 30m |
| `--set` | - | Override values (key=value, repeatable) | - |
| `--values` | `-f` | Override files (repeatable) | - |

### Runtime Configuration

The pipeline runs using the container image and configuration specified in `spec.runtime` of dp.yaml.
Environment variables are automatically mapped from bindings (e.g., `input.events.brokers` → `INPUT_EVENTS_BROKERS`).

### Override Precedence

When using `-f` and `--set` flags:

1. **Base**: dp.yaml values
2. **Files**: Values from `-f` files (applied in order)
3. **Set flags**: `--set` values (applied in order, highest precedence)

### Examples

```bash
# Run pipeline
dp run ./my-pipeline
```

```bash
# With custom bindings
dp run ./my-pipeline --bindings bindings.local.yaml
```

```bash
# Override image for testing
dp run ./my-pipeline --set spec.runtime.image=local:dev
```

```bash
# Apply environment-specific overrides
dp run ./my-pipeline -f production.yaml
```

```bash
# Combine overrides (--set wins over -f)
dp run ./my-pipeline -f production.yaml --set spec.runtime.timeout=1h
```

```bash
# With environment variables
dp run ./my-pipeline --env API_KEY=secret --env DEBUG=true
```

---

## dp show

Show the effective manifest after applying overrides.

```bash
dp show [package-dir] [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--set` | - | Override values (key=value, repeatable) | - |
| `--values` | `-f` | Override files (repeatable) | - |
| `--output` | `-o` | Output format (yaml, json) | yaml |

### Description

The `dp show` command displays the merged manifest that would be used when running the pipeline.
This is useful for previewing the effect of override files and `--set` flags before executing.

### Examples

```bash
# Show manifest as-is
dp show ./my-pipeline
```

```bash
# Show with override file applied
dp show ./my-pipeline -f production.yaml
```

```bash
# Show with inline overrides
dp show ./my-pipeline --set spec.runtime.image=myimage:v2
```

```bash
# Show combined overrides (--set wins over -f)
dp show ./my-pipeline -f base.yaml --set spec.runtime.timeout=1h
```

```bash
# Output as JSON
dp show ./my-pipeline -o json
```

---

## dp test

Run pipeline tests with sample data.

```bash
dp test [package-dir] [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--input-type` | Input data type | auto |
| `--input-file` | Custom input file | - |
| `--timeout` | Test timeout | 5m |

### Examples

```bash
# Run tests
dp test ./my-pipeline
```

```bash
# With custom input
dp test ./my-pipeline --input-file ./testdata/sample.json
```

---

## dp build

Build OCI artifact for package.

```bash
dp build [package-dir] [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--tag` | Artifact tag | `<version from dp.yaml>` |
| `--no-cache` | Build without cache | false |

### Examples

```bash
# Build package
dp build ./my-pipeline
```

```bash
# With custom tag
dp build ./my-pipeline --tag v1.0.0-rc1
```

### Output

```
▶ Building package: my-pipeline
  → Validating manifest...
  → Bundling files...
  → Creating OCI artifact...
✓ Built: my-pipeline:v1.0.0

Artifact: ghcr.io/org/my-pipeline:v1.0.0
Size: 2.3 MB
```

---

## dp publish

Publish package to OCI registry.

```bash
dp publish [package-dir] [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--registry` | Registry URL | `$DP_REGISTRY` |
| `--tag` | Override tag | - |
| `--dry-run` | Print what would publish | false |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DP_REGISTRY` | Default registry URL |
| `DP_REGISTRY_USER` | Registry username |
| `DP_REGISTRY_TOKEN` | Registry access token |

### Examples

```bash
# Publish to default registry
dp publish ./my-pipeline
```

```bash
# Publish to specific registry
dp publish ./my-pipeline --registry ghcr.io/myorg
```

```bash
# Dry run
dp publish ./my-pipeline --dry-run
```

---

## dp promote

Promote package to an environment.

```bash
dp promote <package-name> <version> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target environment | **required** |
| `--dry-run` | Print what would change | false |
| `--auto-merge` | Automatically merge PR | false |
| `--rollback` | Mark as rollback (expedited) | false |

### Examples

```bash
# Promote to dev
dp promote my-pipeline v1.0.0 --to dev
```

```bash
# Promote to production with dry run
dp promote my-pipeline v1.0.0 --to prod --dry-run
```

```bash
# Emergency rollback
dp promote my-pipeline v0.9.0 --to prod --rollback
```

### Output

```
Promotion Request: my-pipeline v1.0.0 → dev
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Pre-flight Checks:
  ✓ Package exists in registry
  ✓ Version not already in dev
  ✓ Passed lint validation

Created PR: https://github.com/org/deploys/pull/123
```

---

## dp status

Show package status across environments.

```bash
dp status [package-name] [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--env` | Filter by environment | all |
| `--namespace` | Filter by namespace | all |

### Examples

```bash
# Show status of all packages
dp status
```

```bash
# Show specific package
dp status my-pipeline
```

```bash
# Filter by environment
dp status --env prod
```

### Output

```
Package: my-pipeline
━━━━━━━━━━━━━━━━━━━

Environment  Version   Status    Last Run
───────────  ───────   ──────    ────────
dev          v1.0.0    Synced    5 min ago
int          v0.9.0    Synced    1 day ago
prod         v0.9.0    Synced    6 hours ago
```

---

## dp logs

Stream logs from running pipeline.

```bash
dp logs <run-id> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--follow` | Follow log output | false |
| `--tail` | Lines to show | all |
| `--timestamps` | Show timestamps | false |

### Examples

```bash
# Get logs
dp logs my-pipeline-20250122-120000
```

```bash
# Follow logs
dp logs my-pipeline-20250122-120000 --follow
```

```bash
# Last 100 lines
dp logs my-pipeline-20250122-120000 --tail 100
```

---

## dp rollback

Rollback to a previous version.

```bash
dp rollback <package-name> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target version | previous |
| `--env` | Environment | **required** |
| `--dry-run` | Print what would change | false |

### Examples

```bash
# Rollback to previous version
dp rollback my-pipeline --env prod
```

```bash
# Rollback to specific version
dp rollback my-pipeline --to v1.0.0 --env prod
```

---

## dp lineage

View data lineage for a package.

```bash
dp lineage <package-name> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--upstream` | Show upstream sources | true |
| `--downstream` | Show downstream consumers | true |
| `--depth` | Maximum depth to traverse | 3 |
| `--refresh` | Force refresh from backend | false |

### Examples

```bash
# View lineage
dp lineage my-pipeline
```

```bash
# Only downstream impact
dp lineage my-pipeline --upstream=false
```

### Output

```
Lineage for: my-pipeline
━━━━━━━━━━━━━━━━━━━━━━━━━━━

Upstream:
  ├─ kafka://production/user-events
  └─ postgres://users-db/users

Downstream:
  ├─ s3://analytics-bucket/processed/
  └─ dashboard/user-metrics
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Validation error |
| 3 | Network/connectivity error |
| 4 | Authentication error |

---

## See Also

- [Configuration Reference](configuration.md) - Configuration file and environment variables
- [Manifest Schema](manifest-schema.md) - Package manifest reference
- [Quickstart](../getting-started/quickstart.md) - Get started with dp CLI
