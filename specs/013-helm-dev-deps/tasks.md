# Tasks: Helm-Based Dev Dependencies

**Input**: Design documents from `/specs/013-helm-dev-deps/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/chart-definition.md, quickstart.md

**Tests**: Not explicitly requested in the feature specification. Test tasks are omitted. Existing tests will be updated as part of implementation tasks.

**Organization**: Tasks are grouped by user story (P1–P3) to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Define the core types and chart registry that all user stories depend on

- [X] T001 Create `ChartDefinition`, `PortForward`, and `DisplayEndpoint` types in sdk/localdev/charts/chart.go
- [X] T002 Create `ChartOverride` type in sdk/localdev/charts/chart.go
- [X] T003 Define `DefaultCharts` registry with all 4 chart definitions in sdk/localdev/charts/embed.go
- [X] T004 [P] Add `helm-deps` Makefile target to run `helm dependency build` for subchart charts in ./Makefile

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the uniform deployment function and refactor the k3d manager — MUST complete before user stories can be validated

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 Implement `DeployCharts()` function in sdk/localdev/charts/deploy.go (extract from embed.FS, apply overrides, parallel helm upgrade --install, return DeployResult)
- [X] T006 Implement `extractChart()` helper to write embedded chart to temp directory in sdk/localdev/charts/deploy.go
- [X] T007 Implement `applyOverrides()` helper to merge ChartOverride values into helm install args in sdk/localdev/charts/deploy.go
- [X] T008 Add unit tests for `DeployCharts`, `extractChart`, and `applyOverrides` in sdk/localdev/charts/deploy_test.go
- [X] T009 Refactor `K3dManager.deployCharts()` in sdk/localdev/k3d.go to delegate to `DeployCharts()` instead of inline chart deployment
- [X] T010 Refactor port-forwarding in sdk/localdev/portforward.go to derive rules from `ChartDefinition.PortForwards` instead of hardcoded map
- [X] T011 Refactor health checking in sdk/localdev/k3d.go to derive targets from `ChartDefinition.HealthLabels` instead of hardcoded labels
- [X] T012 Update `DefaultPorts` in sdk/localdev/ports.go to derive from `DefaultCharts` registry instead of hardcoded map
- [X] T013 Add `Charts map[string]ChartOverride` field to `DevConfig` struct in sdk/localdev/config.go
- [X] T014 Wire config overrides into `DeployCharts()` call path in sdk/localdev/k3d.go
- [X] T015 Update `cli/cmd/dev.go` to derive endpoint display and port lists from `DefaultCharts` instead of hardcoded values
- [X] T016 Update existing unit tests in sdk/localdev/k3d_test.go for new deployment flow
- [X] T017 Update existing unit tests in sdk/localdev/ports_test.go for derived port map
- [X] T018 Update existing unit tests in sdk/localdev/portforward_test.go for ChartDefinition-based forwarding
- [X] T019 Update existing config tests in sdk/localdev/config_test.go for `Charts` field parsing

**Checkpoint**: Foundation ready — uniform deploy mechanism in place, all orchestration code driven by ChartDefinition registry. User story chart work can now begin.

---

## Phase 3: User Story 1 — Uniform Dev Stack Startup via Helm Charts (Priority: P1) 🎯 MVP

**Goal**: All 4 dev dependencies deploy via the uniform Helm mechanism. `dp dev up/down/status` works end-to-end with the new architecture.

**Independent Test**: Run `dp dev up` on a fresh k3d cluster. All services (Redpanda, LocalStack, PostgreSQL, Marquez) start and become healthy. `dp dev status` shows per-chart health. `dp dev down` tears everything down.

### Implementation for User Story 1

- [X] T020 [P] [US1] Update embed directive in sdk/localdev/charts/embed.go to include `marquez` directory
- [X] T021 [P] [US1] Create Marquez chart directory with Chart.yaml (name: marquez, version: 0.2.0, appVersion: 0.51.1) in sdk/localdev/charts/marquez/Chart.yaml
- [X] T022 [P] [US1] Create Marquez default values in sdk/localdev/charts/marquez/values.yaml (API image, web image, ports, DB connection to shared postgres)
- [X] T023 [P] [US1] Create Marquez API server deployment template in sdk/localdev/charts/marquez/templates/deployment.yaml
- [X] T024 [P] [US1] Create Marquez Web UI deployment template in sdk/localdev/charts/marquez/templates/deployment-web.yaml
- [X] T025 [P] [US1] Create Marquez API service template (ports 5000, 5001) in sdk/localdev/charts/marquez/templates/service.yaml
- [X] T026 [P] [US1] Create Marquez Web service template (port 3000) in sdk/localdev/charts/marquez/templates/service-web.yaml
- [X] T027 [US1] Bump LocalStack image tag from 3.0 to 3.8.1 in sdk/localdev/charts/localstack/values.yaml
- [X] T028 [US1] Update LocalStack Chart.yaml version to 0.2.0 in sdk/localdev/charts/localstack/Chart.yaml
- [ ] T029 [US1] Verify `dp dev up` deploys all 4 charts, `dp dev status` reports health, `dp dev down` tears down (manual E2E validation)

**Checkpoint**: All 4 dev dependencies deploy uniformly via Helm. The dev stack is functional with basic services (no init jobs yet). This is the MVP — a working local dev environment with the new architecture.

---

## Phase 4: User Story 2 — Upstream Charts as Subcharts (Priority: P2)

**Goal**: Redpanda and PostgreSQL charts wrap upstream subcharts for production-grade defaults and reduced maintenance.

**Independent Test**: Inspect `Chart.yaml` for redpanda and postgres charts — each declares an upstream subchart dependency. Run `helm dependency build`, verify `.tgz` archives in `charts/`. Run `dp dev up` and verify services start using upstream chart templates.

### Implementation for User Story 2

- [X] T030 [P] [US2] Refactor sdk/localdev/charts/redpanda/Chart.yaml to declare upstream redpanda subchart dependency (version 25.3.2 from https://charts.redpanda.com)
- [X] T031 [P] [US2] Refactor sdk/localdev/charts/postgres/Chart.yaml to declare upstream bitnami/postgresql subchart dependency (version 18.3.0 from oci://registry-1.docker.io/bitnamicharts/postgresql)
- [X] T032 [US2] Configure dev-mode values for Redpanda upstream subchart in sdk/localdev/charts/redpanda/values.yaml (1 replica, no TLS, no persistence, console enabled, tuning disabled, resource limits)
- [X] T033 [US2] Remove custom Redpanda deployment.yaml and service.yaml templates from sdk/localdev/charts/redpanda/templates/ (delegated to subchart)
- [X] T034 [US2] Configure dev-mode values for PostgreSQL upstream subchart in sdk/localdev/charts/postgres/values.yaml (standalone, no persistence, nano preset, auth credentials)
- [X] T035 [US2] Remove custom PostgreSQL deployment.yaml and service.yaml templates from sdk/localdev/charts/postgres/templates/ (delegated to subchart)
- [X] T036 [US2] Run `helm dependency build` for redpanda chart, commit Chart.lock and charts/*.tgz to sdk/localdev/charts/redpanda/
- [X] T037 [US2] Run `helm dependency build` for postgres chart, commit Chart.lock and charts/*.tgz to sdk/localdev/charts/postgres/
- [ ] T038 [US2] Verify `dp dev up` deploys Redpanda and PostgreSQL using upstream subchart templates (manual E2E validation)

**Checkpoint**: Redpanda and PostgreSQL use upstream subcharts. LocalStack and Marquez remain custom charts. All 4 services deploy uniformly.

---

## Phase 5: User Story 3 — Init Jobs for Data Seeding (Priority: P2)

**Goal**: Each chart automatically seeds required resources (topics, buckets, schemas, namespaces) via Helm hooks so the dev environment is immediately usable.

**Independent Test**: Run `dp dev up` on a fresh cluster. Verify: Kafka topics exist (`dp.raw.events`, etc.), S3 buckets exist (`cdpp-raw`, etc.), PostgreSQL schema/tables exist, Marquez namespaces exist (`dp`, `dp-dev`, `analytics`).

### Implementation for User Story 3

- [X] T039 [P] [US3] Create Redpanda init-topics post-install hook Job template in sdk/localdev/charts/redpanda/templates/init-topics.yaml (creates topics: dp.raw.events, dp.processed.events, dp.errors.dlq, dp.audit.logs, dp.test.input, dp.test.output)
- [X] T040 [P] [US3] Create LocalStack init-buckets post-install hook Job template in sdk/localdev/charts/localstack/templates/init-buckets.yaml (creates buckets: cdpp-raw, cdpp-staging, cdpp-curated, cdpp-artifacts, cdpp-test)
- [X] T041 [P] [US3] Add initdb.scripts to PostgreSQL values for schema/table creation in sdk/localdev/charts/postgres/values.yaml
- [X] T042 [P] [US3] Create Marquez init-job post-install hook template in sdk/localdev/charts/marquez/templates/init-job.yaml (create marquez DB in shared PostgreSQL, seed namespaces dp, dp-dev, analytics)
- [ ] T043 [US3] Verify all init jobs complete successfully and resources are seeded after `dp dev up` (manual E2E validation)

**Checkpoint**: Dev environment is fully usable immediately after `dp dev up` — no manual setup steps required. Feature parity with docker-compose.

---

## Phase 6: User Story 4 — Chart Version Overrides via Config (Priority: P3)

**Goal**: Developers can override chart versions via `dp config set` to test with different dependency versions.

**Independent Test**: Run `dp config set dev.charts.redpanda.version 25.2.0`, then `dp dev up`, verify Redpanda deploys with the overridden version. Run `dp config unset dev.charts.redpanda.version`, verify revert to default.

### Implementation for User Story 4

- [X] T044 [US4] Implement version override resolution logic in sdk/localdev/charts/deploy.go (if ChartOverride.Version set, use override chart archive instead of embedded default)
- [X] T045 [US4] Add clear error message when override version is invalid or unavailable in sdk/localdev/charts/deploy.go
- [X] T046 [US4] Update config unit tests to verify `dp config set dev.charts.<name>.version` round-trip in sdk/localdev/config_test.go
- [ ] T047 [US4] Verify version override end-to-end: set override, deploy, verify version, unset, re-deploy (manual E2E validation)

**Checkpoint**: Chart version overrides work through the config system. Default embedded versions are used when no override is set.

---

## Phase 7: User Story 5 — Extra Helm Values via Config (Priority: P3)

**Goal**: Developers can pass additional Helm values to any chart via config for customization.

**Independent Test**: Run `dp config set dev.charts.postgres.values.primary.resources.limits.memory 512Mi`, then `dp dev up`, verify PostgreSQL pod has custom memory limit.

### Implementation for User Story 5

- [X] T048 [US5] Implement values override merging in sdk/localdev/charts/deploy.go (convert ChartOverride.Values map to --set flags for helm install)
- [X] T049 [US5] Add config unit tests for `dp config set dev.charts.<name>.values.<path>` parsing in sdk/localdev/config_test.go
- [ ] T050 [US5] Verify values override end-to-end: set custom value, deploy, verify applied, unset, re-deploy (manual E2E validation)

**Checkpoint**: Extra Helm values flow from config through to helm install. Power users can customize any chart parameter without editing chart files.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, build integration, E2E tests, and final validation

- [X] T051 [P] Update getting-started documentation in docs/getting-started/quickstart.md to reflect new Marquez endpoints and Helm-based stack
- [X] T052 [P] Update CLI reference documentation in docs/reference/cli.md for new `dp dev status` output format
- [X] T053 [P] Update architecture documentation in docs/architecture.md to describe uniform Helm chart mechanism
- [X] T054 Add or update E2E test for `dp dev up/down/status` with all 4 charts in tests/e2e/
- [X] T055 Verify offline operation: build CLI, disconnect network, run `dp dev up` with default embedded charts (SC-004 validation)
- [X] T056 Verify backward compatibility: run existing E2E and integration tests against new implementation (SC-006 validation)
- [X] T057 Run quickstart.md validation — execute all commands from specs/013-helm-dev-deps/quickstart.md against a fresh k3d cluster
- [X] T058 Code cleanup: remove any remaining hardcoded service lists, port maps, or health targets across sdk/localdev/ and cli/cmd/

**Checkpoint**: Feature complete — all documentation updated, E2E tests passing, backward compatibility confirmed, offline operation verified.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Phase 2 — delivers MVP (all 4 charts deploy uniformly)
- **User Story 2 (Phase 4)**: Depends on Phase 2 — can run in parallel with US1 (different chart files)
- **User Story 3 (Phase 5)**: Depends on Phase 3 (Marquez chart) and Phase 4 (upstream subcharts for init hooks) — must wait for chart structure to be finalized
- **User Story 4 (Phase 6)**: Depends on Phase 2 — config override plumbing in foundational phase
- **User Story 5 (Phase 7)**: Depends on Phase 6 — builds on version override logic
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational — creates Marquez chart, bumps LocalStack
- **US2 (P2)**: Independent after Foundational — refactors Redpanda and PostgreSQL to upstream subcharts
- **US3 (P2)**: Depends on US1 (Marquez chart exists) and US2 (upstream chart structure finalized for init hooks)
- **US4 (P3)**: Independent after Foundational — adds version override logic
- **US5 (P3)**: Depends on US4 — extends override logic to values

### Within Each User Story

- Models/types before services
- Services before CLI integration
- Core implementation before validation
- Story complete before next priority (unless parallelized)

### Parallel Opportunities

- Phase 1: T004 (Makefile) can run in parallel with T001–T003
- Phase 2: T016–T019 (test updates) can be parallelized after T009–T015
- Phase 3: T020–T026 (Marquez chart templates) are all parallelizable
- Phase 4: T030–T031 (Chart.yaml refactors) can run in parallel
- Phase 5: T039–T042 (all init job templates) can run in parallel
- Phase 8: T051–T053 (doc updates) can run in parallel

---

## Parallel Example: User Story 1 (Phase 3)

```bash
# Launch all Marquez chart templates in parallel:
Task: T021 "Create Marquez Chart.yaml in sdk/localdev/charts/marquez/Chart.yaml"
Task: T022 "Create Marquez values.yaml in sdk/localdev/charts/marquez/values.yaml"
Task: T023 "Create Marquez API deployment in sdk/localdev/charts/marquez/templates/deployment.yaml"
Task: T024 "Create Marquez Web deployment in sdk/localdev/charts/marquez/templates/deployment-web.yaml"
Task: T025 "Create Marquez API service in sdk/localdev/charts/marquez/templates/service.yaml"
Task: T026 "Create Marquez Web service in sdk/localdev/charts/marquez/templates/service-web.yaml"

