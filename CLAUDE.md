# CLAUDE.md — Project Instructions for Claude Code

## Project Overview

DataKit (`dk`) is a CLI and SDK for building, testing, and deploying data pipelines.
Go 1.25 multi-module monorepo (`cli/`, `sdk/`, `contracts/`, `platform/controller/`, `tests/e2e/`).

## Off-Limits Directories

- **`specs/`** — Managed exclusively by GitHub Spec Kit agents. **Never read, modify, create, or delete files in this directory** unless you are an agent specifically designated to work with specs.

## Commit Messages

**NEVER add any Claude/Anthropic attribution to commit messages.** No `Co-Authored-By: Claude`, no `Generated with Claude Code`, nothing referencing Claude or Anthropic. Commit messages should only describe the change.

## Key Conventions

- Pipeline = reactive dependency graph derived from Transform/DataSet manifests (`dk.yaml`). There is no `pipeline.yaml` manifest.
- The only CLI pipeline command is `dk pipeline show` (graph visualization).
- Manifest kinds: `Transform`, `DataSet`, `DataSetGroup`, `Connector`, `Store`.
- The project is pre-release (alpha). Prefer clean deletion over deprecation or backward-compat shims.

## dk CLI Reference

**Before working with dk manifests or CLI commands, run `dk docs -o llm` to get the full
structured reference** (every command, flag, manifest schema, error code, and enum value
in a single YAML document). This is faster and more reliable than reading source files.

### Workflow

```
init → dev up → run → lint → test → build → publish → promote
```

### Quick Reference

```bash
# Full CLI reference (structured YAML — preferred for LLM consumption)
dk docs -o llm

# Scaffold a project and transform
dk project init <name>
dk init <transform> --runtime cloudquery --mode batch --namespace <ns> --team <team>

# Validate
dk lint <package-dir>                    # validate dk.yaml
dk lint --scan-dir .                     # validate entire project
dk dataset validate <path> --offline     # validate dataset manifests
dk pipeline show --scan-dir .            # view dependency graph

# Local development
dk doctor                                # check environment prerequisites
dk dev up                                # start local infra (postgres, s3, kafka, etc.)
dk dev seed <package-dir>                # seed sample data into local stores
dk run <package-dir>                     # execute locally via Docker
dk run --dry-run                         # validate without executing
dk test <package-dir>                    # run tests

# Build & deploy
dk build <package-dir>                   # build OCI artifact
dk build --dry-run                       # validate build without producing artifact
dk publish <package-dir>                 # push OCI artifact to registry
dk promote <name> <version> --to <env>              # promote to env (default cell c0)
dk promote <name> <version> --to <env> --cell <c>  # promote to specific cell in env

# dbt (transparent wrapper — resolves stores, generates profiles.yml automatically)
dk dbt run                              # build dbt models
dk dbt test                             # run dbt tests
dk dbt debug                            # verify connection
dk dbt run --select my_model            # pass-through args to dbt
dk dbt run --cell canary                # resolve stores from a cell
```

### Manifest Kinds

| Kind | Description | Owner |
|------|-------------|-------|
| `Transform` | Unit of computation (dk.yaml) | data engineer |
| `DataSet` | Data contract with schema | data engineer |
| `DataSetGroup` | Bundles multiple DataSets from one materialisation | data engineer |
| `Store` | Named infra instance with connection details + secrets | infra owner |
| `Connector` | Storage technology type (postgres, s3, kafka) | platform team |

### Runtimes & Modes

- **Runtimes:** `cloudquery`, `generic-go`, `generic-python`, `dbt`
- **Modes:** `batch` (default), `streaming` (not supported by dbt)
- Generic runtimes (`generic-go`, `generic-python`, `dbt`) require `spec.image`

### Common Validation Error Codes

| Code | Meaning |
|------|---------|
| E001 | Name not DNS-safe |
| E040 | spec.runtime is required |
| E041 | spec.image required for generic-* runtimes |
| E220 | spec.store required for DataSet |
| E230 | spec.inputs must be non-empty |
| E231 | spec.outputs must be non-empty |

Run `dk docs -o llm` for the complete error code table.
