# Research: Consolidate DataPackage Manifest

**Feature**: 004-consolidate-manifest
**Date**: 2026-01-22

## Research Tasks

### 1. Existing Pipeline Config in pipeline.yaml

**Question**: What configuration exists in pipeline.yaml that needs to move to dp.yaml?

**Finding**: Current `PipelineSpec` in [contracts/pipeline.go](../../contracts/pipeline.go) contains:

| Field | Type | Purpose | Move to dp.yaml? |
|-------|------|---------|------------------|
| `image` | string | Container image | ✅ Yes → `spec.runtime.image` |
| `command` | []string | Override entrypoint | ✅ Yes → `spec.runtime.command` |
| `args` | []string | Entrypoint args | ✅ Yes → `spec.runtime.args` |
| `env` | []EnvVar | Custom env vars | ✅ Yes → `spec.runtime.env` |
| `envFrom` | []EnvFromSource | Secret/ConfigMap refs | ✅ Yes → `spec.runtime.envFrom` |
| `replicas` | int | Parallel instances | ✅ Yes → `spec.runtime.replicas` |
| `bindings` | []BindingRef | Binding references | ❌ No - already in dp.yaml via inputs/outputs |
| `serviceAccountName` | string | K8s SA | ✅ Yes → `spec.runtime.serviceAccountName` |

**Additional fields from examples**:
- `timeout` (2h) → `spec.runtime.timeout`
- `retries` (2) → `spec.runtime.retries`
- `successfulJobsHistoryLimit` (3) → `spec.runtime.successfulJobsHistoryLimit`
- `failedJobsHistoryLimit` (5) → `spec.runtime.failedJobsHistoryLimit`

**Decision**: Create `RuntimeSpec` struct with all above fields, embed in `DataPackageSpec`.

---

### 2. Binding-to-EnvVar Mapping Convention

**Question**: How should bindings be automatically mapped to environment variables?

**Finding**: Current approach in pipeline.yaml is explicit:
```yaml
env:
  - name: KAFKA_BROKERS
    valueFrom:
      bindingRef: input.events.brokers
```

**Options Considered**:

| Option | Format | Example |
|--------|--------|---------|
| A. Dot-to-underscore | `input.events.brokers` → `INPUT_EVENTS_BROKERS` | Simple, predictable |
| B. Prefixed | `input.events.brokers` → `DP_INPUT_EVENTS_BROKERS` | Avoids collisions |
| C. Camel-to-snake | `input.events.brokers` → `INPUT_EVENTS_BROKERS` | Same as A |

**Decision**: Option A - Simple dot-to-underscore, uppercase.
- Algorithm: `strings.ToUpper(strings.ReplaceAll(bindingPath, ".", "_"))`
- `input.events.brokers` → `INPUT_EVENTS_BROKERS`
- `output.lake.bucket` → `OUTPUT_LAKE_BUCKET`

**Collision Handling**: If multiple bindings produce the same env var name, last one wins with a warning logged.

---

### 3. Override Merging Strategy

**Question**: How should `-f overrides.yaml` and `--set key=value` merge with dp.yaml?

**Finding**: Helm uses strategic merge with precedence:
1. Base values (values.yaml)
2. Override files in order
3. --set flags in order

**Decision**: Same approach:
1. Load dp.yaml as base
2. For each `-f file.yaml`, deep merge into base
3. For each `--set key=value`, set path in merged result
4. Validate final merged result

**Deep Merge Rules**:
- Scalars: Override replaces
- Maps: Recursive merge
- Arrays: Override replaces (no array merge - too complex)

**--set Path Syntax**:
- Dot notation: `spec.resources.memory=8Gi`
- Nested: `spec.runtime.env[0].value=debug`
- Arrays not supported in --set (use -f for complex changes)

---

### 4. Deprecation Warning for pipeline.yaml

**Question**: How should we handle existing pipeline.yaml files?

**Decision**:
- If `pipeline.yaml` exists alongside `dp.yaml` with runtime section: Warning, ignore pipeline.yaml
- If `pipeline.yaml` exists but no runtime in dp.yaml: Error with migration instructions
- If only dp.yaml (no runtime): Error - runtime required

**Warning Message**:
```
⚠ Warning: pipeline.yaml is deprecated and will be ignored.
  Runtime configuration should be in dp.yaml under spec.runtime.
  See https://docs.dp.io/migration/consolidate-manifest for migration guide.
```

---

### 5. dp show Command Design

**Question**: What should `dp show` output?

**Decision**:
- Output: YAML of merged manifest to stdout
- Flags:
  - `-f file.yaml` - Apply overrides before showing
  - `--set key=value` - Apply overrides before showing
  - `--output json|yaml` - Output format (default yaml)
  - `--effective` - Show only effective values (strip null/empty)

**Example**:
```bash
dp show                              # Show dp.yaml as-is
dp show -f prod.yaml                 # Show merged with prod overrides
dp show --set spec.resources.memory=8Gi  # Show with override applied
```

---

### 6. Files Requiring Updates

**Documentation** (20+ references to pipeline.yaml):
- docs/concepts/data-packages.md
- docs/concepts/index.md
- docs/concepts/overview.md
- docs/getting-started/quickstart.md
- docs/reference/cli.md
- docs/reference/index.md
- docs/reference/manifest-schema.md
- docs/troubleshooting/common-issues.md
- docs/troubleshooting/faq.md
- docs/tutorials/kafka-to-s3.md
- docs/tutorials/promoting-packages.md

**Examples**:
- examples/kafka-s3-pipeline/pipeline.yaml → Remove, merge into dp.yaml

**Code**:
- contracts/datapackage.go - Add RuntimeSpec
- sdk/manifest/parser.go - Handle runtime section
- sdk/runner/docker.go - Auto-map bindings
- cli/cmd/run.go - Add --set and -f flags
- cli/cmd/show.go - New command

---

## Summary of Decisions

| Topic | Decision |
|-------|----------|
| Runtime section | New `spec.runtime` in DataPackageSpec |
| Binding mapping | `input.events.brokers` → `INPUT_EVENTS_BROKERS` |
| Override precedence | dp.yaml < -f files < --set flags |
| Merge strategy | Deep merge maps, replace scalars/arrays |
| Deprecation | Warning if pipeline.yaml present, error if no runtime |
| dp show | Output merged manifest with override support |
