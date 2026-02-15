# Tasks: End-to-End Demo Recordings

**Input**: Design documents from `/specs/014-demo-recordings/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/dialog-file-format.md, quickstart.md

**Tests**: Not explicitly requested as TDD in the feature specification. However, test infrastructure (E2E demo tests) is a core deliverable per User Story 2 (P1) and is included as implementation tasks, not separate pre-written test tasks.

**Organization**: Tasks are grouped by user story (P1–P3) to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the top-level `demos/` directory structure and placeholder files

- [X] T001 Create `demos/` directory structure with `quickstart/recordings/` and `dev-lifecycle/recordings/` subdirectories
- [X] T002 [P] Add `.gitkeep` files to `demos/quickstart/recordings/` and `demos/dev-lifecycle/recordings/` to preserve empty recording directories in Git

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the shared runner script that ALL user stories depend on — the core engine that interprets dialog files

**⚠️ CRITICAL**: No demo content, tests, or documentation can be validated until the runner script is complete and functional

- [X] T003 Create the demo runner script in demos/run_demo.sh with the main loop: read dialog file line by line, dispatch by directive prefix (`SAY:`, `CMD:`, `WAIT:`, `REQUIRE:`, `#`, blank). Include usage message when called without arguments. Make executable (`chmod +x`). (FR-001, FR-002, FR-013)
- [X] T004 Implement `SAY:` directive handling in demos/run_demo.sh — print narration text in cyan bold using tput, with a blank line before each narration block for readability (FR-001, FR-013)
- [X] T005 Implement `CMD:` directive handling in demos/run_demo.sh — print `$ <command>` with green prompt, execute via `eval`, show output, track step number, stop on non-zero exit code with error message reporting step number, command, and exit code (FR-002, FR-003, FR-005, FR-013)
- [X] T006 Implement `WAIT:` directive handling in demos/run_demo.sh — sleep for specified seconds (supports decimals), default to 1 second for missing or invalid values (FR-001)
- [X] T007 Implement `REQUIRE:` directive handling in demos/run_demo.sh — collect all `REQUIRE:` lines in a first pass, detect environment variables (all-uppercase with underscores) vs commands, check prerequisites before executing any directives, exit with code 2 and list all missing prerequisites (FR-004)

**Checkpoint**: Runner script is complete and functional. Running `./demos/run_demo.sh <dialog-file>` correctly executes all directive types, stops on errors, and checks prerequisites. All subsequent phases can now be validated.

---

## Phase 3: User Story 1 — Scripted, Reproducible Demo Execution (Priority: P1) 🎯 MVP

**Goal**: A developer can author a plain-text dialog file and run it through the runner to produce clean, deterministic terminal output. The quickstart demo (`dp init` → `dp lint` → `dp show`) is the first proof of the system.

**Independent Test**: Run `./demos/run_demo.sh demos/quickstart/demo.txt` after `make build`. Verify narration prints in color, commands execute with output, and the demo completes with exit code 0. Break a command (e.g., change `dp lint` to `dp lintx`) and verify the runner stops with an error.

### Implementation for User Story 1

- [X] T008 [P] [US1] Create the quickstart dialog file in demos/quickstart/demo.txt with SAY/CMD/WAIT directives covering: dp init → ls → dp lint → dp show workflow. Use a temp directory name to avoid conflicts. (FR-001, FR-011)
- [X] T009 [P] [US1] Create the quickstart demo README in demos/quickstart/README.md with demo title, what it shows, prerequisites (dp binary via make build), and how to run it (FR-009)
- [X] T010 [US1] Validate the quickstart demo end-to-end: run `make build` then `./demos/run_demo.sh demos/quickstart/demo.txt`, verify all commands succeed, narration is displayed, and exit code is 0. Fix any dialog file issues. (manual validation)

**Checkpoint**: The demo runner and quickstart dialog file are working. A developer can run `./demos/run_demo.sh demos/quickstart/demo.txt` and see a clean, scripted walkthrough of the dp init workflow. This is the MVP — the foundational dialog-file-to-terminal pipeline.

---

## Phase 4: User Story 2 — Test-Backed Demo Verification (Priority: P1)

**Goal**: Each demo is backed by a Go test that runs the dialog file through the runner and verifies all commands succeed. Tests follow existing E2E patterns: `skipIfShort`, environment-variable gating, `CommandResult` capture.

**Independent Test**: Run `go test ./tests/e2e/... -run TestDemo_Quickstart -v` and verify the quickstart demo executes and passes. Run `go test ./tests/e2e/... -run TestDemo -v` and verify all non-infrastructure demos pass without special environment variables.

### Implementation for User Story 2

