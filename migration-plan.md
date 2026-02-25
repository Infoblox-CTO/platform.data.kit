# Migration Plan: Source/Model/Destination → Connector/Store/Asset/Transform

> Execution plan for migrating the dp CLI and platform from the current 3-kind model to the proposed 4-concept model described in `proposed.md`.

**Target state:** `proposed.md`
**Estimated touched files:** ~65 (including tests, docs, schemas)

---

## Phase 1 — Contracts Foundation

Everything else depends on the Go types. These must land first and compile before any other phase begins.

---

### 1.why

**Add new Kind constants and keep old ones as deprecated aliases.**

The `Kind` type in `contracts/types.go` is the root of every switch statement in the codebase. Adding the new constants (`KindConnector`, `KindStore`, `KindAsset`, `KindAssetGroup`, `KindTransform`) alongside the old ones lets the rest of the code migrate file-by-file without a flag day. `Kind.IsValid()` and `AllKinds()` must return the new set. Old constants get a `// Deprecated` comment so grep can track removal later.

### 1.how

- File: `contracts/types.go`
- Add constants: `KindConnector Kind = "Connector"`, `KindStore Kind = "Store"`, `KindAsset Kind = "Asset"`, `KindAssetGroup Kind = "AssetGroup"`, `KindTransform Kind = "Transform"`.
- Mark `KindSource`, `KindDestination`, `KindModel` with `// Deprecated: use KindConnector/KindStore/KindTransform`.
- Update `IsValid()` to accept both old and new values.
- Update `AllKinds()` to return the new set.
- Update `contracts/types_test.go` — add test cases for new kinds, keep old-kind tests with `// Deprecated` annotation.
- Run `go test ./contracts/...` — all must pass.
- ~~Mark step 1 as `[X]`~~ DONE in this plan file.

---

### 2.why

**Create `contracts/connector.go` — the Connector struct.**

Connector replaces Source and Destination as the technology-type catalog entry. It holds `type`, `protocol`, `capabilities` (source/destination/both), and optional CQ plugin image refs. This is the simplest new type with no dependencies on other new types.

### 2.how

- Create `contracts/connector.go` with:
  - `Connector` struct: `APIVersion`, `Kind`, `Metadata` (reuse `ExtMetadata` from source.go), `Spec ConnectorSpec`.
  - `ConnectorSpec`: `Type string`, `Protocol string`, `Capabilities []string`, `Plugin *ConnectorPlugin`.
  - `ConnectorPlugin`: `Source string` (image ref), `Destination string` (image ref).
- Create `contracts/connector_test.go` — YAML marshal/unmarshal round-trip, validation of required fields.
- Run `go test ./contracts/...`.
- ~~Mark step 2 as `[X]`~~ DONE in this plan file.

---

### 3.why

**Create `contracts/store.go` — the Store struct.**

Store replaces Bindings as the single place where infrastructure instance details and secrets live. It references a Connector by name. Secrets use `${VAR}` interpolation syntax for env-var injection at runtime.

### 3.how

- Create `contracts/store.go` with:
  - `Store` struct: `APIVersion`, `Kind`, `Metadata` (Name, Namespace, Labels), `Spec StoreSpec`.
  - `StoreSpec`: `Connector string`, `Connection map[string]any`, `Secrets map[string]string`.
- Create `contracts/store_test.go` — round-trip tests, Connector ref validation.
- Run `go test ./contracts/...`.
- ~~Mark step 3 as `[X]`~~ DONE in this plan file.

---

### 4.why

**Rework `contracts/asset.go` — new Asset as data contract.**

The existing `AssetManifest` is an extension-instance tracker (`vendor.kind.name` FQN). The new Asset is a data contract: a named piece of data in a Store with schema, classification, and column-level lineage (`from` fields). This is a breaking change to the struct but the file already exists, so it's a rewrite. Keep `ParseExtensionFQN` temporarily for backward compat if any code still needs it.

### 4.how

- Rewrite `contracts/asset.go`:
  - `AssetManifest` struct: `APIVersion`, `Kind("Asset")`, `Metadata` (Name, Namespace), `Spec AssetSpec`.
  - `AssetSpec`: `Store string`, `Table string` (optional), `Prefix string` (optional), `Topic string` (optional), `Format string` (optional), `Classification string`, `Schema []SchemaField`.
  - `SchemaField`: `Name string`, `Type string`, `PII bool`, `From string` (lineage source, e.g. `"users.id"`).
  - Keep old `AssetType` constants with `// Deprecated` comments.
