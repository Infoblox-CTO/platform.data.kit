---
title: FAQ
description: Frequently asked questions about the Data Platform
---

# Frequently Asked Questions

Answers to common questions about the Data Platform.

## General Questions

### What is the Data Platform?

The Data Platform (DP) is a system for building, publishing, and operating data pipelines with built-in governance, lineage tracking, and GitOps-based deployment. It provides:

- **Data Packages**: Self-contained, versioned bundles for data pipelines
- **Local Development**: Docker-based development stack
- **Lineage Tracking**: Automatic OpenLineage integration
- **GitOps Deployment**: Environment promotion through pull requests

### What problem does DP solve?

DP addresses common challenges in data engineering:

| Challenge | DP Solution |
|-----------|-------------|
| Pipeline deployment complexity | GitOps-based promotion workflow |
| Lack of data lineage | Automatic OpenLineage events |
| Configuration drift | Immutable OCI artifacts |
| Governance gaps | Built-in classification and policies |
| Development friction | Local Docker stack |

### How is DP different from Airflow/Dagster/Prefect?

DP is complementary to orchestrators, not a replacement:

| Aspect | DP | Orchestrators |
|--------|----|----|
| Focus | Packaging & deployment | Workflow scheduling |
| Unit of work | Data package | Task/DAG |
| Runtime | OCI containers | Python/containers |
| Lineage | Native OpenLineage | Plugin-based |

DP packages can be scheduled by any orchestrator.

### What languages/runtimes are supported?

DP supports any containerized runtime:

- **Python** (most common)
- **Java/Scala** (Spark, Flink)
- **Go**
- **Node.js**
- Any language that runs in a container

---

## Development Questions

### How do I start developing locally?

```bash
# 1. Install dp CLI
make build
export PATH=$PATH:$(pwd)/bin

# 2. Start local stack
dp dev up

# 3. Create a package
dp init my-pipeline --kind model --runtime generic-python

# 4. Run locally
dp run ./my-pipeline
```

See the [Quickstart](../getting-started/quickstart.md) for details.

### What's included in the local development stack?

| Service | Purpose | Port |
|---------|---------|------|
| Kafka | Message streaming | 9092 |
| MinIO | S3-compatible storage | 9000, 9001 |
| Marquez | Lineage tracking | 5000 |
| PostgreSQL | Marquez database | 5432 |

### Can I use my own Kafka/S3 locally?

Yes! Override bindings in a local file:

```yaml title="bindings.local.yaml"
spec:
  bindings:
    input.events:
      type: kafka-topic
      ref: my-kafka/events
      config:
        bootstrap-servers: my-kafka:9092
```

```bash
dp run --bindings bindings.local.yaml
```

### How do I add custom services to the dev stack?

Create `.dp/docker-compose.override.yaml`:

```yaml
version: '3.8'
services:
  redis:
    image: redis:7
    ports:
      - "6379:6379"
```

### How do I persist data between runs?

Data is stored in Docker volumes. To reset:

```bash
# Keep data
dp dev down

# Remove data
dp dev down --volumes
```

---

## Package Questions

### What files are in a data package?

| File | Purpose | Required |
|------|---------|----------|
| `dp.yaml` | Package metadata, inputs/outputs, runtime config | Yes |
| `bindings.yaml` | Infrastructure references | No |

The `dp.yaml` file is a consolidated manifest that includes all configuration, including the `spec.runtime` section for pipeline execution settings (image, timeout, retries, env vars).

### What package kinds are available?

| Kind | Use Case |
|------|----------|
| `model` | Data transformation (most common) |
| `source` | Data source/extraction (CloudQuery) |
| `destination` | Data sink/loading (CloudQuery) |

Each kind supports multiple runtimes:

| Runtime | Description |
|---------|-------------|
| `cloudquery` | CloudQuery SDK sync |
| `generic-go` | Go container |
| `generic-python` | Python container |
| `dbt` | dbt transformations |

```bash
dp init my-pkg --kind model --runtime generic-python
dp init my-pkg --kind source --runtime cloudquery
```

### How do I version packages?

Packages use semantic versioning:

```bash
# Build with version
dp build --tag v1.0.0

# Increment for changes
v1.0.0 → v1.0.1  # Bug fix
v1.0.0 → v1.1.0  # New feature
v1.0.0 → v2.0.0  # Breaking change
```

