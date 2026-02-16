# Gap Analysis: Current State vs. Revised Platform Design

> **Date:** 2026-02-16  
> **Branch:** `015-revised-taxonomy`  
> **Reference:** `revised-platform-design_2.md`

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Done |
| ⚠️ | Partial |
| ❌ | Missing |

---

## Constitution

These principles govern all implementation decisions for the revised taxonomy. They are non-negotiable.

1. **No backward compatibility before v1.0.** The project is pre-release. Legacy concepts (`DataPackage`, `PipelineManifest`, `PackageType`, `pipeline.yaml`) must be fully removed, not preserved behind compat shims. Keeping dead concepts around creates confusion and slows contributors.

2. **Three kinds, nothing else.** The only valid manifest kinds are `Source`, `Destination`, and `Model`. There is no `DataPackage` kind. There is no `Pipeline` kind. Code that checks for or accepts these old kinds must be deleted.

3. **`runtime` is always a string.** The `runtime` field in `dp.yaml` is a plain string (`cloudquery`, `generic-go`, `generic-python`, `dbt`). The legacy `RuntimeSpec` object form is dead. Custom `UnmarshalYAML` dual-format handling must be removed.

4. **Delete, don't deprecate.** When replacing a concept, delete the old code entirely. Do not add `// Deprecated` comments, hidden flags, type aliases, or translation layers. If something is replaced, it is gone.

5. **One path through the code.** There must not be parallel legacy/new code paths (e.g., checking `spec.type` *and* `Kind`). Every CLI command, validator, runner, and parser must operate exclusively on the new taxonomy.

6. **Tests prove the new world.** All tests must use the new taxonomy exclusively. No test should scaffold a `DataPackage` or reference `--type pipeline`. If a test does that, it is wrong.

---

## 1. Contracts Layer

| Item | Status | Details |
|------|--------|---------|
| `Kind` type (Source/Destination/Model) | ✅ | Defined in `contracts/types.go` with `IsValid()` |
| `Runtime` type (cloudquery/generic-go/generic-python/dbt) | ✅ | Defined with `IsValid()` |
| `Mode` type (batch/streaming) | ✅ | Defined with `IsValid()` + `Default()` |
| `Source` struct | ✅ | `contracts/source.go` — `ExtMetadata`, `SourceSpec`, `ConfigSchema` |
| `Destination` struct | ✅ | `contracts/destination.go` — `DestSpec`, `Accepts` |
| `Model` struct | ✅ | `contracts/model.go` — `ModelSpec`, `ExtensionRef`, full field set |
| `ExtensionRef` struct | ✅ | Name/Namespace/Version for referencing published extensions |
| `ConfigSchema` / `ConfigProperty` | ✅ | Defined for extension config validation |
| Delete `DataPackage` struct | ❌ | `contracts/datapackage.go` still exists. The `DataPackage` struct, `DataPackageSpec`, custom `UnmarshalYAML` dual-format handler, and `RuntimeSpec` object must all be deleted. All code using `DataPackage` must be migrated to use `Source`, `Destination`, or `Model` directly |
| Delete `PipelineManifest` | ❌ | `contracts/pipeline.go` still present and actively imported. Shared types (`EnvVar`, `Probe`, `EnvFromSource`, etc.) must be extracted to a `contracts/shared.go` or similar, then `pipeline.go` deleted entirely |
| Delete `PackageType` | ❌ | `PackageType` (`pipeline`, `cloudquery`) still defined in `contracts/types.go`. Must be deleted along with all code paths that reference `spec.type` |
| Delete `KindDataPackage` constant | ❌ | `KindDataPackage = "DataPackage"` still in `contracts/types.go` and accepted by `Kind.IsValid()`. Must be removed — only Source/Destination/Model are valid |
| Delete `PipelineMode` alias | ❌ | `PipelineMode = Mode` alias and `PipelineModeBatch`/`PipelineModeStreaming` constants still in `contracts/types.go`. Must be removed |
| Delete legacy JSON schemas | ❌ | `contracts/schemas/dp-manifest.schema.json` and `contracts/schemas/pipeline-manifest.schema.json` still hardcode `kind: DataPackage`. Must be deleted |
| New JSON schemas for Source/Destination/Model | ❌ | No JSON schemas exist for the new taxonomy kinds. Need `source.schema.json`, `destination.schema.json`, `model.schema.json` |

---

## 2. CLI — `dp init`

| Item | Status | Details |
|------|--------|---------|
| `--kind` flag (source/destination/model) | ✅ | Default: `model` |
| `--runtime` flag (required) | ✅ | cloudquery, generic-go, generic-python, dbt |
| `--mode` flag (model only) | ✅ | batch/streaming, default batch |
| Delete legacy flags (`--type`, `--language`, `--role`) | ❌ | Hidden flags still exist and map to new flags. Must be deleted entirely — they teach the wrong vocabulary |
| Template directories for Source | ⚠️ | `source/cloudquery/` ✅, `source/generic-go/` ✅, **`source/generic-python/` ❌** |
| Template directories for Destination | ⚠️ | `destination/cloudquery/` ✅, `destination/generic-go/` ✅, **`destination/generic-python/` ❌** |
| Template directories for Model | ✅ | `model/cloudquery/` ✅, `model/generic-go/` ✅, `model/generic-python/` ✅, `model/dbt/` ✅ |
| Delete legacy root `dp.yaml.tmpl` | ❌ | Still present at templates root |
| Delete legacy `cloudquery/` template dir | ❌ | Still present alongside new kind-based dirs |

