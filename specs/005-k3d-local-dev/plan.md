# Implementation Plan: k3d Local Development Environment

**Branch**: `005-k3d-local-dev` | **Date**: January 25, 2026 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-k3d-local-dev/spec.md`

## Summary

Add k3d-based local Kubernetes development environment support to the `dp dev` commands. This introduces a `--runtime` flag accepting `compose` (default) or `k3d`, creates a `K3dManager` in the SDK that mirrors the existing `ComposeManager` interface, and deploys Redpanda, LocalStack, and PostgreSQL services with automatic port forwarding. Kubernetes manifests are embedded in the CLI binary for location-independent execution.

## Technical Context

**Language/Version**: Go 1.25 (per go.work and .tool-versions)  
**Primary Dependencies**: cobra (CLI), k3d CLI (exec), kubectl CLI (exec), embed (Go stdlib for manifests)  
**Storage**: N/A (k3d manages volumes internally)  
**Testing**: go test with table-driven tests, mocks for exec commands  
**Target Platform**: macOS, Linux, Windows WSL2 (k3d supported platforms)
**Project Type**: Multi-module Go workspace (cli, sdk, contracts, platform/controller)  
**Performance Goals**: Cluster startup < 3 min (cold), < 30 sec (warm); Port forwards stable 8+ hours  
**Constraints**: Docker must be running; k3d and kubectl must be installed for k3d runtime  
**Scale/Scope**: Single local cluster per developer workstation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Gate | Status | Notes |
|------|--------|-------|
| **Article I - Developer Experience** | ✅ PASS | Happy path: `dp dev up --runtime=k3d` with clear output |
| **Article II - Stable Contracts** | ✅ PASS | New `RuntimeManager` interface; existing `ComposeManager` unchanged |
| **Article III - Immutability** | ✅ N/A | Local dev only; no production artifacts |
| **Article IV - Separation of Concerns** | ✅ PASS | k3d abstraction separate from pipeline definitions |
| **Article V - Security** | ✅ PASS | Local dev only; no secrets in manifests |
| **Article VI - Observability** | ✅ PASS | Status command shows pod states and health |
| **Article VII - Quality Gates** | ✅ PASS | Unit tests required; prerequisite checks before operations |
| **Article VIII - Pragmatism** | ✅ PASS | Uses k3d/kubectl exec; no custom scheduler |
| **Article IX - Maintainability** | ✅ PASS | Clean module boundaries: cli → sdk/localdev |
| **CLI Name** | ✅ PASS | Uses `dp` command name |
| **Unit Testing** | ✅ PASS | All new code requires unit tests |

| Pre-Implementation Gate | Status | Notes |
|------------------------|--------|-------|
| **Workflow Demo** | ✅ | `dp dev up/down/status --runtime=k3d` workflow |
| **Contract Schema** | ✅ | RuntimeManager interface is the contract |
| **Promotion/Rollback** | N/A | Local dev feature only |
| **Observability** | ✅ | Pod status, health checks, port forward status |
| **Security/Compliance** | ✅ | Local dev; no secrets handling required |

## Project Structure

### Documentation (this feature)

```text
specs/005-k3d-local-dev/
├── plan.md              # This file
├── research.md          # Phase 0 output - k3d best practices
├── data-model.md        # Phase 1 output - entity definitions
├── quickstart.md        # Phase 1 output - usage guide
├── contracts/           # Phase 1 output - interface definitions
│   └── runtime_manager.go
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
sdk/
└── localdev/
    ├── compose.go           # Existing ComposeManager
    ├── compose_test.go      # Existing tests
    ├── k3d.go               # NEW: K3dManager implementation
    ├── k3d_test.go          # NEW: K3dManager tests
    ├── runtime.go           # NEW: RuntimeManager interface
    ├── runtime_test.go      # NEW: Runtime selection tests
    ├── portforward.go       # NEW: Port forwarding management
    ├── portforward_test.go  # NEW: Port forward tests
    └── manifests/           # NEW: Embedded K8s manifests
        ├── embed.go         # go:embed directives
        ├── redpanda.yaml
        ├── localstack.yaml
        └── postgres.yaml

cli/
└── cmd/
    ├── dev.go               # MODIFY: Add --runtime flag
    └── dev_test.go          # MODIFY: Add runtime tests

hack/
└── k3d/                     # NEW: Reference manifests (for testing)
    ├── redpanda.yaml
    ├── localstack.yaml
    └── postgres.yaml
```

**Structure Decision**: Extend existing `sdk/localdev` package with new k3d functionality. Manifests embedded in SDK for portability. CLI modifications minimal (flag addition and runtime selection).

## Complexity Tracking

No constitution violations requiring justification. Implementation follows established patterns.
