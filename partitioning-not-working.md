# Partitioning Model — Implementation Gap Analysis

This document records the gaps between `partitioning.md` (the design spec) and the current codebase, discovered during the end-to-end audit. Each item explains what the spec requires, what exists today, and why it cannot be implemented in this pass.

---

## What IS Implemented

These spec items are fully working and tested:

| Area | Spec Item | Implementation |
|---|---|---|
| **Contracts** | Transform, Connector, Store, DataSet, DataSetGroup types | `contracts/*.go` — all 5 manifest kinds with YAML/JSON round-trip |
| **DataSetRef.Cell** | Optional `cell` field on DataSetRef for cross-cell routing | `contracts/transform.go` — `Cell string \`yaml:"cell,omitempty"\`` |
| **Cross-cell test** | Cell-qualified outputs in full Transform YAML | `contracts/transform_test.go` — `TestTransform_CrossCellOutputs` |
| **Cell CRD type** | Cluster-scoped Cell with Namespace, Labels, Status | `platform/controller/api/v1alpha1/cell_types.go` |
| **Store CRD type** | Namespaced Store with Connector, Connection, Secrets | `platform/controller/api/v1alpha1/store_types.go` |
| **PackageDeployment CRD** | Namespaced with cell, mode, schedule, resources | `platform/controller/api/v1alpha1/packagedeployment_types.go` |
| **CellResolver** | kubectl-based store resolution from dp-\<cell\> namespace | `sdk/runner/cellresolver.go` — ResolveStore, ListStores, CellExists |
| **3-tier store resolution** | (1) assetRef.Cell → per-asset cell, (2) --cell → deployment cell, (3) package store/ fallback | `sdk/runner/cqconfig.go` — resolveStore closure chain |
| **dk run --cell** | `--cell` and `--context` flags on run command | `cli/cmd/run.go` |
| **dk cell list/show/stores** | Cell management CLI with --context support | `cli/cmd/cell.go` |
| **dk build** | 4-step: validate → git → OCI bundle → Helm chart | `cli/cmd/build.go` |
| **dk publish** | Build + helm push to OCI registry | `cli/cmd/publish.go` |
| **Helm chart generation** | Chart.yaml + values.yaml + templates/packagedeployment.yaml + manifests/ (no store/) | `sdk/registry/helmchart.go` |
| **PackageDeployment template** | namespace: dp-\{\{.Values.cell\}\}, correct spec fields | `sdk/registry/helmchart.go` — `generatePackageDeploymentTemplate()` |
| **dk init scaffolding** | connector/, dataset/, store/ directories with templates | `cli/internal/templates/transform/cloudquery/` |
| **Package directory structure** | dk.yaml + connector/ + dataset/ + store/ + src/ convention | Matches spec exactly |
| **GitOps CRDs** | Cell + Store + PackageDeployment CRDs in gitops base | `gitops/base/crds/{cell,store,packagedeployment}.yaml` |

---

## What Is NOT Implemented

### 1. `dk dev up --cell <name>` — Cell Lifecycle in Local Development

**Spec requires** (partitioning.md §Cell Lifecycle, §Infrastructure Sharing):

```bash
dk dev up --cell canary
```

Should:
1. Create k8s namespace `dp-canary`
2. Create database `dp_canary` in shared PostgreSQL
3. Create S3 bucket `dp-canary-raw` in LocalStack
4. Create topic prefix `canary.*` in Redpanda
5. Apply Cell CR and Store CRs to the k3d cluster

**What exists:**
- `cli/cmd/dev.go` — `dk dev up` only has `--compose`, `--runtime`, `--volumes` flags. No `--cell` flag.
- `sdk/localdev/` — Manages k3d clusters and Helm chart deployment. Zero cell awareness (no references to Cell CRs, namespace creation, per-cell databases, buckets, or topic prefixes).
- Shared infrastructure provisioning (PostgreSQL, LocalStack, Redpanda, Marquez) works via Helm chart init jobs, but only for the default setup.

