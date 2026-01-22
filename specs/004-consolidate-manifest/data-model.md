# Data Model: Consolidate DataPackage Manifest

**Feature**: 004-consolidate-manifest
**Date**: 2026-01-22

## Entities

### RuntimeSpec (NEW)

Container runtime configuration, previously in pipeline.yaml.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `image` | string | ✅ Yes | - | Container image with optional `${VAR}` substitution |
| `command` | []string | No | - | Override container entrypoint |
| `args` | []string | No | - | Arguments to entrypoint |
| `env` | []EnvVar | No | - | Custom environment variables |
| `envFrom` | []EnvFromSource | No | - | Secret/ConfigMap references |
| `timeout` | duration | No | `1h` | Maximum execution time |
| `retries` | int | No | `3` | Retry count on failure |
| `replicas` | int | No | `1` | Parallel instances |
| `serviceAccountName` | string | No | - | Kubernetes service account |
| `successfulJobsHistoryLimit` | int | No | `3` | Jobs to keep on success |
| `failedJobsHistoryLimit` | int | No | `5` | Jobs to keep on failure |

### DataPackageSpec (MODIFIED)

Add `Runtime` field to existing spec.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | PackageType | ✅ Yes | Package type (pipeline, model, dataset) |
| `description` | string | ✅ Yes | Human-readable description |
| `owner` | string | ✅ Yes | Team or individual owner |
| `inputs` | []ArtifactContract | No | Input dependencies |
| `outputs` | []ArtifactContract | No | Output artifacts |
| `schedule` | *ScheduleSpec | No | Scheduling configuration |
| `resources` | *ResourceSpec | No | CPU/memory requirements |
| `lineage` | *LineageSpec | No | Lineage tracking config |
| **`runtime`** | ***RuntimeSpec** | **✅ Yes** | **Container runtime config (NEW)** |

### EnvVar (REUSED from pipeline.go)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ Yes | Environment variable name |
| `value` | string | No | Static value |
| `valueFrom` | *EnvVarSource | No | Dynamic value source |

### EnvFromSource (REUSED from pipeline.go)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `prefix` | string | No | Prefix to add to all var names |
| `secretRef` | *SecretRef | No | Reference to K8s secret |
| `configMapRef` | *ConfigMapRef | No | Reference to K8s configmap |

## Relationships

```
DataPackage
    └── spec: DataPackageSpec
            ├── inputs: []ArtifactContract
            │       └── binding: string → auto-maps to env vars
            ├── outputs: []ArtifactContract
            │       └── binding: string → auto-maps to env vars
            ├── schedule: ScheduleSpec
            ├── resources: ResourceSpec
            └── runtime: RuntimeSpec (NEW)
                    ├── image: string
                    ├── env: []EnvVar
                    └── envFrom: []EnvFromSource
```

## Binding-to-EnvVar Mapping

### Input Mapping

| Binding Path | Generated Env Var |
|-------------|-------------------|
| `input.events.brokers` | `INPUT_EVENTS_BROKERS` |
| `input.events.topic` | `INPUT_EVENTS_TOPIC` |
| `input.events.consumerGroup` | `INPUT_EVENTS_CONSUMERGROUP` |

### Output Mapping

| Binding Path | Generated Env Var |
|-------------|-------------------|
| `output.lake.bucket` | `OUTPUT_LAKE_BUCKET` |
| `output.lake.prefix` | `OUTPUT_LAKE_PREFIX` |
| `output.lake.region` | `OUTPUT_LAKE_REGION` |

### Mapping Algorithm

```go
func BindingToEnvVar(bindingPath string) string {
    return strings.ToUpper(strings.ReplaceAll(bindingPath, ".", "_"))
}
```

## Override Merging

### Merge Precedence

1. **Base**: dp.yaml (lowest priority)
2. **Override Files**: `-f file1.yaml -f file2.yaml` (in order)
3. **Set Flags**: `--set key=value` (highest priority)

### Merge Rules

| Type | Behavior |
|------|----------|
| Scalar (string, int, bool) | Replace |
| Map | Recursive merge |
| Array | Replace entirely |
| Null | Remove key |

### Example

**dp.yaml**:
```yaml
spec:
  resources:
    cpu: "2"
    memory: "4Gi"
  runtime:
    timeout: 1h
```

**prod-overrides.yaml**:
```yaml
spec:
  resources:
    memory: "8Gi"
  runtime:
    timeout: 2h
```

**Command**: `dp run -f prod-overrides.yaml --set spec.runtime.retries=5`

**Result**:
```yaml
spec:
  resources:
    cpu: "2"        # From base
    memory: "8Gi"   # From override file
  runtime:
    timeout: 2h     # From override file
    retries: 5      # From --set flag
```

## State Transitions

N/A - This feature doesn't introduce stateful entities.

## Validation Rules

### RuntimeSpec Validation

| Field | Rule | Error Message |
|-------|------|---------------|
| `image` | Required, non-empty | "spec.runtime.image is required" |
| `image` | Valid image format | "spec.runtime.image must be a valid container image reference" |
| `timeout` | Positive duration | "spec.runtime.timeout must be positive" |
| `retries` | Non-negative integer | "spec.runtime.retries must be >= 0" |
| `replicas` | Positive integer | "spec.runtime.replicas must be >= 1" |

### Override Validation

| Scenario | Error Message |
|----------|---------------|
| Path doesn't exist | "override path 'foo.bar' does not exist in schema" |
| Type mismatch | "cannot set 'spec.resources.cpu' to non-string value" |
| Invalid value | "value 'abc' is not valid for 'spec.runtime.retries' (expected integer)" |