- [X] T011 [US2] Add `repoRootDir(t)` helper function to tests/e2e/helpers.go that returns the absolute path to the repository root (reusable by both `demoRunnerPath` and `dpBinaryPath`). Refactor `dpBinaryPath` to use it.
- [X] T012 [US2] Add `demoRunnerPath(t)` helper function to tests/e2e/helpers.go that locates demos/run_demo.sh relative to repo root, verifies it exists, and returns the absolute path
- [X] T013 [US2] Add `runDemo(t, demoName)` helper function to tests/e2e/helpers.go that executes `bash <runner-path> demos/<demoName>/demo.txt` with the dp binary directory prepended to PATH, captures stdout/stderr, and returns `*CommandResult`. Set working directory to repo root so relative paths in dialog files work.
- [X] T014 [US2] Create tests/e2e/demo_test.go with `TestDemo_Quickstart` function — calls `skipIfShort(t)`, `runDemo(t, "quickstart")`, asserts exit code 0, and verifies stdout contains expected narration text (FR-006, FR-008)
- [X] T015 [US2] Add `TestDemo_DevLifecycle` function to tests/e2e/demo_test.go — gated by `DP_E2E_DEV=1` and `skipIfShort(t)`, calls `runDemo(t, "dev-lifecycle")`, asserts exit code 0 (FR-006, FR-007)
- [X] T016 [US2] Validate test infrastructure: run `go test ./tests/e2e/... -run TestDemo_Quickstart -v` after `make build`, verify the test passes. Verify `TestDemo_DevLifecycle` is skipped when `DP_E2E_DEV` is not set. (manual validation)

**Checkpoint**: Demo tests run as part of the standard E2E test suite. CLI-only demos pass without infrastructure; infrastructure demos skip gracefully. A broken dialog file causes a test failure — demos are now CI-protected.

---

## Phase 5: User Story 3 — Asciinema Recording of Scripted Demos (Priority: P2)

**Goal**: A developer can record a demo as an asciinema `.cast` file by running the runner inside `asciinema rec`. The recording captures clean scripted output. This story is about ensuring the runner's output is recording-friendly and documenting the recording workflow.

**Independent Test**: Install asciinema (`brew install asciinema`), record the quickstart demo, play it back, and verify the `.cast` file contains narration text and command output.

### Implementation for User Story 3

- [X] T017 [US3] Verify runner output is recording-clean: run `./demos/run_demo.sh demos/quickstart/demo.txt 2>&1 | cat` and confirm no shell prompt decorations, ANSI codes render correctly, and output is consistently formatted. Fix any issues in demos/run_demo.sh. (FR-010, FR-013)
- [X] T018 [US3] Add recording instructions to demos/quickstart/README.md — include the `asciinema rec` command with `-c`, `--overwrite`, `-i 2` flags, and playback command with `-i 1 -s 1.5`. Document recommended terminal size (120×30). (FR-010)
- [X] T019 [P] [US3] Add recording instructions to demos/dev-lifecycle/README.md — same asciinema rec/play pattern, noting the demo requires a running k3d cluster (FR-010)

**Checkpoint**: The recording workflow is documented and validated. Developers can produce `.cast` files from any demo using the documented asciinema commands. The runner output is verified to be recording-friendly.

---

## Phase 6: User Story 4 — Organized Demo Library (Priority: P2)

**Goal**: Demos are organized in a consistent directory structure. Each demo is self-contained with a dialog file, README, and recordings directory. The dev-lifecycle demo (infrastructure demo) is added as the second demo in the library.

**Independent Test**: Both `demos/quickstart/` and `demos/dev-lifecycle/` exist with `demo.txt` and `README.md`. Each can be run independently. The test runner discovers and validates both.

### Implementation for User Story 4

- [X] T020 [P] [US4] Create the dev-lifecycle dialog file in demos/dev-lifecycle/demo.txt with REQUIRE directives for k3d and DP_E2E_DEV, SAY/CMD directives covering dp dev up → dp dev status → dp dev down workflow (FR-009, FR-012)
- [X] T021 [P] [US4] Create the dev-lifecycle demo README in demos/dev-lifecycle/README.md with demo title, description, prerequisites (dp binary, k3d cluster, DP_E2E_DEV=1), and how to run (FR-009)
- [X] T022 [US4] Validate that both demos can be run independently: run quickstart demo (no infra), verify dev-lifecycle demo correctly reports missing prerequisites when k3d/DP_E2E_DEV are not available (exit code 2). (manual validation)

**Checkpoint**: Two independently runnable demos exist in the library. The quickstart demo is CLI-only; the dev-lifecycle demo requires infrastructure. Both follow the same directory convention.

---

## Phase 7: User Story 5 — Demo Authoring Guide (Priority: P3)

**Goal**: A contributor guide documents how to author new demos: the dialog file format, all directives, best practices, and how to add test coverage. Any team member can create a new demo by following this guide.

**Independent Test**: A new contributor reads the guide and creates a working demo (dialog file + test + README) without additional help.

### Implementation for User Story 5