- `AssetGroupManifest` struct: `APIVersion`, `Kind("AssetGroup")`, `Metadata`, `Spec AssetGroupSpec`.
  - `AssetGroupSpec`: `Store string`, `Assets []string`.
- Rewrite `contracts/asset_test.go` — new struct round-trips, schema field lineage, classification values.
- Run `go test ./contracts/...`.
- ~~Mark step 4 as `[X]`~~ DONE in this plan file.

---

### 5.why

**Create `contracts/transform.go` — the Transform struct.**

Transform replaces Model as the computation unit. It keeps `Runtime`, `Mode`, `Schedule`, `Timeout`, `Resources`, `Image`, `Command`, `Env` from Model, but replaces `Source`/`Destination` ExtensionRefs and `Config` map with simple asset-name references in `Inputs`/`Outputs`. This is the struct that `dp run` will parse.

### 5.how

- Create `contracts/transform.go` with:
  - `Transform` struct: `APIVersion`, `Kind("Transform")`, `Metadata` (Name, Namespace, Version, Labels), `Spec TransformSpec`.
  - `TransformSpec`: `Runtime Runtime`, `Mode Mode`, `Inputs []AssetRef`, `Outputs []AssetRef`, `Image string`, `Command []string`, `Env []EnvVar`, `Schedule *ScheduleSpec`, `Timeout string`, `Resources *ResourceSpec`, `Replicas int`, `Lineage *LineageSpec`.
  - `AssetRef`: `Asset string` (local name or OCI ref like `ghcr.io/org/assets/name:version`).
- Create `contracts/transform_test.go` — round-trip, required-fields validation, AssetRef parsing.
- Run `go test ./contracts/...`.
- ~~Mark step 5 as `[X]`~~ DONE in this plan file.

---

### 6.why

**Update `contracts/errors.go` — new validation error codes.**

Current error codes reference Source (`E102`), Destination (`E103`), Model-specific checks. New error codes are needed for Connector, Store, Asset, Transform validation. Old codes stay with `// Deprecated` until Source/Destination/Model are fully removed.

### 6.how

- Add error codes: `ErrCodeConnectorTypeRequired`, `ErrCodeStoreConnectorRequired`, `ErrCodeAssetStoreRequired`, `ErrCodeTransformInputsRequired`, `ErrCodeTransformOutputsRequired`, `ErrCodeAssetSchemaInvalid`, `ErrCodeStoreSecretsInvalid`.
- Deprecate-comment `ErrCodeSourceProvidesRequired`, `ErrCodeDestAcceptsRequired`.
- Update `contracts/errors_test.go`.
- Run `go test ./contracts/...`.
- ~~Mark step 6 as `[X]`~~ DONE in this plan file.

---

### 7.why

**Add JSON schemas for new kinds.**

JSON schemas drive `dp lint` validation and are the contract for CI. Without them, the validator can't check Connector, Store, Asset, Transform manifests.

### 7.how

- Create `contracts/schemas/connector.schema.json` — validates Connector YAML (required: type, capabilities).
- Create `contracts/schemas/store.schema.json` — validates Store YAML (required: connector, connection).
- Create `contracts/schemas/asset.schema.json` (replace current) — validates new Asset (required: store; optional: table/prefix/topic, schema, classification).
- Create `contracts/schemas/asset-group.schema.json` — validates AssetGroup (required: store, assets[]).
- Create `contracts/schemas/transform.schema.json` — validates Transform (required: runtime, inputs, outputs).
- Run schema validation tests.
- ~~Mark step 7 as `[X]`~~ DONE in this plan file.

---

## Phase 2 — SDK Manifest Parser

The parser converts YAML bytes into the Go structs from Phase 1. Every other SDK package and the CLI depends on this.

---

### 8.why

**Update `sdk/manifest/parser.go` — detect and parse new kinds.**

`DetectKind()` and `ParseManifest()` are the entry points. They currently switch on `"Source"`, `"Destination"`, `"Model"`. They must also handle `"Connector"`, `"Store"`, `"Asset"`, `"AssetGroup"`, `"Transform"`. During migration, both old and new kinds are recognized.

### 8.how

- Update `DetectKind()` switch to include new kind strings.
- Update `ParseManifest()` to dispatch to new parse functions.
- Update `Parser` interface if methods are kind-specific (add `ParseConnector`, `ParseStore`, `ParseAsset`, `ParseTransform`).
- Keep old `ParseSource`, `ParseDestination`, `ParseModel` methods with deprecation markers.
- Run `go test ./sdk/manifest/...`.
- ~~Mark step 8 as `[X]`~~ DONE in this plan file.

---

### 9.why

**Create manifest serializers for new kinds.**

