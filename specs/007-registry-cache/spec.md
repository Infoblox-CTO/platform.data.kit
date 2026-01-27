# Feature Specification: Registry Pull-Through Cache for k3d Local Development

**Feature Branch**: `007-registry-cache`  
**Created**: 2026-01-28  
**Status**: Draft  
**Input**: User description: "Add Docker registry pull-through cache for k3d local development to avoid re-downloading images when clusters are deleted/recreated"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Developer Starts Local Environment with Image Cache (Priority: P1)

As a developer, I want `dp dev up` to automatically start a Docker registry pull-through cache before creating the k3d cluster, so that container images are cached locally and don't need to be re-downloaded when I delete and recreate my cluster.

**Why this priority**: This is the core value proposition. Developers frequently delete/recreate k3d clusters during development. Without caching, each cluster recreation downloads all images from Docker Hub again, wasting time and bandwidth. This directly impacts developer productivity.

**Independent Test**: Can be fully tested by running `dp dev up`, pulling an image inside k3d, then running `dp dev down && dp dev up` and verifying the image is available without re-downloading from Docker Hub.

**Acceptance Scenarios**:

1. **Given** a fresh workstation with Docker installed, **When** I run `dp dev up`, **Then** a registry cache container starts before k3d is created, and the k3d cluster is configured to use it as a mirror for docker.io.

2. **Given** the dev environment is running, **When** I pull `nginx:latest` inside the k3d cluster, **Then** the image is fetched via the cache and stored locally in the cache volume.

3. **Given** I have pulled images and then run `dp dev down && dp dev up`, **When** I pull the same image again, **Then** the image is served from the local cache without contacting Docker Hub.

4. **Given** the registry cache container is already running with matching configuration, **When** I run `dp dev up`, **Then** the existing container is reused (not recreated), preserving the cached images.

---

### User Story 2 - CI/CD Environments Skip Registry Cache (Priority: P1)

As a CI/CD system, I need the `dev up` and `dev down` commands to skip the registry cache functionality entirely so that CI builds don't attempt to modify Docker infrastructure or create persistent caches.

**Why this priority**: CI/CD environments have their own caching strategies and often run with restricted permissions. The registry cache must not interfere with CI builds or cause failures.

**Independent Test**: Can be tested by setting `CI=true` environment variable and running `dp dev up`, verifying no registry container is created.

**Acceptance Scenarios**:

1. **Given** the environment variable `CI=true` is set, **When** I run `dp dev up`, **Then** the registry cache is not started, no error occurs, and the command succeeds.

2. **Given** the environment variable `GITHUB_ACTIONS=true` is set, **When** I run `dp dev up`, **Then** the registry cache is skipped.

3. **Given** the environment variable `JENKINS_URL` is non-empty, **When** I run `dp dev up`, **Then** the registry cache is skipped.

4. **Given** CI environment is detected, **When** k3d cluster is created, **Then** it does not include `--registry-config` flag (uses default Docker Hub access).

---

### User Story 3 - Developer Stops Local Environment (Priority: P2)

As a developer, I want `dp dev down` to stop the registry cache container so that resources are freed when I'm not actively developing.

**Why this priority**: Clean shutdown is important for resource management, but less critical than the core caching functionality.

**Independent Test**: Run `dp dev down` and verify the registry container is stopped.

**Acceptance Scenarios**:

1. **Given** the dev environment is running with the registry cache, **When** I run `dp dev down`, **Then** the registry cache container is stopped.

2. **Given** the dev environment is running, **When** I run `dp dev down`, **Then** the cache volume (`dev_registry_cache`) is preserved (not deleted) by default.

3. **Given** I run `dp dev down --volumes`, **When** the command completes, **Then** the cache volume is also removed.

---

### User Story 4 - Idempotent Operations (Priority: P2)

As a developer, I want `dp dev up` to be safe to run multiple times so that I don't have to worry about the current state before running the command.

**Why this priority**: Idempotency is critical for developer experience and scripting.

**Independent Test**: Run `dp dev up` twice in a row and verify no errors and the cache remains functional.

**Acceptance Scenarios**:

1. **Given** the registry cache is already running with matching configuration, **When** I run `dp dev up`, **Then** the command succeeds and reuses the existing container.

2. **Given** the registry cache container exists but is stopped, **When** I run `dp dev up`, **Then** the container is started (not recreated).

3. **Given** the registry cache configuration has changed, **When** I run `dp dev up`, **Then** the old container is removed and a new one is created with the updated configuration.

4. **Given** no registry cache exists, **When** I run `dp dev up`, **Then** a new registry cache container is created.

---

### User Story 5 - Cross-Platform Compatibility (Priority: P2)

As a developer on macOS or Linux using Docker or Rancher Desktop, I want the registry cache to work correctly so that my development environment functions regardless of my host OS or Docker runtime.

**Why this priority**: The team uses diverse development setups. The cache must work across all supported environments.

