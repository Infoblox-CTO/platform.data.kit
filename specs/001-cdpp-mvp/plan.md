# Implementation Plan: CDPP MVP

**Branch**: `001-cdpp-mvp` | **Date**: 2026-01-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-cdpp-mvp/spec.md`

## Summary

Build a Kubernetes-native data pipeline platform enabling teams to contribute reusable, versioned "data packages" with a complete developer workflow: bootstrap → local run → validate → publish → promote. The MVP proves end-to-end value with a single example pipeline (Kafka → transform → S3) demonstrating immutable versioning, GitOps promotion, observability, and governance metadata (PII tagging + lineage).

**Technical Approach**: Go monorepo with independent modules (`contracts`, `sdk`, `cli`, `platform/controller`), OCI artifacts for packages, Flux-based GitOps for promotion, Dagster for DAG orchestration, Prometheus/Grafana for observability.

## Technical Context

**Language/Version**: Go (latest stable per constitution)  
**Primary Dependencies**: Cobra (CLI), client-go (K8s), ORAS (OCI), Flux (GitOps), Dagster (orchestration)  
**Storage**: OCI registry (package artifacts), PostgreSQL (catalog metadata), S3 (data artifacts)  
**Testing**: go test, testcontainers-go for integration tests  
**Target Platform**: Linux containers on Kubernetes (EKS initial target)  
**Project Type**: Monorepo with multiple Go modules  
**Performance Goals**: CLI commands complete in <5s for local ops; pipeline scheduling latency <30s  
**Constraints**: Packages must be <500MB OCI artifacts; local dev stack must run on 8GB RAM  
**Scale/Scope**: MVP supports 10-50 packages, 1-3 environments; post-MVP scales to 500+ packages

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Article | Requirement | Status | Evidence |
|---------|-------------|--------|----------|
| **I. Developer Experience** | Happy path: bootstrap → local run → validate → publish → promote | ✅ PASS | CLI commands cover full workflow; 30-min time-to-first-pipeline target |
| **II. Stable Contracts** | Machine-readable schemas, additive evolution | ✅ PASS | JSON Schema for dp.yaml, pipeline.yaml; versioned with SemVer |
| **III. Immutability** | Immutable artifacts, auditable promotions, easy rollback | ✅ PASS | OCI immutability; PR-based promotions; rollback = pin previous version |
| **IV. Separation of Concerns** | Infra vs pipelines separated via bindings | ✅ PASS | Bindings contract abstracts infrastructure; no hardcoded identifiers |
| **V. Security by Default** | Least privilege, no committed secrets, PII metadata | ✅ PASS | Secrets via K8s/external-secrets; PII tags required in manifest |
| **VI. Observability** | Metrics, structured logs, dashboards | ✅ PASS | Prometheus metrics, correlation IDs, Grafana dashboards |
| **VII. Quality Gates** | Contract validation, tests before publish/promote | ✅ PASS | `dp lint` and `dp test` gates; CI enforcement |
| **VIII. Pragmatism** | MVP end-to-end value; defer advanced features | ✅ PASS | Single example pipeline; marketplace/multi-tenancy deferred |
| **IX. Maintainability** | Clear module boundaries, dependency direction | ✅ PASS | contracts ← sdk ← cli; contracts ← platform/controller |
| **Technology Standards** | Go latest stable | ✅ PASS | go.mod specifies latest stable Go version |

### Pre-Implementation Gates

| Gate | Status | Evidence |
|------|--------|----------|
| **Workflow Demo** | ✅ | Plan includes end-to-end developer workflow in milestones |
| **Contract Schema** | ✅ | dp.yaml and pipeline.yaml schemas defined in data-model.md |
| **Promotion/Rollback** | ✅ | GitOps PR workflow with pinned versions; rollback = version change |
| **Observability** | ✅ | Prometheus metrics + Grafana dashboards in Milestone 2 |
| **Security/Compliance** | ✅ | PII metadata required; secrets via K8s; audit via Git history |

## Project Structure

### Documentation (this feature)

```text
specs/001-cdpp-mvp/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (OpenAPI/JSON Schema)
│   ├── dp-manifest.schema.json
│   ├── pipeline-manifest.schema.json
│   ├── bindings.schema.json
│   └── lineage-event.schema.json
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
# Go Monorepo with Independent Modules
contracts/                    # contracts/vX.Y.Z
├── go.mod
├── manifest/
│   ├── dp.go                 # DataPackage manifest types
│   ├── pipeline.go           # Pipeline manifest types
│   └── bindings.go           # Bindings types
├── artifact/
│   ├── ref.go                # Artifact reference types
│   └── lineage.go            # Lineage event types
├── validation/
│   └── schema.go             # JSON Schema validation
└── testdata/
    └── *.yaml                # Example manifests for tests

