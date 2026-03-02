<!--
================================================================================
SYNC IMPACT REPORT
================================================================================
Version Change: 2.0.0 → 3.0.0
Bump Rationale: Product rebrand from DP to DK (DataKit), pre-production
  backward compatibility policy, K8s API group change to datakit.infoblox.dev.

Modified Principles:
  - Article II: Removed backward compat requirement for pre-production phase

Modified Sections:
  - CLI Name: dp → dk (DataKit)
  - Versioning and Compatibility Rules: Added pre-production exemption
  - MVP Guardrails: Updated dp → dk references, added no-backward-compat rule
  - Title: DP Constitution → DK Constitution

Previous Change (2.0.0):
  - Article IV: Expanded from "Infra vs Pipelines" to "Platform vs Domain"
  - Added Articles X, XI, Python Testing, Schema-validated gate, Persona Mapping gate

Removed Sections: N/A

Templates Requiring Updates:
  - plan-template.md: Constitution Check table needs Articles X and XI rows

Follow-up TODOs:
  - Update CONTRIBUTING.md constitution summary to reflect DK rename
  - Ensure all future specs use dk in CLI references
================================================================================
-->

# DK Constitution (Project Governance)

## Purpose

This constitution defines the non-negotiable principles for building DataKit (DK). All specs, plans, tasks, and implementation decisions MUST align with these principles.

## Scope

Applies to:
- CLI tooling
- Shared contracts/types and SDK
- Platform runtime components (controllers/services/operators integrations)
- GitOps environment structure and promotion mechanics
- Example/reference packages shipped with the platform

## Core Principles (Articles)

### Article I — Developer Experience is a First-Class Product

The developer workflow is a product deliverable, not an afterthought.

- The "happy path" MUST be simple: bootstrap → local run → validate → publish → promote.
- Defaults MUST work with minimal configuration.
- Every action MUST provide clear, actionable output (what happened, what to do next).

**Rationale**: Poor DX leads to workarounds, inconsistent usage, and slow adoption. If the happy path is painful, users will avoid the platform.

### Article II — Specs Define Contracts; Contracts Must Be Stable

Package manifests and artifact contracts are APIs that external consumers depend upon.

- Changes to contracts require explicit versioning and compatibility policy.
- Prefer additive evolution; avoid breaking changes unless justified and versioned.
- Contract definitions MUST be machine-readable (schemas, not prose).

**Rationale**: Unstable contracts break downstream integrations and erode trust. Treating contracts as APIs forces discipline.

### Article III — Immutability and Auditability

Once released, artifacts cannot be silently modified.

- Released package artifacts are immutable (no "mutable latest" in production).
- Promotions between environments are auditable and reviewable (PR-based workflow).
- Rollback MUST be straightforward (pin previous version).

**Rationale**: Mutability creates debugging nightmares and compliance violations. Auditability enables incident response and governance.

### Article IV — Separation of Concerns (Platform vs Domain)

The platform has four distinct layers — extension definitions, asset instances, pipelines, and infrastructure bindings — each with clear ownership boundaries.

- Extension *definitions* (schemas, templates, version policy) MUST be independent of any specific domain's configuration.
- Asset *instances* MUST reference extensions by fully-qualified name and version, never by embedding extension internals.
- Pipelines MUST reference assets by name, not by embedding asset configuration inline.
- Pipelines MUST reference infrastructure through bindings/contracts, not hardcoded identifiers.
- Multiple infrastructure "flavors" may exist as long as contracts remain satisfied.
- Environment-specific configuration (bindings, secrets, quotas) MUST NOT leak into extension, asset, or pipeline definitions.

**Rationale**: Tight coupling prevents portability and makes testing across environments impossible. Clean layer boundaries enable platform and domain teams to evolve independently.

### Article V — Security and Compliance by Default

Security is not an opt-in feature.

- Least privilege is the default posture.
- Secrets are NEVER committed; use dedicated secret mechanisms.
- Produced artifacts require classification metadata; PII handling is explicit, not implied.
- All significant actions (publish, promote, access grant) MUST be auditable.

**Rationale**: Retrofitting security is expensive and error-prone. Default-secure prevents accidental exposure.

### Article VI — Observability is Not Optional

