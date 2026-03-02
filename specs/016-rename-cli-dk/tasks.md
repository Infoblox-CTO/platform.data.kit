# Tasks: Rename CLI from `dp` to `dk` (DataKit) & Add Interactive Banner

**Input**: Design documents from `/specs/016-rename-cli-dk/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Test file updates are included as part of implementation tasks. No separate TDD cycle was requested.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Confirm green baseline before any changes

- [X] T001 Run all existing tests (`go test ./...` in cli/, sdk/, contracts/, platform/controller/) to confirm green baseline

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core changes required before ANY user story work can begin

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Update Makefile — rename all binary targets from `dp` to `dk` (bin/dp → bin/dk, install target, cross-compilation targets dp-linux-amd64 → dk-linux-amd64, etc.)
- [X] T003 Update cli/cmd/root.go — change root command `Use: "dp"` → `Use: "dk"`, Short/Long descriptions from "DP" / "Data Platform" to "DK" / "DataKit", version output format `dp version` → `dk version`
- [X] T004 Update cli/main.go — change package comment from `dp CLI` to `dk CLI`

**Checkpoint**: `make build-cli` produces `bin/dk` and `bin/dk --help` shows "DK (DataKit)" branding

---

## Phase 3: User Story 1 — CLI Rename from `dp` to `dk` (Priority: P1) 🎯 MVP

**Goal**: Every subcommand, flag, and workflow that previously worked under `dp` works identically under `dk`. All K8s API groups, labels, infrastructure identifiers, config paths, and image names are updated.

**Independent Test**: Build `dk`, run `dk version`, `dk --help`, and execute the full workflow (`dk init` → `dk dev up` → `dk lint` → `dk run` → `dk build` → `dk publish` → `dk promote`). All commands succeed with no `dp` references in output.

### CLI Subcommands (cli/cmd/)

- [X] T005 [P] [US1] Update cli/cmd/init.go — rename all `dp` → `dk` in examples, help text, and `dp.yaml` → `dk.yaml` references
- [X] T006 [P] [US1] Update cli/cmd/dev.go and cli/cmd/dev_seed.go — rename `dp` → `dk` in examples, status messages, and help text
- [X] T007 [P] [US1] Update cli/cmd/run.go — rename `dp` → `dk` in examples and help text
- [X] T008 [P] [US1] Update cli/cmd/config.go — rename `dp` → `dk` in config paths (`.dp/` → `.dk/`, `~/.config/dp/` → `~/.config/dk/`), examples, and help text
- [X] T009 [P] [US1] Update cli/cmd/asset.go, cli/cmd/asset_create.go, cli/cmd/asset_list.go, cli/cmd/asset_show.go, cli/cmd/asset_validate.go — rename `dp` → `dk` in examples
- [X] T010 [P] [US1] Update cli/cmd/pipeline.go, cli/cmd/pipeline_create.go, cli/cmd/pipeline_run.go, cli/cmd/pipeline_show.go, cli/cmd/pipeline_backfill.go — rename `dp` → `dk` in examples
- [X] T011 [P] [US1] Update cli/cmd/cell.go and cli/cmd/logs.go — rename `dp` → `dk` in examples and CRD references
- [X] T012 [P] [US1] Update remaining cli/cmd/ source files (build.go, lint.go, promote.go, publish.go, show.go, test.go, rollback.go, status.go) — rename `dp` → `dk` in examples and help text

### CLI Tests (cli/cmd/)

- [X] T013 [P] [US1] Update cli/cmd/ test files (root_test.go, init_test.go, dev_test.go, run_test.go, config_test.go) — rename `dp` → `dk` in expected output strings and test fixtures
- [X] T014 [P] [US1] Update cli/cmd/ test files (asset_create_test.go, asset_list_test.go, asset_show_test.go, asset_validate_test.go, build_test.go) — rename `dp` → `dk` in expected output strings
- [X] T015 [P] [US1] Update cli/cmd/ test files (pipeline_create_test.go, pipeline_run_test.go, pipeline_show_test.go, pipeline_backfill_test.go, cell_test.go, logs_test.go, lint_test.go, promote_test.go, publish_test.go, show_test.go, test_test.go) — rename `dp` → `dk` in expected output strings

### SDK Module (sdk/)

- [X] T016 [P] [US1] Update sdk/localdev/k3d.go — rename cluster name and namespace `dp-local` → `dk-local`
- [X] T017 [P] [US1] Update sdk/localdev/charts/embed.go — rename Helm releases `dp-redpanda` → `dk-redpanda`, `dp-localstack` → `dk-localstack`, `dp-postgres` → `dk-postgres`, `dp-marquez` → `dk-marquez`, `dp-marquez-web` → `dk-marquez-web`
- [X] T018 [P] [US1] Update sdk/localdev/config.go — rename config paths `.dp/config.yaml` → `.dk/config.yaml`, `~/.config/dp/` → `~/.config/dk/`
- [X] T019 [P] [US1] Update sdk/runner/docker.go — rename image prefix `dp/` → `dk/`, comment `# DP Pipeline Image` → `# DK Pipeline Image`
- [X] T020 [P] [US1] Update sdk/pipeline/executor.go — rename images `dp-sync:latest` → `dk-sync:latest`, `dp-transform:latest` → `dk-transform:latest`, `dp-test:latest` → `dk-test:latest`
- [X] T021 [P] [US1] Update sdk/lineage/heartbeat.go — rename producer `dp-runner` → `dk-runner`
- [X] T022 [P] [US1] Update sdk/registry/bundler.go — rename builder prefix `dp/` → `dk/`
- [X] T023 [P] [US1] Update sdk/promotion/pr.go — rename labels `dp.io/package` → `datakit.infoblox.dev/package`, `dp.io/environment` → `datakit.infoblox.dev/environment`
- [X] T024 [P] [US1] Update sdk/promotion/kustomize.go — rename labels `dp.io/package` → `datakit.infoblox.dev/package`

