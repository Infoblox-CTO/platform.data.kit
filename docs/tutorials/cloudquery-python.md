---
title: "CloudQuery Python Plugin"
description: "Build a Python CloudQuery source plugin from scratch"
---
# CloudQuery Python Plugin

Build a Python-based CloudQuery source plugin that fetches data from an API and syncs it to file or PostgreSQL destinations.

**Time**: ~15 minutes
**Difficulty**: Beginner
**Prerequisites**: DK CLI installed, Docker running, `dk dev up` completed

## Overview

You'll learn the full developer lifecycle for a Python CloudQuery plugin:

1. **Scaffold** — Generate a working plugin project
2. **Understand** — Explore the project structure
3. **Test** — Run unit tests locally
4. **Run** — Build and deploy to the local k3d cluster
5. **Sync** — Extract data to file and PostgreSQL destinations
6. **Develop** — Edit tables and resolvers for your own data source

## 1. Scaffold a Python Plugin

```bash
dk init my-source --runtime cloudquery
cd my-source
```

This creates a complete Python plugin project:

```text
my-source/
├── .gitignore                 # Python-specific ignore patterns
├── .datakit/
│   └── Makefile.common        # Managed targets (do not edit)
├── Makefile                   # Project Makefile (add your own targets here)
├── dk.yaml                    # Package manifest
├── main.py                    # gRPC server entry point
├── requirements.txt           # pip dependencies (used by Docker build)
├── pyproject.toml             # Python project metadata + test config
├── plugin/
│   ├── __init__.py
│   ├── plugin.py              # Plugin class (init, get_tables, sync, close)
│   ├── client.py              # API client stub
│   ├── spec.py                # JSON config → dataclass
│   └── tables/
│       ├── __init__.py
│       └── example_resource.py  # Table schema + resolver
└── tests/
    └── test_example_resource.py  # Unit tests
```

## 2. Understand the Project

### dk.yaml — Package Manifest

```bash
cat dk.yaml
```

The manifest declares this as a **Transform** with the CloudQuery runtime:

```yaml
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: my-source
  namespace: default
  version: 0.1.0
  labels:
    team: my-team
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - asset: my-source-source-table
  outputs:
    - asset: my-source-dest-table
  timeout: 30m
```

The `kind: Transform` with `runtime: cloudquery` tells DK that this package is a CloudQuery plugin.
The `inputs` and `outputs` declare the assets this transform reads from and writes to.
No container `image` is required — plugin images come from the Connector manifest.

### plugin/tables/example_resource.py — Table Definition

Each table defines its **schema** (columns and types) and a **resolver** that fetches rows:

```python
class ExampleResourceTable(schema.Table):
    def __init__(self):
        super().__init__(
            name="example_resource",
            columns=[
                schema.Column(name="id", type=pa.string(), primary_key=True),
                schema.Column(name="name", type=pa.string()),
                schema.Column(name="value", type=pa.int64()),
                schema.Column(name="active", type=pa.bool_()),
            ],
        )

class ExampleResourceResolver(TableResolver):
    def resolve(self, client, parent_resource):
        yield {"id": "example-1", "name": "Example Item 1", "value": 100, "active": True}
        yield {"id": "example-2", "name": "Example Item 2", "value": 200, "active": False}
```

### plugin/plugin.py — Plugin Class

The plugin wires together tables, resolvers, and the scheduler:

```python
class mySourcePlugin(plugin.Plugin):
    def init(self, spec, no_connection=False):
        # Parse config, create API client
    def get_tables(self, options):
        # Return list of table schemas
    def sync(self, options):
        # Schedule resolvers, yield sync messages
```

### pyproject.toml — Python Metadata

```toml
[project]
requires-python = ">=3.12"
dependencies = [
    "cloudquery-plugin-sdk>=0.1.52",
    "pyarrow>=23.0.0",
]

[project.optional-dependencies]
dev = ["pytest>=8.0"]

[tool.pytest.ini_options]
pythonpath = ["."]
```

### Makefile — Common Targets

Every scaffolded project includes a `Makefile` with standard targets. Run `make` or `make help` to see them:

```text
$ make
Usage: make <target>

Targets:
  build                Build the plugin container image
  clean                Remove build artifacts, venv, and sync output
  fmt                  Format Python source code with black
  help                 Show this help message
  lint                 Run dk lint on the package
  run                  Build and deploy to k3d, discover tables
  sync                 Run a full sync to local files
  sync-pg              Run a full sync to PostgreSQL
  test                 Run unit tests
  typecheck            Run mypy type checking
  venv                 Create virtual environment and install deps
```

The `Makefile` includes `.datakit/Makefile.common` which is **managed by the dk CLI** — do not edit it. It is automatically kept in sync when you run `dk build` or `dk run`. Add your own targets to the root `Makefile` using `## ` comments so they appear in `make help`:

```makefile
# In your Makefile:
my-target: ## My custom description
	echo "hello"
```

## 3. Run Tests

### Using the DK CLI (recommended)

```bash
dk test
```

On the first run, `dk test` automatically:

1. Creates a `.venv/` virtual environment
2. Installs project dependencies (including pytest)
3. Runs `pytest -v`

