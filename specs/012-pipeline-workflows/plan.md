# Implementation Plan: Pipeline Workflows

**Branch**: `012-pipeline-workflows` | **Date**: 2026-02-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/012-pipeline-workflows/spec.md`

## Summary

Extend the DP platform to support multi-step pipeline workflows defined in `pipeline.yaml`. Each step references assets by name and has a type (sync, transform, test, publish, custom). The CLI gains `dp pipeline create`, `dp pipeline run`, `dp pipeline backfill`, and `dp pipeline show` commands. A `schedule.yaml` file supports cron-based scheduling. The existing single-container `dp run` path is preserved via the `custom` step type for full backward compatibility. Implementation follows the established three-module pattern: contracts (types + schemas) → sdk (loader, executor, validator) → cli (commands).

## Technical Context

**Language/Version**: Go 1.25 (all modules)
**Primary Dependencies**: github.com/spf13/cobra v1.8.1 (CLI), gopkg.in/yaml.v3 v3.0.1 (serialization), github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 (JSON Schema validation), oras.land/oras-go/v2 v2.5.0 (OCI registry)
**Storage**: Filesystem — `pipeline.yaml` and `schedule.yaml` as YAML files in project root, assets under `assets/` directory
**Testing**: Go standard library `testing` package, table-driven tests, `t.TempDir()` for filesystem tests, no external assertion libraries
**Target Platform**: macOS/Linux CLI (developer workstation), Docker/k3d for local execution
**Project Type**: Go multi-module monorepo — `contracts/` ← `sdk/` ← `cli/` with `replace` directives
**Performance Goals**: CLI commands respond in <2s for validation and scaffolding; pipeline execution is bounded by container runtime
**Constraints**: No new external dependencies unless justified; backward compatibility with all existing `dp run` / `dp validate` behavior; sequential step execution only (no DAG)
**Scale/Scope**: ~5 new CLI commands, ~3 new contract types, ~2 new JSON schemas, ~4 new SDK packages (pipeline loader, executor, templates, validator extensions)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Article | Requirement | Status | Notes |
|---------|-------------|--------|-------|
| **I — Developer Experience** | Happy path simple; clear actionable output | ✅ PASS | `dp pipeline create --template` → edit → `dp pipeline run` follows bootstrap→run→validate pattern. Step-prefixed logs provide clear output. |
| **II — Contracts Stable** | Machine-readable schemas; additive evolution | ✅ PASS | New `pipeline.yaml` and `schedule.yaml` are additive — no changes to existing `dp.yaml` or `pipeline-manifest.schema.json`. New schemas use JSON Schema Draft 2020-12. |
| **III — Immutability** | Released artifacts immutable; rollback clear | ✅ PASS | Pipeline definitions are project files (not published artifacts). Existing artifact immutability is unchanged. |
| **IV — Separation of Concerns** | Four-layer separation: extension → asset → pipeline → binding | ✅ PASS | Pipelines reference assets by name (not inline config). Assets reference extensions by FQN. Bindings remain separate. Clean layer boundaries maintained. |
| **V — Security** | Least privilege; no committed secrets | ✅ PASS | Pipeline steps inherit env vars from bindings, not hardcoded. No new secret surfaces. |
| **VI — Observability** | Metrics/logs with correlation IDs | ✅ PASS | Step-prefixed structured logs (`[step-name] ...`). Existing lineage events extended per-step. |
| **VII — Quality Gates** | Automated validation before publish/promote | ✅ PASS | `dp validate` extended to validate `pipeline.yaml` and `schedule.yaml`. Asset reference resolution checked at validate time. |
| **VIII — Pragmatism** | Ship incrementally; defer complexity | ✅ PASS | Sequential execution only. No DAG, no parallel steps, no remote execution. Templates embedded in binary. |
| **IX — Maintainability** | Clear module boundaries: contracts ← sdk ← cli | ✅ PASS | Types in contracts, loader/executor in sdk, commands in cli. Dependency direction preserved. |
| **X — Persona Boundaries** | Platform vs data engineer separation | ✅ PASS | Data engineers own pipeline definitions (pipeline.yaml, schedule.yaml). Platform engineers own the step type definitions and execution infrastructure. Pipeline artifacts validatable without platform credentials. |
| **XI — Extensions are Contracts** | Schema-validated at dp validate time | ✅ PASS | `pipeline.yaml` and `schedule.yaml` get JSON schemas. Validation at `dp validate` time, not runtime. |

### Pre-Implementation Gates

| Gate | Status | Evidence |
|------|--------|----------|
| **Workflow Demo** | ✅ | `dp pipeline create` → edit `pipeline.yaml` → `dp pipeline run --env dev` → `dp pipeline show` |
| **Contract Schema** | ✅ | `pipeline-workflow.schema.json` and `schedule.schema.json` will be defined in Phase 1 |
| **Promotion/Rollback** | ✅ | Pipeline definitions travel with the data package — existing promotion/rollback mechanics apply unchanged |
| **Observability** | ✅ | Step-prefixed logs; existing lineage emitter extended per-step |
| **Security/Compliance** | ✅ | No new secret surfaces; env vars from bindings only; PII metadata unchanged |
| **Persona Mapping** | ✅ | Data engineer: owns pipeline.yaml, schedule.yaml, asset references. Platform engineer: owns step type semantics, runtime infrastructure |

## Project Structure

### Documentation (this feature)

```text
specs/012-pipeline-workflows/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (schema drafts)
└── tasks.md             # Phase 2 output (/speckit.tasks — NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
contracts/
├── pipeline_workflow.go          # PipelineWorkflow, Step, StepType types
├── pipeline_workflow_test.go     # Type validation tests
├── schedule.go                   # ScheduleManifest type (dedicated file)
├── schedule_test.go              # Schedule validation tests
└── schemas/
    ├── pipeline-workflow.schema.json   # JSON Schema for pipeline.yaml
    └── schedule.schema.json            # JSON Schema for schedule.yaml