### Contracts Module (contracts/)

- [X] T025 [P] [US1] Update contracts/connector.go — rename labels `dp.infoblox.com/provider` → `datakit.infoblox.dev/provider`, `dp.infoblox.com/channel` → `datakit.infoblox.dev/channel`
- [X] T026 [P] [US1] Update contracts/ test files (connector_test.go and any others referencing dp labels) — rename label domain references

### Controller Module (platform/controller/)

- [X] T027 [P] [US1] Update platform/controller/api/v1alpha1/groupversion_info.go — rename API group `dp.io` → `datakit.infoblox.dev` in `+groupName` marker and `Group` constant
- [X] T028 [P] [US1] Update platform/controller/internal/controller/job.go — rename labels `dp.io/package` → `datakit.infoblox.dev/package`, `dp.io/mode` → `datakit.infoblox.dev/mode`, controller name `dp-controller` → `dk-controller`
- [X] T029 [P] [US1] Update platform/controller/internal/controller/deployment.go — rename labels `dp.io/*` → `datakit.infoblox.dev/*`
- [X] T030 [P] [US1] Update platform/controller/cmd/main.go — rename leader election ID `dp-controller.dp.io` → `dk-controller.datakit.infoblox.dev`
- [X] T031 [P] [US1] Update platform/controller/config/deployment.yaml — rename RBAC API group `dp.io` → `datakit.infoblox.dev`

### GitOps Manifests (gitops/)

- [X] T032 [P] [US1] Update gitops/base/crds/packagedeployment.yaml — rename CRD group `dp.io` → `datakit.infoblox.dev`
- [X] T033 [P] [US1] Update gitops/base/crds/cell.yaml — rename CRD group `dp.io` → `datakit.infoblox.dev`
- [X] T034 [P] [US1] Update gitops/base/crds/store.yaml — rename CRD group `dp.io` → `datakit.infoblox.dev`
- [X] T035 [P] [US1] Update gitops/base/kustomization.yaml — rename labels `dp.io/managed-by` → `datakit.infoblox.dev/managed-by`
- [X] T036 [P] [US1] Update gitops/argocd/applicationset.yaml — rename labels `dp.io/environment` → `datakit.infoblox.dev/environment`, API group `dp.io` → `datakit.infoblox.dev`

### Manifest Filename (cross-cutting)

- [X] T037 [US1] Rename all references to manifest filename `dp.yaml` → `dk.yaml` across sdk/, contracts/, and remaining Go source files
- [X] T038 [US1] Rename quickstart-demo/dp.yaml file to quickstart-demo/dk.yaml and update quickstart-demo/main.go and quickstart-demo/cmd/ references

### Verification

- [X] T039 [US1] Run all unit tests across all modules (`go test ./...` in cli/, sdk/, contracts/, platform/controller/) — all must pass

