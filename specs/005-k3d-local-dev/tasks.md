# Tasks: k3d Local Development Environment

**Input**: Design documents from `/specs/005-k3d-local-dev/`  
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Project initialization and manifest structure

- [X] T001 Create sdk/localdev/manifests/ directory structure
- [X] T002 [P] Create Redpanda Kubernetes manifest in sdk/localdev/manifests/redpanda.yaml
- [X] T003 [P] Create LocalStack Kubernetes manifest in sdk/localdev/manifests/localstack.yaml
- [X] T004 [P] Create PostgreSQL Kubernetes manifest in sdk/localdev/manifests/postgres.yaml
- [X] T005 Create manifest embedding with go:embed in sdk/localdev/manifests/embed.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 Define RuntimeManager interface in sdk/localdev/runtime.go
- [X] T007 Define RuntimeType enum and StackStatus types in sdk/localdev/runtime.go
- [X] T008 [P] Create unit tests for RuntimeManager type selection in sdk/localdev/runtime_test.go
- [X] T009 Update ComposeManager to implement RuntimeManager interface in sdk/localdev/compose.go
- [X] T010 [P] Create prerequisite checker (k3d, kubectl, docker) in sdk/localdev/prerequisites.go
- [X] T011 [P] Create unit tests for prerequisite checker in sdk/localdev/prerequisites_test.go
- [X] T012 [P] Create port availability checker in sdk/localdev/ports.go
- [X] T013 [P] Create unit tests for port checker in sdk/localdev/ports_test.go
- [X] T014 Add --runtime flag to dp dev commands in cli/cmd/dev.go
- [X] T015 Create runtime selection logic based on flag and config in cli/cmd/dev.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 & 2 - Start k3d Environment & Access Services (Priority: P1) 🎯 MVP

**Goal**: Create k3d cluster with Redpanda, LocalStack, PostgreSQL and establish port forwards

**Independent Test**: Run `dp dev up --runtime=k3d` and verify services accessible at localhost ports

### Implementation for User Stories 1 & 2

- [X] T016 Create K3dManager struct and constructor in sdk/localdev/k3d.go
- [X] T017 [P] Create unit tests for K3dManager constructor in sdk/localdev/k3d_test.go
- [X] T018 Implement K3dManager.clusterExists() to check if cluster exists in sdk/localdev/k3d.go
- [X] T019 Implement K3dManager.createCluster() to create k3d cluster in sdk/localdev/k3d.go
- [X] T020 Implement K3dManager.startCluster() to start existing cluster in sdk/localdev/k3d.go
- [X] T021 Implement K3dManager.deployManifests() to apply embedded K8s manifests in sdk/localdev/k3d.go
- [X] T022 [P] Create unit tests for cluster operations (mock exec) in sdk/localdev/k3d_test.go
- [X] T023 Create PortForwardManager struct in sdk/localdev/portforward.go
- [X] T024 Implement PortForwardManager.Start() to launch kubectl port-forward processes in sdk/localdev/portforward.go
- [X] T025 Implement PortForwardManager.Stop() to terminate port forward processes in sdk/localdev/portforward.go
- [X] T026 Implement PortForwardManager.Status() to check active port forwards in sdk/localdev/portforward.go
- [X] T027 [P] Create unit tests for PortForwardManager in sdk/localdev/portforward_test.go
- [X] T028 Implement K3dManager.WaitForHealthy() using kubectl wait in sdk/localdev/k3d.go
- [X] T029 [P] Create unit tests for WaitForHealthy in sdk/localdev/k3d_test.go
- [X] T030 Implement K3dManager.Up() orchestrating create/start, deploy, health, port-forward in sdk/localdev/k3d.go
- [X] T031 Wire K3dManager into runDevUp() in cli/cmd/dev.go
- [X] T032 [P] Create integration test for dp dev up --runtime=k3d in cli/cmd/dev_test.go

**Checkpoint**: User Stories 1 & 2 complete - k3d cluster starts with accessible services

---

## Phase 4: User Story 3 - Stop k3d Environment (Priority: P2)

**Goal**: Stop k3d cluster and terminate port forwards, optionally delete volumes

**Independent Test**: Run `dp dev down --runtime=k3d` and verify cluster stopped

### Implementation for User Story 3

- [X] T033 [US3] Implement K3dManager.stopCluster() to stop k3d cluster in sdk/localdev/k3d.go
- [X] T034 [US3] Implement K3dManager.deleteCluster() to delete cluster with volumes in sdk/localdev/k3d.go
- [X] T035 [P] [US3] Create unit tests for stop/delete cluster in sdk/localdev/k3d_test.go
- [X] T036 [US3] Implement K3dManager.Down() orchestrating stop/delete and port-forward cleanup in sdk/localdev/k3d.go
- [X] T037 [US3] Wire K3dManager.Down() into runDevDown() in cli/cmd/dev.go
- [X] T038 [P] [US3] Create integration test for dp dev down --runtime=k3d in cli/cmd/dev_test.go

**Checkpoint**: User Story 3 complete - can stop and clean up k3d environment

---

## Phase 5: User Story 4 - Check Environment Status (Priority: P2)

**Goal**: Display k3d cluster and service status

**Independent Test**: Run `dp dev status --runtime=k3d` and verify accurate status display

### Implementation for User Story 4

