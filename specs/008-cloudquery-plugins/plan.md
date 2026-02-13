# Implementation Plan: CloudQuery Plugin Package Type

**Branch**: `008-cloudquery-plugins` | **Date**: 2026-02-12 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-cloudquery-plugins/spec.md`

## Summary

Add a new `cloudquery` package type to the Data Platform so that `dp init --type cloudquery` scaffolds a complete, immediately-runnable CloudQuery source plugin (Python or Go), and `dp run/test/lint/build/publish/promote` all work for CloudQuery packages. This requires: a new `PackageTypeCloudQuery` constant and `CloudQuerySpec` struct in contracts, directory-based template scaffolding in the CLI template engine, CloudQuery-specific execution logic in `dp run`, type-aware validation in `dp lint`, and template trees for both Python and Go CloudQuery plugins.

## Technical Context

**Language/Version**: Go 1.25 (CLI + SDK), Python 3.13+ and Go 1.25 (generated plugin targets)  
**Primary Dependencies**: Cobra CLI framework, testify (CLI/SDK tests); cloudquery-plugin-sdk/pyarrow/pytest (Python plugins); github.com/cloudquery/plugin-sdk/v4 (Go plugins)  
**Storage**: N/A (CloudQuery syncs to external destinations; local dev uses PostgreSQL from dp dev)  
**Testing**: `go test` with testify (CLI/SDK), pytest (generated Python plugins), go test (generated Go plugins)  
**Target Platform**: macOS/Linux development machines (CLI), Docker containers (plugins)  
**Project Type**: Go monorepo with multiple modules (cli, sdk, contracts)  
**Performance Goals**: `dp init` scaffolds a project in < 2 seconds; `dp run` overhead (excluding sync time) < 30 seconds  
**Constraints**: `cloudquery` CLI is a user-installed prerequisite; gRPC port 7777 default must not conflict with dp dev services  
**Scale/Scope**: ~25 new template files, ~500 lines of new Go code across CLI/SDK/contracts, 7 modified existing files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Requirement | Status | Notes |
|------|-------------|--------|-------|
| **Workflow Demo** | Plan demonstrates end-to-end developer workflow | ✅ PASS | `dp init` → `dp test` → `dp run` → `dp lint` → `dp build` → `dp publish` → `dp promote` — same flow as pipeline type |
| **Contract Schema** | Contract schemas and validation strategy are explicit | ✅ PASS | `CloudQuerySpec` struct defined in [contracts/go-types.md](contracts/go-types.md); JSON Schema changes in [contracts/dp-manifest-schema-changes.json](contracts/dp-manifest-schema-changes.json); validation error codes E060–E063 defined |
| **Promotion/Rollback** | Promotion and rollback mechanics are explicit | ✅ PASS | Reuses existing pipeline OCI artifact promotion — no changes needed (per research R-007) |
| **Observability** | Observability requirements defined | ✅ PASS | `dp run` displays sync summary (tables, rows, errors); CloudQuery CLI produces structured logs; container logs available via `dp logs` |
| **Security/Compliance** | Secrets, least privilege, PII addressed | ✅ PASS | Plugin config is passed via CloudQuery spec (not env vars with secrets); container runs with existing dp security posture; PII classification carried through dp.yaml outputs if applicable |

## Project Structure

### Documentation (this feature)

```text
specs/008-cloudquery-plugins/
├── spec.md                                    # Feature specification
├── plan.md                                    # This file
├── research.md                                # Phase 0: SDK research, template approach, decisions
├── data-model.md                              # Phase 1: Entity definitions
├── quickstart.md                              # Phase 1: Getting started guide
├── checklists/
│   └── requirements.md                        # Quality checklist
└── contracts/
    ├── go-types.md                            # Go type definitions (CloudQuerySpec, CloudQueryRole)
    ├── dp-manifest-contract.md                # dp.yaml manifest format and validation rules
    ├── dp-manifest-schema-changes.json        # JSON Schema delta for dp-manifest.schema.json
    └── sync-config-contract.md                # Generated CloudQuery sync config format
