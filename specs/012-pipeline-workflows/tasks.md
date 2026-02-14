# Tasks: Pipeline Workflows

**Input**: Design documents from `/specs/012-pipeline-workflows/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included — the constitution requires unit tests for all exported functions (Article VII, Technology Standards). Tests follow the established Go table-driven pattern using stdlib `testing` only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Contracts module**: `contracts/` (types, schemas, validation helpers)
- **SDK module**: `sdk/` (pipeline loader, executor, scaffolder, validators)
- **CLI module**: `cli/cmd/` (cobra commands)
- **E2E tests**: `tests/e2e/`
- **Docs**: `docs/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization — new contract types, JSON schemas, and SDK package scaffolding

- [X] T001 Create PipelineWorkflow, Step, StepType, NotifyConfig types and StepType.IsValid() in contracts/pipeline_workflow.go
- [X] T002 Create PipelineWorkflow type validation tests (StepType.IsValid, ValidStepTypes) in contracts/pipeline_workflow_test.go
- [X] T003 [P] Create ScheduleManifest type in contracts/schedule.go
- [X] T004 [P] Create ScheduleManifest type tests in contracts/schedule_test.go
- [X] T005 [P] Create StepStatus, StepResult, PipelineRunResult execution types in contracts/pipeline_workflow.go (append to existing)
- [X] T006 [P] Add pipeline-workflow.schema.json to contracts/schemas/pipeline-workflow.schema.json
- [X] T007 [P] Add schedule.schema.json to contracts/schemas/schedule.schema.json

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: SDK pipeline loader and validator infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T008 Create sdk/pipeline/ package with LoadPipeline() and FindPipeline() in sdk/pipeline/loader.go
- [X] T009 Create loader tests (valid YAML, missing file, invalid YAML, malformed steps) in sdk/pipeline/loader_test.go
- [X] T010 [P] Create PipelineWorkflowValidator implementing Validator interface in sdk/validate/pipeline_workflow.go — validates required fields (E080–E083), step names (E084–E085), step types (E086), type-specific required fields (E087, E090), pipeline name (E091)
- [X] T011 [P] Create PipelineWorkflowValidator tests (table-driven, all error codes E080–E091) in sdk/validate/pipeline_workflow_test.go
- [X] T012 [P] Create ScheduleValidator implementing Validator interface in sdk/validate/schedule.go — validates cron via gronx (E100–E101), timezone via time.LoadLocation (E102), apiVersion/kind (E103)
- [X] T013 [P] Create ScheduleValidator tests (table-driven, all error codes E100–E103) in sdk/validate/schedule_test.go
- [X] T014 Add `github.com/adhocore/gronx` dependency to sdk/go.mod for cron validation
- [X] T015 Create asset reference resolver — ValidateAssetReferences() checks step asset names resolve to existing assets with correct type in sdk/validate/pipeline_workflow.go (E088–E089)
- [X] T016 Create asset reference resolver tests (missing asset, type mismatch per step type) in sdk/validate/pipeline_workflow_test.go
- [X] T017 Integrate PipelineWorkflowValidator and ScheduleValidator into AggregateValidator in sdk/validate/aggregate.go — detect pipeline.yaml and schedule.yaml if present

**Checkpoint**: Foundation ready — contracts compiled, pipeline.yaml loadable, validators functional

---

## Phase 3: User Story 1 — Scaffold a Multi-Step Pipeline (Priority: P1) 🎯 MVP

**Goal**: `dp pipeline create <name> --template <template>` scaffolds a `pipeline.yaml` with pre-configured steps referencing project assets

**Independent Test**: Run `dp pipeline create security-pipeline --template sync-transform-test` in a project with assets. Verify `pipeline.yaml` is created with four steps. Run `dp validate` and confirm it passes.

### Implementation for User Story 1