- [X] T039 [US4] Implement K3dManager.getClusterInfo() to get cluster state in sdk/localdev/k3d.go
- [X] T040 [US4] Implement K3dManager.getPodStatuses() to get pod states in sdk/localdev/k3d.go
- [X] T041 [P] [US4] Create unit tests for cluster/pod status in sdk/localdev/k3d_test.go
- [X] T042 [US4] Implement K3dManager.Status() returning StackStatus in sdk/localdev/k3d.go
- [X] T043 [US4] Wire K3dManager.Status() into runDevStatus() in cli/cmd/dev.go
- [X] T044 [US4] Format and display status output with service health in cli/cmd/dev.go
- [X] T045 [P] [US4] Create integration test for dp dev status --runtime=k3d in cli/cmd/dev_test.go

**Checkpoint**: User Story 4 complete - can check environment status

---

## Phase 6: User Story 5 - Run from Any Directory (Priority: P2)

**Goal**: Enable dp dev up --runtime=k3d from any directory using embedded manifests

**Independent Test**: Run `dp dev up --runtime=k3d` from /tmp and verify it works

### Implementation for User Story 5

- [X] T046 [US5] Implement DP_WORKSPACE_PATH environment variable support in cli/cmd/dev.go
- [X] T047 [US5] Update findComposeFile() to use DP_WORKSPACE_PATH in cli/cmd/dev.go
- [X] T048 [P] [US5] Create unit tests for workspace path resolution in cli/cmd/dev_test.go
- [X] T049 [US5] Verify k3d runtime works without workspace path (uses embedded manifests) in cli/cmd/dev.go

**Checkpoint**: User Story 5 complete - location-independent execution works

---

## Phase 7: User Story 6 - Backward Compatibility (Priority: P3)

**Goal**: Ensure compose runtime remains default and works unchanged

**Independent Test**: Run `dp dev up` without --runtime flag and verify compose workflow

### Implementation for User Story 6

- [X] T050 [US6] Create config file loader for ~/.config/dp/config.yaml in sdk/localdev/config.go
- [X] T051 [P] [US6] Create unit tests for config loader in sdk/localdev/config_test.go
- [X] T052 [US6] Implement default runtime selection from config in cli/cmd/dev.go
- [X] T053 [US6] Ensure compose is default when no config or flag in cli/cmd/dev.go
- [X] T054 [P] [US6] Create integration test for backward compatibility in cli/cmd/dev_test.go

**Checkpoint**: User Story 6 complete - existing workflow unchanged

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final validation

- [X] T055 [P] Update docs/getting-started/quickstart.md with k3d instructions
- [X] T056 [P] Update docs/reference/cli.md with --runtime flag documentation
- [X] T057 [P] Copy quickstart.md from specs/005-k3d-local-dev/ to docs/tutorials/k3d-local-dev.md
- [X] T058 Run quickstart.md validation end-to-end
- [X] T059 Update README.md with k3d prerequisites section

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories 1&2 (Phase 3)**: Depends on Foundational - MVP delivery
- **User Story 3 (Phase 4)**: Depends on Phase 3 (needs K3dManager)
- **User Story 4 (Phase 5)**: Depends on Phase 3 (needs K3dManager)
- **User Story 5 (Phase 6)**: Depends on Phase 3 (needs working k3d)
- **User Story 6 (Phase 7)**: Can start after Foundational
- **Polish (Phase 8)**: Depends on all user stories complete

### User Story Dependencies

- **User Stories 1 & 2 (P1)**: Combined as MVP - create cluster and access services
- **User Story 3 (P2)**: Depends on K3dManager from US1&2
- **User Story 4 (P2)**: Depends on K3dManager from US1&2
- **User Story 5 (P2)**: Can run after US1&2
- **User Story 6 (P3)**: Independent - only needs Foundational

### Within Each Phase

- Tasks marked [P] can run in parallel
- Sequential tasks must complete in order
- Unit tests should be written alongside or before implementation

### Parallel Opportunities

```text
# Phase 1 - All manifests in parallel:
T002, T003, T004 (different files)

# Phase 2 - Infrastructure in parallel:
T008, T010, T011, T012, T013 (different files)

# Phase 3 - Tests can parallel with implementation:
T017, T022, T027, T029, T032 (test files)
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2)

1. Complete Phase 1: Setup (manifest files)
2. Complete Phase 2: Foundational (interface, prerequisites)
3. Complete Phase 3: User Stories 1 & 2 (core k3d functionality)
4. **STOP and VALIDATE**: Test `dp dev up --runtime=k3d`
5. Demo/deploy if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add User Stories 1 & 2 → MVP: `dp dev up --runtime=k3d` works
3. Add User Story 3 → Can stop cluster
4. Add User Story 4 → Can check status
5. Add User Story 5 → Works from any directory
6. Add User Story 6 → Backward compatible
7. Polish → Documentation complete

---

## Summary

| Metric | Count |
|--------|-------|
| Total Tasks | 59 |
| Phase 1 (Setup) | 5 |
| Phase 2 (Foundational) | 10 |
| Phase 3 (US1 & US2 - MVP) | 17 |
| Phase 4 (US3 - Stop) | 6 |
| Phase 5 (US4 - Status) | 7 |
| Phase 6 (US5 - Any Directory) | 4 |
| Phase 7 (US6 - Backward Compat) | 5 |
| Phase 8 (Polish) | 5 |
| Parallel Tasks [P] | 24 |
| MVP Tasks (Phases 1-3) | 32 |
