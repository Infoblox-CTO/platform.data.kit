# Tasks: CloudQuery Plugin Package Type

**Input**: Design documents from `/specs/008-cloudquery-plugins/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Tests**: Included — the constitution mandates unit tests for all packages, and the spec explicitly requires generated scaffolds to pass tests out of the box.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization — add the CloudQuery package type to the contracts layer so all downstream code compiles.

- [X] T001 Add `PackageTypeCloudQuery` constant to `contracts/types.go`
- [X] T002 Create `CloudQueryRole` type with `IsValid()`, `IsSupported()` methods and `CloudQuerySpec` struct with `Default()` method in `contracts/cloudquery.go`
- [X] T003 Add `CloudQuery *CloudQuerySpec` field to `DataPackageSpec` struct in `contracts/datapackage.go`
- [X] T004 [P] Write unit tests for `CloudQueryRole.IsValid()`, `CloudQueryRole.IsSupported()`, and `CloudQuerySpec.Default()` in `contracts/cloudquery_test.go`
- [X] T005 [P] Update JSON Schema: add `"cloudquery"` to `spec.type` enum, add `cloudquerySpec` definition, add `if-then` conditional requiring `cloudquery` and `runtime` when `type=cloudquery` in `contracts/schemas/dp-manifest.schema.json`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Validation framework and template engine extensions that MUST be complete before any user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Validation (dp lint support)

- [X] T006 Add `PackageTypeCloudQuery` to `validTypes` slice in `sdk/validate/datapackage.go`
- [X] T007 Skip outputs-required check (E003) for `PackageTypeCloudQuery` in `sdk/validate/datapackage.go`
- [X] T008 Add runtime-required check for `PackageTypeCloudQuery` (reuse existing E040/E041) in `sdk/validate/datapackage.go`
- [X] T009 Create `CloudQueryValidator` with validation rules E060 (cloudquery section required), E061 (role required and valid), W060 (destination not supported), E062 (grpcPort range), E063 (concurrency > 0) in `sdk/validate/cloudquery.go`
- [X] T010 Wire `CloudQueryValidator` into `AggregateValidator` — call when `spec.Type == PackageTypeCloudQuery` in `sdk/validate/aggregate.go`
- [X] T011 [P] Write unit tests for all CloudQuery validation rules (E060, E061, W060, E062, E063) and the skipped outputs check in `sdk/validate/cloudquery_test.go`

### Template Engine (directory-based scaffolding)

- [X] T012 Add `Type`, `Role`, `GRPCPort`, `Concurrency` fields to `PackageConfig` struct in `cli/internal/templates/renderer.go`
- [X] T013 Add `//go:embed cloudquery/**/*.tmpl` embedded FS variable in `cli/internal/templates/renderer.go`
- [X] T014 Implement `RenderDirectory(outputDir, templateSubDir string, config PackageConfig) error` method on `Renderer` — walks embedded template tree, creates subdirectories, renders each `.tmpl` file stripping the `.tmpl` suffix in `cli/internal/templates/renderer.go`
- [X] T015 [P] Write unit tests for `RenderDirectory` verifying directory creation, file rendering, and template variable substitution in `cli/internal/templates/renderer_test.go`

**Checkpoint**: Foundation ready — validation accepts `type: cloudquery`, template engine can scaffold directory trees. User story implementation can now begin.

---

## Phase 3: User Story 1 — Scaffold Python CloudQuery Plugin (Priority: P1) 🎯 MVP

**Goal**: `dp init --type cloudquery --lang python` scaffolds a complete, immediately-runnable Python CloudQuery source plugin.

**Independent Test**: Run `dp init --type cloudquery --lang python --name my-source --namespace acme` in a temp directory. Verify all expected files are created with correct content. Verify generated tests pass with `pytest`. Verify gRPC server starts with `python main.py`.

### Python Plugin Templates

