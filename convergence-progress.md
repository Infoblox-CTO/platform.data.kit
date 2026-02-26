# Convergence Progress: Partitioning Model

Tracking implementation of the partitioning model from `partitioning.md` across cli, sdk, controller, and docs.

**Status: ALL PHASES COMPLETE** ✅

## Target State Summary

- **Package × Cell** deployment model
- **Cell** = cluster-scoped k8s CRD (`kubectl get cells`)
- **Store** = namespaced k8s CRD in cell's namespace (`kubectl get stores -n dp-canary`)
- **AssetRef** gains optional `Cell` field for cross-cell routing
- **dp run --cell** resolves stores from cell namespace instead of package `store/` dir
- **dp build** produces Helm chart tarball (4-step flow)
- **dp publish** pushes Helm chart to OCI registry (4-step flow)
- **dp cell list/show/stores** subcommands for cell discovery

---

## Phase 1: Core Types (contracts + controller) ✅

| Task | Status | Files |
|------|--------|-------|
| Add `Cell` field to `AssetRef` | ✅ DONE | `contracts/transform.go` |
| Update AssetRef tests | ✅ DONE | `contracts/transform_test.go` |
| Add Cell CRD type to controller | ✅ DONE | `platform/controller/api/v1alpha1/cell_types.go` (new) |
| Add Store CRD type to controller | ✅ DONE | `platform/controller/api/v1alpha1/store_types.go` (new) |
| Add `Cell` field to PackageDeploymentSpec | ✅ DONE | `platform/controller/api/v1alpha1/packagedeployment_types.go` |
| DeepCopy methods for Cell/Store | ✅ DONE | `platform/controller/api/v1alpha1/zz_generated.deepcopy.go` |

## Phase 2: SDK — Cell-aware Store Resolution ✅

| Task | Status | Files |
|------|--------|-------|
| Add `Cell` + `KubeContext` to `RunOptions` | ✅ DONE | `sdk/runner/runner.go` |
| Implement CellResolver (kubectl-based) | ✅ DONE | `sdk/runner/cellresolver.go` (new) |
| Update `generateCQConfig` for cell | ✅ DONE | `sdk/runner/cqconfig.go` |
| Wire CellResolver into Docker runner | ✅ DONE | `sdk/runner/docker.go` |
| Cell resolver tests | ✅ DONE | `sdk/runner/cellresolver_test.go` (new) |

## Phase 3: CLI — `--cell` flag + cell subcommands ✅

| Task | Status | Files |
|------|--------|-------|
| Add `--cell` flag to `dp run` | ✅ DONE | `cli/cmd/run.go` |
| Add `--context` flag to `dp run` | ✅ DONE | `cli/cmd/run.go` |
| Add `dp cell list` subcommand | ✅ DONE | `cli/cmd/cell.go` (new) |
| Add `dp cell show` subcommand | ✅ DONE | `cli/cmd/cell.go` |
| Add `dp cell stores` subcommand | ✅ DONE | `cli/cmd/cell.go` |
| Tests for cell commands + flags | ✅ DONE | `cli/cmd/cell_test.go` (new) |

## Phase 4: Helm Chart Generation (dp build + dp publish) ✅

| Task | Status | Files |
|------|--------|-------|
| Helm chart generator | ✅ DONE | `sdk/registry/helmchart.go` (new) |
| Helm chart tests | ✅ DONE | `sdk/registry/helmchart_test.go` (new) |
| Update `dp build` (4-step: validate → git → OCI → Helm) | ✅ DONE | `cli/cmd/build.go` |
| Update `dp publish` (4-step: build → chart → check → push) | ✅ DONE | `cli/cmd/publish.go` |
| Exclude store/ from published chart | ✅ DONE | `sdk/registry/helmchart.go` |
| Helm push via helm CLI | ✅ DONE | `cli/cmd/publish.go` (`pushHelmChart()`) |
| Reuse existing chart in dist/ | ✅ DONE | `cli/cmd/publish.go` (`findHelmChart()`) |

