# Feature Specification: Asset Instances

**Feature Branch**: `011-asset-instances`
**Created**: 2026-02-14
**Status**: Draft
**Input**: User description: "Add an asset abstraction so data engineers can declare configured instances of approved extensions. `dp asset create <name> --ext cloudquery.source.aws --interactive` scaffolds an asset.yaml referencing the extension FQN and version, with a config block validated against the extension's schema.json. Assets are config-only (no code) — they declare what to run, not how. Asset types include source, sink, and model-engine. `dp asset validate` resolves the referenced extension from the registry and validates the config against its schema. The dp.yaml manifest gains an `assets` section that references asset definitions by name. Bindings remain per-environment but are now associated with assets rather than the top-level package."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Create a Source Asset from an Extension (Priority: P1)

A data engineer wants to pull data from AWS into the platform. Rather than writing code, they run `dp asset create aws_security --ext cloudquery.source.aws` and receive a scaffolded `asset.yaml` file referencing the extension by fully-qualified name and pinned version. The asset file contains a `config` block pre-populated with the extension's required fields (drawn from the extension's `schema.json`) and sensible defaults for optional fields. The data engineer fills in their specific configuration — AWS accounts, regions, tables — and the asset is ready to be wired into a pipeline.

**Why this priority**: Asset creation is the entry point for the data engineer persona. Without it, there is no way to declare "what to run" separately from "how it runs." This is the foundational building block that all subsequent features (pipelines, environments, policies) depend on.

**Independent Test**: Run `dp asset create my-source --ext cloudquery.source.aws` in a project directory. Verify it creates `assets/sources/my-source/asset.yaml` with the correct extension reference, version, and a config block whose keys match the extension's schema.

**Acceptance Scenarios**:

1. **Given** a project directory with a `dp.yaml`, **When** the data engineer runs `dp asset create my-source --ext cloudquery.source.aws`, **Then** the CLI creates `assets/sources/my-source/asset.yaml` with `extension: cloudquery.source.aws`, a `version` field pinned to the latest available version, and a `config` block containing the schema's required fields with placeholder values.
2. **Given** a project with no prior assets, **When** the data engineer creates an asset, **Then** the CLI creates the `assets/sources/` directory structure automatically.
3. **Given** the extension `cloudquery.source.aws` is not found in the registry, **When** the data engineer runs `dp asset create my-source --ext cloudquery.source.aws`, **Then** the CLI reports a clear error: "extension 'cloudquery.source.aws' not found in registry — check the extension FQN or run `dp ext list`."
4. **Given** the `--interactive` flag is provided, **When** the data engineer runs `dp asset create my-source --ext cloudquery.source.aws --interactive`, **Then** the CLI prompts for each required config field with descriptions from the schema, and writes the completed config to the asset file.
5. **Given** a source-type extension, **When** the asset is created, **Then** it is placed under `assets/sources/`. **Given** a sink-type extension, it is placed under `assets/sinks/`. **Given** a model-engine extension, it is placed under `assets/models/`.

---

### User Story 2 — Validate an Asset Against its Extension Schema (Priority: P1)

A data engineer has filled in their asset configuration and wants to verify it is correct before wiring it into a pipeline. They run `dp asset validate assets/sources/my-source/` (or `dp validate` from the project root, which now includes asset validation). The CLI resolves the extension by FQN and version from the registry, fetches its `schema.json`, and validates the asset's `config` block against it. Validation errors reference the specific field and constraint that failed, with a suggestion from the schema description.

**Why this priority**: Validation is co-equal with creation — an asset that can't be validated is just a YAML file. Shift-left validation against the extension schema is the core value proposition of the asset model (errors at `dp validate` time, not at runtime).

**Independent Test**: Create an asset with an invalid config field (wrong type, missing required field). Run `dp asset validate`. Verify the CLI reports the specific field, the expected type/constraint, and the schema description.

**Acceptance Scenarios**:

1. **Given** an asset referencing `cloudquery.source.aws@v24.0.2` with a valid config block, **When** the data engineer runs `dp asset validate assets/sources/my-source/`, **Then** the CLI reports "asset 'my-source' is valid" with exit code 0.
2. **Given** an asset with a missing required config field `tables`, **When** the data engineer runs `dp asset validate`, **Then** the CLI reports: "asset 'my-source': config.tables is required by extension cloudquery.source.aws — 'List of CloudQuery table names to sync'."
3. **Given** an asset with a config field of the wrong type (string instead of array), **When** the data engineer runs `dp asset validate`, **Then** the CLI reports the field, expected type, and actual type.
4. **Given** an asset referencing an extension version that does not exist in the registry, **When** the data engineer runs `dp asset validate`, **Then** the CLI reports: "extension 'cloudquery.source.aws@v99.0.0' not found — available versions: v24.0.2, v23.1.0, ..."
5. **Given** a project with multiple assets, **When** the data engineer runs `dp validate` from the project root, **Then** the CLI validates all assets in `assets/` in addition to dp.yaml, pipeline.yaml, and bindings.

---

