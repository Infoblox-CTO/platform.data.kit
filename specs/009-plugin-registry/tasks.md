# Tasks: Plugin Registry & Configuration Management

**Input**: Design documents from `/specs/009-plugin-registry/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Included — user explicitly requested CLI tests for config setting/loading.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Extend existing Config struct and project scaffolding for the new feature

- [X] T001 Add PluginsConfig and PluginOverride types to sdk/localdev/config.go (new struct fields with yaml tags, backward-compatible with existing Config)
- [X] T002 Add DefaultPluginRegistry, DefaultPluginVersions constants and ConfigScope type to sdk/localdev/config.go
- [X] T003 [P] Create empty cli/cmd/config.go with package declaration and import block (scaffold for dp config command)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core config loading infrastructure that ALL user stories depend on — hierarchical merge, git root detection, validation, and scope-aware read/write

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Tests

- [X] T004 [P] Add TestGitRepoRoot table-driven tests in sdk/localdev/config_test.go (in-repo, not-in-repo, git-not-found edge cases)
- [X] T005 [P] Add TestLoadHierarchicalConfig table-driven tests in sdk/localdev/config_test.go (system-only, user-only, repo-only, merge-all-three, missing-files-use-defaults, invalid-yaml-returns-error)
- [X] T006 [P] Add TestConfigScopePaths tests in sdk/localdev/config_test.go (RepoConfigPath, UserConfigPath, SystemConfigPath resolution)
- [X] T007 [P] Add TestValidate and TestValidateField table-driven tests in sdk/localdev/config_test.go (valid registry, invalid registry, valid version, invalid version, unknown key, mutually-exclusive image/version)
- [X] T008 [P] Add TestConfigSetField and TestConfigUnsetField tests in sdk/localdev/config_test.go (set scalar, set nested override, unset scalar, unset nested key)
- [X] T009 [P] Add TestBackwardCompatibility test in sdk/localdev/config_test.go (load existing dev-only YAML into new Config struct, verify Plugins gets zero value, save and reload round-trip)

### Implementation

- [X] T010 Implement gitRepoRoot() function in sdk/localdev/config.go (exec git rev-parse --show-toplevel, return empty string on error)
- [X] T011 Implement RepoConfigPath(), UserConfigPath(), SystemConfigPath() functions in sdk/localdev/config.go
- [X] T012 Implement LoadHierarchicalConfig() in sdk/localdev/config.go (load system → user → repo, yaml.Unmarshal chain, apply defaults for zero values)
- [X] T013 Implement Validate() method on Config struct in sdk/localdev/config.go (collect all errors, check registry URL format, version semver format, runtime enum, cluster name DNS-safe)
- [X] T014 Implement ValidateField(key, value string) function in sdk/localdev/config.go (single-field validation for dp config set, return error for unknown keys)
- [X] T015 Implement SetField(key, value string) and UnsetField(key string) methods on Config struct in sdk/localdev/config.go (dot-separated key path navigation, set/remove value at path)
- [X] T016 Implement LoadConfigForScope(scope ConfigScope) and SaveConfigForScope(config *Config, scope ConfigScope) in sdk/localdev/config.go

**Checkpoint**: Foundation ready — config loading, validation, and scope-aware I/O all work. User story implementation can begin.

---

## Phase 3: User Story 1 — Pull Destination Plugins from OCI Registry (Priority: P1) 🎯 MVP

**Goal**: Replace sparse-clone-and-build with docker pull → k3d image import → deploy as pod → gRPC sync

**Independent Test**: Run `dp run . --sync --destination postgresql` and verify image is pulled from ghcr.io, deployed as a pod, and sync completes via gRPC

### Tests for User Story 1

- [X] T017 [P] [US1] Add TestResolvePluginImage table-driven tests in cli/cmd/run_test.go (default registry+version, custom registry, version override, image override, image-beats-version precedence)
- [X] T018 [P] [US1] Add TestPullDestinationImage tests in cli/cmd/run_test.go (docker-not-found error, invalid-image-name error, mock success path)
- [X] T019 [P] [US1] Add TestGenerateSyncConfig_GrpcDestination tests in cli/cmd/run_test.go (verify destination uses registry: grpc with localhost port, not registry: local with binary path)
- [X] T020 [P] [US1] Update existing TestGenerateSyncConfig tests in cli/cmd/run_test.go (replace registry: local expectations with registry: grpc for file, postgresql, s3 destinations)

### Implementation for User Story 1

- [X] T021 [US1] Implement resolvePluginImage(name string, cfg *localdev.Config) function in cli/cmd/run.go (image resolution state machine: override.image → override.version → default, per data-model.md)
- [X] T022 [US1] Implement pullDestinationImage(imageRef string) function in cli/cmd/run.go (docker pull with os/exec, error handling for docker-not-running and image-not-found)
- [X] T023 [US1] Implement deployDestinationPod(imageRef, clusterName, namespace string) function in cli/cmd/run.go (k3d image import → kubectl run with --image-pull-policy=Never --port=7777, kubectl wait --for=condition=Ready)
- [X] T024 [US1] Implement cleanupDestinationPod(podName, namespace string) function in cli/cmd/run.go (kubectl delete pod, called in defer)
- [X] T025 [US1] Update generateSyncConfig in cli/cmd/run.go to use registry: grpc for destination (path: localhost:<destPort> instead of registry: local with binary path)
- [X] T026 [US1] Refactor runCmd.RunE sync block in cli/cmd/run.go to use new OCI flow: LoadHierarchicalConfig → resolvePluginImage → pullDestinationImage → deployDestinationPod → port-forward → generateSyncConfig → runCloudQuerySync → cleanup
- [X] T027 [US1] Add --registry flag to runCmd in cli/cmd/run.go (overrides config plugins.registry for single invocation, per FR-017)
- [X] T028 [US1] Remove ensureDestinationPlugin sparse-clone-and-build function from cli/cmd/run.go (replaced by pullDestinationImage + deployDestinationPod)
- [X] T029 [US1] Update supportedDestinations map in cli/cmd/run.go to remove tag/pluginSubDir fields (only need name and default version now)

**Checkpoint**: `dp run . --sync --destination file|postgresql|s3` works end-to-end with OCI image pulls. MVP is functional.

---

## Phase 4: User Story 2 — Hierarchical Configuration File (Priority: P2)

**Goal**: CLI loads config from three scopes (.dp/config.yaml → ~/.config/dp/config.yaml → /etc/datakit/config.yaml) with merge precedence

**Independent Test**: Create .dp/config.yaml with custom plugins.registry, run dp run . --sync, verify CLI uses the repo-scoped registry

### Tests for User Story 2

- [X] T030 [P] [US2] Add TestRunCmd_UsesHierarchicalConfig test in cli/cmd/run_test.go (create temp dir with .dp/config.yaml containing custom registry, verify resolvePluginImage uses it)
- [X] T031 [P] [US2] Add TestRunCmd_ConfigPrecedence test in cli/cmd/run_test.go (set different registry in user and repo scopes, verify repo wins)
- [X] T032 [P] [US2] Add TestRunCmd_RegistryFlagOverridesConfig test in cli/cmd/run_test.go (set registry in config, pass --registry flag, verify flag wins)

### Implementation for User Story 2

- [X] T033 [US2] Wire LoadHierarchicalConfig into runCmd.RunE in cli/cmd/run.go (replace LoadConfig call with LoadHierarchicalConfig, pass config to resolvePluginImage)
- [X] T034 [US2] Implement --registry flag precedence in cli/cmd/run.go (if --registry set, override config.Plugins.Registry before resolvePluginImage)
- [X] T035 [US2] Add observability logging to config loading in cli/cmd/run.go (print which config files were loaded and which registry is being used)

**Checkpoint**: Hierarchical config is live. `dp run --sync` reads from .dp/config.yaml → ~/.config/dp/config.yaml → /etc/datakit/config.yaml with correct precedence.

---

## Phase 5: User Story 3 — `dp config` Subcommand (Priority: P3)

**Goal**: `dp config set/get/unset/list` commands for managing settings without manual YAML editing

**Independent Test**: Run `dp config set plugins.registry ghcr.io/myteam`, then `dp config get plugins.registry`, verify it returns the set value

### Tests for User Story 3

- [X] T036 [P] [US3] Add TestConfigSetCmd table-driven tests in cli/cmd/config_test.go (set valid key, set invalid key, set with --scope repo, set with --scope user, invalid value rejected, creates config file if missing)
- [X] T037 [P] [US3] Add TestConfigGetCmd table-driven tests in cli/cmd/config_test.go (get existing key shows value+source, get unknown key errors, get built-in default shows "built-in")
- [X] T038 [P] [US3] Add TestConfigUnsetCmd tests in cli/cmd/config_test.go (unset existing key, unset non-existent key is no-op, --scope flag targets correct file)
- [X] T039 [P] [US3] Add TestConfigListCmd tests in cli/cmd/config_test.go (list all shows built-in + configured values with sources, --scope filters to single scope, table output format)

### Implementation for User Story 3

- [X] T040 [US3] Implement configCmd (parent) and configSetCmd cobra commands in cli/cmd/config.go (set key value with --scope flag, calls ValidateField + LoadConfigForScope + SetField + SaveConfigForScope)
- [X] T041 [US3] Implement configGetCmd cobra command in cli/cmd/config.go (load all scopes, resolve effective value, print value + source scope)
- [X] T042 [US3] Implement configUnsetCmd cobra command in cli/cmd/config.go (load scope config, UnsetField, save)
- [X] T043 [US3] Implement configListCmd cobra command in cli/cmd/config.go (load all scopes, iterate known keys, print table with KEY/VALUE/SOURCE columns, --scope filter)
- [X] T044 [US3] Register configCmd with rootCmd in cli/cmd/root.go (rootCmd.AddCommand(configCmd) in init())

**Checkpoint**: `dp config set/get/unset/list` all work. Developers can manage configuration without editing YAML.

---

## Phase 6: User Story 4 — Plugin Version and Image Overrides (Priority: P4)

**Goal**: Pin plugin versions or override entire image references via config, affecting `dp run --sync`

**Independent Test**: Run `dp config set plugins.overrides.postgresql.version v8.13.0`, then `dp run . --sync --destination postgresql`, verify v8.13.0 is pulled

### Tests for User Story 4

- [X] T045 [P] [US4] Add TestConfigSet_PluginOverride tests in cli/cmd/config_test.go (set version override, set image override, verify YAML serialization with nested overrides map)
- [X] T046 [P] [US4] Add TestResolvePluginImage_WithConfig table-driven tests in cli/cmd/run_test.go (version override changes tag, image override bypasses naming convention, unset override uses default)

### Implementation for User Story 4

- [X] T047 [US4] Extend ValidateField in sdk/localdev/config.go to handle plugins.overrides.<name>.version and plugins.overrides.<name>.image dynamic key patterns
- [X] T048 [US4] Extend SetField/UnsetField in sdk/localdev/config.go to handle nested map paths (plugins.overrides.postgresql.version creates map entry if needed)
- [X] T049 [US4] Extend configListCmd in cli/cmd/config.go to iterate plugins.overrides map entries and display each as plugins.overrides.<name>.version or plugins.overrides.<name>.image

**Checkpoint**: Plugin version pinning and image overrides work end-to-end through config and run commands.

---

## Phase 7: User Story 5 — Mirror Management (Priority: P5)

**Goal**: Add/remove fallback registries that are tried when the primary registry fails

**Independent Test**: Run `dp config add-mirror ghcr.io/backup-org`, verify mirror appears in `dp config list`, and dp run tries mirrors on primary failure

### Tests for User Story 5

- [X] T050 [P] [US5] Add TestConfigAddMirrorCmd tests in cli/cmd/config_test.go (add mirror, duplicate rejected, invalid URL rejected, --scope flag works)
- [X] T051 [P] [US5] Add TestConfigRemoveMirrorCmd tests in cli/cmd/config_test.go (remove existing mirror, remove non-existent errors)
- [X] T052 [P] [US5] Add TestPullWithMirrorFallback tests in cli/cmd/run_test.go (primary fails → tries mirrors in order, all fail → error lists all attempted registries)

### Implementation for User Story 5

- [X] T053 [US5] Implement configAddMirrorCmd cobra command in cli/cmd/config.go (validate URL, load scope config, append to mirrors with dedup, save)
- [X] T054 [US5] Implement configRemoveMirrorCmd cobra command in cli/cmd/config.go (load scope config, remove from mirrors, error if not found, save)
- [X] T055 [US5] Implement pullWithMirrorFallback(imageRef string, mirrors []string) function in cli/cmd/run.go (try primary, on failure try each mirror by replacing registry prefix, log which mirror succeeded)
- [X] T056 [US5] Wire mirror fallback into runCmd.RunE sync block in cli/cmd/run.go (replace direct pullDestinationImage call with pullWithMirrorFallback using config.Plugins.Mirrors)

**Checkpoint**: Mirror management works. `dp run --sync` falls back to mirrors when primary registry fails.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, and validation across all user stories

- [X] T057 [P] Update docs/reference/configuration.md to add plugins section documentation (registry, mirrors, overrides fields with examples, update config file locations to include .dp/config.yaml repo scope and /etc/datakit/config.yaml system scope)
- [X] T058 [P] Update docs/reference/cli.md to add dp config subcommand reference (set, get, unset, list, add-mirror, remove-mirror with synopsis, flags, examples)
- [X] T059 [P] Add help text to all cobra commands in cli/cmd/config.go (Short, Long, Example fields per contracts/dp-config-cli.md)
- [X] T060 Run go test ./... from cli/ and sdk/ to validate all tests pass (existing 150 + new tests)
- [X] T061 Run quickstart.md validation (execute steps 1-9 from specs/009-plugin-registry/quickstart.md against running k3d cluster)
- [X] T062 Remove stale sparse-clone helper functions from cli/cmd/run.go (sparseClonePlugin, pluginCacheDir, any unused destination binary path logic)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (T001, T002) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 completion — core MVP
- **US2 (Phase 4)**: Depends on Phase 2 + Phase 3 (uses resolvePluginImage + config loading together)
- **US3 (Phase 5)**: Depends on Phase 2 only (config set/get/unset/list operates on config infra, does not need OCI pull)
- **US4 (Phase 6)**: Depends on Phase 5 (extends dp config set with override key patterns)
- **US5 (Phase 7)**: Depends on Phase 3 + Phase 5 (extends pull flow with fallback, extends dp config with mirror commands)
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 — no dependencies on other stories
- **US2 (P2)**: Depends on US1 (wires hierarchical config into runCmd that already has OCI flow)
- **US3 (P3)**: Can start after Phase 2 — independent of US1/US2 (config commands don't need OCI pull)
- **US4 (P4)**: Depends on US3 (extends config set/unset with override patterns)
- **US5 (P5)**: Depends on US1 + US3 (extends both pull flow and config commands)

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Infrastructure/models before services
- Services before CLI wiring
- Core implementation before integration

### Parallel Opportunities

- All Phase 1 setup tasks can run in parallel (T001, T002, T003)
- All Phase 2 tests (T004-T009) can run in parallel
- Phase 2 implementation follows: T010 → T011 → T012, T013, T014, T015 can partially parallel → T016
- US1 tests (T017-T020) can all run in parallel
- US3 tests (T036-T039) can all run in parallel
- US3 (Phase 5) can run in parallel with US1 (Phase 3) after Phase 2 completes
- US4 tests (T045-T046) can run in parallel
- US5 tests (T050-T052) can run in parallel
- Phase 8 doc tasks (T057, T058, T059) can all run in parallel

---

## Parallel Example: After Phase 2 Completes

```
# Stream A: User Story 1 (OCI pull + pod deploy)
T017-T020: All US1 tests in parallel
T021-T029: US1 implementation sequentially

