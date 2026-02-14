# Data Model: Pipeline Workflows

**Feature**: 012-pipeline-workflows  
**Date**: 2026-02-14  
**Source**: [spec.md](spec.md), [research.md](research.md)

## Entity Relationship Overview

```text
DataPackage (dp.yaml)
 ├── assets: []string ─────────────────► Asset (asset.yaml)
 │                                        ├── type: source | sink | model-engine
 │                                        ├── extension: FQN
 │                                        └── config: map
 │
 ├── schedule: ScheduleSpec (inline)       ScheduleManifest (schedule.yaml)
 │   └── cron, timezone, suspend           ├── cron, timezone, suspend
 │                                         └── (used when pipeline.yaml exists)
 │
 └── runtime: RuntimeSpec ◄─── "custom" step fallback
      └── image, command, args, env

PipelineWorkflow (pipeline.yaml)
 ├── metadata: name, description
 └── steps: []Step (ordered sequence)
      ├── name: unique DNS-safe identifier
      ├── type: StepType discriminator
      │
      ├── [sync]
      │    ├── source ──────► Asset (type=source)
      │    └── sink ────────► Asset (type=sink)
      │
      ├── [transform]
      │    └── asset ───────► Asset (type=model-engine)
      │
      ├── [test]
      │    ├── asset ───────► Asset (any type)
      │    └── command: []string
      │
      ├── [publish]
      │    ├── notify: NotifyConfig
      │    └── promote: bool
      │
      └── [custom]
           ├── image: string
           ├── command: []string
           ├── args: []string
           └── env: []EnvVar
```

## Entities

### 1. PipelineWorkflow

Represents a multi-step pipeline definition stored in `pipeline.yaml` at the project root.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `string` | Yes | API version. Value: `data.infoblox.com/v1alpha1` |
| `kind` | `string` | Yes | Resource kind. Value: `PipelineWorkflow` |
| `metadata.name` | `string` | Yes | Pipeline name. DNS-safe: lowercase, 3–63 chars, starts with letter. |
| `metadata.description` | `string` | No | Human-readable description of the pipeline's purpose. |
| `steps` | `[]Step` | Yes | Ordered sequence of pipeline steps. Minimum 1 step required. |

**Constraints**:
- One `pipeline.yaml` per data package (project root)
- Step names must be unique within the pipeline
- At least one step is required

**Lifecycle**: Created → Validated → Executed → (optionally Scheduled)

### 2. Step

A single unit of work within a pipeline. Steps execute sequentially top-to-bottom.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | `string` | Yes | Unique step identifier. DNS-safe: lowercase, 3–63 chars, starts with letter. |
| `type` | `StepType` | Yes | Discriminator: `sync`, `transform`, `test`, `publish`, `custom` |
| `description` | `string` | No | Human-readable step description. |
| `source` | `string` | sync only | Name of source asset. Must resolve to existing asset of type `source`. |
| `sink` | `string` | sync only | Name of sink asset. Must resolve to existing asset of type `sink`. |
| `asset` | `string` | transform, test | Name of asset. Transform requires `model-engine` type. |
| `command` | `[]string` | test only | Command to execute for test step (e.g., `["dbt", "test"]`). |
| `image` | `string` | custom only | Container image for custom step. |
| `args` | `[]string` | custom only | Container arguments. |
| `env` | `[]EnvVar` | No | Additional environment variables for this step. |
| `notify` | `NotifyConfig` | publish only | Notification configuration. |
| `promote` | `bool` | No | Whether to trigger environment promotion (publish step). Default: `false`. |

**Validation Rules per Type**:

| Step Type | Required Fields | Asset Type Constraint |
|-----------|----------------|----------------------|
| `sync` | `source`, `sink` | source → `AssetType=source`, sink → `AssetType=sink` |
| `transform` | `asset` | `AssetType=model-engine` |
| `test` | `asset`, `command` | Any asset type |
| `publish` | (none required) | N/A |
| `custom` | `image` | N/A (no asset reference) |

### 3. StepType

Enumeration of supported step types.

| Value | Purpose | Asset Reference |
|-------|---------|----------------|
| `sync` | Source-to-sink data movement | source + sink assets |
| `transform` | Model engine execution (e.g., dbt run) | model-engine asset |
| `test` | Validation/assertion (e.g., dbt test) | Any asset + command |
| `publish` | Notification and optional promotion | None |
| `custom` | Single container execution (backward compat) | None (uses image directly) |