sdk/
├── pipeline/
│   ├── loader.go                 # LoadPipeline(), FindPipeline()
│   ├── loader_test.go
│   ├── executor.go               # Execute() — sequential step runner
│   ├── executor_test.go
│   ├── backfill.go               # Backfill() — date-range re-execution
│   ├── backfill_test.go
│   ├── templates.go              # Embedded pipeline templates
│   ├── templates_test.go
│   ├── scaffolder.go             # Scaffold() — create pipeline.yaml from template
│   └── scaffolder_test.go
├── validate/
│   ├── pipeline_workflow.go      # PipelineWorkflowValidator (extends existing)
│   ├── pipeline_workflow_test.go
│   ├── schedule.go               # ScheduleValidator
│   └── schedule_test.go

cli/cmd/
├── pipeline.go                   # Parent "dp pipeline" command
├── pipeline_create.go            # dp pipeline create <name> --template
├── pipeline_create_test.go
├── pipeline_run.go               # dp pipeline run <name> --env --step
├── pipeline_run_test.go
├── pipeline_backfill.go          # dp pipeline backfill <name> --from --to
├── pipeline_backfill_test.go
├── pipeline_show.go              # dp pipeline show <name> --output
├── pipeline_show_test.go

tests/e2e/
└── pipeline_workflow_test.go     # End-to-end workflow test
```

**Structure Decision**: Follows the established three-module pattern (`contracts` ← `sdk` ← `cli`). New types in `contracts/`, new SDK package `sdk/pipeline/` for loader/executor/scaffolder (parallels `sdk/asset/`), validator extensions in existing `sdk/validate/`, CLI commands in `cli/cmd/` following `pipeline_*.go` naming (parallels `asset_*.go`).

## Complexity Tracking

> No constitution violations detected — all gates pass. No complexity justifications required.