# Then sequentially:
Task: T020 "Update embed directive in sdk/localdev/charts/embed.go"
Task: T027 "Bump LocalStack image tag in sdk/localdev/charts/localstack/values.yaml"
Task: T028 "Update LocalStack Chart.yaml version in sdk/localdev/charts/localstack/Chart.yaml"
Task: T029 "Manual E2E validation of dp dev up/down/status"
```

## Parallel Example: User Story 3 (Phase 5)

```bash
# Launch all init job templates in parallel:
Task: T039 "Create Redpanda init-topics hook in sdk/localdev/charts/redpanda/templates/init-topics.yaml"
Task: T040 "Create LocalStack init-buckets hook in sdk/localdev/charts/localstack/templates/init-buckets.yaml"
Task: T041 "Add initdb scripts to PostgreSQL values in sdk/localdev/charts/postgres/values.yaml"
Task: T042 "Create Marquez init-job hook in sdk/localdev/charts/marquez/templates/init-job.yaml"

# Then sequentially:
Task: T043 "Manual E2E validation of init job completion"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T004)
2. Complete Phase 2: Foundational (T005–T019)
3. Complete Phase 3: User Story 1 (T020–T029)
4. **STOP and VALIDATE**: `dp dev up` deploys all 4 services uniformly
5. This delivers: uniform Helm mechanism, Marquez added, LocalStack bumped

### Incremental Delivery

1. Setup + Foundational → Uniform deploy mechanism ready
2. User Story 1 → MVP: all 4 charts deploy uniformly → Validate
3. User Story 2 → Upstream subcharts for Redpanda + PostgreSQL → Validate
4. User Story 3 → Init jobs seed all resources automatically → Validate
5. User Story 4 → Chart version overrides via config → Validate
6. User Story 5 → Extra Helm values via config → Validate
7. Polish → Docs, E2E tests, final validation

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (Marquez chart + LocalStack bump)
   - Developer B: User Story 2 (upstream subcharts for Redpanda + PostgreSQL)
3. After US1 + US2 complete:
   - Developer A: User Story 3 (init jobs — needs both chart structures finalized)
   - Developer B: User Story 4 + 5 (config overrides)
4. Team completes Polish together

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable after Foundational phase
- Tests were NOT explicitly requested in the spec — existing test files are updated as part of implementation
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- The `helm dependency build` tasks (T036, T037) require network access — run before offline validation
