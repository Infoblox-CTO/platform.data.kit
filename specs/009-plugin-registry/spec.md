# Feature Specification: Plugin Registry & Configuration Management

**Feature Branch**: `009-plugin-registry`
**Created**: 2026-02-13
**Status**: Draft
**Input**: User description: "Pull CloudQuery destination plugins from a public OCI registry (ghcr.io/infobloxopen/), support configurable plugin mirrors/registries, provide a `dp config` subcommand, and implement a hierarchical config file lookup (git repo → $HOME → /etc/datakit/)."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Pull Destination Plugins from OCI Registry (Priority: P1)

A developer runs `dp run ./my-source --sync` and the CLI pulls the required destination plugin as a pre-built OCI image from the default public registry (`ghcr.io/infobloxopen/cloudquery-plugin-<name>:<version>`) instead of building from source. The image is already compatible with the k3d development cluster — it runs as a gRPC server on port 7777 — so the CLI deploys it as a Pod alongside the source plugin and runs `cloudquery sync`.

**Why this priority**: This is the core value — replacing the slow sparse-clone-and-build flow (minutes) with a fast image pull (seconds), using pre-built images from the Infoblox builder repository.

**Independent Test**: Run `dp run ./my-source --sync --destination postgresql` and verify the CLI pulls the image from `ghcr.io/infobloxopen/cloudquery-plugin-postgresql:v8.14.1`, deploys it as a Pod in the k3d cluster, and successfully syncs data from the source to the destination.

**Acceptance Scenarios**:

1. **Given** a CloudQuery source plugin project, **When** the user runs `dp run . --sync`, **Then** the CLI pulls `ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1` into the k3d cluster and runs a full sync.
2. **Given** a CloudQuery source plugin project, **When** the user runs `dp run . --sync --destination postgresql`, **Then** the CLI pulls the PostgreSQL destination image and syncs data to the PostgreSQL instance running in the dev environment.
3. **Given** the destination image is already imported into the k3d cluster from a prior run, **When** the user runs `dp run . --sync` again, **Then** the CLI reuses the cached image (no redundant pull) and the sync completes faster.
4. **Given** the registry is unreachable, **When** the user runs `dp run . --sync`, **Then** the CLI reports a clear error message indicating the pull failed and suggests checking network connectivity or providing an alternative mirror.

---

### User Story 2 — Hierarchical Configuration File (Priority: P2)

A developer uses a configuration file to set persistent defaults — plugin versions, registry mirrors, and other CLI settings — without needing command-line flags on every invocation. The CLI looks for configuration in a well-defined hierarchy: first in the current Git repository root (`.dp/config.yaml`), then in the user's home directory (`~/.config/dp/config.yaml`), then at a system-wide path (`/etc/datakit/config.yaml`). Settings from narrower scopes override broader ones.

**Why this priority**: Enables teams to share project-level defaults (checked into the repo) while allowing individual users and platform administrators to set their own overrides.

**Independent Test**: Create a `.dp/config.yaml` in a Git repo root with a custom registry mirror. Run `dp run . --sync` and verify the CLI uses the mirror from the config file.

**Acceptance Scenarios**:

1. **Given** a `.dp/config.yaml` file in the Git repo root with `plugins.registry: ghcr.io/myorg`, **When** the user runs `dp run . --sync`, **Then** the CLI pulls images from `ghcr.io/myorg/cloudquery-plugin-<name>:<version>`.
2. **Given** no repo-level config but `~/.config/dp/config.yaml` exists, **When** the user runs any `dp` command, **Then** the CLI uses settings from the home-level config.
3. **Given** config files exist at both repo and home levels with overlapping keys, **When** the user runs `dp run . --sync`, **Then** repo-level settings take precedence over home-level settings.
4. **Given** no config files exist at any level, **When** the user runs `dp run . --sync`, **Then** the CLI uses built-in defaults (public Infoblox registry, latest pinned versions).

---

### User Story 3 — `dp config` Subcommand for Managing Settings (Priority: P3)

A developer manages CLI configuration through the `dp config` subcommand instead of manually editing YAML files. They can view current configuration, set/unset values, and manage plugin mirrors — all with intuitive commands.

