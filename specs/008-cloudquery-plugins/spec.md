# Feature Specification: CloudQuery Plugin Package Type

**Feature Branch**: `008-cloudquery-plugins`  
**Created**: 2026-02-12  
**Status**: Draft  
**Input**: User description: "Add a new cloudquery package type to the Data Platform that scaffolds, runs, tests, and publishes CloudQuery plugins as first-class data packages — supporting both Python and Go source plugins."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scaffold a Python CloudQuery Source Plugin (Priority: P1)

A data engineer wants to build a new CloudQuery source plugin in Python to extract data from an internal API. They run `dp init --type cloudquery --lang python`, provide a package name and namespace, and receive a fully scaffolded project directory that includes a working gRPC plugin server, example table with resolver, unit tests, Dockerfile, and all necessary dependency files. The generated project starts and passes all tests immediately — no manual wiring required.

**Why this priority**: Scaffolding is the entry point for every new CloudQuery plugin. Without it, no other command matters. Python is the default language because the CloudQuery Python SDK has lower friction for data engineers who are the primary audience.

**Independent Test**: Run `dp init --type cloudquery --lang python --name my-source --namespace acme` in an empty directory. Verify the generated project starts a gRPC server on port 7777 and passes `pytest` with no modifications.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** the user runs `dp init --type cloudquery --lang python --name my-source --namespace acme`, **Then** the CLI creates a project with `dp.yaml`, `main.py`, `plugin/plugin.py`, `plugin/client.py`, `plugin/spec.py`, `plugin/tables/example_resource.py`, `tests/`, `Dockerfile`, `pyproject.toml`, and `requirements.txt`.
2. **Given** a freshly scaffolded Python CloudQuery project, **When** the user runs `pytest` in the project directory, **Then** all generated unit tests pass.
3. **Given** a freshly scaffolded Python CloudQuery project, **When** the user runs `python main.py`, **Then** the plugin starts a gRPC server listening on port 7777.
4. **Given** a freshly scaffolded Python CloudQuery project, **When** the user inspects `dp.yaml`, **Then** it contains `type: cloudquery`, `spec.cloudquery.role: source`, and `runtime.image` is set.
5. **Given** the user runs `dp init --type cloudquery` without specifying `--lang`, **Then** the language defaults to `python`.

---

### User Story 2 - Scaffold a Go CloudQuery Source Plugin (Priority: P2)

A platform engineer wants to build a high-performance CloudQuery source plugin in Go. They run `dp init --type cloudquery --lang go`, and receive a complete Go project with the CloudQuery Go SDK wired up, an example table with resolver and test, a multi-stage Dockerfile, and a `go.mod` pinning the SDK. The project compiles and passes `go test` immediately.

**Why this priority**: Go support is essential for performance-sensitive plugins and teams with Go expertise, but Python is the more common starting point for data engineers.

**Independent Test**: Run `dp init --type cloudquery --lang go --name my-source --namespace acme` in an empty directory. Verify the project compiles with `go build ./...` and passes `go test ./...` with no modifications.

**Acceptance Scenarios**:

1. **Given** an empty directory, **When** the user runs `dp init --type cloudquery --lang go --name my-source --namespace acme`, **Then** the CLI creates a project with `dp.yaml`, `main.go`, `plugin/plugin.go`, `plugin/spec.go`, `internal/client/client.go`, `internal/tables/example_resource.go`, `internal/tables/example_resource_test.go`, `Dockerfile`, and `go.mod`.
2. **Given** a freshly scaffolded Go CloudQuery project, **When** the user runs `go test ./...`, **Then** all generated tests pass.
3. **Given** a freshly scaffolded Go CloudQuery project, **When** the user runs `go build ./...`, **Then** the project compiles without errors.
4. **Given** a freshly scaffolded Go CloudQuery project, **When** the user runs the built binary, **Then** the plugin starts a gRPC server listening on port 7777.

---

### User Story 3 - Run a CloudQuery Source Plugin Locally (Priority: P1)