Subsequent runs reuse the existing `.venv/` and go straight to pytest.

```text
Setting up Python virtual environment...
  Python: /opt/homebrew/bin/python3
  Venv:   .venv/

Installing dependencies...
...
tests/test_example_resource.py::TestExampleResourceTable::test_table_name PASSED
tests/test_example_resource.py::TestExampleResourceTable::test_table_has_columns PASSED
...
✓ CloudQuery unit tests PASSED
```

### Without the CLI

If you prefer to manage your own environment:

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
pytest -v
```

This is exactly what `dk test` does under the hood.

## 4. Build and Run the Plugin

```bash
dk run
```

This builds a Docker container, imports it into the k3d cluster, and starts the plugin:

```text
Building CloudQuery plugin image: default/my-source:latest (lang=python)
...
Discovered 1 table(s):

  example_resource
    An example resource table.
    Columns:
      id      utf8
      name    utf8
      value   int64
      active  bool

✓ CloudQuery plugin is working correctly
```

!!! note "No local Python required"
    `dk run` builds everything inside Docker. You don't need Python installed locally to build and run the plugin — only to run tests.

## 5. Sync Data

### Sync to local files

```bash
dk run --sync
```

Writes JSON output to `./cq-sync-output/`:

```text
✓ Sync completed: my-source → file
  Output directory: ./cq-sync-output/
```

Inspect the output:

```bash
cat cq-sync-output/*.json
```

### Sync to PostgreSQL

```bash
dk run --sync --destination postgresql
```

Uses the PostgreSQL instance running in the k3d dev environment:

```text
✓ Sync completed: my-source → postgresql
```

Verify the data:

```bash
kubectl exec -it deploy/dk-postgres-postgres -n dk-local -- \
  psql -U postgres -c "SELECT * FROM example_resource;"
```

## 6. Develop Your Own Tables

### Add a new table

Create `plugin/tables/users.py`:

```python
import pyarrow as pa
from cloudquery.sdk import schema
from cloudquery.sdk.scheduler import TableResolver


class UsersTable(schema.Table):
    def __init__(self):
        super().__init__(
            name="users",
            title="Users",
            description="Users from the external API",
            columns=[
                schema.Column(name="id", type=pa.string(), primary_key=True),
                schema.Column(name="email", type=pa.string()),
                schema.Column(name="created_at", type=pa.timestamp("us")),
            ],
        )


class UsersResolver(TableResolver):
    def resolve(self, client, parent_resource):
        # Replace with actual API calls using client
        for user in client.list_users():
            yield {
                "id": user["id"],
                "email": user["email"],
                "created_at": user["created_at"],
            }
```

### Register it in plugin.py

```python
from plugin.tables.users import UsersTable, UsersResolver

class mySourcePlugin(plugin.Plugin):
    def __init__(self):
        super().__init__(name="my-source", version="0.1.0")
        self._tables = [
            ExampleResourceTable(),
            UsersTable(),  # new
        ]
        self._resolvers = [
            ExampleResourceResolver(table=self._tables[0]),
            UsersResolver(table=self._tables[1]),  # new
        ]
```

### Add tests

Create `tests/test_users.py`:

```python
from plugin.tables.users import UsersTable

class TestUsersTable:
    def test_table_name(self):
        assert UsersTable().name == "users"

    def test_has_email_column(self):
        cols = [c.name for c in UsersTable().columns]
        assert "email" in cols
```

### Test and run

```bash
dk test                              # Run unit tests
dk run                               # Verify table discovery
dk run --sync                        # Sync data to files
dk run --sync --destination postgresql  # Sync to PostgreSQL
```

## Command Reference

| Command                                    | What it does                                    | Needs Docker/k3d? |
| ------------------------------------------ | ----------------------------------------------- | ----------------- |
| `dk init <name> --runtime cloudquery`    | Scaffold a new plugin                           | No                |
| `dk test`                                | Create venv, install deps, run pytest           | No                |
| `dk run`                                 | Build container, deploy to k3d, discover tables | Yes               |
| `dk run --sync`                          | Sync data to local JSON files                   | Yes               |
| `dk run --sync --destination postgresql` | Sync data to PostgreSQL                         | Yes               |
| `dk test --integration`                  | Full build + sync integration test              | Yes               |
| `make`                                   | Show all available Make targets                 | No                |
| `make test`                              | Create venv + run pytest (same as `dk test`)  | No                |
| `make sync`                              | Build + sync to local files                     | Yes               |

## Troubleshooting

### `dk test` fails with "python3 not found"

Install Python 3.12+. On macOS: `brew install python@3.12`

### `dk run` build fails with import errors

Check `requirements.txt` matches your imports. The Docker build uses `requirements.txt`, not `pyproject.toml`.

### Tests pass locally but `dk run --sync` fails

The Docker container uses Python 3.11 (distroless runtime). Avoid syntax or features exclusive to 3.12+.

## Next Steps

- [Go CloudQuery Plugin](cloudquery-go.md) — Same workflow in Go
- [Promoting Packages](promoting-packages.md) — Deploy to environments
- [CLI Reference](../reference/cli.md) — All available commands