**Why this priority**: Lowers the barrier to configuration — users do not need to learn the config file schema or remember YAML syntax.

**Independent Test**: Run `dp config set plugins.registry ghcr.io/myteam` and verify the value is persisted and used by subsequent commands.

**Acceptance Scenarios**:

1. **Given** no existing configuration, **When** the user runs `dp config set plugins.registry ghcr.io/myteam`, **Then** a config file is created at the default location with the specified value.
2. **Given** an existing config file, **When** the user runs `dp config get plugins.registry`, **Then** the current effective value is displayed (showing which scope it came from).
3. **Given** an existing mirror, **When** the user runs `dp config unset plugins.registry`, **Then** the custom value is removed and the CLI reverts to the built-in default.
4. **Given** no configuration, **When** the user runs `dp config list`, **Then** all effective settings are displayed with their source (built-in, system, user, or repo).

---

### User Story 4 — Plugin Version and Image Override Management (Priority: P4)

A developer pins a specific plugin to a custom version or overrides its image name entirely — for example, to use an internal fork or a locally-patched build. These overrides are persisted in the config file and apply to all subsequent runs.

**Why this priority**: Enables advanced customization — teams can run internal forks of plugins, pin to approved versions, or use entirely custom images.

**Independent Test**: Run `dp config set plugins.overrides.postgresql.version v8.13.0` and then run `dp run . --sync --destination postgresql`. Verify the CLI pulls the v8.13.0 image instead of the default.

**Acceptance Scenarios**:

1. **Given** no overrides, **When** the user runs `dp config set plugins.overrides.postgresql.version v8.13.0`, **Then** subsequent syncs pull the v8.13.0 image for PostgreSQL.
2. **Given** no overrides, **When** the user runs `dp config set plugins.overrides.postgresql.image internal.registry.io/custom-pg:latest`, **Then** subsequent syncs use the full custom image name, bypassing the registry + naming convention entirely.
3. **Given** an existing version override, **When** the user runs `dp config unset plugins.overrides.postgresql.version`, **Then** the plugin reverts to the built-in default version.
4. **Given** a custom image override, **When** the user runs `dp config list`, **Then** the override is visible alongside other settings with its scope and value.

---

### User Story 5 — Mirror Management Commands (Priority: P5)

A developer adds fallback registries (mirrors) so that if the primary registry is unavailable, the CLI tries alternative locations. This is especially useful for air-gapped environments or when migrating between registries.

**Why this priority**: Resilience and flexibility for enterprise environments where registry availability cannot be guaranteed.

**Independent Test**: Run `dp config add-mirror ghcr.io/backup-org`, then disconnect from the primary registry, and verify the CLI falls back to the mirror.

**Acceptance Scenarios**:

1. **Given** no mirrors configured, **When** the user runs `dp config add-mirror ghcr.io/backup-org`, **Then** the mirror is added to the config and displayed in `dp config list`.
2. **Given** multiple mirrors configured, **When** the primary registry is unreachable, **Then** the CLI tries each mirror in order until one succeeds.
3. **Given** a mirror is configured, **When** the user runs `dp config remove-mirror ghcr.io/backup-org`, **Then** the mirror is removed from the config.
4. **Given** mirrors at both repo and user level, **When** the user runs `dp config list`, **Then** all mirrors are shown with their scope, and the effective order is clear.

---

### Edge Cases