### 4. ScheduleManifest

Cron-based execution timing for a pipeline. Stored in `schedule.yaml` at the project root.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `string` | Yes | API version. Value: `data.infoblox.com/v1alpha1` |
| `kind` | `string` | Yes | Resource kind. Value: `Schedule` |
| `cron` | `string` | Yes | Standard 5-field cron expression (e.g., `"0 */6 * * *"`). |
| `timezone` | `string` | No | IANA timezone (e.g., `"UTC"`, `"America/New_York"`). Default: `UTC`. |
| `suspend` | `bool` | No | Whether the schedule is suspended. Default: `false`. |

**Constraints**:
- `cron` must be a valid 5-field cron expression
- `timezone` must be a valid IANA timezone identifier
- One `schedule.yaml` per data package
- When `pipeline.yaml` exists, `schedule.yaml` applies to the pipeline
- When only `dp.yaml` exists (no `pipeline.yaml`), the inline `spec.schedule` continues to work

### 5. NotifyConfig

Configuration for publish step notifications.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `channels` | `[]string` | No | Notification channels (e.g., Slack channels). |
| `recipients` | `[]string` | No | Notification recipients (e.g., email addresses). |

### 6. Template

Pre-defined pipeline structure used by `dp pipeline create --template`.

| Template Name | Steps Generated | Description |
|---------------|-----------------|-------------|
| `sync-transform-test` | sync → transform → test → publish | Full ETL workflow with validation |
| `sync-only` | sync | Simple data synchronization |
| `custom` | custom | Single container (backward compatible) |

Templates are embedded in the SDK binary via `embed.FS`.

## State Transitions

### Pipeline Execution State

```text
                     ┌──── step fails ────┐
                     ▼                    │
PENDING ──► RUNNING ──► step success ──► next step? ──► COMPLETED
                     │                    │ no
                     │                    ▼
                     │               COMPLETED
                     │
                     └── SIGINT/SIGTERM ──► CANCELLED
                     │
                     └── step fails ──► FAILED (step N)
```

### Step Execution State

```text
PENDING ──► RUNNING ──► COMPLETED
                    ├──► FAILED
                    └──► SKIPPED (prior step failed or pipeline cancelled)
```

## Validation Error Codes

| Code | Entity | Condition |
|------|--------|-----------|
| E080 | PipelineWorkflow | Missing required field (`apiVersion`, `kind`, `metadata.name`, `steps`) |
| E081 | PipelineWorkflow | Invalid `apiVersion` value |
| E082 | PipelineWorkflow | Invalid `kind` value (must be `PipelineWorkflow`) |
| E083 | PipelineWorkflow | Empty steps array (at least one step required) |
| E084 | Step | Invalid step name (not DNS-safe, wrong length) |
| E085 | Step | Duplicate step name within pipeline |
| E086 | Step | Invalid step type (not one of: sync, transform, test, publish, custom) |
| E087 | Step | Missing required field for step type (e.g., sync missing source/sink) |
| E088 | Step | Referenced asset not found in project |
| E089 | Step | Asset type mismatch (e.g., sync source references a `model-engine` asset) |
| E090 | Step | Custom step missing `image` field |
| E091 | PipelineWorkflow | Pipeline name invalid (not DNS-safe) |
| E100 | Schedule | Missing required field (`cron`) |
| E101 | Schedule | Invalid cron expression syntax |
| E102 | Schedule | Invalid timezone identifier |
| E103 | Schedule | Invalid `apiVersion` or `kind` |

## File Layout

```text
my-data-package/
├── dp.yaml                    # DataPackage manifest (existing)
├── pipeline.yaml              # PipelineWorkflow manifest (NEW)
├── schedule.yaml              # ScheduleManifest (NEW, optional)
├── assets/
│   ├── sources/
│   │   └── aws-security/
│   │       └── asset.yaml
│   ├── sinks/
│   │   └── raw-output/
│   │       └── asset.yaml
│   └── models/
│       └── dbt-transform/
│           └── asset.yaml
└── bindings.yaml              # Infrastructure bindings (existing)
```
