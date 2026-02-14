# Feature Specification: Pipeline Workflows

**Feature Branch**: `012-pipeline-workflows`  
**Created**: 2026-02-14  
**Status**: Draft  
**Input**: User description: "Refactor pipelines to wire assets into multi-step workflows. `dp pipeline create <name> --template sync-transform-test` scaffolds a pipeline.yaml that chains steps: sync (source asset → sink asset), transform (model-engine asset running dbt), test (dbt test), and publish/notify. Each step references assets by name. `dp pipeline run <name> --env dev` executes the pipeline locally, running steps sequentially. `dp pipeline backfill <name> --from <date> --to <date>` re-executes a sync step over a historical date range. The existing single-container pipeline mode is preserved as a 'custom' step type for backward compatibility. Pipelines can also define a schedule.yaml for cron-based execution."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Scaffold a Multi-Step Pipeline (Priority: P1) 🎯 MVP

A data engineer wants to create a pipeline that syncs data from a source, transforms it, tests the output, and publishes the result. Instead of manually composing YAML, they run a single scaffolding command that produces a ready-to-customize pipeline definition with pre-wired step references.

**Why this priority**: Without a pipeline definition file and scaffolding, no other feature (run, backfill, schedule) can function. This story creates the core data model and developer entry point.

**Independent Test**: Run `dp pipeline create security-pipeline --template sync-transform-test` in a project that already has assets `aws-security` (source), `raw-output` (sink), and `dbt-transform` (model-engine). Verify `pipeline.yaml` is created with four steps (sync, transform, test, publish) referencing assets by name. Verify `dp validate` accepts the resulting file.

**Acceptance Scenarios**:

1. **Given** a project with at least one source asset and one sink asset, **When** the engineer runs `dp pipeline create my-pipeline --template sync-transform-test`, **Then** a `pipeline.yaml` is created in the project root with steps that reference those assets by name.
2. **Given** no assets exist in the project, **When** the engineer runs `dp pipeline create my-pipeline --template sync-transform-test`, **Then** the command succeeds but generates placeholder step references with `# TODO` comments indicating which assets to add.
3. **Given** a `pipeline.yaml` already exists, **When** the engineer runs `dp pipeline create my-pipeline --template sync-transform-test`, **Then** the command fails with a clear message and suggests using `--force` to overwrite.
4. **Given** any project, **When** the engineer runs `dp pipeline create my-pipeline --template custom`, **Then** a `pipeline.yaml` is created with a single `custom` step that mirrors the current single-container pipeline behavior.
5. **Given** a valid `pipeline.yaml`, **When** the engineer runs `dp validate`, **Then** the validator checks that all step asset references resolve to existing assets, step types are valid, and the step order is syntactically correct.

---

### User Story 2 — Execute a Pipeline Locally (Priority: P1)

A data engineer wants to run their multi-step pipeline end-to-end in a local development environment. Steps execute sequentially: if any step fails, the pipeline halts and reports which step failed and why.

**Why this priority**: Local execution is the core developer loop — without it, the pipeline definition is unverifiable. This story requires US1 (the pipeline definition exists).

**Independent Test**: Create a pipeline with three steps. Run `dp pipeline run my-pipeline --env dev`. Verify steps execute in order, logs show step-by-step progress, and a failure in step 2 stops execution and reports the error.

**Acceptance Scenarios**:

1. **Given** a valid `pipeline.yaml` with three steps and all referenced assets configured, **When** the engineer runs `dp pipeline run my-pipeline --env dev`, **Then** each step executes sequentially and the command exits 0 on success.
2. **Given** a pipeline where step 2 fails, **When** the engineer runs `dp pipeline run my-pipeline --env dev`, **Then** step 1 completes, step 2 fails, step 3 is skipped, and the command exits non-zero with a message identifying step 2 as the failure.
3. **Given** a pipeline, **When** the engineer runs `dp pipeline run my-pipeline --env dev --step sync`, **Then** only the named step executes (not the full pipeline).
4. **Given** no `pipeline.yaml` exists, **When** the engineer runs `dp pipeline run my-pipeline`, **Then** the command fails with a clear error message suggesting `dp pipeline create`.
5. **Given** a valid pipeline, **When** execution is in progress, **Then** the engineer sees real-time logs prefixed with the current step name (e.g., `[sync] Syncing 3 tables...`).