- What happens when the same plugin is overridden at both repo and user scope with different values? Repo scope wins, with a warning displayed to the user.
- What happens when an image tag does not exist in the registry? Clear error: "image not found: <full-image-ref>. Run `dp config list` to check your settings."
- What happens when Docker/containerd is not running? Error with remediation: "Docker is not running. Start Docker and retry."
- What happens when the user specifies `--destination` with a plugin name that has no known image? Error listing the supported plugins and suggesting `dp config set plugins.overrides.<name>.image <image>` to register a custom one.
- What happens when the config file has invalid YAML? Error with the file path and parse error, suggesting the user inspect or recreate the file.
- What happens when two config scopes define conflicting mirrors? Mirrors are merged (repo-level mirrors are checked first, then user-level, then system-level). Duplicates are deduplicated.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST pull CloudQuery destination plugins as OCI container images from a configurable registry using `docker pull`.
- **FR-002**: System MUST use the default image naming convention `<registry>/cloudquery-plugin-<name>:<version>` where registry defaults to `ghcr.io/infobloxopen`.
- **FR-003**: System MUST deploy pulled destination plugin images as Kubernetes Pods in the k3d cluster (matching the existing source plugin deployment pattern).
- **FR-004**: System MUST look for configuration files in the following order (highest priority first): `.dp/config.yaml` in the current Git repository root, `~/.config/dp/config.yaml` in the user home directory, `/etc/datakit/config.yaml` as a system-wide fallback.
- **FR-005**: System MUST merge configuration from all scopes, with narrower scopes overriding broader scopes for scalar values.
- **FR-006**: System MUST provide a `dp config set <key> <value>` command that writes to the user-level config file by default.
- **FR-007**: System MUST provide a `dp config get <key>` command that displays the effective value and its source scope.
- **FR-008**: System MUST provide a `dp config unset <key>` command that removes a value from the config.
- **FR-009**: System MUST provide a `dp config list` command that displays all effective settings with their source scope.
- **FR-010**: System MUST support per-plugin version overrides via `plugins.overrides.<name>.version`.
- **FR-011**: System MUST support per-plugin image overrides via `plugins.overrides.<name>.image`, which bypasses the registry + naming convention entirely.
- **FR-012**: System MUST provide `dp config add-mirror <registry>` and `dp config remove-mirror <registry>` commands for managing fallback registries.
- **FR-013**: System MUST try mirrors in order when the primary registry is unreachable, reporting which mirror succeeded.
- **FR-014**: System MUST provide a `dp config set <key> <value> --scope repo` option to write settings to the repo-level config file (and `--scope user` for home-level, `--scope system` for system-level).
- **FR-015**: System MUST maintain backward compatibility with the existing `~/.config/dp/config.yaml` used by `dp dev`.
- **FR-016**: System MUST validate configuration values at write time (e.g., reject invalid registry URLs, invalid version formats).
- **FR-017**: System MUST support a `--registry` CLI flag on `dp run` that overrides the config file for a single invocation.

### Key Entities

- **Plugin Registry**: The OCI container registry from which destination plugin images are pulled. Has an address (e.g., `ghcr.io/infobloxopen`), an image naming convention, and an optional list of mirrors.
- **Plugin Override**: A per-plugin configuration that allows changing the version, or the full image reference. Stored as a map keyed by plugin short name (e.g., `postgresql`, `file`, `s3`).
- **Config Scope**: One of three levels — repo (`.dp/config.yaml` in Git root), user (`~/.config/dp/config.yaml`), or system (`/etc/datakit/config.yaml`). Determines precedence when merging.
- **Mirror**: An alternative registry address tried when the primary registry is unreachable. Ordered list, tried sequentially.
- **Config File**: A YAML file containing persistent CLI settings. Multiple scopes are merged at runtime.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can execute `dp run . --sync` and get a working source-to-destination sync using a pre-built container image in under 30 seconds (excluding first image pull).
- **SC-002**: First-time image pull for a destination plugin completes in under 60 seconds on a typical broadband connection.
- **SC-003**: Developers can configure a custom registry and see it take effect on the next `dp run` invocation without restarting any services.
- **SC-004**: 100% of configuration operations via `dp config` produce valid config files that are parseable by subsequent commands.
- **SC-005**: All existing `dp dev` and `dp run` functionality continues to work without any changes to existing config files.
- **SC-006**: Developers unfamiliar with the config schema can successfully set a custom registry using only `dp config` commands (no manual YAML editing required).

### Assumptions

- Docker is available and can pull images from the configured registry.
- The k3d cluster is running and accessible via `kubectl` (verified by existing `verifyClusterRunning` check).
- The container images at `ghcr.io/infobloxopen/` follow the naming convention `cloudquery-plugin-<name>:<version>` and expose gRPC on port 7777.
- The entrypoint of each destination image runs the plugin in gRPC serve mode (consistent with the cloudquery-plugins-builder runtime contract).
- Git is available to find the repository root for repo-scoped config lookup.
- The existing `~/.config/dp/config.yaml` format (used by `dp dev`) is extended — never replaced — by the new plugin configuration keys.