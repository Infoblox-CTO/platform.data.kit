# Tasks: Registry Pull-Through Cache for k3d Local Development

**Input**: Design documents from `/specs/007-registry-cache/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Unit tests REQUIRED per Constitution Article VII (Quality Gates)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

## Path Conventions

Based on plan.md project structure:
- SDK: `sdk/localdev/` (cache.go, cache_test.go)
- CLI: `cli/cmd/` (dev.go modifications)
- Runtime: `.cache/` (generated config files)

---

## Phase 1: Setup

**Purpose**: Project initialization and basic structure

- [X] T001 Add `.cache/` to `.gitignore` at repository root
- [X] T002 [P] Create `sdk/localdev/cache.go` with package declaration and imports
- [X] T003 [P] Create `sdk/localdev/cache_test.go` with package declaration and test imports

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and utilities that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Define constants (DefaultContainerName, DefaultVolumeName, DefaultNetworkName, DefaultPort, DefaultCacheDir, DefaultRemoteURL, RegistryImage) in `sdk/localdev/cache.go`
- [X] T005 [P] Define CacheConfig struct with fields (ContainerName, VolumeName, NetworkName, Port, CacheDir, MirrorHost) in `sdk/localdev/cache.go`
- [X] T006 [P] Define CacheStatus struct with fields (Exists, Running, ConfigHash, Endpoint, VolumeSize) in `sdk/localdev/cache.go`
- [X] T007 [P] Define RegistryConfig and related YAML structs (LogConfig, StorageConfig, HTTPConfig, ProxyConfig) in `sdk/localdev/cache.go`
- [X] T008 [P] Define RegistriesYAML and RegistryMirror structs for k3d config in `sdk/localdev/cache.go`
- [X] T009 Define CacheManager struct with fields per data-model.md in `sdk/localdev/cache.go`
- [X] T010 [P] Implement NewCacheManager constructor with functional options in `sdk/localdev/cache.go`
- [X] T011 [P] Implement WithMirrorHost, WithPort, WithCacheDir option functions in `sdk/localdev/cache.go`
- [X] T012 [P] Implement detectMirrorHost() helper (check DEV_REGISTRY_MIRROR_HOST env, then default to host.k3d.internal) in `sdk/localdev/cache.go`
- [X] T013 [P] Implement configHash() helper using crypto/sha256 in `sdk/localdev/cache.go`
- [X] T014 [P] Add unit tests for NewCacheManager with default values in `sdk/localdev/cache_test.go`
- [X] T015 [P] Add unit tests for detectMirrorHost with env var override in `sdk/localdev/cache_test.go`
- [X] T016 [P] Add unit tests for configHash in `sdk/localdev/cache_test.go`

**Checkpoint**: Foundation ready - CacheManager can be instantiated, user story implementation can begin

---

## Phase 3: User Story 1 - Developer Starts Local Environment with Image Cache (Priority: P1) 🎯 MVP

**Goal**: `dp dev up` automatically starts registry cache before k3d cluster

**Independent Test**: Run `dp dev up`, verify container `dev-registry-cache` is running, verify `.cache/registries.yaml` exists

### Unit Tests for User Story 1

- [X] T017 [P] [US1] Add unit test for CacheManager.Up() creating new container in `sdk/localdev/cache_test.go`
- [X] T018 [P] [US1] Add unit test for CacheManager.Up() reusing existing container with matching hash in `sdk/localdev/cache_test.go`
- [X] T019 [P] [US1] Add unit test for writeRegistryConfig() generating valid YAML in `sdk/localdev/cache_test.go`
- [X] T020 [P] [US1] Add unit test for writeRegistriesYAML() generating valid k3d config in `sdk/localdev/cache_test.go`
- [X] T021 [P] [US1] Add unit test for GetRegistriesYAMLPath() returning correct path in `sdk/localdev/cache_test.go`

### Implementation for User Story 1

- [X] T022 [US1] Implement ensureCacheDir() to create .cache directory in `sdk/localdev/cache.go`
- [X] T023 [US1] Implement writeRegistryConfig() to write registry-config.yml per contracts/registry-config.schema.md in `sdk/localdev/cache.go`
- [X] T024 [US1] Implement writeRegistriesYAML() to write registries.yaml per contracts/registries.schema.md in `sdk/localdev/cache.go`
- [X] T025 [US1] Implement containerExists() using docker inspect in `sdk/localdev/cache.go`
- [X] T026 [US1] Implement containerRunning() using docker inspect state in `sdk/localdev/cache.go`
- [X] T027 [US1] Implement getContainerConfigHash() to read label dev.cache.config_sha256 in `sdk/localdev/cache.go`
- [X] T028 [US1] Implement ensureNetwork() to create Docker network if not exists in `sdk/localdev/cache.go`
- [X] T029 [US1] Implement createContainer() with all labels per contracts/container-labels.schema.md in `sdk/localdev/cache.go`
- [X] T030 [US1] Implement startContainer() using docker start in `sdk/localdev/cache.go`
- [X] T031 [US1] Implement removeContainer() using docker rm -f in `sdk/localdev/cache.go`
- [X] T032 [US1] Implement CacheManager.Up() orchestrating all above functions in `sdk/localdev/cache.go`
- [X] T033 [US1] Implement GetRegistriesYAMLPath() returning absolute path to registries.yaml in `sdk/localdev/cache.go`
- [X] T034 [US1] Implement Endpoint() returning full endpoint URL in `sdk/localdev/cache.go`
- [X] T035 [US1] Modify K3dManager.createCluster() to accept optional registriesPath and pass --registry-config flag in `sdk/localdev/k3d.go`
- [X] T036 [US1] Add unit test for K3dManager.createCluster() with registry config in `sdk/localdev/k3d_test.go`
- [X] T037 [US1] Modify runDevUp() to create CacheManager and call Up() before k3d in `cli/cmd/dev.go`
- [X] T038 [US1] Pass CacheManager.GetRegistriesYAMLPath() to K3dManager when creating cluster in `cli/cmd/dev.go`

**Checkpoint**: `dp dev up` starts cache before k3d, images can be pulled via cache

---

## Phase 4: User Story 2 - CI/CD Environments Skip Registry Cache (Priority: P1)

**Goal**: Cache operations are skipped when CI environment is detected

**Independent Test**: Set `CI=true`, run `dp dev up`, verify no dev-registry-cache container exists

### Unit Tests for User Story 2

- [X] T039 [P] [US2] Add unit test for IsCI() with CI=true in `sdk/localdev/cache_test.go`
- [X] T040 [P] [US2] Add unit test for IsCI() with GITHUB_ACTIONS=true in `sdk/localdev/cache_test.go`
- [X] T041 [P] [US2] Add unit test for IsCI() with JENKINS_URL set in `sdk/localdev/cache_test.go`
- [X] T042 [P] [US2] Add unit test for IsCI() returning false when no CI vars set in `sdk/localdev/cache_test.go`
- [X] T043 [P] [US2] Add unit test for CacheManager.Up() returning early in CI in `sdk/localdev/cache_test.go`

### Implementation for User Story 2

- [X] T044 [US2] Implement IsCI() checking CI, GITHUB_ACTIONS, JENKINS_URL env vars in `sdk/localdev/cache.go`
- [X] T045 [US2] Modify CacheManager.Up() to return early with message when IsCI() is true in `sdk/localdev/cache.go`
- [X] T046 [US2] Modify GetRegistriesYAMLPath() to return empty string when IsCI() is true in `sdk/localdev/cache.go`
- [X] T047 [US2] Modify K3dManager.createCluster() to skip --registry-config when path is empty in `sdk/localdev/k3d.go`

**Checkpoint**: CI environments work without registry cache, no errors

---

## Phase 5: User Story 3 - Developer Stops Local Environment (Priority: P2)

**Goal**: `dp dev down` stops cache container and optionally removes volume

**Independent Test**: Run `dp dev down`, verify container stopped; run with --volumes, verify volume removed

### Unit Tests for User Story 3

- [X] T048 [P] [US3] Add unit test for CacheManager.Down() stopping container in `sdk/localdev/cache_test.go`
- [X] T049 [P] [US3] Add unit test for CacheManager.Down() preserving volume by default in `sdk/localdev/cache_test.go`
- [X] T050 [P] [US3] Add unit test for CacheManager.Down() with removeVolume=true in `sdk/localdev/cache_test.go`

### Implementation for User Story 3

- [X] T051 [US3] Implement stopContainer() using docker stop in `sdk/localdev/cache.go`
- [X] T052 [US3] Implement removeVolume() using docker volume rm in `sdk/localdev/cache.go`
- [X] T053 [US3] Implement CacheManager.Down() orchestrating stop and optional volume removal in `sdk/localdev/cache.go`
- [X] T054 [US3] Modify runDevDown() to call CacheManager.Down() after k3d is stopped in `cli/cmd/dev.go`

**Checkpoint**: `dp dev down` properly cleans up cache resources

---

## Phase 6: User Story 4 - Idempotent Operations (Priority: P2)

**Goal**: Running `dp dev up` multiple times is safe and handles state transitions

**Independent Test**: Run `dp dev up` twice in a row, verify no errors

### Unit Tests for User Story 4

- [X] T055 [P] [US4] Add unit test for CacheManager.Up() when container already running with same config in `sdk/localdev/cache_test.go`
- [X] T056 [P] [US4] Add unit test for CacheManager.Up() when container stopped in `sdk/localdev/cache_test.go`
- [X] T057 [P] [US4] Add unit test for CacheManager.Up() when config hash differs (recreate) in `sdk/localdev/cache_test.go`

### Implementation for User Story 4

- [X] T058 [US4] Enhance CacheManager.Up() to detect running container with matching hash and return early in `sdk/localdev/cache.go`
- [X] T059 [US4] Enhance CacheManager.Up() to start stopped container with matching hash in `sdk/localdev/cache.go`
- [X] T060 [US4] Enhance CacheManager.Up() to remove and recreate container when hash differs in `sdk/localdev/cache.go`
- [X] T061 [US4] Add informative output messages for each state transition in `sdk/localdev/cache.go`

**Checkpoint**: Idempotent operations verified

---

## Phase 7: User Story 5 - Cross-Platform Compatibility (Priority: P2)

**Goal**: Cache works on macOS and Linux with different Docker runtimes

**Independent Test**: Test on macOS with Docker Desktop and Linux with native Docker

### Unit Tests for User Story 5

- [X] T062 [P] [US5] Add unit test for detectMirrorHost() with DEV_REGISTRY_MIRROR_HOST override in `sdk/localdev/cache_test.go`
- [X] T063 [P] [US5] Add unit test for detectMirrorHost() defaulting to host.k3d.internal in `sdk/localdev/cache_test.go`

### Implementation for User Story 5

- [X] T064 [US5] Verify detectMirrorHost() handles all platforms correctly per research.md in `sdk/localdev/cache.go`
- [X] T065 [US5] Add documentation comment explaining platform behavior in `sdk/localdev/cache.go`

**Checkpoint**: Cross-platform support verified

---

## Phase 8: Status Integration (P2)

**Goal**: `dp dev status` shows registry cache state

### Unit Tests

- [X] T066 [P] Add unit test for CacheManager.Status() returning correct state in `sdk/localdev/cache_test.go`

### Implementation

- [X] T067 Implement CacheManager.Status() returning CacheStatus with all fields in `sdk/localdev/cache.go`
- [X] T068 Modify runDevStatus() to include cache status in output in `cli/cmd/dev.go`

**Checkpoint**: Status command shows cache info

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, validation

- [X] T069 [P] Add package documentation comment to `sdk/localdev/cache.go`
- [X] T070 [P] Update `docs/tutorials/k3d-local-dev.md` with registry cache information
- [X] T071 [P] Add troubleshooting section to `docs/troubleshooting/common-issues.md` for cache issues
- [X] T072 Run all unit tests and verify passing: `go test ./sdk/localdev/...`
- [ ] T073 Run quickstart.md validation manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - can start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 - BLOCKS all user stories
- **Phase 3-7 (User Stories)**: All depend on Phase 2 completion
  - US1 and US2 are both P1 priority - implement US1 first (core functionality)
  - US3, US4, US5 are P2 and can proceed after US1/US2
- **Phase 8 (Status)**: Depends on US1 completion
- **Phase 9 (Polish)**: Depends on all user stories being complete

### User Story Dependencies

| Story | Depends On | Can Start After |
|-------|------------|-----------------|
| US1 (Cache Start) | Foundational | Phase 2 complete |
| US2 (CI Skip) | Foundational | Phase 2 complete (parallel with US1) |
| US3 (Cache Stop) | US1 | US1 complete |
| US4 (Idempotent) | US1 | US1 complete (parallel with US3) |
| US5 (Cross-Platform) | US1 | US1 complete (parallel with US3, US4) |

### Parallel Opportunities

**Within Phase 2 (Foundational)**:
```
T005, T006, T007, T008 (struct definitions) - all parallel
T010, T011, T012, T013 (helpers) - all parallel
T014, T015, T016 (unit tests) - all parallel
```

**Within User Story 1**:
```
T017, T018, T019, T020, T021 (tests) - all parallel
```

**Across User Stories (after Phase 2)**:
```
US1 and US2 can start in parallel (different concerns)
US3, US4, US5 can run in parallel after US1
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T016)
3. Complete Phase 3: User Story 1 (T017-T038)
4. **STOP and VALIDATE**: Run `dp dev up`, verify cache works
5. Deploy/demo if ready