```

### Source Code (repository root)

```text
contracts/
├── types.go                    # MODIFY: Add PackageTypeCloudQuery constant
├── cloudquery.go               # NEW: CloudQuerySpec, CloudQueryRole types
├── datapackage.go              # MODIFY: Add CloudQuery field to DataPackageSpec
└── schemas/
    └── dp-manifest.schema.json # MODIFY: Add cloudquery type + cloudquerySpec definition

sdk/
└── validate/
    ├── datapackage.go          # MODIFY: Add cloudquery to validTypes, add type-specific rules
    └── cloudquery.go           # NEW: CloudQuery-specific validator (E060–E063)
    └── cloudquery_test.go      # NEW: Validation tests

cli/
├── cmd/
│   ├── init.go                 # MODIFY: Add cloudquery type routing, default lang=python
│   ├── init_test.go            # MODIFY: Add cloudquery scaffolding tests
│   ├── run.go                  # MODIFY: Add cloudquery run path (build → gRPC → sync)
│   ├── run_test.go             # MODIFY: Add cloudquery run tests
│   ├── test.go                 # MODIFY: Add cloudquery test detection (pytest/go test)
│   └── lint.go                 # No changes needed (delegates to validator)
└── internal/
    └── templates/
        ├── renderer.go                                 # MODIFY: Add RenderDirectory, update PackageConfig
        ├── renderer_test.go                            # MODIFY: Add directory rendering tests
        └── cloudquery/                                 # NEW: Directory-based template tree
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

docs/
├── concepts/
│   └── data-packages.md        # MODIFY: Document cloudquery package type
├── getting-started/
│   └── quickstart.md           # MODIFY: Add cloudquery quickstart section
└── reference/
    ├── cli.md                  # MODIFY: Document --type cloudquery flag
    └── manifest-schema.md      # MODIFY: Document spec.cloudquery section
```

**Structure Decision**: This feature extends the existing monorepo structure. No new top-level modules are needed. The primary addition is a `cloudquery/` directory under `cli/internal/templates/` containing the directory-based template trees for Python and Go plugins. All new Go code lives in the existing `contracts`, `sdk`, and `cli` modules.

## Phases

### Phase 1: Contracts & Data Model (Foundation)

**Goal**: Define all types and schemas so the rest of the system can compile against them.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 1.1 | Add `PackageTypeCloudQuery` constant | `contracts/types.go` | All |
| 1.2 | Create `CloudQuerySpec`, `CloudQueryRole` types | `contracts/cloudquery.go` (new) | US-5 |
| 1.3 | Add `CloudQuery *CloudQuerySpec` field to `DataPackageSpec` | `contracts/datapackage.go` | All |
| 1.4 | Update JSON Schema: add `cloudquery` to type enum, add `cloudquerySpec` definition, add conditional `required` | `contracts/schemas/dp-manifest.schema.json` | US-5 |
| 1.5 | Unit tests for `CloudQueryRole.IsValid()`, `IsSupported()`, `Default()` | `contracts/cloudquery_test.go` (new) | US-5 |

### Phase 2: Validation (Lint)

**Goal**: `dp lint` validates cloudquery-specific manifest fields.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 2.1 | Add `PackageTypeCloudQuery` to `validTypes` slice | `sdk/validate/datapackage.go` | US-5 |
| 2.2 | Skip outputs-required check for cloudquery type | `sdk/validate/datapackage.go` | US-5 |
| 2.3 | Add runtime-required check for cloudquery type | `sdk/validate/datapackage.go` | US-5 |
| 2.4 | Create `CloudQueryValidator` with rules E060–E063 | `sdk/validate/cloudquery.go` (new) | US-5 |
| 2.5 | Wire `CloudQueryValidator` into `AggregateValidator` | `sdk/validate/aggregate.go` | US-5 |
| 2.6 | Unit tests for all CloudQuery validation rules | `sdk/validate/cloudquery_test.go` (new) | US-5 |

### Phase 3: Template Engine (Directory-Based Scaffolding)

**Goal**: The template renderer can scaffold entire directory trees, not just single files.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 3.1 | Add `Type`, `Role`, `GRPCPort`, `Concurrency` fields to `PackageConfig` | `cli/internal/templates/renderer.go` | US-1, US-2 |
| 3.2 | Add second `//go:embed cloudquery/**/*.tmpl` FS | `cli/internal/templates/renderer.go` | US-1, US-2 |
| 3.3 | Implement `RenderDirectory(outputDir, templateSubDir string, config PackageConfig)` method | `cli/internal/templates/renderer.go` | US-1, US-2 |
| 3.4 | Unit tests for `RenderDirectory` | `cli/internal/templates/renderer_test.go` | US-1, US-2 |

