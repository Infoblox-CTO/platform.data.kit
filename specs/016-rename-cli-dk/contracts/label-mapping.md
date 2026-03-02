# Label Domain: `datakit.infoblox.dev`

**Feature**: 016-rename-cli-dk | **Date**: 2026-03-01

## Overview

All DataKit K8s labels use the single canonical domain `datakit.infoblox.dev`. This replaces the previous `dp.io` and `dp.infoblox.com` domains. No backward compatibility is provided.

## Labels

### Controller Labels (applied to K8s workloads)

| Label Key | Used In |
|-----------|---------|
| `datakit.infoblox.dev/package` | Job, Deployment metadata |
| `datakit.infoblox.dev/mode` | Job, Deployment metadata |

### Promotion Labels (applied to GitOps resources)

| Label Key | Used In |
|-----------|---------|
| `datakit.infoblox.dev/package` | PR labels, Kustomization |
| `datakit.infoblox.dev/environment` | PR labels, ArgoCD ApplicationSet |
| `datakit.infoblox.dev/managed-by` | Kustomization base |

### Connector Labels (applied to connector annotations)

| Label Key | Used In |
|-----------|---------|
| `datakit.infoblox.dev/provider` | Connector metadata |
| `datakit.infoblox.dev/channel` | Connector metadata |

## Contract: Label Format

All DataKit labels MUST follow the format:

```
datakit.infoblox.dev/<key>
```

Where `<key>` is a DNS-compatible label name (lowercase, alphanumeric, hyphens allowed).