- [X] T016 [P] [US1] Create CloudQuery Python `dp.yaml.tmpl` with `type: cloudquery`, `spec.cloudquery` section (role, tables, grpcPort, concurrency), and `runtime.image` in `cli/internal/templates/cloudquery/python/dp.yaml.tmpl`
- [X] T017 [P] [US1] Create `main.py.tmpl` with `serve.PluginCommand` entry point in `cli/internal/templates/cloudquery/python/main.py.tmpl`
- [X] T018 [P] [US1] Create `plugin/plugin.py.tmpl` with Plugin subclass implementing `init`, `get_tables`, `sync`, `close` in `cli/internal/templates/cloudquery/python/plugin/plugin.py.tmpl`
- [X] T019 [P] [US1] Create `plugin/client.py.tmpl` with Client class and `id()` method in `cli/internal/templates/cloudquery/python/plugin/client.py.tmpl`
- [X] T020 [P] [US1] Create `plugin/spec.py.tmpl` with Spec dataclass (concurrency field) in `cli/internal/templates/cloudquery/python/plugin/spec.py.tmpl`
- [X] T021 [P] [US1] Create `plugin/tables/example_resource.py.tmpl` with ExampleTable (Table subclass, columns with pyarrow types) and ExampleResolver (TableResolver subclass, resolve generator) in `cli/internal/templates/cloudquery/python/plugin/tables/example_resource.py.tmpl`
- [X] T022 [P] [US1] Create `plugin/__init__.py.tmpl` and `plugin/tables/__init__.py.tmpl` as empty Python package init files in `cli/internal/templates/cloudquery/python/plugin/__init__.py.tmpl` and `cli/internal/templates/cloudquery/python/plugin/tables/__init__.py.tmpl`
- [X] T023 [P] [US1] Create `tests/test_example_resource.py.tmpl` with passing unit tests for ExampleTable columns and ExampleResolver output in `cli/internal/templates/cloudquery/python/tests/test_example_resource.py.tmpl`
- [X] T024 [P] [US1] Create `Dockerfile.tmpl` (python:3.13-slim base, pip install, EXPOSE 7777, ENTRYPOINT serve) in `cli/internal/templates/cloudquery/python/Dockerfile.tmpl`
- [X] T025 [P] [US1] Create `pyproject.toml.tmpl` and `requirements.txt.tmpl` pinning `cloudquery-plugin-sdk>=0.1.52` and `pyarrow>=23.0.0` in `cli/internal/templates/cloudquery/python/pyproject.toml.tmpl` and `cli/internal/templates/cloudquery/python/requirements.txt.tmpl`

### CLI Init Wiring for Python

- [X] T026 [US1] Add `"cloudquery"` to `isValidPackageType()` switch in `cli/cmd/init.go`
- [X] T027 [US1] Default language to `python` when `--type cloudquery` and no `--lang` specified in `cli/cmd/init.go`
- [X] T028 [US1] Add `--role` flag (default `"source"`) to init command in `cli/cmd/init.go`
- [X] T029 [US1] Reject `--role destination` with "Destination plugins are not yet supported" message and exit in `cli/cmd/init.go`
- [X] T030 [US1] Skip `--mode` validation for cloudquery type (mode is pipeline-specific) in `cli/cmd/init.go`
- [X] T031 [US1] Add cloudquery scaffolding path: populate `PackageConfig` with Type/Role/GRPCPort/Concurrency, call `renderer.RenderDirectory(outputDir, "cloudquery/python", config)` in `cli/cmd/init.go`
- [X] T032 [US1] Write unit tests for `dp init --type cloudquery --lang python` verifying all files are created with expected content in `cli/cmd/init_test.go`

**Checkpoint**: `dp init --type cloudquery --lang python` produces a complete project. Generated `pytest` tests pass. gRPC server starts. `dp lint` validates the generated `dp.yaml`.

---

## Phase 4: User Story 2 — Scaffold Go CloudQuery Plugin (Priority: P2)

**Goal**: `dp init --type cloudquery --lang go` scaffolds a complete, immediately-compilable Go CloudQuery source plugin.

**Independent Test**: Run `dp init --type cloudquery --lang go --name my-source --namespace acme` in a temp directory. Verify all expected files are created. Verify `go build ./...` and `go test ./...` pass.

### Go Plugin Templates

