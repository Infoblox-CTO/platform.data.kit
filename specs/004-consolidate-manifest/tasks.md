# Tasks: Consolidate DataPackage Manifest

**Input**: Design documents from `/specs/004-consolidate-manifest/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Unit tests included per constitution requirement (Article VII - Quality Gates)

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project structure and foundational types

- [X] T001 Add RuntimeSpec struct with all fields to contracts/datapackage.go
- [X] T002 [P] Add RuntimeSpec unit tests to contracts/datapackage_test.go
- [X] T003 [P] Mark PipelineManifest as deprecated in contracts/pipeline.go with doc comment

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create binding-to-envvar mapper in sdk/runner/envmapper.go
- [X] T005 [P] Add unit tests for envmapper in sdk/runner/envmapper_test.go
- [X] T006 Create override merge logic in sdk/manifest/merge.go
- [X] T007 [P] Add unit tests for merge logic in sdk/manifest/merge_test.go
- [X] T008 Update sdk/manifest/datapackage.go to parse runtime section
- [X] T009 [P] Add runtime parsing tests to sdk/manifest/datapackage_test.go
- [X] T010 Add pipeline.yaml deprecation detection in sdk/manifest/parser.go

**Checkpoint**: Foundation ready - user story implementation can now begin ✅

---

## Phase 3: User Story 1 - Define Complete DataPackage in One File (Priority: P1) 🎯 MVP

**Goal**: Users can define entire DataPackage in single dp.yaml with spec.runtime section

**Independent Test**: Create dp.yaml with runtime section, run `dp validate` and `dp run` successfully

### Unit Tests for User Story 1

- [X] T011 [P] [US1] Add runtime validation tests to sdk/validate/datapackage_test.go
- [X] T012 [P] [US1] Add docker runner binding-mapping tests to sdk/runner/docker_test.go

### Implementation for User Story 1

- [X] T013 [US1] Update sdk/validate/datapackage.go to require spec.runtime section
- [X] T014 [US1] Update sdk/runner/docker.go to use runtime from dp.yaml
- [X] T015 [US1] Update sdk/runner/docker.go to auto-map bindings to env vars
- [X] T016 [US1] Add deprecation warning when pipeline.yaml detected in sdk/runner/docker.go
- [X] T017 [US1] Update examples/kafka-s3-pipeline/dp.yaml with runtime section
- [X] T018 [US1] Delete examples/kafka-s3-pipeline/pipeline.yaml

**Checkpoint**: User Story 1 complete - `dp run` works with single dp.yaml ✅

---

## Phase 4: User Story 2 - Override Configuration at Runtime (Priority: P2)

**Goal**: Users can override config values with `--set` and `-f` flags

**Independent Test**: Run `dp run --set spec.resources.memory=8Gi` and verify override applied

### Unit Tests for User Story 2

- [X] T019 [P] [US2] Add --set flag parsing tests to cli/cmd/run_test.go
- [X] T020 [P] [US2] Add -f override file tests to cli/cmd/run_test.go

### Implementation for User Story 2

- [X] T021 [US2] Add --set flag (repeatable) to cli/cmd/run.go
- [X] T022 [US2] Add -f flag (repeatable) to cli/cmd/run.go
- [X] T023 [US2] Integrate merge logic in cli/cmd/run.go to apply overrides before running
- [X] T024 [US2] Add override precedence handling (dp.yaml < -f files < --set)
- [X] T025 [US2] Add validation error for invalid override paths in cli/cmd/run.go

**Checkpoint**: User Story 2 complete - overrides work via CLI flags ✅

---

## Phase 5: User Story 3 - View Effective Configuration (Priority: P3)

**Goal**: Users can see merged config before running via `dp show`

**Independent Test**: Run `dp show -f overrides.yaml` and verify merged output

### Unit Tests for User Story 3

- [X] T026 [P] [US3] Add dp show command tests to cli/cmd/show_test.go

### Implementation for User Story 3

- [X] T027 [US3] Create cli/cmd/show.go with dp show command
- [X] T028 [US3] Add -f and --set flags to dp show command
- [X] T029 [US3] Add --output flag for json/yaml format selection
- [X] T030 [US3] Register show command in cli/cmd/root.go

**Checkpoint**: User Story 3 complete - `dp show` displays merged manifest ✅

---

## Phase 6: User Story 4 - Validate with Overrides (Priority: P3)

**Goal**: Users can validate merged manifest before running

**Independent Test**: Run `dp validate -f overrides.yaml` with valid and invalid overrides

### Unit Tests for User Story 4

- [X] T031 [P] [US4] Add dp validate override tests to cli/cmd/lint_test.go

### Implementation for User Story 4

- [X] T032 [US4] Add -f and --set flags to cli/cmd/lint.go (dp validate)
- [X] T033 [US4] Apply merge logic before validation in cli/cmd/lint.go
- [X] T034 [US4] Add descriptive error messages for invalid merged config

**Checkpoint**: User Story 4 complete - `dp validate` works with overrides ✅

---

## Phase 7: Polish & Documentation

**Purpose**: Update all documentation to reflect single-file approach

- [X] T035 [P] Update docs/concepts/data-packages.md - remove pipeline.yaml references
- [X] T036 [P] Update docs/concepts/overview.md - remove pipeline.yaml from structure
- [X] T037 [P] Update docs/concepts/index.md - remove pipeline.yaml mention
- [X] T038 [P] Update docs/getting-started/quickstart.md - single dp.yaml workflow
- [X] T039 [P] Update docs/reference/cli.md - add --set and -f flags, add dp show
- [X] T040 [P] Update docs/reference/index.md - remove pipeline.yaml from manifest list
- [X] T041 [P] Update docs/reference/manifest-schema.md - add runtime section, remove pipeline.yaml schema
- [X] T042 [P] Update docs/tutorials/kafka-to-s3.md - consolidated dp.yaml example
- [X] T043 [P] Update docs/tutorials/promoting-packages.md - update YAML examples
- [X] T044 [P] Update docs/troubleshooting/common-issues.md - update examples
- [X] T045 [P] Update docs/troubleshooting/faq.md - update manifest comparison table
- [X] T046 Run quickstart.md validation from specs/004-consolidate-manifest/quickstart.md

**Checkpoint**: Phase 7 complete - Documentation updated ✅

---

## Summary

All phases completed successfully:
- ✅ Phase 1: Setup - RuntimeSpec struct, tests, deprecation markers
- ✅ Phase 2: Foundational - envmapper, merge logic, runtime parsing
- ✅ Phase 3: User Story 1 - Single dp.yaml with runtime section (MVP)
- ✅ Phase 4: User Story 2 - Override configuration with --set and -f flags
- ✅ Phase 5: User Story 3 - dp show command for effective configuration
- ✅ Phase 6: User Story 4 - Validate with overrides
- ✅ Phase 7: Documentation updates

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - can start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 - BLOCKS all user stories
- **Phase 3-6 (User Stories)**: All depend on Phase 2 completion
  - User stories can proceed in priority order (P1 → P2 → P3)
  - Or in parallel if resources allow
- **Phase 7 (Polish)**: Can start after Phase 3 (US1) is complete

### User Story Dependencies

| Story | Depends On | Can Start After |
|-------|------------|-----------------|
| US1 (P1) | Phase 2 | T010 complete |
| US2 (P2) | Phase 2 | T010 complete |
| US3 (P3) | Phase 2, sdk/manifest/merge.go | T007 complete |
| US4 (P3) | Phase 2, sdk/manifest/merge.go | T007 complete |

### Within Each User Story

1. Tests MUST be written first (tasks marked with tests)
2. Implementation follows tests
3. Story complete before moving to next priority

### Parallel Opportunities

```text
# Phase 1 - All parallel:
T001 | T002 | T003