**Checkpoint**: All code references renamed. `dk` builds and all subcommands work identically to old `dp`. No `dp` references remain in Go source or K8s manifests.

---

## Phase 4: User Story 2 — SLICK ASCII Banner in Interactive Prompts (Priority: P2)

**Goal**: When a developer begins an interactive session (e.g., `dk init` without all required flags), a styled ASCII art "DataKit" banner is displayed before the first prompt.

**Independent Test**: Run `dk init` without arguments in an interactive terminal — banner appears. Run `dk init my-project --runtime cloudquery` — no banner. Pipe input via `echo "" | dk init` — no banner.

### Implementation

- [X] T040 [US2] Add charmbracelet/lipgloss as direct dependency in cli/go.mod (`go get github.com/charmbracelet/lipgloss`)
- [X] T041 [US2] Create cli/cmd/banner.go — implement `ShowBanner()` function with ASCII art "DataKit" branding, lipgloss styling (blue/cyan ANSI colors), terminal width detection (skip if < 40 cols), color fallback for non-color terminals, and plain-text fallback
- [X] T042 [US2] Integrate banner into cli/cmd/init.go — call `ShowBanner()` before interactive huh form when `prompt.IsInteractive()` returns true and no required flags are provided
- [X] T043 [P] [US2] Create cli/cmd/banner_test.go — test banner render output, TTY suppression logic, narrow terminal fallback (< 40 cols), and plain-text mode

**Checkpoint**: Interactive `dk init` shows a styled "DataKit" banner. Non-interactive and piped scenarios show no banner.

---

## Phase 5: User Story 3 — Update Documentation and Build Artifacts (Priority: P3)

**Goal**: All project documentation, build scripts, demos, and references reflect the `dk` name. Build outputs produce `dk` binaries.

**Independent Test**: Search the repository for `dp` references in user-facing text — zero matches. `make build` produces `bin/dk`. All docs show `dk` commands.

### Build & CI

- [X] T044 [P] [US3] Update .github/workflows/ci.yaml — rename artifact name from `cdpp` to `dk`, update build commands and artifact paths

### Root Documentation

- [X] T045 [P] [US3] Update README.md — rename all `dp` → `dk` CLI references
- [X] T046 [P] [US3] Update CONTRIBUTING.md — rename `dp` → `dk` in any CLI references
- [X] T047 [P] [US3] Update RELEASING.md — rename `dp` → `dk` in build/release instructions

### Docs Site (docs/)

- [X] T048 [P] [US3] Update docs/index.md, docs/architecture.md, docs/contributing.md, docs/gap-analysis.md, docs/target-state.md, docs/testing.md — rename `dp` → `dk`
- [X] T049 [P] [US3] Update docs/concepts/ (overview.md, data-packages.md, manifests.md, pipelines.md, pipeline-modes.md, cells.md, assets.md, environments.md, lineage.md, governance.md, index.md) — rename `dp` → `dk` and `dp.yaml` → `dk.yaml`
- [X] T050 [P] [US3] Update docs/getting-started/ (index.md, prerequisites.md, installation.md, quickstart.md) — rename `dp` → `dk`
- [X] T051 [P] [US3] Update docs/reference/ (cli.md, configuration.md, manifest-schema.md, index.md) — rename `dp` → `dk`, update config paths `.dp/` → `.dk/`, `dp.yaml` → `dk.yaml`
- [X] T052 [P] [US3] Update docs/tutorials/ (local-development.md, k3d-local-dev.md, cloudquery-go.md, cloudquery-python.md, kafka-to-s3.md, streaming-pipeline.md, deploying-to-cells.md, promoting-packages.md, index.md) — rename `dp` → `dk`
- [X] T053 [P] [US3] Update docs/troubleshooting/ (common-issues.md, faq.md, index.md) — rename `dp` → `dk`

### Demos & Examples

- [X] T054 [P] [US3] Update demos/README.md, demos/run_demo.sh, and demos/ subdirectory scripts — rename `dp` → `dk`
- [X] T055 [P] [US3] Update examples/kafka-s3-pipeline/ and examples/reactive-pipeline/ — rename `dp` → `dk` and `dp.yaml` → `dk.yaml`

### Verification

- [X] T056 [US3] Run `grep -rn '\bdp\b' docs/ demos/ examples/ README.md CONTRIBUTING.md RELEASING.md` to verify zero remaining `dp` CLI references in documentation

**Checkpoint**: All documentation, demos, examples, and CI artifacts reference `dk`. No `dp` remnants in user-facing text.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation across all user stories