sdk/                          # sdk/vX.Y.Z
├── go.mod
├── registry/
│   ├── client.go             # OCI registry client
│   └── publish.go            # Package publish logic
├── runtime/
│   ├── executor.go           # Local execution runtime
│   └── dagster.go            # Dagster integration
├── gitops/
│   ├── promote.go            # Promotion PR generation
│   └── environments.go       # Environment resolution
└── lineage/
    └── emitter.go            # OpenLineage event emission

cli/                          # cli/vX.Y.Z
├── go.mod
├── main.go
├── cmd/
│   ├── root.go
│   ├── init.go               # dp init
│   ├── dev.go                # dp dev (start local stack)
│   ├── run.go                # dp run (execute locally)
│   ├── lint.go               # dp lint (validate)
│   ├── test.go               # dp test
│   ├── build.go              # dp build
│   ├── publish.go            # dp publish
│   └── promote.go            # dp promote
└── internal/
    ├── output/               # Structured CLI output
    └── config/               # CLI configuration

platform/
├── controller/               # platform/controller/vX.Y.Z
│   ├── go.mod
│   ├── main.go
│   ├── controllers/
│   │   ├── package_controller.go
│   │   └── run_controller.go
│   ├── api/
│   │   └── v1alpha1/         # CRD types
│   └── webhooks/
│       └── validation.go
└── operator/                 # Helm chart for platform
    ├── Chart.yaml
    └── templates/

gitops/                       # Environment definitions (separate repo in prod)
├── environments/
│   ├── dev/
│   │   ├── kustomization.yaml
│   │   ├── bindings.yaml
│   │   └── packages/
│   │       └── example-pipeline.yaml  # Pinned version
│   ├── integration/
│   └── prod/
└── flux-system/

examples/                     # Reference packages
├── hello-pipeline/           # Minimal example
│   ├── dp.yaml
│   ├── pipeline.yaml
│   ├── src/
│   └── Dockerfile
└── kafka-s3-pipeline/        # Full MVP example
    ├── dp.yaml
    ├── pipeline.yaml
    ├── src/
    ├── bindings.local.yaml
    └── docker-compose.yaml

hack/                         # Development utilities
├── local-stack/
│   ├── docker-compose.yaml   # LocalStack, Kafka, MinIO, Postgres
│   └── seed-data.sh
└── kind/
    └── cluster.yaml          # Kind cluster for parity testing

dashboards/                   # Grafana dashboards
├── platform-overview.json
└── package-runs.json
```

**Structure Decision**: Go monorepo with independent module versioning. Each module (contracts, sdk, cli, platform/controller) has its own go.mod and release tags (e.g., `contracts/v0.1.0`). This enables:
- Independent versioning per module
- Clear dependency direction (contracts ← sdk ← cli)
- Separate release cadence for platform vs CLI

## Complexity Tracking

No constitution violations requiring justification. The multi-module monorepo is the prescribed architecture per Article IX.

---

## Module Architecture

### Dependency Graph

```
┌─────────────────────────────────────────────────────────────────┐
│                         External Users                          │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                        CLI                                │   │
│  │                    cli/vX.Y.Z                            │   │
│  │  Commands: init, dev, run, lint, test, build,           │   │
│  │            publish, promote, status, logs                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                        SDK                                │   │
│  │                    sdk/vX.Y.Z                            │   │
│  │  Packages: registry, runtime, gitops, lineage           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                     Contracts                             │   │
│  │                  contracts/vX.Y.Z                        │   │
│  │  Packages: manifest, artifact, validation                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              ▲                                   │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                Platform Controller                        │   │
│  │              platform/controller/vX.Y.Z                  │   │
│  │  CRDs: DataPackage, PackageRun                          │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Module Responsibilities

