# Quickstart: Pipeline Workflows

This guide walks through the complete pipeline workflow — from scaffolding a multi-step pipeline to executing it locally with backfill and scheduling.

## Prerequisites

- `dp` CLI installed and on your PATH
- Docker running locally
- An existing data package initialized with `dp init`
- At least one source asset and one sink asset created (see `dp asset create`)

## 1. Set Up a Project with Assets

If you don't have assets yet, create them first:

```bash
# Initialize a new data package
dp init my-data-package --type pipeline --owner data-team

cd my-data-package

# Create a source asset (e.g., AWS S3 source)
dp asset create aws-security --ext cloudquery.source.aws --version v1

# Create a sink asset (e.g., PostgreSQL destination)
dp asset create raw-output --ext cloudquery.sink.postgresql --version v1

# Create a model-engine asset (e.g., dbt transformation)
dp asset create dbt-transform --ext dbt.model-engine.core --version v1
```

## 2. Scaffold a Pipeline

Use `dp pipeline create` to generate a `pipeline.yaml` from a template:

```bash
# Create a full ETL pipeline (sync → transform → test → publish)
dp pipeline create security-pipeline --template sync-transform-test
```

This generates `pipeline.yaml` at the project root:

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: security-pipeline
  description: "Pipeline created from sync-transform-test template"
steps:
  - name: sync-data
    type: sync
    source: aws-security
    sink: raw-output
  - name: transform-data
    type: transform
    asset: dbt-transform
  - name: test-output
    type: test
    asset: dbt-transform
    command: ["dbt", "test"]
  - name: publish-results
    type: publish
    notify:
      channels: ["#data-alerts"]
    promote: false
```

### Other Templates

```bash
# Simple sync-only pipeline
dp pipeline create my-sync --template sync-only

# Single container (backward compatible with existing dp run)
dp pipeline create legacy-job --template custom
```

## 3. Validate the Pipeline

Run `dp validate` to check the pipeline definition and all asset references:

```bash
dp validate
```

Expected output:

```
✓ dp.yaml is valid
✓ pipeline.yaml is valid (4 steps)
✓ All asset references resolved
✓ bindings.yaml is valid
```

If an asset reference is missing:

```
✗ pipeline.yaml: step "sync-data" references asset "aws-security" which was not found (E088)
  Hint: Run 'dp asset create aws-security --ext <extension-fqn>' to create it.
```

## 4. Run the Pipeline Locally

Execute all steps sequentially:

```bash
dp pipeline run security-pipeline --env dev
```

Expected output:

```
▶ Running pipeline "security-pipeline" (4 steps)

[sync-data] Starting sync: aws-security → raw-output
[sync-data] Syncing 3 tables...
[sync-data] ✓ Completed in 12s

[transform-data] Starting transform: dbt-transform
[transform-data] Running dbt run...
[transform-data] ✓ Completed in 8s

[test-output] Starting test: dbt-transform
[test-output] Running dbt test...
[test-output] ✓ 14 tests passed in 5s

[publish-results] Starting publish
[publish-results] Notified #data-alerts
[publish-results] ✓ Completed in 1s

✓ Pipeline "security-pipeline" completed in 26s (4/4 steps)
```

### Run a Single Step

```bash
# Execute only the sync step
dp pipeline run security-pipeline --env dev --step sync-data
```

### Failure Handling

If a step fails, subsequent steps are skipped:

```
[sync-data] ✓ Completed in 12s
[transform-data] ✗ Failed: exit code 1 — "dbt compilation error in model stg_events"
[test-output] ⊘ Skipped (prior step failed)
[publish-results] ⊘ Skipped (prior step failed)

✗ Pipeline "security-pipeline" failed at step "transform-data" (1/4 steps completed)
```

## 5. Backfill Historical Data

Re-execute the sync step for a specific date range:

```bash
dp pipeline backfill security-pipeline --from 2026-01-01 --to 2026-01-31
```

Expected output:

```
▶ Backfill "security-pipeline" sync steps (2026-01-01 to 2026-01-31)

[sync-data] Starting sync: aws-security → raw-output
[sync-data] Backfill range: 2026-01-01 to 2026-01-31
[sync-data] ✓ Completed in 45s

✓ Backfill completed in 45s
```

The date range is injected as environment variables `DP_BACKFILL_FROM` and `DP_BACKFILL_TO` into the sync step's container.

## 6. Add a Schedule (Optional)

Create a `schedule.yaml` to run the pipeline on a cron schedule:

```yaml
# schedule.yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Schedule
cron: "0 */6 * * *"
timezone: "UTC"
suspend: false
```

Validate it:

```bash
dp validate
```

Expected output:

```
✓ dp.yaml is valid
✓ pipeline.yaml is valid (4 steps)
✓ schedule.yaml is valid (every 6 hours, UTC)
✓ All asset references resolved
```

## 7. Inspect the Pipeline

View the pipeline definition:

```bash
dp pipeline show security-pipeline
```

Expected output:

```
Pipeline: security-pipeline
Description: Pipeline created from sync-transform-test template

Steps:
  1. sync-data       (sync)       aws-security → raw-output
  2. transform-data  (transform)  dbt-transform
  3. test-output     (test)       dbt-transform [dbt test]
  4. publish-results (publish)    #data-alerts

Schedule: every 6 hours (UTC)
```

JSON output:

```bash
dp pipeline show security-pipeline --output json
```

## 8. Backward Compatibility

Existing packages without `pipeline.yaml` continue to work unchanged:

```bash
# This still works exactly as before for packages without pipeline.yaml
dp run --env dev
```

## Summary of Commands

| Command | Description |
|---------|-------------|
| `dp pipeline create <name> --template <tmpl>` | Scaffold a pipeline.yaml from a template |
| `dp pipeline run <name> --env <env>` | Execute pipeline steps sequentially |
| `dp pipeline run <name> --env <env> --step <step>` | Execute a single step |
| `dp pipeline backfill <name> --from <date> --to <date>` | Re-execute sync steps for a date range |
| `dp pipeline show <name>` | Display pipeline definition |
| `dp pipeline show <name> --output json` | Display pipeline as JSON |
| `dp validate` | Validate pipeline.yaml, schedule.yaml, and asset references |
