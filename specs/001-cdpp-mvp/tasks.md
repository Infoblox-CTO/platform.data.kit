# Tasks: CDPP MVP

**Input**: Design documents from `/specs/001-cdpp-mvp/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

**Tests**: Not explicitly requested - test tasks excluded per template guidelines

**Organization**: Tasks grouped by user story to enable independent implementation and testing

---

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: User story mapping (US1, US2, US3, US4, US5, US6)
- All paths are absolute from repository root

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Initialize Go monorepo with module structure per plan.md

- [x] T001 Create project directory structure per plan.md (contracts/, sdk/, cli/, platform/controller/, gitops/, examples/, hack/, dashboards/)
- [x] T002 Initialize root go.work file for Go workspace mode
- [x] T003 [P] Initialize contracts/ Go module with go.mod (github.com/Infoblox-CTO/data.platform.kit/contracts)
- [x] T004 [P] Initialize sdk/ Go module with go.mod (github.com/Infoblox-CTO/data.platform.kit/sdk)
- [x] T005 [P] Initialize cli/ Go module with go.mod (github.com/Infoblox-CTO/data.platform.kit/cli)
- [x] T006 [P] Initialize platform/controller/ Go module with go.mod (github.com/Infoblox-CTO/data.platform.kit/platform/controller)
- [x] T007 [P] Create .golangci.yml linting configuration at repository root
- [x] T008 [P] Create Makefile with build/lint/test targets at repository root
- [x] T009 [P] Create .github/workflows/ci.yaml for PR validation (lint, test, build)
- [x] T010 [P] Add LICENSE file (per constitution Article IX)
- [x] T011 Copy JSON Schema contracts to contracts/schemas/ (dp-manifest.schema.json, pipeline-manifest.schema.json, bindings.schema.json, lineage-event.schema.json)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Core Domain Types (contracts/)

- [x] T012 Create manifest version constants in contracts/version.go
- [x] T013 [P] Create DataPackage struct in contracts/datapackage.go (maps to dp.yaml schema)
- [x] T014 [P] Create ArtifactContract struct in contracts/artifact.go (inputs/outputs with classification)
- [x] T015 [P] Create PipelineManifest struct in contracts/pipeline.go (maps to pipeline.yaml schema)
- [x] T016 [P] Create Binding types in contracts/binding.go (S3Prefix, KafkaTopic, PostgresTable)
- [x] T017 [P] Create Environment struct in contracts/environment.go
- [x] T018 [P] Create PackageVersion struct in contracts/version.go
- [x] T019 Create validation error types in contracts/errors.go (E001-E031 per data-model.md)

### YAML Parsing (sdk/)

- [x] T020 Create manifest parser interface in sdk/manifest/parser.go
- [x] T021 Create dp.yaml parser in sdk/manifest/datapackage.go (uses contracts types)
- [x] T022 Create pipeline.yaml parser in sdk/manifest/pipeline.go (uses contracts types)
- [x] T023 Create bindings.yaml parser in sdk/manifest/bindings.go (uses contracts types)

### CLI Foundation (cli/)

- [x] T024 Add cobra dependency to cli/go.mod
- [x] T025 Create root command with global flags in cli/cmd/root.go (-o/--output flag per research.md)
- [x] T026 Create output formatter interface in cli/internal/output/formatter.go (table, json, yaml)
- [x] T027 Create table formatter in cli/internal/output/table.go
- [x] T028 [P] Create json formatter in cli/internal/output/json.go
- [x] T029 [P] Create yaml formatter in cli/internal/output/yaml.go
- [x] T030 Create cli/main.go entry point

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Bootstrap Package (Priority: P1) 🎯 MVP

**Goal**: Data engineer can scaffold a new package with minimal friction

**Independent Test**: Run `cdpp init my-package --type pipeline` and verify dp.yaml + pipeline.yaml created

### Implementation for User Story 1

- [x] T031 [US1] Create package type enum in contracts/types.go (pipeline, model, dataset)
- [x] T032 [US1] Create dp.yaml template in cli/internal/templates/dp.yaml.tmpl
- [x] T033 [P] [US1] Create pipeline.yaml template in cli/internal/templates/pipeline.yaml.tmpl
- [x] T034 [P] [US1] Create model type template in cli/internal/templates/model.dp.yaml.tmpl
- [x] T035 [P] [US1] Create dataset type template in cli/internal/templates/dataset.dp.yaml.tmpl
- [x] T036 [US1] Create template renderer in cli/internal/templates/renderer.go
- [x] T037 [US1] Implement `cdpp init` command in cli/cmd/init.go (--type flag, template selection)
- [x] T038 [US1] Add init command to root in cli/cmd/root.go
- [x] T039 [US1] Create example package in examples/kafka-s3-pipeline/dp.yaml
- [x] T040 [P] [US1] Create example pipeline manifest in examples/kafka-s3-pipeline/pipeline.yaml

**Checkpoint**: `cdpp init` works - can scaffold new packages ✓

---

## Phase 4: User Story 2 - Local Development (Priority: P1) 🎯 MVP

**Goal**: Data engineer can run pipelines locally and validate manifests without Kubernetes

**Independent Test**: Run `cdpp dev up` to start stack, `cdpp run` to execute, `cdpp lint` to validate

### Local Dev Stack (hack/)

- [X] T041 [US2] Create docker-compose.yaml in hack/compose/docker-compose.yaml (Redpanda, LocalStack, PostgreSQL per research.md)
- [X] T042 [P] [US2] Create LocalStack init script in hack/compose/localstack-init.sh (S3 buckets)
- [X] T043 [P] [US2] Create Redpanda init script in hack/compose/redpanda-init.sh (topics)
- [X] T044 [P] [US2] Create sample bindings for local env in hack/compose/bindings.local.yaml
- [X] T045 [US2] Create docker-compose manager in sdk/localdev/compose.go

### Validation (sdk/)

- [X] T046 [US2] Create manifest validator interface in sdk/validate/validator.go
- [X] T047 [US2] Implement dp.yaml validator in sdk/validate/datapackage.go (rules E001-E003)
- [X] T048 [P] [US2] Implement pipeline.yaml validator in sdk/validate/pipeline.go (rules E030-E031)
- [X] T049 [P] [US2] Implement artifact contract validator in sdk/validate/artifact.go (rules E004-E005)
- [X] T050 [US2] Implement bindings validator in sdk/validate/bindings.go (rules E010-E011)
- [X] T051 [US2] Create aggregate validator in sdk/validate/aggregate.go

### CLI Commands (cli/)

- [X] T052 [US2] Implement `cdpp dev up` command in cli/cmd/dev.go (starts compose stack)
- [X] T053 [US2] Implement `cdpp dev down` command in cli/cmd/dev.go (stops compose stack)
- [X] T054 [US2] Implement `cdpp dev status` command in cli/cmd/dev.go (shows container status)
- [X] T055 [US2] Create local runner interface in sdk/runner/runner.go
- [X] T056 [US2] Implement Docker-based local runner in sdk/runner/docker.go
- [X] T057 [US2] Implement `cdpp run` command in cli/cmd/run.go (executes pipeline locally)
- [X] T058 [US2] Implement `cdpp lint` command in cli/cmd/lint.go (validates all manifests)
- [X] T059 [US2] Implement `cdpp test` command in cli/cmd/test.go (runs sample data through pipeline)
- [X] T060 [US2] Add dev, run, lint, test commands to root in cli/cmd/root.go

**Checkpoint**: Full local dev loop works - scaffold, validate, run locally ✓

---

## Phase 5: User Story 3 - Publish Package (Priority: P2)

**Goal**: Data engineer can build and publish immutable packages to OCI registry

**Independent Test**: Run `cdpp build && cdpp publish` and verify artifact in registry with correct digest

### OCI Registry (sdk/)

- [X] T061 [US3] Add oras-go v2 dependency to sdk/go.mod
- [X] T062 [US3] Create OCI client interface in sdk/registry/client.go
- [X] T063 [US3] Implement oras-based OCI client in sdk/registry/oras.go (push, pull, resolve)
- [X] T064 [US3] Create artifact bundler in sdk/registry/bundler.go (collects manifests + code into artifact)
- [X] T065 [US3] Implement tag immutability check in sdk/registry/oras.go (resolve before push, fail if exists per research.md)
- [X] T066 [P] [US3] Create digest reference generator in sdk/registry/digest.go

### CLI Commands (cli/)

- [X] T067 [US3] Implement `cdpp build` command in cli/cmd/build.go (validates + bundles artifact)
- [X] T068 [US3] Implement `cdpp publish` command in cli/cmd/publish.go (pushes to OCI registry)
- [X] T069 [US3] Add build, publish commands to root in cli/cmd/root.go
- [X] T070 [US3] Add registry authentication handling in cli/internal/auth/registry.go (docker config)

**Checkpoint**: Can build and publish immutable packages to OCI registry ✓

---

## Phase 6: User Story 4 - Promote Package (Priority: P2)

**Goal**: Data engineer can promote packages between environments via GitOps PR workflow

**Independent Test**: Run `cdpp promote kafka-s3-pipeline v1.0.0 --to int` and verify PR created

### GitOps Structure (gitops/)

- [X] T071 [US4] Create base Kustomization in gitops/base/kustomization.yaml
- [X] T072 [P] [US4] Create dev overlay in gitops/environments/dev/kustomization.yaml
- [X] T073 [P] [US4] Create int overlay in gitops/environments/int/kustomization.yaml
- [X] T074 [P] [US4] Create prod overlay in gitops/environments/prod/kustomization.yaml
- [X] T075 [US4] Create PackageDeployment CRD base in gitops/base/crds/packagedeployment.yaml
- [X] T076 [US4] Create ArgoCD ApplicationSet in gitops/argocd/applicationset.yaml (folder generator per research.md)

### Promotion Logic (sdk/)

- [X] T077 [US4] Create promotion service interface in sdk/promotion/service.go
- [X] T078 [US4] Implement Kustomize overlay updater in sdk/promotion/kustomize.go (updates version field)
- [X] T079 [US4] Create GitHub PR client interface in sdk/promotion/github.go
- [X] T080 [US4] Implement PR creation for promotion in sdk/promotion/pr.go (creates branch, commits, opens PR)
- [X] T081 [US4] Create PromotionRecord generator in sdk/promotion/record.go

### CLI Commands (cli/)

- [X] T082 [US4] Implement `cdpp promote` command in cli/cmd/promote.go (--to env, --dry-run)
- [X] T083 [US4] Add promote command to root in cli/cmd/root.go

### Controller (platform/controller/)

- [X] T084 [US4] Add controller-runtime dependency to platform/controller/go.mod
- [X] T085 [US4] Create PackageDeployment CRD types in platform/controller/api/v1alpha1/packagedeployment_types.go
- [X] T086 [US4] Generate CRD manifests with controller-gen in platform/controller/config/crd/
- [X] T087 [US4] Create PackageDeployment reconciler in platform/controller/internal/controller/packagedeployment_controller.go
- [X] T088 [US4] Implement OCI artifact pull in reconciler in platform/controller/internal/controller/packagedeployment_controller.go
- [X] T089 [US4] Implement Kubernetes Job creation from PipelineManifest in platform/controller/internal/controller/job.go
- [X] T090 [US4] Create controller main.go in platform/controller/cmd/main.go
- [X] T091 [US4] Create controller Dockerfile in platform/controller/Dockerfile

### GitHub Actions Workflow

- [X] T092 [P] [US4] Create promotion workflow in .github/workflows/promote.yaml (triggered on PR merge to gitops/)

**Checkpoint**: Full GitOps promotion flow works - promote creates PR, merge triggers ArgoCD sync

---

## Phase 7: User Story 5 - Observability (Priority: P3)

**Goal**: Data engineer can monitor pipeline health and troubleshoot failures

**Independent Test**: Run pipeline, then `cdpp status` shows run status, `cdpp logs` shows output

### Metrics (sdk/)

- [X] T093 [US5] Create metrics interface in sdk/metrics/metrics.go
- [X] T094 [US5] Implement Prometheus metrics in sdk/metrics/prometheus.go (run_total, run_duration, run_status)
- [X] T095 [P] [US5] Create structured logger using slog in sdk/logging/logger.go (per research.md)

### Run Tracking (sdk/)

- [X] T096 [US5] Create RunRecord service in sdk/runs/service.go
- [X] T097 [US5] Implement run history storage in sdk/runs/store.go (PostgreSQL)
- [X] T098 [US5] Create run status aggregator in sdk/runs/status.go

### CLI Commands (cli/)

- [X] T099 [US5] Implement `cdpp status` command in cli/cmd/status.go (shows package status across envs)
- [X] T100 [US5] Implement `cdpp logs` command in cli/cmd/logs.go (streams pod logs)
- [X] T101 [US5] Implement `cdpp rollback` command in cli/cmd/rollback.go (promotes previous version)
- [X] T102 [US5] Add status, logs, rollback commands to root in cli/cmd/root.go

### Controller Observability (platform/controller/)

- [X] T103 [US5] Add Prometheus metrics to controller in platform/controller/internal/metrics/metrics.go
- [X] T104 [US5] Add structured logging to controller in platform/controller/internal/controller/packagedeployment_controller.go
- [X] T105 [P] [US5] Create ServiceMonitor for controller in platform/controller/config/prometheus/servicemonitor.yaml

### Dashboards (dashboards/)

- [X] T106 [P] [US5] Create Grafana dashboard JSON for pipeline health in dashboards/pipeline-health.json
- [X] T107 [P] [US5] Create Grafana dashboard JSON for controller metrics in dashboards/controller.json
- [X] T108 [US5] Create ConfigMap for dashboard provisioning in dashboards/configmap.yaml

**Checkpoint**: Full observability - metrics, logs, status, rollback all work ✓

---

## Phase 8: User Story 6 - Data Governance (Priority: P3)

**Goal**: Data steward can track lineage and ensure PII classification compliance

**Independent Test**: Run pipeline with outputs, verify lineage events in Marquez, verify PII classification exists

### Lineage (sdk/)

- [X] T109 [US6] Create LineageEvent struct per OpenLineage spec in sdk/lineage/event.go (matches lineage-event.schema.json)
- [X] T110 [US6] Create lineage emitter interface in sdk/lineage/emitter.go
- [X] T111 [US6] Implement HTTP emitter for Marquez in sdk/lineage/marquez.go
- [X] T112 [US6] Integrate lineage emission in runner in sdk/runner/docker.go (emit START, COMPLETE, FAIL)

### PII Validation (sdk/)

- [X] T113 [US6] Create PII classification validator in sdk/validate/pii.go (ensures outputs have classification)
- [X] T114 [US6] Add PII validation to aggregate validator in sdk/validate/aggregate.go
- [X] T115 [US6] Update lint command to run PII validation in cli/cmd/lint.go

### Catalog Integration (sdk/)

- [X] T116 [US6] Create catalog record type in sdk/catalog/record.go
- [X] T117 [US6] Create catalog client interface in sdk/catalog/client.go
- [X] T118 [US6] Implement Marquez-based catalog client in sdk/catalog/marquez.go

### Local Lineage Stack (hack/)

- [X] T119 [P] [US6] Add Marquez service to docker-compose in hack/compose/docker-compose.yaml
- [X] T120 [P] [US6] Create Marquez init script in hack/compose/marquez-init.sh

### Example Package Update

- [X] T121 [US6] Add PII classification to kafka-s3-pipeline outputs in examples/kafka-s3-pipeline/dp.yaml
- [X] T122 [US6] Add lineage facets to kafka-s3-pipeline in examples/kafka-s3-pipeline/dp.yaml

**Checkpoint**: Full governance - lineage tracked, PII validated, catalog records created ✓

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, and final validation

- [X] T123 [P] Update README.md with project overview and getting started
- [X] T124 [P] Create CONTRIBUTING.md with development workflow
- [X] T125 [P] Create docs/architecture.md with system diagrams
- [X] T126 [P] Create docs/cli-reference.md with all commands documented
- [ ] T127 Run quickstart.md validation (30-minute end-to-end test)
- [X] T128 Create release scripts in hack/release/ for multi-module tagging
- [X] T129 Security review: validate no credentials in code
- [X] T130 Performance check: CLI command startup time < 100ms (actual: ~25ms)

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1: Setup
    │
    ▼
Phase 2: Foundational ──────────────────┐
    │                                   │
    ▼                                   ▼
Phase 3: US1 (Bootstrap)          [BLOCKS ALL]
    │                                   │
    ▼                                   │
Phase 4: US2 (Local Dev)          ◄─────┘
    │
    ├──────────────────┐
    ▼                  ▼
Phase 5: US3       Phase 6: US4
(Publish)          (Promote)
    │                  │
    └──────┬───────────┘
           ▼
    Phase 7: US5 (Observability)
           │
           ▼
    Phase 8: US6 (Governance)
           │
           ▼
    Phase 9: Polish
```

