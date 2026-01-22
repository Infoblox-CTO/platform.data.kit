# Feature Specification: Unit and End-to-End Tests

**Feature Branch**: `002-unit-e2e-tests`  
**Created**: 2026-01-22  
**Status**: Draft  
**Input**: User description: "Add unit tests to all modules and end-to-end tests for the workflow"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Validates Code Changes (Priority: P1) 🎯 MVP

A developer making changes to the DP codebase needs confidence that their changes don't break existing functionality. They run the test suite locally before committing and get immediate feedback on any regressions.

**Why this priority**: Unit tests are foundational - they catch regressions early, document expected behavior, and enable confident refactoring. Without tests, any code change is a risk.

**Independent Test**: Run `go test ./...` from repository root and verify all tests pass. Delivers immediate feedback on code correctness.

**Acceptance Scenarios**:

1. **Given** a developer has made changes to `contracts/` package, **When** they run `go test ./contracts/...`, **Then** all unit tests pass and any breaking changes are caught
2. **Given** a developer has modified validation logic, **When** they run `go test ./sdk/validate/...`, **Then** validation tests catch any regressions in error detection
3. **Given** the CI pipeline runs on a PR, **When** tests execute, **Then** failing tests block the merge until fixed

---

### User Story 2 - Developer Validates Manifest Parsing (Priority: P1) 🎯 MVP

A developer needs assurance that YAML manifest parsing correctly handles valid and invalid inputs, including edge cases like missing required fields, malformed YAML, and unknown fields.

**Why this priority**: Manifest parsing is the entry point for all user interactions. Incorrect parsing leads to silent failures or confusing errors.

**Independent Test**: Run `go test ./sdk/manifest/...` and verify all parsing scenarios work correctly.

**Acceptance Scenarios**:

1. **Given** a valid dp.yaml file, **When** parsed, **Then** all fields are correctly populated in the DataPackage struct
2. **Given** a dp.yaml with missing required fields, **When** parsed, **Then** appropriate validation errors are returned
3. **Given** malformed YAML, **When** parsed, **Then** a clear error message indicates the parsing failure

---

### User Story 3 - Developer Validates CLI Commands (Priority: P2)

A developer needs confidence that CLI commands correctly parse flags, handle arguments, and invoke the appropriate underlying SDK functions.

**Why this priority**: CLI is the user interface - incorrect flag handling leads to poor user experience.

**Independent Test**: Run `go test ./cli/cmd/...` and verify command initialization and execution work correctly.

**Acceptance Scenarios**:

1. **Given** the `dp lint` command, **When** invoked with `--strict` flag, **Then** strict validation mode is enabled
2. **Given** the `dp init` command, **When** invoked with `--type pipeline`, **Then** pipeline templates are used
3. **Given** invalid arguments, **When** a command is run, **Then** helpful error messages are displayed

---

### User Story 4 - Developer Validates End-to-End Workflow (Priority: P2)

A developer needs to verify that the complete DP workflow (init → lint → run → build) works correctly as an integrated system, not just individual components.

**Why this priority**: End-to-end tests catch integration issues that unit tests miss - ensuring components work together correctly.

**Independent Test**: Run `go test ./tests/e2e/...` and verify the complete workflow executes successfully.

**Acceptance Scenarios**:

1. **Given** a fresh directory, **When** running `dp init my-pkg --type pipeline`, **Then** valid manifest files are created
2. **Given** a valid package, **When** running `dp lint`, **Then** validation passes with no errors
3. **Given** a validated package, **When** running `dp build`, **Then** an OCI artifact is created locally
4. **Given** a running dev stack, **When** running `dp run`, **Then** the pipeline executes successfully

---

### User Story 5 - CI Pipeline Enforces Test Quality (Priority: P3)

The CI/CD pipeline must enforce test quality gates - all tests must pass and coverage thresholds must be met before code can be merged.

**Why this priority**: Automated enforcement prevents human error and maintains code quality over time.

**Independent Test**: Push a PR with failing tests or low coverage and verify CI blocks the merge.

