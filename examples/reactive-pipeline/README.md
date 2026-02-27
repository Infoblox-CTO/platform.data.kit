# Reactive Pipeline Example

This example demonstrates three chained transforms using the reactive pipeline
model. Each transform is independently deployable and declares its own trigger
policy.

## Structure

```
assets/
  raw-events/dp.yaml          Kafka topic (source events)
  raw-events-parquet/dp.yaml   S3 parquet (staging)
  enriched-events/dp.yaml      S3 parquet (curated, with lineage)
  event-summary/dp.yaml        Warehouse table (aggregated)

transforms/
  ingest/dp.yaml               CloudQuery: Kafka → S3   (trigger: on-change)
  enrich/dp.yaml               Python: add user info     (trigger: on-change)
  aggregate/dp.yaml            dbt: daily rollups        (trigger: schedule)
```

## View the Dependency Graph

```bash
# Text tree
dp pipeline show --all --scan-dir ./examples/reactive-pipeline

# Mermaid diagram
dp pipeline show --all --scan-dir ./examples/reactive-pipeline --output mermaid

# Filter to a specific destination
dp pipeline show --destination event-summary --scan-dir ./examples/reactive-pipeline

# JSON format
dp pipeline show --all --scan-dir ./examples/reactive-pipeline --output json

# Graphviz DOT
dp pipeline show --all --scan-dir ./examples/reactive-pipeline --output dot
```

## Key Concepts

1. **No central pipeline.yaml** — each transform declares its own inputs,
   outputs, and trigger policy.
2. **Reactive triggers** — `ingest` and `enrich` run when their inputs change;
   `aggregate` runs on a 6-hour schedule.
3. **Pipeline is a query** — `dp pipeline show` scans manifests and renders
   the dependency graph. It is not a deployed artifact.
4. **Assets carry version** — each asset has a semver version in metadata,
   enabling tag-based resolution for loose coupling.
