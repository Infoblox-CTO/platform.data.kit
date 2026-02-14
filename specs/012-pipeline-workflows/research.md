# Research: Pipeline Workflows

**Feature**: 012-pipeline-workflows  
**Date**: 2026-02-14  
**Purpose**: Resolve technical unknowns and select implementation patterns before design phase.

## R-001: Sequential Docker Container Orchestration

**Decision**: Use the existing `os/exec.Command("docker", ...)` shell-out pattern with a new step-aware executor in `sdk/pipeline/executor.go`. Each step runs as a separate `docker run` invocation, sequentially.

**Rationale**:
- Proven in the codebase — `sdk/runner/docker.go` already builds `docker run` args and dispatches via `exec.Command`. No reason to introduce Docker SDK when shell-out is working and tested.
- Step-prefixed output: Wrap each step's stdout/stderr pipes with a `PrefixWriter` that prepends `[step-name]` to every line. The existing `streamOutput` function in `docker.go` already does line-by-line streaming — extend with a prefix.
- Signal handling: The existing `RunStreaming()` demonstrates the pattern — catch SIGINT/SIGTERM, send SIGTERM to the current container's process, wait up to 30s. For multi-step, add a `cancelled` atomic bool; when the signal arrives, stop the current container and break out of the step loop.
- Env var passing: Use a `map[string]string` that accumulates across steps. Each step gets base env (from bindings + opts) plus backfill params when applicable.

**Alternatives Considered**:
| Alternative | Why Rejected |
|---|---|
| Docker SDK (`github.com/docker/docker/client`) | Adds ~50 transitive packages. Constitution Article VIII (Pragmatism) says "defer complexity." Shell-out is sufficient. |
| Docker Compose per-pipeline | Compose models services (long-running), not sequential batch jobs. |
| `docker-compose run --rm` per step | Ties pipelines to Compose files unnecessarily. |

## R-002: pipeline.yaml Schema Design

**Decision**: Use a discriminator field (`type`) on each step, with type-specific fields nested flat under the step. Steps are an ordered list (YAML sequence). Kind is `PipelineWorkflow` (distinct from existing `Pipeline`).

**Rationale**:
- **Discriminator over implicit union**: YAML has no native union. An explicit `type` field maps directly to JSON Schema `if/then/else` with `const` discriminator. GitHub Actions uses implicit detection (fragile); Tekton uses explicit `taskRef` (too verbose). Explicit `type` is clearest.
- **Ordered list over map**: Steps execute top-to-bottom (FR-006). YAML sequences preserve order by definition. Maps have no guaranteed order.
- **Flat fields per step**: Each step type has specific required fields (`source`/`sink` for sync, `asset` for transform/test). Flat is simpler to read/write than nested sub-objects.
- **Kind = `PipelineWorkflow`**: Avoids collision with existing `Pipeline` kind in `contracts/pipeline.go` which handles single-container runtime config.

**Alternatives Considered**:
| Alternative | Why Rejected |
|---|---|
| GitHub Actions-style implicit type detection | Fragile, hard to validate with JSON Schema |
| Tekton TaskRun references | Over-engineered for sequential CLI execution |
| dbt DAG inference from files | Project uses explicit ordering, not DAG |
| Nested type sub-objects (`sync: { ... }`) | Unnecessary nesting; flat is simpler |

**Example Schema**:

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: security-pipeline
  description: "Sync, transform, test, and publish security data"
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

## R-003: Cron Expression Validation

**Decision**: Use `github.com/adhocore/gronx` for cron validation. Use `time.LoadLocation` from the standard library for timezone validation.

**Rationale**:
- `gronx.IsValid(expr)` is a standalone zero-dependency function for validating standard 5-field cron expressions.
- `time.LoadLocation(tz)` handles IANA timezone validation perfectly — returns error for invalid strings.
- Both are pure functions, easily testable, no side effects.
- Validation at `dp validate` time per Article VII (Quality Gates).

**Alternatives Considered**:
| Alternative | Why Rejected |
|---|---|
| `robfig/cron/v3` | Full scheduler with goroutines/logging — overkill for validation only |
| Hand-rolled regex | Fragile, doesn't catch semantic errors (day 32, month 13) |
| Defer to Kubernetes CronJob validation | Violates Article VII — errors must be caught at validate time |

## R-004: Backfill Date Range Handling

