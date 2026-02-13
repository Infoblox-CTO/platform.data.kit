# Tasks: Python CloudQuery Plugin Support

**Input**: Design documents from `/specs/010-python-cloudquery-plugins/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

Files affected by this feature:

```text
cli/
├── cmd/
│   ├── run.go                      # Dockerfile template (python:3.13→3.11, python3.13→python3.11)
│   ├── run_test.go                 # New test: cloudQueryDockerfile Python output
│   └── init_test.go                # New assertion: pyproject.toml >=3.12
└── internal/
    └── templates/
        └── cloudquery/
            └── python/
                └── pyproject.toml.tmpl   # requires-python >=3.13 → >=3.12
```

---

## Phase 1: Setup

**Purpose**: No project initialization needed — this feature modifies an existing Go CLI codebase.

- [X] T001 Review existing code: read `cloudQueryDockerfile()` in cli/cmd/run.go (lines 770–791), python template files in cli/internal/templates/cloudquery/python/, and existing test patterns in cli/cmd/run_test.go and cli/cmd/init_test.go

---

## Phase 2: Foundational (Core Code Changes)

**Purpose**: The two source code changes that ALL user stories depend on. No user story can be validated until these are complete.

**⚠️ CRITICAL**: These changes unblock US1 through US5.

- [X] T002 Update Python Dockerfile template in cli/cmd/run.go: change `python:3.13-slim` to `python:3.11-slim` on line 776, change `python3.13` to `python3.11` on lines 786 and 788 (build must match distroless runtime Python 3.11 due to grpcio ABI requirement — see research.md Decision 1)
- [X] T003 [P] Update requires-python version in cli/internal/templates/cloudquery/python/pyproject.toml.tmpl: change `requires-python = ">=3.13"` to `requires-python = ">=3.12"` on line 9
- [X] T004 Add `TestCloudQueryDockerfile_Python` in cli/cmd/run_test.go: call `cloudQueryDockerfile("python", 7777)` and assert the result contains `python:3.11-slim` (build stage), `gcr.io/distroless/python3-debian12:nonroot` (runtime stage), `python3.11/site-packages` (PYTHONPATH), `ENTRYPOINT ["python3", "main.py", "serve", "--address", "[::]:7777"]`, and does NOT contain `python:3.13` or `python3.13`
- [X] T005 [P] Add `TestCloudQueryDockerfile_Go` in cli/cmd/run_test.go: call `cloudQueryDockerfile("go", 7777)` and assert the result contains `golang:` (build stage), `gcr.io/distroless/static-debian12:nonroot` (runtime stage) — ensures Go path is not broken by the Python changes
- [X] T006 Add pyproject.toml version assertion to `TestInitCmd_CloudQueryPython` in cli/cmd/init_test.go: after the existing dp.yaml assertions, read the scaffolded `pyproject.toml` file and assert it contains `requires-python = ">=3.12"` and does NOT contain `3.13`
- [X] T007 Run full test suite: execute `cd cli && go test ./cmd/... -v -count=1` and verify all existing tests still pass plus the new tests T004, T005, T006 pass

**Checkpoint**: Foundation ready — all code changes applied, all unit tests pass

---

## Phase 3: User Story 1 — Scaffold a Python CloudQuery Source Plugin (Priority: P1)

**Goal**: `dp init -t cloudquery -l python foo` creates a project with correct Python 3.12 metadata.

**Independent Test**: Run `dp init -t cloudquery -l python foo` and verify all files exist with correct versions.

### Implementation for User Story 1

- [X] T008 [US1] Validate scaffold end-to-end: run `dp init -t cloudquery -l python test-scaffold` in a temp directory, then verify (a) all 11 expected files exist per data-model.md, (b) `pyproject.toml` contains `requires-python = ">=3.12"`, (c) `dp.yaml` has `language: python` and `role: source`, (d) `main.py` is syntactically valid Python

**Checkpoint**: User Story 1 complete — scaffold produces correct Python 3.12 project

---

## Phase 4: User Story 2 — Build and Run a Python Plugin Locally (Priority: P1) 🎯 MVP

**Goal**: `dp run` builds a Python plugin container with `python:3.12-slim` build stage and distroless runtime, deploys to k3d, discovers tables.

**Independent Test**: From a scaffolded Python plugin directory, run `dp run` and verify image builds, pod deploys, tables are discovered.

### Implementation for User Story 2

- [X] T009 [US2] Validate build and run end-to-end: from the scaffolded project directory, run `dp run` and verify (a) build output shows `python:3.11-slim` base image (must match distroless 3.11 runtime for grpcio ABI compatibility), (b) pod reaches Ready state, (c) gRPC port-forward succeeds, (d) table discovery shows `example_resource` with columns `id`, `name`, `value`, `active`, (e) cleanup removes pod

**Checkpoint**: User Story 2 complete — Python plugin builds with 3.12 and runs in k3d

---

## Phase 5: User Story 3 — Sync a Python Plugin to File Output (Priority: P1) 🎯 MVP

**Goal**: `dp run --sync` syncs Python plugin data to local JSON files in `./cq-sync-output/`.

**Independent Test**: Run `dp run --sync` and verify JSON files appear with example resource data.

### Implementation for User Story 3

- [X] T010 [US3] Validate file sync end-to-end: from the scaffolded project directory, run `dp run --sync` and verify (a) sync completes with zero errors, (b) `./cq-sync-output/` directory contains JSON file(s), (c) output contains 2 example resource records, (d) CLI summary shows resource count and zero errors

**Checkpoint**: User Story 3 complete — Python plugin syncs data to local files

---

## Phase 6: User Story 4 — Sync a Python Plugin to PostgreSQL (Priority: P2)

**Goal**: `dp run --sync --destination postgresql` syncs Python plugin data to the auto-detected PostgreSQL in k3d.

**Independent Test**: Run `dp run --sync --destination postgresql` and verify data appears in the database.

### Implementation for User Story 4

- [X] T011 [US4] Validate PostgreSQL sync end-to-end: from the scaffolded project directory, run `dp run --sync --destination postgresql` and verify (a) sync completes with zero errors, (b) query `SELECT * FROM example_resource` in the k3d PostgreSQL returns 2 rows, (c) CLI summary shows resource count and zero errors

**Checkpoint**: User Story 4 complete — Python plugin syncs to PostgreSQL

---

## Phase 7: User Story 5 — Run Unit Tests for a Python Plugin (Priority: P3)

**Goal**: `dp test` runs pytest on the scaffolded Python plugin and all tests pass.

**Independent Test**: Run `dp test` and verify pytest executes with all tests passing.

### Implementation for User Story 5

- [X] T012 [US5] Validate dp test end-to-end: from the scaffolded project directory, run `dp test` and verify (a) pytest is invoked, (b) all 8 template-provided tests pass, (c) exit code is 0

**Checkpoint**: User Story 5 complete — dp test runs pytest successfully

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and documentation

- [X] T013 [P] Run quickstart.md validation: execute all 9 steps from specs/010-python-cloudquery-plugins/quickstart.md sequentially and confirm all validation criteria pass
- [X] T014 Commit all changes with message `feat(010): python cloudquery plugins use python 3.11 build matching distroless 3.11 runtime`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — code review only
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — specifically T003 (pyproject.toml fix)
- **US2 (Phase 4)**: Depends on Phase 2 — specifically T002 (Dockerfile fix); also depends on US1 (need scaffold to test build)
- **US3 (Phase 5)**: Depends on US2 (need working build/run)
- **US4 (Phase 6)**: Depends on US2 (need working build/run); independent of US3
- **US5 (Phase 7)**: Depends on US1 (need scaffold); independent of US2–US4
- **Polish (Phase 8)**: Depends on all user stories complete

### User Story Dependencies

```
Phase 2 (Foundational)
  ├── US1 (Scaffold) ←── US2 (Build/Run) ←── US3 (File Sync)
  │                  │                   └── US4 (PostgreSQL Sync)
  │                  └── US5 (dp test)