- [X] T018 [P] [US1] Create embedded pipeline templates (sync-transform-test.tmpl, sync-only.tmpl, custom.tmpl) in sdk/pipeline/templates/
- [X] T019 [P] [US1] Create template renderer with ListTemplates() and RenderTemplate() in sdk/pipeline/templates.go
- [X] T020 [US1] Create template renderer tests (list available, render each template, template output valid YAML) in sdk/pipeline/templates_test.go
- [X] T021 [US1] Create pipeline scaffolder — ScaffoldPipeline() with ScaffoldOpts (name, template, projectDir, force) that discovers assets and renders template in sdk/pipeline/scaffolder.go
- [X] T022 [US1] Create scaffolder tests (with assets, without assets TODO placeholders, force overwrite, existing file error) in sdk/pipeline/scaffolder_test.go
- [X] T023 [US1] Create parent `dp pipeline` cobra command in cli/cmd/pipeline.go
- [X] T024 [US1] Create `dp pipeline create` command with --template, --force flags in cli/cmd/pipeline_create.go
- [X] T025 [US1] Create pipeline_create tests (flag registration, template validation, --force behavior, output message) in cli/cmd/pipeline_create_test.go

**Checkpoint**: `dp pipeline create` works end-to-end, `dp validate` accepts the generated pipeline.yaml

---

## Phase 4: User Story 2 — Execute a Pipeline Locally (Priority: P1)

**Goal**: `dp pipeline run <name> --env <env>` executes pipeline steps sequentially with step-prefixed logs, halting on failure

**Independent Test**: Create a pipeline with three steps. Run `dp pipeline run my-pipeline --env dev`. Verify steps execute in order, logs show `[step-name]` prefixes, and failure in step 2 stops execution.

### Implementation for User Story 2

- [X] T026 [P] [US2] Create PrefixWriter (io.Writer wrapper that prepends `[step-name]` to each line) in sdk/pipeline/executor.go
- [X] T027 [P] [US2] Create PrefixWriter tests (single line, multi-line, partial lines, empty) in sdk/pipeline/executor_test.go
- [X] T028 [US2] Create StepExecutor — executeStep() that resolves step type to docker run args, builds env map, runs container via os/exec in sdk/pipeline/executor.go
- [X] T029 [US2] Create pipeline Execute() — iterates steps sequentially, handles cancellation (atomic bool + SIGINT/SIGTERM), builds PipelineRunResult in sdk/pipeline/executor.go
- [X] T030 [US2] Create executor tests (all steps pass, step failure skips remaining, single step --step flag, cancellation, missing pipeline) in sdk/pipeline/executor_test.go
- [X] T031 [US2] Create `dp pipeline run` command with --env, --step flags in cli/cmd/pipeline_run.go
- [X] T032 [US2] Create pipeline_run tests (flag registration, env required, --step validation, no pipeline.yaml error message) in cli/cmd/pipeline_run_test.go

**Checkpoint**: `dp pipeline run` executes steps sequentially with prefixed logs, fails fast on step failure

---

## Phase 5: User Story 5 — Backward Compatibility (Priority: P1)

**Goal**: Existing `dp run` works unchanged for packages without `pipeline.yaml`. Custom step type bridges to existing single-container execution.

**Independent Test**: Take an existing package with `dp.yaml` and no `pipeline.yaml`. Run `dp run`. Verify identical behavior to before this feature.

### Implementation for User Story 5

- [X] T033 [US5] Add custom step execution path in sdk/pipeline/executor.go — executeCustomStep() uses image/command/args/env directly (same as existing DockerRunner pattern)
- [X] T034 [US5] Add custom step tests (image-only, image+command+args, env vars passed through) in sdk/pipeline/executor_test.go
- [X] T035 [US5] Verify existing `dp run` path in cli/cmd/run.go is not broken — add guard that skips pipeline workflow logic when no pipeline.yaml exists
- [X] T036 [US5] Add backward compatibility tests in cli/cmd/run_test.go — verify dp run without pipeline.yaml works unchanged, verify dp run ignores pipeline.yaml

**Checkpoint**: Existing `dp run` unchanged. `dp pipeline run` with custom step produces same result as `dp run`.

---

## Phase 6: User Story 3 — Backfill Historical Data (Priority: P2)