### User Story Dependencies

| Story | Depends On | Can Parallel With |
|-------|------------|-------------------|
| US1 (Bootstrap) | Foundational | None (start first) |
| US2 (Local Dev) | US1 | None |
| US3 (Publish) | US2 | US4 (different files) |
| US4 (Promote) | US2 | US3 (different files) |
| US5 (Observability) | US3, US4 | None |
| US6 (Governance) | US5 | None |

### Within Each User Story

1. SDK components before CLI commands
2. Contracts/types before implementations
3. Core logic before integrations

### Parallel Opportunities per Phase

**Phase 1 (Setup)**:
- T003, T004, T005, T006 (module inits)
- T007, T008, T009, T010 (configs)

**Phase 2 (Foundational)**:
- T013, T014, T015, T016, T017, T018 (contract types)
- T027, T028, T029 (output formatters)

**Phase 3 (US1)**:
- T033, T034, T035 (templates)
- T039, T040 (examples)

**Phase 4 (US2)**:
- T042, T043, T044 (init scripts)
- T048, T049 (validators)

**Phase 5 & 6 (US3, US4)**: Can run entire phases in parallel

---

## Parallel Example: Phase 2 Foundational

```bash
# Contract types (all parallel - different files):
T013: contracts/datapackage.go
T014: contracts/artifact.go
T015: contracts/pipeline.go
T016: contracts/binding.go
T017: contracts/environment.go
T018: contracts/version.go

# Output formatters (all parallel - different files):
T027: cli/internal/output/table.go
T028: cli/internal/output/json.go
T029: cli/internal/output/yaml.go
```

