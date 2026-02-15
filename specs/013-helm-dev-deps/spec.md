# Feature Specification: Helm-Based Dev Dependencies

**Feature Branch**: `013-helm-dev-deps`
**Created**: 2026-02-15
**Status**: Draft
**Input**: User description: "Ensure dependencies for dev environments are modeled as Helm charts and we have a uniform way to load these dev dependencies instead of bespoke code. Dev environment dependencies should be Helm charts with upstream subcharts where available, embedded into the CLI binary, with future support for chart version overrides via config."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Uniform Dev Stack Startup via Helm Charts (Priority: P1)

A developer runs `dp dev up` and all dev environment dependencies (message broker, object storage, database, lineage service) are provisioned using a single, uniform Helm-based mechanism. Each dependency is defined as a Helm chart maintained in the repository, and the CLI deploys them identically -- no bespoke code paths per service. The developer does not need to install or manage charts separately; they are embedded in the CLI binary.

**Why this priority**: This is the core value -- replacing ad-hoc, per-service deployment code with a consistent Helm chart pattern. It eliminates maintenance burden, reduces bugs from inconsistent provisioning logic, and makes adding new dependencies trivial.

**Independent Test**: Can be fully tested by running `dp dev up` on a fresh k3d cluster and verifying all services start and become healthy using the new Helm-based deployment. Delivers a working local dev environment identical to today's behavior but with a uniform architecture.

**Acceptance Scenarios**:

1. **Given** a developer has a k3d cluster available, **When** they run `dp dev up`, **Then** all dependencies (Redpanda with Console, LocalStack, PostgreSQL, Marquez) are deployed via Helm charts and become healthy within 2 minutes.
2. **Given** the CLI binary is built, **When** no external chart repositories or files are present on disk, **Then** the CLI still deploys all charts successfully because they are embedded in the binary.
3. **Given** a running dev stack, **When** the developer runs `dp dev status`, **Then** the status of each Helm-deployed service is shown with health information.
4. **Given** a running dev stack, **When** the developer runs `dp dev down`, **Then** all Helm releases are uninstalled and resources are cleaned up.

---

### User Story 2 - Upstream Charts as Subcharts (Priority: P2)

Where a well-maintained upstream Helm chart exists for a dependency (e.g., Redpanda, PostgreSQL), the repo's chart wraps it as a subchart. This gives the team upstream updates and best practices while allowing local value overrides. Where no suitable upstream chart exists (e.g., LocalStack, Marquez), the repo maintains a simple custom chart.

**Why this priority**: Using upstream charts reduces maintenance and provides production-grade defaults. However, this is secondary to the uniform mechanism itself -- custom charts that work uniformly are better than no charts at all.

**Independent Test**: Can be tested by inspecting each chart's `Chart.yaml` for subchart dependencies, running `helm dependency update`, and verifying that upstream chart versions are pulled and used during `dp dev up`.

**Acceptance Scenarios**:

1. **Given** a dependency has a well-maintained upstream Helm chart (e.g., Redpanda, PostgreSQL), **When** the repo's chart for that dependency is examined, **Then** it declares the upstream chart as a subchart dependency in `Chart.yaml`.
2. **Given** the upstream subchart is declared, **When** `helm dependency build` is run on the chart, **Then** the upstream chart archive is downloaded into the `charts/` subdirectory.
3. **Given** the upstream subchart is used, **When** the chart is deployed, **Then** local `values.yaml` overrides are applied on top of upstream defaults for dev-appropriate configuration (minimal resources, persistence disabled, dev ports).

---

### User Story 3 - Init Jobs for Data Seeding (Priority: P2)

After the core services start, initialization tasks run automatically -- creating S3 buckets, Kafka topics, database schemas, and lineage namespaces. These init tasks are modeled as Helm chart hooks or init containers within each dependency's chart, ensuring they run as part of the standard Helm deploy lifecycle rather than as separate bespoke scripts.

**Why this priority**: Without init tasks, the dev environment starts but is not usable. However, this builds on top of the chart mechanism (P1) and can be phased after the base charts work.

**Independent Test**: Can be tested by running `dp dev up` and verifying that expected resources exist (S3 buckets, Kafka topics, DB tables) without any manual setup steps.

**Acceptance Scenarios**:

1. **Given** the dev stack is starting up, **When** the Redpanda chart is deployed, **Then** all required Kafka topics (e.g., `dp.raw.events`, `dp.processed.events`, `dp.errors.dlq`) are created automatically via a Helm post-install hook.
2. **Given** the dev stack is starting up, **When** the LocalStack chart is deployed, **Then** all required S3 buckets (e.g., `cdpp-raw`, `cdpp-staging`, `cdpp-curated`) are created automatically.
3. **Given** the dev stack is starting up, **When** the PostgreSQL chart is deployed, **Then** the database schema, tables, and indexes are created automatically.

---

### User Story 4 - Chart Version Overrides via Config (Priority: P3)

A developer can override the version of any dependency chart through `dp config set`, allowing testing with newer or older versions of dependencies without modifying the repository. This leverages the existing hierarchical config system (repo, user, system levels).

**Why this priority**: This is a power-user feature for advanced testing scenarios. The default embedded versions should cover the vast majority of use cases.

**Independent Test**: Can be tested by running `dp config set dev.charts.redpanda.version X.Y.Z`, then `dp dev up`, and verifying the specified version is deployed.

**Acceptance Scenarios**:

1. **Given** a developer wants to test against a different Redpanda version, **When** they run `dp config set dev.charts.redpanda.version 24.3.1`, **Then** the next `dp dev up` deploys Redpanda using chart version 24.3.1 instead of the embedded default.
2. **Given** a chart version override is set, **When** the developer runs `dp config unset dev.charts.redpanda.version`, **Then** subsequent `dp dev up` reverts to the embedded default version.
3. **Given** a chart version override is invalid or incompatible, **When** `dp dev up` is run, **Then** a clear error message is shown explaining the version mismatch and suggesting the developer reset to the default.

---

### User Story 5 - Extra Helm Values via Config (Priority: P3)

A developer can pass additional Helm values to any dependency chart via config, enabling customization of resource limits, ports, or other chart parameters without editing charts. This supports the existing config hierarchy and allows team-wide overrides at the repo level.

**Why this priority**: This is an escape hatch for edge cases. Most developers will use defaults.

**Independent Test**: Can be tested by setting extra values via `dp config set dev.charts.postgres.values.resources.limits.memory 512Mi`, running `dp dev up`, and verifying the PostgreSQL pod has the custom memory limit.

**Acceptance Scenarios**:

1. **Given** a developer needs more memory for PostgreSQL, **When** they set `dp config set dev.charts.postgres.values.resources.limits.memory 512Mi`, **Then** the next `dp dev up` applies the custom memory limit to the PostgreSQL deployment.
2. **Given** extra values are set at the repo config level, **When** any developer on the team runs `dp dev up`, **Then** the repo-level values are applied (unless overridden at user or system level).

---

### Edge Cases

- What happens when a developer has an existing dev stack running the old (non-Helm) deployment? The system should detect and offer to migrate or require `dp dev down` first.
- What happens when the k3d cluster has stale Helm releases from a previous version? `dp dev up` should handle upgrades gracefully via `helm upgrade --install`.
- What happens when port-forwarding fails after Helm deployment? The system should retry port-forward setup and provide clear diagnostics.
- What happens when an upstream subchart is unavailable (network issues)? Since charts are embedded, offline operation should still work with the bundled version. Override versions that require network access should fail fast with a clear message.
- What happens when a developer downgrades to a CLI version with older embedded charts? `helm upgrade --install` handles rollbacks; the system should warn about version changes.
- What happens when some charts deploy successfully but others fail? The system keeps successful releases running, reports failures with clear errors, and a re-run of `dp dev up` fixes the failed charts without touching the healthy ones.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST embed all dev dependency Helm charts in the binary so that no external chart files or repositories are required for standard operation.
- **FR-002**: The CLI MUST maintain one Helm chart per dev dependency in the existing repository subdirectory (`sdk/localdev/charts/`), with each chart following standard Helm chart structure (`Chart.yaml`, `values.yaml`, `templates/`). Existing charts (redpanda, localstack, postgres) MUST be refactored in-place to become subchart wrappers where upstream charts are available, preserving directory structure and Git history.
- **FR-003**: Where a well-maintained upstream Helm chart exists for a dependency, the repo's chart MUST declare it as a subchart dependency in `Chart.yaml` with a pinned version.
- **FR-004**: Where no suitable upstream chart exists, the repo MUST maintain a custom chart with the minimal resources needed for local development.
- **FR-005**: The CLI MUST deploy all dev dependency charts using a single, uniform deployment function -- no per-service bespoke deployment code.
- **FR-006**: Each chart MUST include initialization logic (via Helm hooks, init containers, or post-install jobs) to create required resources (topics, buckets, schemas, tables, namespaces) automatically upon deployment.
- **FR-007**: The `dp dev up` command MUST deploy all dependency charts in parallel where possible and wait for all to become healthy. If some charts fail while others succeed, the successful releases MUST be kept running and a clear error report MUST list the failed charts with reasons. A subsequent `dp dev up` MUST fix the failed charts idempotently without redeploying the healthy ones.
- **FR-008**: The `dp dev down` command MUST uninstall all Helm releases and clean up associated resources.
- **FR-009**: The `dp dev status` command MUST report the status of each Helm release including chart version, app version, and health.
- **FR-010**: The CLI MUST support overriding the chart version for any dependency via the existing config system (`dp config set dev.charts.<name>.version <version>`).
- **FR-011**: The CLI MUST support passing additional Helm values for any dependency via config (`dp config set dev.charts.<name>.values.<path> <value>`).
- **FR-012**: The Marquez dependency (lineage service, database, and web UI) MUST be added as a Helm chart for the k3d runtime, bringing parity with the docker-compose runtime.
- **FR-013**: Port-forwarding for k3d-deployed services MUST continue to expose the same localhost ports as the current implementation (19092 for Kafka, 8080 for Redpanda Console, 4566 for S3, 5432 for PostgreSQL, 5000/5001 for Marquez API, 3000 for Marquez Web).
- **FR-014**: The embedded chart archives (`.tgz` for subchart dependencies) MUST be included in the binary embed, so `helm dependency build` is not required at runtime.

