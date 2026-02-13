# Implementation Plan: Plugin Registry & Configuration Management

**Branch**: `009-plugin-registry` | **Date**: 2026-02-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/009-plugin-registry/spec.md`

## Summary

Replace the sparse-clone-and-build flow for CloudQuery destination plugins with OCI image pulls from `ghcr.io/infobloxopen/`. Add hierarchical YAML configuration (repo → user → system) so developers can set persistent defaults for registries, plugin versions, and mirrors. Provide a `dp config` subcommand for managing settings without manual YAML editing. Zero new dependencies — uses existing `gopkg.in/yaml.v3` and shells out to `docker` and `git` (both already required).

## Technical Context

**Language/Version**: Go 1.25 (all three modules: cli, sdk, contracts)
**Primary Dependencies**: cobra (CLI), gopkg.in/yaml.v3 (config), os/exec (docker pull, git, k3d, kubectl)
**Storage**: YAML config files at three scopes — `.dp/config.yaml` (repo), `~/.config/dp/config.yaml` (user), `/etc/datakit/config.yaml` (system)
**Testing**: `go test`, table-driven tests, no external test framework. Existing pattern: direct cobra RunE calls, temp dirs, inline dp.yaml. 150 existing CLI tests.
**Target Platform**: macOS/Linux CLI, k3d local Kubernetes cluster
**Project Type**: Go monorepo with `go.work` — three modules (cli, sdk, contracts)
**Performance Goals**: First image pull <60s on broadband; cached sync startup <30s
**Constraints**: Zero new dependencies; backward compatible with existing `~/.config/dp/config.yaml` (used by `dp dev`); Docker must be running
**Scale/Scope**: 3 destination plugins initially (postgresql, s3, file); ~10 config keys; 5 new CLI subcommands (`config set/get/unset/list`, `config add-mirror/remove-mirror`)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Article | Status | Notes |
|---------|--------|-------|
| **I — DX First** | ✅ PASS | `dp config` commands are intuitive; hierarchical defaults minimize flags; `dp run --sync` is the happy path |
| **II — Contracts Stable** | ✅ PASS | Config schema is additive (new `plugins` section); existing `dev` section unchanged; backward compatible |
| **III — Immutability** | ✅ PASS | Plugin images are tagged with semver versions (`ghcr.io/.../cloudquery-plugin-<name>:<version>`); no mutable `:latest` in defaults |
| **IV — Separation** | ✅ PASS | Config is separate from pipeline definitions; plugin registry is infrastructure configuration, not pipeline logic |
| **V — Security** | ✅ PASS | No secrets in config files (registry URLs only); Docker handles auth via `docker login`; least privilege default (public GHCR images) |
| **VI — Observability** | ✅ PASS | CLI logs which registry/mirror was used for each pull; `dp config list` shows effective configuration with sources |
| **VII — Quality Gates** | ✅ PASS | Config validation at write time (FR-016); `Validate()` method catches errors before they reach runtime |
| **VIII — Pragmatism** | ✅ PASS | Zero new dependencies; defers private registry auth to Docker's existing mechanism; incremental delivery (P1–P5) |
| **IX — Maintainability** | ✅ PASS | Config logic in `sdk/localdev/` (shared); CLI commands in `cli/cmd/`; clear dependency direction `contracts ← sdk ← cli` |

### Pre-Implementation Gates

| Gate | Status | Evidence |
|------|--------|----------|
| **Workflow Demo** | ✅ | `dp run . --sync` demonstrates end-to-end: config load → image pull → k3d import → pod deploy → sync |
| **Contract Schema** | ✅ | Config YAML schema defined in `contracts/config-schema.yaml`; validation strategy explicit (struct `Validate()` method) |
| **Promotion/Rollback** | N/A | CLI config feature — does not affect deployments or artifact contracts |
| **Observability** | ✅ | CLI prints which registry was used, which mirror (if any), and config source for each setting |
| **Security/Compliance** | ✅ | No secrets in config; registry URLs only; Docker handles authentication; no PII |

## Project Structure

### Documentation (this feature)

```text
specs/009-plugin-registry/
├── plan.md              # This file
├── research.md          # Phase 0: resolved unknowns
├── data-model.md        # Phase 1: config entities and relationships
├── quickstart.md        # Phase 1: developer quickstart guide
├── contracts/           # Phase 1: config schema and CLI contract
│   ├── config-schema.yaml
│   └── dp-config-cli.md
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
sdk/localdev/
├── config.go            # MODIFY: extend Config struct with Plugins section;
│                        #   add hierarchical LoadHierarchicalConfig();
│                        #   add Validate() method; add gitRepoRoot() helper
├── config_test.go       # MODIFY: add tests for hierarchical merge, validation,
│                        #   git root detection, scope precedence
└── constants.go         # existing constants (DefaultClusterName, etc.)

cli/cmd/
├── config.go            # NEW: dp config command (set, get, unset, list,
│                        #   add-mirror, remove-mirror)
├── config_test.go       # NEW: tests for all config subcommands
├── run.go               # MODIFY: replace ensureDestinationPlugin (sparse clone)
│                        #   with pullDestinationImage (docker pull + k3d import);
│                        #   deploy destination as pod; update sync config to use
│                        #   registry: grpc for both source and destination
├── run_test.go          # MODIFY: update sync/destination tests for new OCI flow
└── root.go              # MODIFY: add configCmd to rootCmd

docs/reference/
├── configuration.md     # MODIFY: add plugins section documentation,
│                        #   update config file locations and precedence
└── cli.md               # MODIFY: add dp config subcommand reference
```

**Structure Decision**: Extends the existing monorepo structure. Config logic stays in `sdk/localdev/` (the shared config package). New CLI commands go in `cli/cmd/` following the existing cobra pattern (package-level vars, init() registration). No new packages or modules needed.

## Complexity Tracking

> No constitution violations — no complexity justifications needed.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