# Stream B: User Story 3 (dp config commands) — CAN RUN IN PARALLEL WITH STREAM A
T036-T039: All US3 tests in parallel
T040-T044: US3 implementation sequentially
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T016) — CRITICAL, blocks all stories
3. Complete Phase 3: User Story 1 (T017-T029)
4. **STOP and VALIDATE**: `dp run . --sync --destination file` works with OCI pull
5. Deploy/demo if ready — this is the MVP

### Incremental Delivery

1. Setup + Foundational → Config infrastructure ready
2. Add US1 → OCI pull works → **MVP!**
3. Add US2 → Hierarchical config live → Config files drive behavior
4. Add US3 → `dp config` commands → Self-service config management
5. Add US4 → Version/image overrides → Advanced customization
6. Add US5 → Mirror fallback → Enterprise resilience
7. Each story adds value without breaking previous stories

### Parallel Team Strategy

With two developers after Phase 2 completes:
- **Developer A**: US1 (Phase 3) → US2 (Phase 4) → US5 (Phase 7)
- **Developer B**: US3 (Phase 5) → US4 (Phase 6) → Phase 8 (Polish)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Total: 62 tasks (T001–T062)
- Tests: 27 test tasks (T004-T009, T017-T020, T030-T032, T036-T039, T045-T046, T050-T052)
- Implementation: 29 implementation tasks
- Polish: 6 tasks
