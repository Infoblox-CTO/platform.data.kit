# Reactive Pipeline Example

This example demonstrates three chained transforms using the reactive pipeline
model. Each transform is independently deployable and declares its own trigger
policy.

## Structure

```
datasets/
  raw-events/dk.yaml          Kafka topic (source events)
  raw-events-parquet/dk.yaml   S3 parquet (staging)
  enriched-events/dk.yaml      S3 parquet (curated, with lineage)
  event-summary/dk.yaml        Warehouse table (aggregated)

transforms/
  ingest/dk.yaml               CloudQuery: Kafka → S3   (trigger: on-change)
  enrich/dk.yaml               Python: add user info     (trigger: on-change)
  aggregate/dk.yaml            dbt: daily rollups        (trigger: schedule)
```

## View the Dependency Graph

```bash
# Text tree
dk pipeline show --scan-dir ./examples/reactive-pipeline

# Mermaid diagram
dk pipeline show --scan-dir ./examples/reactive-pipeline --output mermaid

# Filter to a specific destination
dk pipeline show --destination event-summary --scan-dir ./examples/reactive-pipeline

# JSON format
dk pipeline show --scan-dir ./examples/reactive-pipeline --output json

# Graphviz DOT
dk pipeline show --scan-dir ./examples/reactive-pipeline --output dot
```

## Key Concepts

1. **No central pipeline manifest** — each transform declares its own inputs,
   outputs, and trigger policy.
2. **Reactive triggers** — `ingest` and `enrich` run when their inputs change;
   `aggregate` runs on a 6-hour schedule.
3. **Pipeline is a query** — `dk pipeline show` scans manifests and renders
   the dependency graph. It is not a deployed artifact.
4. **DataSets carry version** — each dataset has a semver version in metadata,
   enabling tag-based resolution for loose coupling.