**Decision**: Use `time.Parse("2006-01-02", dateStr)` for ISO 8601 date parsing. Inject as `DP_BACKFILL_FROM` and `DP_BACKFILL_TO` environment variables.

**Rationale**:
- Go standard library handles ISO 8601 date parsing natively.
- `DP_` prefix follows existing codebase convention (e.g., `DP_WORKSPACE_PATH`).
- `FROM`/`TO` is more precise than `START`/`END` for date ranges.
- Env vars follow the established pattern in `docker.go` where env vars are appended as `--env KEY=VALUE`.

**Alternatives Considered**:
| Alternative | Why Rejected |
|---|---|
| Container args instead of env vars | Env vars are the established pattern; args would require entrypoint changes |
| `DP_BACKFILL_START/END` naming | FROM/TO is clearer for date ranges |
| RFC 3339 timestamps | Spec explicitly limits to date-only (YYYY-MM-DD) |
| JSON payload via mounted file | Over-engineered for two date strings |

## R-005: Template Embedding Pattern

**Decision**: Use `embed.FS` with Go `text/template` for pipeline templates, following the exact pattern in `cli/internal/templates/`. Place pipeline workflow templates under `sdk/pipeline/templates/` with `.tmpl` extension.

**Rationale**:
- Consistency with existing `cli/internal/templates/renderer.go` which uses `embed.FS` + `text/template`.
- Go templates support conditionals, loops, and the existing `lower`/`title` helpers.
- Templates co-locate with the scaffolder in `sdk/pipeline/` (same as `sdk/asset/` keeps schemas close to the scaffolder).
- Template discovery for `--template` flag: iterate embedded FS entries, strip `.tmpl` suffix → template names.

**Alternatives Considered**:
| Alternative | Why Rejected |
|---|---|
| Raw YAML with `strings.ReplaceAll` | Can't handle conditional step inclusion |
| External template files | Violates assumption: "templates are embedded in the CLI binary" |
| Templates in `cli/internal/templates/` | Couples templates to CLI layer; scaffolder is in SDK |
| YAML anchors/aliases | Brittle, poorly supported by tooling |
| Remote template registry | Out of scope per spec assumptions |

## R-006: Error Code Allocation

**Decision**: Use error code range `E080–E099` for pipeline workflow validation errors. Use `E100–E109` for schedule validation errors.

**Rationale**:
- Existing error code ranges: E001–E003 (DataPackage), E010–E011 (Binding), E020–E021 (Version), E030–E031 (Pipeline), E040–E041 (Runtime), E050–E056 (Pipeline mode), E070–E077 (Asset).
- Next available range: E080. Allocate E080–E099 for pipeline workflows (20 codes covers step validation, asset resolution, name uniqueness, etc.).
- E100–E109 for schedule validation (cron syntax, timezone, etc.).

**Alternatives Considered**:
| Alternative | Why Rejected |
|---|---|
| Continue from E078 | Too tight; leaves no room for future asset error codes |
| Use E-PW-001 prefixed codes | Breaks existing numeric convention |

## R-007: Relationship Between pipeline.yaml and Existing Pipeline Manifest

**Decision**: `pipeline.yaml` (new) and `pipeline-manifest.yaml` (existing) are independent files. The existing `PipelineManifest` describes single-container runtime config (image, env, probes, mode). The new `PipelineWorkflow` describes multi-step orchestration. When both exist, each step in the workflow resolves its runtime from the referenced asset's configuration — the `PipelineManifest` is not used for workflow steps.

**Rationale**:
- Backward compatibility (FR-015): Existing packages with `pipeline-manifest.yaml` and no `pipeline.yaml` continue to use `dp run` unchanged.
- Clean separation: The workflow orchestrates *what* runs and in *what order*. The pipeline manifest (or asset config) describes *how* each individual container runs.
- `custom` step type bridges the two: it references the existing runtime configuration (image, command, args) for backward compatibility.

**Alternatives Considered**:
| Alternative | Why Rejected |
|---|---|
| Merge workflow steps into existing `pipeline-manifest.yaml` | Breaks backward compatibility (Article II); existing consumers depend on current schema |
| Replace `pipeline-manifest.yaml` entirely | Too aggressive; violates incremental delivery (Article VIII) |
| Have workflow steps inherit from PipelineManifest | Creates confusing inheritance; each step should be self-describing via its asset reference |
