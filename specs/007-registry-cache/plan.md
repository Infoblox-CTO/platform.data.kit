# Implementation Plan: Registry Pull-Through Cache for k3d Local Development

**Branch**: `007-registry-cache` | **Date**: 2026-01-28 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/007-registry-cache/spec.md`

## Summary

Implement a Docker registry pull-through cache for k3d local development that automatically starts before cluster creation, persists cached images across cluster recreations, and provides transparent proxying to Docker Hub. The cache eliminates redundant image downloads, reducing cluster startup time and bandwidth usage. Implementation uses Docker CLI exec-based container management integrated into the existing `sdk/localdev` package.

## Technical Context

**Language/Version**: Go 1.25 (matches existing codebase per go.mod)  
**Primary Dependencies**: Docker CLI (exec-based), k3d CLI, registry:2 image  
**Storage**: Docker volume `dev_registry_cache` for cached image layers  
**Testing**: go test with table-driven tests (per constitution)  
**Target Platform**: macOS and Linux with Docker (Docker Desktop or native Docker)  
**Project Type**: Single CLI/SDK project (extends existing structure)  
**Performance Goals**: Cached image pulls <5 seconds (vs 30+ seconds from Docker Hub)  
**Constraints**: Port 5000 availability, Docker running, k3d v5+ installed  
**Scale/Scope**: Single developer workstation, ~10GB typical cache size

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Article | Requirement | Status | Evidence |
|---------|-------------|--------|----------|
| **I — Developer Experience** | Happy path simple; clear output | ✅ PASS | `dp dev up` auto-starts cache; status shows cache state |
| **II — Stable Contracts** | Contracts machine-readable | ✅ PASS | Config files use YAML with documented schema |
| **III — Immutability/Auditability** | Auditable operations | ✅ PASS | Container labels track config hash; volume lifecycle explicit |
| **IV — Separation of Concerns** | Infra via bindings | ✅ PASS | k3d cluster uses registries.yaml binding, not hardcoded |
| **V — Security by Default** | Least privilege; no secrets | ✅ PASS | Cache runs with minimal privileges; no secrets needed |
| **VI — Observability** | Metrics/logs exposed | ✅ PASS | Registry logs accessible via `docker logs`; status in `dp dev status` |
| **VII — Quality Gates** | Automated validation | ✅ PASS | Unit tests for all exported functions; idempotency tests |
| **VIII — Pragmatism** | Incremental delivery | ✅ PASS | Single feature; uses standard registry:2 image |
| **IX — Maintainability** | Clear module boundaries | ✅ PASS | New cache.go in sdk/localdev; follows existing patterns |

**Pre-Implementation Gates**:

| Gate | Requirement | Status |
|------|-------------|--------|
| **Workflow Demo** | Cache starts before k3d, images served from cache | ✅ Defined in spec |
| **Contract Schema** | registries.yaml format documented | ✅ k3d standard format |
| **Promotion/Rollback** | Cache volume preserved by default | ✅ Explicit in FR-015 |
| **Observability** | Container labels, docker logs | ✅ Defined in FR-008 |
| **Security/Compliance** | No secrets, minimal privileges | ✅ Read-only proxy to Docker Hub |

## Project Structure

### Documentation (this feature)

```text
specs/007-registry-cache/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (YAML schemas)
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
sdk/localdev/
├── cache.go             # NEW: Registry cache management
├── cache_test.go        # NEW: Unit tests for cache
├── k3d.go               # MODIFY: Add cache integration
└── k3d_test.go          # MODIFY: Add cache integration tests

cli/cmd/
└── dev.go               # MODIFY: Wire cache into dev up/down flow

.cache/                   # RUNTIME GENERATED (gitignored)
├── registry-config.yml   # Registry configuration
└── registries.yaml       # k3d registry mirror config
```

**Structure Decision**: Extends existing `sdk/localdev` package with new `cache.go` file. Follows established pattern of `<feature>.go` + `<feature>_test.go` in the same package. CLI modifications minimal—only wiring changes in existing `dev.go`.

## Complexity Tracking

> No violations identified. Design fits within existing architecture.

| Aspect | Decision | Rationale |
|--------|----------|-----------|
| Single new file | cache.go contains all cache logic | Follows existing package organization |
| Docker CLI exec | Use exec.Command for docker operations | Consistent with k3d.go pattern |
| No new dependencies | Only standard library + gopkg.in/yaml.v3 | Already in go.mod |