## Phase 5: Documentation ✅

| Task | Status | Files |
|------|--------|-------|
| Add `docs/concepts/cells.md` | ✅ DONE | `docs/concepts/cells.md` (new) |
| Add deploy-to-cell tutorial | ✅ DONE | `docs/tutorials/deploying-to-cells.md` (new) |
| Update CLI reference (--cell, --context, dp cell) | ✅ DONE | `docs/reference/cli.md` |
| Update concepts index | ✅ DONE | `docs/concepts/index.md` |
| Update tutorials index | ✅ DONE | `docs/tutorials/index.md` |
| Update mkdocs.yml nav | ✅ DONE | `mkdocs.yml` |

## Phase 6: Smoke Tests ✅

| Task | Status | Notes |
|------|--------|-------|
| All contracts tests pass | ✅ DONE | `go test ./...` in contracts |
| All SDK tests pass | ✅ DONE | `go test ./...` in sdk (runner, registry, etc.) |
| All CLI tests pass | ✅ DONE | `go test ./...` in cli |
| Controller compiles | ✅ DONE | `go build ./...` in platform/controller |
| `dp init smoke-cq --runtime cloudquery` | ✅ DONE | Scaffolds Transform with connector/, asset/, store/ dirs |
| `dp lint` passes | ✅ DONE | Validates scaffolded package |
| `dp build` produces Helm chart | ✅ DONE | 4-step flow, chart in dist/ |
| `dp publish --dry-run` works | ✅ DONE | Finds existing chart, shows what would be pushed |
| `dp cell --help` shows subcommands | ✅ DONE | list, show, stores subcommands |
| `dp run --help` shows --cell/--context | ✅ DONE | Flags registered and documented |

---

## Files Changed / Created

### New Files
- `platform/controller/api/v1alpha1/cell_types.go` — Cell CRD type (cluster-scoped)
- `platform/controller/api/v1alpha1/store_types.go` — Store CRD type (namespaced)
- `sdk/runner/cellresolver.go` — kubectl-based cell store resolver
- `sdk/runner/cellresolver_test.go` — Cell resolver unit tests
- `sdk/registry/helmchart.go` — Helm chart generator
- `sdk/registry/helmchart_test.go` — Helm chart tests
- `cli/cmd/cell.go` — `dp cell list/show/stores` subcommands
- `cli/cmd/cell_test.go` — Cell command tests
- `docs/concepts/cells.md` — Cells & Stores concept doc
- `docs/tutorials/deploying-to-cells.md` — Deploy-to-cells tutorial

### Modified Files
- `contracts/transform.go` — Added `Cell` field to `AssetRef`
- `contracts/transform_test.go` — Added cell tests
- `platform/controller/api/v1alpha1/packagedeployment_types.go` — Added `Cell` to spec
- `platform/controller/api/v1alpha1/zz_generated.deepcopy.go` — DeepCopy for Cell/Store
- `sdk/runner/runner.go` — Added `Cell`/`KubeContext` to `RunOptions`
- `sdk/runner/cqconfig.go` — Cell-aware store resolution in CQ config generation
- `sdk/runner/docker.go` — Wire CellResolver when --cell is used
- `cli/cmd/run.go` — `--cell`/`--context` flags
- `cli/cmd/build.go` — 4-step build with Helm chart generation
- `cli/cmd/publish.go` — 4-step publish with Helm chart push
- `docs/reference/cli.md` — CLI ref for --cell, --context, dp cell
- `docs/concepts/index.md` — Added Cells card
- `docs/tutorials/index.md` — Added Deploying to Cells card
- `mkdocs.yml` — Navigation entries for cells docs

## Notes

- `dp dev up --cell` requires k3d/compose changes — deferred to future work
- Controller reconcilers for Cell/Store — deferred (types only in this pass)
- Cross-cell fan-out resolution — AssetRef.Cell field added, runtime resolution works in cqconfig.go
- CellResolver uses `kubectl` exec (no k8s client-go dependency in SDK)