### Phase 4: Python Plugin Templates

**Goal**: `dp init --type cloudquery --lang python` scaffolds a complete, working Python source plugin.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 4.1 | Create `dp.yaml.tmpl` for Python CloudQuery | `cli/internal/templates/cloudquery/python/dp.yaml.tmpl` | US-1 |
| 4.2 | Create `main.py.tmpl` (entry point) | `cli/internal/templates/cloudquery/python/main.py.tmpl` | US-1 |
| 4.3 | Create `plugin/plugin.py.tmpl` (Plugin subclass) | `cli/internal/templates/cloudquery/python/plugin/plugin.py.tmpl` | US-1 |
| 4.4 | Create `plugin/client.py.tmpl` (Client class) | `cli/internal/templates/cloudquery/python/plugin/client.py.tmpl` | US-1 |
| 4.5 | Create `plugin/spec.py.tmpl` (Spec dataclass) | `cli/internal/templates/cloudquery/python/plugin/spec.py.tmpl` | US-1 |
| 4.6 | Create `plugin/tables/example_resource.py.tmpl` (Table + Resolver) | `cli/internal/templates/cloudquery/python/plugin/tables/example_resource.py.tmpl` | US-1 |
| 4.7 | Create `plugin/__init__.py.tmpl`, `plugin/tables/__init__.py.tmpl` | `cli/internal/templates/cloudquery/python/plugin/*.tmpl` | US-1 |
| 4.8 | Create `tests/test_example_resource.py.tmpl` | `cli/internal/templates/cloudquery/python/tests/test_example_resource.py.tmpl` | US-1 |
| 4.9 | Create `Dockerfile.tmpl` | `cli/internal/templates/cloudquery/python/Dockerfile.tmpl` | US-1 |
| 4.10 | Create `pyproject.toml.tmpl`, `requirements.txt.tmpl` | `cli/internal/templates/cloudquery/python/*.tmpl` | US-1 |

### Phase 5: Go Plugin Templates

**Goal**: `dp init --type cloudquery --lang go` scaffolds a complete, working Go source plugin.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 5.1 | Create `dp.yaml.tmpl` for Go CloudQuery | `cli/internal/templates/cloudquery/go/dp.yaml.tmpl` | US-2 |
| 5.2 | Create `main.go.tmpl` (entry point) | `cli/internal/templates/cloudquery/go/main.go.tmpl` | US-2 |
| 5.3 | Create `resources/plugin/plugin.go.tmpl` (Plugin constructor) | `cli/internal/templates/cloudquery/go/resources/plugin/plugin.go.tmpl` | US-2 |
| 5.4 | Create `internal/client/client.go.tmpl` (Client struct) | `cli/internal/templates/cloudquery/go/internal/client/client.go.tmpl` | US-2 |
| 5.5 | Create `internal/client/spec.go.tmpl` (Spec struct) | `cli/internal/templates/cloudquery/go/internal/client/spec.go.tmpl` | US-2 |
| 5.6 | Create `internal/tables/example_resource.go.tmpl` (Table + Resolver) | `cli/internal/templates/cloudquery/go/internal/tables/example_resource.go.tmpl` | US-2 |
| 5.7 | Create `internal/tables/example_resource_test.go.tmpl` | `cli/internal/templates/cloudquery/go/internal/tables/example_resource_test.go.tmpl` | US-2 |
| 5.8 | Create `Dockerfile.tmpl` (multi-stage) | `cli/internal/templates/cloudquery/go/Dockerfile.tmpl` | US-2 |
| 5.9 | Create `go.mod.tmpl` | `cli/internal/templates/cloudquery/go/go.mod.tmpl` | US-2 |

