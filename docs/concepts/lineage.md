---
title: Data Lineage
description: Understanding how DataKit tracks data lineage using OpenLineage
---

# Data Lineage

Data lineage tracks the origin, movement, and transformation of data through your pipelines. DataKit uses [OpenLineage](https://openlineage.io/) to capture and visualize lineage automatically.

## What is Lineage?

Lineage answers critical questions:

| Question | Lineage Answer |
|----------|----------------|
| Where did this data come from? | **Upstream** sources and transformations |
| What depends on this data? | **Downstream** consumers |
| What happened to this run? | **Job execution** events and status |
| Is this data fresh? | **Last successful run** timestamp |

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                     Lineage Flow                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐         ┌──────────────┐        ┌──────────┐ │
│  │   Pipeline   │────────▶│   Lineage    │───────▶│ Marquez  │ │
│  │   Runtime    │ events  │   Collector  │ store  │    UI    │ │
│  └──────────────┘         └──────────────┘        └──────────┘ │
│         │                        ▲                      │       │
│         │                        │                      │       │
│         ▼                        │                      ▼       │
│  ┌──────────────┐         ┌──────────────┐        ┌──────────┐ │
│  │ dk.yaml      │         │  OpenLineage │        │ Lineage  │ │
│  │ manifest     │────────▶│    Events    │        │  Graph   │ │
│  └──────────────┘ defines └──────────────┘        └──────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Automatic Lineage

When you run `dk run`, the CLI automatically:

1. **Reads** the manifest to understand inputs/outputs
2. **Emits START** event when pipeline begins
3. **Emits COMPLETE/FAIL** event when pipeline ends
4. **Sends** events to Marquez (or configured backend)

No code changes required in your pipeline!

### Manual Lineage (Optional)

For custom lineage, use the OpenLineage SDK in your pipeline:

```python
from openlineage.client import OpenLineageClient
from openlineage.client.run import Run, Job

client = OpenLineageClient.from_environment()

# Emit custom lineage event
client.emit(
    RunEvent(
        eventType=RunState.RUNNING,
        job=Job(namespace="analytics", name="my-pipeline"),
        run=Run(runId=str(uuid.uuid4())),
        inputs=[...],
        outputs=[...]
    )
)
```

## Viewing Lineage

### Local Development

With `dk dev up`, Marquez is available at http://localhost:5000:

```bash
dk dev up
dk run ./my-package
# Open http://localhost:5000 to see lineage
```

### CLI Lineage Command

!!! warning "Not Yet Implemented"
    The `dk lineage` CLI command is planned but not yet available.
    Use the Marquez Web UI at http://localhost:3000 to view lineage graphs.

Planned usage:

```bash
dk lineage my-package
```

Output:

```
Lineage for: my-package
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Upstream:
  ├─ kafka://production/user-events
  └─ postgres://users-db/users

Downstream:
  ├─ s3://analytics-bucket/processed/
  └─ dashboard/user-metrics

Last Run:
  Status: COMPLETE
  Started: 2025-01-22T10:00:00Z
  Finished: 2025-01-22T10:05:32Z
  Duration: 5m 32s
```

### Marquez UI

The Marquez UI provides a visual lineage graph:

```
┌─────────────┐     ┌─────────────────┐     ┌─────────────┐
│ user-events │────▶│  my-pipeline    │────▶│ processed/  │
│   (kafka)   │     │                 │     │   (s3)      │
└─────────────┘     └─────────────────┘     └─────────────┘
                            │
                            ▼
                    ┌─────────────┐
                    │ user-metrics│
                    │ (dashboard) │
                    └─────────────┘
```

## OpenLineage Events

The DK CLI emits standard OpenLineage events:

### Run Events

| Event Type | When Emitted |
|------------|--------------|
| `START` | Pipeline begins execution |
| `RUNNING` | Periodic heartbeat (optional) |
| `COMPLETE` | Pipeline finished successfully |
| `FAIL` | Pipeline failed with error |
| `ABORT` | Pipeline was cancelled |

### Event Structure

```json
{
  "eventType": "COMPLETE",
  "eventTime": "2025-01-22T10:05:32.000Z",
  "job": {
    "namespace": "analytics",
    "name": "my-package"
  },
  "run": {
    "runId": "run-abc123"
  },
  "inputs": [
    {
      "namespace": "kafka",
      "name": "production/user-events"
    }
  ],
  "outputs": [
    {
      "namespace": "s3",
      "name": "analytics-bucket/processed/"
    }
  ]
}
```

## Lineage Configuration

### Backend Configuration

Configure the OpenLineage backend in your environment:

```bash
# Environment variables
export OPENLINEAGE_URL=http://localhost:5000/api/v1/lineage
export OPENLINEAGE_API_KEY=your-api-key  # if required
```

Or in `~/.dk/config.yaml`:

```yaml
lineage:
  backend: marquez
  endpoint: http://localhost:5000/api/v1/lineage
  api_key: your-api-key
```

### Supported Backends

| Backend | Description |
|---------|-------------|
| **Marquez** | Default, open-source lineage server |
| **DataHub** | LinkedIn's data catalog |
| **OpenMetadata** | Open-source metadata platform |
| **Custom** | Any OpenLineage-compatible endpoint |

## Lineage Best Practices

### 1. Meaningful Names

Use descriptive, consistent names:

```yaml
# Good
metadata:
  name: user-events-to-s3-parquet
  namespace: analytics

# Bad
metadata:
  name: pipeline1
  namespace: default
```

### 2. Document Inputs/Outputs

Include descriptions in your manifest:

```yaml
inputs:
  - name: user-events
    type: kafka-topic
    description: "Real-time user behavior events from web app"
```

### 3. Use Namespaces

Group related packages:

```yaml
metadata:
  namespace: analytics     # All analytics pipelines
  # or
  namespace: ml-training   # All ML training jobs
```

### 4. Tag Sensitive Data

Use classification for governance:

```yaml
outputs:
  - name: customer-data
    classification:
      pii: true
      sensitivity: confidential
```

## Troubleshooting Lineage

### Events Not Appearing

1. Check Marquez is running: `dk dev status`
2. Verify endpoint: `echo $OPENLINEAGE_URL`
3. Check connectivity: `curl $OPENLINEAGE_URL/api/v1/namespaces`

### Missing Connections

If upstream/downstream links are missing:

1. Ensure consistent naming across packages
2. Check that binding references match
3. Verify packages are in the same namespace

### Stale Lineage

If lineage shows old data:

```bash
# Planned: dk lineage my-package --refresh
# For now, check directly in the Marquez UI at http://localhost:3000
```

## Next Steps

- [Governance](governance.md) - How lineage enables governance
- [Environments](environments.md) - Lineage across environments
- [Troubleshooting](../troubleshooting/common-issues.md) - Common lineage issues