**Why not in this pass:**
This requires significant new infrastructure code across two packages:
- `sdk/localdev/` needs a new `CellProvisioner` that connects to the running k3d cluster's PostgreSQL to create databases, calls LocalStack S3 API to create buckets, creates Kafka topic prefixes via Redpanda admin API, generates Cell/Store CR YAML, and applies them via kubectl.
- `cli/cmd/dev.go` needs the `--cell` flag, cell name validation, and orchestration of the provisioning steps.
- Each infrastructure type (PG, S3, Kafka) needs its own provisioning/teardown logic with error handling, idempotency, and cleanup on failure.
- The provisioning depends on the specific Helm chart configurations (port numbers, credentials) which are managed by `sdk/localdev/charts/`.

**Estimate:** 500-800 lines of new code across 3-4 files. This is a standalone feature that doesn't block any other functionality — `dk run --cell` works against any pre-existing Cell/Store CRs applied manually via kubectl.

---

### 2. `dk promote` — Cell-Based Promotion (vs Environment-Based)

**Spec requires** (partitioning.md §Developer Journey Step 5, §CLI Commands):

```bash
dk promote pg-to-s3 1.2.4-g29aef --to stable    # "stable" is a CELL
```

Should update `cm-repo/apps/pg-to-s3-stable/version.txt` with the new version. The `--to` target is a cell name, not an environment.

**What exists:**
- `cli/cmd/promote.go` — `--to` accepts environments (dev, int, prod) validated via `promotion.Environment`.
- `sdk/promotion/` — Creates GitHub PRs to update Kustomize overlays per environment, not version.txt+values.yaml per cell.
- Hard-coded environment validation: `dev`, `int`, `prod` only.

**Why not in this pass:**
This requires rearchitecting the promotion flow:
- The `promotion.Environment` type and its `Valid()` method constrain targets to dev/int/prod.
- The `Promoter` creates PRs that modify Kustomize overlay files. The spec expects PRs that write `version.txt` + `values.yaml` in a `cm-repo/apps/<package>-<cell>/` directory structure.
- The CM repo path convention changes from `gitops/environments/<env>/` to either `apps/<package>-<cell>/` or `clusters/<cluster>/apps/<package>-<cell>/`.
- This also involves the `dk promote status` subcommand.

**Estimate:** Requires redesign of `sdk/promotion/` package (~300 lines) + updates to `cli/cmd/promote.go` (~50 lines). The current environment-based promotion is functional for the existing gitops structure; migrating requires the gitops structure to change first (see item 8).

---

### 3. `dk rollback` — Cell-Based Rollback

**Spec requires:**

Rollback targeting cells rather than environments. The spec's promotion model (version.txt in CM repo) implies rollback = writing an older version to version.txt.

**What exists:**
- `cli/cmd/rollback.go` — Uses `--environment` flag (dev/int/prod) and delegates to the same `promotion.Promoter` that uses Kustomize overlays.
- Automatic previous-version detection is a TODO.

**Why not in this pass:**
Same architectural dependency as `dk promote` — the rollback flow mirrors promotion and shares the same `sdk/promotion/` package. The rollback would need to read version history from the CM repo (git log on version.txt) to determine the previous version. Blocked on the promote/gitops migration.

---

### 4. `dk status` — Cell-Aware Status Display

**Spec requires** (partitioning.md §CLI Commands):

Status should show package deployments across cells, not environments.

**What exists:**
- `cli/cmd/status.go` — 166 lines of pure placeholder code with hardcoded mock data:
  ```go
  {"dev", "v1.2.0", "2 hours ago", "success", "✓ healthy"},
  {"int", "v1.1.0", "1 day ago", "success", "✓ healthy"},
  {"prod", "v1.0.0", "3 days ago", "success", "✓ healthy"},
  ```
- Columns are ENVIRONMENT | VERSION | LAST RUN | STATUS | HEALTH.
- No actual data retrieval — no kubectl calls, no API queries.

**Why not in this pass:**
The status command needs:
- kubectl queries to list PackageDeployments across all cell namespaces (`kubectl get packagedeployments -A`)
- Parsing of PackageDeployment status/phase/conditions
- Correlation of packages to cells
- Real data from the run service (last run time, health checks)

The prerequisite is a functioning controller that maintains accurate PackageDeployment status (see item 7). Without controller reconciliation, status would show stale or empty data.

---