- [X] T033 [P] [US2] Create CloudQuery Go `dp.yaml.tmpl` with `type: cloudquery`, `spec.cloudquery` section, and `runtime.image` in `cli/internal/templates/cloudquery/go/dp.yaml.tmpl`
- [X] T034 [P] [US2] Create `main.go.tmpl` with `serve.Plugin(p).Serve(ctx)` entry point in `cli/internal/templates/cloudquery/go/main.go.tmpl`
- [X] T035 [P] [US2] Create `resources/plugin/plugin.go.tmpl` with Plugin constructor (`plugin.NewPlugin` with name, version, Configure func) in `cli/internal/templates/cloudquery/go/resources/plugin/plugin.go.tmpl`
- [X] T036 [P] [US2] Create `internal/client/client.go.tmpl` with Client struct implementing `plugin.Client` (Tables, Sync, Close) and Configure function in `cli/internal/templates/cloudquery/go/internal/client/client.go.tmpl`
- [X] T037 [P] [US2] Create `internal/client/spec.go.tmpl` with Spec struct, `SetDefaults()`, and `Validate()` methods in `cli/internal/templates/cloudquery/go/internal/client/spec.go.tmpl`
- [X] T038 [P] [US2] Create `internal/tables/example_resource.go.tmpl` with `ExampleResourceTable()` returning `*schema.Table` and `fetchExampleResource` resolver in `cli/internal/templates/cloudquery/go/internal/tables/example_resource.go.tmpl`
- [X] T039 [P] [US2] Create `internal/tables/example_resource_test.go.tmpl` with passing table test in `cli/internal/templates/cloudquery/go/internal/tables/example_resource_test.go.tmpl`
- [X] T040 [P] [US2] Create multi-stage `Dockerfile.tmpl` (golang:1.25-alpine builder, alpine:3.21 runtime, CGO_ENABLED=0, EXPOSE 7777) in `cli/internal/templates/cloudquery/go/Dockerfile.tmpl`
- [X] T041 [P] [US2] Create `go.mod.tmpl` pinning `github.com/cloudquery/plugin-sdk/v4`, `github.com/apache/arrow-go/v18`, and `github.com/rs/zerolog` in `cli/internal/templates/cloudquery/go/go.mod.tmpl`

### CLI Init Wiring for Go

- [X] T042 [US2] Add Go cloudquery scaffolding path: call `renderer.RenderDirectory(outputDir, "cloudquery/go", config)` when `--lang go` in `cli/cmd/init.go`
- [X] T043 [US2] Write unit tests for `dp init --type cloudquery --lang go` verifying all files are created with expected content in `cli/cmd/init_test.go`

**Checkpoint**: `dp init --type cloudquery --lang go` produces a complete project. `go build` and `go test` pass. `dp lint` validates the generated `dp.yaml`.

---

## Phase 5: User Story 3 — Run CloudQuery Plugin Locally (Priority: P1) 🎯 MVP

**Goal**: `dp run` detects `type: cloudquery` from `dp.yaml` and orchestrates: build container → start gRPC server → generate sync config → run `cloudquery sync` → display summary.

**Independent Test**: In a scaffolded CloudQuery project with `dp dev` running, run `dp run`. Verify the sync completes and summary shows tables synced.

### Implementation

- [X] T044 [US3] Add `cloudquery` binary detection via `exec.LookPath("cloudquery")` with clear error message and install instructions in `cli/cmd/run.go`
- [X] T045 [US3] Add type detection after parsing `dp.yaml`: if `spec.Type == contracts.PackageTypeCloudQuery`, call `runCloudQuery()` instead of pipeline run path in `cli/cmd/run.go`
- [X] T046 [US3] Implement container build and start: build plugin Docker image, start container in detached mode with port mapping (`-p grpcPort:grpcPort`) in `cli/cmd/run.go`
- [X] T047 [US3] Implement gRPC health check: TCP connect to `localhost:{grpcPort}` with 30-second timeout and retry loop in `cli/cmd/run.go`
- [X] T048 [US3] Implement sync config generation: create temp YAML file with source (registry: grpc, path: localhost:port) and destination (PostgreSQL from dp dev) per `contracts/sync-config-contract.md` in `cli/cmd/run.go`
- [X] T049 [US3] Implement `cloudquery sync` execution: run `cloudquery sync <temp-config>` as subprocess, stream output, parse sync summary (tables, rows, errors) in `cli/cmd/run.go`
- [X] T050 [US3] Implement container cleanup: stop and remove plugin container on completion, error, and OS interrupt (signal handling) in `cli/cmd/run.go`
- [X] T051 [US3] Display formatted sync summary: tables synced count, total rows fetched, errors per table in `cli/cmd/run.go`
- [X] T052 [US3] Write unit tests for cloudquery run path: binary detection, type routing, sync config generation, error handling in `cli/cmd/run_test.go`