**Independent Test**: Test on macOS with Docker Desktop and on Linux with Docker, verifying k3d can reach the cache in both environments.

**Acceptance Scenarios**:

1. **Given** I'm on macOS with Docker Desktop, **When** I run `dp dev up`, **Then** k3d can reach the registry cache via `host.k3d.internal:5000`.

2. **Given** I'm on Linux with Docker, **When** I run `dp dev up`, **Then** k3d can reach the registry cache via the appropriate host endpoint.

3. **Given** `host.k3d.internal` is not resolvable, **When** the cache is configured, **Then** the system falls back to `host.docker.internal`.

4. **Given** I set the `DEV_REGISTRY_MIRROR_HOST` environment variable, **When** I run `dp dev up`, **Then** the specified host is used for the registry endpoint.

---

### Edge Cases

- What happens when Docker is not running? → `dev up` fails early with a clear error message before attempting to create the cache.
- What happens when port 5000 is already in use by another process? → `dev up` detects the conflict and provides a clear error message.
- What happens when the cache volume is corrupted? → Running `dp dev down --volumes && dp dev up` resets the cache.
- What happens when Docker Hub rate limits apply? → The cache helps avoid rate limits by serving cached images; if the cache is cold, rate limits still apply to initial pulls.
- What happens when the user has no network connectivity? → Cached images continue to work; uncached images fail with a network error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST start a Docker registry container in pull-through cache mode before creating the k3d cluster.
- **FR-002**: System MUST configure the registry to proxy requests to `https://registry-1.docker.io`.
- **FR-003**: System MUST persist the cache using a Docker volume named `dev_registry_cache` mounted at `/var/lib/registry`.
- **FR-004**: System MUST expose the registry on host port 5000.
- **FR-005**: System MUST generate a registry configuration file at `.cache/registry-config.yml`.
- **FR-006**: System MUST compute a SHA256 hash of the configuration and store it as a container label (`dev.cache.config_sha256`).
- **FR-007**: System MUST reuse the existing container if the configuration hash matches; otherwise, recreate it.
- **FR-008**: System MUST add identifying labels to the container:
  - `dev.capability=cache-registry`
  - `dev.cache.backend=filesystem`
  - `dev.cache.mode=pull-through`
  - `dev.cache.mirror=docker.io`
  - `dev.cache.endpoint=<computed endpoint>`
- **FR-009**: System MUST create a Docker network named `devcache` if it doesn't exist and attach the registry container to it.
- **FR-010**: System MUST generate `.cache/registries.yaml` for k3d with the mirror endpoint for `docker.io`.
- **FR-011**: System MUST pass `--registry-config .cache/registries.yaml` when creating the k3d cluster in local-dev mode.
- **FR-012**: System MUST skip all registry cache operations and return success when running in CI/CD (detected via `CI=true`, `GITHUB_ACTIONS=true`, or non-empty `JENKINS_URL`).
- **FR-013**: System MUST NOT pass `--registry-config` to k3d when running in CI/CD.
- **FR-014**: System MUST stop the registry cache container when `dp dev down` is executed.
- **FR-015**: System MUST preserve the cache volume by default; remove it only when `--volumes` flag is passed.
- **FR-016**: System MUST determine the k3d-accessible endpoint by testing connectivity:
  1. Prefer `host.k3d.internal`
  2. Fall back to `host.docker.internal`
  3. Allow override via `DEV_REGISTRY_MIRROR_HOST` environment variable

### Key Entities

- **Registry Container**: The Docker Distribution (registry:2) container running in pull-through cache mode. Identified by container name `dev-registry-cache`.
- **Cache Volume**: Docker volume `dev_registry_cache` that persists cached image layers across container restarts.
- **Registry Config**: YAML configuration file at `.cache/registry-config.yml` that configures the registry's proxy behavior.
- **Registries YAML**: k3d configuration file at `.cache/registries.yaml` that tells k3s to use the local mirror for docker.io.
- **DevCache Network**: Docker network `devcache` that provides connectivity between the registry and other dev containers.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After initial image pull, subsequent pulls of the same image complete in under 5 seconds (compared to 30+ seconds from Docker Hub).
- **SC-002**: Developers can delete and recreate k3d clusters without re-downloading cached images.
- **SC-003**: `dp dev up` completes successfully on both macOS and Linux with Docker.
- **SC-004**: `dp dev up` and `dp dev down` can be run multiple times without errors (idempotent).
- **SC-005**: CI/CD pipelines (Jenkins, GitHub Actions) continue to work without modification or slowdown.
- **SC-006**: Cache persists across `dp dev down` and `dp dev up` cycles (unless `--volumes` is used).

## Assumptions

- Docker is installed and running on the developer's machine.
- Port 5000 is available for the registry (or the user will configure an alternative).
- The `registry:2` image is available from Docker Hub (or cached locally).
- Developers have sufficient disk space for the image cache.
- k3d version supports `--registry-config` flag (k3d v5+).
