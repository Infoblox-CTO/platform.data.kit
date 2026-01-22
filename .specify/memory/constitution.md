<!--
================================================================================
SYNC IMPACT REPORT
================================================================================
Version Change: 1.1.0 → 1.2.0
Bump Rationale: Added Unit Testing Requirements and CLI Name standards

Modified Principles: N/A

Added Sections:
  - Technology Standards
    - Unit Testing Requirements: Comprehensive unit test mandate
    - CLI Name: dp as the official CLI name

Removed Sections: N/A

Templates Requiring Updates: None

Follow-up TODOs:
  - Add unit tests to all packages
  - Ensure CI/CD runs go test ./... 
  - Verify test coverage meets 80% threshold
================================================================================
-->

# DP Constitution (Project Governance)

## Purpose

This constitution defines the non-negotiable principles for building the Data Platform (DP). All specs, plans, tasks, and implementation decisions MUST align with these principles.

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

### Article IV — Separation of Concerns (Infra vs Pipelines)

Infrastructure provisioning/configuration is separate from pipeline definitions.

- Pipelines MUST reference infrastructure through bindings/contracts, not hardcoded identifiers.
- Multiple infrastructure "flavors" may exist as long as contracts remain satisfied.
- Environment-specific configuration MUST NOT leak into package definitions.

**Rationale**: Tight coupling prevents portability and makes testing across environments impossible.

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

### CLI Name

- The CLI binary MUST be named `dp` (short for Data Platform).
- All documentation, examples, and help text MUST use `dp` as the command name.

**Rationale**: Short CLI names improve developer ergonomics and reduce typing friction.

## Definition of Done (for MVP increments)

A feature is "done" only when:

1. **Tested**: It has tests appropriate to risk (unit and/or integration).
2. **Documented**: It has documentation (README or docs section) for usage and troubleshooting.
3. **Observable**: It has observable behavior (metrics/logs) if it affects runtime execution.
4. **Reversible**: It has a rollback story if it affects deployments or contracts.

## Versioning and Compatibility Rules

- Contracts and CLI/platform modules use SemVer (MAJOR.MINOR.PATCH).
- Backward compatibility is favored:
  - Add fields as optional first.
  - Keep old fields for at least one minor release with deprecation notes.
- Breaking changes require:
  - Major version bump (or explicit vNext manifest apiVersion).
  - Migration guidance documentation.

## MVP Guardrails (Anti-Goals)

The following are explicitly out of scope for MVP:

- **Do NOT** build a new scheduler.
- **Do NOT** build a full enterprise data marketplace in MVP.
- **Do NOT** introduce complex multi-tenancy until MVP workflow is proven.
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

**Version**: 1.2.0 | **Ratified**: 2026-01-22 | **Last Amended**: 2026-01-22