### User Story 3 — Reference Assets in dp.yaml (Priority: P2)

A data engineer has created source and sink assets and wants to declare them as part of their data package. They add an `assets` section to `dp.yaml` that references asset definitions by name. The CLI resolves these references during `dp validate` and `dp build`, ensuring all referenced assets exist and are valid.

**Why this priority**: Connecting assets to the package manifest is what makes assets meaningful beyond standalone YAML files. However, assets can be created and validated independently first (P1), so this wiring is a follow-on step.

**Independent Test**: Add an `assets` section to dp.yaml referencing two asset names. Run `dp validate` and verify it resolves both assets and reports errors for any missing references.

**Acceptance Scenarios**:

1. **Given** a dp.yaml with `assets: [aws_security, snowflake_raw]` and both asset files exist under `assets/`, **When** the data engineer runs `dp validate`, **Then** the CLI resolves both references and validation passes.
2. **Given** a dp.yaml referencing an asset `missing_source` that has no corresponding asset.yaml, **When** the data engineer runs `dp validate`, **Then** the CLI reports: "asset 'missing_source' referenced in dp.yaml but not found in assets/ — run `dp asset create missing_source --ext <extension>`."
3. **Given** a dp.yaml with an `assets` section, **When** the data engineer runs `dp show`, **Then** the effective manifest includes the resolved asset names, their extension FQNs, and versions.

---

### User Story 4 — Associate Bindings with Assets (Priority: P2)

A data engineer has assets declared and wants to provide environment-specific infrastructure bindings. Instead of bindings being associated with the top-level package (as they are today), each binding is now associated with a specific asset by name. A sink asset references `binding: snowflake_raw`, and the bindings file maps `snowflake_raw` to concrete Snowflake connection details per environment. The existing bindings.yaml format is extended to associate binding entries with asset names.

**Why this priority**: Bindings are how assets connect to real infrastructure. This is essential before assets can be used in `dp run` or promoted across environments. However, it builds on the asset model from P1, so it follows creation and validation.

**Independent Test**: Create an asset with `binding: snowflake_raw`. Create a bindings.yaml with a `snowflake_raw` entry. Run `dp validate` and verify the CLI confirms the binding resolves correctly for the asset.

**Acceptance Scenarios**:

1. **Given** an asset `snowflake_raw` with `binding: snowflake_raw` and a bindings.yaml containing a `snowflake_raw` entry, **When** the data engineer runs `dp validate`, **Then** the CLI confirms the binding is resolved for the asset.
2. **Given** an asset referencing `binding: missing_binding` with no corresponding entry in bindings.yaml, **When** the data engineer runs `dp validate`, **Then** the CLI reports: "asset 'snowflake_raw' references binding 'missing_binding' which is not defined in bindings.yaml."
3. **Given** multiple assets each referencing different bindings, **When** bindings.yaml contains entries for all of them, **Then** `dp validate` passes and each asset's binding is independently resolved.
4. **Given** existing projects using the current top-level binding model, **When** the data engineer runs `dp validate`, **Then** the existing binding resolution continues to work — backward compatibility is maintained.

---

### User Story 5 — List and Inspect Assets (Priority: P3)

A data engineer wants to see all assets in their project, their extensions, versions, and validation status. They run `dp asset list` to get a summary table and `dp asset show <name>` to see the full resolved configuration for a specific asset.

**Why this priority**: Discoverability and inspection are important for usability but are not blocking for the core create-validate-reference workflow.

**Independent Test**: Create three assets in a project. Run `dp asset list` and verify all three appear with their extension, version, and type. Run `dp asset show <name>` and verify the full config is displayed.

**Acceptance Scenarios**:

1. **Given** a project with three assets (two sources, one sink), **When** the data engineer runs `dp asset list`, **Then** the CLI displays a table with columns: Name, Type, Extension, Version, Status.
2. **Given** an asset named `aws_security`, **When** the data engineer runs `dp asset show aws_security`, **Then** the CLI displays the full asset.yaml content with the resolved extension metadata.
3. **Given** a project with no assets, **When** the data engineer runs `dp asset list`, **Then** the CLI displays: "No assets found. Run `dp asset create <name> --ext <extension>` to create one."
4. **Given** the `--output json` flag, **When** the data engineer runs `dp asset list --output json`, **Then** the CLI outputs a JSON array of asset summaries.

---

### Edge Cases