---

## Parallel Example: US3 + US4

```bash
# These two user stories can run in parallel after US2 completion:

# Developer A: US3 (Publish)
T061-T070: sdk/registry/*, cli/cmd/build.go, cli/cmd/publish.go

# Developer B: US4 (Promote)
T071-T092: gitops/*, sdk/promotion/*, cli/cmd/promote.go, platform/controller/*
```

---

## Implementation Strategy

### MVP First (US1 + US2)

1. Complete Phase 1: Setup (T001-T011)
2. Complete Phase 2: Foundational (T012-T030)
3. Complete Phase 3: US1 Bootstrap (T031-T040)
4. Complete Phase 4: US2 Local Dev (T041-T060)
5. **STOP and VALIDATE**: Can scaffold, validate, and run locally
6. ✅ **MVP Deliverable**: Local development workflow works end-to-end

### Incremental Delivery

| Milestone | User Stories | Weeks | Deliverable |
|-----------|--------------|-------|-------------|
| M0 | Setup + Foundation + US1 | 1-2 | Package scaffold works |
| M0.5 | US2 | 2 | Local dev loop complete |
| M1 | US3 + US4 | 3-4 | Publish + GitOps promotion |
| M2 | US5 | 5-6 | Observability + rollback |
| M3 | US6 | 7-8 | Lineage + PII + catalog |