**Goal**: `dp pipeline backfill <name> --from <date> --to <date>` re-executes sync steps with date range injected as DP_BACKFILL_FROM/DP_BACKFILL_TO env vars

**Independent Test**: Create a pipeline with a sync step. Run `dp pipeline backfill my-pipeline --from 2026-01-01 --to 2026-01-31`. Verify sync step executes with date range env vars.

### Implementation for User Story 3

- [X] T037 [P] [US3] Create Backfill() function — validates dates (time.Parse "2006-01-02"), validates from < to, finds sync steps, injects DP_BACKFILL_FROM/DP_BACKFILL_TO, calls executeStep() in sdk/pipeline/backfill.go
- [X] T038 [P] [US3] Create backfill tests (valid range, invalid range from > to, invalid date format, no sync step error, env var injection) in sdk/pipeline/backfill_test.go
- [X] T039 [US3] Create `dp pipeline backfill` command with --from, --to flags in cli/cmd/pipeline_backfill.go
- [X] T040 [US3] Create pipeline_backfill tests (flag registration, required flags, date validation, no-sync-step error message) in cli/cmd/pipeline_backfill_test.go

**Checkpoint**: `dp pipeline backfill` re-executes sync steps for a date range

---

## Phase 7: User Story 4 — Schedule Pipeline Execution (Priority: P2)

**Goal**: `schedule.yaml` defines cron-based execution timing. `dp validate` validates schedule. `dp show` displays schedule alongside pipeline.

**Independent Test**: Create a `schedule.yaml` with valid cron. Run `dp validate`. Verify schedule is accepted. Verify `dp pipeline show` displays schedule info.

### Implementation for User Story 4

- [X] T041 [P] [US4] Create LoadSchedule() in sdk/pipeline/loader.go — reads and parses schedule.yaml
- [X] T042 [P] [US4] Create LoadSchedule tests (valid, missing file returns nil, invalid YAML) in sdk/pipeline/loader_test.go
- [X] T043 [US4] Verify schedule.yaml validation integrated into dp validate output — confirm AggregateValidator (from T017) includes schedule validation results in output
- [X] T044 [US4] Update `dp show` command to display schedule info when schedule.yaml exists alongside pipeline in cli/cmd/show.go

**Checkpoint**: `schedule.yaml` loadable, validated, and displayed in `dp show` output

---

## Phase 8: User Story 6 — List and Inspect Pipelines (Priority: P3)

**Goal**: `dp pipeline show <name>` displays pipeline definition with step details in table or JSON format

**Independent Test**: Create a pipeline.yaml with four steps. Run `dp pipeline show my-pipeline`. Verify step names, types, and asset references displayed. Verify `--output json` works.

### Implementation for User Story 6

- [X] T045 [US6] Create `dp pipeline show` command with --output flag (table/json/yaml) in cli/cmd/pipeline_show.go
- [X] T046 [US6] Create pipeline_show tests (table output with steps, JSON output, no pipeline.yaml message, schedule display) in cli/cmd/pipeline_show_test.go

**Checkpoint**: `dp pipeline show` displays pipeline definition in table and JSON formats

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, e2e tests, and cross-cutting quality improvements

- [X] T047 [P] Create pipeline workflows concept documentation in docs/concepts/pipelines.md
- [X] T048 [P] Update CLI reference documentation with pipeline commands (create, run, backfill, show) in docs/reference/cli.md
- [X] T049 [P] Update manifest schema documentation with pipeline.yaml and schedule.yaml schemas in docs/reference/manifest-schema.md
- [X] T050 Add Pipelines nav entry to mkdocs.yml under Concepts section
- [X] T051 Update docs/concepts/index.md with Pipeline Workflows card and learning path
- [X] T052 Create end-to-end workflow test (create → validate → run → backfill → show) in tests/e2e/pipeline_workflow_test.go
- [X] T053 Verify quickstart.md commands align with implementation — run through all commands from specs/012-pipeline-workflows/quickstart.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 contracts — BLOCKS all user stories
- **US1 Scaffold (Phase 3)**: Depends on Phase 2 (loader + validator)
- **US2 Execute (Phase 4)**: Depends on Phase 2 (loader) + Phase 3 (scaffolder for test fixtures)
- **US5 Backward Compat (Phase 5)**: Depends on Phase 4 (executor with custom step support)
- **US3 Backfill (Phase 6)**: Depends on Phase 4 (executor)
- **US4 Schedule (Phase 7)**: Depends on Phase 2 (loader + validator) — can run in parallel with US2/US3
- **US6 Inspect (Phase 8)**: Depends on Phase 2 (loader) — can run in parallel with US2/US3
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

