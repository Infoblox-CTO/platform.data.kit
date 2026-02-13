# Quickstart: CloudQuery Plugin Package

Get a CloudQuery source plugin running in under 5 minutes.

## Prerequisites

- `dp` CLI installed
- Docker running
- `cloudquery` CLI installed ([installation guide](https://docs.cloudquery.io/docs/quickstart))
- Local dev environment running (`dp dev`)

## Step 1: Scaffold a Python Source Plugin

```bash
mkdir my-source && cd my-source
dp init --type cloudquery --name my-source --namespace acme
```

This creates a complete project:

```
my-source/
├── dp.yaml                          # Package manifest
├── main.py                          # Plugin entry point (gRPC server)
├── Dockerfile                       # Container image
├── pyproject.toml                   # Python project config
├── requirements.txt                 # Python dependencies
├── plugin/
│   ├── __init__.py
│   ├── plugin.py                    # Plugin class (init, get_tables, sync)
│   ├── client.py                    # API client (auth, HTTP session)
│   ├── spec.py                      # Plugin config dataclass
│   └── tables/
│       ├── __init__.py
│       └── example_resource.py      # Example table + resolver
└── tests/
    └── test_example_resource.py     # Unit tests
```

The default language is Python. For Go, add `--lang go`.

## Step 2: Run Unit Tests

```bash
dp test
```

All generated tests pass immediately — no setup needed.

## Step 3: Run the Plugin (End-to-End Sync)

```bash
dp run
```

This will:
1. Build the plugin container
2. Start it as a gRPC server on port 7777
3. Generate a sync config targeting your local PostgreSQL
4. Run `cloudquery sync` to fetch data into PostgreSQL
5. Display a sync summary (tables, rows, errors)

## Step 4: Customize Your Plugin

Edit `plugin/tables/example_resource.py` to add your own tables, columns, and resolvers:

```python
class MyApiTable(Table):
    def __init__(self):
        super().__init__(
            name="my_api_resource",
            columns=[
                Column(name="id", type=pa.string(), primary_key=True, not_null=True),
                Column(name="created_at", type=pa.timestamp("us")),
                Column(name="status", type=pa.string()),
            ],
            title="My API Resource",
            description="Resources fetched from My API",
        )
```

Edit `plugin/client.py` to add your API authentication:

```python
class Client:
    def __init__(self, spec):
        self._spec = spec
        self._session = requests.Session()
        self._session.headers["Authorization"] = f"Bearer {spec.api_token}"

    def id(self):
        return "my-api"
```

## Step 5: Lint, Build, and Publish

```bash
dp lint                            # Validate manifest
dp build                           # Build OCI artifact
dp publish                         # Push to registry
dp promote --env staging           # Promote to staging
```

## Go Plugin Alternative

```bash
mkdir my-source && cd my-source
dp init --type cloudquery --lang go --name my-source --namespace acme
```

Creates a Go project with the CloudQuery Go SDK (`plugin-sdk/v4`), multi-stage Dockerfile, and example table.

```bash
dp test    # Runs go test ./...
dp run     # Builds container, starts gRPC, runs cloudquery sync
```

## What's Next

- Add more tables in `plugin/tables/` (Python) or `internal/tables/` (Go)
- Configure your plugin spec in `plugin/spec.py` or `internal/client/spec.go`
- Run `dp test --integration` for full end-to-end validation
- Check `dp lint` output for manifest issues
