---
title: Configuration Reference
description: Complete reference for dp CLI configuration
---

# Configuration Reference

This document covers all configuration options for the `dp` CLI including configuration files, environment variables, and per-project settings.

## Configuration File

The dp CLI looks for configuration in these locations (in order):

1. `$XDG_CONFIG_HOME/dp/config.yaml`
2. `~/.config/dp/config.yaml`
3. `~/.dp/config.yaml`
4. `.dp/config.yaml` (project-specific)

### Full Configuration

```yaml
# ~/.dp/config.yaml

# Registry configuration
registry:
  default: ghcr.io/myorg          # Default registry for publish
  credentials:                     # Registry credentials
    - registry: ghcr.io
      username: ${DP_REGISTRY_USER}
      token: ${DP_REGISTRY_TOKEN}
    - registry: docker.io
      username: ${DOCKER_USER}
      token: ${DOCKER_TOKEN}

# Environment configuration
environments:
  dev:
    gitops: https://github.com/myorg/gitops.git
    path: environments/dev
    auto_merge: true
  int:
    gitops: https://github.com/myorg/gitops.git
    path: environments/int
    approvers:
      - "@team-leads"
  prod:
    gitops: https://github.com/myorg/gitops.git
    path: environments/prod
    approvers:
      - "@team-leads"
      - "@security"
    approval_count: 2

# Lineage configuration
lineage:
  backend: marquez                 # marquez, datahub, custom
  endpoint: http://marquez:5000/api/v1/lineage
  api_key: ${LINEAGE_API_KEY}      # Optional

# Default settings
defaults:
  output: table                    # table, json, yaml
  timeout: 30m
  namespace: default
  log_level: info                  # debug, info, warn, error

# Local development stack
dev_stack:
  compose_file: docker-compose.yaml
  network: dp-network
  services:
    kafka:
      port: 9092
    minio:
      port: 9000
    marquez:
      port: 5000
    postgres:
      port: 5432
```

### Configuration Sections

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

# Custom bindings path
bindings:
  default_path: config/bindings
```

### Configuration Precedence

1. Command-line flags (highest priority)
2. Environment variables
3. Project configuration (`.dp/config.yaml`)
4. User configuration (`~/.dp/config.yaml`)
5. Default values (lowest priority)

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