```text
Phase 1 (Setup)
    │
    ▼
Phase 2 (Foundational) ──── BLOCKS ALL ────┐
    │                                       │
    ▼                                       ▼
Phase 3 (US1: Scaffold) ──P1       Phase 7 (US4: Schedule) ──P2
    │                               Phase 8 (US6: Inspect) ──P3
    ▼                                       │
Phase 4 (US2: Execute) ──P1                 │
    │                                       │
    ├──► Phase 5 (US5: Backward) ──P1       │
    │                                       │
    ├──► Phase 6 (US3: Backfill) ──P2       │
    │                                       │
    ▼                                       ▼
Phase 9 (Polish) ◄──────────────────────────┘
```

### Within Each User Story

- Models/types before services/logic
- SDK implementation before CLI commands
- Core implementation before integration
- Tests alongside each implementation file

### Parallel Opportunities

- **Phase 1**: T003, T004, T005, T006, T007 can run in parallel (different files)
- **Phase 2**: T010+T011, T012+T013 can run in parallel (different validator files). T010 and T012 can run in parallel with each other.
- **Phase 3**: T018 and T019 can run in parallel (templates vs renderer)
- **Phase 4**: T026+T027 can run in parallel with each other (PrefixWriter is independent)
- **Phase 6**: T037+T038 can run in parallel with T039+T040 (SDK vs CLI)
- **Phase 7**: T041+T042 can run in parallel (loader is independent of validator)
- **Phase 9**: T047, T048, T049 can all run in parallel (different doc files)

---

## Parallel Example: User Story 1

```bash
# After Phase 2 is complete, launch these in parallel:
Task T018: "Create embedded pipeline templates in sdk/pipeline/templates/"
Task T019: "Create template renderer in sdk/pipeline/templates.go"

# Then sequentially:
Task T020: "Create template renderer tests"
Task T021: "Create pipeline scaffolder in sdk/pipeline/scaffolder.go"
Task T022: "Create scaffolder tests"
Task T023: "Create parent dp pipeline cobra command"
Task T024: "Create dp pipeline create command"
Task T025: "Create pipeline_create tests"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 + 5)

1. Complete Phase 1: Setup (contracts + schemas)
2. Complete Phase 2: Foundational (loader + validators)
3. Complete Phase 3: User Story 1 — Scaffold
4. Complete Phase 4: User Story 2 — Execute
5. Complete Phase 5: User Story 5 — Backward Compatibility
6. **STOP and VALIDATE**: Run all tests. Verify `dp pipeline create` → `dp pipeline run` → `dp run` all work.

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 (Scaffold) → `dp pipeline create` works → **First demo possible**
3. Add US2 (Execute) + US5 (Backward Compat) → `dp pipeline run` + `dp run` → **MVP complete!**
4. Add US3 (Backfill) → `dp pipeline backfill` works
5. Add US4 (Schedule) → `schedule.yaml` validated and displayed
6. Add US6 (Inspect) → `dp pipeline show` works
7. Polish → docs, e2e tests, quickstart validation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1 (Scaffold) → US2 (Execute) → US5 (Backward Compat)
   - Developer B: US4 (Schedule) + US6 (Inspect) — independent of execution path
3. After US2 is done:
   - Developer A: US3 (Backfill) — depends on executor
   - Developer B: continues with Polish

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Tests are written alongside implementation (not TDD — per project convention)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- All tests use Go stdlib `testing` with table-driven patterns — no external assertion libraries
- Error codes E080–E091 (pipeline workflow) and E100–E103 (schedule) are pre-allocated in data-model.md
