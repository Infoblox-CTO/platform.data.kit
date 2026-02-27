---
title: Pipeline Workflows
description: Multi-step pipeline execution and orchestration
---

# Pipeline Workflows

Pipeline workflows define the ordered sequence of steps that compose a data pipeline. Each step has a specific type (sync, transform, test, publish, or custom) and executes as an isolated container.

## Overview

A pipeline workflow is defined in `pipeline.yaml` and describes:

- **Steps**: The ordered sequence of operations to execute
- **Step types**: What kind of work each step performs
- **Asset references**: Which project assets each step operates on
- **Environment variables**: Configuration injected at runtime

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: security-pipeline
  description: Ingest, transform, and publish security data
steps:
  - name: sync-data
    type: sync
    source: aws-security
    sink: postgres-warehouse

  - name: transform-data
    type: transform
    asset: dbt-security-model

  - name: run-tests
    type: test
    asset: dbt-security-model
    command: ["dbt", "test"]

  - name: notify-team
    type: publish
    promote: true
    notify:
      channels: ["#data-platform"]
```

## Step Types

### Sync

Moves data from a source asset to a sink asset. Used for data ingestion.

| Field    | Required | Description                |
|----------|----------|----------------------------|
| `source` | Yes      | Name of the source asset   |
| `sink`   | Yes      | Name of the sink/dest asset|

### Transform

Executes a transformation engine (e.g., dbt) against an asset.

| Field   | Required | Description                    |
|---------|----------|--------------------------------|
| `asset` | Yes      | Name of the transform asset    |

### Test

Runs validation or assertion commands against an asset.

| Field     | Required | Description                     |
|-----------|----------|---------------------------------|
| `asset`   | Yes      | Name of the asset to test       |
| `command` | Yes      | Command and args to execute     |

### Publish

Sends notifications and optionally triggers environment promotion.

| Field     | Required | Description                        |
|-----------|----------|------------------------------------|
| `promote` | No       | Whether to trigger promotion       |
| `notify`  | No       | Notification config (channels, recipients) |

### Custom

Runs an arbitrary container image. Provides backward compatibility with existing single-container pipelines.

| Field   | Required | Description                    |
|---------|----------|--------------------------------|
| `image` | Yes      | Container image to run         |
| `args`  | No       | Arguments passed to container  |

## Execution Model

Steps execute **sequentially** in the order defined. If any step fails:

1. The failed step is marked with status `failed`
2. All remaining steps are marked `skipped`
3. The pipeline result reports the failed step

```
step-1 (sync)      → ✓ completed [2.3s]
step-2 (transform) → ✗ failed [0.8s] — exit code 1
step-3 (test)      → ⊘ skipped
step-4 (publish)   → ⊘ skipped
```

Each step's output is prefixed with `[step-name]` for easy identification in logs.

## Scheduling

An optional `schedule.yaml` alongside `pipeline.yaml` defines cron-based execution timing:

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Schedule
cron: "0 6 * * *"
timezone: America/New_York
```

| Field      | Required | Default | Description                        |
|------------|----------|---------|------------------------------------|
| `cron`     | Yes      | —       | Standard 5-field cron expression   |
| `timezone` | No       | UTC     | IANA timezone for cron evaluation  |
| `suspend`  | No       | false   | Pause scheduled execution          |

## Backfill

The backfill feature re-executes **sync steps only** with a date range injected as environment variables:

- `DP_BACKFILL_FROM`: Start date (YYYY-MM-DD)
- `DP_BACKFILL_TO`: End date (YYYY-MM-DD)

```bash
dp pipeline backfill --from 2026-01-01 --to 2026-01-31
```

Non-sync steps (transform, test, publish, custom) are excluded from backfill runs.

## CLI Commands

| Command                  | Description                                  |
|--------------------------|----------------------------------------------|
| `dp pipeline create`    | Scaffold a new pipeline.yaml from a template |
| `dp pipeline run`       | Execute the pipeline workflow                |
| `dp pipeline backfill`  | Re-execute sync steps for a date range       |
| `dp pipeline show`      | Display pipeline definition, schedule, or dependency graph |

### Creating a Pipeline

```bash
# Create with the default template (sync → transform → test → publish)
dp pipeline create my-pipeline

# Use a specific template
dp pipeline create my-pipeline --template sync-only

# List available templates
dp pipeline create --list-templates
```

### Running a Pipeline

```bash
# Run all steps
dp pipeline run

# Run a single step
dp pipeline run --step sync-data

# Pass environment variables
dp pipeline run --env DEBUG=true --env LOG_LEVEL=info
```

### Inspecting a Pipeline

```bash
# Show full reactive dependency graph
dp pipeline show --all

# Show graph leading to a specific destination
dp pipeline show --destination event-summary

# Render as Mermaid diagram
dp pipeline show --all --output mermaid

# Render as Graphviz DOT
dp pipeline show --all --output dot

# JSON adjacency list
dp pipeline show --all --output json

# Scan specific directories
dp pipeline show --all --scan-dir ./transforms --scan-dir ./assets

# Legacy: table view
dp pipeline show

# Legacy: JSON output
dp pipeline show --output json
```

## Backward Compatibility

The existing `dp run` command continues to work unchanged for packages that use `dp.yaml` without a `pipeline.yaml`. The pipeline workflow feature is additive — it does not modify the existing single-container execution path.

## Validation

Pipeline workflows are validated by `dp validate` (via the aggregate validator):

- Required fields: `apiVersion`, `kind`, `metadata.name`, `steps`
- Step names must be unique and DNS-safe (3–63 lowercase chars)
- Step type must be one of: sync, transform, test, publish, custom
- Type-specific required fields are enforced (e.g., sync requires source + sink)
- Asset references are cross-validated against project assets
- Schedule cron expressions and timezones are validated