### 5. `dk show` — Cell/Store Resolution Display

**Spec implies:**

`dk show` should resolve and display store connections when `--cell` is provided, showing where assets would actually connect.

**What exists:**
- `cli/cmd/show.go` — Shows the effective dk.yaml manifest with overrides and resolved asset details (store name, classification). No `--cell` flag, no store resolution from cells, no connection string display.

**Why not in this pass:**
Adding a `--cell` flag to `dk show` and using the CellResolver to display resolved store connections is technically feasible (~100 lines). However, it creates a UX question: should `dk show --cell canary` actually call kubectl, or should it remain a local-only manifest preview? The spec doesn't explicitly define this command's behavior with cells. Deferring to avoid premature UX decisions.

**Note:** This is the most implementable of the deferred items. If cell-resolved preview becomes a priority, it can be added by importing `sdk/runner.CellResolver` into the show command and displaying resolved store connections alongside the manifest.

---

### 6. Controller Reconcilers — Cell and Store

**Spec requires** (partitioning.md §Concepts):

Cell and Store are k8s Custom Resources, implying controllers that reconcile their state.

**What exists:**
- `platform/controller/api/v1alpha1/cell_types.go` — Cell type definition. **No controller.**
- `platform/controller/api/v1alpha1/store_types.go` — Store type definition. **No controller.**
- `platform/controller/internal/controller/` — Only contains `packagedeployment_controller.go`, `job.go`, `deployment.go`.

**Why not in this pass:**

**Cell Controller** would need to:
- Watch Cell resources
- Create/ensure the cell's namespace exists
- Count Store CRs in the namespace and update `status.storeCount`
- Count PackageDeployments targeting this cell and update `status.packageCount`
- Set `status.ready` based on namespace existence and readiness checks
- Handle Cell deletion (garbage collect namespace? just update status?)

**Store Controller** would need to:
- Watch Store resources
- Optionally verify the connection (ping the database, check S3 bucket, etc.)
- Set `status.ready` based on connection verification
- Handle credential resolution from Kubernetes Secrets

These are non-trivial operator implementations requiring:
- Controller-runtime setup with watches, predicates, and RBAC
- Cross-resource indexing (Cell → namespace → Stores)
- Connection health checking with timeouts and retries
- Proper finalizer and deletion handling
- Comprehensive unit and integration tests

**Estimate:** 400-600 lines per controller + tests. The type definitions are sufficient for the current workflow since CellResolver queries Stores via kubectl directly.

---

### 7. PackageDeployment Controller — TODO Stubs

**Spec requires:**

A controller that pulls OCI packages, verifies digests, and creates Jobs/Deployments.

**What exists:**
- `platform/controller/internal/controller/packagedeployment_controller.go` — 318 lines. The reconciliation loop handles phases (Pending → Pulling → Ready → Running → Failed). However:
  - `handlePulling`: "TODO: Implement actual package pulling from OCI registry. For MVP, we simulate successful pull."
  - Digest verification: "TODO: Implement digest verification"
  - `handleReady`: Job/Deployment objects are generated but not applied — `_ = job` (assigned to blank identifier)
  - Cron parsing: returns hardcoded 1-hour duration

**Why not in this pass:**

Completing the controller requires:
- **OCI pulling**: Import an OCI client library (e.g., `oras-go`) to pull Helm chart tarballs from the registry, extract manifests, and store them in a ConfigMap or volume for the runner container.
- **Digest verification**: SHA-256 comparison of pulled artifact against `spec.package.digest`.
- **Job/Deployment creation**: Replace `_ = job` with `r.Create(ctx, job)` + ownership references + status tracking.
- **Cron scheduling**: Parse cron expressions and create a CronJob or use a timer-based requeue.
- **Status management**: Update last-run timestamps, success/failure tracking, retry logic.

This is the core controller logic and represents the most complex piece of the system. Estimate: 300-500 lines of changes plus integration tests requiring a test k8s environment (envtest).

---

### 8. GitOps ApplicationSet — Dynamic Git Generator

**Spec requires** (partitioning.md §Cells Across Clusters):

```yaml
generators:
  - git:
      repoURL: https://github.com/Infoblox-CTO/platform.data.cm.git
      revision: main
      directories:
        - path: clusters/dp-staging/apps/*
```

