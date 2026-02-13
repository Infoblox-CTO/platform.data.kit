# Contract: Python Plugin Template Files

**Feature**: 010-python-cloudquery-plugins
**Type**: Template file specification

## Overview

These are the template files rendered by `dp init -t cloudquery -l python <name>`. Each file is a Go text/template that receives the template variables defined in data-model.md.

## File List

| Template Path | Output Path | Purpose |
|--------------|-------------|---------|
| `cloudquery/python/dp.yaml.tmpl` | `dp.yaml` | Package manifest |
| `cloudquery/python/main.py.tmpl` | `main.py` | Plugin entry point |
| `cloudquery/python/requirements.txt.tmpl` | `requirements.txt` | pip dependencies |
| `cloudquery/python/pyproject.toml.tmpl` | `pyproject.toml` | Python project metadata |
| `cloudquery/python/plugin/__init__.py.tmpl` | `plugin/__init__.py` | Package init |
| `cloudquery/python/plugin/plugin.py.tmpl` | `plugin/plugin.py` | Plugin class |
| `cloudquery/python/plugin/client.py.tmpl` | `plugin/client.py` | API client |
| `cloudquery/python/plugin/spec.py.tmpl` | `plugin/spec.py` | Config spec |
| `cloudquery/python/plugin/tables/__init__.py.tmpl` | `plugin/tables/__init__.py` | Tables package |
| `cloudquery/python/plugin/tables/example_resource.py.tmpl` | `plugin/tables/example_resource.py` | Sample table |
| `cloudquery/python/tests/test_example_resource.py.tmpl` | `tests/test_example_resource.py` | Unit tests |

## Required Changes

### pyproject.toml.tmpl

**Before** (current):
```toml
requires-python = ">=3.13"
```

**After** (required):
```toml
requires-python = ">=3.12"
```

### All other templates

No changes required. The CloudQuery Python SDK API usage in the templates has been verified against SDK v0.1.52:
- `serve.PluginCommand` ✓
- `plugin.Plugin` ✓
- `schema.Table`, `schema.Column` ✓
- `scheduler.TableResolver`, `scheduler.Scheduler` ✓
- PyArrow column types ✓
- `--address` CLI flag ✓
