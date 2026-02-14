# Tasks: Asset Instances

**Input**: Design documents from `/specs/011-asset-instances/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Tests**: Tests are included — the spec requires unit tests (Constitution Article VII, Go Testing Requirements).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Exact file paths included in every task description

## Path Conventions

- **contracts/**: Shared types and schemas (`contracts` module)
- **sdk/**: Validation, loading, scaffolding (`sdk` module)
- **cli/cmd/**: CLI commands (`cli` module)
- **tests/e2e/**: End-to-end tests

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization — new dependencies and embedded schemas

- [ ] T001 Add `github.com/santhosh-tekuri/jsonschema/v6` dependency to `sdk/go.mod`
- [ ] T002 [P] Create embedded CloudQuery source extension schema in `sdk/asset/schemas/cloudquery.source.schema.json` with `//go:embed` in `sdk/asset/schemas/embed.go`
- [ ] T003 [P] Create `asset.schema.json` in `contracts/schemas/asset.schema.json` (copy from `specs/011-asset-instances/contracts/asset.schema.json`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and utilities that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 Implement `AssetType`, `AssetManifest`, `ParseExtensionFQN()`, and media type constants in `contracts/asset.go` per `specs/011-asset-instances/contracts/go-types.md`
- [ ] T005 [P] Implement unit tests for `AssetType.IsValid()`, `ParseExtensionFQN()`, and `AssetManifest` serialization in `contracts/asset_test.go`
- [ ] T006 Add `Assets []string` field (with `omitempty`) to `DataPackageSpec` in `contracts/datapackage.go` after the `Outputs` field
- [ ] T007 [P] Add `Asset string` field (with `omitempty`) to `Binding` struct in `contracts/binding.go` after the `Name` field
- [ ] T008 [P] Update existing `contracts/datapackage_test.go` to verify `Assets` field serializes/deserializes correctly with backward compatibility (no assets = no field in YAML)
- [ ] T009 [P] Update existing `contracts/binding_test.go` to verify `Asset` field serializes/deserializes correctly with backward compatibility (no asset = no field in YAML)
- [ ] T010 Add `assets` property to `spec.properties` in `contracts/schemas/dp-manifest.schema.json` — array of DNS-safe strings
- [ ] T011 [P] Add optional `asset` property to `$defs.binding.properties` in `contracts/schemas/bindings.schema.json` — DNS-safe string pattern
- [ ] T012 Implement asset YAML loader in `sdk/asset/loader.go`: `LoadAsset(path) (*contracts.AssetManifest, error)` and `LoadAllAssets(projectDir) ([]*contracts.AssetManifest, error)` — walks `assets/` directory tree, parses each `asset.yaml`
- [ ] T013 Implement unit tests for asset loader in `sdk/asset/loader_test.go` — valid asset, missing file, malformed YAML, all three type subdirectories
- [ ] T014 Implement extension schema resolver in `sdk/asset/resolver.go`: `SchemaResolver` interface with `ResolveSchema(ctx, fqn, version) ([]byte, error)`, `EmbeddedResolver` (uses `//go:embed`), `CachingResolver` (wraps registry + local cache at `~/.cache/dp/schemas/`)
- [ ] T015 [P] Implement unit tests for schema resolver in `sdk/asset/resolver_test.go` — embedded fallback, cache hit, cache miss with mock registry

**Checkpoint**: Foundation ready — contracts extended, loader and resolver available, all existing tests pass

---

## Phase 3: User Story 1 — Create a Source Asset from an Extension (Priority: P1) 🎯 MVP

**Goal**: Data engineer runs `dp asset create <name> --ext <fqn>` and gets a scaffolded `asset.yaml` with schema-aware config placeholders

**Independent Test**: Run `dp asset create my-source --ext cloudquery.source.aws` in a project directory. Verify `assets/sources/my-source/asset.yaml` is created with correct extension reference, version, and config block matching the schema's required fields.

### Tests for User Story 1

- [ ] T016 [P] [US1] Unit tests for scaffolder in `sdk/asset/scaffolder_test.go` — scaffold from embedded schema, correct directory placement by type, placeholder config generation, duplicate name detection, name validation (DNS-safe)
- [ ] T017 [P] [US1] Unit tests for `dp asset create` command in `cli/cmd/asset_create_test.go` — table-driven tests for success, missing `--ext`, invalid FQN, duplicate asset, `--interactive` flag, `--force` overwrite

### Implementation for User Story 1

- [ ] T018 [US1] Implement scaffolder in `sdk/asset/scaffolder.go`: `Scaffold(opts ScaffoldOpts) error` — resolves extension schema, extracts required fields, creates `assets/<type>/<name>/asset.yaml` with placeholder config and inline YAML comments from schema descriptions
- [ ] T019 [US1] Implement `dp asset` root command in `cli/cmd/asset.go` — Cobra parent command with `Use: "asset"`, `Short: "Manage data package assets"`, registered as subcommand of root in `cli/cmd/root.go`
- [ ] T020 [US1] Implement `dp asset create` command in `cli/cmd/asset_create.go` — flags: `--ext` (required), `--interactive`, `--force`; validates name (DNS-safe, FR-014), parses FQN, calls scaffolder, prints success message with file path
- [ ] T021 [US1] Implement interactive mode in `cli/cmd/asset_create.go` — when `--interactive` is set, prompt for each required config field using schema descriptions; write completed config to `asset.yaml`

**Checkpoint**: `dp asset create <name> --ext <fqn>` works end-to-end. Asset files are scaffolded with correct structure. Tests pass.

---

## Phase 4: User Story 2 — Validate an Asset Against its Extension Schema (Priority: P1)

**Goal**: Data engineer runs `dp asset validate` and gets schema-validated feedback on their config block — errors reference specific fields with schema descriptions

**Independent Test**: Create an asset with an invalid config (wrong type, missing required field). Run `dp asset validate`. Verify specific field, expected type/constraint, and schema description are reported.

### Tests for User Story 2

- [ ] T022 [P] [US2] Unit tests for `AssetValidator` in `sdk/validate/asset_test.go` — valid asset, missing required config field, wrong type, invalid FQN, invalid version, type/kind mismatch, empty config with no required fields
- [ ] T023 [P] [US2] Unit tests for `dp asset validate` command in `cli/cmd/asset_validate_test.go` — single asset path, project-wide validation, offline mode, non-existent path

### Implementation for User Story 2

- [ ] T024 [US2] Implement `AssetValidator` in `sdk/validate/asset.go`: structural validation (required fields, name pattern, FQN format, version semver, type/kind match) + schema validation (resolve extension schema via `SchemaResolver`, validate `Config` map against it using `jsonschema/v6`, convert `ValidationError` causes to `contracts.ValidationError` with field path, constraint, and schema description)
- [ ] T025 [US2] Integrate asset validation into `AggregateValidator.Validate()` in `sdk/validate/aggregate.go` — after bindings validation, walk `assets/` directory, load and validate each asset; include results in `ValidationResult`
- [ ] T026 [US2] Implement `dp asset validate` command in `cli/cmd/asset_validate.go` — accepts optional path argument (defaults to `assets/` in current directory), calls `AssetValidator`, formats errors with field path and schema description, exit code 0 on success / 1 on failure
- [ ] T027 [US2] Add asset error codes to `sdk/validate/validator.go`: `ErrAssetRequired = "E070"`, `ErrAssetInvalidFQN = "E071"`, `ErrAssetInvalidVersion = "E072"`, `ErrAssetTypeMismatch = "E073"`, `ErrAssetSchemaValidation = "E074"`, `ErrAssetExtNotFound = "E075"`

**Checkpoint**: `dp asset validate` validates single assets and all assets via `dp validate`. Schema errors report specific fields with actionable messages. Tests pass.

---

## Phase 5: User Story 3 — Reference Assets in dp.yaml (Priority: P2)

**Goal**: Data engineer adds `assets: [name1, name2]` to dp.yaml; `dp validate` resolves references and verifies assets exist and are valid

**Independent Test**: Add `assets` section to dp.yaml referencing two asset names. Run `dp validate`. Verify both resolve correctly; verify missing references produce clear errors.

### Tests for User Story 3

- [ ] T028 [P] [US3] Unit tests for dp.yaml asset reference validation in `sdk/validate/datapackage_test.go` — test `assets` field with existing assets, missing assets, empty list, and unreferenced asset warning (FR-017)

### Implementation for User Story 3

- [ ] T029 [US3] Extend `DataPackageValidator.Validate()` in `sdk/validate/datapackage.go` — if `Assets` field is non-empty, resolve each name to an `asset.yaml` under `assets/`; report `E076` for missing references with actionable suggestion (`run dp asset create <name> --ext <extension>`)
- [ ] T030 [US3] Add orphan asset warning in `sdk/validate/aggregate.go` — after asset validation, compare assets found in `assets/` vs. assets referenced in `dp.yaml`; emit warning (FR-017) for any unreferenced assets
- [ ] T031 [US3] Update `dp show` command in `cli/cmd/show.go` — when dp.yaml has `assets` section, include resolved asset names, extension FQNs, and versions in the effective manifest output
- [ ] T032 [US3] Add test data for dp.yaml with assets in `sdk/validate/testdata/` — `valid-with-assets/dp.yaml`, `missing-asset-ref/dp.yaml`

**Checkpoint**: `dp validate` resolves asset references in dp.yaml. Missing references produce actionable errors. Orphan assets emit warnings. `dp show` displays resolved assets.

---

## Phase 6: User Story 4 — Associate Bindings with Assets (Priority: P2)

**Goal**: Bindings become asset-scoped — each asset's `binding` field references a named entry in `bindings.yaml`. Backward compatibility with top-level bindings is maintained.

**Independent Test**: Create an asset with `binding: raw-output`. Create bindings.yaml with a `raw-output` entry. Run `dp validate`. Verify binding resolves correctly. Then test with a missing binding name.

### Tests for User Story 4

- [ ] T033 [P] [US4] Unit tests for asset-binding resolution in `sdk/validate/bindings_test.go` — asset binding matches, asset binding missing, mixed asset-scoped and top-level bindings, backward compat with no asset field

### Implementation for User Story 4

- [ ] T034 [US4] Extend `BindingsValidator.Validate()` in `sdk/validate/bindings.go` — when asset loader provides loaded assets, cross-validate: for each asset with a `binding` field, verify a binding entry with that `name` exists in bindings.yaml; report `E077` for unresolved asset bindings
- [ ] T035 [US4] Ensure backward compatibility in `sdk/validate/bindings.go` — existing validation logic for top-level bindings (without `asset` field) continues to work unchanged; add integration test in `sdk/validate/bindings_test.go` verifying existing test fixtures still pass
- [ ] T036 [US4] Add test data for asset-scoped bindings in `sdk/validate/testdata/` — `asset-bindings/bindings.yaml`, `asset-bindings/assets/sources/my-source/asset.yaml`

**Checkpoint**: Bindings resolve for assets. Backward compatibility is confirmed — existing projects without assets pass `dp validate` with no changes.

---

## Phase 7: User Story 5 — List and Inspect Assets (Priority: P3)

**Goal**: Data engineer runs `dp asset list` for a summary table and `dp asset show <name>` for full details

**Independent Test**: Create three assets. Run `dp asset list` and verify table output. Run `dp asset show <name>` and verify full config display.

### Tests for User Story 5

- [ ] T037 [P] [US5] Unit tests for `dp asset list` in `cli/cmd/asset_list_test.go` — table output with 3 assets, empty project, `--output json` flag
- [ ] T038 [P] [US5] Unit tests for `dp asset show` in `cli/cmd/asset_show_test.go` — valid asset name, non-existent asset name, YAML vs JSON output

### Implementation for User Story 5

- [ ] T039 [US5] Implement `dp asset list` command in `cli/cmd/asset_list.go` — loads all assets via `sdk/asset/loader.go`, displays table with columns: Name, Type, Extension, Version, Status; supports `--output json` flag for JSON array output; displays helpful message when no assets found
- [ ] T040 [US5] Implement `dp asset show` command in `cli/cmd/asset_show.go` — takes asset name argument, resolves to `asset.yaml` path, loads and displays full asset content with resolved extension metadata; supports `--output json` flag

**Checkpoint**: `dp asset list` and `dp asset show` work correctly. All user stories are independently functional.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, validation completeness, and end-to-end testing

- [ ] T041 [P] Add asset documentation section to `docs/concepts/` — create `docs/concepts/assets.md` covering asset abstraction, types, lifecycle, and relationship to extensions
- [ ] T042 [P] Update `docs/reference/cli.md` with `dp asset create`, `dp asset validate`, `dp asset list`, `dp asset show` command reference
- [ ] T043 [P] Update `docs/reference/manifest-schema.md` with `asset.yaml` schema reference and dp.yaml `assets` field documentation
- [ ] T044 Add `assets.md` to navigation in `mkdocs.yml` under concepts section
- [ ] T045 Implement end-to-end test in `tests/e2e/asset_test.go` — full workflow: create asset → validate → add to dp.yaml → validate project → list → show
- [ ] T046 Run quickstart.md validation — execute all commands from `specs/011-asset-instances/quickstart.md` against a test project to verify documentation accuracy

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — first user story, scaffolding
- **US2 (Phase 4)**: Depends on Phase 2 — can run in parallel with US1 (different files)
- **US3 (Phase 5)**: Depends on Phase 2 + T012 (loader) — can overlap with US1/US2
- **US4 (Phase 6)**: Depends on Phase 2 + T012 (loader) — can overlap with US1/US2/US3
- **US5 (Phase 7)**: Depends on Phase 2 + T012 (loader) — can overlap with US1-US4
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends only on Foundational (Phase 2). No dependencies on other stories.
- **US2 (P1)**: Depends only on Foundational (Phase 2). Can run in parallel with US1.
- **US3 (P2)**: Depends on Foundational. Uses asset loader (T012) but no dependency on US1/US2 commands.
- **US4 (P2)**: Depends on Foundational. Uses asset loader (T012) but no dependency on US1/US2/US3 commands.
- **US5 (P3)**: Depends on Foundational. Uses asset loader (T012) but no dependency on US1-US4 commands.

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Foundation types (contracts) before SDK logic
- SDK logic before CLI commands
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- T002 and T003 (Phase 1) can run in parallel
- T005, T007, T008, T009, T011, T015 (Phase 2) can run in parallel after T004/T006
- US1 and US2 can start in parallel after Phase 2 (different files entirely)
- US3, US4, US5 can start in parallel after Phase 2 (different files)
- Within each US, test tasks marked [P] can run in parallel
- All Phase 8 doc tasks marked [P] can run in parallel

---

## Parallel Example: User Story 1

```bash
# After Phase 2 is complete, launch tests first:
Task T016: "Unit tests for scaffolder in sdk/asset/scaffolder_test.go"
Task T017: "Unit tests for dp asset create in cli/cmd/asset_create_test.go"

# Then implement (T018 first, then T019-T021):
Task T018: "Implement scaffolder in sdk/asset/scaffolder.go"
Task T019: "Implement dp asset root command in cli/cmd/asset.go"  # can parallel with T018
Task T020: "Implement dp asset create in cli/cmd/asset_create.go"  # depends on T018+T019
Task T021: "Implement interactive mode in cli/cmd/asset_create.go"  # depends on T020
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Setup (3 tasks)
2. Complete Phase 2: Foundational (12 tasks)
3. Complete Phase 3: US1 — Asset Creation (6 tasks)
4. Complete Phase 4: US2 — Asset Validation (6 tasks)
5. **STOP and VALIDATE**: `dp asset create` + `dp asset validate` works end-to-end
6. Deploy/demo if ready — this is the MVP

### Incremental Delivery

1. Setup + Foundational → Foundation ready (15 tasks)
2. Add US1 + US2 → Test independently → **MVP!** (12 tasks)
3. Add US3 → dp.yaml integration → Deploy/Demo (5 tasks)
4. Add US4 → Binding association → Deploy/Demo (4 tasks)
5. Add US5 → List/Show commands → Deploy/Demo (4 tasks)
6. Polish → Documentation + E2E → Ship (6 tasks)

### Parallel Team Strategy

With two developers after Phase 2:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: US1 (create) → US3 (dp.yaml refs) → US5 (list/show)
   - Developer B: US2 (validate) → US4 (bindings) → Polish
3. Stories integrate cleanly — different files, shared contracts

---

## Summary

| Metric | Count |
|--------|-------|
| **Total tasks** | **46** |
| Phase 1 (Setup) | 3 |
| Phase 2 (Foundational) | 12 |
| US1 — Create Asset (P1) | 6 |
| US2 — Validate Asset (P1) | 6 |
| US3 — dp.yaml References (P2) | 5 |
| US4 — Asset Bindings (P2) | 4 |
| US5 — List/Show (P3) | 4 |
| Phase 8 (Polish) | 6 |
| **Parallel tasks** | **20** (43%) |
| **MVP scope (US1+US2)** | **27 tasks** |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Tests written before implementation (TDD within each story)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Backward compatibility verified in US4 before merging binding changes