```

- **US1 → US2**: Must scaffold before testing build
- **US2 → US3, US4**: Must build before testing sync
- **US1 → US5**: Must scaffold before testing pytest
- **US3 ∥ US4**: File sync and PostgreSQL sync are independent of each other
- **US5 ∥ US3, US4**: dp test is independent of sync stories

### Within Each User Story

- Foundational code changes before validation
- Unit tests before end-to-end validation
- Core flow before extended scenarios

### Parallel Opportunities

- **Phase 2**: T002 and T003 can run in parallel (different files)
- **Phase 2**: T004 and T005 can run in parallel (different test functions in same file)
- **Phase 5 ∥ Phase 6**: US3 and US4 can be validated in parallel after US2
- **Phase 7 ∥ Phases 5–6**: US5 can be validated in parallel with sync stories

---

## Parallel Example: Phase 2 (Foundational)

```text
# These two code changes are in different files, can run in parallel:
T002: "Update Python Dockerfile in cli/cmd/run.go"
T003: "Update pyproject.toml.tmpl in cli/internal/templates/cloudquery/python/"

# These two tests are in the same file but different functions, can run in parallel:
T004: "Add TestCloudQueryDockerfile_Python in cli/cmd/run_test.go"
T005: "Add TestCloudQueryDockerfile_Go in cli/cmd/run_test.go"
```

## Parallel Example: Sync Validation (after US2 complete)

```text
# These three stories are independent, can run in parallel:
T010: "Validate file sync end-to-end (US3)"
T011: "Validate PostgreSQL sync end-to-end (US4)"
T012: "Validate dp test end-to-end (US5)"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 + 3)

1. Complete Phase 2: Foundational (2 file changes + 3 test additions)
2. Validate US1: Scaffold produces correct output
3. Validate US2: Build and run works end-to-end
4. Validate US3: File sync works end-to-end
5. **STOP and VALIDATE**: Python plugin works from init through sync-to-file
6. This is the MVP — a developer can go from zero to synced data

### Incremental Delivery

1. Foundational changes → Unit tests pass → Code is correct
2. US1 (Scaffold) → Correct Python 3.12 project generated
3. US2 (Build/Run) → Container builds and runs in k3d
4. US3 (File Sync) → Full source→file pipeline works (MVP complete!)
5. US4 (PostgreSQL Sync) → Extended destination support
6. US5 (dp test) → Developer testing workflow complete

### Scope Note

This feature requires **only 2 files changed** (4 line substitutions) plus **3 new test functions**. The user stories represent validation scenarios for the cascading effects of those changes, not new feature code. The existing scaffolding, build, run, sync, and test infrastructure already supports Python — the only fixes are version numbers.

---

## Notes

- The Dockerfile uses `python:3.11-slim` for build and `gcr.io/distroless/python3-debian12:nonroot` (Python 3.11) for runtime — both must be 3.11 because grpcio Cython extensions are ABI-specific per minor version (see research.md Decision 1)
- Pip `--target=/deps` creates a standalone deps directory copied to the runtime stage
- No changes to init.go, build.go, test.go — Python paths already work
- No changes to any template files except pyproject.toml.tmpl
- End-to-end validation (T008–T012) requires a running k3d cluster (`dp dev up`)
- Commit after Phase 2 unit tests pass; validate end-to-end before final commit