ArgoCD discovers deployments dynamically from `cm-repo/apps/<package>-<cell>/` directories, each containing `version.txt` + `values.yaml`.

**What exists:**
- `gitops/argocd/applicationset.yaml` — Uses a static `list` generator with hardcoded environments (dev, int, prod). Points to `gitops/environments/{{environment}}` in the same repo.
- `gitops/environments/` — Kustomize overlays per environment (dev, int, prod).

**Why not in this pass:**

Migrating requires:
- A separate CM repo (or repo structure) with the `apps/<package>-<cell>/` convention
- Replacing the `list` generator with a `git` directory generator
- ArgoCD must be configured to pull Helm charts from OCI registries (plugin or native OCI source)
- Each application needs to render a Helm chart from OCI with cell-specific values, which requires either ArgoCD's Helm OCI support or a custom plugin
- Multi-cluster support requires ArgoCD cluster registration and per-cluster ApplicationSets

This is an infrastructure/GitOps architecture change, not a code change. The current environment-based approach is functional for single-cluster deployments. Migration to the cell-based git generator pattern should be done alongside the promote/rollback migration (items 2-3) to maintain consistency.

---

### 9. `sdk/localdev` Package — Cell Awareness

**Spec requires:**

The localdev package should understand cells, creating per-cell infrastructure within the shared dev stack.

**What exists:**
- `sdk/localdev/` — Contains `cache.go`, `compose.go`, `config.go`, `k3d.go`, `portforward.go`, `ports.go`, `prerequisites.go`, `runtime.go`. Zero references to Cell, Store, or cell-related concepts.

**Why not in this pass:**
This is the backend for item 1 (`dk dev up --cell`). All the same reasons apply — it requires infrastructure provisioning code for PostgreSQL databases, S3 buckets, Kafka topics, and k8s resource application. This is the implementation layer that the CLI command would call.

---

## Summary

| # | Gap | Severity | Blocked By | Estimate |
|---|---|---|---|---|
| 1 | `dk dev up --cell` | Medium | Nothing — standalone feature | 500-800 LOC |
| 2 | `dk promote` cell targets | Medium | GitOps migration (#8) | 350 LOC |
| 3 | `dk rollback` cell targets | Low | Promote migration (#2) | 100 LOC |
| 4 | `dk status` cell display | Low | Controller (#7) for real data | 200 LOC |
| 5 | `dk show --cell` preview | Low | Nothing — standalone, smallest gap | 100 LOC |
| 6 | Cell/Store controllers | Medium | Nothing — standalone | 800-1200 LOC |
| 7 | PackageDeployment TODOs | High | OCI library, envtest setup | 300-500 LOC |
| 8 | GitOps ApplicationSet | Medium | External infrastructure decisions | Config change |
| 9 | `sdk/localdev` cells | Medium | Same as #1 | Included in #1 |

### Recommended Implementation Order

1. **`dk show --cell`** (#5) — smallest gap, high developer value, no dependencies
2. **`dk dev up --cell`** (#1 + #9) — enables local cell workflows, unblocks developer iteration
3. **Cell/Store controllers** (#6) — enables real status data
4. **PackageDeployment controller TODOs** (#7) — completes the deployment loop
5. **GitOps + promote + rollback** (#8, #2, #3) — migration to cell-based gitops, done together
6. **`dk status`** (#4) — useful only after controller produces real status data

### What Works Today Without These Gaps

The core developer workflow is complete:
- `dk init` → scaffolds a package with connector/, dataset/, store/
- `dk run` → runs locally using package store/ fallback
- `dk run --cell canary` → resolves stores from cell's k8s namespace (if Cell/Store CRs are applied manually)
- `dk build` → validates and produces Helm chart
- `dk publish` → pushes Helm chart to OCI registry
- `dk cell list/show/stores` → discovers cells and stores via kubectl
- Cross-cell routing via `DataSetRef.Cell` is supported in the resolution chain

The gaps are in **lifecycle automation** (cell provisioning, promotion, rollback) and **controller logic** (reconciliation, status). The data model, resolution chain, and CLI commands for the core workflow are solid.
