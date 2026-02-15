# Implementation Plan: End-to-End Demo Recordings

**Branch**: `014-demo-recordings` | **Date**: 2026-02-15 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-demo-recordings/spec.md`

## Summary

Build an end-to-end demo recording system using scripted dialog files, a bash runner engine, Go test-backed verification, and optional asciinema recording. Demos are plain-text dialog files (`demo.txt`) with simple directives (`SAY:`, `CMD:`, `WAIT:`, `REQUIRE:`, `#`) executed deterministically by a shared runner script. Each demo is backed by a Go test that runs the dialog through the runner and verifies all commands succeed. Demos are organized in self-contained directories under `demos/` at the repository root. asciinema recording is optional — demos are independently runnable and testable without it.

## Technical Context

**Language/Version**: Bash (runner script) + Go (latest stable, per go.mod — test infrastructure)
**Primary Dependencies**: bash (runner), asciinema (optional, for recording), existing E2E test helpers (`tests/e2e/helpers.go`)
**Storage**: N/A (plain-text dialog files, `.cast` recording artifacts)
**Testing**: `go test`, E2E tests in `tests/e2e/`, environment-variable gating for infrastructure demos
**Target Platform**: macOS/Linux developer workstations
**Project Type**: Multi-module Go monorepo (contracts ← sdk ← cli)
**Performance Goals**: Demos execute within 30s (CLI-only), playback under 2 minutes with `-i 1` (SC-004)
**Constraints**: No external dependencies for CLI-only demos; `dp` binary must be pre-built (`make build`)
**Scale/Scope**: 2 demos at launch (quickstart + dev-lifecycle), extensible to N demos via authoring guide

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Requirement | Status | Notes |
|------|-------------|--------|-------|
| **Workflow Demo** (Art. I) | Plan demonstrates end-to-end developer workflow | PASS | The demos themselves ARE workflow demonstrations; quickstart.md shows authoring + running + testing + recording workflow |
| **Contract Schema** (Art. II) | Contract schemas and validation strategy are explicit | PASS | Dialog file format specified in contracts/dialog-file-format.md; no changes to existing contracts/schemas |
| **Promotion/Rollback** (Art. III) | Promotion and rollback mechanics are explicit | N/A | Demo tooling is dev-only; no promotion or rollback mechanics involved |
| **Observability** (Art. VI) | Observability requirements are defined | PASS | Runner produces clear step-by-step output; test results provide CI observability; failures identify exact step number |
| **Security/Compliance** (Art. V) | Secrets, least privilege, PII metadata | PASS | Dialog files contain only CLI commands with no secrets; `REQUIRE:` enforces explicit prerequisite declaration |
| **Persona Mapping** (Art. X) | Which persona owns each artifact | PASS | Platform engineers maintain runner + demo tests; data engineers and platform engineers author demos; contributors follow authoring guide |
| **Unit Tests** (Tech Standards) | Comprehensive unit tests | PASS | Demo tests via Go E2E tests; runner tested by executing dialog files; per-demo test functions |
| **Definition of Done** | Tested, documented, observable, reversible, schema-validated | PASS | Tests (E2E), docs (authoring guide + READMEs), observable (runner output), reversible (N/A — no deployments), schema (dialog format contract) |

## Project Structure

### Documentation (this feature)

```text
specs/014-demo-recordings/
├── plan.md                          # This file
├── research.md                      # Phase 0: format design, runner language, asciinema, Go test integration
├── data-model.md                    # Phase 1: DialogFile, DemoDirectory, DemoTest, Recording entities
├── quickstart.md                    # Phase 1: developer workflow for running, testing, authoring demos
├── contracts/
│   └── dialog-file-format.md        # Phase 1: dialog file directive specification
└── tasks.md                         # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
demos/                               # NEW: top-level demo directory
├── run_demo.sh                      # NEW: shared bash runner script (FR-002, FR-003, FR-004, FR-005, FR-013)
├── README.md                        # NEW: demo index + authoring guide (FR-014)
├── quickstart/                      # NEW: CLI-only demo (FR-011)
│   ├── demo.txt                     # NEW: dialog file — dp init → dp lint → dp show
│   ├── README.md                    # NEW: demo description + prerequisites
│   └── recordings/                  # NEW: directory for .cast files
└── dev-lifecycle/                   # NEW: infrastructure demo (FR-012)
    ├── demo.txt                     # NEW: dialog file — dp dev up → status → down
    ├── README.md                    # NEW: demo description + prerequisites
    └── recordings/                  # NEW: directory for .cast files

tests/e2e/
├── demo_test.go                     # NEW: TestDemo_Quickstart, TestDemo_DevLifecycle (FR-006, FR-007, FR-008)
└── helpers.go                       # MODIFY: add runDemo(), demoRunnerPath() helpers
```

**Structure Decision**: `demos/` at repository root parallels existing top-level directories (`docs/`, `tests/`, `examples/`). Demos are first-class artifacts per Article I (DX is product). Tests remain in `tests/e2e/` following existing patterns. The runner script lives alongside the demos it serves.

## Complexity Tracking

No constitution violations requiring justification. All gates pass.

## Phase 2 Summary (for /speckit.tasks)

### Phase 1: Runner Script & Dialog Format
- Create `demos/run_demo.sh` — bash runner that parses dialog files and executes directives
- Implement all directives: `SAY:`, `CMD:`, `WAIT:`, `REQUIRE:`, `#`, blank lines
- Error handling: stop on first command failure, report step number and exit code
- Prerequisite checking: collect `REQUIRE:` lines, validate before executing

### Phase 2: Quickstart Demo
- Create `demos/quickstart/demo.txt` — dialog file covering `dp init` → `dp lint` → `dp show`
- Create `demos/quickstart/README.md` — description, prerequisites, how to run
- Create `demos/quickstart/recordings/` directory

### Phase 3: Dev Lifecycle Demo
- Create `demos/dev-lifecycle/demo.txt` — dialog file covering `dp dev up` → `dp dev status` → `dp dev down`
- Include `REQUIRE: k3d` and `REQUIRE: DP_E2E_DEV` directives
- Create `demos/dev-lifecycle/README.md`
- Create `demos/dev-lifecycle/recordings/` directory

### Phase 4: E2E Test Integration
- Add `runDemo(t, demoName)` helper to `tests/e2e/helpers.go`
- Add `demoRunnerPath(t)` helper to locate runner script
- Create `tests/e2e/demo_test.go` with `TestDemo_Quickstart` (no infra gate)
- Add `TestDemo_DevLifecycle` (gated by `DP_E2E_DEV=1`)

### Phase 5: Authoring Guide & Index
- Create `demos/README.md` — demo index, authoring guide (FR-014), dialog format reference, best practices for stable output