- [X] T023 [US5] Create the demo authoring guide and index in demos/README.md covering: overview of the demo system, list of available demos with descriptions and prerequisites, complete dialog file format reference (all 5 directives with syntax and examples), step-by-step guide for creating a new demo (directory → dialog file → README → test → record), best practices for stable output (filtering with `| head`, `--json | jq`, `| sed`, avoiding non-deterministic output), recommended terminal size (120×30), and recording workflow (FR-014)
- [X] T024 [US5] Validate the authoring guide: follow the guide to create a minimal new demo `demos/hello/demo.txt` with 2 SAY + 1 CMD steps, add `TestDemo_Hello` to tests/e2e/demo_test.go, run it, verify it works, then remove the hello demo (leave as validation only). (manual validation per SC-001, SC-006)

**Checkpoint**: The authoring guide enables self-service demo creation. SC-001 validated: a new demo can be created in under 15 minutes.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation links, and cleanup

- [X] T025 [P] Add `demos/` section to the top-level README.md or docs/index.md pointing developers to the demo system
- [X] T026 [P] Add a `.gitignore` entry in demos/ to ignore `*.cast` files in recordings/ directories (or document that recordings should be committed — choose one policy and document it in demos/README.md)
- [X] T027 Run quickstart.md validation — execute all commands from specs/014-demo-recordings/quickstart.md and verify they work end-to-end (SC-002 validation)
- [X] T028 Final validation: verify SC-003 (two demos exist), SC-005 (break a command in quickstart dialog, verify test fails), and clean up any temp artifacts

**Checkpoint**: Feature complete — all demos functional, tests passing, authoring guide written, documentation linked, success criteria validated.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Phase 2 — delivers MVP (runner + quickstart demo)
- **User Story 2 (Phase 4)**: Depends on Phase 3 — needs a working demo to test against
- **User Story 4 (Phase 6)**: Depends on Phase 2 — needs runner for dev-lifecycle demo; can run in parallel with US1/US2
- **User Story 3 (Phase 5)**: Depends on Phase 3 — needs working demo output to validate recording
- **User Story 5 (Phase 7)**: Depends on Phase 6 — needs both demos to exist for guide examples
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational — creates quickstart demo content
- **US2 (P1)**: Depends on US1 — needs at least one demo to test against
- **US3 (P2)**: Depends on US1 — needs working runner output to validate recording-friendliness
- **US4 (P2)**: Independent after Foundational — creates dev-lifecycle demo content (parallel with US1)
- **US5 (P3)**: Depends on US4 — needs both demos for guide examples and index

### Within Each User Story

- Content creation (dialog files, READMEs) before validation
- Helper functions before test functions
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- Phase 1: T001 and T002 can run together
- Phase 2: T004–T007 can run in parallel after T003 (runner skeleton)
- Phase 3: T008 and T009 can run in parallel (dialog file + README)
- Phase 4: T011 and T012 can run in parallel (both are new helpers)
- Phase 5: T018 and T019 can run in parallel (both are README updates)
- Phase 6: T020 and T021 can run in parallel (dialog file + README)
- Phase 8: T025 and T026 can run in parallel

---

## Parallel Example: User Story 1 (Phase 3)

```bash
# Launch dialog file and README in parallel:
Task: T008 "Create quickstart dialog file in demos/quickstart/demo.txt"
Task: T009 "Create quickstart README in demos/quickstart/README.md"

# Then sequentially:
Task: T010 "Validate quickstart demo end-to-end"
```

## Parallel Example: Foundational (Phase 2)

```bash
# First create the runner skeleton:
Task: T003 "Create demo runner script in demos/run_demo.sh"

# Then implement directives in parallel (each is a self-contained function):
Task: T004 "Implement SAY: directive"
Task: T005 "Implement CMD: directive"
Task: T006 "Implement WAIT: directive"
Task: T007 "Implement REQUIRE: directive"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (T001–T002)
2. Complete Phase 2: Foundational (T003–T007)
3. Complete Phase 3: User Story 1 (T008–T010)
4. **STOP and VALIDATE**: `./demos/run_demo.sh demos/quickstart/demo.txt` works
5. Complete Phase 4: User Story 2 (T011–T016)
6. **STOP and VALIDATE**: `go test ./tests/e2e/... -run TestDemo_Quickstart -v` passes
7. This delivers: working runner, quickstart demo, test-backed verification

### Incremental Delivery

1. Setup + Foundational → Runner script ready
2. User Story 1 → MVP: scripted quickstart demo runs cleanly → Validate
3. User Story 2 → Tests protect demos from CLI drift → Validate
4. User Story 4 → Second demo (dev-lifecycle) added → Validate (can parallel with US2)
5. User Story 3 → Recording workflow validated → Validate
6. User Story 5 → Authoring guide enables self-service → Validate
7. Polish → Documentation links, final SC validation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (quickstart demo) → then User Story 2 (test infra)
   - Developer B: User Story 4 (dev-lifecycle demo) → then User Story 3 (recording)
3. After US1–US4 complete:
   - Either developer: User Story 5 (authoring guide)
4. Team completes Polish together

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Tests are a CORE DELIVERABLE (US2), not optional TDD pre-work
- The runner script (Phase 2) is the critical path — all other work depends on it
- Dialog files use relative paths from repo root — runner must be invoked from repo root
- `make build` is a prerequisite for running any demo that uses `dp` commands
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