A data engineer has built a CloudQuery source plugin and wants to test it end-to-end by syncing data into a local PostgreSQL database. They run `dp run` from the project directory. The CLI reads `dp.yaml`, detects `type: cloudquery`, builds the plugin container, starts it as a gRPC server, generates a temporary CloudQuery sync configuration pointing to the plugin and the local PostgreSQL destination from `dp dev`, runs `cloudquery sync`, and displays a summary of tables synced, rows fetched, and any errors.

**Why this priority**: Running a plugin end-to-end is the core developer loop — it validates the plugin works against real data before publishing. This is equal priority to scaffolding because a scaffold that cannot be run has limited value.

**Independent Test**: In a scaffolded CloudQuery project with `dp dev` running (local PostgreSQL available), run `dp run`. Verify the sync completes and the summary shows at least one table synced.

**Acceptance Scenarios**:

1. **Given** a CloudQuery project directory with `dp.yaml` containing `type: cloudquery`, **When** the user runs `dp run`, **Then** the CLI builds the plugin container, starts the gRPC server, generates a sync config, runs `cloudquery sync`, and displays a sync summary.
2. **Given** the `cloudquery` CLI binary is not installed on the user's machine, **When** the user runs `dp run` in a CloudQuery project, **Then** the CLI fails with a clear error message that includes installation instructions for the `cloudquery` binary.
3. **Given** a CloudQuery project and `dp dev` is running with a local PostgreSQL instance, **When** the user runs `dp run`, **Then** the generated sync configuration targets the local PostgreSQL as the destination.
4. **Given** a CloudQuery plugin that encounters errors during sync (e.g., auth failure), **When** the sync completes, **Then** the summary displays per-table error counts alongside successful row counts.

---

### User Story 4 - Test a CloudQuery Plugin (Priority: P2)

A data engineer wants to run tests on their CloudQuery plugin. Running `dp test` executes unit tests using the appropriate test runner (pytest for Python, go test for Go). For deeper validation, they run `dp test --integration`, which builds the container, starts the gRPC server, runs a real `cloudquery sync`, and reports per-table results.

**Why this priority**: Testing is critical for quality but depends on scaffolding (P1) and run (P1) being functional first.

**Independent Test**: In a scaffolded CloudQuery project, run `dp test` and verify unit tests execute. Then run `dp test --integration` with `dp dev` running and verify a full sync completes with per-table reporting.

**Acceptance Scenarios**:

1. **Given** a Python CloudQuery project, **When** the user runs `dp test`, **Then** the CLI runs `pytest` and reports test results.
2. **Given** a Go CloudQuery project, **When** the user runs `dp test`, **Then** the CLI runs `go test ./...` and reports test results.
3. **Given** a CloudQuery project with `dp dev` running, **When** the user runs `dp test --integration`, **Then** the CLI builds the container, starts the gRPC server, runs `cloudquery sync`, and reports per-table sync results (tables synced, rows fetched, errors).

---

### User Story 5 - Validate CloudQuery Manifest Fields (Priority: P2)

A data engineer runs `dp lint` on their CloudQuery project. The CLI validates that the `dp.yaml` manifest includes valid CloudQuery-specific fields: `spec.cloudquery.role` must be present and set to `source` (or `destination`), `spec.cloudquery.tables` should be a list, `spec.cloudquery.grpcPort` should be a valid port number, and `runtime.image` must be set.

**Why this priority**: Validation catches configuration errors early and depends on the manifest schema being defined first (scaffolding, P1).

**Independent Test**: Create a `dp.yaml` with missing or invalid CloudQuery fields and run `dp lint`. Verify specific validation errors are reported.

**Acceptance Scenarios**:

1. **Given** a `dp.yaml` with `type: cloudquery` but missing `spec.cloudquery.role`, **When** the user runs `dp lint`, **Then** the CLI reports a validation error indicating `role` is required.
2. **Given** a `dp.yaml` with `spec.cloudquery.role: destination`, **When** the user runs `dp lint`, **Then** the CLI reports that destination plugins are not yet supported.
3. **Given** a valid CloudQuery `dp.yaml`, **When** the user runs `dp lint`, **Then** the CLI reports no errors.
4. **Given** a `dp.yaml` with `spec.cloudquery.grpcPort` set to a non-numeric value, **When** the user runs `dp lint`, **Then** the CLI reports a validation error for the port field.