### Can I publish private packages?

Yes! Push to a private registry:

```bash
# Use private registry
dp publish --registry ghcr.io/my-private-org

# Ensure authentication
docker login ghcr.io
```

---

## Deployment Questions

### How does promotion work?

DP uses GitOps for deployment:

1. `dp promote` creates a PR in the GitOps repository
2. PR is reviewed and approved
3. After merge, ArgoCD syncs to Kubernetes
4. Package runs in the target environment

### What are the standard environments?

| Environment | Approval | Purpose |
|-------------|----------|---------|
| dev | Auto-merge | Development |
| int | 1 approval | Integration testing |
| prod | 2 approvals | Production |

### Can I skip environments?

Not recommended, but possible:

```bash
# This will work but triggers a warning
dp promote my-pkg v1.0.0 --to prod
# Warning: Skipping dev and int environments
```

### How do I rollback a deployment?

```bash
# Rollback to previous version
dp rollback my-pkg --env prod

# Rollback to specific version
dp rollback my-pkg --to v1.0.0 --env prod
```

### How do I know what version is deployed?

```bash
dp status my-pkg
```

```
Environment  Version   Status
dev          v1.2.0    Synced
int          v1.1.0    Synced
prod         v1.1.0    Synced
```

---

## Lineage Questions

### What is data lineage?

Lineage tracks:

- Where data comes from (upstream)
- Where data goes to (downstream)
- What transformations were applied
- When the pipeline ran

### How does DP track lineage?

DP automatically emits [OpenLineage](https://openlineage.io/) events:

1. Reads inputs/outputs from `dp.yaml`
2. Emits START event when pipeline begins
3. Emits COMPLETE/FAIL event when pipeline ends
4. Events sent to Marquez (or configured backend)

### Where can I view lineage?

- **Local**: http://localhost:3000 (Marquez Web UI)
- **CLI**: `dp lineage my-pipeline` *(not yet implemented)*
- **Production**: Your organization's lineage backend

### Can I use a different lineage backend?

Yes! Configure in `~/.dp/config.yaml`:

```yaml
lineage:
  backend: datahub  # or: marquez, custom
  endpoint: http://datahub:8080/api/lineage
```

### Why isn't my lineage showing up?

Common causes:

1. Marquez not running: `dp dev status`
2. Wrong endpoint: Check `OPENLINEAGE_URL`
3. Pipeline never completed successfully

---

## Governance Questions

### What data classification levels are available?

| Level | Description |
|-------|-------------|
| `internal` | Internal use only |
| `confidential` | Limited access, may contain PII |
| `restricted` | Highly sensitive |

### How do I mark data as containing PII?

```yaml
outputs:
  - name: customer-data
    classification:
      pii: true
      sensitivity: confidential
```

### Are classification policies enforced?

Yes! `dp lint` enforces policies:

```bash
dp lint
# Error: output 'customer-data': pii=true requires sensitivity level
```

### How do I view what packages handle PII?

```bash
dp governance report --filter pii=true
```

---

## Troubleshooting Questions

### dp command not found

Add the binary to your PATH:

```bash
export PATH=$PATH:/path/to/data-platform/bin
```

### dp dev up fails

1. Check Docker is running: `docker info`
2. Check port conflicts: `lsof -i :9092`
3. Clean up: `dp dev down --volumes`

### Pipeline can't connect to Kafka

1. Wait for Kafka to be ready: `dp dev status`
2. Check Kafka logs: `docker compose -p dp logs kafka`
3. Verify bootstrap server: `localhost:9092`

### Push to registry fails

1. Check authentication: `docker login ghcr.io`
2. Check permissions on the registry
3. Check network connectivity

### Promotion PR not created

1. Check `GITHUB_TOKEN` is set
2. Verify GitOps repository URL in config
3. Check network connectivity

---

## More Questions?

If your question isn't answered here:

1. Check [Common Issues](common-issues.md)
2. Search [GitHub Issues](https://github.com/Infoblox-CTO/platform.data.kit/issues)
3. Ask in #data-platform-support on Slack
4. Open a new issue on GitHub

---

## See Also

- [Common Issues](common-issues.md) - Detailed troubleshooting
- [CLI Reference](../reference/cli.md) - Command documentation
- [Quickstart](../getting-started/quickstart.md) - Getting started guide