### Recommended Order

1. **Phases 1-2**: Setup + Foundational (T001-T016)
2. **Phase 3**: US1 - Core cache functionality (T017-T038)
3. **Phase 4**: US2 - CI skip (T039-T047) - Can be parallel with US1
4. **Phase 5**: US3 - Cache stop (T048-T054)
5. **Phase 6**: US4 - Idempotency (T055-T061)
6. **Phase 7**: US5 - Cross-platform (T062-T065)
7. **Phase 8**: Status integration (T066-T068)
8. **Phase 9**: Polish (T069-T073)

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| Setup | T001-T003 | Project initialization |
| Foundational | T004-T016 | Core types and utilities |
| US1 (P1) | T017-T038 | Developer starts with cache 🎯 MVP |
| US2 (P1) | T039-T047 | CI/CD skips cache |
| US3 (P2) | T048-T054 | Developer stops environment |
| US4 (P2) | T055-T061 | Idempotent operations |
| US5 (P2) | T062-T065 | Cross-platform support |
| Status | T066-T068 | Status integration |
| Polish | T069-T073 | Documentation and validation |

**Total Tasks**: 73
**MVP Scope**: T001-T038 (38 tasks for US1)
**P1 Scope**: T001-T047 (47 tasks for US1+US2)

---

## Notes

- All tasks use table-driven tests per Constitution
- Docker CLI operations follow patterns from existing k3d.go
- Each user story is independently testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
