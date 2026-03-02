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

3. **`runtime` is always a string.** The `runtime` field in `dk.yaml` is a plain string (`cloudquery`, `generic-go`, `generic-python`, `dbt`). The legacy `RuntimeSpec` object form is dead. Custom `UnmarshalYAML` dual-format handling must be removed.

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
| Delete `DataPackage` struct | ✅ | `contracts/datapackage.go` deleted. `DataPackage`, `DataPackageSpec`, custom `UnmarshalYAML`, and `RuntimeSpec` removed. All callers migrated to `Source`, `Destination`, or `Model` |
| Delete `PipelineManifest` | ✅ | `contracts/pipeline.go` renamed to `contracts/shared.go`. Shared types (`EnvVar`, `Probe`, `EnvFromSource`, etc.) preserved; `PipelineManifest` struct deleted |
| Delete `PackageType` | ✅ | `PackageType` and all `spec.type` references deleted from `contracts/types.go` and all CLI/SDK code |
| Delete `KindDataPackage` constant | ✅ | Removed from `contracts/types.go`. `Kind.IsValid()` only accepts Source/Destination/Model |
| Delete `PipelineMode` alias | ✅ | `PipelineMode` alias and `PipelineModeBatch`/`PipelineModeStreaming` constants removed from `contracts/types.go` |
| Delete legacy JSON schemas | ✅ | `dk-manifest.schema.json` and `pipeline-manifest.schema.json` deleted |
| New JSON schemas for Source/Destination/Model | ❌ | No JSON schemas exist for the new taxonomy kinds. Need `source.schema.json`, `destination.schema.json`, `model.schema.json` |

---

## 2. CLI — `dk init`

| Item | Status | Details |
|------|--------|---------|
| `--kind` flag (source/destination/model) | ✅ | Default: `model` |
| `--runtime` flag (required) | ✅ | cloudquery, generic-go, generic-python, dbt |
| `--mode` flag (model only) | ✅ | batch/streaming, default batch |
| Delete legacy flags (`--type`, `--language`, `--role`) | ❌ | Hidden flags still exist and map to new flags. Must be deleted entirely — they teach the wrong vocabulary |
| Template directories for Source | ⚠️ | `source/cloudquery/` ✅, `source/generic-go/` ✅, **`source/generic-python/` ❌** |
| Template directories for Destination | ⚠️ | `destination/cloudquery/` ✅, `destination/generic-go/` ✅, **`destination/generic-python/` ❌** |
| Template directories for Model | ✅ | `model/cloudquery/` ✅, `model/generic-go/` ✅, `model/generic-python/` ✅, `model/dbt/` ✅ |
| Delete legacy root `dk.yaml.tmpl` | ❌ | Still present at templates root |
| Delete legacy `cloudquery/` template dir | ❌ | Still present alongside new kind-based dirs |

---

## 3. CLI — `dk lint` / Validation

| Item | Status | Details |
|------|--------|---------|
| Delete `DataPackageValidator` | ✅ | Replaced with `ManifestValidator` with kind-dispatch to `validateSource()`, `validateDestination()`, `validateModel()` |
| **E100**: kind must be Source or Destination | ✅ | Validated via `Kind.IsValid()` in manifest validator |
| **E101**: `spec.runtime` must be known Runtime | ✅ | `Runtime.IsValid()` checked in `validateSource()`, `validateDestination()`, `validateModel()` |
| **E102**: Source requires `provides` / Dest requires `accepts` | ✅ | E102 (Source provides) and E103 (Dest accepts) error codes implemented |
| **E103**: `spec.image` required for generic-* runtimes | ✅ | E104 error code; `Runtime.IsGeneric()` helper checks `generic-go`/`generic-python` for Source, Destination, Model |
| **E104**: `configSchema` recommended (warning if missing) | ✅ | W104 warning for Source and Destination when `ConfigSchema` is nil |
| **E105**: `metadata.version` required + valid semver | ✅ | Covered by existing `validateMetadata()` |
| **E200**: kind must be Model | ✅ | Kind-dispatched via `ManifestValidator.Validate()` |
| **E201**: `spec.runtime` must be known Runtime | ✅ | `Runtime.IsValid()` in `validateModel()` |
| **E202**: `spec.mode` must be batch/streaming | ✅ | Validated when non-empty via `Mode.IsValid()` |
| **E203**: `spec.outputs` required | ✅ | `ErrCodeOutputsRequired` in `validateModel()` |
| **E204**: Classification required on all outputs | ✅ | `ErrCodeClassificationRequired` enforced on output artifacts via `requireClassification` flag |
| **E205**: `spec.source` must resolve to published Source | ❌ | No ExtensionRef resolution (P2) |
| **E206**: `spec.destination` must resolve to published Dest | ❌ | No ExtensionRef resolution (P2) |
| **E207**: `spec.config` validated against configSchemas | ❌ | Only exists for Assets, not Model config (P2) |
| **E208**: `spec.image` required for generic-* (unless ext provides it) | ✅ | Covered by E104; ext-check deferred to E205/E206 resolution |
| **E209**: Schedule required for batch mode (warning) | ✅ | W209 warning when `Mode.Default() == batch` and `Schedule` is nil |

