# Feature Specification: k3d Local Development Environment

**Feature Branch**: `005-k3d-local-dev`  
**Created**: January 25, 2026  
**Status**: Draft  
**Input**: User description: "k3d-based local development environment support for dp dev commands"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start k3d Development Environment (Priority: P1)

As a developer, I want to start a local Kubernetes development environment using k3d so that I can develop and test data packages in an environment that mirrors production.

**Why this priority**: This is the core functionality - without being able to start the k3d cluster with required services, no other features can be used. It delivers immediate value by providing a Kubernetes-native development experience.

**Independent Test**: Can be fully tested by running `dp dev up --runtime=k3d` and verifying that a k3d cluster is created with Redpanda, LocalStack, and PostgreSQL services accessible at expected ports.

**Acceptance Scenarios**:

1. **Given** k3d is installed on the system and no dp cluster exists, **When** user runs `dp dev up --runtime=k3d`, **Then** a k3d cluster named "dp-local" is created with all required services deployed and healthy
2. **Given** a dp k3d cluster already exists and is stopped, **When** user runs `dp dev up --runtime=k3d`, **Then** the existing cluster is started without recreating it
3. **Given** k3d is not installed on the system, **When** user runs `dp dev up --runtime=k3d`, **Then** a clear error message indicates k3d is required with installation instructions

---

### User Story 2 - Access Development Services (Priority: P1)

As a developer, I want the development services (Redpanda, LocalStack, PostgreSQL) to be accessible at predictable localhost ports so that my data package code can connect without configuration changes.

**Why this priority**: Without accessible services, the k3d cluster is not useful for development. Port forwarding is essential for the development workflow.

**Independent Test**: After `dp dev up --runtime=k3d` completes, verify that connections to localhost:19092 (Redpanda), localhost:4566 (LocalStack), and localhost:5432 (PostgreSQL) succeed.

**Acceptance Scenarios**:

1. **Given** k3d cluster is running with services deployed, **When** cluster startup completes, **Then** port forwards are automatically established to all services at standard ports
2. **Given** port forwards are active, **When** user's data package connects to localhost:19092, **Then** connection reaches Redpanda broker in the cluster
3. **Given** port forwards are active, **When** user's data package connects to localhost:4566, **Then** connection reaches LocalStack S3 endpoint

---

### User Story 3 - Stop k3d Development Environment (Priority: P2)

As a developer, I want to stop the k3d development environment to free system resources when I'm not actively developing.

**Why this priority**: Resource management is important but secondary to being able to start and use the environment.

**Independent Test**: Run `dp dev down --runtime=k3d` and verify cluster is stopped and port forwards are terminated.

**Acceptance Scenarios**:

1. **Given** a running dp k3d cluster, **When** user runs `dp dev down --runtime=k3d`, **Then** the cluster is stopped and all port forwards are terminated
2. **Given** a running dp k3d cluster with data volumes, **When** user runs `dp dev down --runtime=k3d --volumes`, **Then** the cluster is deleted including all persistent data
3. **Given** no dp k3d cluster exists, **When** user runs `dp dev down --runtime=k3d`, **Then** command completes successfully with informational message

---

### User Story 4 - Check Development Environment Status (Priority: P2)

As a developer, I want to check the status of my k3d development environment to see which services are running and healthy.

**Why this priority**: Status visibility helps troubleshoot issues but is not required for basic functionality.

**Independent Test**: Run `dp dev status --runtime=k3d` and verify accurate cluster and pod status is displayed.

**Acceptance Scenarios**:

1. **Given** a running dp k3d cluster, **When** user runs `dp dev status --runtime=k3d`, **Then** status shows cluster name, pod states, and service health
2. **Given** no dp k3d cluster exists, **When** user runs `dp dev status --runtime=k3d`, **Then** status indicates no cluster is running

---

### User Story 5 - Run from Any Directory (Priority: P2)

As a developer, I want to run `dp dev up` from my data package directory without needing to navigate to the DP workspace so that my workflow is more convenient.

**Why this priority**: Improves developer experience but workaround exists (navigate to workspace directory).

**Independent Test**: From `/tmp/my-pipeline` directory, run `dp dev up --runtime=k3d` and verify cluster starts successfully.

**Acceptance Scenarios**:

