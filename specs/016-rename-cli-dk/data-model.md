# Data Model: Rename CLI from `dp` to `dk` (DataKit)

**Feature**: 016-rename-cli-dk | **Date**: 2026-03-01

## Overview

This feature does not introduce new data entities. It renames existing identifiers across multiple layers. This document serves as the canonical rename mapping table — the authoritative reference for all string replacements.

## Rename Mapping Tables

### 1. CLI Identity

| Entity | Current Value | New Value | Files |
|--------|--------------|-----------|-------|
| Binary name | `dp` | `dk` | Makefile, CI |
| Cobra root command | `Use: "dp"` | `Use: "dk"` | cli/cmd/root.go |
| CLI short description | `DP - Data Platform CLI` | `DK - DataKit CLI` | cli/cmd/root.go |
| CLI long description | `DP (Data Platform) is a...` | `DK (DataKit) is a...` | cli/cmd/root.go |
| Version output | `dp version %s` | `dk version %s` | cli/cmd/root.go |
| Package comment (main) | `entry point for the dp CLI` | `entry point for the dk CLI` | cli/main.go |
| Package comment (cmd) | `CLI commands for dp` | `CLI commands for dk` | cli/cmd/root.go, config.go |

### 2. Manifest Filename

| Entity | Current Value | New Value | Impact |
|--------|--------------|-----------|--------|
| Package manifest | `dp.yaml` | `dk.yaml` | ~30 Go files, all examples, all docs |

### 3. K8s API Group & Labels

| Entity | Current Value | New Value | Files |
|--------|--------------|-----------|-------|
| API group | `dp.io` | `datakit.infoblox.dev` | CRDs, controller, promoter, gitops |
| Package label | `dp.io/package` | `datakit.infoblox.dev/package` | controller, promoter |
| Mode label | `dp.io/mode` | `datakit.infoblox.dev/mode` | controller |
| Environment label | `dp.io/environment` | `datakit.infoblox.dev/environment` | promoter, gitops |
| Managed-by label | `dp.io/managed-by` | `datakit.infoblox.dev/managed-by` | gitops |
| Connector labels | `dp.infoblox.com/provider` | `datakit.infoblox.dev/provider` | contracts |
| Connector labels | `dp.infoblox.com/channel` | `datakit.infoblox.dev/channel` | contracts |
| CRD name (PackageDeployment) | `packagedeployments.dp.io` | `packagedeployments.datakit.infoblox.dev` | gitops CRD |
| CRD name (Cell) | `cells.dp.io` | `cells.datakit.infoblox.dev` | gitops CRD |
| CRD name (Store) | `stores.dp.io` | `stores.datakit.infoblox.dev` | gitops CRD |

### 4. Infrastructure Identifiers

| Entity | Current Value | New Value | Files |
|--------|--------------|-----------|-------|
| k3d cluster name | `dp-local` | `dk-local` | sdk/localdev/k3d.go |
| k3d namespace | `dp-local` | `dk-local` | sdk/localdev/k3d.go |
| Redpanda release | `dp-redpanda` | `dk-redpanda` | sdk/localdev/charts/embed.go |
| LocalStack release | `dp-localstack` | `dk-localstack` | sdk/localdev/charts/embed.go |
| PostgreSQL release | `dp-postgres` | `dk-postgres` | sdk/localdev/charts/embed.go |
| Marquez release | `dp-marquez` | `dk-marquez` | sdk/localdev/charts/embed.go |
| Marquez web service | `dp-marquez-web` | `dk-marquez-web` | sdk/localdev/charts/embed.go |
| Controller name | `dp-controller` | `dk-controller` | platform/controller |
| Leader election ID | `dp-controller.dp.io` | `dk-controller.datakit.infoblox.dev` | platform/controller/cmd/main.go |
| Runner producer | `dp-runner` | `dk-runner` | sdk/lineage/heartbeat.go |

### 5. Docker Image Identifiers

| Entity | Current Value | New Value | Files |
|--------|--------------|-----------|-------|
| Image prefix | `dp/` | `dk/` | sdk/runner/docker.go, sdk/registry/bundler.go |
| Sync image | `dp-sync:latest` | `dk-sync:latest` | sdk/pipeline/executor.go |
| Transform image | `dp-transform:latest` | `dk-transform:latest` | sdk/pipeline/executor.go |
| Test image | `dp-test:latest` | `dk-test:latest` | sdk/pipeline/executor.go |
| Dockerfile comment | `# DP Pipeline Image` | `# DK Pipeline Image` | sdk/runner/docker.go |

### 6. Config Paths

| Entity | Current Value | New Value | Files |
|--------|--------------|-----------|-------|
| Repo config dir | `.dp/config.yaml` | `.dk/config.yaml` | sdk/localdev/config.go, cli/cmd/config.go |
| User config dir | `~/.config/dp/config.yaml` | `~/.config/dk/config.yaml` | sdk/localdev/config.go, cli/cmd/config.go |
| System config | `/etc/datakit/config.yaml` | `/etc/datakit/config.yaml` (unchanged) | — |

### 7. Build Artifacts

| Entity | Current Value | New Value | Files |
|--------|--------------|-----------|-------|
| Local binary | `bin/dp` | `bin/dk` | Makefile |
| Install target | `$(GOPATH)/bin/dp` | `$(GOPATH)/bin/dk` | Makefile |
| Linux amd64 | `bin/dp-linux-amd64` | `bin/dk-linux-amd64` | Makefile |
| Linux arm64 | `bin/dp-linux-arm64` | `bin/dk-linux-arm64` | Makefile |
| Darwin amd64 | `bin/dp-darwin-amd64` | `bin/dk-darwin-amd64` | Makefile |
| Darwin arm64 | `bin/dp-darwin-arm64` | `bin/dk-darwin-arm64` | Makefile |
| CI artifact | `cdpp` | `dk` | .github/workflows/ci.yaml |

### 8. New Entity: ASCII Banner

| Property | Value |
|----------|-------|
| Location | `cli/cmd/banner.go` (new file) |
| Content | ASCII art displaying "DataKit" branding |
| Style | charmbracelet/lipgloss with blue/cyan ANSI colors |
| Trigger | Called from interactive prompt paths when `prompt.IsInteractive() == true` |
| Suppression | Non-TTY, piped input, redirected output, terminal width < 40 |
| Fallback | Plain text "DataKit" when colors not supported |

## State Transitions

No state transitions — this feature is a pure rename with no behavioral changes to data flow or entity lifecycle.

## Validation Rules

- After rename, `grep -r '"dp "' cli/ sdk/ contracts/ platform/` must return zero matches (excluding `dp.yaml` if that file still exists in git history)
- After rename, `grep -r 'dp\.io' platform/ gitops/ sdk/promotion/` must return zero matches
- All `go test ./...` must pass in each module
- `dk version` must output `dk version <semver>`
- `dk --help` must not contain any `dp` references