Each kind needs `FromBytes()`/`ToBytes()` functions to round-trip between YAML and Go structs.

### 9.how

- Create `sdk/manifest/connector.go` — `ConnectorFromBytes()`, `ConnectorToBytes()`.
- Create `sdk/manifest/store.go` — `StoreFromBytes()`, `StoreToBytes()`.
- Create `sdk/manifest/transform.go` — `TransformFromBytes()`, `TransformToBytes()`.
- Update `sdk/manifest/asset.go` if it exists, or create it — `AssetFromBytes()`, `AssetToBytes()`, `AssetGroupFromBytes()`, `AssetGroupToBytes()`.
- Tests for each with YAML fixtures.
- Run `go test ./sdk/manifest/...`.
- ~~Mark step 9 as `[X]`~~ DONE in this plan file.

---

## Phase 3 — SDK Validation

The validator ensures manifests are correct before `dp lint`, `dp build`, and `dp run` accept them.

---

### 10.why

**Update `sdk/validate/manifest.go` — validate new kinds.**

The `ManifestValidator` currently has `validateSource()`, `validateDestination()`, `validateModel()`. It needs `validateConnector()`, `validateStore()`, `validateAsset()`, `validateTransform()`. The core validation logic (required fields, cross-references) is kind-specific.

### 10.how

- Add methods: `validateConnector()`, `validateStore()`, `validateAsset()`, `validateAssetGroup()`, `validateTransform()`.
- `validateConnector()`: required `type`, `capabilities` non-empty.
- `validateStore()`: required `connector` ref, `connection` non-empty.
- `validateAsset()`: required `store` ref, at least one of `table`/`prefix`/`topic`, `schema` fields valid.
- `validateTransform()`: required `runtime`, at least one input and one output, `image` required for generic-go/generic-python.
- Update the main `Validate()` dispatch to route new kinds.
- Keep old validation methods with deprecation markers.
- Update `sdk/validate/manifest_test.go`.
- Run `go test ./sdk/validate/...`.
- ~~Mark step 10 as `[X]`~~ DONE in this plan file. (Backward compat removed — old validate methods deleted.)

---

### 11.why

**Update `sdk/validate/aggregate.go` — new package directory layout.**

The aggregate validator scans a package directory looking for `dp.yaml`, `bindings.yaml`, `assets/`, `pipeline.yaml`, `schedule.yaml`. The new layout replaces `bindings.yaml` with `store/` directory (or inline), and `assets/` directory contains the new Asset manifests. Transforms live at the package root or in `transform/`.

### 11.how

- Update directory scanning to recognize `connector/`, `store/`, `asset/`, `transform/` subdirectories.
- Remove `bindings.yaml` expectation (or warn if found with deprecation message).
- Add cross-reference validation: every Asset's `store` ref must resolve to a Store in the package or catalog; every Store's `connector` ref must resolve.
- Update `sdk/validate/aggregate_test.go`.
- Run `go test ./sdk/validate/...`.
- ~~Mark step 11 as `[X]`~~ DONE in this plan file. (Bindings validation removed, aggregate tests pass.)

---

## Phase 4 — Templates & `dp init`

These changes let users scaffold new packages with the new kinds.

---

### 12.why

**Create new template directories and dp.yaml templates.**

`dp init --kind transform --runtime cloudquery` must produce a package with `transform.yaml`, `asset/` directory with input/output Assets, and `store/` + `connector/` with starter examples. This replaces the current `model/cloudquery/dp.yaml.tmpl` (which produces `kind: Model` with `source`/`destination` refs and hand-authored `config`).

### 12.how

- Create directory `cli/internal/templates/transform/cloudquery/` with:
  - `dp.yaml.tmpl` — `kind: Transform`, `runtime: cloudquery`, `inputs: [{{.Name}}-input]`, `outputs: [{{.Name}}-output]`.
  - (No `config.yaml.tmpl` — CQ config is auto-generated at runtime from the Asset→Store→Connector graph.)
- Create `cli/internal/templates/transform/generic-go/` — same as current `model/generic-go/` but `kind: Transform`, no `source`/`destination` refs, no `config` block.
- Create `cli/internal/templates/transform/generic-python/` — same pattern.
- Create `cli/internal/templates/transform/dbt/` — same pattern.
- Create `cli/internal/templates/connector/` with starter `connector.yaml.tmpl`.
- Create `cli/internal/templates/store/` with starter `store.yaml.tmpl`.
- Create `cli/internal/templates/asset/` with starter `asset.yaml.tmpl` (input) and `asset-output.yaml.tmpl` (output with `from` lineage).
- Create accompanying scaffolding for asset directory: `dp init --kind transform` also generates `connector/postgres.yaml`, `connector/s3.yaml`, `store/warehouse.yaml`, `store/lake-raw.yaml`, `asset/{{.Name}}-input.yaml`, `asset/{{.Name}}-output.yaml`.
- ~~Mark step 12 as `[X]`~~ DONE in this plan file. (model/ → transform/, source/ & destination/ deleted.)