### Parallel Team Strategy

With 2 developers after Foundational phase:

1. **Week 1-2**: Both on Setup + Foundational
2. **Week 2**: Both on US1 → US2
3. **Week 3-4**: 
   - Dev A: US3 (Publish)
   - Dev B: US4 (Promote)
4. **Week 5-6**: Both on US5 (Observability)
5. **Week 7-8**: Both on US6 (Governance)

---

## Summary

| Phase | Task Range | Count | Parallelizable |
|-------|------------|-------|----------------|
| Setup | T001-T011 | 11 | 8 |
| Foundational | T012-T030 | 19 | 10 |
| US1 Bootstrap | T031-T040 | 10 | 6 |
| US2 Local Dev | T041-T060 | 20 | 6 |
| US3 Publish | T061-T070 | 10 | 1 |
| US4 Promote | T071-T092 | 22 | 5 |
| US5 Observability | T093-T108 | 16 | 4 |
| US6 Governance | T109-T122 | 14 | 2 |
| Polish | T123-T130 | 8 | 4 |
| **Total** | | **130** | **46** |

---

## Notes

- All Go modules use latest stable Go version per constitution v1.1.0
- [P] tasks indicate parallelization opportunity (different files, no deps)
- [US#] labels map tasks to user stories for traceability
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Each user story should be demo-able after completion