| Module | Purpose | Key Exports |
|--------|---------|-------------|
| **contracts** | Shared types and validation | `manifest.DataPackage`, `artifact.Contract`, `validation.Validate()` |
| **sdk** | Business logic and integrations | `registry.Client`, `runtime.Executor`, `gitops.Promote()`, `lineage.Emit()` |
| **cli** | User-facing commands | Binary: `dp` |
| **platform/controller** | Kubernetes operator | CRDs, reconciliation loops |

### Versioning Strategy

| Module | Tag Format | Compatibility |
|--------|------------|---------------|
| contracts | `contracts/v1.0.0` | Breaking changes = major bump |
| sdk | `sdk/v1.0.0` | Tracks contracts major version |
| cli | `cli/v1.0.0` | User-facing; may evolve faster |
| platform/controller | `platform/controller/v1.0.0` | Matches CRD apiVersion |

---

## CLI Command Specifications

### `dp init`

Bootstrap a new package skeleton.

```
Usage: dp init <name> [flags]

Arguments:
  name          Package name (DNS-safe, lowercase)

Flags:
  --type        Package type: pipeline, infra, report (default: pipeline)
  --template    Template to use (default: basic)
  --dir         Output directory (default: ./<name>)
  -o, --output  Output format: table, json, yaml (default: table)

Examples:
  dp init my-pipeline
  dp init my-infra --type infra
  dp init my-pipeline --template kafka-s3

Exit Codes:
  0  Success
  1  Invalid arguments
  2  Directory already exists
```

### `dp dev`

Manage local development stack.

```
Usage: dp dev [command] [flags]

Commands:
  (default)     Start local stack
  stop          Stop local stack
  status        Show stack status
  logs          View stack logs

Flags:
  --compose     Custom docker-compose file
  --detach      Run in background (default: true)
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Success
  1  Docker not available
  2  Compose file not found
  3  Stack failed to start
```

### `dp run`

Execute pipeline locally.

```
Usage: dp run [flags]

Flags:
  --bindings    Bindings file (default: bindings.local.yaml)
  --build       Rebuild image before run (default: true)
  --watch       Watch for changes and re-run
  --timeout     Run timeout (default: 1h)
  --env         Additional environment variables (KEY=VALUE)
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Pipeline succeeded
  1  Pipeline failed
  2  Build failed
  3  Bindings validation failed
  4  Timeout exceeded
```

### `dp lint`

Validate package manifests and contracts.

```
Usage: dp lint [flags]

Flags:
  --strict      Fail on warnings
  --fix         Auto-fix fixable issues
  -o, --output  Output format: table, json, yaml

Checks:
  - dp.yaml schema validation
  - pipeline.yaml schema validation
  - Schema file references exist
  - Classification metadata present
  - Binding references consistent

Exit Codes:
  0  All checks passed
  1  Validation errors found
  2  Manifest file not found
```

### `dp test`

Run package tests.

```
Usage: dp test [flags]

Flags:
  --coverage    Generate coverage report
  --timeout     Test timeout (default: 10m)
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  All tests passed
  1  Tests failed
  2  No tests found
```

### `dp build`

Build release artifact.

```
Usage: dp build [flags]

Flags:
  --version     Version tag (required)
  --registry    Target registry (default: from config)
  --platform    Target platform (default: linux/amd64)
  --push        Push to registry after build
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Build succeeded
  1  Build failed
  2  Invalid version format
  3  Lint failed (runs lint first)
```

### `dp publish`

Publish artifact to registry.

```
Usage: dp publish [flags]

Flags:
  --version     Version to publish (required)
  --registry    Target registry (default: from config)
  --force       Skip confirmation (for CI)
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Publish succeeded
  1  Publish failed
  2  Version already exists (immutability)
  3  Authentication failed
  4  Registry unreachable
```

### `dp promote`

Promote package to environment.

```
Usage: dp promote <environment> [flags]

Arguments:
  environment   Target environment: dev, integration, staging, prod

Flags:
  --version     Version to promote (required)
  --package     Package name (default: current directory)
  --dry-run     Show changes without creating PR
  --auto-merge  Enable auto-merge after approval
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Promotion PR created
  1  Environment not found
  2  Version not found
  3  Bindings validation failed
  4  Git operation failed
```

### `dp status`

View package deployment status.

