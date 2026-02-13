# Feature Specification: Python CloudQuery Plugin Support

**Feature Branch**: `010-python-cloudquery-plugins`  
**Created**: 2026-02-13  
**Status**: Draft  
**Input**: User description: "data kit should allow python based cloudquery plugins. it should default to python 3.12 and use venv. the resulting docker container should use a distroless version or chainguard version."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Scaffold a Python CloudQuery Source Plugin (Priority: P1)

A developer runs `dp init -t cloudquery -l python foo` to create a new Python-based CloudQuery source plugin project. The scaffolded project includes a working plugin structure, dependency management via `requirements.txt` and `pyproject.toml`, a sample table with resolver, and unit tests. The project uses Python 3.12 as the minimum version and is ready to run without additional setup beyond creating a virtual environment.

**Why this priority**: Scaffolding is the entry point for the entire Python plugin workflow. Without a correctly generated project, no other user story can function.

**Independent Test**: Run `dp init -t cloudquery -l python foo`, verify the directory structure is created with all expected files, and confirm the generated code is syntactically valid Python.

**Acceptance Scenarios**:

1. **Given** an empty workspace, **When** the developer runs `dp init -t cloudquery -l python foo`, **Then** a `foo/` directory is created containing `dp.yaml`, `main.py`, `requirements.txt`, `pyproject.toml`, `plugin/` package (with `__init__.py`, `plugin.py`, `client.py`, `spec.py`), `plugin/tables/` package (with `__init__.py`, `example_resource.py`), and `tests/test_example_resource.py`.
2. **Given** the scaffolded project, **When** the developer inspects `dp.yaml`, **Then** `spec.type` is `cloudquery`, `spec.cloudquery.language` is `python`, and `spec.cloudquery.role` is `source`.
3. **Given** the scaffolded project, **When** the developer inspects `pyproject.toml`, **Then** `requires-python` specifies `>=3.12` and dependencies include the CloudQuery Python SDK and PyArrow.
4. **Given** the scaffolded project, **When** the developer inspects `requirements.txt`, **Then** it lists the CloudQuery Python SDK and PyArrow with minimum version constraints.

---

### User Story 2 — Build and Run a Python Plugin Locally (Priority: P1) 🎯 MVP

A developer runs `dp run` from a Python CloudQuery plugin directory. The CLI auto-detects Python (via `pyproject.toml` or `requirements.txt`), generates a Dockerfile targeting Python 3.12 with a distroless or Chainguard runtime image, builds the container, imports it into the k3d cluster, deploys it as a pod, port-forwards the gRPC endpoint, and discovers tables — identical to the Go plugin flow.

**Why this priority**: This is the core developer loop. A plugin that can't build and run has no value.

**Independent Test**: From the scaffolded `foo/` directory, run `dp run` and verify the plugin image is built, the pod is deployed, gRPC responds, and tables are discovered and displayed.

**Acceptance Scenarios**:

1. **Given** a scaffolded Python plugin project, **When** the developer runs `dp run`, **Then** the CLI detects Python, builds a container image using Python 3.12, imports it into the k3d cluster, deploys it as a pod, and displays the discovered tables.
2. **Given** the built container image, **When** inspected, **Then** it uses a distroless or Chainguard base image for the runtime stage (not a full Python distribution).
3. **Given** a Python plugin with a syntax error in `main.py`, **When** the developer runs `dp run`, **Then** the build fails with a clear error message indicating the Python issue.
4. **Given** a Python plugin with a missing dependency in `requirements.txt`, **When** the developer runs `dp run`, **Then** the build fails with a clear error indicating which package could not be installed.

---

### User Story 3 — Sync a Python Plugin to File Output (Priority: P1) 🎯 MVP

A developer runs `dp run --sync` from a Python CloudQuery plugin directory. The sync writes the plugin's data to local JSON files in `./cq-sync-output/`, identical to the Go plugin sync-to-file flow.

**Why this priority**: File output is the simplest destination and proves the full source→destination pipeline works end-to-end.

**Independent Test**: Run `dp run --sync` from the scaffolded project and verify JSON files appear in `./cq-sync-output/` containing the example resource data.

**Acceptance Scenarios**:

1. **Given** a scaffolded Python plugin, **When** the developer runs `dp run --sync`, **Then** the sync completes successfully, writing JSON output to `./cq-sync-output/`.
2. **Given** the sync output files, **When** inspected, **Then** they contain the two example resource records from the scaffold template.
3. **Given** a successful sync, **When** the CLI displays the summary, **Then** it shows the number of resources synced and zero errors.

---

### User Story 4 — Sync a Python Plugin to PostgreSQL (Priority: P2)

A developer runs `dp run --sync --destination postgresql` from a Python CloudQuery plugin directory. The sync writes data to the PostgreSQL instance running in the k3d cluster, auto-detected from the cluster services.

**Why this priority**: PostgreSQL output validates that the Python plugin works with external destinations, not just file — but depends on the P1 stories working first.

**Independent Test**: Run `dp run --sync --destination postgresql` and verify data appears in the PostgreSQL database.

**Acceptance Scenarios**:

1. **Given** a scaffolded Python plugin and a running k3d dev environment with PostgreSQL, **When** the developer runs `dp run --sync --destination postgresql`, **Then** the sync completes successfully with zero errors.
2. **Given** a successful PostgreSQL sync, **When** the developer queries the database, **Then** the `example_resource` table exists with the two example rows.

---

### User Story 5 — Run Unit Tests for a Python Plugin (Priority: P3)

A developer runs `dp test` from a Python CloudQuery plugin directory. The CLI detects Python and runs pytest against the project's test suite.