---

## 3. CLI — `dp lint` / Validation

| Item | Status | Details |
|------|--------|---------|
| Delete `DataPackageValidator` | ❌ | All validation still funnels through `DataPackageValidator` with an `isNewKind()` branch. Must be replaced with `SourceValidator`, `DestinationValidator`, `ModelValidator` — no legacy validator wrapper |
| **E100**: kind must be Source or Destination | ❌ | No Source/Destination-specific validator exists at all |
| **E101**: `spec.runtime` must be known Runtime | ❌ | `validateNewKindSpec()` doesn't check runtime |
| **E102**: Source requires `provides` / Dest requires `accepts` | ❌ | Not validated |
| **E103**: `spec.image` required for generic-* runtimes | ❌ | Not validated |
| **E104**: `configSchema` recommended (warning if missing) | ❌ | Not checked |
| **E105**: `metadata.version` required + valid semver | ✅ | Covered by existing `validateMetadata()` |
| **E200**: kind must be Model | ⚠️ | Kind is validated, but not routed to a Model-specific validator |
| **E201**: `spec.runtime` must be known Runtime | ❌ | Not validated for Model |
| **E202**: `spec.mode` must be batch/streaming | ❌ | Not validated |
| **E203**: `spec.outputs` required | ✅ | Checked in `validateNewKindSpec()` for Model |
| **E204**: Classification required on all outputs | ❌ | Only checked *if present*, not *required* |
| **E205**: `spec.source` must resolve to published Source | ❌ | No ExtensionRef resolution |
| **E206**: `spec.destination` must resolve to published Dest | ❌ | No ExtensionRef resolution |
| **E207**: `spec.config` validated against configSchemas | ❌ | Only exists for Assets, not Model config |
| **E208**: `spec.image` required for generic-* (unless ext provides it) | ❌ | Not validated |
| **E209**: Schedule required for batch mode (warning) | ❌ | Not checked |

---

## 4. CLI — `dp build`

| Item | Status | Details |
|------|--------|---------|
| Route on `Kind` (Source/Destination/Model) | ❌ | Only checks `spec.type` (legacy `PackageType`). All `spec.type` checks must be replaced with `Kind`-based dispatch |
| Source kind build (Docker image + OCI bundle) | ❌ | No Source-specific build logic exists |
| Destination kind build | ❌ | No Destination-specific build logic exists |
| Model kind build | ❌ | Works accidentally via legacy DataPackage path. Must be explicitly routed via `Kind` |
| Delete `spec.type` checks | ❌ | Build command still references `PackageType`. All such references must be removed |

---

## 5. CLI — `dp run`

| Item | Status | Details |
|------|--------|---------|
| Route on `Kind` + `Runtime` | ❌ | Routes on `spec.type == cloudquery` (legacy). Must be replaced with `Kind`/`Runtime`-based dispatch. All `spec.type` checks must be deleted |
| Read `Mode` from `spec.mode` directly | ❌ | Still checks `runtime.mode` (legacy `RuntimeSpec` object) first, then falls back to `spec.mode`. The fallback chain must be removed — `spec.mode` is the only source |
| Delete `RuntimeSpec` mode/image reads | ❌ | Runner reads image and mode from `Spec.Runtime.*` (legacy object). Must read from `Spec.Image` and `Spec.Mode` directly |
| Source kind execution | ❌ | No Source-specific run logic |
| Destination kind execution | ❌ | No Destination-specific run logic |
| Model kind execution with ExtensionRef composition | ❌ | Doesn't compose Source + Destination from ExtensionRefs |

---

## 6. CLI — `dp show`

| Item | Status | Details |
|------|--------|---------|
| Generic YAML/JSON display | ✅ | Works for all kinds (generic map-based) |
| Kind-specific enrichment | ❌ | No ExtensionRef resolution, no configSchema display |

---

## 7. Extension Registry / Publishing

| Item | Status | Details |
|------|--------|---------|
| `dp publish` command | ❌ | Exists but says "Push not implemented yet" |
| Extension registry for Source/Destination | ❌ | No mechanism to publish extensions separately from data packages |
| ExtensionRef resolution at runtime | ❌ | `ExtensionRef` struct exists but nothing resolves it |
| Extension discovery / search | ❌ | No `dp extension list` or `dp extension search` |

---

## 8. SDK — Manifest Parsing