---

### User Story 6 - Build, Publish, and Promote CloudQuery Packages (Priority: P3)

A data engineer wants to publish their CloudQuery plugin as an OCI artifact and promote it across environments. `dp build`, `dp publish`, and `dp promote` work identically to pipeline packages — packaging the container image and manifest as an OCI artifact and pushing it to the registry.

**Why this priority**: Publishing and promotion reuse existing pipeline infrastructure and are needed only after the plugin is developed and tested (P1, P2).

**Independent Test**: Run `dp build` in a CloudQuery project, then `dp publish`, and verify the OCI artifact is pushed to the registry.

**Acceptance Scenarios**:

1. **Given** a CloudQuery project, **When** the user runs `dp build`, **Then** the CLI builds the container image and packages it as an OCI artifact.
2. **Given** a built CloudQuery package, **When** the user runs `dp publish`, **Then** the OCI artifact is pushed to the configured registry.
3. **Given** a published CloudQuery package, **When** the user runs `dp promote --env staging`, **Then** the package is promoted to the staging environment following the same promotion rules as pipeline packages.

---

### Edge Cases

- What happens when the user runs `dp init --type cloudquery --role destination`? The CLI accepts the flag but returns a clear message: "Destination plugins are not yet supported. Only source plugins are available at this time." No files are scaffolded.
- What happens when the user runs `dp run` in a CloudQuery project but `dp dev` is not running? The CLI fails with a clear error message indicating that a local development environment is required and suggests running `dp dev` first.
- What happens when the CloudQuery plugin's gRPC server fails to start within a reasonable timeout? The CLI fails with a timeout error and displays the container logs for debugging.
- What happens when the user provides an invalid `--lang` value (e.g., `--lang rust`) for a cloudquery package? The CLI reports that only `python` and `go` are supported for CloudQuery plugins.
- What happens when the user runs `dp init --type cloudquery` in a directory that already contains a `dp.yaml`? The CLI fails with an error indicating a package already exists in this directory.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The `dp init` command MUST accept `--type cloudquery` as a valid package type alongside the existing `pipeline` type.
- **FR-002**: The `dp init --type cloudquery --lang python` command MUST scaffold a complete Python CloudQuery source plugin project that starts a gRPC server and passes unit tests without modification.
- **FR-003**: The `dp init --type cloudquery --lang go` command MUST scaffold a complete Go CloudQuery source plugin project that compiles and passes tests without modification.
- **FR-004**: When `--type cloudquery` is specified without `--lang`, the language MUST default to `python`.
- **FR-005**: The scaffolded `dp.yaml` manifest MUST include a `spec.cloudquery` section with fields: `role` (required, value: `source`), `tables` (list), `grpcPort` (integer, default: 7777), and `concurrency` (integer, default: 10000).
- **FR-006**: The `dp run` command MUST detect `type: cloudquery` from `dp.yaml` and execute the CloudQuery-specific run workflow: build container → start gRPC server → generate sync config → run `cloudquery sync` → display summary.
- **FR-007**: The `dp run` command MUST fail with a clear error message and installation instructions if the `cloudquery` CLI binary is not found on the system PATH.
- **FR-008**: The `dp test` command MUST run the appropriate unit test framework based on language: `pytest` for Python, `go test` for Go.
- **FR-009**: The `dp test --integration` flag MUST perform a full end-to-end sync: build container, start gRPC server, run `cloudquery sync`, and report per-table results.
- **FR-010**: The `dp lint` command MUST validate CloudQuery-specific manifest fields: `spec.cloudquery.role` is required and must be `source` or `destination`; `spec.cloudquery.grpcPort` must be a valid port number; `runtime.image` must be present.
- **FR-011**: The `dp lint` command MUST report a clear warning when `spec.cloudquery.role` is set to `destination`, indicating that destination plugins are not yet supported.
- **FR-012**: The `dp build`, `dp publish`, and `dp promote` commands MUST work for CloudQuery packages identically to pipeline packages, packaging and distributing them as OCI artifacts.
- **FR-013**: The `dp init --type cloudquery --role destination` command MUST return a clear "not yet supported" message without scaffolding any files.
- **FR-014**: The Python scaffold MUST include: `main.py`, `plugin/plugin.py`, `plugin/client.py`, `plugin/spec.py`, `plugin/tables/example_resource.py`, `tests/`, `Dockerfile`, `pyproject.toml`, and `requirements.txt` pinning `cloudquery-plugin-sdk`.
- **FR-015**: The Go scaffold MUST include: `main.go`, `plugin/plugin.go`, `plugin/spec.go`, `internal/client/client.go`, `internal/tables/example_resource.go`, `internal/tables/example_resource_test.go`, `Dockerfile` (multi-stage), and `go.mod` pinning `github.com/cloudquery/plugin-sdk/v4`.
- **FR-016**: The `dp run` sync configuration MUST target the local PostgreSQL destination provided by `dp dev` as the default sync destination.
- **FR-017**: The `dp run` output MUST display a sync summary including tables synced, total rows fetched, and any errors encountered.