**Checkpoint**: `dp run` in a scaffolded CloudQuery project completes a full sync cycle. Clear error when `cloudquery` CLI is missing.

---

## Phase 6: User Story 5 — Validate CloudQuery Manifest Fields (Priority: P2)

**Goal**: `dp lint` validates all CloudQuery-specific `dp.yaml` fields with actionable error messages.

**Independent Test**: Create `dp.yaml` files with missing/invalid CloudQuery fields. Run `dp lint`. Verify specific error codes and messages.

> **Note**: Validation infrastructure was built in Phase 2 (Foundational). This phase adds integration tests to verify the full `dp lint` command experience for CloudQuery packages.

- [X] T053 [US5] Write integration tests for `dp lint` with CloudQuery manifests: valid manifest passes, missing `spec.cloudquery` fails (E060), missing role fails (E061), `role: destination` warns (W060), invalid grpcPort fails (E062), invalid concurrency fails (E063) in `cli/cmd/lint_test.go`

**Checkpoint**: `dp lint` catches 100% of invalid CloudQuery manifest fields with actionable messages.

---

## Phase 7: User Story 4 — Test CloudQuery Plugins (Priority: P2)

**Goal**: `dp test` runs unit tests (pytest/go test); `dp test --integration` runs full end-to-end sync.

**Independent Test**: In a scaffolded project, run `dp test` and verify unit tests execute. Run `dp test --integration` with `dp dev` running and verify sync completes.

- [X] T054 [US4] Add cloudquery type detection in test command: read `dp.yaml`, check `spec.Type == PackageTypeCloudQuery` in `cli/cmd/test.go`
- [X] T055 [US4] Route cloudquery unit tests to `pytest` (Python) or `go test ./...` (Go) based on language detection from project files in `cli/cmd/test.go`
- [X] T056 [US4] Implement `--integration` flag for cloudquery type: reuse container build + gRPC start + sync config + `cloudquery sync` from `runCloudQuery()`, add per-table result reporting in `cli/cmd/test.go`
- [X] T057 [US4] Write unit tests for cloudquery test command: type detection, language routing, integration flag behavior in `cli/cmd/test_test.go`

**Checkpoint**: `dp test` runs appropriate unit tests. `dp test --integration` performs full sync with per-table reporting.

---

## Phase 8: User Story 6 — Build, Publish, Promote (Priority: P3)

**Goal**: `dp build`, `dp publish`, and `dp promote` work for CloudQuery packages identically to pipeline packages.

**Independent Test**: Run `dp build` in a CloudQuery project, then `dp publish`. Verify OCI artifact is created and pushed.

> **Note**: Per research R-007, build/publish/promote commands already work for any package type that passes validation. The only requirement is that validation accepts `type: cloudquery` (done in Phase 2). This phase verifies the integration end-to-end.

- [X] T058 [US6] Write integration test for `dp build` with a CloudQuery package: verify container image build and OCI artifact packaging in `cli/cmd/build_test.go`
- [X] T059 [US6] Write integration test for `dp publish` with a CloudQuery package: verify artifact is pushed to registry in `cli/cmd/publish_test.go`

**Checkpoint**: CloudQuery packages build, publish, and promote using existing OCI pipeline.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates and cross-cutting improvements.

- [X] T060 [P] Add cloudquery package type documentation to `docs/concepts/data-packages.md`
- [X] T061 [P] Add cloudquery quickstart section to `docs/getting-started/quickstart.md`
- [X] T062 [P] Document `--type cloudquery` flag and `--role` flag in `docs/reference/cli.md`
- [X] T063 [P] Document `spec.cloudquery` section (role, tables, grpcPort, concurrency) in `docs/reference/manifest-schema.md`
- [X] T064 Run quickstart.md end-to-end validation: scaffold Python plugin, run tests, run lint, run sync, build artifact

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (contracts must compile) — BLOCKS all user stories
- **US1 Python Scaffold (Phase 3)**: Depends on Phase 2 — MVP entry point
- **US2 Go Scaffold (Phase 4)**: Depends on Phase 2 — can run in parallel with Phase 3
- **US3 dp run (Phase 5)**: Depends on Phase 3 (needs a scaffolded project to run) — MVP execution
- **US5 dp lint integration (Phase 6)**: Depends on Phase 2 — can run in parallel with Phases 3–5
- **US4 dp test (Phase 7)**: Depends on Phase 3 (needs scaffolded project) and Phase 5 (reuses runCloudQuery for --integration)
- **US6 Build/Publish (Phase 8)**: Depends on Phase 2 (validation) — can run in parallel with Phases 3–7
- **Polish (Phase 9)**: Depends on all prior phases

