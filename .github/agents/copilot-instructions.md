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
- 007-registry-cache: Added Go 1.25 (matches existing codebase per go.mod) + Docker CLI (exec-based), k3d CLI, registry:2 image
- 005-k3d-local-dev: Added Go 1.25 (per go.work and .tool-versions) + cobra (CLI), k3d CLI (exec), kubectl CLI (exec), embed (Go stdlib for manifests)
- 004-consolidate-manifest: Added Go 1.25 (per go.work) + github.com/spf13/cobra (CLI), gopkg.in/yaml.v3 (parsing)


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
