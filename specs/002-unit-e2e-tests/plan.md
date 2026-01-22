# Implementation Plan: Unit and End-to-End Tests

**Branch**: `002-unit-e2e-tests` | **Date**: 2026-01-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-unit-e2e-tests/spec.md`

## Summary

Add comprehensive unit tests to all Go modules (`contracts/`, `sdk/`, `cli/`, `platform/controller/`) and end-to-end tests validating the complete workflow (init → lint → run → build). Uses Go's standard testing framework with table-driven tests, mock interfaces for external dependencies, and `testdata/` fixtures for sample manifests.

## Technical Context

**Language/Version**: Go 1.25 (per go.mod files in all modules)  
**Primary Dependencies**: `testing` (stdlib), `github.com/stretchr/testify` (assertions), Cobra (CLI testing)  
**Storage**: N/A (tests use temp directories and `testdata/` fixtures)  
**Testing**: `go test ./...`, table-driven tests, race detection (`-race`), coverage (`-cover`)  
**Target Platform**: Linux/macOS (CI runs on ubuntu-latest)  
**Project Type**: Multi-module Go workspace (4 modules: contracts, sdk, cli, platform/controller)  
**Performance Goals**: Unit tests < 2 minutes total, E2E tests < 5 minutes  
**Constraints**: Unit tests must run without external dependencies (Docker, network)  
**Scale/Scope**: ~25 Go source files across 4 modules require test coverage

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Requirement | Status | Notes |
|------|-------------|--------|-------|
| **Article VII - Quality Gates** | Automated validation, unit tests where applicable | ✅ PASS | This feature directly implements Article VII |
| **Unit Testing Requirements** | All packages MUST have comprehensive unit tests | ✅ PASS | This feature fulfills the constitution mandate |
| **Article I - Developer Experience** | Clear, actionable output | ✅ PASS | Tests provide immediate feedback on failures |
| **Article IX - Maintainability** | Clear module boundaries | ✅ PASS | Tests follow same module structure |
| **Definition of Done - Tested** | Features have tests appropriate to risk | ✅ PASS | This feature adds tests to entire codebase |

**Pre-Implementation Gates:**

| Gate | Requirement | Status |
|------|-------------|--------|
| **Workflow Demo** | N/A - Tests validate existing workflow | ✅ PASS |
| **Contract Schema** | N/A - No new contracts introduced | ✅ PASS |
| **Promotion/Rollback** | N/A - No deployment changes | ✅ PASS |
| **Observability** | N/A - No runtime changes | ✅ PASS |
| **Security/Compliance** | No secrets in test fixtures | ✅ PASS |

## Project Structure

### Documentation (this feature)

```text
specs/002-unit-e2e-tests/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output - Go testing best practices
├── data-model.md        # Phase 1 output - Test entity definitions
├── quickstart.md        # Phase 1 output - Developer testing guide
├── contracts/           # Phase 1 output - N/A (no API contracts for tests)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
# Existing multi-module Go workspace structure (tests added in-place)

contracts/
├── *.go                 # Existing source files
├── *_test.go            # NEW: Unit tests for contract types
└── testdata/            # NEW: Test fixtures (sample data)

sdk/
├── validate/
│   ├── *.go             # Existing validators
│   ├── *_test.go        # NEW: Validation tests
│   └── testdata/        # NEW: Valid/invalid manifests
├── manifest/
│   ├── *.go             # Existing parsers
│   ├── *_test.go        # NEW: Parsing tests
│   └── testdata/        # NEW: Sample YAML files
├── registry/
│   ├── *.go             # Existing OCI client
│   ├── *_test.go        # NEW: Registry tests with mocks
│   └── mocks/           # NEW: Mock interfaces
├── runner/
│   ├── *.go             # Existing runner
│   ├── *_test.go        # NEW: Runner tests
│   └── mocks/           # NEW: Docker client mocks
├── lineage/
│   ├── *.go             # Existing lineage
│   └── *_test.go        # NEW: Lineage event tests
└── catalog/
    ├── *.go             # Existing catalog
    └── *_test.go        # NEW: Catalog tests

cli/
├── cmd/
│   ├── *.go             # Existing commands
│   └── *_test.go        # NEW: Command tests
└── internal/
    └── testutil/        # NEW: CLI test helpers

platform/controller/
├── internal/
│   └── *.go             # Existing controller logic
├── *_test.go            # NEW: Controller tests
└── mocks/               # NEW: K8s client mocks

tests/                   # NEW: Top-level E2E tests
└── e2e/
    ├── workflow_test.go # NEW: Full workflow E2E
    ├── testdata/        # NEW: E2E fixtures
    └── helpers.go       # NEW: E2E test utilities
```

**Structure Decision**: Tests are co-located with source files following Go conventions (`*_test.go` in same package). E2E tests are in a new top-level `tests/e2e/` directory to keep integration tests separate from unit tests.

## Complexity Tracking

> No constitution violations - this feature implements required testing per Article VII and Unit Testing Requirements.

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Test Framework | Go stdlib `testing` | Constitution prefers proven components; stdlib is sufficient |
| Assertions | `testify/assert` (optional) | Table-driven tests with stdlib work; testify optional for readability |
| Mocking | Interface-based mocks | No external mocking framework required; interfaces already defined |
| E2E Approach | CLI binary execution | Matches spec FR-011; tests real user experience |
