# Implementation Plan: Rename CLI from `dp` to `dk` (DataKit) & Add Interactive Banner

**Branch**: `016-rename-cli-dk` | **Date**: 2026-03-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/016-rename-cli-dk/spec.md`

## Summary

Rename the CLI binary and all user-facing references from `dp` to `dk` (DataKit). Update the K8s API group from `dp.io` to `datakit.infoblox.dev`. Add a styled ASCII art banner displayed during interactive prompting sessions. This is a clean break — no backward compatibility or migration path is provided. The project is pre-production and all effort goes toward consistency with currently proposed concepts.

## Technical Context

**Language/Version**: Go (latest stable, per constitution)
**Primary Dependencies**: cobra (CLI framework), charmbracelet/huh (interactive TUI forms), golang.org/x/term (TTY detection), charmbracelet/lipgloss (terminal styling — new dependency for banner)
**Storage**: N/A (no data storage changes)
**Testing**: `go test ./...` across cli/, sdk/, contracts/, platform/controller/ modules
**Target Platform**: Linux (amd64/arm64), macOS (amd64/arm64) — CLI binaries
**Project Type**: Multi-module Go monorepo (existing structure)
**Performance Goals**: Banner render < 100ms, zero impact on non-interactive paths
**Constraints**: ASCII-only banner characters for terminal compatibility; ANSI colors only when TTY supports them
**Scale/Scope**: ~25 Go source files, ~15 markdown files, 5 CRD/gitops YAMLs, 2 demo scripts, Makefile, CI workflow — estimated 400+ individual string replacements

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Article | Status | Notes |
|---------|--------|-------|
| I — Developer Experience | **PASS** | `dk` is shorter than `dp`; same ergonomics. Banner improves DX for interactive flows. |
| II — Specs Define Contracts | **PASS** | K8s API group changes from `dp.io` to `datakit.infoblox.dev`. No backward compat needed per pre-production policy. |
| III — Immutability and Auditability | **PASS** | No artifact immutability changes. |
| IV — Separation of Concerns | **PASS** | Layer boundaries unchanged; only labels/names change. |
| V — Security and Compliance | **PASS** | No security posture changes. |
| VI — Observability | **PASS** | No observability changes. |
| VII — Quality Gates | **PASS** | All existing tests updated; new banner tests added. |
| VIII — Pragmatism | **PASS** | Rename is a single deliverable; banner is incremental. No compat overhead aligns with pragmatism. |
| IX — Maintainability | **PASS** | Module boundaries unchanged. `contracts ← sdk ← cli` preserved. |
| X — Persona Boundaries | **PASS** | No persona boundary changes. |
| XI — Extensions are Contracts | **PASS** | Extension schemas unchanged. |
| **Technology Standards / CLI Name** | **AMENDMENT INCLUDED** | Constitution amended from `dp` to `dk` as part of this feature (constitution v3.0.0). |
| **Versioning / Compat Rules** | **AMENDMENT INCLUDED** | Constitution amended to codify pre-production no-backward-compatibility policy. |

**Gate Result**: PASS. All violations resolved by constitution amendments included in this feature (v2.0.0 → v3.0.0).

## Project Structure

### Documentation (this feature)

```text
specs/016-rename-cli-dk/
├── plan.md              # This file
├── research.md          # Phase 0 output — rename scope analysis
├── data-model.md        # Phase 1 output — rename mapping tables
├── quickstart.md        # Phase 1 output — verification guide
├── contracts/           # Phase 1 output
│   ├── api-group.md     # K8s API group definition (no migration — clean break)
│   └── label-mapping.md # Label domain mapping
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cli/
├── main.go                          # Package comment: dp → dk
├── cmd/
│   ├── root.go                      # Use: "dk", Short/Long descriptions, version cmd
│   ├── init.go                      # Examples, help text, banner call
│   ├── dev.go                       # Examples, status messages
│   ├── run.go                       # Examples
│   ├── config.go                    # Config paths, help text
│   ├── asset*.go                    # Examples
│   ├── pipeline*.go                 # Examples
│   ├── cell.go                      # Examples, CRD references
│   ├── logs.go                      # Examples
│   ├── banner.go                    # NEW: ASCII banner rendering
│   └── ...                          # All other commands
└── internal/
    └── prompt/
        └── prompt.go                # Existing TTY detection (unchanged)

sdk/
├── localdev/
│   ├── k3d.go                       # Cluster/namespace: dp-local → dk-local
│   ├── charts/embed.go              # Helm releases: dp-* → dk-*
│   └── config.go                    # Config paths: .dp/ → .dk/
├── runner/
│   ├── docker.go                    # Image prefix: dp/ → dk/
│   └── *.go                         # Package comments
├── pipeline/executor.go             # Image names: dp-sync → dk-sync
├── lineage/heartbeat.go             # Producer: dp-runner → dk-runner
├── registry/bundler.go              # Builder: dp/ → dk/
└── promotion/
    ├── pr.go                        # Labels: dp.io/ → datakit.infoblox.dev/
    └── kustomize.go                 # Labels

contracts/
├── connector.go                     # Label domain: dp.infoblox.com/ → datakit.infoblox.dev/

platform/controller/
├── api/v1alpha1/groupversion_info.go   # API group: dp.io → datakit.infoblox.dev
├── internal/controller/
│   ├── job.go                       # Labels, controller name
│   └── deployment.go                # Labels
├── cmd/main.go                      # Leader election ID
└── config/deployment.yaml           # RBAC

gitops/
├── base/crds/
│   ├── packagedeployment.yaml       # CRD group
│   ├── cell.yaml                    # CRD group
│   └── store.yaml                   # CRD group
├── base/kustomization.yaml          # Labels
└── argocd/applicationset.yaml       # Labels, group

Makefile                             # Binary names: dp → dk
.github/workflows/ci.yaml           # Artifact names
.specify/memory/constitution.md      # Amended to v3.0.0
```

**Structure Decision**: This is a rename within the existing multi-module Go monorepo. No structural changes needed. All modifications are in-place string replacements across existing files, plus one new file (`cli/cmd/banner.go`).

## Complexity Tracking

No constitution violations remain — all were resolved by amendments included in this feature.
