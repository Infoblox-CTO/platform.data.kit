---
title: Reference
description: Complete reference documentation for the DK CLI and configuration
---

# Reference

Complete reference documentation for the `dk` CLI, manifest schemas, and configuration options.

## Reference Guides

<div class="grid" markdown>

<div class="card" markdown>
### :terminal: CLI Commands
Complete reference for all `dk` commands with examples.

[CLI Reference →](cli.md)
</div>

<div class="card" markdown>
### :page_facing_up: Manifest Schema
Full schema reference for dk.yaml and the manifest kinds (Transform, Asset, Connector, Store).

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
dk init <name>

# Local development
dk dev up|down|status

# Validation
dk lint <path>

# Execution
dk run <path>

# Publishing
dk build <path>
dk publish <path>

# Promotion
dk promote <package> <version> --to <environment>

# Observability
dk status <package>
dk logs <package>
```

### Manifest Files

| File | Purpose |
|------|---------|
| `dk.yaml` | Transform manifest — metadata, runtime, inputs, outputs, classification |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DK_REGISTRY` | OCI registry URL | `ghcr.io` |
| `DK_NAMESPACE` | Default namespace | `default` |
| `DK_LOG_LEVEL` | Log verbosity | `info` |
| `DK_OUTPUT_FORMAT` | Output format | `table` |

## Related Documentation

- [Architecture](../architecture.md) - Platform architecture details
- [Testing](../testing.md) - Testing guide for the platform
- [Contributing](../contributing.md) - Contribution guidelines