```
Usage: dp status <package> [flags]

Arguments:
  package       Package name

Flags:
  --env         Environment filter
  --runs        Number of recent runs (default: 5)
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Success
  1  Package not found
```

### `dp logs`

View run logs.

```
Usage: dp logs <run-id> [flags]

Arguments:
  run-id        Run identifier

Flags:
  --follow      Stream logs
  --tail        Lines to show (default: 100)
  --since       Show logs since timestamp
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Success
  1  Run not found
```

### `dp rollback`

Rollback to previous version (alias for promote with previous version).

```
Usage: dp rollback <environment> [flags]

Arguments:
  environment   Target environment

Flags:
  --version     Version to rollback to (default: previous deployed)
  --package     Package name
  -o, --output  Output format: table, json, yaml

Exit Codes:
  0  Rollback PR created
  1  No previous version found
```

---

## MVP Milestone Plan

### Milestone 0: Hello World + Local Run (Week 1-2)

**Goal**: Prove the developer bootstrap and local execution loop.

**Deliverables**:
- [ ] `contracts/` module with `DataPackage` and `Pipeline` types
- [ ] JSON Schema validation for dp.yaml and pipeline.yaml
- [ ] `cli/` module with `init`, `dev`, `run`, `lint` commands
- [ ] `hello-pipeline` example package
- [ ] `hack/local-stack/docker-compose.yaml` with LocalStack + Redpanda

**Success Criteria**:
- `dp init hello && cd hello && dp dev && dp run` works end-to-end
- `dp lint` validates manifest correctly
- Time from empty directory to running pipeline < 15 minutes

**Constitution Gates**:
- ✓ Article I: Developer experience demonstrated
- ✓ Article VII: Lint validates contracts

---

### Milestone 1: Publish + Deploy to Dev (Week 3-4)

**Goal**: Prove immutable artifact publishing and GitOps deployment.

**Deliverables**:
- [ ] `sdk/registry/` OCI artifact publishing with oras-go
- [ ] `cli/` commands: `build`, `publish`, `promote`
- [ ] `platform/controller/` Kubernetes controller (basic)
- [ ] CRDs: `DataPackage`, `PackageRun`
- [ ] `gitops/` environment structure with ArgoCD manifests
- [ ] Bindings validation before deployment
- [ ] GitHub Actions workflow for promotion PR

**Success Criteria**:
- `dp publish --version v0.1.0` creates immutable OCI artifact
- `dp promote dev --version v0.1.0` creates PR to gitops repo
- ArgoCD deploys package to dev cluster
- Republish same version fails with immutability error

**Constitution Gates**:
- ✓ Article II: Contracts versioned in OCI
- ✓ Article III: Immutable artifacts, auditable promotions
- ✓ Article IV: Bindings separate from pipeline

---

### Milestone 2: Observability + Rollback (Week 5-6)

**Goal**: Prove operational visibility and recovery capabilities.

**Deliverables**:
- [ ] Prometheus metrics in controller (`cdpp_pipeline_runs_total`, etc.)
- [ ] Structured logging with slog + correlation IDs
- [ ] `dashboards/` Grafana dashboards (platform + package level)
- [ ] `cli/` commands: `status`, `logs`, `rollback`
- [ ] Run history retention in controller

**Success Criteria**:
- Dashboard shows run counts, success rates, durations
- Failed runs visible with error details and log links
- `dp rollback dev --version v0.0.9` promotes previous version
- Rollback completes in < 5 minutes

**Constitution Gates**:
- ✓ Article VI: Observability metrics and dashboards
- ✓ Article III: Rollback demonstrated

---

### Milestone 3: Lineage + PII + Catalog (Week 7-8)

**Goal**: Prove governance metadata flows through the system.

**Deliverables**:
- [ ] `sdk/lineage/` OpenLineage event emitter
- [ ] Marquez integration for lineage backend
- [ ] PII classification validation in lint
- [ ] Catalog record creation on artifact publish
- [ ] `kafka-s3-pipeline` full example with governance

**Success Criteria**:
- Pipeline run emits OpenLineage events viewable in Marquez
- PII tags visible in Marquez dataset facets
- Catalog shows produced artifacts with classification
- End-to-end Kafka → transform → S3 demonstrated

**Constitution Gates**:
- ✓ Article V: PII metadata required and visible
- ✓ Definition of Done: Observable, documented, tested