---

### User Story 3 — Backfill Historical Data (Priority: P2)

A data engineer needs to re-sync data for a specific historical date range — for example, after discovering a schema change or data quality issue. They want to re-execute only the sync step of their pipeline for a given date window.

**Why this priority**: Backfill is a common operational need but depends on US1 (pipeline definition) and US2 (execution engine). It's valuable but not required for the MVP workflow.

**Independent Test**: Create a pipeline with a sync step that references a source asset. Run `dp pipeline backfill my-pipeline --from 2026-01-01 --to 2026-01-31`. Verify the sync step executes with the date range passed as parameters.

**Acceptance Scenarios**:

1. **Given** a pipeline with a sync step, **When** the engineer runs `dp pipeline backfill my-pipeline --from 2026-01-01 --to 2026-01-31`, **Then** the sync step executes with the date range injected as environment variables (e.g., `DP_BACKFILL_FROM`, `DP_BACKFILL_TO`).
2. **Given** a pipeline with a sync step, **When** the engineer runs `dp pipeline backfill my-pipeline --from 2026-02-01 --to 2026-01-01` (invalid range), **Then** the command fails with a clear error about the invalid date range.
3. **Given** a pipeline with no sync step, **When** the engineer runs `dp pipeline backfill`, **Then** the command fails with a message explaining that backfill requires a sync step.
4. **Given** a backfill in progress, **When** the engineer views logs, **Then** the logs clearly indicate the backfill date range and show progress.

---

### User Story 4 — Schedule Pipeline Execution (Priority: P2)

A data engineer wants their pipeline to run automatically on a recurring schedule. They define a `schedule.yaml` alongside their `pipeline.yaml` to specify cron timing and timezone.

**Why this priority**: Scheduling is essential for production use but is independent of the local development workflow (US1–US3). The existing `DataPackageSpec.Schedule` field already supports inline scheduling — this story extends it to pipeline-specific scheduling in a dedicated file.

**Independent Test**: Create a `schedule.yaml` with `cron: "0 */6 * * *"` and `timezone: UTC`. Run `dp validate`. Verify the schedule is accepted. Verify `dp show` displays the resolved schedule alongside the pipeline steps.

**Acceptance Scenarios**:

1. **Given** a project with `pipeline.yaml` and `schedule.yaml`, **When** the engineer runs `dp validate`, **Then** the schedule is validated (valid cron expression, valid timezone) and no errors are reported.
2. **Given** a `schedule.yaml` with an invalid cron expression, **When** the engineer runs `dp validate`, **Then** a clear validation error identifies the problem.
3. **Given** a `schedule.yaml` with `suspend: true`, **When** the pipeline is deployed, **Then** the schedule is registered but not active until resumed.
4. **Given** a project with only `dp.yaml` (no `pipeline.yaml`), **When** a `schedule.yaml` exists with the existing inline `spec.schedule` format, **Then** the existing behavior is preserved — backward compatibility is maintained.

---

### User Story 5 — Backward Compatibility with Single-Container Pipelines (Priority: P1)

Existing data packages that use the current single-container pipeline mode (batch or streaming) must continue to work without modification. The `custom` step type preserves this behavior, and existing `dp run` continues to function for packages without a `pipeline.yaml`.

**Why this priority**: Breaking existing workflows is unacceptable. This story must be ensured alongside US1 and US2.

**Independent Test**: Take an existing data package with `dp.yaml` (type: pipeline, runtime: image) and no `pipeline.yaml`. Run `dp run`. Verify it works exactly as before. Then add a `pipeline.yaml` with a single `custom` step. Run `dp pipeline run`. Verify identical behavior.

**Acceptance Scenarios**:

1. **Given** an existing package with `dp.yaml` and no `pipeline.yaml`, **When** the engineer runs `dp run`, **Then** the package executes as a single container exactly as before.
2. **Given** an existing package, **When** the engineer adds a `pipeline.yaml` with a single `custom` step referencing the runtime image, **Then** `dp pipeline run` produces the same result as `dp run`.
3. **Given** a `pipeline.yaml` with mixed step types (sync + custom), **When** the engineer runs `dp pipeline run`, **Then** both step types execute correctly in sequence.
4. **Given** the existing `dp.yaml` `spec.schedule` field, **When** no `schedule.yaml` exists, **Then** the inline schedule continues to function for non-pipeline packages.

