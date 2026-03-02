# data-platform Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-22

## Active Technologies
- Go 1.25 (per go.mod files in all modules) + `testing` (stdlib), `github.com/stretchr/testify` (assertions), Cobra (CLI testing) (002-unit-e2e-tests)
- N/A (tests use temp directories and `testdata/` fixtures) (002-unit-e2e-tests)
- Python 3.11+ (for MkDocs tooling), Markdown (content) + MkDocs 1.5+, mkdocs-material 9.5+ (theme), mkdocs-minify-plugin (optimization) (003-docs-getting-started)
- N/A (static files in `docs/` directory) (003-docs-getting-started)
- Go 1.25 (per go.work) + github.com/spf13/cobra (CLI), gopkg.in/yaml.v3 (parsing) (004-consolidate-manifest)
- N/A (file-based manifests) (004-consolidate-manifest)
- Go 1.25 (per go.work and .tool-versions) + cobra (CLI), k3d CLI (exec), kubectl CLI (exec), embed (Go stdlib for manifests) (005-k3d-local-dev)
- N/A (k3d manages volumes internally) (005-k3d-local-dev)
- Go 1.25 (matches existing codebase per go.mod) + Docker CLI (exec-based), k3d CLI, registry:2 image (007-registry-cache)
- Docker volume `dev_registry_cache` for cached image layers (007-registry-cache)
- Go 1.25 (CLI + SDK), Python 3.13+ and Go 1.25 (generated plugin targets) + Cobra CLI framework, testify (CLI/SDK tests); cloudquery-plugin-sdk/pyarrow/pytest (Python plugins); github.com/cloudquery/plugin-sdk/v4 (Go plugins) (008-cloudquery-plugins)
- N/A (CloudQuery syncs to external destinations; local dev uses PostgreSQL from dk dev) (008-cloudquery-plugins)
- Go 1.25 (all three modules: cli, sdk, contracts) + cobra (CLI), gopkg.in/yaml.v3 (config), os/exec (docker pull, git, k3d, kubectl) (009-plugin-registry)
- YAML config files at three scopes — `.dk/config.yaml` (repo), `~/.config/dk/config.yaml` (user), `/etc/datakit/config.yaml` (system) (009-plugin-registry)
- Go 1.25 (multi-module monorepo: `cli/`, `sdk/`, `contracts/`) + `gopkg.in/yaml.v3` (parsing), `github.com/santhosh-tekuri/jsonschema/v6` (JSON Schema validation), `oras.land/oras-go/v2` (OCI registry), `github.com/spf13/cobra` (CLI) (011-asset-instances)
- Local filesystem (`assets/` directory tree); OCI registry for extension schema resolution (011-asset-instances)
- Go 1.25 (all modules) + github.com/spf13/cobra v1.8.1 (CLI), gopkg.in/yaml.v3 v3.0.1 (serialization), github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 (JSON Schema validation), oras.land/oras-go/v2 v2.5.0 (OCI registry) (012-pipeline-workflows)
- Filesystem — `pipeline.yaml` and `schedule.yaml` as YAML files in project root, assets under `assets/` directory (012-pipeline-workflows)
- Bash (runner script) + Go (latest stable, per go.mod — test infrastructure) + bash (runner), asciinema (optional, for recording), existing E2E test helpers (`tests/e2e/helpers.go`) (014-demo-recordings)
- N/A (plain-text dialog files, `.cast` recording artifacts) (014-demo-recordings)
- Go (latest stable, per constitution) + cobra (CLI framework), charmbracelet/huh (interactive TUI forms), golang.org/x/term (TTY detection), charmbracelet/lipgloss (terminal styling — new dependency for banner) (016-rename-cli-dk)
- N/A (no data storage changes) (016-rename-cli-dk)

- Go (latest stable per constitution) + Cobra (CLI), client-go (K8s), ORAS (OCI), Flux (GitOps), Dagster (orchestration) (001-cdpp-mvp)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go (latest stable per constitution)

## Code Style

Go (latest stable per constitution): Follow standard conventions

## Recent Changes
- 016-rename-cli-dk: Added Go (latest stable, per constitution) + cobra (CLI framework), charmbracelet/huh (interactive TUI forms), golang.org/x/term (TTY detection), charmbracelet/lipgloss (terminal styling — new dependency for banner)
- 016-rename-cli-dk: Added Go (latest stable, per constitution) + cobra (CLI framework), charmbracelet/huh (interactive TUI forms), golang.org/x/term (TTY detection), charmbracelet/lipgloss (terminal styling — new dependency for banner)
- 014-demo-recordings: Added Bash (runner script) + Go (latest stable, per go.mod — test infrastructure) + bash (runner), asciinema (optional, for recording), existing E2E test helpers (`tests/e2e/helpers.go`)


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