- [X] T057 Run quickstart.md verification — execute all 10 steps from specs/016-rename-cli-dk/quickstart.md
- [X] T058 Run final remnant scan — `grep -rn '"dp ' cli/ sdk/ contracts/ platform/` and `grep -rn 'dp\.io' platform/ gitops/ sdk/promotion/` must return zero matches
- [X] T059 Run full test suite — `go test ./...` in cli/, sdk/, contracts/, platform/controller/ — all tests pass
- [X] T060 Verify `dk version` outputs correct version string and `dk --help` contains no `dp` references

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2) — can start after T002–T004 complete
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2) — can run in parallel with US1 (different files)
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2) — can run in parallel with US1/US2 (different files)
- **Polish (Phase 6)**: Depends on ALL user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: No dependencies on other stories. Core rename — most tasks are [P] since they touch different files.
- **User Story 2 (P2)**: Logically depends on US1 for consistent branding in banner, but technically can be implemented in parallel (banner.go is a new file). T042 (integration into init.go) should run after T005 (init.go rename).
- **User Story 3 (P3)**: Independent of US1/US2 — documentation and CI are separate files. Can run fully in parallel.

### Within User Story 1

- CLI subcommands (T005–T012) — all [P], different files
- CLI tests (T013–T015) — all [P], different files; should run after corresponding source file updates
- SDK tasks (T016–T024) — all [P], different files
- Contracts (T025–T026) — [P], different module
- Controller (T027–T031) — all [P], different files
- GitOps (T032–T036) — all [P], different files
- Manifest filename (T037–T038) — sequential, touches files across modules (run after module-specific tasks)
- Verification (T039) — run last within US1

### Parallel Opportunities

Within Phase 3 (US1), these groups can all run in parallel:
```
Group A (CLI):    T005, T006, T007, T008, T009, T010, T011, T012
Group B (Tests):  T013, T014, T015
Group C (SDK):    T016, T017, T018, T019, T020, T021, T022, T023, T024
Group D (Infra):  T025, T026, T027, T028, T029, T030, T031, T032, T033, T034, T035, T036
```

Within Phase 5 (US3), all documentation tasks (T044–T055) can run in parallel.

---

## Parallel Example: User Story 1

```bash
# Launch all CLI subcommand renames together:
Task: T005 "Update cli/cmd/init.go — dp → dk"
Task: T006 "Update cli/cmd/dev.go and dev_seed.go — dp → dk"
Task: T007 "Update cli/cmd/run.go — dp → dk"
Task: T008 "Update cli/cmd/config.go — dp → dk"

# Launch all SDK renames together:
Task: T016 "Update sdk/localdev/k3d.go — dp-local → dk-local"
Task: T017 "Update sdk/localdev/charts/embed.go — dp-* → dk-*"
Task: T019 "Update sdk/runner/docker.go — dp/ → dk/"
Task: T020 "Update sdk/pipeline/executor.go — dp-sync → dk-sync"

# Launch all GitOps renames together:
Task: T032 "Update gitops/base/crds/packagedeployment.yaml"
Task: T033 "Update gitops/base/crds/cell.yaml"
Task: T034 "Update gitops/base/crds/store.yaml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup — confirm green baseline
2. Complete Phase 2: Foundational — Makefile + root command = `dk` builds
3. Complete Phase 3: User Story 1 — all code references renamed
4. **STOP and VALIDATE**: Run `dk version`, `dk --help`, full test suite
5. `dk` is fully functional — deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → `dk` binary builds
2. Add User Story 1 → All code renamed → Test independently → **MVP complete**
3. Add User Story 2 → Interactive banner works → Test independently
4. Add User Story 3 → All docs updated → Test independently
5. Polish → Final validation against quickstart.md

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (3 tasks)
2. Once Foundational is done:
   - Developer A: User Story 1 (code rename — largest scope)
   - Developer B: User Story 2 (banner — new file, independent) + User Story 3 (docs)
3. Stories complete and validate independently

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks in same phase
- [Story] label maps task to specific user story for traceability
- No backward compatibility — this is a clean break per constitution v3.0.0
- Manifest filename `dp.yaml` → `dk.yaml` affects ~30 Go files; handled by T037 after module-specific tasks
- Total estimated scope: 400+ individual string replacements across 60 tasks
- Commit after each logical group of tasks to maintain reviewable history
