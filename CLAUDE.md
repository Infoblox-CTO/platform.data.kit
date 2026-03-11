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
- Manifest kinds: `Transform`, `Source`, `Destination`, `DataSet`, `DataSetGroup`, `Connector`, `Store`.
- The project is pre-release (alpha). Prefer clean deletion over deprecation or backward-compat shims.