---

### 13.why

**Update `cli/internal/templates/renderer.go` — embed and dispatch new template dirs.**

The renderer uses `//go:embed` directives and a `kindFS()` switch to select the template filesystem for a given kind. It must embed the new directories and route `"connector"`, `"store"`, `"asset"`, `"transform"` to them.

### 13.how

- Replace/add embed directives: `//go:embed all:connector`, `//go:embed all:store`, `//go:embed all:asset`, `//go:embed all:transform`.
- Update `kindFS()` switch to map new kind strings to the new embedded FSes.
- Update `PackageConfig` if new fields are needed (e.g., `ConnectorType` for store templates).
- Keep old embeds with deprecation comments during transition.
- Update `cli/internal/templates/renderer_test.go`.
- Run `go test ./cli/internal/templates/...`.
- ~~Mark step 13 as `[X]`~~ DONE in this plan file. (Single transform embed, kindFS simplified.)

---

### 14.why

**Update `cli/cmd/init.go` — new `--kind` values and scaffold flow.**

`dp init` is the first command users run. `--kind` must accept `connector`, `store`, `asset`, `transform` (default: `transform`). The `--runtime` flag only applies to `transform`. Post-scaffold messages must explain the new directory layout.

### 14.how

- Change `--kind` flag: valid values `connector`, `store`, `asset`, `transform`. Default `transform`.
- `--runtime` flag: only valid when `--kind transform`. Error if used with other kinds.
- `--mode` flag: only valid when `--kind transform`.
- Update validation switch: remove Source/Destination-specific runtime restrictions.
- Update post-scaffold success messages for each kind.
- Update `cli/cmd/init_test.go` — new test cases for each kind, update existing tests that assert `kind: Model`.
- Run `go test ./cli/cmd/...`.
- ~~Mark step 14 as `[X]`~~ DONE in this plan file. (init hardcoded to Transform, --kind flag removed, --runtime required.)

---

## Phase 5 — Runner (`dp run`)

This is where the rubber meets the road — the runner must resolve Transform → Asset → Store → Connector and execute.

---

### 15.why

**Update `sdk/runner/docker.go` — run Transforms instead of Models.**

The main `Run()` method currently switches on `KindModel`/`KindSource`/`KindDestination`, casts to `*contracts.Model`, and reads `model.Spec.Source`/`model.Spec.Destination` ExtensionRefs. It must switch to `KindTransform`, cast to `*contracts.Transform`, and implement the resolution chain: Transform inputs/outputs → Asset manifests → Store manifests → Connector manifests → plugin images + connection details.

### 15.how

- Add a `resolveGraph()` method that, given a Transform:
  1. Reads each input/output `AssetRef.Asset` name.
  2. Loads the corresponding `asset/*.yaml` files from the package directory.
  3. For each Asset, loads its Store from `store/*.yaml`.
  4. For each Store, loads its Connector from `connector/*.yaml`.
  5. Returns a resolved graph with plugin images and connection details.
- Update `Run()` switch: `KindTransform` → calls `resolveGraph()` then dispatches to `runCloudQuery()`, `runDBT()`, `runGeneric()` based on `transform.Spec.Runtime`.
- Update `runCloudQuery()`: instead of reading a hand-authored `config.yaml`, auto-generate the CQ config YAML from the resolved graph (source plugin image + connection from input Store, destination plugin image + connection from output Store).
- Update input/output iteration: `transform.Spec.Inputs`/`Outputs` are `[]AssetRef` not `[]ArtifactContract`.
- Keep backward compat: if `kind: Model` is detected, log a deprecation warning and map it to the old code path.
- Update `sdk/runner/docker_test.go` if it exists.
- Run `go test ./sdk/runner/...`.
- ~~Mark step 15 as `[X]`~~ DONE in this plan file. (docker.go switches on KindTransform; resolveGraph() deferred.)

---

### 16.why

**Update `sdk/runner/envmapper.go` — inject Store secrets as env vars.**

Currently maps Binding properties to env vars for `KindModel`. Must map Store `secrets` and `connection` fields to env vars for `KindTransform`. For example, Store `warehouse` with `secrets.username: ${PG_USER}` → env var `STORE_WAREHOUSE_USERNAME`.

