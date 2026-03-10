# Implementation Plan: Canonical Lock & Catalog Model

**Branch**: `017-canonical-lock-catalog` | **Date**: 2026-03-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/017-canonical-lock-catalog/spec.md`

## Summary

Introduce two canonical data structures — `LockFile` and `CatalogEntry` — in the `contracts` package, unify the existing duplicate `ArtifactRef`/`PackageRef` types into a single `PackageRef`, and wire these through the CLI (`dk lock`, `dk lint`, `dk publish`, `dk catalog search`) and SDK (`sdk/lock`, `sdk/catalog`, `sdk/validate`) layers. The lockfile enables deterministic, digest-pinned dependency resolution; the catalog entry enables queryable package discovery. Both are YAML-serialised, schema-versioned, and validated through existing quality gates.

## Technical Context

**Language/Version**: Go 1.25 (per constitution latest-stable requirement)
**Primary Dependencies**: `github.com/Masterminds/semver/v3` (semver range resolution — no semver library is currently a direct dep; `blang/semver/v4` is only transitive via k8s), `oras-go/v2` (existing OCI registry client), `github.com/spf13/cobra` (existing CLI framework)
**Storage**: OCI registry (existing ORAS-based `sdk/registry`); lockfile as `dk.lock` YAML on local filesystem; catalog metadata as OCI manifest annotations + config blob layer
**Testing**: `go test ./...` — table-driven tests, mock registry client interface
**Target Platform**: CLI (macOS, Linux), OCI registries (GHCR, generic OCI)
**Project Type**: Multi-module Go workspace (`contracts` ← `sdk` ← `cli`, `contracts` ← `platform/controller`)
**Performance Goals**: `dk lock` resolves up to 50 dependencies in <10s; `dk catalog search` returns <5s for 1,000-entry catalogs (per SC-005)
**Constraints**: Deterministic YAML output (SC-001); no new runtime dependencies; flat dependency resolution only (MVP)
**Scale/Scope**: ~4 new files in `contracts`, ~2 new packages in `sdk`, ~2 new CLI commands, modifications to `dk publish` and `dk lint`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Article | Requirement | Status | Notes |
|---------|-------------|--------|-------|
| **I — Developer Experience** | Happy path simple, clear output | ✅ PASS | `dk lock` auto-runs on first build; `dk doctor` already checks prerequisites; error messages suggest next action (e.g., "run dk lock --update") |
| **II — Contracts Stable** | Machine-readable, versioned, additive | ✅ PASS | `LockFile`, `CatalogEntry`, unified `PackageRef` are new additive types in `contracts`; `lockVersion` field enables future migration; existing `ArtifactRef` users migrated in-tree (pre-production exemption) |
| **III — Immutability** | Released artifacts immutable, auditable | ✅ PASS | Lockfile pins OCI digests (content-addressable); catalog entries are immutable per version (existing ImmutabilityError on re-push) |
| **IV — Separation of Concerns** | Clean layer boundaries | ✅ PASS | Lockfile is a domain-engineer artifact (lives alongside dk.yaml); catalog entries are platform-level metadata (stored in registry); no cross-layer leakage |
| **V — Security/Compliance** | Classification, PII explicit, auditable | ✅ PASS | CatalogEntry includes `classification` field; lockfile validation ensures digest integrity; publish action already auditable |
| **VI — Observability** | Metrics, structured logs | ✅ PASS | `dk lock` and `dk catalog search` are CLI-only (no runtime component); publish-time catalog write logged with existing structured logging |
| **VII — Quality Gates** | Schema validation, tests | ✅ PASS | `dk lint` extended with lockfile-vs-manifest validation; lockfile format has JSON Schema; contract types have unit tests |
| **VIII — Pragmatism** | MVP-scoped, incremental | ✅ PASS | Flat dependencies only; catalog stored as OCI annotations (no separate catalog service); `dk catalog search` queries registry tags+annotations |
| **IX — Maintainability** | Clear module boundaries | ✅ PASS | Dependency direction preserved: `contracts` ← `sdk/lock` ← `cli`; `contracts` ← `sdk/catalog`; no new cross-module deps |
| **X — Persona Boundaries** | Clean ownership | ✅ PASS | Data engineers own `dk.lock` (alongside dk.yaml); platform engineers own catalog metadata (registry-side); each validatable independently |
| **XI — Extensions are Contracts** | Schema-validated, versioned | ✅ PASS | `dk.lock` format has `lockVersion` for schema evolution; `CatalogEntry` includes schema fingerprints for referenced assets |

## Pre-Implementation Gates

| Gate | Requirement | Status | Evidence |
|------|-------------|--------|----------|
| **Workflow Demo** | End-to-end developer workflow demonstrated | ✅ | `dk init` → `dk dev up` → `dk run` → `dk lock` → `dk lint` → `dk build` → `dk publish` (catalog auto-populated) → `dk promote` |
| **Contract Schema** | Schemas and validation explicit | ✅ | `LockFile`, `LockedDependency`, `CatalogEntry`, `PackageRef` defined as Go structs with JSON/YAML tags; JSON Schema generated; `dk lint` validates lockfile consistency |
| **Promotion/Rollback** | Mechanics explicit | ✅ | Existing promotion unchanged; lockfile pins versions for build reproducibility; rollback via `dk lock --update <name>` to re-resolve or git-revert of `dk.lock` |
| **Observability** | Requirements addressed | ✅ | No runtime components; CLI commands log structured output; catalog entries have `createdAt` timestamps |
| **Security/Compliance** | Secrets, PII addressed | ✅ | No secrets in lockfile or catalog; CatalogEntry.Classification field for PII metadata; digest-based integrity verification |
| **Persona Mapping** | Ownership boundaries clean | ✅ | Data engineers: `dk.lock` (versioned in their repo); Platform engineers: catalog entries (registry-side, read via `dk catalog search`) |

## Project Structure

### Documentation (this feature)

```text
specs/017-canonical-lock-catalog/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0: research findings
├── data-model.md        # Phase 1: entity definitions
├── quickstart.md        # Phase 1: developer quickstart
├── contracts/           # Phase 1: contract schemas
│   ├── lockfile.schema.json
│   └── catalog-entry.schema.json
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