### Key Entities

- **CloudQuery Data Package**: A data package with `type: cloudquery` that contains a CloudQuery plugin (source or destination). Extends the existing DataPackageSpec with a `cloudquery` section in the spec.
- **CloudQuery Spec (manifest section)**: Configuration within `dp.yaml` under `spec.cloudquery` containing `role` (source|destination), `tables` (list of table names the plugin provides), `grpcPort` (port the gRPC server listens on), and `concurrency` (max concurrent table resolvers).
- **Plugin**: The core component — a gRPC server that implements the CloudQuery protocol (Init, GetTables, Sync, Close). In Python: a class extending `cloudquery.sdk.plugin.Plugin`. In Go: a struct implementing `plugin.Client`.
- **Client**: The API authentication and HTTP session manager used by table resolvers to fetch data from external sources. Initialized during plugin Init with configuration from the Spec.
- **Table**: A schema definition declaring the columns (name, Arrow type, description) that the plugin provides. Each table has an associated TableResolver.
- **TableResolver**: A function that fetches data from an external source via the Client and yields Apache Arrow RecordBatches to the CloudQuery sync framework.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A data engineer can go from zero to a running CloudQuery source plugin (Python or Go) in under 5 minutes using `dp init` and `dp run`.
- **SC-002**: All scaffolded projects (Python and Go) pass their respective unit tests on first run with zero modifications.
- **SC-003**: `dp run` successfully completes a full end-to-end sync (plugin start → data fetch → write to destination) for the scaffolded example plugin.
- **SC-004**: `dp lint` catches 100% of missing or invalid CloudQuery-specific manifest fields and reports actionable error messages.
- **SC-005**: `dp build` and `dp publish` successfully package and push a CloudQuery plugin as an OCI artifact using the same workflow as pipeline packages.
- **SC-006**: Users who are familiar with the existing `dp init --type pipeline` workflow can adopt `dp init --type cloudquery` with no additional documentation beyond the `--help` output.
- **SC-007**: The `dp run` command provides a clear, actionable error when the `cloudquery` CLI is missing, and 90% of users can resolve the issue on first attempt using the provided instructions.

## Assumptions

- The `cloudquery` CLI binary is a user-installed prerequisite and is expected to be on the system PATH. The platform will not bundle or install it.
- The local development environment (`dp dev`) provides a PostgreSQL instance that can serve as the default sync destination for `dp run`.
- The CloudQuery Python SDK (`cloudquery-plugin-sdk`) and Go SDK (`github.com/cloudquery/plugin-sdk/v4`) are stable and their APIs will not undergo breaking changes during the implementation period.
- Destination plugin support (`role: destination`) is reserved in the data model but will not be implemented in this feature — it is deferred to a future iteration.
- The gRPC protocol version used is CloudQuery protocol v3.
- The default gRPC port (7777) does not conflict with other services in the local development environment.
- OCI artifact packaging for CloudQuery packages uses the same registry infrastructure as pipeline packages — no new registry setup is required.
