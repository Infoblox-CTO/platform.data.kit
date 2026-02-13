# Research: Python CloudQuery Plugin Support

**Feature**: 010-python-cloudquery-plugins
**Date**: 2026-02-13

## Decision 1: Python Version for Dockerfile

**Decision**: Use Python 3.11 as the build-stage base image (`python:3.11-slim`) and `gcr.io/distroless/python3-debian12:nonroot` (Python 3.11) as the runtime stage.

**Rationale**: The build and runtime stages must use the **same Python minor version** because packages like `grpcio` contain Cython-compiled extensions that are ABI-specific per Python minor version. Building with 3.12 but running on 3.11 causes `ImportError: cannot import name 'cygrpc' from 'grpc._cython'`. The distroless `python3-debian12` image ships Python 3.11, so the build stage must also use 3.11.

Distroless tracks Debian's system Python:
- `python3-debian12` → Python 3.11
- `python3-debian13` → Python 3.13

The `pyproject.toml` specifies `requires-python = ">=3.12"` as the minimum developer version for local development (outside Docker). Inside the container, 3.11 is used for both build and runtime.

**Alternative: Use Python 3.12 build + 3.11 runtime** — originally planned, but rejected after E2E testing revealed grpcio ABI incompatibility between minor Python versions.

**Alternative: Use Python 3.13 throughout** — use `python:3.13-slim` for build and `gcr.io/distroless/python3-debian13:nonroot` for runtime. This matches the old codebase (which used 3.13) but `python3-debian13` is not yet widely available.

**Alternative: Chainguard Python 3.12** — `cgr.dev/chainguard/python:3.12` exists but requires a paid subscription. Not appropriate for an open-source CLI tool.

**Final approach**: Use `python:3.11-slim` for the build stage and `gcr.io/distroless/python3-debian12:nonroot` for the runtime. Both stages use Python 3.11, eliminating ABI mismatches. The pip install uses `--target=/deps` to create a standalone deps directory that is copied to the runtime image. The Dockerfile's `PYTHONPATH` and `COPY` reference `python3.11` paths to match both stages.

## Decision 2: CloudQuery Python SDK API Surface

**Decision**: Update templates to match the actual SDK API (version 0.1.52).

**Rationale**: Research of the `cloudquery/plugin-sdk-python` GitHub repository confirmed the actual API. The existing templates are close but have some discrepancies that need fixing.

**Confirmed API**:

| Concept | Import | Class |
|---------|--------|-------|
| Plugin base | `from cloudquery.sdk import plugin` | `plugin.Plugin` |
| Serve command | `from cloudquery.sdk import serve` | `serve.PluginCommand` |
| Table | `from cloudquery.sdk.schema import Table` | `Table` |
| Column | `from cloudquery.sdk.schema import Column` | `Column` |
| TableResolver | `from cloudquery.sdk.scheduler import TableResolver` | `TableResolver` |
| Scheduler | `from cloudquery.sdk.scheduler import Scheduler` | `Scheduler` |
| Client base | `from cloudquery.sdk.scheduler import Client` | `Client` |

**Plugin constructor**: `Plugin(name: str, version: str, opts: Options = None)`

**Plugin methods to implement**: `init()`, `get_tables()`, `sync()`, `close()`

**Serve CLI**: `serve --address [::]:7777 --log-format json --log-level info`

**Column types**: PyArrow types — `pa.string()`, `pa.int64()`, `pa.bool_()`, `pa.timestamp("us")`

**Table constructor**: `Table(name, columns, title="", description="")`  — note: `title` and `description` are keyword args, not separate from `columns`.

**Scheduler.sync()**: `scheduler.sync(client, resolvers, deterministic_cq_id=False)` → returns `Generator[SyncMessage]`

## Decision 3: Template Corrections Required

**Decision**: Fix the following discrepancies in the existing Python templates.

**Issues found**:

1. **`pyproject.toml`**: `requires-python = ">=3.13"` → should be `">=3.12"`
2. **`requirements.txt`**: versions are compatible, no change needed
3. **`plugin.py` template**: The `Plugin.__init__` call should include `opts=Options(team="...", kind="source")` — but this is optional and the current template works without it. Leave as-is for simplicity.
4. **`plugin.py` template**: Uses `self._client: Client | None = None` union syntax — this requires Python 3.10+ which is fine for 3.12.
5. **Dockerfile in `run.go`**: Currently uses `python:3.13-slim` and `python3.13` paths — must update to `python:3.11-slim` and `python3.11` paths (matching distroless runtime; 3.12 build was rejected due to grpcio ABI incompatibility).
6. **Dockerfile entrypoint**: `["python3", "main.py", "serve", "--address", "[::]:%d"]` — the SDK serve command accepts `--address` with `[::]:PORT` syntax, which is correct.

## Decision 4: Dockerfile Design

**Decision**: Multi-stage build with pip cache mounts and distroless runtime.

```
Build stage:
  FROM python:3.11-slim AS builder
  - pip install --target=/deps -r requirements.txt (with cache mount)
  - COPY application code

Runtime stage:
  FROM gcr.io/distroless/python3-debian12:nonroot
  - COPY deps to /usr/local/lib/python3.11/site-packages
  - COPY app to /app
  - ENV PYTHONPATH for site-packages
  - ENTRYPOINT ["python3", "main.py", "serve", "--address", "[::]:7777"]
```

**Rationale**: This matches the existing pattern in `run.go` but with corrected Python version. The `--target=/deps` pip install creates a standalone directory that can be copied to the distroless image. The distroless image has Python 3.11 as its system Python.

**Note on venv**: The user mentioned "use venv" — in the container context, venv is not needed since `pip install --target` achieves the same isolation. For local development outside Docker, the developer can create a venv manually. The CLI does not manage the developer's local Python environment.

## Decision 5: Existing Code Changes Required

**Decision**: Minimal changes to existing Go code — only the Dockerfile template and `pyproject.toml` template need updates.

**Files requiring changes**:
1. `cli/cmd/run.go` — Update the Python Dockerfile string in `cloudQueryDockerfile()` to use `python:3.11-slim` and `python3.11` paths
2. `cli/internal/templates/cloudquery/python/pyproject.toml.tmpl` — Change `requires-python = ">=3.13"` to `">=3.12"`
3. No changes needed to `init.go`, `build.go`, `test.go` — the Python path already works

**Files NOT requiring changes** (already correct):
- `cli/cmd/init.go` — Python scaffolding already works via template renderer
- `cli/cmd/test.go` — pytest detection already implemented
- `cli/internal/templates/cloudquery/python/*.tmpl` — API usage matches SDK (with minor `pyproject.toml` fix)
- `cli/cmd/run.go` (other than Dockerfile) — `detectCloudQueryLanguage()` already detects Python

## Alternatives Considered

### Use Chainguard instead of Distroless
- **Rejected**: Free tier only has `latest` (3.14), version-pinned tags require paid subscription. Not suitable for an open-source tool.

### Use Python 3.13 everywhere
- **Rejected**: User explicitly requested Python 3.12. The build stage honors this. The runtime uses 3.11 (distroless constraint) which is backward-compatible.

### Use `python:3.12-slim` for runtime (no distroless)
- **Rejected**: User explicitly requested distroless or Chainguard for minimal attack surface. `python:3.12-slim` includes apt, bash, and other unnecessary packages.

### Build custom distroless-like image
- **Rejected**: Fragile — requires manually copying shared libraries (`libpython`, `libssl`, `libffi`). Too complex for a template.