---

## 4. CLI — `dk build`

| Item | Status | Details |
|------|--------|---------|
| Route on `Kind` (Source/Destination/Model) | ⚠️ | Uses `manifest.ParseManifestFile()` which returns `Kind` via `DetectKind()`. Displays Kind in output. No `spec.type` checks remain. But no kind-specific build paths yet (all kinds follow the same validate→bundle flow) |
| Source kind build (Docker image + OCI bundle) | ❌ | No Source-specific build logic exists — P1 item |
| Destination kind build | ❌ | No Destination-specific build logic exists — P1 item |
| Model kind build | ✅ | Works via generic validate→bundle path. No legacy `DataPackage` path |
| Delete `spec.type` checks | ✅ | All `spec.type` and `PackageType` references deleted from `build.go` |

---

## 5. CLI — `dk run`

| Item | Status | Details |
|------|--------|---------|
| Route on `Kind` + `Runtime` | ⚠️ | Uses `manifest.ParseManifestFile()` which returns `Kind`. No `spec.type` checks remain. But no Kind-specific run dispatch yet (all kinds follow same Docker path) — P1 item |
| Read `Mode` from `spec.mode` directly | ✅ | Reads from `model.Spec.Mode` (flat field). No `RuntimeSpec` fallback chain. No legacy `runtime.mode` path |
| Delete `RuntimeSpec` mode/image reads | ✅ | All `RuntimeSpec` references deleted from `run.go` and `runner.go`. Mode and Image read from top-level spec fields |
| Source kind execution | ❌ | No Source-specific run logic — P1 item |
| Destination kind execution | ❌ | No Destination-specific run logic — P1 item |
| Model kind execution with ExtensionRef composition | ❌ | Doesn't compose Source + Destination from ExtensionRefs — P2 item |

---

## 6. CLI — `dk show`

| Item | Status | Details |
|------|--------|---------|
| Generic YAML/JSON display | ✅ | Works for all kinds (generic map-based) |
| Kind-specific enrichment | ❌ | No ExtensionRef resolution, no configSchema display |

---

## 7. Extension Registry / Publishing

| Item | Status | Details |
|------|--------|---------|
| `dk publish` command | ❌ | Exists but says "Push not implemented yet" |
| Extension registry for Source/Destination | ❌ | No mechanism to publish extensions separately from data packages |
| ExtensionRef resolution at runtime | ❌ | `ExtensionRef` struct exists but nothing resolves it |
| Extension discovery / search | ❌ | No `dk extension list` or `dk extension search` |

---

## 8. SDK — Manifest Parsing

| Item | Status | Details |
|------|--------|---------|
| `ParseSource()` | ✅ | In `sdk/manifest/source.go` |
| `ParseDestination()` | ✅ | In `sdk/manifest/destination.go` |
| `ParseModel()` | ✅ | In `sdk/manifest/model.go` |
| `DetectKind()` from raw YAML | ✅ | In `sdk/manifest/parser.go` |
| Delete `ParseDataPackage()` | ✅ | Deleted. `ParseManifest()` now uses `DetectKind()` → `ParseSource()`/`ParseDestination()`/`ParseModel()`. Parser test confirms `kind: DataPackage` returns an error |
| Delete `ParsePipeline()` | ✅ | Deleted along with `contracts/pipeline.go` (renamed to `contracts/shared.go`). No legacy pipeline parsing path exists |

---

## 9. SDK — Runner