---

### Milestone Timeline

```
Week 1-2: M0 - Hello World + Local Run
  ├── contracts/ types + validation
  ├── cli/ init, dev, run, lint
  ├── hello-pipeline example
  └── local stack docker-compose

Week 3-4: M1 - Publish + Deploy
  ├── sdk/registry/ OCI publishing
  ├── cli/ build, publish, promote
  ├── platform/controller/ basic
  ├── gitops/ environment structure
  └── GitHub Actions promotion

Week 5-6: M2 - Observability + Rollback
  ├── Prometheus metrics
  ├── Structured logging
  ├── Grafana dashboards
  ├── cli/ status, logs, rollback
  └── Run history retention

Week 7-8: M3 - Lineage + PII + Catalog
  ├── sdk/lineage/ OpenLineage
  ├── Marquez integration
  ├── PII validation
  ├── Catalog records
  └── kafka-s3-pipeline example
```

---

## Release/Versioning Strategy

### Module Versioning

All modules follow SemVer (MAJOR.MINOR.PATCH):

| Module | Initial Version | Tag |
|--------|-----------------|-----|
| contracts | v0.1.0 | `contracts/v0.1.0` |
| sdk | v0.1.0 | `sdk/v0.1.0` |
| cli | v0.1.0 | `cli/v0.1.0` |
| platform/controller | v0.1.0 | `platform/controller/v0.1.0` |

### Compatibility Policy

**contracts/**:
- MAJOR: Breaking changes to manifest schemas
- MINOR: New optional fields, new artifact types
- PATCH: Documentation, validation fixes

**sdk/**:
- Must track contracts major version
- MINOR: New integrations, performance improvements
- PATCH: Bug fixes

**cli/**:
- May evolve independently for UX improvements
- MAJOR: Breaking changes to command syntax
- MINOR: New commands, new flags
- PATCH: Bug fixes

**platform/controller/**:
- Matches CRD apiVersion lifecycle
- MAJOR: CRD breaking changes (requires migration)
- MINOR: New CRD features
- PATCH: Bug fixes

### Contributed Package Versioning

Packages published by contributors follow SemVer:
- Published versions are immutable
- Rollback = promote previous version
- Breaking changes require major version bump

---

## Risks, Tradeoffs, and Deferred Items

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Dagster Pipes complexity | Medium | Medium | Start with simple K8s Job; Dagster optional initially |
| ArgoCD learning curve | Low | Medium | Provide complete gitops/ examples |
| OCI registry limitations | Low | High | Test with multiple registries (ECR, GHCR, Harbor) |
| Schema evolution conflicts | Medium | High | Additive-only policy; deprecation periods |

### Tradeoffs Made

| Decision | Tradeoff | Rationale |
|----------|----------|-----------|
| Dagster over Airflow | Python ecosystem dependency | Better asset model, Pipes for external jobs |
| ArgoCD over Flux | Heavier deployment | Better UI, Kargo integration for future |
| Single gitops repo | All environments visible | Simpler promotion; can split later |
| Kustomize over Helm | Less abstraction | Transparent diffing for reviews |
| slog over zap | Slightly less performant | Stdlib, no dependency, good enough |

### Deferred (Post-MVP)

| Item | Reason | Target |
|------|--------|--------|
| **Kargo automation** | MVP uses manual GitHub Actions | v0.2.0 |
| **Multi-tenancy** | Not needed for MVP workflow | v0.3.0 |
| **Spark operator integration** | Kafka+S3 sufficient for MVP | v0.2.0 |
| **Databricks connector** | Phased per requirements | v0.3.0 |
| **Artifact signing/SBOM** | Fast-follow security milestone | v0.2.0 |
| **Access marketplace** | Beyond MVP scope | v0.4.0 |
| **Full catalog UI** | Marquez provides basic UI | v0.3.0 |
| **Alerting rules** | Dashboards first | v0.2.0 |
| **Dependency orchestration** | Basic ordering only in MVP | v0.3.0 |

---

## Related Documents

- [spec.md](spec.md) - Feature specification
- [research.md](research.md) - Technical research and decisions
- [data-model.md](data-model.md) - Entity definitions and schemas
- [quickstart.md](quickstart.md) - Developer quickstart guide
- [contracts/](contracts/) - JSON Schema definitions
