---
title: "CloudQuery Go Plugin"
description: "Build a Go CloudQuery source plugin from scratch"
---

# CloudQuery Go Plugin

Build a Go-based CloudQuery source plugin that fetches data from an API and syncs it to file or PostgreSQL destinations.

**Time**: ~15 minutes  
**Difficulty**: Beginner  
**Prerequisites**: DP CLI installed, Go 1.25+, Docker running, `dp dev up` completed

## Overview

You'll learn the full developer lifecycle for a Go CloudQuery plugin:

1. **Scaffold** — Generate a working plugin project
2. **Understand** — Explore the project structure
3. **Test** — Run unit tests locally
4. **Run** — Build and deploy to the local k3d cluster
5. **Sync** — Extract data to file and PostgreSQL destinations
6. **Develop** — Add tables and resolvers for your own data source

## 1. Scaffold a Go Plugin

```bash
dp init my-source --runtime cloudquery
cd my-source
```

This creates a complete Go plugin project:

```text
my-source/
├── .gitignore                       # Go-specific ignore patterns
├── .datakit/
│   └── Makefile.common              # Managed targets (do not edit)
├── Makefile                         # Project Makefile (add your own targets here)
├── dp.yaml                          # Package manifest
├── go.mod                           # Go module definition
├── main.go                          # Plugin entry point
├── resources/
│   └── plugin/
│       └── plugin.go                # Plugin factory + Configure function
└── internal/
    ├── client/
    │   ├── client.go                # Client (Tables, Sync, Close)
    │   └── spec.go                  # Config deserialization
    └── tables/
        ├── example_resource.go      # Table schema + resolver
        └── example_resource_test.go # Unit tests
```

## 2. Understand the Project

### dp.yaml — Package Manifest

```bash
cat dp.yaml
```

The manifest declares this as a **Transform** with the CloudQuery runtime:

```yaml
apiVersion: data.infoblox.com/v1alpha1
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

The `kind: Transform` with `runtime: cloudquery` tells DP that this package is a CloudQuery plugin.
The `inputs` and `outputs` declare the assets this transform reads from and writes to.
No container `image` is required — plugin images come from the Connector manifest.

### internal/tables/example_resource.go — Table Definition

Each table defines its **schema** (columns and Arrow types) and a **resolver** function that fetches rows:

```go
func ExampleResourceTable() *schema.Table {
    return &schema.Table{
        Name:     "example_resource",
        Resolver: fetchExampleResource,
        Columns: []schema.Column{
            {Name: "id",     Type: arrow.BinaryTypes.String,       Resolver: schema.PathResolver("ID")},
            {Name: "name",   Type: arrow.BinaryTypes.String,       Resolver: schema.PathResolver("Name")},
            {Name: "active", Type: arrow.FixedWidthTypes.Boolean,  Resolver: schema.PathResolver("Active")},
        },
    }
}

func fetchExampleResource(_ context.Context, _ schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
    resources := []ExampleResource{
        {ID: "res-001", Name: "example-1", Active: true},
        {ID: "res-002", Name: "example-2", Active: false},
    }
    for _, r := range resources {
        res <- r
    }
    return nil
}
```

### internal/client/client.go — Client

The client implements `plugin.Client` and wires together tables, the scheduler, and the sync loop:

```go
func (c *Client) Tables(ctx context.Context, opts plugin.TableOptions) (schema.Tables, error) {
    return []*schema.Table{tables.ExampleResourceTable()}, nil
}

func (c *Client) Sync(ctx context.Context, options plugin.SyncOptions, res chan<- message.SyncMessage) error {
    tt, _ := c.Tables(ctx, plugin.TableOptions{})
    return c.scheduler.Sync(ctx, c, tt, res, ...)
}
```

### resources/plugin/plugin.go — Plugin Factory

```go
func Plugin() *plugin.Plugin {
    return plugin.NewPlugin(Name, Version, Configure,
        plugin.WithKind(Kind),
        plugin.WithTeam(Team),
    )
}

func Configure(ctx context.Context, logger zerolog.Logger, specBytes []byte, opts plugin.NewClientOptions) (plugin.Client, error) {
    return client.New(logger, specBytes, opts), nil
}
```

### Makefile — Common Targets

Every scaffolded project includes a `Makefile` with standard targets. Run `make` or `make help` to see them:

```text
$ make
Usage: make <target>

Targets:
  build                Build the plugin container image
  clean                Remove build artifacts and sync output
  fmt                  Format Go source code
  help                 Show this help message
  lint                 Run dp lint on the package
  run                  Build and deploy to k3d, discover tables
  sync                 Run a full sync to local files
  sync-pg              Run a full sync to PostgreSQL
  test                 Run unit tests
  tidy                 Run go mod tidy
  vet                  Run go vet
```

The `Makefile` includes `.datakit/Makefile.common` which is **managed by the dp CLI** — do not edit it. It is automatically kept in sync when you run `dp build` or `dp run`. Add your own targets to the root `Makefile` using `##` comments so they appear in `make help`:

```makefile
# In your Makefile:
my-target: ## My custom description
	echo "hello"
```

