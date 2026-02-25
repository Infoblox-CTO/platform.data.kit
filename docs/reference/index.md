---
title: Reference
description: Complete reference documentation for the DP CLI and configuration
---

# Reference

Complete reference documentation for the `dp` CLI, manifest schemas, and configuration options.

## Reference Guides

<div class="grid" markdown>

<div class="card" markdown>
### :terminal: CLI Commands
Complete reference for all `dp` commands with examples.

[CLI Reference →](cli.md)
</div>

<div class="card" markdown>
### :page_facing_up: Manifest Schema
Full schema reference for dp.yaml and the manifest kinds (Transform, Asset, Connector, Store).

[Manifest Schema →](manifest-schema.md)
</div>

<div class="card" markdown>
### :gear: Configuration
Environment variables and configuration options.

[Configuration →](configuration.md)
</div>

</div>

## Quick Reference

### Common Commands

```bash
# Initialize a new package
dp init <name>

# Local development
dp dev up|down|status

# Validation
dp lint <path>

# Execution
dp run <path>

# Publishing
dp build <path>
dp publish <path>

# Promotion
dp promote <package> <version> --to <environment>

# Observability
dp status <package>
dp logs <package>
```

### Manifest Files

| File | Purpose |
|------|---------|
| `dp.yaml` | Transform manifest — metadata, runtime, inputs, outputs, classification |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DP_REGISTRY` | OCI registry URL | `ghcr.io` |
| `DP_NAMESPACE` | Default namespace | `default` |
| `DP_LOG_LEVEL` | Log verbosity | `info` |
| `DP_OUTPUT_FORMAT` | Output format | `table` |

## Related Documentation

- [Architecture](../architecture.md) - Platform architecture details
- [Testing](../testing.md) - Testing guide for the platform
- [Contributing](../contributing.md) - Contribution guidelines