| Item | Status | Details |
|------|--------|---------|
| DockerRunner works for Model (batch/streaming) | ✅ | `runner.go` uses `contracts.Mode` for mode, `RunOptions.Mode` field. No `spec.type` or `RuntimeSpec` references |
| Kind-aware runner dispatch | ⚠️ | Runner accepts `Kind` via `MapBindingsToEnvVars()`. No `spec.type` references. But no Kind-specific run strategies yet — P1 item |
| Source-specific runner | ❌ | No logic to run a Source extension standalone — P1 item |
| Destination-specific runner | ❌ | No logic to run a Destination extension standalone — P1 item |
| Model runner with extension composition | ❌ | No logic to pull Source+Destination from refs and compose — P2 item |

---

## 10. Test Coverage

| Item | Status | Details |
|------|--------|---------|
| `Kind.IsValid()` tests | ✅ | `contracts/types_test.go` — `TestKind_Constants`, `TestKind_IsValid` |
| `Runtime.IsValid()` tests | ✅ | `contracts/types_test.go` — `TestRuntime_Constants`, `TestRuntime_IsValid` |
| `Mode.IsValid()` / `Mode.Default()` tests | ✅ | `contracts/types_test.go` — `TestMode_IsValid`, `TestMode_Default` |
| `Source` / `Destination` / `Model` parsing tests | ✅ | `sdk/manifest/parser_test.go` — tests for all three kinds via `ParseManifest` and `ParseManifestFile`. `DataPackage` kind confirmed to return error |
| `ManifestValidator` tests (Source/Destination/Model) | ✅ | `sdk/validate/manifest_test.go` — tests `ManifestValidator` with kind-dispatch to `validateSource()`, `validateDestination()`, `validateModel()` |
| E2E tests with Source/Destination kinds | ❌ | E2E tests in `tests/e2e/` still use `--type pipeline` (legacy). Must be rewritten — P1 item |
| Template rendering tests for new kinds | ✅ | `cli/internal/templates/renderer_test.go` — tests for `source/`, `destination/`, `model/` template rendering |
| Delete all legacy test fixtures | ⚠️ | CLI testdata updated (Model kind). SDK testdata updated. E2E testdata still uses `DataPackage` kind in `tests/e2e/` |

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

### P0 — Legacy Removal ✅ COMPLETE

1. ✅ **Delete `DataPackage` struct and all references** — `contracts/datapackage.go` deleted (`DataPackage`, `DataPackageSpec`, `RuntimeSpec`, `UnmarshalYAML`). `KindDataPackage`, `PackageType`, `PipelineMode` alias all removed. `contracts/pipeline.go` renamed to `contracts/shared.go` (shared types preserved).
2. ✅ **Delete legacy code paths in CLI** — `spec.type` checks deleted from `build.go` and `run.go`. `--type`/`--language`/`--role` flags deleted from `init.go`. `datapackage` compat shim deleted. `ParseDataPackage()`/`ParsePipeline()` deleted from SDK. `root.go` example text fixed.
3. ✅ **Delete legacy templates and schemas** — root `dk.yaml.tmpl` deleted, `cloudquery/` template dir deleted, `dk-manifest.schema.json` and `pipeline-manifest.schema.json` deleted.
4. ✅ **Rewrite all tests** — all unit tests purged of `DataPackage`, `--type pipeline`, `spec.type`, legacy fixtures. CLI testdata updated. SDK testdata updated. E2E tests still need rewrite (separate P1 item).
5. ✅ **Fix error messages/comments** — `contracts/errors.go` comments and message templates updated from legacy `PipelineManifest`/`RuntimeSpec`/`PackageType` references.

### P1 — New Taxonomy Implementation (core functionality)

6. **Kind-specific validators** — `validateSource()`, `validateDestination()`, `validateModel()` exist but missing E102/E103/E208/E209 rules.
7. **Kind-aware `dk build`** — Legacy removed, Kind detected. No kind-specific build routing yet.
8. **Kind-aware `dk run`** — Legacy removed, Mode read correctly. No kind-specific run dispatch yet.
9. **Test coverage** — Unit tests exist for types, parsers, validators, templates. E2E tests need rewrite.

### P2 — Feature Completion

10. **Extension Registry** — `dk publish`, ExtensionRef resolution, extension discovery.
11. Missing templates — `source/generic-python/`, `destination/generic-python/`.
12. New JSON schemas — `source.schema.json`, `destination.schema.json`, `model.schema.json`.
13. Classification *required* on all outputs.

### P3 — Polish

14. Lineage auto-derivation from extension contracts.
15. Docs rewrite around two personas.
16. `dk show` enrichment with ExtensionRef/configSchema display.