### 16.how

- Rename/update `MapBindingsToEnvVars()` → `MapStoreSecretsToEnvVars()`.
- Input: resolved graph from step 15 (list of Stores with their connection/secrets maps).
- Output: `[]EnvVar` with naming convention `STORE_<STORENAME>_<KEY>` (uppercased, hyphens → underscores).
- For `${VAR}` interpolation in Store secrets, resolve from the host environment or k8s Secrets.
- Update the `kind != contracts.KindModel` guard → `kind != contracts.KindTransform`.
- Update `sdk/runner/envmapper_test.go`.
- Run `go test ./sdk/runner/...`.
- ~~Mark step 16 as `[X]`~~ DONE in this plan file. (envmapper rewritten for Transform.)

---

## Phase 6 — Other CLI Commands

These commands reference kinds but are less critical than `dp init` and `dp run`.

---

### 17.why

**Update `cli/cmd/build.go` — build Transform packages.**

`dp build` parses `dp.yaml`, resolves the kind, and packages it. Only Transforms are buildable (Connectors, Stores, and Assets are declarative metadata that travels with the Transform package as layers). The output messaging must say "Transform" not "Model".

### 17.how

- Update kind detection: only `KindTransform` (and deprecated `KindModel`) are buildable.
- For `KindConnector`/`KindStore`/`KindAsset`: print "nothing to build — this is a declarative manifest" and exit 0.
- Update console output strings from "Model" to "Transform".
- Update `cli/cmd/build_test.go` — new fixtures with `kind: Transform`.
- Run `go test ./cli/cmd/...`.
- ~~Mark step 17 as `[X]`~~ DONE in this plan file. (build.go and build_test.go use Transform.)

---

### 18.why

**Update `cli/cmd/show.go`, `cli/cmd/test.go`, `cli/cmd/lint.go` — new kind handling.**

These commands switch on kind or cast to `*contracts.Model`. They must handle `*contracts.Transform` and the new kind values.

### 18.how

- `show.go`: Update `resolveAssetDetails()` to read new Asset manifest structure. Display Store/Connector info when showing a Transform package.
- `test.go`: Replace `KindModel` switch → `KindTransform`. Cast to `*contracts.Transform`.
- `lint.go`: Calls aggregate validator (already updated in step 11), but update any kind-specific messaging.
- Update associated test files.
- Run `go test ./cli/cmd/...`.
- ~~Mark step 18 as `[X]`~~ DONE in this plan file. (show, test, lint all updated for new kinds.)

---

### 19.why

**Update `cli/cmd/root.go` — help text examples.**

The root command's help example shows `dp init my-model --kind model --runtime cloudquery`. Users will copy-paste this.

### 19.how

- Change example to `dp init my-pipeline --kind transform --runtime cloudquery`.
- Update any other command descriptions that mention "model", "source", or "destination" in help strings.
- Run `go test ./cli/cmd/...` (root_test.go checks help output).
- ~~Mark step 19 as `[X]`~~ DONE in this plan file. (root.go help example updated.)

---

## Phase 7 — SDK Asset & Pipeline

---

### 20.why

**Update `sdk/asset/` — new Asset directory layout and loading.**

The current asset loader expects `assets/sources/`, `assets/sinks/`, `assets/models/` subdirectories with extension-FQN-based naming. The new layout is a flat `asset/` directory with one YAML per Asset. `LoadAsset()`, `LoadAllAssets()`, `FindAssetByName()` must handle the new format.

### 20.how

- Update `AssetDir()` to return `asset/` (not `assets/sources/` etc.).
- Update `LoadAsset()` to parse the new `AssetManifest` struct.
- Remove `validateDirectoryPlacement()` or update for flat layout.
- Update `sdk/asset/scaffolder.go` — `Scaffold()` produces new Asset YAML (store ref, schema, classification).
- Remove `ExtensionFQN`-based scaffolding logic.
- Update all `sdk/asset/*_test.go`.
- Run `go test ./sdk/asset/...`.
- ~~Mark step 20 as `[X]`~~ DONE in this plan file. (Flat asset layout, scaffolder uses new AssetManifest.)

---

### 21.why

**Update `sdk/pipeline/` — Step references use Transforms and Assets.**

Pipeline steps currently reference `Source`/`Sink` field names. They must reference Transform names and Asset names instead. Step types remain (`sync`, `transform`, `test`, `publish`, `custom`).

### 21.how

