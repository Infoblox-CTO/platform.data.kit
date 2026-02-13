# Research: CloudQuery Plugin Package Type

**Feature**: 008-cloudquery-plugins  
**Date**: 2026-02-12

## R-001: CloudQuery Python SDK API Surface

**Decision**: Use `cloudquery-plugin-sdk` (PyPI) with the Plugin subclass pattern  
**Rationale**: This is the official CloudQuery Python SDK. The API is stable (v0.1.52+) and follows a clear pattern: subclass `plugin.Plugin`, implement `init/get_tables/sync/close`, use `serve.PluginCommand` to start the gRPC server.  
**Alternatives considered**:
- Raw gRPC stubs: Rejected — too low-level, SDK handles protocol negotiation
- Custom plugin framework: Rejected — unnecessary abstraction over a well-designed SDK

**Key API findings**:
- Entry point: `serve.PluginCommand(plugin_instance).run(sys.argv[1:])`
- Plugin base: `cloudquery.sdk.plugin.Plugin` — constructor takes `(name, version, opts=Options(team, kind))`
- Required overrides: `init(spec, no_connection)`, `get_tables(options)`, `sync(options)`, `close()`
- Table definition: Subclass `cloudquery.sdk.schema.Table` — constructor takes `(name, columns, title, description)`
- Column types: `pyarrow` types — `pa.string()`, `pa.int64()`, `pa.bool_()`, `pa.float64()`, `pa.timestamp("us")`
- Resolver: Subclass `cloudquery.sdk.scheduler.TableResolver` — implement `resolve(client, parent_resource)` as a generator yielding dicts
- Scheduler: `cloudquery.sdk.scheduler.Scheduler(concurrency=N)` — `sync(client, resolvers)` returns message generator
- Client: Any class with `id()` method — holds API auth/HTTP session
- Spec: Plain dataclass for config deserialization
- Default serve address: `[::]:7777`
- CLI args: `serve --address [::]:7777 --log-format json --log-level info`
- Dependencies: `cloudquery-plugin-sdk>=0.1.52`, `pyarrow>=23.0.0`

## R-002: CloudQuery Go SDK API Surface

**Decision**: Use `github.com/cloudquery/plugin-sdk/v4` with the `plugin.Client` interface pattern  
**Rationale**: This is the official CloudQuery Go SDK (v4 series). The API uses a functional constructor pattern with `plugin.NewPlugin` and a `NewClientFunc` callback.  
**Alternatives considered**:
- Older v3 SDK: Rejected — v4 is the current stable series
- Interface-first approach (client struct directly): The SDK's `NewPlugin + Configure func` pattern is idiomatic

**Key API findings**:
- Entry point: `serve.Plugin(p).Serve(ctx)` where `p = plugin.NewPlugin(name, version, ConfigureFunc, opts...)`
- Plugin constructor: `plugin.NewPlugin(name, version, newClientFunc, plugin.WithKind("source"), plugin.WithTeam("team"))`
- Configure function: `func(ctx, zerolog.Logger, []byte, plugin.NewClientOptions) (plugin.Client, error)` — creates Client from JSON spec
- Client interface: `Tables(ctx, TableOptions)`, `Sync(ctx, SyncOptions, chan<- message.SyncMessage)`, `Close(ctx)`, plus `plugin.UnimplementedDestination` for stubs
- Table definition: `*schema.Table` struct literal with `Name`, `Description`, `Resolver`, `Columns`
- Column types: `arrow.BinaryTypes.String`, `arrow.PrimitiveTypes.Int64`, `arrow.FixedWidthTypes.Boolean`
- Resolver: `func(ctx context.Context, meta schema.ClientMeta, parent *schema.Resource, res chan<- any) error`
- Column resolver: `schema.PathResolver("field_name")` — auto-resolves from struct/map keys
- Scheduler: `scheduler.NewScheduler(opts...).Sync(ctx, client, tables, resCh)`
- Spec: Struct with `SetDefaults()` and `Validate() error` methods
- Default serve address: `[::]:7777`
- Dependencies: `github.com/cloudquery/plugin-sdk/v4`, `github.com/apache/arrow-go/v18`, `github.com/rs/zerolog`
- Project layout convention: `resources/plugin/plugin.go` (constructor), `internal/client/client.go` + `spec.go`, `internal/tables/*.go`

## R-003: Directory-Based Template Scaffolding

**Decision**: Implement a directory-based template system alongside the existing single-file template system  
**Rationale**: CloudQuery plugins require 10+ files across multiple directories. The current `renderer.go` uses `//go:embed *.tmpl` and `RenderToFile()` for individual templates. We need a complementary approach that can scaffold entire directory trees.  
**Alternatives considered**:
- Multiple `RenderToFile()` calls from init.go: Rejected — puts all scaffolding logic in the command layer; hard to maintain 10+ file paths
- External tool (cookiecutter, copier): Rejected — adds external dependency; violates constitution pragmatism
- Single tar/zip embedded archive: Rejected — hard to template individual files

**Implementation approach**:
- Create `cli/internal/templates/cloudquery/` directory with subdirectories for `python/` and `go/`
- Each subdirectory mirrors the output project structure with `.tmpl` files
- Add a new `RenderDirectory(outputDir, templateDir, config)` method to the `Renderer` that walks the embedded template tree, creates directories, and renders each `.tmpl` file
- Use `//go:embed cloudquery/**/*.tmpl` to embed the tree (separate from existing `*.tmpl`)
- Template filenames: strip `.tmpl` suffix → becomes the output filename
- Template directory structure matches the output directory structure exactly
- The `PackageConfig` struct gains new fields: `Role`, `GRPCPort`, `Concurrency`, `Tables`

