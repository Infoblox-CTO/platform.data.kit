# Research: Rename CLI from `dp` to `dk` (DataKit)

**Feature**: 016-rename-cli-dk | **Date**: 2026-03-01

## Research Tasks & Findings

### R1: Scope of `dp` → `dk` rename in CLI source code

**Decision**: Rename all user-facing string references from `dp` to `dk` in ~25 Go files across `cli/cmd/`, keeping Go package paths and module names unchanged.

**Rationale**: The Go module path (`github.com/Infoblox-CTO/platform.data.kit/cli`) does not contain `dp`, so no module path changes are needed. Only user-visible strings (command `Use:` fields, example text, help messages, print statements) need updating.

**Alternatives considered**:
- Rename Go modules too → Rejected: modules don't contain `dp`; unnecessary churn and import path breaks across all consumers.
- Create a `dk` wrapper binary that delegates to `dp` → Rejected: adds complexity; clean break is simpler and spec calls for a direct rename.

**Files affected** (CLI cmd layer):
- `cli/cmd/root.go` — `Use: "dp"` → `Use: "dk"`, Short/Long descriptions, version output
- `cli/cmd/init.go` — ~10 example strings, help output strings
- `cli/cmd/dev.go` — ~7 example strings, status messages
- `cli/cmd/run.go` — ~10 example strings
- `cli/cmd/config.go` — ~15 references (config paths, examples, help text)
- `cli/cmd/asset.go`, `asset_create.go`, `asset_list.go`, `asset_show.go` — ~8 examples
- `cli/cmd/pipeline.go`, `pipeline_create.go`, `pipeline_run.go`, `pipeline_show.go`, `pipeline_backfill.go` — ~12 examples
- `cli/cmd/cell.go` — ~2 references
- `cli/cmd/logs.go` — ~5 examples
- `cli/cmd/dev_seed.go` — ~6 examples
- `cli/main.go` — package comment

---

### R2: K8s API group migration from `dp.io` to `datakit.infoblox.dev`

**Decision**: Change the K8s API group from `dp.io` to `datakit.infoblox.dev` across all CRD definitions, controller code, labels, and gitops manifests. Per user's direction: K8s types use `datakit.infoblox.dev`.

**Rationale**: `dp.io` is a placeholder domain not owned by Infoblox. Kubernetes best practices require API groups to use domains the organization owns. `datakit.infoblox.dev` is the correct corporate domain for the DataKit product.

**Alternatives considered**:
- Keep `dp.io` and only rename the CLI binary → Rejected: inconsistent branding and uses an unowned domain.
- Use `dk.infoblox.dev` → Rejected: user explicitly specified `datakit.infoblox.dev`.
- Use `datakit.infoblox.com` instead of `.dev` → Rejected: user specified `.dev`.

**Files affected**:
- `platform/controller/api/v1alpha1/groupversion_info.go` — `+groupName=dp.io`, `Group: "dp.io"`
- `platform/controller/internal/controller/packagedeployment_controller.go` — RBAC annotations
- `platform/controller/internal/controller/job.go` — labels `dp.io/package`, `dp.io/mode`
- `platform/controller/internal/controller/deployment.go` — labels
- `platform/controller/cmd/main.go` — `LeaderElectionID: "dp-controller.dp.io"` → `"dk-controller.datakit.infoblox.dev"`
- `platform/controller/config/deployment.yaml` — RBAC group
- `gitops/base/crds/packagedeployment.yaml` — `group: dp.io`
- `gitops/base/crds/cell.yaml` — `group: dp.io`
- `gitops/base/crds/store.yaml` — `group: dp.io`
- `gitops/base/kustomization.yaml` — `dp.io/managed-by`
- `gitops/argocd/applicationset.yaml` — `dp.io/environment`, `group: dp.io`
- `sdk/promotion/pr.go` — `dp.io/package`, `dp.io/environment`
- `sdk/promotion/kustomize.go` — `dp.io/package`
- `contracts/connector.go` — `dp.infoblox.com/provider` → `datakit.infoblox.dev/provider`

**No migration path**: This is a clean break. The project is pre-production — no backward compatibility is provided per constitution policy.

---

### R3: Infrastructure naming conventions (`dp-local`, `dp-*` Helm releases)

**Decision**: Rename all infrastructure identifiers from `dp-*` to `dk-*`.

**Rationale**: Consistency with the CLI rename. Infrastructure names should match the product branding.

**Alternatives considered**: None — consistency with the rename is the only option. Dev environments are ephemeral and easily recreated.

