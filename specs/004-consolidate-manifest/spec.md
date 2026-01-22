# Feature Specification: Consolidate DataPackage Manifest

**Feature Branch**: `004-consolidate-manifest`  
**Created**: 2026-01-22  
**Status**: Draft  
**Input**: User description: "Merge pipeline.yaml into dp.yaml to have a single manifest file per DataPackage"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Define a Complete DataPackage in One File (Priority: P1)

As a data engineer, I want to define my entire DataPackage in a single `dp.yaml` file so that I don't have to maintain two tightly-coupled files and can understand my pipeline configuration at a glance.

**Why this priority**: This is the core value proposition. Eliminating the second file reduces cognitive load and maintenance burden for every DataPackage.

**Independent Test**: Create a new DataPackage with only dp.yaml (no pipeline.yaml) containing metadata, inputs, outputs, schedule, resources, and runtime configuration. Run `dp validate` and `dp run` successfully.

**Acceptance Scenarios**:

1. **Given** a dp.yaml with a `spec.runtime` section, **When** I run `dp validate`, **Then** the manifest validates successfully without requiring a separate pipeline.yaml
2. **Given** a dp.yaml with inputs that have bindings, **When** I run `dp run`, **Then** binding values are automatically mapped to environment variables using the convention `input.events.brokers` → `INPUT_EVENTS_BROKERS`
3. **Given** a dp.yaml without a runtime section, **When** I run `dp validate`, **Then** validation fails with a clear error indicating the runtime section is required

---

### User Story 2 - Override Configuration at Runtime (Priority: P2)

As a data engineer, I want to override specific configuration values when running a DataPackage so that I can tune resources, schedules, or other settings for different environments without modifying the source manifest.

**Why this priority**: Enables environment-specific tuning without file duplication, which is essential for promoting packages across environments.

**Independent Test**: Run a DataPackage with `--set` flags and verify the overridden values take effect in the generated Kubernetes resources.

**Acceptance Scenarios**:

1. **Given** a dp.yaml with `resources.memory: "4Gi"`, **When** I run `dp run --set resources.memory=8Gi`, **Then** the container runs with 8Gi memory
2. **Given** an overrides.yaml file with schedule changes, **When** I run `dp run -f overrides.yaml`, **Then** the schedule from the overrides file is used
3. **Given** both `-f overrides.yaml` and `--set key=value`, **When** I run `dp run`, **Then** `--set` values take precedence over values in the overrides file
4. **Given** an override for a non-existent path, **When** I run `dp run --set invalid.path=value`, **Then** the command fails with a clear error message

---

### User Story 3 - View Effective Configuration (Priority: P3)

As a data engineer, I want to see the final merged configuration after applying overrides so that I can verify what will actually be deployed before running.

**Why this priority**: Debugging and verification capability that builds confidence in the override system.

**Independent Test**: Run `dp show` with various override combinations and verify the output reflects correct merging.

**Acceptance Scenarios**:

1. **Given** a dp.yaml and an overrides file, **When** I run `dp show -f overrides.yaml`, **Then** I see the merged manifest with overrides applied
2. **Given** multiple `--set` flags, **When** I run `dp show --set a=1 --set b=2`, **Then** both overrides are reflected in the output
3. **Given** no overrides, **When** I run `dp show`, **Then** I see the original dp.yaml content unchanged

---

### User Story 4 - Validate with Overrides (Priority: P3)

As a data engineer, I want to validate my manifest with overrides applied so that I can catch errors before attempting to run.

**Why this priority**: Validation before execution prevents runtime failures.

**Independent Test**: Run `dp validate -f overrides.yaml` with valid and invalid overrides.

**Acceptance Scenarios**:

1. **Given** valid overrides, **When** I run `dp validate -f overrides.yaml`, **Then** validation passes
2. **Given** overrides that would result in invalid configuration (e.g., negative memory), **When** I run `dp validate -f overrides.yaml`, **Then** validation fails with descriptive errors

---

### Edge Cases

- What happens when both dp.yaml and pipeline.yaml exist in the same directory? The CLI outputs a warning that pipeline.yaml is deprecated and ignored.
- What happens when a binding reference in dp.yaml doesn't exist in the environment's bindings? The CLI fails with a clear error listing the missing bindings.
- What happens when an override path is valid but the value type is wrong (e.g., `--set resources.memory=not-a-size`)? Validation fails with a type error.
- What happens when env var naming convention creates a collision? The last binding in document order wins, with a warning logged.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a `spec.runtime` section in dp.yaml for container runtime configuration
- **FR-002**: System MUST auto-map bindings to environment variables using convention: `{category}.{name}.{property}` → `{CATEGORY}_{NAME}_{PROPERTY}` (uppercase, dots to underscores)
- **FR-003**: System MUST support `spec.runtime.image` for container image specification with `${VAR}` substitution
- **FR-004**: System MUST support `spec.runtime.timeout` with default of `1h`
- **FR-005**: System MUST support `spec.runtime.retries` with default of `3`
- **FR-006**: System MUST support `spec.runtime.env` for custom environment variables beyond auto-mapped bindings
- **FR-007**: System MUST support `spec.runtime.envFrom` for secretRef and configMapRef references
- **FR-008**: `dp run` command MUST accept `--set key=value` flag for runtime overrides (repeatable)
- **FR-009**: `dp run` command MUST accept `-f filename.yaml` flag for override files (repeatable)
- **FR-010**: `dp validate` command MUST validate merged manifest when `-f` or `--set` flags provided
- **FR-011**: `dp show` command MUST display effective manifest after applying overrides
- **FR-012**: System MUST output a deprecation warning when pipeline.yaml is detected alongside dp.yaml
- **FR-013**: Override precedence MUST be: dp.yaml < -f files (in order) < --set flags (in order)
- **FR-014**: All documentation MUST be updated to reflect single-file approach
- **FR-015**: Example DataPackages MUST be updated to use consolidated dp.yaml format

### Key Entities

- **DataPackage Manifest (dp.yaml)**: Single file defining the complete DataPackage including metadata, inputs, outputs, schedule, resources, governance, and runtime configuration
- **Runtime Section**: New section in dp.yaml containing image, timeout, retries, env, and envFrom
- **Overrides File**: Optional YAML file containing partial configuration to merge with dp.yaml
- **Binding-to-EnvVar Mapping**: Automatic conversion of binding references to container environment variables

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can define a complete DataPackage with a single dp.yaml file (no pipeline.yaml required)
- **SC-002**: Users can run the same DataPackage with different configurations using only CLI flags
- **SC-003**: All existing example DataPackages work with the new consolidated format
- **SC-004**: Documentation clearly explains the single-file approach with no references to the deprecated pipeline.yaml pattern
- **SC-005**: Users can preview effective configuration before running via `dp show`
- **SC-006**: Override merging follows predictable precedence rules

## Assumptions

- The binding system remains unchanged; this feature only affects how binding values are mapped to environment variables
- Environment variable naming collisions are rare and the "last wins with warning" approach is acceptable
- The `${VAR}` syntax for image substitution uses shell environment variables at runtime
- Override files use the same YAML structure as dp.yaml (partial trees that merge at matching paths)