### Key Entities

- **Dev Dependency Chart**: A Helm chart representing a single infrastructure dependency needed for local development. Contains chart metadata, default values tuned for dev use, templates for Kubernetes resources, and optional init jobs for resource seeding.
- **Chart Registry (embedded)**: The collection of all dev dependency charts embedded in the CLI binary via Go's `embed.FS`. Provides the source of truth for default chart versions and configurations.
- **Chart Override Config**: Configuration entries under `dev.charts.<name>` in the hierarchical config system that allow version and value overrides per dependency.
- **Init Job**: A Kubernetes Job or Helm hook that runs after a dependency is deployed to create required resources (topics, buckets, schemas). Modeled as part of the dependency's Helm chart.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All dev dependencies (Redpanda, LocalStack, PostgreSQL, Marquez) are deployed via a uniform Helm-based mechanism -- zero bespoke per-service deployment code remains.
- **SC-002**: A developer can run `dp dev up` on a fresh k3d cluster and have a fully functional dev environment (all services healthy, all resources seeded) within 3 minutes.
- **SC-003**: Adding a new dev dependency requires only adding a new chart directory and registering it -- no changes to deployment orchestration code.
- **SC-004**: The dev environment works fully offline (no network access) when using default embedded charts.
- **SC-005**: Chart version overrides applied via config take effect on the next `dp dev up` without requiring a CLI rebuild.
- **SC-006**: All existing E2E and integration tests continue to pass with the new Helm-based deployment, confirming backward-compatible behavior.
- **SC-007**: The k3d dev environment reaches feature parity with the docker-compose runtime (Marquez is included).

## Assumptions

- **A-001**: The k3d runtime is the primary target for Helm chart-based dependencies. The docker-compose runtime will continue to work as-is and is out of scope for this feature.
- **A-002**: Upstream Helm charts exist and are suitable for: Redpanda (redpanda-data/helm-charts) and PostgreSQL (bitnami/charts). LocalStack and Marquez will use custom charts.
- **A-003**: Helm 3 is already a prerequisite for the k3d runtime and does not need to be newly introduced.
- **A-004**: The existing `embed.FS` pattern used for current charts will be extended to accommodate subchart archives (`.tgz` files).
- **A-005**: Chart version overrides will pull from public Helm registries; the user is responsible for network access when using non-default versions.
- **A-006**: Resource requirements for local dev charts will be minimal (no persistence, low CPU/memory limits) to run on developer laptops.

## Clarifications

### Session 2026-02-15

- Q: Should Redpanda Console (web UI for Kafka, port 8080) be included as a separate chart, bundled into the Redpanda chart, or excluded? → A: Bundle Redpanda Console into the Redpanda chart as a second deployment (single chart, two pods).
- Q: If some charts deploy successfully but others fail during parallel deployment, what should happen? → A: Keep successful services running, report failures with clear errors, and let a re-run of `dp dev up` fix the failed charts idempotently.
- Q: Should the existing custom chart directories be refactored in-place to become subchart wrappers, or replaced with fresh charts? → A: Refactor in-place — add subchart dependency to existing `Chart.yaml`, replace custom templates with upstream values overrides, preserving directory structure and Git history.