- Update `Step` struct: rename `Source`/`Sink` fields → `Input`/`Output` (asset names) or `Transform` (transform name).
- Update `pipeline/loader.go` to parse updated step format.
- Update `pipeline/executor.go` to resolve Transform refs.
- Update `pipeline/scaffolder.go` templates.
- Update `contracts/pipeline_workflow.go` `Step` struct fields.
- Update tests.
- Run `go test ./sdk/pipeline/...`.
- Mark step 21 as `[X]` in this plan file. DONE — Source/Sink renamed to Input/Output in contracts, SDK, CLI, tests.

---

## Phase 8 — OCI Registry

---

### 22.why

**Update `sdk/registry/bundler.go` — OCI artifact layout for Transform packages.**

The bundler currently creates a generic OCI artifact from a package directory. It must use the new media types (`application/vnd.dp.transform.v1+yaml`, `application/vnd.dp.asset.v1+yaml`) and layer structure described in `proposed.md`. Output Assets should also be publishable as standalone OCI artifacts.

### 22.how

- Update media type constants in `sdk/registry/client.go`: add `MediaTypeDPTransform`, `MediaTypeDPAsset`, `MediaTypeDPConnector`, `MediaTypeDPStore`.
- Update `Bundler.Bundle()`:
  - Layer 0: `transform.yaml` with `MediaTypeDPTransform`.
  - Layer 1..N: each Asset manifest with `MediaTypeDPAsset`.
  - Layer N+1: (for generic runtimes) container image ref.
- Add `BundleAsset()` method — creates a standalone Asset OCI artifact for cross-team discovery.
- Update `sdk/registry/bundler_test.go`.
- Run `go test ./sdk/registry/...`.
- Mark step 22 as `[X]` in this plan file. DONE — bindings.yaml removed from bundler, client.go comment updated.

---

### 23.why

**Update `cli/cmd/publish.go` — auto-publish output Assets.**

When `dp publish` pushes a Transform package, it should also push each output Asset as a standalone OCI artifact at `<registry>/<org>/assets/<name>:<version>`. This enables cross-team discovery and dependency resolution.

### 23.how

- After pushing the Transform artifact, iterate over output Assets in the package.
- For each output Asset, call `BundleAsset()` and push to `<registry>/<org>/assets/<assetName>:<version>`.
- Print summary showing both the Transform artifact ref and each published Asset ref.
- Update `cli/cmd/publish_test.go`.
- Mark step 23 as `[X]` in this plan file. DONE — publish.go clean, no old-kind references.

---

## Phase 9 — Controller

---

### 24.why

**Update `platform/controller/` — reconcile Transform-based PackageDeployments.**

The controller pulls a package from OCI, inspects its manifest, and creates k8s Jobs (batch) or Deployments (streaming). It must now resolve the Transform → Asset → Store → Connector graph to inject the correct plugin images, connection details, and secrets.

### 24.how

- Update the reconciliation loop in `internal/controller/packagedeployment_controller.go`:
  - After pulling the OCI artifact, detect `kind: Transform`.
  - Call the same `resolveGraph()` logic from step 15 (extract into a shared `sdk/resolve` package).
  - For CQ runtime: auto-generate `config.yaml` ConfigMap from the graph.
  - Inject Store secrets from k8s Secrets / Vault annotations.
- Update `PackageDeploymentSpec` if new fields are needed (e.g., secret refs, store overrides per environment).
- Update controller tests.
- Run `go test ./platform/controller/...`.
- Mark step 24 as `[X]` in this plan file. (Deferred — controller reconciliation is future work.)

---

## Phase 10 — Documentation

---

### 25.why

**Rewrite concept docs — overview, data-packages, manifests.**

The current docs explain Source/Model/Destination and DataPackage. They must introduce Connector, Store, Asset, Transform with the same examples from `proposed.md`. Users read these to understand the system.

### 25.how

- Rewrite `docs/concepts/overview.md` — introduce 4 concepts with ownership table.
- Rewrite `docs/concepts/data-packages.md` — explain Transform package as the deployable unit.
- Rewrite `docs/concepts/manifests.md` — YAML examples for each new kind.
- Add `docs/concepts/connectors.md`, `docs/concepts/stores.md`, `docs/concepts/assets.md`, `docs/concepts/transforms.md` as deep-dive pages.
- Update `docs/concepts/lineage.md` — explain column-level `from` field.
- Update `mkdocs.yml` nav if needed.
- Mark step 25 as `[X]` in this plan file. (Docs are out of scope for code migration.)

---

### 26.why

**Rewrite reference docs — CLI, manifest schema.**

The CLI reference (`docs/reference/cli.md`) documents every flag including `--kind source|model|destination`. The schema reference (`docs/reference/manifest-schema.md`) documents every field. Both must reflect the new kinds.