| Item | Status | Details |
|------|--------|---------|
| `ParseSource()` | ✅ | In `sdk/manifest/source.go` |
| `ParseDestination()` | ✅ | In `sdk/manifest/destination.go` |
| `ParseModel()` | ✅ | In `sdk/manifest/model.go` |
| `DetectKind()` from raw YAML | ✅ | In `sdk/manifest/parser.go` |
| Delete `ParseDataPackage()` | ❌ | Still exists as a backward-compat funnel that accepts all kinds via the `DataPackage` struct. Must be deleted — callers should use `DetectKind()` → `ParseSource()`/`ParseDestination()`/`ParseModel()` |
| Delete `ParsePipeline()` | ❌ | Still exists for legacy `pipeline.yaml`. Must be deleted along with `contracts/pipeline.go` |

---

## 9. SDK — Runner

| Item | Status | Details |
|------|--------|---------|
| DockerRunner works for Model (batch/streaming) | ❌ | Works accidentally via legacy pipeline path. Must be rewritten to use `Kind`/`Runtime`/`Mode` directly, not `spec.type` or `RuntimeSpec` |
| Kind-aware runner dispatch | ❌ | Runner doesn't check `Kind`, only `spec.type`. All `spec.type` references must be replaced |
| Source-specific runner | ❌ | No logic to run a Source extension standalone |
| Destination-specific runner | ❌ | No logic to run a Destination extension standalone |
| Model runner with extension composition | ❌ | No logic to pull Source+Destination from refs and compose |

---

## 10. Test Coverage

| Item | Status | Details |
|------|--------|---------|
| `Kind.IsValid()` tests | ❌ | No unit tests |
| `Runtime.IsValid()` tests | ❌ | No unit tests |
| `Mode.IsValid()` / `Mode.Default()` tests | ❌ | No unit tests |
| `Source` / `Destination` / `Model` parsing tests | ❌ | No `ParseSource`/`ParseDestination`/`ParseModel` tests |
| `SourceValidator` / `DestinationValidator` / `ModelValidator` tests | ❌ | No test cases for kind-specific validation |
| E2E tests with Source/Destination kinds | ❌ | All e2e tests use `--type pipeline` (legacy) or Model kind. Legacy flags must be removed from all tests |
| Template rendering tests for new kinds | ❌ | No tests for `source/`, `destination/`, `model/` templates |
| Delete all legacy test fixtures | ❌ | Any test that creates a `DataPackage` manifest or uses `--type` flag must be rewritten |

---

## 11. Lineage

| Item | Status | Details |
|------|--------|---------|
| OpenLineage event emission | ✅ | Generic, works for any workload |
| Source/Destination/Model awareness | ❌ | No Kind-specific lineage (e.g., auto-derive datasets from Source `provides` / Dest `accepts`) |

---

## 12. Docs

| Item | Status | Details |
|------|--------|---------|
| Rewrite around two personas (infra/data engineer) | ❌ | Docs still reference "DataPackage" and "pipeline" throughout |

---

## Priority Summary

### P0 — Legacy Removal (do first, unblocks everything else)

1. **Delete `DataPackage` struct and all references** — `contracts/datapackage.go` (the struct, `DataPackageSpec`, `RuntimeSpec` object, custom `UnmarshalYAML`), `KindDataPackage` constant, `PackageType`, `PipelineMode` alias. Migrate shared types (`EnvVar`, `Probe`, etc.) from `contracts/pipeline.go` to `contracts/shared.go`, then delete `pipeline.go`.
2. **Delete legacy code paths in CLI** — remove `spec.type` checks in `build.go` and `run.go`, remove `--type`/`--language`/`--role` flags from `init.go`, remove `ParseDataPackage()`/`ParsePipeline()` from SDK.
3. **Delete legacy templates and schemas** — root `dp.yaml.tmpl`, `cloudquery/` template dir, `dp-manifest.schema.json`, `pipeline-manifest.schema.json`.
4. **Rewrite all tests** — purge every test that uses `DataPackage`, `--type pipeline`, or legacy fixtures.

### P1 — New Taxonomy Implementation (core functionality)

5. **Kind-specific validators** — `SourceValidator`, `DestinationValidator`, `ModelValidator` implementing all E1xx/E2xx rules from the design doc.
6. **Kind-aware `dp build`** — route on `Kind`, not `spec.type`.
7. **Kind-aware `dp run`** — route on `Kind`/`Runtime`, read `Mode` and `Image` from top-level spec fields only.
8. **Test coverage** — unit tests for all new types, parsers, validators. E2E tests for Source and Destination kinds.

### P2 — Feature Completion

9. **Extension Registry** — `dp publish`, ExtensionRef resolution, extension discovery.
10. Missing templates — `source/generic-python/`, `destination/generic-python/`.
11. New JSON schemas — `source.schema.json`, `destination.schema.json`, `model.schema.json`.
12. Classification *required* on all outputs.

### P3 — Polish

13. Lineage auto-derivation from extension contracts.
14. Docs rewrite around two personas.
15. `dp show` enrichment with ExtensionRef/configSchema display.
