---
title: Configuration Reference
description: Complete reference for dp CLI configuration
---

# Configuration Reference

This document covers all configuration options for the `dp` CLI including configuration files, environment variables, and per-project settings.

## Configuration File

The dp CLI uses hierarchical YAML configuration. Settings are loaded from three scopes (lowest to highest precedence):

1. **System**: `/etc/datakit/config.yaml`
2. **User**: `~/.config/dp/config.yaml`
3. **Repo**: `{git-root}/.dp/config.yaml`

Higher-precedence scopes override lower ones. Command-line flags override all scopes.

### Full Configuration

```yaml
# .dp/config.yaml — Example with all supported settings

# Local development settings
dev:
  runtime: k3d               # Runtime type: k3d or compose
  workspace: /path/to/work   # Path to DP workspace (optional)
  k3d:
    clusterName: dp-local    # k3d cluster name (DNS-safe)

# Plugin registry settings
plugins:
  registry: ghcr.io/infobloxopen   # Default OCI registry for plugins
  mirrors:                          # Fallback registries (tried in order)
    - ghcr.io/backup-org
    - internal.registry.io
  overrides:                        # Per-plugin version/image overrides
    postgresql:
      version: v8.13.0             # Pin a specific version
    s3:
      image: custom-s3:v1          # Full image override (bypasses registry)
  destinations:                     # Per-destination connection overrides
    postgresql:
      connection_string: "postgresql://user:pass@host:5432/mydb?sslmode=disable"
    s3:
      bucket: my-output-bucket
      region: us-west-2
      endpoint: "http://localstack:4566"
    file:
      path: /custom/output/path
```

### Configuration Sections

#### dev

Local development settings.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `dev.runtime` | string | Runtime type: `k3d` or `compose` | `k3d` |
| `dev.workspace` | string | Path to DP workspace | (none) |
| `dev.k3d.clusterName` | string | k3d cluster name (DNS-safe) | `dp-local` |

#### plugins

Plugin registry and override settings.

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `plugins.registry` | string | Default OCI registry for destination plugins | `ghcr.io/infobloxopen` |
| `plugins.mirrors` | string[] | Fallback registries tried in order when primary fails | `[]` |
| `plugins.overrides.<name>.version` | string | Pin a specific version for a plugin (semver) | built-in default |
| `plugins.overrides.<name>.image` | string | Full image reference (bypasses registry + naming) | (none) |

**Image resolution precedence:**

1. `plugins.overrides.<name>.image` → used as-is
2. `plugins.overrides.<name>.version` → `{registry}/cloudquery-plugin-{name}:{version}`
3. Default → `{registry}/cloudquery-plugin-{name}:{built-in-version}`

#### plugins.destinations

Per-destination connection and spec overrides. These settings control how `dp run` connects destination plugins to their backing services (databases, object stores, etc.).

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `plugins.destinations.<name>.connection_string` | string | Full connection string (postgresql) | auto-detected |
| `plugins.destinations.<name>.bucket` | string | S3 bucket name | `dp-output` |
| `plugins.destinations.<name>.region` | string | AWS region for S3 | `us-east-1` |
| `plugins.destinations.<name>.endpoint` | string | Custom S3 endpoint (e.g., LocalStack) | auto-detected |
| `plugins.destinations.<name>.path` | string | Output directory for file destination | `/home/nonroot/cq-sync-output` |

**Spec resolution order** (highest to lowest precedence):

1. **Config override** — explicit values in `plugins.destinations.<name>.*`
2. **In-cluster auto-detect** — discovered from running k3d services via `kubectl`
3. **Built-in default** — hardcoded fallback values

**Auto-detection details:**

During `dp run`, the CLI queries the k3d cluster for known services:

- **PostgreSQL**: Looks for `dp-postgres-postgres` service in the current namespace. If found, builds the connection string using the in-cluster DNS name (`dp-postgres-postgres.<namespace>.svc.cluster.local:5432`) with default credentials (`postgres:postgres`, database `postgres`).
- **S3 (LocalStack)**: Looks for `dp-localstack-localstack` service in the current namespace. If found, uses its in-cluster DNS endpoint (`http://dp-localstack-localstack.<namespace>.svc.cluster.local:4566`), sets `force_path_style: true`, and uses the `dp-output` bucket.
- **File**: No auto-detection needed. Defaults to `/home/nonroot/cq-sync-output` inside the container, which is bind-mounted to `./cq-sync-output/` on the host.

**Examples:**

Override the PostgreSQL connection string for a custom database:

```bash
dp config set plugins.destinations.postgresql.connection_string \
  "postgresql://myuser:mypass@custom-host:5432/analytics?sslmode=disable"
```

Point S3 output at a custom bucket and endpoint:

```bash
dp config set plugins.destinations.s3.bucket my-data-lake
dp config set plugins.destinations.s3.endpoint "http://minio:9000"
```

Change the file output path:

```bash
dp config set plugins.destinations.file.path /data/output
```

View the effective destination configuration:

```bash
dp config list | grep destinations
```

#### registry

Registry settings for artifact publishing.

| Field | Type | Description |
|-------|------|-------------|
| `default` | string | Default registry URL for `dp publish` |
| `credentials` | array | List of registry credentials |
| `credentials[].registry` | string | Registry hostname |
| `credentials[].username` | string | Username (supports env vars) |
| `credentials[].token` | string | Access token (supports env vars) |