**Why this priority**: Testing is important for plugin quality but is not blocking the core init→run→sync flow.

**Independent Test**: Run `dp test` from the scaffolded project and verify pytest executes and all template-provided tests pass.

**Acceptance Scenarios**:

1. **Given** a scaffolded Python plugin, **When** the developer runs `dp test`, **Then** pytest runs and all template-provided unit tests pass.
2. **Given** a Python plugin with a failing test, **When** the developer runs `dp test`, **Then** the CLI reports the test failures clearly.

---

### Edge Cases

- What happens when Python 3.12 is not installed on the developer's machine? The container build uses its own Python — host Python is only needed for local development outside Docker. The CLI should document this.
- What happens when the developer has Python 3.13+ installed locally but the container targets 3.12? The container is authoritative; local Python version differences do not affect `dp run` or `dp run --sync`.
- What happens when `requirements.txt` references a package that doesn't exist? The Docker build fails with a clear pip error message.
- What happens when the plugin's gRPC serve address format differs between Python and Go SDK? The Dockerfile entrypoint must match the Python SDK's expected CLI arguments.
- What happens when the developer runs `dp init -t cloudquery` without specifying `-l`? The language defaults to `python` (existing behavior).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST scaffold a complete Python CloudQuery source plugin project when the developer runs `dp init -t cloudquery -l python <name>`.
- **FR-002**: The scaffolded Python project MUST include `dp.yaml`, `main.py`, `requirements.txt`, `pyproject.toml`, a `plugin/` package (with `__init__.py`, `plugin.py`, `client.py`, `spec.py`), a `plugin/tables/` package (with `__init__.py`, `example_resource.py`), and a `tests/` directory (with `test_example_resource.py`).
- **FR-003**: The `pyproject.toml` MUST specify `requires-python = ">=3.12"` and list the CloudQuery Python SDK and PyArrow as dependencies.
- **FR-004**: The `requirements.txt` MUST list the same dependencies with minimum version constraints compatible with Python 3.12.
- **FR-005**: The CLI MUST auto-detect Python plugins by checking for `pyproject.toml`, `requirements.txt`, or `main.py` in the project directory.
- **FR-006**: The CLI MUST generate a Dockerfile for Python plugins that uses Python 3.12 as the build-stage base image.
- **FR-007**: The generated Dockerfile MUST use a distroless or Chainguard base image for the runtime stage to minimize container attack surface.
- **FR-008**: The generated Dockerfile MUST use pip with cache mounts for fast rebuilds of dependencies.
- **FR-009**: The CLI MUST build, import, deploy, port-forward, and discover tables for Python plugins using the same flow as Go plugins (`dp run`).
- **FR-010**: The CLI MUST support `dp run --sync` for Python plugins, syncing data to the file destination (JSON output to `./cq-sync-output/`).
- **FR-011**: The CLI MUST support `dp run --sync --destination postgresql` for Python plugins, syncing data to the auto-detected PostgreSQL instance in the k3d cluster.
- **FR-012**: The CLI MUST run pytest when the developer executes `dp test` on a Python CloudQuery plugin project.
- **FR-013**: The scaffolded Python plugin MUST produce at least one table with sample data that can be synced end-to-end without code changes.
- **FR-014**: The Python plugin template MUST use the CloudQuery Python SDK's plugin, schema, and scheduler APIs correctly so the plugin starts, serves gRPC, and responds to table discovery and sync requests.
- **FR-015**: When language is not specified for a cloudquery package type, the CLI MUST default to Python.

### Key Entities

- **Python Plugin Project**: A directory containing `dp.yaml` (type: cloudquery, language: python), `main.py`, `requirements.txt`, `pyproject.toml`, and a `plugin/` Python package. Represents a CloudQuery source plugin implemented in Python.
- **Dockerfile (Python)**: A CLI-generated multi-stage Dockerfile. Build stage uses Python 3.12 with pip cache mounts. Runtime stage uses a distroless or Chainguard Python base. Never committed to the user's project.
- **Plugin Template**: The set of template files under the CLI's template directory for Python CloudQuery plugins that are rendered with project metadata (name, role, description) during `dp init`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can go from zero to a working Python CloudQuery plugin in under 5 minutes by running `dp init`, `dp run`, `dp run --sync`, and `dp run --sync --destination postgresql` in sequence.
- **SC-002**: The full end-to-end test (`dp init -t cloudquery -l python foo && cd foo && dp run && dp run --sync && dp run --sync --destination postgresql`) completes successfully with zero errors.
- **SC-003**: The scaffolded Python plugin's unit tests pass when run via `dp test` with no modifications.
- **SC-004**: The Python plugin container image uses a minimal runtime base (distroless or Chainguard), resulting in no unnecessary OS packages in the final image.
- **SC-005**: Python and Go CloudQuery plugins have feature parity for the `dp run`, `dp run --sync`, and `dp run --sync --destination postgresql` workflows — same output format, same CLI UX, same cleanup behavior.

## Assumptions

- The CloudQuery Python SDK (`cloudquery-plugin-sdk`) version >=0.1.52 is compatible with Python 3.12 and provides the `serve`, `plugin`, `schema`, `scheduler`, and `message` modules used in the templates.
- The `gcr.io/distroless/python3-debian12:nonroot` image (or a Chainguard equivalent) supports Python 3.12 and can run the plugin's entrypoint.
- The developer's machine has Docker installed and running; Python 3.12 is NOT required on the host since all execution happens inside containers.
- The k3d dev environment is already running (`dp dev up`) before `dp run` is invoked.
- The existing destination plugin flow (OCI pull, pod deploy, port-forward, gRPC sync) works identically regardless of whether the source plugin is Python or Go.