Every runtime component and pipeline execution MUST expose observability signals.

- Standard metrics: success/failure/duration/throughput.
- Structured logs with correlation identifiers: package, version, run ID, environment.
- Dashboards MUST exist for both platform operators and package owners.

**Rationale**: Without observability, debugging is guesswork. Operators need visibility; package owners need self-service insights.

### Article VII — Quality Gates Over Heroics

Automated validation replaces manual heroics.

Before a package can be published/promoted, it MUST pass:
- Contract/schema validation
- Unit tests (where applicable)
- Basic integration sanity checks for the MVP example(s)

Prefer explicit validation errors over implicit behavior.

**Rationale**: Heroic saves don't scale. Automated gates catch issues before they reach production.

### Article VIII — Pragmatism and Incremental Delivery

Ship working software early; defer complexity.

- The MVP MUST demonstrate end-to-end value quickly.
- Defer advanced capabilities (full marketplace automation, multi-cloud abstractions) unless required for MVP success.
- Prefer proven, replaceable components over bespoke frameworks.

**Rationale**: Premature abstraction wastes effort. Incremental delivery validates assumptions early.

### Article IX — Maintainability and Operability

Code is read more than written; systems are operated longer than built.

- Keep dependencies and coupling minimal (especially in shared contracts).
- Enforce clear module boundaries and dependency direction:
  - `contracts` ← `sdk` ← `cli`
  - `contracts` ← `platform/controller`
- Documentation is REQUIRED for user-visible behaviors and operational procedures.

**Rationale**: Unmaintainable code becomes legacy quickly. Clear boundaries enable independent evolution.

### Article X — Persona Boundaries and Least Authority

The platform serves two distinct personas with different authorities.

- **Platform engineers** own extension definitions, managed environments, and policies. They define *what is allowed*.
- **Data engineers** own asset instances, pipelines, and models. They define *what runs and what it produces*.
- Each persona's artifacts MUST be independently validatable without requiring the other's tooling or credentials.
- Data engineers MUST NOT need to understand infrastructure implementation details — they declare *capabilities needed*, not *resources provisioned*.
- Platform engineers MUST NOT need to understand domain business logic — they define *guardrails*, not *pipeline content*.

**Rationale**: Unclear ownership creates friction, security gaps, and operational confusion. Clean persona boundaries enable independent velocity while maintaining governance.

### Article XI — Extensions are Contracts

Extensions define the approved building blocks of the platform. They are APIs between platform and domain teams.

- Every extension MUST have a machine-readable schema (`schema.json`) that validates consumer configuration.
- Extension schemas MUST be versioned; breaking changes require a major version bump.
- Extensions MUST be self-describing: schema, documentation, examples, and version policy travel together.
- The platform MUST validate asset configuration against the referenced extension's schema at `dp validate` time — not at runtime.

**Rationale**: Unschematized configuration leads to runtime failures and support burden. Schema-validated extensions shift errors left and create a reliable contract between platform and domain teams.

## Technology Standards

### Go Version

- All Go code MUST use the **latest stable version** of Go.
- When a new stable Go version is released, codebases MUST be updated within one minor release cycle.
- The `go.mod` file MUST specify the current latest stable version.

**Rationale**: Staying current with Go versions ensures access to performance improvements, security patches, and language features while avoiding technical debt from version lag.

### Unit Testing Requirements

- All packages MUST have comprehensive unit tests.
- Test files MUST follow the `*_test.go` naming convention in the same package directory.
- Unit tests MUST cover:
  - All exported functions and methods
  - Edge cases and error conditions
  - Input validation logic
- Prefer table-driven tests for functions with multiple input scenarios.
- Mock external dependencies (HTTP clients, databases, file systems) in unit tests.
- Test coverage SHOULD aim for 80% or higher for business logic packages.
- Tests MUST be runnable via `go test ./...` without external dependencies.

**Rationale**: Unit tests catch regressions early, document expected behavior, and enable confident refactoring. Table-driven tests improve maintainability and coverage.

### Python Testing Requirements

- Python packages MUST have tests runnable via `pytest` without external dependencies.
- Test files MUST follow the `test_*.py` naming convention.
- Tests MUST cover exported functions, edge cases, and input validation.
- Test coverage SHOULD aim for 80% or higher for business logic modules.
- Mock external dependencies (HTTP clients, databases, file systems) in unit tests.