## 3. Run Tests

### Using the DP CLI (recommended)

```bash
dp test
```

For Go plugins, `dp test` runs `go test ./... -v`:

```text
------------------------------------------------------------
CloudQuery Unit Tests
------------------------------------------------------------

=== RUN   TestExampleResourceTable
--- PASS: TestExampleResourceTable (0.00s)
PASS

✓ CloudQuery unit tests PASSED
```

### Without the CLI

```bash
go test ./... -v
```

This is exactly what `dp test` does. No additional setup needed — Go's toolchain handles everything.

## 4. Build and Run the Plugin

```bash
dp run
```

This builds a Docker container using a multi-stage Dockerfile (Go 1.25 builder → distroless static runtime), imports it into the k3d cluster, and starts the plugin:

```text
Building CloudQuery plugin image: default/my-source:latest (lang=go)
...
Discovered 1 table(s):

  example_resource
    An example table demonstrating the CloudQuery plugin pattern.
    Columns:
      id      utf8
      name    utf8
      active  bool

✓ CloudQuery plugin is working correctly
```

## 5. Sync Data

### Sync to local files

```bash
dp run --sync
```

Writes JSON output to `./cq-sync-output/`:

```text
✓ Sync completed: my-source → file
  Output directory: ./cq-sync-output/
```

### Sync to PostgreSQL

```bash
dp run --sync --destination postgresql
```

Uses the PostgreSQL instance running in the k3d dev environment:

```text
✓ Sync completed: my-source → postgresql
```

Verify the data:

```bash
kubectl exec -it deploy/dp-postgres-postgres -n dp-local -- \
  psql -U postgres -c "SELECT * FROM example_resource;"
```

## 6. Develop Your Own Tables

### Add a new table

Create `internal/tables/users.go`:

```go
package tables

import (
    "context"

    "github.com/apache/arrow-go/v18/arrow"
    "github.com/cloudquery/plugin-sdk/v4/schema"
)

func UsersTable() *schema.Table {
    return &schema.Table{
        Name:        "users",
        Description: "Users from the external API.",
        Resolver:    fetchUsers,
        Columns: []schema.Column{
            {Name: "id",    Type: arrow.BinaryTypes.String,    Resolver: schema.PathResolver("ID")},
            {Name: "email", Type: arrow.BinaryTypes.String,    Resolver: schema.PathResolver("Email")},
        },
    }
}

type User struct {
    ID    string
    Email string
}

func fetchUsers(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
    // Replace with actual API calls
    res <- User{ID: "u-001", Email: "alice@example.com"}
    res <- User{ID: "u-002", Email: "bob@example.com"}
    return nil
}
```

### Register it in client.go

```go
func (c *Client) Tables(_ context.Context, _ plugin.TableOptions) (schema.Tables, error) {
    return []*schema.Table{
        tables.ExampleResourceTable(),
        tables.UsersTable(),  // new
    }, nil
}
```

### Add tests

Create `internal/tables/users_test.go`:

```go
package tables

import "testing"

func TestUsersTable(t *testing.T) {
    table := UsersTable()
    if table.Name != "users" {
        t.Errorf("table name = %q, want %q", table.Name, "users")
    }
    if len(table.Columns) != 2 {
        t.Fatalf("expected 2 columns, got %d", len(table.Columns))
    }
}
```

### Test and run

```bash
dp test                              # Run unit tests
dp run                               # Verify table discovery
dp run --sync                        # Sync data to files
dp run --sync --destination postgresql  # Sync to PostgreSQL
```

## Command Reference

| Command | What it does | Needs Docker/k3d? |
|---------|--------------|-------------------|
| `dp init <name> --runtime cloudquery` | Scaffold a new Go plugin | No |
| `dp test` | Run `go test ./... -v` | No |
| `dp run` | Build container, deploy to k3d, discover tables | Yes |
| `dp run --sync` | Sync data to local JSON files | Yes |
| `dp run --sync --destination postgresql` | Sync data to PostgreSQL | Yes |
| `dp test --integration` | Full build + sync integration test | Yes |
| `make` | Show all available Make targets | No |
| `make test` | Run `go test ./... -v` (same as `dp test`) | No |
| `make sync` | Build + sync to local files | Yes |

## Python vs Go Comparison

| Aspect | Go | Python |
|--------|----|----|
| SDK | `plugin-sdk/v4` | `cloudquery-plugin-sdk` |
| Table types | Apache Arrow Go | PyArrow |
| Resolver pattern | `func(ctx, meta, parent, res chan<-)` | `def resolve(self, client, parent)` → yield |
| Test runner | `go test` | pytest |
| Build image | `golang:1.25-alpine` | `python:3.11-slim` |
| Runtime image | `distroless/static-debian12` | `distroless/python3-debian12` |
| Local deps | Go 1.25+ (auto-managed) | Python 3.12+ + venv (auto-created by `dp test`) |

## Next Steps

- [Python CloudQuery Plugin](cloudquery-python.md) — Same workflow in Python
- [Promoting Packages](promoting-packages.md) — Deploy to environments
- [CLI Reference](../reference/cli.md) — All available commands