### User Story Dependencies

```
Phase 1 (Setup)
    └──▶ Phase 2 (Foundational) ──BLOCKS──┐
                                           ├──▶ Phase 3 (US1: Python Scaffold) ──┐
                                           ├──▶ Phase 4 (US2: Go Scaffold)       │
                                           ├──▶ Phase 6 (US5: Lint Integration)  │
                                           └──▶ Phase 8 (US6: Build/Publish)     │
                                                                                  │
                                           Phase 3 ──▶ Phase 5 (US3: dp run) ──┐ │
                                                                                │ │
                                           Phase 3 + Phase 5 ──▶ Phase 7 (US4) │ │
                                                                                │ │
                                           All ──▶ Phase 9 (Polish) ◀──────────┘─┘
```

### Within Each User Story

- Templates (marked [P]) can all be written in parallel
- CLI wiring depends on templates being complete
- Tests verify the integrated result

### Parallel Opportunities

**Within Phase 1**: T004 and T005 can run in parallel (different files)
**Within Phase 2**: T011 and T015 can run in parallel (test files for different modules)
**Within Phase 3**: All template tasks T016–T025 can run in parallel (independent files)
**Within Phase 4**: All template tasks T033–T041 can run in parallel (independent files)
**Across Phases**: After Phase 2 completes, Phases 3, 4, 6, and 8 can all start in parallel

---

## Parallel Example: Phase 3 (User Story 1)

```bash
# All Python template files can be written simultaneously:
T016: dp.yaml.tmpl
T017: main.py.tmpl
T018: plugin/plugin.py.tmpl
T019: plugin/client.py.tmpl
T020: plugin/spec.py.tmpl
T021: plugin/tables/example_resource.py.tmpl
T022: plugin/__init__.py.tmpl + plugin/tables/__init__.py.tmpl
T023: tests/test_example_resource.py.tmpl
T024: Dockerfile.tmpl
T025: pyproject.toml.tmpl + requirements.txt.tmpl

# Then CLI wiring (sequential, same file):
T026 → T027 → T028 → T029 → T030 → T031

# Then integration test:
T032
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 3 Only)

1. Complete Phase 1: Setup (contracts)
2. Complete Phase 2: Foundational (validation + template engine)
3. Complete Phase 3: User Story 1 — Python scaffold
4. Complete Phase 5: User Story 3 — dp run
5. **STOP and VALIDATE**: Scaffold a Python plugin, run `dp lint`, run `dp run` with dp dev
6. This is the MVP — a data engineer can create and run a CloudQuery plugin

### Incremental Delivery

1. Setup + Foundational → Types compile, validation works
2. Add US1 (Python scaffold) → `dp init --type cloudquery` works → Demo scaffolding
3. Add US3 (dp run) → `dp run` syncs data → Demo end-to-end (MVP!)
4. Add US2 (Go scaffold) → Go users can scaffold too
5. Add US5 (lint integration) → Full validation coverage
6. Add US4 (dp test) → Test workflow complete
7. Add US6 (build/publish) → Full lifecycle
8. Polish → Docs complete

### Parallel Team Strategy

With multiple developers after Phase 2 completes:

- **Developer A**: Phase 3 (Python templates + init wiring) → Phase 5 (dp run)
- **Developer B**: Phase 4 (Go templates + init wiring) → Phase 7 (dp test)
- **Developer C**: Phase 6 (lint integration tests) → Phase 8 (build/publish tests) → Phase 9 (docs)

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [US*] label maps task to specific user story for traceability
- All template files (T016–T025, T033–T041) can be written in parallel — they are independent files
- CLI wiring tasks within a phase must be sequential (same file: `cli/cmd/init.go`)
- Tests should be written alongside implementation per constitution Article VII
- `dp build`/`dp publish`/`dp promote` need no code changes — they already work for any validated package type
- Commit after each task or logical group