1. **Given** user is in a data package directory outside DP workspace, **When** user runs `dp dev up --runtime=k3d`, **Then** the cluster starts successfully using embedded manifest configuration
2. **Given** user has set `DP_WORKSPACE_PATH` environment variable, **When** user runs `dp dev up --runtime=compose`, **Then** docker-compose file is found at the workspace path

---

### User Story 6 - Backward Compatibility with Docker Compose (Priority: P3)

As a developer who prefers Docker Compose, I want the existing `dp dev up` workflow to continue working unchanged so that I can choose my preferred runtime.

**Why this priority**: Existing users should not experience breaking changes, but k3d is the new recommended approach.

**Independent Test**: Run `dp dev up` (without --runtime flag) from DP workspace and verify docker-compose workflow works as before.

**Acceptance Scenarios**:

1. **Given** user is in DP workspace with docker-compose.yaml, **When** user runs `dp dev up` without --runtime flag, **Then** docker-compose stack starts as before
2. **Given** user has configured `dev.runtime: k3d` in config file, **When** user runs `dp dev up`, **Then** k3d cluster starts as the configured default

---

### Edge Cases

- What happens when k3d cluster creation times out? System displays timeout error with troubleshooting steps.
- How does system handle port conflicts on localhost? Check for port availability before starting, display clear error if ports in use.
- What happens if kubectl is not installed? Display error message with kubectl installation instructions.
- How does system handle interrupted cluster startup? Provide `dp dev down --force` to clean up partial state.
- What happens if user runs `dp dev up --runtime=k3d` while compose stack is running? Warn user about potential port conflicts.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `--runtime` flag on `dp dev` commands accepting values `compose` or `k3d`
- **FR-002**: System MUST default to `compose` runtime when no `--runtime` flag is specified (backward compatibility)
- **FR-003**: System MUST create a k3d cluster named "dp-local" when `--runtime=k3d` is used with `dp dev up`
- **FR-004**: System MUST deploy Redpanda, LocalStack, and PostgreSQL services into the k3d cluster
- **FR-005**: System MUST establish port forwards: Redpanda (19092), LocalStack (4566), PostgreSQL (5432)
- **FR-006**: System MUST wait for all services to become healthy before returning from `dp dev up`
- **FR-007**: System MUST stop the k3d cluster and terminate port forwards on `dp dev down --runtime=k3d`
- **FR-008**: System MUST delete cluster data volumes when `--volumes` flag is used with `dp dev down`
- **FR-009**: System MUST display cluster status including pod states on `dp dev status --runtime=k3d`
- **FR-010**: System MUST verify k3d and kubectl are installed before attempting cluster operations
- **FR-011**: System MUST embed Kubernetes manifests in the CLI binary (no external file dependencies)
- **FR-012**: System MUST support `DP_WORKSPACE_PATH` environment variable for locating compose files
- **FR-013**: System MUST support runtime configuration via `~/.config/dp/config.yaml` file
- **FR-014**: System MUST check for port availability before starting services and report conflicts

### Key Entities

- **Cluster**: Represents the k3d Kubernetes cluster; has name, state (running/stopped), and creation timestamp
- **Service**: Represents a deployed workload (Redpanda, LocalStack, PostgreSQL); has name, health status, and port mappings
- **PortForward**: Represents an active port forward; has local port, target service, and process ID
- **Configuration**: User preferences stored in config file; includes default runtime, workspace path, cluster settings

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can start a fully functional k3d development environment in under 3 minutes on first run
- **SC-002**: Users can start an existing stopped cluster in under 30 seconds
- **SC-003**: All three services (Redpanda, LocalStack, PostgreSQL) are accessible and healthy after startup
- **SC-004**: Port forwarding remains stable for the duration of a development session (8+ hours)
- **SC-005**: Existing Docker Compose workflow continues to function identically (zero breaking changes)
- **SC-006**: Users can successfully run `dp dev up --runtime=k3d` from any directory on the filesystem
- **SC-007**: Clear error messages with actionable guidance when prerequisites are missing

## Assumptions

- Users have Docker installed and running (required by k3d)
- Users have sufficient system resources (4GB RAM minimum recommended for local cluster)
- Standard localhost ports (19092, 4566, 5432) are available or can be configured
- Users on macOS, Linux, and Windows WSL2 are supported (k3d platform compatibility)

## Out of Scope

- Multi-cluster support
- Remote k3d clusters
- Custom service configurations per data package
- Integration with kind or minikube
- Cluster auto-scaling
- Persistent storage across cluster deletions (without --volumes flag, data persists within cluster lifetime only)