# Phase 2 - Partial parallel:
T004 → T005 (envmapper)
T006 → T007 (merge)
T008 → T009 (parsing)
T010 (deprecation detection - depends on T008)

# Phase 3 (US1) - Tests then implementation:
T011 | T012 (tests parallel)
T013 → T014 → T015 → T016 → T017 → T018 (sequential)

# Phase 4 (US2) - Tests then implementation:
T019 | T020 (tests parallel)
T021 → T022 → T023 → T024 → T025 (sequential)

# Phase 5 (US3):
T026 (test)
T027 → T028 → T029 → T030 (sequential)

# Phase 6 (US4):
T031 (test)
T032 → T033 → T034 (sequential)

# Phase 7 - All parallel:
T035 | T036 | T037 | T038 | T039 | T040 | T041 | T042 | T043 | T044 | T045
T046 (validation - after docs complete)
```

---

## Parallel Example: User Story 1

```bash
# Launch tests first (parallel):
Task T011: "Add runtime validation tests to sdk/validate/validate_test.go"
Task T012: "Add docker runner binding-mapping tests to sdk/runner/docker_test.go"

# Then implementation (sequential within story):
Task T013: "Update sdk/validate/validate.go to require spec.runtime section"
Task T014: "Update sdk/runner/docker.go to use runtime from dp.yaml"
...
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T010)
3. Complete Phase 3: User Story 1 (T011-T018)
4. **STOP and VALIDATE**: `dp validate` and `dp run` work with single dp.yaml
5. Deploy/demo if ready - core value delivered

### Incremental Delivery

1. Phase 1 + Phase 2 → Foundation ready
2. Add Phase 3 (US1) → Test → **MVP complete!**
3. Add Phase 4 (US2) → Test → Override support added
4. Add Phase 5+6 (US3+US4) → Test → Show and validate with overrides
5. Add Phase 7 (Docs) → Documentation updated

### Task Count Summary

| Phase | Tasks | Parallel |
|-------|-------|----------|
| Phase 1: Setup | 3 | 3 |
| Phase 2: Foundational | 7 | 4 |
| Phase 3: US1 (MVP) | 8 | 2 |
| Phase 4: US2 | 7 | 2 |
| Phase 5: US3 | 5 | 1 |
| Phase 6: US4 | 4 | 1 |
| Phase 7: Docs | 12 | 11 |
| **Total** | **46** | **24** |

---

## Notes

- [P] tasks = different files, no dependencies - can run in parallel
- [Story] label maps task to specific user story for traceability
- Unit tests per constitution Article VII (Quality Gates)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
