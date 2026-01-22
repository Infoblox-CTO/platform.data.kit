# Tasks: Unit and End-to-End Tests

**Input**: Design documents from `/specs/002-unit-e2e-tests/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, quickstart.md ✅

**Tests**: This feature IS about adding tests - all tasks create test files.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, etc.)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Test Infrastructure)

**Purpose**: Project initialization for testing infrastructure

- [X] T001 Create testdata directory structure for sdk/validate/testdata/ with valid/ and invalid/ subdirectories
- [X] T002 [P] Create testdata directory structure for sdk/manifest/testdata/ with valid/ and invalid/ subdirectories
- [X] T003 [P] Create tests/e2e/ directory for end-to-end tests with go.mod file
- [X] T004 [P] Add test helper package in cli/internal/testutil/helpers.go for shared test utilities
- [X] T005 Update Makefile to add test-unit, test-e2e, and test-coverage targets

---

## Phase 2: Foundational (Test Fixtures & Mocks)

**Purpose**: Core test infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: User story tests depend on these fixtures and mocks being in place

- [X] T006 Create valid pipeline fixture in sdk/validate/testdata/valid/pipeline-full.yaml
- [X] T007 [P] Create valid datapackage fixture in sdk/validate/testdata/valid/datapackage-basic.yaml
- [X] T008 [P] Create invalid fixture (missing name) in sdk/validate/testdata/invalid/missing-name.yaml
- [X] T009 [P] Create invalid fixture (PII violation) in sdk/validate/testdata/invalid/pii-no-classification.yaml
- [X] T010 [P] Create manifest fixtures for parsing tests in sdk/manifest/testdata/
- [X] T011 [P] Create MockRegistryClient in sdk/registry/mocks/client_mock.go
- [X] T012 [P] Create MockDockerRunner in sdk/runner/mocks/runner_mock.go
- [X] T013 [P] Create MockLineageEmitter in sdk/lineage/mocks/emitter_mock.go

**Checkpoint**: Test fixtures and mocks ready - user story implementation can begin

---

## Phase 3: User Story 1 - Developer Validates Code Changes (Priority: P1) 🎯 MVP

**Goal**: Add unit tests to contracts/ package so developers can validate code changes

**Independent Test**: Run `cd contracts && go test -v ./...` and verify all tests pass

### Implementation for User Story 1

- [X] T014 [P] [US1] Create datapackage_test.go with table-driven tests for DataPackage struct in contracts/datapackage_test.go
- [X] T015 [P] [US1] Create pipeline_test.go with table-driven tests for Pipeline struct in contracts/pipeline_test.go
- [X] T016 [P] [US1] Create binding_test.go with table-driven tests for Binding struct in contracts/binding_test.go
- [X] T017 [P] [US1] Create errors_test.go with tests for error types and codes in contracts/errors_test.go
- [X] T018 [P] [US1] Create types_test.go with tests for shared types in contracts/types_test.go
- [X] T019 [P] [US1] Create validator_test.go with table-driven validation tests in sdk/validate/validator_test.go
- [X] T020 [P] [US1] Create datapackage_test.go for datapackage validation in sdk/validate/datapackage_test.go
- [X] T021 [P] [US1] Create pipeline_test.go for pipeline validation in sdk/validate/pipeline_test.go
- [X] T022 [P] [US1] Create pii_test.go for PII validation logic in sdk/validate/pii_test.go
- [X] T023 [P] [US1] Create bindings_test.go for binding validation in sdk/validate/bindings_test.go
- [X] T024 [P] [US1] Create aggregate_test.go for aggregate validation in sdk/validate/aggregate_test.go
- [X] T025 [P] [US1] Create artifact_test.go for artifact validation in sdk/validate/artifact_test.go

**Checkpoint**: contracts/ and sdk/validate/ have 80%+ coverage, `go test` passes

---

## Phase 4: User Story 2 - Developer Validates Manifest Parsing (Priority: P1) 🎯 MVP

**Goal**: Add unit tests to sdk/manifest/ package for YAML parsing validation

**Independent Test**: Run `cd sdk && go test -v ./manifest/...` and verify parsing tests pass

### Implementation for User Story 2

- [X] T026 [P] [US2] Create parser_test.go with YAML parsing tests in sdk/manifest/parser_test.go
- [X] T027 [P] [US2] Create datapackage_test.go for datapackage manifest parsing in sdk/manifest/datapackage_test.go
- [X] T028 [P] [US2] Create pipeline_test.go for pipeline manifest parsing in sdk/manifest/pipeline_test.go
- [X] T029 [P] [US2] Create bindings_test.go for bindings manifest parsing in sdk/manifest/bindings_test.go
- [X] T030 [P] [US2] Add edge case tests for malformed YAML in sdk/manifest/parser_test.go
- [X] T031 [P] [US2] Add edge case tests for missing required fields in sdk/manifest/parser_test.go

**Checkpoint**: sdk/manifest/ has 80%+ coverage, all parsing scenarios covered

---

## Phase 5: User Story 3 - Developer Validates CLI Commands (Priority: P2)

**Goal**: Add unit tests to cli/cmd/ package for command flag parsing and execution

**Independent Test**: Run `cd cli && go test -v ./cmd/...` and verify command tests pass

### Implementation for User Story 3

- [X] T032 [P] [US3] Create lint_test.go with command tests in cli/cmd/lint_test.go
- [X] T033 [P] [US3] Create init_test.go with command tests in cli/cmd/init_test.go
- [X] T034 [P] [US3] Create build_test.go with command tests in cli/cmd/build_test.go
- [X] T035 [P] [US3] Create run_test.go with command tests in cli/cmd/run_test.go
- [X] T036 [P] [US3] Create publish_test.go with command tests in cli/cmd/publish_test.go
- [X] T037 [P] [US3] Create promote_test.go with command tests in cli/cmd/promote_test.go
- [X] T038 [P] [US3] Create root_test.go with root command tests in cli/cmd/root_test.go
- [X] T039 [P] [US3] Add test fixtures for CLI commands in cli/cmd/testdata/

**Checkpoint**: cli/cmd/ tests verify flag parsing and argument handling

---

## Phase 6: User Story 4 - Developer Validates End-to-End Workflow (Priority: P2)

**Goal**: Add E2E tests that validate complete init → lint → run → build workflow

**Independent Test**: Run `go test -v ./tests/e2e/...` (requires Docker for full test)

### SDK Additional Tests (needed for E2E)

- [X] T040 [P] [US4] Create client_test.go with registry client tests in sdk/registry/client_test.go
- [X] T041 [P] [US4] Create bundler_test.go with bundler tests in sdk/registry/bundler_test.go
- [X] T042 [P] [US4] Create runner_test.go with runner tests in sdk/runner/runner_test.go
- [X] T043 [P] [US4] Create events_test.go with lineage event tests in sdk/lineage/events_test.go
- [X] T044 [P] [US4] Create emitter_test.go with emitter tests in sdk/lineage/emitter_test.go
- [X] T045 [P] [US4] Create catalog_test.go with catalog tests in sdk/catalog/catalog_test.go

### E2E Tests

- [X] T046 [US4] Create E2E test helpers in tests/e2e/helpers.go
- [X] T047 [US4] Create E2E context setup in tests/e2e/setup_test.go
- [X] T048 [US4] Create init workflow E2E test in tests/e2e/init_test.go
- [X] T049 [US4] Create lint workflow E2E test in tests/e2e/lint_test.go
- [X] T050 [US4] Create build workflow E2E test in tests/e2e/build_test.go
- [X] T051 [US4] Create full workflow E2E test (init→lint→build) in tests/e2e/workflow_test.go
- [X] T052 [P] [US4] Create E2E test fixtures in tests/e2e/testdata/

**Checkpoint**: E2E tests validate complete workflow, skippable with -short flag

---

## Phase 7: User Story 5 - CI Pipeline Enforces Test Quality (Priority: P3)

**Goal**: CI runs all tests and reports coverage on every PR

**Independent Test**: Push a PR and verify CI runs tests with coverage reporting

### Platform Controller Tests

- [X] T053 [P] [US5] Create reconciler_test.go for controller tests in platform/controller/internal/reconciler_test.go
- [X] T054 [P] [US5] Create MockK8sClient in platform/controller/mocks/k8s_mock.go

### CI Integration

- [X] T055 [US5] Update CI workflow to aggregate coverage reports in .github/workflows/ci.yaml
- [X] T056 [US5] Add coverage threshold check (70% overall) to CI in .github/workflows/ci.yaml
- [X] T057 [US5] Add coverage badge to README.md

**Checkpoint**: CI blocks PRs with failing tests, reports coverage percentage

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements and validation

- [X] T058 [P] Add test documentation to docs/testing.md
- [X] T059 [P] Update CONTRIBUTING.md with testing requirements
- [X] T060 Run `go test -race ./...` across all modules to verify no race conditions
- [X] T061 Run coverage report and verify 70%+ overall coverage
- [X] T062 Validate quickstart.md test commands work correctly
- [X] T063 [P] Add .gitignore entries for coverage output files

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup - creates fixtures and mocks
- **User Story 1 (Phase 3)**: Depends on Foundational - contracts/ and sdk/validate/ tests
- **User Story 2 (Phase 4)**: Depends on Foundational - sdk/manifest/ tests
- **User Story 3 (Phase 5)**: Depends on Foundational - cli/cmd/ tests
- **User Story 4 (Phase 6)**: Depends on US1-US3 - E2E tests need full coverage
- **User Story 5 (Phase 7)**: Depends on US1-US4 - CI needs tests to run
- **Polish (Phase 8)**: Depends on all user stories complete

### User Story Dependencies

- **User Story 1 (P1)**: Independent - foundational contract and validation tests
- **User Story 2 (P1)**: Independent - manifest parsing tests (can parallel with US1)
- **User Story 3 (P2)**: Independent - CLI command tests (can parallel with US1/US2)
- **User Story 4 (P2)**: Depends on US1-US3 - E2E validates all components together
- **User Story 5 (P3)**: Depends on US1-US4 - CI runs all tests

### Within Each User Story

- All test files marked [P] can be created in parallel
- Tests can be written incrementally (one source file at a time)
- Run `go test` after each task to verify

### Parallel Opportunities

**Phase 1-2**: All [P] tasks can run in parallel
**Phase 3-5 (US1, US2, US3)**: Can all run in parallel - different packages
**Phase 6 (US4)**: SDK tests [P] in parallel, E2E tests sequential
**Phase 7-8**: CI updates sequential, docs [P] parallel

---

## Parallel Example: User Story 1

```bash
# All these tests can be written in parallel (different files):
T014: contracts/datapackage_test.go
T015: contracts/pipeline_test.go
T016: contracts/binding_test.go
T017: contracts/errors_test.go
T018: contracts/types_test.go
T019: sdk/validate/validator_test.go
T020: sdk/validate/datapackage_test.go
T021: sdk/validate/pipeline_test.go
T022: sdk/validate/pii_test.go
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T013)
3. Complete Phase 3: User Story 1 - contracts/ & sdk/validate/ tests (T014-T025)
4. Complete Phase 4: User Story 2 - sdk/manifest/ tests (T026-T031)
5. **VALIDATE**: Run `go test ./...` - should pass with 80% coverage on core packages
6. This is a working MVP with core validation covered

### Incremental Delivery

1. MVP (US1+US2) → Core packages tested → Can merge
2. Add US3 → CLI tests → Merge
3. Add US4 → E2E workflow validated → Merge
4. Add US5 → CI enforces quality → Merge
5. Polish → Documentation complete → Final merge

---

## Notes

- All test files use table-driven test pattern per research.md
- Tests mock external dependencies (registry, Docker, K8s)
- E2E tests skip with `-short` flag for quick iteration
- Coverage target: 80% for contracts/ and sdk/validate/, 70% overall
- Total tasks: 63