#### environments

Environment configuration for promotions.

| Field | Type | Description |
|-------|------|-------------|
| `<env>.gitops` | string | GitOps repository URL |
| `<env>.path` | string | Path within repository |
| `<env>.auto_merge` | boolean | Auto-merge PRs (default: false) |
| `<env>.approvers` | array | Required approvers |
| `<env>.approval_count` | integer | Number of approvals required |

#### lineage

OpenLineage backend configuration.

| Field | Type | Description |
|-------|------|-------------|
| `backend` | string | Backend type: marquez, datahub, custom |
| `endpoint` | string | API endpoint URL |
| `api_key` | string | Optional API key |

#### defaults

Default values for CLI flags.

| Field | Type | Description |
|-------|------|-------------|
| `output` | string | Default output format |
| `timeout` | duration | Default command timeout |
| `namespace` | string | Default namespace |
| `log_level` | string | Logging level |

---

## Environment Variables

Environment variables override configuration file values.

### Core Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DP_CONFIG` | Config file path | `/custom/config.yaml` |
| `DP_NAMESPACE` | Default namespace | `analytics` |
| `DP_OUTPUT_FORMAT` | Output format | `json` |
| `DP_LOG_LEVEL` | Log level | `debug` |
| `DP_DEBUG` | Enable debug mode | `true` |

### Registry Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DP_REGISTRY` | Default registry | `ghcr.io/myorg` |
| `DP_REGISTRY_USER` | Registry username | `ci-bot` |
| `DP_REGISTRY_TOKEN` | Registry token | `ghp_xxx...` |

### Lineage Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `OPENLINEAGE_URL` | OpenLineage endpoint | `http://marquez:5000/api/v1/lineage` |
| `OPENLINEAGE_API_KEY` | API key for lineage | `api-key-xxx` |

### Development Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DP_DEV_NETWORK` | Docker network name | `dp-network` |
| `DP_DEV_TIMEOUT` | Dev stack timeout | `120s` |

---

## Project Configuration

Project-specific settings in `.dp/config.yaml`:

```yaml
# .dp/config.yaml (in project root)

# Project-specific registry
registry:
  default: ghcr.io/myteam

# Project namespace
defaults:
  namespace: my-project
```

### Configuration Precedence

1. Command-line flags (highest priority) — e.g., `--registry`
2. Environment variables
3. Repo configuration (`{git-root}/.dp/config.yaml`)
4. User configuration (`~/.config/dp/config.yaml`)
5. System configuration (`/etc/datakit/config.yaml`)
6. Built-in defaults (lowest priority)

Use `dp config list` to see the effective value and source for each setting.

---

## Local Development Stack

### docker-compose.yaml

The default development stack includes:

```yaml
# docker-compose.yaml (managed by dp dev)
version: '3.8'

services:
  kafka:
    image: confluentinc/cp-kafka:7.5.0
    ports:
      - "9092:9092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      
  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
      
  postgres:
    image: postgres:15
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: marquez
      POSTGRES_PASSWORD: marquez
      POSTGRES_DB: marquez
      
  marquez:
    image: marquezproject/marquez:0.41.0
    ports:
      - "5000:5000"
    depends_on:
      - postgres
    environment:
      MARQUEZ_PORT: 5000
      MARQUEZ_ADMIN_PORT: 5001
```

### Customizing the Stack

Override services in `.dp/docker-compose.override.yaml`:

```yaml
# .dp/docker-compose.override.yaml
version: '3.8'

services:
  # Add Redis for caching
  redis:
    image: redis:7
    ports:
      - "6379:6379"
      
  # Override Kafka memory
  kafka:
    environment:
      KAFKA_HEAP_OPTS: "-Xmx1G -Xms1G"
```

---

## Governance Configuration

### policies.yaml

Define organization-wide policies:

```yaml
# .dp/policies.yaml

policies:
  # Require classification on all outputs
  require_classification: true
  
  # Owner email pattern
  owner_pattern: "^[a-z-]+@example\\.com$"
  
  # Maximum retention for PII data
  max_pii_retention_days: 730
  
  # Required tags for confidential data
  confidential_required_tags:
    - gdpr
    
  # Require description on packages
  require_description: true
  
  # Minimum description length
  min_description_length: 20
```

### Policy Enforcement

Policies are checked by `dp lint`:

```bash
dp lint --policy .dp/policies.yaml
```

---

## Shell Completion

Enable tab completion for better CLI experience.

### Bash

```bash
# Add to ~/.bashrc
source <(dp completion bash)
```

### Zsh

```bash
# Add to ~/.zshrc
source <(dp completion zsh)
```

### Fish

```bash
dp completion fish | source
```

---

## Logging

### Log Levels

| Level | Description |
|-------|-------------|
| `debug` | Verbose output for troubleshooting |
| `info` | Normal operation (default) |
| `warn` | Warnings only |
| `error` | Errors only |

### Setting Log Level

```bash
# Environment variable
export DP_LOG_LEVEL=debug

# Config file
defaults:
  log_level: debug

# Command line
dp run --log-level debug
```

### Log Output

```bash
# JSON format for parsing
export DP_LOG_FORMAT=json

# Include timestamps
export DP_LOG_TIMESTAMPS=true
```

---

## See Also

- [CLI Reference](cli.md) - Command reference
- [Installation](../getting-started/installation.md) - Initial setup
- [Environments](../concepts/environments.md) - Environment configuration
