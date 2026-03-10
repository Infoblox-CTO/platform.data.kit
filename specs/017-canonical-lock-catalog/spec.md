# Feature Specification: Canonical Lock & Catalog Model

**Feature Branch**: `017-canonical-lock-catalog`
**Created**: 2026-03-07
**Status**: Draft
**Input**: User description: "Create a consistent, trustworthy data model for apx.lock and catalog metadata. The feature should ensure that lockfiles and catalog entries use one canonical structure, capture the data developers need for reproducible dependency resolution and discovery, and avoid conflicting representations across docs and code. Success means dependency state and catalog state are deterministic, well-defined, and easy to inspect."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Deterministic Dependency Locking (Priority: P1)

A data engineer declares dependencies in `dk.yaml` (connectors, stores, assets) using version constraints (e.g., `">=1.2.0"`). When they run `dk build` or a new `dk lock` command, the system resolves every constraint against available versions, selects the best match, and writes a `dk.lock` file that pins each dependency to an exact version and OCI digest. On subsequent builds or runs, the lockfile is honoured so the same artifacts are used regardless of what new versions have been published upstream.

**Why this priority**: Without deterministic locking, builds are non-reproducible. A pipeline that works today may break tomorrow because a newer connector version was published. This is the foundational capability that enables every other user story.

**Independent Test**: Create a package with a version-range dependency, run `dk lock`, verify the lockfile contents, then publish a new version of the dependency and confirm that `dk build` still resolves to the locked version.

**Acceptance Scenarios**:

1. **Given** a `dk.yaml` with `connectorVersion: ">=1.0.0"`, **When** I run `dk lock`, **Then** a `dk.lock` file is written containing the resolved version, OCI digest, and registry for every dependency.
2. **Given** an existing `dk.lock` file, **When** I run `dk build`, **Then** the build uses the exact versions from the lockfile without re-resolving against the registry.
3. **Given** an existing `dk.lock` file and a newly published connector version that satisfies the constraint, **When** I run `dk build`, **Then** the locked version is still used, not the newly published one.
4. **Given** a `dk.lock` whose pinned version no longer exists in the registry, **When** I run `dk build`, **Then** the command fails with a clear error stating which dependency is missing and suggesting `dk lock --update`.

---

### User Story 2 - Lock Update and Refresh (Priority: P2)

A data engineer wants to update one or all locked dependencies to pick up newer versions that still satisfy their declared constraints. Running `dk lock --update` re-resolves all dependencies and writes a new lockfile. Running `dk lock --update <name>` re-resolves only a single dependency. The old and new lockfile can be compared via standard diff tools.

**Why this priority**: Locking without a controlled update path leads to stale dependencies. Update must be intentional and auditable.

**Independent Test**: Lock dependencies, publish a new version of one, run `dk lock --update`, verify the lockfile changes.

**Acceptance Scenarios**:

1. **Given** a locked package, **When** I run `dk lock --update`, **Then** every dependency is re-resolved to the latest version satisfying its constraint, and the lockfile is overwritten.
2. **Given** a locked package with dependency `postgres-cdc`, **When** I run `dk lock --update postgres-cdc`, **Then** only that dependency is re-resolved; all other entries remain unchanged.
3. **Given** a locked package, **When** I run `dk lock --update` and no newer versions exist, **Then** the lockfile content is unchanged and the command reports "already up to date".

---

### User Story 3 - Publishing Populates Catalog Entry (Priority: P2)

When a package is published via `dk publish`, the registry stores not only the OCI artifact but also a machine-readable catalog entry describing the package: its name, version, kind, declared dependencies, input/output assets, owner, classification, and schema fingerprint. This entry is queryable so that other teams can discover available packages.

**Why this priority**: A lockfile references dependencies by name and version, but without a catalog there is no way to enumerate what is available. Publishing must produce catalog metadata or the lock resolution has nothing to query.

**Independent Test**: Publish a package, then query the catalog for that package name and version.

**Acceptance Scenarios**:

1. **Given** a valid package, **When** I run `dk publish`, **Then** the registry stores an OCI artifact AND a catalog entry with name, version, kind, digest, owner, inputs, outputs, and created timestamp.
2. **Given** two published versions of the same package, **When** I query the catalog by package name, **Then** both versions are returned with their metadata and I can sort by version.
3. **Given** a published package, **When** I inspect the catalog entry, **Then** the `digest` matches the OCI manifest digest of the published artifact.

---

### User Story 4 - Catalog Discovery and Search (Priority: P3)

A data engineer can search the catalog to find available packages, connectors, or assets. They can filter by kind, namespace, owner, or tags. The catalog returns enough metadata to decide whether a dependency is suitable without pulling the full artifact.

**Why this priority**: Discovery is the read-side complement to publishing. Without it, engineers must know exact names in advance.

**Independent Test**: Publish several packages with different kinds and tags, then query the catalog with filters and verify filtered results.

**Acceptance Scenarios**:

1. **Given** multiple published packages, **When** I run `dk catalog search --kind Connector`, **Then** only connector packages are returned.
2. **Given** packages with various tags, **When** I run `dk catalog search --tag production`, **Then** only packages tagged "production" are returned.
3. **Given** a search that matches no packages, **When** I run `dk catalog search --name nonexistent`, **Then** an empty result is returned with a clear message.

---

### User Story 5 - Lockfile Validation and Integrity Check (Priority: P3)

Running `dk lint` or `dk validate` on a package with a `dk.lock` verifies that every lockfile entry has a valid digest, that the lockfile is consistent with the constraints in `dk.yaml`, and that no lockfile entry references a dependency not declared in the manifest. This catches drift between the manifest and lock.

**Why this priority**: A lockfile is only trustworthy if it can be validated independently. Without validation, a manually edited or stale lockfile can silently introduce wrong versions.

**Independent Test**: Tamper with a lockfile entry, run `dk lint`, verify the error.

**Acceptance Scenarios**:

1. **Given** a `dk.yaml` and a matching `dk.lock`, **When** I run `dk lint`, **Then** lockfile validation passes.
2. **Given** a `dk.lock` entry whose version does not satisfy the `dk.yaml` constraint, **When** I run `dk lint`, **Then** a validation error identifies the mismatched dependency.
3. **Given** a `dk.lock` with an entry for a dependency not declared in `dk.yaml`, **When** I run `dk lint`, **Then** a validation error reports the orphaned lockfile entry.
4. **Given** a `dk.yaml` dependency not present in `dk.lock`, **When** I run `dk lint`, **Then** a validation error reports the unlocked dependency.

---

### Edge Cases

- What happens when two dependencies transitively require conflicting versions of the same connector? The system reports a clear conflict error during `dk lock` rather than silently choosing one.
- How does the system handle a lockfile generated on a different platform (e.g., Linux vs macOS) where OCI digests are multi-arch? The lockfile pins the platform-independent manifest digest, not a platform-specific one.
- What happens when a lockfile exists but the user has added a new dependency to `dk.yaml`? `dk build` fails and instructs the user to run `dk lock` to resolve the new dependency.
- What happens when the registry is unreachable during `dk lock`? The command fails with a network error rather than writing a partial lockfile.
- What happens when `dk.lock` is not committed to version control? `dk lint` warns that the lockfile is missing (not an error, since first `dk lock` may not have been run yet).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST define a single canonical lockfile format (YAML) named `dk.lock`, placed alongside `dk.yaml` in the package root.
- **FR-002**: Each lockfile entry MUST contain: dependency name, resolved version (exact, no ranges), OCI registry, OCI digest (content-addressable hash), and the kind of the dependency (Connector, Store, Asset).
- **FR-003**: The lockfile MUST be deterministic — given the same `dk.yaml` constraints and the same registry state, `dk lock` MUST produce byte-identical output.
- **FR-004**: The system MUST define a single canonical catalog entry structure that is stored alongside the OCI artifact at publish time.
- **FR-005**: Each catalog entry MUST contain: package name, version, kind, namespace, owner, created timestamp, OCI digest, declared inputs (assets consumed), declared outputs (assets produced), tags, and a schema fingerprint for each declared asset.
- **FR-006**: The `dk lock` command MUST resolve all dependency constraints declared in `dk.yaml` against the registry catalog and write the `dk.lock` file.
- **FR-007**: The `dk build` command MUST honour the `dk.lock` file when present, using pinned versions instead of re-resolving constraints.
- **FR-008**: The `dk lock --update` command MUST re-resolve all (or a named subset of) dependencies to the latest versions satisfying constraints and overwrite the lockfile.
- **FR-009**: The `dk lint` command MUST validate that the lockfile is consistent with `dk.yaml` constraints: no orphaned entries, no missing entries, no version-constraint violations.
- **FR-010**: The `dk publish` command MUST create or update the catalog entry for the published package version.
- **FR-011**: The catalog MUST be queryable by name, kind, namespace, owner, and tags.
- **FR-012**: The lockfile and catalog entry structures MUST be defined as Go structs in the `contracts` package with JSON and YAML struct tags, serving as the single source of truth for serialisation.
- **FR-013**: The lockfile format MUST include a schema version field so that future format changes can be detected and migrated.
- **FR-014**: The lockfile MUST be sorted deterministically (alphabetical by dependency name) to produce stable diffs.
- **FR-015**: The existing `ArtifactRef` (contracts) and `PackageRef` (controller) types MUST be unified into a single reference type used by both lockfile entries and catalog entries.

