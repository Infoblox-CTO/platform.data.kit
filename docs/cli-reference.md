# DP CLI Reference

Complete reference for all `dp` CLI commands.

## Global Flags

These flags apply to all commands:

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--output` | `-o` | Output format (table, json, yaml) | table |
| `--help` | `-h` | Show help | - |
| `--version` | `-v` | Show version | - |

## Commands

### dp init

Create a new data package.

```bash
dp init <package-name> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--type` | Package type (pipeline, model, dataset) | pipeline |
| `--dir` | Directory to create package in | . |

**Examples:**

```bash
# Create a pipeline package
dp init my-pipeline --type pipeline

# Create in specific directory
dp init kafka-processor --dir ./packages
```

---

### dp dev

Manage the local development stack.

#### dp dev up

Start the local development stack.

```bash
dp dev up [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--detach` | Run in background | false |
| `--timeout` | Startup timeout | 60s |

**Examples:**

```bash
# Start local stack
dp dev up

# Start in background
dp dev up --detach
```

#### dp dev down

Stop the local development stack.

```bash
dp dev down [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--volumes` | Remove volumes | false |

**Examples:**

```bash
# Stop stack
dp dev down

# Stop and remove volumes
dp dev down --volumes
```

#### dp dev status

Show status of local development stack.

```bash
dp dev status
```

**Output:**

```
Service         Status    Ports
redpanda        running   9092, 9644
localstack      running   4566
postgres        running   5432
marquez         running   5000
```

---

### dp lint

Validate package manifests.

```bash
dp lint [package-dir] [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--strict` | Treat warnings as errors | false |
| `--skip-pii` | Skip PII classification validation | false |

**Validated files:**
- `dp.yaml` - Package manifest
- `pipeline.yaml` - Pipeline configuration
- `bindings.yaml` - Binding configuration
- `schemas/` - Schema files

**Validation rules:**
- E001-E003: Required fields
- E004-E005: Schema references
- E010-E011: Binding configuration
- E025: PII classification required
- E030-E031: Runtime configuration

**Examples:**

```bash
# Lint current directory
dp lint

# Lint specific package
dp lint ./my-pipeline

# Strict mode
dp lint --strict

# Skip PII validation
dp lint --skip-pii
```

---

### dp run

Execute pipeline locally.

```bash
dp run [package-dir] [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--network` | Docker network | dp-network |
| `--env` | Environment variables (KEY=VALUE) | - |
| `--bindings` | Bindings file path | bindings.yaml |
| `--dry-run` | Print what would run | false |
| `--detach` | Run in background | false |
| `--timeout` | Execution timeout | 30m |

**Examples:**

```bash
# Run pipeline
dp run ./my-pipeline

# With custom bindings
dp run ./my-pipeline --bindings bindings.local.yaml

# Dry run
dp run ./my-pipeline --dry-run

# With environment variables
dp run ./my-pipeline --env API_KEY=secret --env DEBUG=true
```

---

### dp test

Run pipeline tests with sample data.

```bash
dp test [package-dir] [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--input-type` | Input data type | auto |
| `--input-file` | Custom input file | - |
| `--timeout` | Test timeout | 5m |

**Examples:**

```bash
# Run tests
dp test ./my-pipeline

# With custom input
dp test ./my-pipeline --input-file ./testdata/sample.json
```

---

### dp build

Build OCI artifact for package.

```bash
dp build [package-dir] [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--tag` | Artifact tag | <version from dp.yaml> |
| `--no-cache` | Build without cache | false |

**Examples:**

```bash
# Build package
dp build ./my-pipeline

# With custom tag
dp build ./my-pipeline --tag v1.0.0-rc1
```

---

### dp publish

Publish package to OCI registry.

```bash
dp publish [package-dir] [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--registry` | Registry URL | $DP_REGISTRY |
| `--tag` | Override tag | - |
| `--dry-run` | Print what would publish | false |

**Environment Variables:**

| Variable | Description |
|----------|-------------|
| `DP_REGISTRY` | Default registry URL |
| `DP_REGISTRY_USER` | Registry username |
| `DP_REGISTRY_TOKEN` | Registry access token |

**Examples:**

```bash
# Publish to default registry
dp publish ./my-pipeline

# Publish to specific registry
dp publish ./my-pipeline --registry ghcr.io/myorg

# Dry run
dp publish ./my-pipeline --dry-run
```

---

### dp promote

Promote package to an environment.

```bash
dp promote <package-name> <version> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target environment | required |
| `--dry-run` | Print what would change | false |
| `--auto-merge` | Automatically merge PR | false |

**Examples:**

```bash
# Promote to dev
dp promote my-pipeline v1.0.0 --to dev

# Promote to production with dry run
dp promote my-pipeline v1.0.0 --to prod --dry-run
```

---

### dp status

Show package status across environments.

```bash
dp status [package-name] [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--env` | Filter by environment | all |
| `--namespace` | Filter by namespace | all |

**Examples:**

```bash
# Show status of all packages
dp status

# Show specific package
dp status my-pipeline

# Filter by environment
dp status --env prod
```

---

### dp logs

Stream logs from running pipeline.

```bash
dp logs <run-id> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--follow` | Follow log output | false |
| `--tail` | Lines to show | all |
| `--timestamps` | Show timestamps | false |

**Examples:**

```bash
# Get logs
dp logs my-pipeline-20240122-120000

# Follow logs
dp logs my-pipeline-20240122-120000 --follow

# Last 100 lines
dp logs my-pipeline-20240122-120000 --tail 100
```

---

### dp rollback

Rollback to a previous version.

```bash
dp rollback <package-name> [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--to` | Target version | previous |
| `--env` | Environment | required |
| `--dry-run` | Print what would change | false |

**Examples:**

```bash
# Rollback to previous version
dp rollback my-pipeline --env prod

# Rollback to specific version
dp rollback my-pipeline --to v1.0.0 --env prod
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

## Configuration

### Config File

DP looks for configuration in:
1. `$XDG_CONFIG_HOME/dp/config.yaml`
2. `~/.config/dp/config.yaml`
3. `~/.dp/config.yaml`

```yaml
# config.yaml
registry:
  default: ghcr.io/myorg
  credentials:
    - registry: ghcr.io
      username: ${DP_REGISTRY_USER}
      token: ${DP_REGISTRY_TOKEN}

environments:
  dev:
    gitops: https://github.com/myorg/gitops.git
    path: environments/dev
  prod:
    gitops: https://github.com/myorg/gitops.git
    path: environments/prod

defaults:
  output: table
  timeout: 30m
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DP_REGISTRY` | Default OCI registry |
| `DP_REGISTRY_USER` | Registry username |
| `DP_REGISTRY_TOKEN` | Registry token |
| `DP_NAMESPACE` | Default namespace |
| `DP_CONFIG` | Config file path |
| `DP_DEBUG` | Enable debug logging |