**Mapping**:
| Current | New |
|---------|-----|
| `dp-local` (cluster name) | `dk-local` |
| `dp-local` (namespace) | `dk-local` |
| `dp-redpanda` | `dk-redpanda` |
| `dp-localstack` | `dk-localstack` |
| `dp-postgres` | `dk-postgres` |
| `dp-marquez` | `dk-marquez` |
| `dp-controller` | `dk-controller` |
| `dp-runner` | `dk-runner` |
| `dp-sync:latest` | `dk-sync:latest` |
| `dp-transform:latest` | `dk-transform:latest` |
| `dp-test:latest` | `dk-test:latest` |
| `dp/` (image prefix) | `dk/` |

---

### R4: Config file paths

**Decision**: Rename config paths from `.dp/` to `.dk/` and `~/.config/dp/` to `~/.config/dk/`.

**Rationale**: Config paths should match the CLI name for discoverability. The system-level path `/etc/datakit/config.yaml` already uses "datakit" and needs no change.

**Mapping**:
| Current | New |
|---------|-----|
| `{git-root}/.dp/config.yaml` | `{git-root}/.dk/config.yaml` |
| `~/.config/dp/config.yaml` | `~/.config/dk/config.yaml` |
| `/etc/datakit/config.yaml` | `/etc/datakit/config.yaml` (unchanged) |

---

### R5: Manifest filename (`dp.yaml`)

**Decision**: Rename `dp.yaml` to `dk.yaml`.

**Rationale**: The manifest filename should match the CLI/product name. `dk.yaml` is consistent with the DataKit branding. This affects ~30 Go files plus all examples and documentation.

**Alternatives considered**:
- Use `datakit.yaml` → Possible but longer; `dk.yaml` matches the CLI shorthand convention.

---

### R6: ASCII banner best practices for Go CLIs

**Decision**: Use `charmbracelet/lipgloss` for styled banner rendering with ANSI color support, leveraging the existing `charmbracelet/huh` dependency chain (which already depends on lipgloss).

**Rationale**: The project already uses `charmbracelet/huh` for interactive forms, which transitively depends on `lipgloss`. Adding lipgloss as a direct dependency adds no new dependency tree weight. lipgloss provides terminal-width detection, automatic color degradation for non-color terminals, and styled string rendering.

**Alternatives considered**:
- Plain `fmt.Println` with hardcoded ANSI codes → Rejected: no automatic color degradation; doesn't detect terminal capabilities.
- `fatih/color` package → Rejected: adds a new dependency tree when lipgloss is already available.
- `pterm` package → Rejected: heavier dependency; lipgloss is already in the dep tree.

**Banner design**:
- Use ASCII characters only (no Unicode box-drawing) for maximum terminal compatibility
- Display "DataKit" in a stylized block-letter banner
- Use blue/cyan ANSI colors when supported, graceful plain-text fallback
- Detect terminal width via lipgloss; skip banner entirely if < 40 columns
- Only call banner from interactive prompt paths (currently only `dk init` interactive mode)

---

### R7: TTY detection infrastructure

**Decision**: Use existing `cli/internal/prompt/prompt.go` which already provides `IsInteractive()` via `golang.org/x/term`. No new TTY detection code needed.

**Rationale**: The infrastructure already exists and works correctly. The banner display decision should call `prompt.IsInteractive()` and also check `lipgloss.HasDarkBackground()` or similar for color support.

---

### R8: CI artifact naming

**Decision**: CI currently builds as `cdpp` (not `dp`). This should be updated to `dk`.

**Rationale**: The CI binary name should match the CLI name. `cdpp` appears to be a legacy/alternate name.

**Files affected**: `.github/workflows/ci.yaml` — build command, artifact name, artifact path.

---

### R9: Constitution amendment scope

**Decision**: Amend `.specify/memory/constitution.md` with three changes:
1. CLI Name standard: `dp` → `dk`, add `datakit.infoblox.dev` API group and label domain requirements
2. Versioning and Compatibility Rules: Add pre-production exemption — no backward compatibility until production release
3. MVP Guardrails: Add explicit no-backward-compat rule, update `dp` → `dk` references
4. Title and purpose: `DP Constitution` → `DK Constitution`

**Rationale**: The constitution is a living governance document. The CLI rename and pre-production compatibility policy are intentional product decisions. Version bumped from 2.0.0 → 3.0.0 (backward-incompatible principle changes).

---

### R10: Label domain consolidation

**Decision**: Consolidate all label domains to `datakit.infoblox.dev`:
- `dp.io/*` → `datakit.infoblox.dev/*`
- `dp.infoblox.com/*` → `datakit.infoblox.dev/*`

**Rationale**: Currently two different label domains are used (`dp.io` in controllers/gitops, `dp.infoblox.com` in connectors). Both should be consolidated to the single canonical domain `datakit.infoblox.dev`.

---

## Unresolved Items

None — all NEEDS CLARIFICATION items from the Technical Context have been resolved through codebase research and user input (K8s types use `datakit.infoblox.dev`).
