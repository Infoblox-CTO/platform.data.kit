# Quickstart: Consolidated DataPackage Manifest

This guide shows how to create and run a DataPackage using the new single-file manifest format.

## Before You Begin

Ensure you have:
- `dp` CLI installed
- Docker running
- Local dev stack running (`dp dev up`)

## 1. Create a DataPackage

Create a single `dp.yaml` file with all configuration:

```yaml
# dp.yaml - Complete DataPackage manifest
apiVersion: dp.io/v1alpha1
kind: DataPackage
metadata:
  name: my-pipeline
  namespace: analytics
spec:
  type: pipeline
  description: Processes events from Kafka to S3
  owner: data-team

  inputs:
    - name: events
      type: kafka-topic
      binding: input.events

  outputs:
    - name: processed
      type: s3-prefix
      binding: output.lake

  schedule:
    cron: "0 */6 * * *"

  resources:
    cpu: "2"
    memory: "4Gi"

  # Runtime configuration (previously in pipeline.yaml)
  runtime:
    image: "${REGISTRY}/my-pipeline:${VERSION}"
    timeout: 2h
    retries: 3
    env:
      - name: LOG_LEVEL
        value: info
    envFrom:
      - secretRef:
          name: pipeline-credentials
```

## 2. Validate

```bash
dp validate
```

Expected output:
```
✓ dp.yaml: valid
  - Package: my-pipeline (pipeline)
  - Inputs: 1 (events)
  - Outputs: 1 (processed)
  - Runtime: image configured
```

## 3. Run Locally

```bash
dp run
```

Bindings are automatically mapped to environment variables:
- `input.events.brokers` → `INPUT_EVENTS_BROKERS`
- `output.lake.bucket` → `OUTPUT_LAKE_BUCKET`

## 4. Override at Runtime

Override settings without editing dp.yaml:

```bash
# Override single values
dp run --set spec.resources.memory=8Gi

# Use an overrides file
dp run -f prod-overrides.yaml

# Combine both (--set takes precedence)
dp run -f prod-overrides.yaml --set spec.runtime.timeout=4h
```

### Example overrides file

```yaml
# prod-overrides.yaml
spec:
  resources:
    cpu: "4"
    memory: "8Gi"
  runtime:
    timeout: 4h
    retries: 5
```

## 5. Preview Merged Configuration

See effective configuration before running:

```bash
dp show -f prod-overrides.yaml
```

## Key Changes from Previous Format

| Before (Two Files) | After (Single File) |
|-------------------|---------------------|
| `dp.yaml` + `pipeline.yaml` | `dp.yaml` only |
| Manual env var mapping | Automatic binding → env var |
| No runtime overrides | `--set` and `-f` flags |
| `pipeline.yaml` had image/timeout | `spec.runtime` section |

## Migration

If you have existing `pipeline.yaml` files:

1. Move `image`, `timeout`, `retries`, `env`, `envFrom` to `spec.runtime` in dp.yaml
2. Remove explicit binding-to-env mappings (now automatic)
3. Delete `pipeline.yaml`
4. Run `dp validate` to confirm

The CLI will warn if it detects a `pipeline.yaml` file.