```text
contracts/
├── lockfile.go          # NEW: LockFile, LockedDependency structs
├── lockfile_test.go     # NEW: unit tests
├── catalog.go           # NEW: CatalogEntry struct
├── catalog_test.go      # NEW: unit tests
├── version.go           # MODIFIED: PackageRef unified, ArtifactRef deprecated
└── version_test.go      # MODIFIED: updated tests

sdk/
├── lock/                # NEW: lockfile resolution package
│   ├── lock.go          # Read/Write/Resolve/Update logic
│   ├── lock_test.go     # Unit tests
│   ├── resolver.go      # Semver constraint resolution against registry
│   └── resolver_test.go
├── validate/
│   ├── aggregate.go     # MODIFIED: add lockfile validation step
│   ├── lockfile.go      # NEW: lockfile-vs-manifest validation
│   └── lockfile_test.go # NEW: unit tests
└── registry/
    ├── client.go         # MODIFIED: add CatalogEntry to ArtifactConfig
    └── catalog.go        # NEW: catalog query helpers

cli/cmd/
├── lock.go              # NEW: dk lock command
├── lock_test.go         # NEW: unit tests
├── catalog.go           # NEW: dk catalog search command
├── catalog_test.go      # NEW: unit tests
├── publish.go           # MODIFIED: populate catalog entry on publish
└── lint.go              # MODIFIED: integrate lockfile validation

platform/controller/api/v1alpha1/
└── packagedeployment_types.go  # MODIFIED: use contracts.PackageRef
```

**Structure Decision**: No new top-level directories. New types live in existing `contracts`, new SDK logic in new `sdk/lock` and expanded `sdk/validate`. CLI commands follow existing patterns in `cli/cmd/`. This preserves the `contracts` ← `sdk` ← `cli` dependency direction per Article IX.

## Constitution Check — Post-Design Re-Evaluation

All gates remain ✅ PASS after Phase 1 design. Key confirmations:

| Article | Re-Evaluation Notes |
|---------|-------------------|
| **II — Contracts Stable** | Confirmed: `LockFile`, `LockedDependency`, `CatalogEntry`, `PackageRef` are additive. `ArtifactRef` aliased (not removed). `lockVersion` field enables future migration. JSON Schemas generated in `contracts/`. |
| **III — Immutability** | Confirmed: Lockfile pins `sha256` digests. Catalog entries are immutable per version (existing `ImmutabilityError` on re-push applies). |
| **VII — Quality Gates** | Confirmed: Error codes E300–E305, W300 defined. `dk lint` validates lockfile consistency. JSON Schema for `dk.lock` enables external validation. |
| **VIII — Pragmatism** | Confirmed: Catalog stored as OCI annotations (no new infrastructure). Flat dependencies only. `Masterminds/semver/v3` is the only new dependency. |
| **IX — Maintainability** | Confirmed: Dependency direction preserved: `contracts` ← `sdk/lock` ← `cli`. No cross-module cycles introduced. |

**No constitution violations detected. No deviations to justify.**

## Complexity Tracking

No constitution violations to justify. All changes are additive within existing module boundaries.