---

### User Story 6 — List and Inspect Pipelines (Priority: P3)

A data engineer wants to see what pipelines are defined and inspect their step configuration.

**Why this priority**: Quality-of-life feature that improves visibility but is not required for core functionality.

**Independent Test**: Create a `pipeline.yaml` with four steps. Run `dp pipeline show my-pipeline`. Verify the full pipeline definition is displayed with step names, types, and asset references.

**Acceptance Scenarios**:

1. **Given** a valid `pipeline.yaml`, **When** the engineer runs `dp pipeline show my-pipeline`, **Then** the output displays step names, types, asset references, and configuration.
2. **Given** a valid `pipeline.yaml`, **When** the engineer runs `dp pipeline show my-pipeline --output json`, **Then** JSON output is returned.
3. **Given** no `pipeline.yaml` exists, **When** the engineer runs `dp pipeline show`, **Then** a helpful message is shown suggesting `dp pipeline create`.

---

### Edge Cases

- What happens when a step references an asset that has been deleted from disk since the pipeline was created? Validation fails with an actionable error (E-code) identifying the missing asset.
- What happens when `pipeline.yaml` defines a circular dependency between steps? Sequential step ordering prevents this — steps execute top-to-bottom; no DAG semantics are supported in this version.
- What happens when the user cancels a pipeline run mid-step (Ctrl+C)? The currently running step's container receives SIGTERM for graceful shutdown; subsequent steps are skipped.
- What happens when two steps reference the same asset? This is allowed — an asset can be used in multiple steps (e.g., the same source in two sync steps with different parameters).
- What happens when `pipeline.yaml` is empty (no steps defined)? Validation fails with a clear error that at least one step is required.
- How does backfill interact with non-sync steps? Backfill only targets sync steps. If the pipeline has no sync step, the backfill command fails with a helpful message.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a `pipeline.yaml` manifest file that defines an ordered list of named steps, where each step has a type, asset references, and optional configuration.
- **FR-002**: System MUST support the following step types: `sync` (source-to-sink data movement), `transform` (model-engine execution), `test` (validation/assertion), `publish` (notification/promotion), and `custom` (single-container execution for backward compatibility).
- **FR-003**: Each step MUST reference assets by name, and those names MUST resolve to existing assets in the project's `assets/` directory.
- **FR-004**: `dp pipeline create <name> --template <template>` MUST scaffold a `pipeline.yaml` with pre-configured steps based on the selected template.
- **FR-005**: The `sync-transform-test` template MUST generate four steps: sync (source → sink), transform (model-engine), test (validation), and publish (notification).
- **FR-006**: `dp pipeline run <name> --env <env>` MUST execute pipeline steps sequentially in the order defined in `pipeline.yaml`.
- **FR-007**: If a step fails during execution, subsequent steps MUST be skipped and the command MUST exit with a non-zero code identifying the failed step.
- **FR-008**: `dp pipeline run` MUST support a `--step <name>` flag to execute a single named step in isolation.
- **FR-009**: `dp pipeline backfill <name> --from <date> --to <date>` MUST re-execute the sync step(s) of a pipeline with the specified date range injected as parameters.
- **FR-010**: Backfill dates MUST be validated: `--from` must precede `--to`, both must be valid ISO 8601 date strings.
- **FR-011**: The `custom` step type MUST preserve full backward compatibility with the existing single-container pipeline execution model (batch and streaming modes).
- **FR-012**: System MUST support a `schedule.yaml` file for defining cron-based pipeline execution schedules with fields for cron expression, timezone, and suspend flag.
- **FR-013**: `dp validate` MUST validate `pipeline.yaml` — checking step types, asset references, required fields, and step name uniqueness.
- **FR-014**: `dp validate` MUST validate `schedule.yaml` — checking cron expression syntax and timezone validity.
- **FR-015**: Existing packages without `pipeline.yaml` MUST continue to work with `dp run` exactly as before — no changes to the current single-container execution path.
- **FR-016**: `dp pipeline show <name>` MUST display the pipeline definition with step details, and support `--output json` for machine-readable output.
- **FR-017**: Step execution logs MUST be prefixed with the step name (e.g., `[sync] ...`) so the engineer can identify which step produced each log line.
- **FR-018**: Templates MUST be extensible — the system should support adding new templates without modifying core logic.
- **FR-019**: The `pipeline.yaml` MUST support an optional `description` field and per-step `description` fields for documentation purposes.
- **FR-020**: The `sync` step type MUST require a `source` (source asset name) and `sink` (sink asset name) reference.
- **FR-021**: The `transform` step type MUST require an `asset` reference to a model-engine asset.
- **FR-022**: The `test` step type MUST require an `asset` reference and support a `command` field to specify the test command to run.
- **FR-023**: The `publish` step type MUST support a `notify` configuration for post-pipeline notifications (channels, recipients) and an optional `promote` flag to trigger environment promotion.
- **FR-024**: Each step MUST have a unique name within the pipeline, following DNS-safe naming conventions (lowercase, 3–63 characters, starts with a letter).

