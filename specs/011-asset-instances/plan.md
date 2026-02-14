# Implementation Plan: Asset Instances

**Branch**: `011-asset-instances` | **Date**: 2026-02-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/011-asset-instances/spec.md`

## Summary

Add an asset abstraction so data engineers can declare configured instances of approved extensions via `dp asset create/validate/list/show`. Assets are config-only YAML files referencing an extension by FQN and version, with a config block validated against the extension's `schema.json` at `dp validate` time. The `dp.yaml` manifest gains an `assets` section, and bindings become asset-scoped. Implementation adds new contracts (`Asset`, `AssetType`, `AssetManifest`), a JSON Schema for `asset.yaml`, a new `sdk/asset/` package for loading/validation, new CLI subcommands under `dp asset`, and modifications to the aggregate validator and dp.yaml schema to include asset-aware validation.

## Technical Context

**Language/Version**: Go 1.25 (multi-module monorepo: `cli/`, `sdk/`, `contracts/`)
**Primary Dependencies**: `gopkg.in/yaml.v3` (parsing), `github.com/santhosh-tekuri/jsonschema/v6` (JSON Schema validation), `oras.land/oras-go/v2` (OCI registry), `github.com/spf13/cobra` (CLI)
**Storage**: Local filesystem (`assets/` directory tree); OCI registry for extension schema resolution
**Testing**: `go test ./...` — table-driven tests, mock registry client via interface
**Target Platform**: macOS / Linux CLI
**Project Type**: Go multi-module monorepo (contracts ← sdk ← cli)
**Performance Goals**: `dp asset list` < 1 second for up to 50 assets; `dp asset validate` < 3 seconds including registry schema fetch
**Constraints**: Backward compatibility with existing `dp.yaml` and `bindings.yaml` formats; offline-capable validation (structure-only) when registry is unreachable
**Scale/Scope**: Up to 50 assets per project; 3 asset types (source, sink, model-engine); 1 initial extension (CloudQuery source)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Requirement | Status | Evidence |
|------|-------------|--------|----------|
| **Art. I — DX** | Happy path: `dp asset create` → `dp asset validate` → reference in dp.yaml | ✅ PASS | Spec US-1 through US-5 define the complete workflow with clear actionable output |
| **Art. II — Contracts** | Asset schema versioned, machine-readable | ✅ PASS | `asset.yaml` validated against `asset.schema.json`; extension schema versioned by semver |
| **Art. III — Immutability** | Assets are config-only, no runtime mutation | ✅ PASS | Assets are declarative YAML files; version pinning ensures reproducibility |
| **Art. IV — Separation** | Four-layer separation: extension → asset → pipeline → binding | ✅ PASS | Assets reference extensions by FQN (not embedding), bindings are asset-scoped |
| **Art. V — Security** | Least privilege, no secrets in asset files | ✅ PASS | Secrets handled via bindings/envFrom, never in asset config |
| **Art. VI — Observability** | Not runtime — CLI-only validation | ⚠️ N/A | Assets are build-time artifacts; no runtime observability needed |
| **Art. VII — Quality Gates** | Schema validation at `dp validate` time | ✅ PASS | FR-005 through FR-009 require schema validation before publish |
| **Art. VIII — Pragmatism** | MVP scope: single extension, 3 types | ✅ PASS | Bootstrap with CloudQuery source; sink and model-engine types defined but not populated |
| **Art. IX — Maintainability** | Module boundary: contracts ← sdk ← cli | ✅ PASS | New types in `contracts/`, logic in `sdk/asset/`, commands in `cli/cmd/` |
| **Art. X — Persona Boundaries** | Data engineer owns assets; platform engineer owns extensions | ✅ PASS | Assets are data-engineer artifacts referencing platform-engineer-published extensions |
| **Art. XI — Extensions are Contracts** | Asset config validated against extension schema.json | ✅ PASS | FR-005: CLI resolves extension from registry, validates config against schema.json |
| **Workflow Demo** | End-to-end workflow demonstrated | ✅ PASS | [quickstart.md](quickstart.md): create → validate → reference → build → publish → promote |
| **Contract Schema** | Machine-readable schema for asset.yaml | ✅ PASS | [asset.schema.json](contracts/asset.schema.json) — draft 2020-12, 10 properties, additionalProperties: false |
| **Promotion/Rollback** | Assets are versioned within package; rollback = pin previous version | ✅ PASS | No change to existing promotion mechanics |
| **Observability** | CLI-only feature; structured error output | ✅ PASS | Validation errors include code, field, message |
| **Security/Compliance** | No secrets in config; PII metadata on artifacts | ✅ PASS | Handled by existing classification metadata |
| **Persona Mapping** | Data engineer: asset create/validate/list/show; Platform engineer: extension publish | ✅ PASS | Clear ownership boundary in spec |

## Project Structure

### Documentation (this feature)

```text
specs/011-asset-instances/
├── plan.md              # This file
├── research.md          # Phase 0: JSON Schema validation, directory patterns, compat
├── data-model.md        # Phase 1: Asset, AssetType, ExtensionRef entities
├── quickstart.md        # Phase 1: End-to-end asset workflow tutorial
├── contracts/           # Phase 1: asset.schema.json, Go type defs, dp.yaml changes
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
contracts/
├── asset.go                    # NEW: Asset, AssetType, AssetManifest types
├── asset_test.go               # NEW: Asset type tests
├── datapackage.go              # MODIFIED: Add Assets field to DataPackageSpec
├── binding.go                  # MODIFIED: Add AssetRef field to Binding
└── schemas/
    └── asset.schema.json       # NEW: JSON Schema for asset.yaml

sdk/
├── asset/                      # NEW package
│   ├── loader.go               # Load asset.yaml files from assets/ directory
│   ├── loader_test.go
│   ├── scaffolder.go           # Generate asset.yaml from extension schema
│   └── scaffolder_test.go
├── validate/
│   ├── asset.go                # NEW: AssetValidator
│   ├── asset_test.go           # NEW: Asset validation tests
│   ├── aggregate.go            # MODIFIED: Add asset validation pass
│   └── datapackage.go          # MODIFIED: Validate assets section references
└── registry/
    └── client.go               # EXISTING: Used to fetch extension schema.json

cli/cmd/
├── asset.go                    # NEW: dp asset root command
├── asset_create.go             # NEW: dp asset create <name> --ext <fqn>
├── asset_create_test.go
├── asset_validate.go           # NEW: dp asset validate [path]
├── asset_validate_test.go
├── asset_list.go               # NEW: dp asset list
├── asset_list_test.go
├── asset_show.go               # NEW: dp asset show <name>
├── asset_show_test.go
└── root.go                     # MODIFIED: Register asset subcommand

tests/e2e/
└── asset_test.go               # NEW: End-to-end asset workflow test
```

**Structure Decision**: Follows existing monorepo module boundary pattern (`contracts` ← `sdk` ← `cli`). New `sdk/asset/` package for asset-specific logic (loading, scaffolding) keeps the SDK organized by domain. CLI commands follow the existing `cmd/*.go` pattern with `_test.go` companions. No new Go modules needed — all additions fit within the three existing modules.

## Complexity Tracking

> No constitution violations detected. All gates pass.