### Key Entities

- **LockFile**: The top-level lockfile structure. Contains a schema version, a reference to the source manifest (`dk.yaml`) hash, and an ordered list of locked dependency entries. Written to `dk.lock` as YAML.
- **LockedDependency**: A single resolved dependency. Contains the dependency name, kind, resolved version (exact), OCI registry, OCI digest, and the original version constraint from `dk.yaml`. Provides enough information to pull exactly the right artifact without re-resolving.
- **CatalogEntry**: Metadata for a published package version. Contains the package name, version, kind, namespace, owner, created timestamp, OCI digest, registry, declared inputs and outputs (as asset references), tags, schema fingerprints, and a description. Stored in the registry alongside the OCI artifact.
- **PackageRef** (unified): A single canonical reference to a package artifact. Contains name, namespace, version, registry, and digest. Replaces the current `ArtifactRef` and controller `PackageRef` with one type in `contracts`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `dk lock` twice on the same package against the same registry state produces byte-identical `dk.lock` files 100% of the time.
- **SC-002**: A package locked with `dk lock` builds and runs identically 30 days later, even if newer dependency versions have been published.
- **SC-003**: `dk lint` detects 100% of lockfile-vs-manifest inconsistencies (orphaned entries, missing entries, version-constraint violations) without false positives.
- **SC-004**: Publishing a package via `dk publish` creates a queryable catalog entry within the same operation, with zero additional commands required from the user.
- **SC-005**: Developers can discover available packages by kind, namespace, or tag using `dk catalog search` and receive results within 5 seconds for catalogs with up to 1,000 entries.
- **SC-006**: The `dk.lock` and catalog entry formats are each defined by exactly one Go struct in `contracts`, eliminating duplicate or conflicting type definitions.
- **SC-007**: The lockfile format passes schema validation and produces clean, human-readable diffs when committed to version control.

## Assumptions

- The OCI registry used for publishing (currently ORAS-based) supports storing metadata annotations or a secondary manifest layer for catalog entries. If not, the catalog can be stored as a dedicated OCI artifact type alongside the package artifact.
- Dependency resolution is flat (direct dependencies only for MVP). Transitive dependency resolution is deferred to a future iteration.
- The existing semver-range syntax already declared in `AssetRef.Version` and `Store.Spec.ConnectorVersion` is the intended constraint format. Standard semver range semantics (^, ~, >=, exact) apply.
- The lockfile is committed to version control alongside `dk.yaml`. This is the recommended workflow but is not enforced by the tool (only warned about by `dk lint`).
- The catalog is registry-scoped: each OCI registry maintains its own catalog. Cross-registry catalog federation is out of scope.