**Acceptance Scenarios**:

1. **Given** a PR with failing unit tests, **When** CI runs, **Then** the PR is blocked from merging
2. **Given** a PR with coverage below 80%, **When** CI runs, **Then** a warning is displayed (soft gate for MVP)
3. **Given** all tests pass, **When** CI completes, **Then** the PR shows green status

---

### Edge Cases

- What happens when tests require external services (Docker, network)?
  - Tests MUST be runnable without external dependencies for unit tests
  - E2E tests MAY require Docker but MUST document this clearly
- How does the system handle flaky tests?
  - Test failures are retried once; consistent failures block CI
- What happens when test coverage decreases?
  - Coverage decrease of >2% blocks merge in strict mode
- How are long-running tests handled?
  - Tests exceeding 30 seconds are marked as slow and can be skipped with `-short` flag

## Requirements *(mandatory)*

### Functional Requirements

#### Unit Tests

- **FR-001**: All packages in `contracts/` MUST have unit tests covering exported functions
- **FR-002**: All packages in `sdk/` MUST have unit tests covering validation, parsing, and core logic
- **FR-003**: CLI commands in `cli/cmd/` MUST have tests verifying flag parsing and argument handling
- **FR-004**: Platform controller in `platform/controller/` MUST have tests for reconciliation logic
- **FR-005**: Unit tests MUST follow `*_test.go` naming convention in the same package directory
- **FR-006**: Unit tests MUST use table-driven tests for functions with multiple input scenarios
- **FR-007**: Unit tests MUST mock external dependencies (HTTP clients, file systems, Docker)
- **FR-008**: Unit tests MUST be runnable via `go test ./...` without external dependencies

#### End-to-End Tests

- **FR-009**: E2E tests MUST validate the complete workflow: init → lint → run → build
- **FR-010**: E2E tests MUST be located in `tests/e2e/` directory
- **FR-011**: E2E tests MUST use real CLI binary, not internal function calls
- **FR-012**: E2E tests MUST clean up created files and containers after execution
- **FR-013**: E2E tests MUST be skippable via `-short` flag for quick local development

#### Test Infrastructure

- **FR-014**: CI workflow MUST run `go test ./...` on every PR
- **FR-015**: CI workflow MUST report test coverage percentage
- **FR-016**: Test helpers MUST be provided for common patterns (temp directories, mock servers)
- **FR-017**: Makefile MUST have `test`, `test-unit`, `test-e2e`, and `test-coverage` targets

### Key Entities

- **TestCase**: Represents a single test scenario with input, expected output, and description
- **TestSuite**: A collection of related test cases for a specific package or feature
- **MockClient**: Interface implementations that simulate external services for isolated testing
- **TestFixture**: Reusable test data (sample manifests, configurations) stored in `testdata/` directories

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All Go packages have corresponding `*_test.go` files with at least one test per exported function
- **SC-002**: Running `go test ./...` from repository root completes successfully in under 2 minutes
- **SC-003**: Test coverage for `contracts/` and `sdk/validate/` packages reaches 80% or higher
- **SC-004**: Test coverage for all packages combined reaches 70% or higher
- **SC-005**: E2E test suite validates complete workflow (init → lint → run → build) in under 5 minutes
- **SC-006**: CI pipeline blocks PRs with failing tests within 10 minutes of push
- **SC-007**: Zero flaky tests - all tests pass consistently across 10 consecutive runs
- **SC-008**: New code contributions require corresponding tests (enforced by PR review)

## Assumptions

- Docker is available for E2E tests that require container execution
- Go 1.22+ is installed with standard testing tools
- Developers have access to run tests locally before pushing
- CI environment has sufficient resources to run tests in parallel
- Test data fixtures can use hardcoded example values (no production data)

## Out of Scope

- Performance/load testing (deferred to separate feature)
- Security/penetration testing (deferred to separate feature)
- UI/visual regression testing (no UI in MVP)
- Mutation testing (advanced technique for future consideration)
- Cross-platform testing beyond Linux/macOS (Windows support deferred)