### Phase 6: CLI Init Command (Wiring)

**Goal**: `dp init --type cloudquery` routes to the correct template tree and scaffolds the project.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 6.1 | Add `"cloudquery"` to `isValidPackageType()` | `cli/cmd/init.go` | US-1, US-2 |
| 6.2 | Default language to `python` when `--type cloudquery` | `cli/cmd/init.go` | US-1 |
| 6.3 | Add `--role` flag (default `source`) | `cli/cmd/init.go` | US-1, US-2 |
| 6.4 | Reject `--role destination` with "not yet supported" message | `cli/cmd/init.go` | Edge case |
| 6.5 | Add cloudquery scaffolding path: call `RenderDirectory("cloudquery/{lang}", ...)` | `cli/cmd/init.go` | US-1, US-2 |
| 6.6 | Remove `--mode` requirement for cloudquery type (mode is pipeline-specific) | `cli/cmd/init.go` | US-1, US-2 |
| 6.7 | Unit tests for cloudquery init (Python and Go) | `cli/cmd/init_test.go` | US-1, US-2 |

### Phase 7: CLI Run Command (CloudQuery Execution)

**Goal**: `dp run` detects `type: cloudquery` and performs the full sync workflow.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 7.1 | Add `cloudquery` binary detection (exec.LookPath) with install instructions | `cli/cmd/run.go` | US-3 |
| 7.2 | Add type detection: if `spec.Type == cloudquery`, call `runCloudQuery()` | `cli/cmd/run.go` | US-3 |
| 7.3 | Implement `runCloudQuery()`: build container, start gRPC, generate sync config, exec `cloudquery sync`, parse output, display summary | `cli/cmd/run.go` | US-3 |
| 7.4 | Implement sync config generation (temp YAML file per [sync-config-contract.md](contracts/sync-config-contract.md)) | `cli/cmd/run.go` | US-3 |
| 7.5 | Add gRPC health check (TCP connect with timeout) | `cli/cmd/run.go` | US-3 |
| 7.6 | Add container cleanup (stop + remove) on completion and on error/interrupt | `cli/cmd/run.go` | US-3 |
| 7.7 | Unit tests for cloudquery run path | `cli/cmd/run_test.go` | US-3 |

### Phase 8: CLI Test Command (CloudQuery Testing)

**Goal**: `dp test` runs unit tests; `dp test --integration` runs full sync.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 8.1 | Add cloudquery type detection in test command | `cli/cmd/test.go` | US-4 |
| 8.2 | Route to `pytest` (Python) or `go test ./...` (Go) for unit tests | `cli/cmd/test.go` | US-4 |
| 8.3 | Implement `--integration` flag for cloudquery: reuse `runCloudQuery()` logic with test reporting | `cli/cmd/test.go` | US-4 |

### Phase 9: Documentation

**Goal**: User-facing docs cover the new cloudquery package type.

| # | Task | File(s) | Story |
|---|------|---------|-------|
| 9.1 | Add cloudquery package type to concepts | `docs/concepts/data-packages.md` | All |
| 9.2 | Add cloudquery quickstart section | `docs/getting-started/quickstart.md` | All |
| 9.3 | Document `--type cloudquery` in CLI reference | `docs/reference/cli.md` | All |
| 9.4 | Document `spec.cloudquery` in manifest schema reference | `docs/reference/manifest-schema.md` | US-5 |

## Complexity Tracking

No constitution violations requiring justification. The feature:
- Adds one new package type (not a new module/project)
- Reuses existing validation, build, publish, promote infrastructure
- Templates follow CloudQuery official SDK patterns (not custom framework)
- Directory-based template rendering is a natural extension of the existing single-file renderer