- **Duplicate asset names**: If the data engineer runs `dp asset create aws_security` and an asset with that name already exists, the CLI reports an error and suggests using a different name or `--force` to overwrite.
- **Extension version conflicts**: If two assets reference different versions of the same extension, both are independently valid — version consistency across assets is a policy concern (feature 014), not an asset-level concern.
- **Asset name validation**: Asset names follow the same DNS-safe rules as package names (lowercase, alphanumeric, hyphens, 3–63 characters).
- **Circular or orphaned assets**: An asset that exists in `assets/` but is not referenced in dp.yaml is valid on its own — it is simply unused. `dp validate` SHOULD emit a warning for unreferenced assets.
- **Extension schema evolution**: If an extension releases a new version with additional required fields, existing assets pinned to the old version continue to validate. Assets referencing the new version must satisfy the new schema.
- **Offline validation**: If the registry is unreachable, `dp asset validate` reports a clear error suggesting `--offline` mode (which skips schema resolution and validates structure only).
- **Empty config block**: If an extension's schema has no required fields, an asset with an empty `config: {}` is valid.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST provide a `dp asset create <name> --ext <extension-fqn>` command that scaffolds an `asset.yaml` file in the appropriate subdirectory under `assets/` based on the extension's kind (sources/, sinks/, models/).
- **FR-002**: The `asset.yaml` file MUST contain: `name`, `type` (source, sink, model-engine), `extension` (FQN), `version` (semver), `owner_team`, and `config` (key-value block).
- **FR-003**: The `config` block in the scaffolded asset.yaml MUST be pre-populated with the extension's required fields (from `schema.json`) with placeholder values and descriptions as inline comments.
- **FR-004**: The CLI MUST provide an `--interactive` flag on `dp asset create` that prompts the user for each required config field with descriptions from the schema.
- **FR-005**: The CLI MUST provide a `dp asset validate` command that resolves the referenced extension from the registry, fetches its `schema.json`, and validates the asset's `config` block against it.
- **FR-006**: Validation errors MUST reference the specific config field, the constraint that failed (required, type, enum, pattern, etc.), and the schema description for that field.
- **FR-007**: The `dp.yaml` manifest MUST support an optional `assets` section containing a list of asset names that are part of the package.
- **FR-008**: `dp validate` MUST resolve all asset references in dp.yaml's `assets` section and verify that corresponding `asset.yaml` files exist under `assets/`.
- **FR-009**: `dp validate` MUST include asset validation (schema check against extension) as part of its standard validation pass when assets exist in the project.
- **FR-010**: Bindings in `bindings.yaml` MUST be resolvable by asset name — each asset's `binding` field references a named entry in the bindings file.
- **FR-011**: Existing projects using the current top-level binding model (without assets) MUST continue to work without modification — backward compatibility is required.
- **FR-012**: The CLI MUST provide `dp asset list` showing all assets with their name, type, extension, version, and validation status.
- **FR-013**: The CLI MUST provide `dp asset show <name>` displaying the full resolved asset configuration.
- **FR-014**: Asset names MUST follow DNS-safe naming rules (lowercase, alphanumeric, hyphens, 3–63 characters).
- **FR-015**: The `version` field in an asset MUST be a valid semver string referencing an available version of the extension.
- **FR-016**: The CLI MUST report a clear error when the referenced extension FQN or version is not found in the registry.
- **FR-017**: `dp validate` SHOULD emit a warning for assets that exist in `assets/` but are not referenced in dp.yaml's `assets` section.

### Key Entities

- **Asset**: A configured instance of an extension. Identified by name, scoped to a project. Contains an extension reference (FQN + version), a config block validated against the extension's schema, an optional binding reference, and an owner team. Types: source, sink, model-engine.
- **Extension Reference**: A fully-qualified name (e.g., `cloudquery.source.aws`) plus a semver version (e.g., `v24.0.2`) that uniquely identifies an extension definition in the registry. The extension provides the `schema.json` used to validate the asset's config.
- **Asset Directory**: The `assets/` directory tree under a project root, organized by type: `assets/sources/`, `assets/sinks/`, `assets/models/`. Each asset lives in its own subdirectory containing `asset.yaml`.
- **Assets Section (dp.yaml)**: An optional list in the package manifest that declares which assets are part of this package by name. Acts as the bridge between the package and its constituent assets.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A data engineer can create a new asset from an extension in under 30 seconds using `dp asset create`, including schema-aware config scaffolding.
- **SC-002**: Invalid asset configurations are caught at `dp validate` time with actionable error messages — zero schema-related failures at runtime.
- **SC-003**: 100% of required extension schema fields appear as placeholders in scaffolded asset files.
- **SC-004**: Existing projects without assets continue to pass `dp validate` with no changes — zero backward-compatibility regressions.
- **SC-005**: `dp asset list` displays all project assets within 1 second for projects with up to 50 assets.
- **SC-006**: Data engineers can create, validate, and reference assets without writing any code — assets are configuration-only.

## Assumptions

- **Extension registry exists**: This feature assumes that extensions are published to an OCI registry with a discoverable `schema.json`. If the extension type system (feature 011) is not yet complete, a built-in/embedded schema for the existing CloudQuery source extension will be used as the bootstrap path.
- **Single project scope**: Assets are scoped to a single project directory. Cross-project asset references are out of scope.
- **Schema format**: Extension schemas use JSON Schema (draft 2020-12 or compatible), consistent with the existing `contracts/schemas/` conventions.
- **Binding compatibility**: The existing `BindingsManifest` structure in contracts can be extended additively to associate bindings with asset names without breaking existing consumers.
- **Asset types are extension-driven**: The asset `type` (source, sink, model-engine) is determined by the extension's `kind` field, not declared independently by the data engineer.