### Key Entities

- **Pipeline**: An ordered sequence of named steps that execute against assets. Defined in `pipeline.yaml`. Has a name, optional description, and a list of steps. One pipeline per data package.
- **Step**: A single unit of work within a pipeline. Has a unique name, a type (sync, transform, test, publish, custom), references to assets by name, and optional configuration. Steps execute sequentially top-to-bottom.
- **Step Type**: Categorizes what a step does. Each type has specific required and optional fields:
    - `sync` — requires source asset and sink asset references
    - `transform` — requires model-engine asset reference
    - `test` — requires asset reference and test command
    - `publish` — notification and optional promotion
    - `custom` — single container image (backward-compatible with current `dp run`)
- **Schedule**: Cron-based execution timing for a pipeline. Defined in `schedule.yaml`. Has a cron expression, timezone, and suspend flag.
- **Template**: A pre-defined pipeline structure used by `dp pipeline create --template`. Templates provide a starting point — engineers customize the generated `pipeline.yaml` for their specific needs.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Engineers can scaffold a multi-step pipeline and begin customizing it in under 1 minute (single command + file edit).
- **SC-002**: A 4-step pipeline (sync → transform → test → publish) executes successfully end-to-end with clear step-by-step log output.
- **SC-003**: Failure in any step is reported within 5 seconds of the step failing, with the step name and error clearly identified.
- **SC-004**: Existing single-container packages pass `dp run` and `dp validate` without any modifications after this feature is implemented (zero regressions).
- **SC-005**: Backfill re-executes a sync step for a specified date range and completes without affecting other pipeline steps.
- **SC-006**: All pipeline and schedule validation errors produce actionable messages with specific error codes.
- **SC-007**: Engineers can inspect any pipeline's structure (steps, assets, configuration) with a single command.
- **SC-008**: 100% of commands documented in the quickstart can be executed successfully against a test project.

## Assumptions

- **Pipeline-per-package**: Each data package has at most one `pipeline.yaml`. Multi-pipeline packages are out of scope for this feature.
- **Sequential execution only**: Steps execute top-to-bottom in order. Parallel step execution and DAG-based dependency graphs are out of scope — they may be addressed in a future feature.
- **Local execution first**: `dp pipeline run` targets local development (Docker/k3d). Remote/cluster execution semantics are handled by the existing deployment model and are not changed by this feature.
- **Asset-based steps**: All non-custom step types reference assets. Custom steps reference a container image directly, preserving current behavior.
- **Schedule file is optional**: If no `schedule.yaml` exists, the pipeline has no schedule. The existing inline `spec.schedule` in `dp.yaml` continues to work for non-pipeline packages.
- **Backfill targets sync steps only**: Backfill is meaningful for data synchronization. Transform and test steps don't have a date-range concept.
- **Notification implementation**: The `publish` step's notification mechanism (Slack, email, webhook) is intentionally left flexible — the config schema will define supported channels, but the specific integrations are out of scope for this feature's initial implementation.
- **Template discovery**: Templates are embedded in the CLI binary. A future registry-based template system is out of scope.
- **Date format**: Backfill dates use ISO 8601 (`YYYY-MM-DD`). Time-of-day precision is not supported in the initial implementation.