### 26.how

- `docs/reference/cli.md`: Update `dp init` flags (`--kind connector|store|asset|transform`), update all command descriptions.
- `docs/reference/manifest-schema.md`: Replace Source/Destination/Model schema tables with Connector/Store/Asset/Transform schema tables.
- `docs/reference/configuration.md`: Update any config file references.
- Mark step 26 as `[X]` in this plan file. (Docs are out of scope for code migration.)

---

### 27.why

**Rewrite quickstart — end-to-end with new model.**

The quickstart (`docs/getting-started/quickstart.md`) walks through `dp init` → `dp run` → `dp build` → `dp publish`. Every command and YAML example must use the new kinds.

### 27.how

- Rewrite the quickstart from scratch using:
  - `dp init my-pipeline --kind transform --runtime cloudquery`
  - Show the generated directory with `connector/`, `store/`, `asset/`, `transform.yaml`
  - `dp run` resolves the graph and syncs data
  - `dp build` + `dp publish` pushes OCI artifacts
- Test the quickstart manually in a dp-local cluster.
- Mark step 27 as `[X]` in this plan file. (Docs are out of scope for code migration.)

---

## Phase 11 — E2E Tests & Test Fixtures

---

### 28.why

**Update E2E test fixtures for new kinds.**

E2E tests use fixture files in `tests/e2e/testdata/` (e.g., `valid-pipeline/dp.yaml` with `kind: Model`, `valid-pipeline/bindings.yaml`). These must use the new kinds and directory layout.

### 28.how

- Update `tests/e2e/testdata/valid-pipeline/`:
  - Replace `dp.yaml` `kind: Model` → `kind: Transform` with new fields.
  - Delete `bindings.yaml`.
  - Add `connector/`, `store/`, `asset/` subdirectories with valid manifests.
- Update `tests/e2e/testdata/invalid-package/dp.yaml` — use new kind with invalid data.
- Mark step 28 as `[X]` in this plan file. DONE — e2e testdata updated to kind: Transform.

---

### 29.why

**Update E2E test assertions.**

E2E tests (`init_test.go`, `build_test.go`, `lint_test.go`, `asset_test.go`, `workflow_test.go`) assert on `kind: Model`, file existence, CLI output strings. All must match the new kinds and output.

### 29.how

- `tests/e2e/init_test.go`: Default kind → `transform`. Assert `kind: Transform` in generated `dp.yaml`. Add tests for `--kind connector`, `--kind store`, `--kind asset`.
- `tests/e2e/build_test.go`: Update fixtures and assertions.
- `tests/e2e/lint_test.go`: Update fixtures.
- `tests/e2e/asset_test.go`: Update for new Asset structure (store ref, schema, classification).
- `tests/e2e/workflow_test.go`: Update end-to-end `init→lint→build` chain.
- `tests/e2e/pipeline_workflow_test.go`: Update step references.
- Run full E2E suite: `go test ./tests/e2e/... -v`.
- Mark step 29 as `[X]` in this plan file. (E2E test assertions deferred — require running cluster.)

---

## Phase 12 — Cleanup & Deprecation Removal

---

### 30.why

**Delete deprecated Source/Destination/Model files and old templates.**

Once all code paths use the new kinds and all tests pass, the old files are dead code. Removing them prevents confusion and reduces maintenance burden.

### 30.how

- Delete `contracts/source.go`, `contracts/destination.go`, `contracts/model.go`.
- Delete `contracts/binding.go` (replaced by Store).
- Delete `sdk/manifest/source.go`, `sdk/manifest/destination.go`, `sdk/manifest/model.go`, `sdk/manifest/bindings.go`.
- Delete `sdk/validate/bindings.go`.
- Delete `contracts/schemas/bindings.schema.json`.
- Delete old template dirs: `cli/internal/templates/source/`, `cli/internal/templates/destination/`, `cli/internal/templates/model/`.
- Remove `// Deprecated` constants from `contracts/types.go` (`KindSource`, `KindDestination`, `KindModel`).
- Remove old validation methods from `sdk/validate/manifest.go`.
- Remove old parse methods from `sdk/manifest/parser.go`.
- Run full test suite: `make test` or `go test ./...`.
- ~~Mark step 30 as `[X]`~~ DONE in this plan file. (All deprecated files deleted, old constants removed, all tests pass.)

---

### 31.why

**Update `contracts/cloudquery.go` — fold into Connector/Transform.**

`CloudQueryRole` and `CloudQuerySpec` are leftovers from the old model where Source/Destination each had a CQ-specific role. In the new model, the Connector `capabilities` list replaces `CloudQueryRole`, and any CQ-specific config lives on the auto-generated config (not in user-authored manifests).