**Template tree (embedded)**:
```
cli/internal/templates/
├── *.tmpl                           # Existing single-file templates (pipeline)
├── renderer.go                      # Existing + new RenderDirectory method
└── cloudquery/
    ├── python/
    │   ├── dp.yaml.tmpl
    │   ├── main.py.tmpl
    │   ├── Dockerfile.tmpl
    │   ├── pyproject.toml.tmpl
    │   ├── requirements.txt.tmpl
    │   ├── plugin/__init__.py.tmpl
    │   ├── plugin/plugin.py.tmpl
    │   ├── plugin/client.py.tmpl
    │   ├── plugin/spec.py.tmpl
    │   ├── plugin/tables/__init__.py.tmpl
    │   ├── plugin/tables/example_resource.py.tmpl
    │   └── tests/test_example_resource.py.tmpl
    └── go/
        ├── dp.yaml.tmpl
        ├── main.go.tmpl
        ├── Dockerfile.tmpl
        ├── go.mod.tmpl
        ├── resources/plugin/plugin.go.tmpl
        ├── internal/client/client.go.tmpl
        ├── internal/client/spec.go.tmpl
        ├── internal/tables/example_resource.go.tmpl
        └── internal/tables/example_resource_test.go.tmpl
```

## R-004: dp run for CloudQuery Packages

**Decision**: Add a CloudQuery-specific run path in `cli/cmd/run.go` that detects `type: cloudquery` and orchestrates: build → start gRPC container → generate sync config → exec `cloudquery sync` → display summary  
**Rationale**: The existing run command dispatches by pipeline mode (batch/streaming). CloudQuery requires a fundamentally different execution model: the plugin is a gRPC server that the `cloudquery` CLI orchestrates. We cannot reuse the pipeline runner directly.  
**Alternatives considered**:
- Reuse existing DockerRunner with post-run hook: Rejected — CloudQuery plugins don't "run to completion"; they serve gRPC and the `cloudquery` CLI drives the sync
- New top-level `dp sync` command: Rejected — violates the unified `dp run` UX that users expect
- Embed `cloudquery` as a Go library: Rejected — CloudQuery is a standalone CLI, embedding it would be fragile

**Implementation approach**:
- In `run.go`, after parsing `dp.yaml`, check `spec.Type`. If `cloudquery`, call a new `runCloudQuery()` function
- `runCloudQuery()` flow:
  1. Check `cloudquery` binary exists on PATH (exec.LookPath) → fail with install instructions if missing
  2. Build the plugin container image (reuse existing Docker build logic)
  3. Start the container with port mapping (host:7777 → container:7777) in detached mode
  4. Wait for gRPC health (TCP connect to port with timeout)
  5. Generate a temporary sync config YAML: source plugin pointing to `grpc://localhost:7777`, destination pointing to local PostgreSQL from `dp dev`
  6. Run `cloudquery sync <temp-config>` and stream output
  7. Parse sync output for summary (tables, rows, errors)
  8. Stop and remove the plugin container
  9. Display formatted summary

## R-005: dp.yaml Manifest Extension for CloudQuery

**Decision**: Add `spec.cloudquery` section to `DataPackageSpec` as an optional struct, present only when `type: cloudquery`  
**Rationale**: This follows the existing pattern where type-specific fields are optional (e.g., pipeline packages use `spec.schedule`, `spec.runtime`, `spec.inputs/outputs`). A dedicated `cloudquery` section cleanly separates CQ-specific config from the generic envelope.  
**Alternatives considered**:
- Flatten CQ fields into the top-level spec: Rejected — pollutes the shared spec namespace
- Separate file (cloudquery.yaml): Rejected — adds complexity; pipeline mode already embeds mode-specific fields in dp.yaml

**Schema**:
```yaml
spec:
  cloudquery:
    role: source          # required: "source" | "destination" (destination reserved)
    tables:               # optional: list of table names the plugin provides
      - example_resource
    grpcPort: 7777        # optional: gRPC serve port (default: 7777)
    concurrency: 10000    # optional: max concurrent table resolvers (default: 10000)
```

## R-006: Validation Strategy for CloudQuery Packages

**Decision**: Add CloudQuery type to the `validTypes` allowlist in `sdk/validate/datapackage.go` and implement type-specific validation rules  
**Rationale**: The existing validation framework dispatches by type. We follow the same pattern.  
**Alternatives considered**:
- Separate validator binary: Rejected — unnecessary complexity
- JSON Schema only: Rejected — runtime validation already exists in Go; schema validates structure but not semantics (e.g., "destination not yet supported")

**Rules**:
- `spec.cloudquery` section required when `type: cloudquery`
- `spec.cloudquery.role` required, must be `source` or `destination`
- `spec.cloudquery.role: destination` triggers warning (not yet supported)
- `spec.cloudquery.grpcPort` must be 1–65535 if provided
- `spec.runtime.image` required for cloudquery type
- `spec.outputs` NOT required for cloudquery (unlike pipeline) — CQ plugins produce tables, not explicit output contracts

## R-007: Build/Publish/Promote Reuse

**Decision**: CloudQuery packages reuse the existing OCI artifact pipeline for build/publish/promote with zero changes  
**Rationale**: The build/publish/promote commands operate on the container image + dp.yaml manifest. They don't inspect package type — they call validation (which we extend) and then package the artifact. No type-specific logic needed.  
**Alternatives considered**: None — this is the correct approach per the existing architecture.