**Rationale**: Python is a supported language for extensions (e.g., CloudQuery plugins). Consistent testing standards across languages prevent quality gaps.

### CLI Name

- The CLI binary MUST be named `dk` (short for DataKit).
- All documentation, examples, and help text MUST use `dk` as the command name.
- The K8s API group MUST be `datakit.infoblox.dev`.
- All K8s labels MUST use the `datakit.infoblox.dev/` domain prefix.

**Rationale**: `dk` (DataKit) is the product identity. Short CLI names improve developer ergonomics. The `datakit.infoblox.dev` domain is owned by Infoblox and follows K8s API group conventions.

## Definition of Done (for MVP increments)

A feature is "done" only when:

1. **Tested**: It has tests appropriate to risk (unit and/or integration).
2. **Documented**: It has documentation (README or docs section) for usage and troubleshooting.
3. **Observable**: It has observable behavior (metrics/logs) if it affects runtime execution.
4. **Reversible**: It has a rollback story if it affects deployments or contracts.
5. **Schema-validated**: If the feature introduces or modifies a user-facing configuration surface, it MUST have a machine-readable schema and validation at write/validate time.

## Versioning and Compatibility Rules

- Contracts and CLI/platform modules use SemVer (MAJOR.MINOR.PATCH).
- **Pre-production phase** (current): Backward compatibility is NOT guaranteed. Breaking changes may be introduced at any time without migration guidance. All effort is focused on making the project consistent with currently proposed concepts.
- **Post-production release**: Once the project is declared production-ready, backward compatibility MUST be maintained:
  - Add fields as optional first.
  - Keep old fields for at least one minor release with deprecation notes.
  - Breaking changes require major version bump (or explicit vNext manifest apiVersion) and migration guidance documentation.

## MVP Guardrails (Anti-Goals)

The following are explicitly out of scope for MVP:

- **Do NOT** build a new scheduler.
- **Do NOT** build a full enterprise data marketplace in MVP.
- **Do NOT** introduce complex multi-tenancy until MVP workflow is proven.
- **Do NOT** build a custom policy engine — start with declarative YAML policies evaluated at `dk validate` time.
- **Do NOT** build dynamic extension discovery or marketplace UI — extensions are registered via `dk ext publish` and referenced by FQN.
- **Do NOT** invest in backward compatibility, migration tooling, or deprecation paths until the project reaches production release. Consistency with current design takes priority over preserving old behavior.
- **Do NOT** build a multi-repo extension resolution protocol — start with a single OCI registry.
- The extension type system SHOULD launch with existing CloudQuery as the first built-in extension, not with a broad catalog.
- **Avoid** over-generalizing beyond current requirements; design for extensibility, not maximal abstraction.

## Pre-Implementation Gates

Before implementing any major milestone, the following MUST be verified:

| Gate | Requirement |
|------|-------------|
| **Workflow Demo** | The plan demonstrates the end-to-end developer workflow |
| **Contract Schema** | Contract schemas and validation strategy are explicit |
| **Promotion/Rollback** | Promotion and rollback mechanics are explicit |
| **Observability** | Observability requirements are defined and addressed |
| **Security/Compliance** | Secrets, least privilege, and PII metadata requirements are addressed (at minimum) |
| **Persona Mapping** | The plan identifies which persona owns each artifact and validates that ownership boundaries are clean |

## Governance

### Amendment Process

1. Proposed amendments MUST be documented with rationale.
2. Amendments require review and approval via PR.
3. Migration guidance MUST be provided for changes affecting existing implementations.

### Versioning Policy

This constitution follows SemVer:
- **MAJOR**: Backward-incompatible principle removals or redefinitions.
- **MINOR**: New principles/sections added or materially expanded guidance.
- **PATCH**: Clarifications, wording, typo fixes, non-semantic refinements.

### Compliance Review

- All plans and specs MUST include a Constitution Check section.
- PRs affecting contracts, security, or observability MUST reference relevant articles.
- Complexity exceeding constitution guidelines MUST be explicitly justified.

---

**Version**: 3.0.0 | **Ratified**: 2026-01-22 | **Last Amended**: 2026-03-01