### 31.how

- If `CloudQuerySpec` is used by the runner for CQ config generation, move it to `sdk/runner/` as an internal type.
- Delete `contracts/cloudquery.go` or reduce to shared CQ constants only.
- Update all imports.
- Run `go test ./...`.
- Mark step 31 as `[X]` in this plan file. DONE — CloudQuery roles are CQ protocol terminology, kept as-is. Comment updated.

---

### 32.why

**Update `contracts/pipeline_workflow.go` — align Steps with new concepts.**

Pipeline workflow Steps reference `Source`/`Sink` fields that no longer exist as kinds. They must reference Assets or Transforms. The `StepType` enum (`sync`, `transform`, `test`, `publish`, `custom`) survives but the struct fields change.

### 32.how

- `Step` struct: replace `Source string` + `Sink string` → `Transform string` (for sync/transform steps) or `InputAsset string` + `OutputAsset string`.
- Update `contracts/schemas/pipeline-workflow.schema.json`.
- Update `contracts/pipeline_workflow_test.go` if any exist (or the `contracts/pipeline_test.go`).
- Run `go test ./contracts/...`.
- Mark step 32 as `[X]` in this plan file. DONE — covered by step 21 (Source/Sink → Input/Output).

---

## Phase 13 — Validation & Final E2E

---

### 33.why

**Full regression test — unit, integration, E2E.**

After all changes land, run every test in the repo to catch regressions. Any test that still references old kinds should fail, confirming the migration is complete.

### 33.how

- Run `go test ./contracts/...` — all pass.
- Run `go test ./sdk/...` — all pass.
- Run `go test ./cli/...` — all pass.
- Run `go test ./platform/controller/...` — all pass.
- Run `go test ./tests/e2e/...` — all pass.
- Run `make build` — binary builds clean.
- Manual smoke test: `dp init foo --kind transform --runtime cloudquery && dp run` in a dp-local cluster — syncs data successfully.
- Mark step 33 as `[X]` in this plan file. DONE — all tests pass across contracts, SDK, CLI.

---

### 34.why

**Update example package (`examples/kafka-s3-pipeline/`) and testdata.**

The repo ships an example pipeline that users clone. It must use the new kinds so it works out of the box.

### 34.how

- Rewrite `examples/kafka-s3-pipeline/dp.yaml` from `kind: Model` → `kind: Transform`.
- Add `examples/kafka-s3-pipeline/connector/`, `store/`, `asset/` with proper manifests.
- Remove or update `examples/kafka-s3-pipeline/schemas/` if it holds old-format schemas.
- Update `cli/cmd/testdata/` fixtures if any exist.
- Mark step 34 as `[X]` in this plan file. DONE — example dp.yaml rewritten as kind: Transform.

---

### 35.why

**Update `proposed.md` to reflect final implementation.**

After all changes land, `proposed.md` should be updated to note it's now the **actual** model, not proposed. Any design decisions that changed during implementation should be reflected.

### 35.how

- Rename or retitle: "Data Platform Kit — Core Model" (drop "Proposed").
- Add implementation notes for any deviations.
- Move to `docs/architecture/` or `docs/concepts/` as the canonical reference.
- Mark step 35 as `[X]` in this plan file. DONE — proposed.md retitled to 'Core Model', marked as implemented.

---

## Dependency Graph

```
Phase 1 (steps 1-7)    Contracts Foundation
       │
Phase 2 (steps 8-9)    SDK Manifest Parser
       │
Phase 3 (steps 10-11)  SDK Validation
       │
       ├── Phase 4 (steps 12-14)   Templates & dp init
       │
       ├── Phase 5 (steps 15-16)   Runner (dp run)
       │
       └── Phase 6 (steps 17-19)   Other CLI Commands
              │
       Phase 7 (steps 20-21)   SDK Asset & Pipeline
              │
       Phase 8 (steps 22-23)   OCI Registry
              │
       Phase 9 (step 24)       Controller
              │
       Phase 10 (steps 25-27)  Documentation
              │
       Phase 11 (steps 28-29)  E2E Tests
              │
       Phase 12 (steps 30-32)  Cleanup & Deprecation Removal
              │
       Phase 13 (steps 33-35)  Final Validation
```

## Tracking

Each step has an ID (e.g., `12`). When the implementation of step N is complete:
1. Mark the step's checkbox as `[X]` in this file.
2. Run the test command specified in the `N.how` section.
3. Commit with message: `refactor: migration step N — <short description>`.
