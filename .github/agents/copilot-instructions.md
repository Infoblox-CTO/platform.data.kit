# data-platform Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-22

## Active Technologies
- Go 1.25 (per go.mod files in all modules) + `testing` (stdlib), `github.com/stretchr/testify` (assertions), Cobra (CLI testing) (002-unit-e2e-tests)
- N/A (tests use temp directories and `testdata/` fixtures) (002-unit-e2e-tests)
- Python 3.11+ (for MkDocs tooling), Markdown (content) + MkDocs 1.5+, mkdocs-material 9.5+ (theme), mkdocs-minify-plugin (optimization) (003-docs-getting-started)
- N/A (static files in `docs/` directory) (003-docs-getting-started)

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
- 003-docs-getting-started: Added Python 3.11+ (for MkDocs tooling), Markdown (content) + MkDocs 1.5+, mkdocs-material 9.5+ (theme), mkdocs-minify-plugin (optimization)
- 002-unit-e2e-tests: Added Go 1.25 (per go.mod files in all modules) + `testing` (stdlib), `github.com/stretchr/testify` (assertions), Cobra (CLI testing)

- 001-cdpp-mvp: Added Go (latest stable per constitution) + Cobra (CLI), client-go (K8s), ORAS (OCI), Flux (GitOps), Dagster (orchestration)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
