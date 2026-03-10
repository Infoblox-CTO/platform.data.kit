---
title: FAQ
description: Frequently asked questions about the Data Platform
---

# Frequently Asked Questions

Answers to common questions about the Data Platform.

## General Questions

### What is the Data Platform?

DataKit is a system for building, publishing, and operating data pipelines with built-in governance, lineage tracking, and GitOps-based deployment. It provides:

- **Data Packages**: Self-contained, versioned bundles for data pipelines
- **Local Development**: Docker-based development stack
- **Lineage Tracking**: Automatic OpenLineage integration
- **GitOps Deployment**: Environment promotion through pull requests

### What problem does DataKit solve?

DataKit addresses common challenges in data engineering:

| Challenge | DataKit Solution |
|-----------|-------------|
| Pipeline deployment complexity | GitOps-based promotion workflow |
| Lack of data lineage | Automatic OpenLineage events |
| Configuration drift | Immutable OCI artifacts |
| Governance gaps | Built-in classification and policies |
| Development friction | Local Docker stack |

### How is DataKit different from Airflow/Dagster/Prefect?

DataKit is complementary to orchestrators, not a replacement:

| Aspect | DataKit | Orchestrators |
|--------|----|----|
| Focus | Packaging & deployment | Workflow scheduling |
| Unit of work | Data package | Task/DAG |
| Runtime | OCI containers | Python/containers |
| Lineage | Native OpenLineage | Plugin-based |

DataKit packages can be scheduled by any orchestrator.

### What languages/runtimes are supported?

DataKit supports any containerized runtime:

- **Python** (most common)
- **Java/Scala** (Spark, Flink)
- **Go**
- **Node.js**
- Any language that runs in a container

---

## Development Questions

### How do I start developing locally?

```bash
# 1. Install dk CLI
make build
export PATH=$PATH:$(pwd)/bin

# 2. Start local stack
dk dev up

# 3. Create a Transform package
dk init my-pipeline --runtime generic-python

# 4. Run locally
dk run ./my-pipeline
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

Yes! Override connection details in your Store manifests:

```yaml title="store/my-kafka.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: my-kafka
spec:
  connector: kafka
  connection:
    bootstrap-servers: my-kafka:9092
```

Assets reference the Store by name — no additional configuration is needed.

### How do I add custom services to the dev stack?

Use `dk config` to customize chart versions and Helm values:

```bash
dk config set dev.charts.redpanda.version 25.2.0
dk config set dev.charts.postgres.values.primary.resources.limits.memory 1Gi
```

### How do I persist data between runs?

Data is stored in Docker volumes. To reset:

```bash
# Keep data
dk dev down

# Remove data
dk dev down --volumes
```

---

## Package Questions

### What files are in a data package?

| File/Directory | Purpose | Required |
|------|---------|----------|
| `dk.yaml` | Transform manifest (runtime, inputs, outputs, schedule) | Yes |
| `connector/` | Connector definitions (technology types) | No |
| `store/` | Store definitions (instances with connection details) | No |
| `dataset/` | DataSet definitions (data contracts with schema) | No |
| `dataset-group/` | DataSetGroup definitions (bundled DataSets) | No |

The `dk.yaml` file is a Transform manifest that references DataSets by name. DataSets reference Stores, and Stores reference Connectors.

### What runtimes are available?

| Runtime | Description |
|---------|-------------|
| `cloudquery` | CloudQuery SDK sync |
| `generic-go` | Go container |
| `generic-python` | Python container |
| `dbt` | dbt transformations |

```bash
dk init my-pkg --runtime generic-python
dk init my-pkg --runtime cloudquery
```

### How do I version packages?

Packages use semantic versioning:

```bash
# Build with version
dk build --tag v1.0.0

# Increment for changes
v1.0.0 → v1.0.1  # Bug fix
v1.0.0 → v1.1.0  # New feature
v1.0.0 → v2.0.0  # Breaking change
```

### Can I publish private packages?

Yes! Push to a private registry:

```bash
# Use private registry
dk publish --registry ghcr.io/my-private-org

# Ensure authentication
docker login ghcr.io
```

---

## Deployment Questions

### How does promotion work?

DataKit uses GitOps for deployment:

1. `dk promote` creates a PR in the GitOps repository
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
dk promote my-pkg v1.0.0 --to prod
# Warning: Skipping dev and int environments
```

### How do I rollback a deployment?

```bash
# Rollback to previous version
dk rollback my-pkg --env prod

# Rollback to specific version
dk rollback my-pkg --to v1.0.0 --env prod
```

### How do I know what version is deployed?

```bash
dk status my-pkg
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

### How does DataKit track lineage?

DataKit automatically emits [OpenLineage](https://openlineage.io/) events:

1. Reads inputs/outputs from `dk.yaml`
2. Emits START event when pipeline begins
3. Emits COMPLETE/FAIL event when pipeline ends
4. Events sent to Marquez (or configured backend)

### Where can I view lineage?

- **Local**: http://localhost:3000 (Marquez Web UI)
- **CLI**: `dk lineage my-pipeline` *(not yet implemented)*
- **Production**: Your organization's lineage backend

### Can I use a different lineage backend?

Yes! Configure in `~/.dk/config.yaml`:

```yaml
lineage:
  backend: datahub  # or: marquez, custom
  endpoint: http://datahub:8080/api/lineage
```

### Why isn't my lineage showing up?

Common causes:

1. Marquez not running: `dk dev status`
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

Use the `classification` and `pii` fields on DataSet schemas:

```yaml title="dataset/customer-data.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: customer-data
spec:
  store: warehouse
  table: public.customers
  classification: confidential
  schema:
    - name: email
      type: string
      pii: true
    - name: name
      type: string
      pii: true
```

### Are classification policies enforced?

Yes! `dk lint` enforces policies:

```bash
dk lint
# Error: output 'customer-data': pii=true requires sensitivity level
```

### How do I view what packages handle PII?

```bash
dk governance report --filter pii=true
```

---

## Troubleshooting Questions

### dk command not found

Add the binary to your PATH:

```bash
export PATH=$PATH:/path/to/datakit/bin
```

### dk dev up fails

1. Check Docker is running: `docker info`
2. Check port conflicts: `lsof -i :9092`
3. Clean up: `dk dev down --volumes`

### Pipeline can't connect to Kafka

1. Wait for Kafka to be ready: `dk dev status`
2. Check Kafka logs: `kubectl --context k3d-dk-local logs -l app=redpanda`
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
3. Ask in #datakit-support on Slack
4. Open a new issue on GitHub

---

## See Also

- [Common Issues](common-issues.md) - Detailed troubleshooting
- [CLI Reference](../reference/cli.md) - Command documentation
- [Quickstart](../getting-started/quickstart.md) - Getting started guide
