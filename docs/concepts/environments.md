---
title: Environments
description: Understanding deployment environments and the promotion workflow
---

# Environments & Cells

DataKit uses a cell-based deployment model. Environments (dev, int, prod) are logical groupings of cells. Promotions target **cells** — named infrastructure instances where packages run.

## Cell-Based Model

```
┌─────────────────────────────────────────────────────────────────┐
│                     Promotion Pipeline                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌───────────┐    ┌───────────┐    ┌───────────┐              │
│   │    dev    │───▶│    int    │───▶│   prod    │              │
│   │  (c0)    │    │  (c0)    │    │  (c0)    │              │
│   └───────────┘    └───────────┘    └───────────┘              │
│                                                                 │
│   Layout: gitops/envs/{env}/cells/{cell}/apps/{pkg}/values.yaml │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Standard Environments

| Environment | Purpose | Approval | Infrastructure |
|-------------|---------|----------|----------------|
| **dev** | Development and testing | Auto-merge | Shared dev cluster |
| **int** | Integration testing | Team approval | Staging cluster |
| **prod** | Production workloads | Multi-party approval | Production cluster |

### dev (Development)

- **Purpose**: Rapid iteration and testing
- **Deployment**: Auto-merge after CI passes
- **Data**: Synthetic or sampled data
- **Access**: All developers

```bash
dk promote my-package v1.0.0 --to dev
# Creates PR updating gitops/envs/dev/cells/c0/apps/my-package/values.yaml
```

### int (Integration)

- **Purpose**: Integration testing with realistic data
- **Deployment**: Requires team lead approval
- **Data**: Production-like (anonymized)
- **Access**: Development team

```bash
dk promote my-package v1.0.0 --to int
# Creates PR, requires 1 approval
```

### prod (Production)

- **Purpose**: Production workloads
- **Deployment**: Requires multiple approvals
- **Data**: Real production data
- **Access**: Limited, audited

```bash
dk promote my-package v1.0.0 --to prod
# Creates PR, requires security + team approval
```

## Promotion Workflow

### 1. Build and Publish

First, build and publish your package:

```bash
cd my-package
dk build --version v1.0.0
dk publish
```

### 2. Promote to a Cell

Request promotion to a cell:

```bash
dk promote my-package v1.0.0 --to dev
```

This creates a GitOps PR that updates the cell's values.yaml:

```
PR: Promote my-package to dev/c0: v1.0.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Changes:
  gitops/envs/dev/cells/c0/apps/my-package/values.yaml

+appVersion: v1.0.0
```

### 3. Approval and Merge

```
dev    → Auto-merge after CI
int    → 1 approval required
prod   → 2 approvals (including security)
```

### 4. ArgoCD Sync

After merge, ArgoCD syncs the changes:

```
┌──────────────────────────────────────────────────────────────┐
│ ArgoCD Sync                                                  │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Git Repository ────▶ ArgoCD ────▶ Kubernetes Cluster       │
│                                                              │
│  1. Git generator discovers gitops/envs/dev/cells/c0/apps/my-package │
│  2. Renders dk-app chart with appVersion from values.yaml     │
│  3. Applies PackageDeployment to dk-c0 namespace              │
│  4. Controller deploys the package                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Cell Stores

Packages reference Stores by name. Each cell provides its own Store instances with cell-specific connection details:

### Package Manifest

```yaml title="dk.yaml"
spec:
  store: events-store  # References a Store by name
```

### Cell-Specific Stores

```yaml title="gitops/envs/dev/cells/c0/stores/events-store.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: events-store
  namespace: dk-c0
spec:
  connector: kafka
  connection:
    brokers: dev-kafka:9092
    topic: user-events-dev
```

```yaml title="gitops/envs/prod/cells/c0/stores/events-store.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: events-store
  namespace: dk-c0
spec:
  connector: kafka
  connection:
    brokers: prod-kafka:9092
    topic: user-events
```

Same package, different infrastructure per cell.

## Checking Status

### Package Status

View your package across environments:

```bash
dk status my-package
```

```
Package: my-package
━━━━━━━━━━━━━━━━━━━

Environment  Version   Status    Last Sync        Last Run
───────────  ───────   ──────    ─────────        ────────
dev          v1.0.0    Synced    10 min ago       5 min ago (success)
int          v0.9.0    Synced    2 days ago       1 day ago (success)
prod         v0.9.0    Synced    2 days ago       6 hours ago (success)
```

### Environment Status

View all packages in an environment:

```bash
dk status --env dev
```

```
Environment: dev
━━━━━━━━━━━━━━━━

Package              Version   Status    Health
───────────          ───────   ──────    ──────
my-package           v1.0.0    Synced    Healthy
analytics-pipeline   v2.1.0    Synced    Healthy
data-loader          v1.5.0    Synced    Degraded ⚠
ml-training          v3.0.0    Synced    Healthy
```

## Rollback

If a deployment causes issues, rollback to a previous version:

```bash
dk rollback my-package --to dev --to-version v0.9.0
```

## GitOps Layout

Cells and their apps are laid out as directories in the GitOps repository:

```
gitops/envs/
├── dev/
│   └── cells/
│       └── c0/
│           ├── stores/          # Cell-specific Store CRDs
│           │   └── warehouse.yaml
│           └── apps/            # Per-package values.yaml (managed by dk promote)
│               └── my-package/
│                   └── values.yaml   # appVersion: v1.0.0
├── int/
│   └── cells/
│       └── c0/
│           ├── stores/
│           └── apps/
└── prod/
    └── cells/
        └── c0/
            ├── stores/
            └── apps/
```

ArgoCD uses a git generator on `gitops/envs/*/cells/*/apps/*` to discover applications automatically.

## Best Practices

### 1. Progressive Promotion

Always promote through environments in order:

```bash
# Good: dev → int → prod
dk promote pkg v1.0.0 --to dev
# Test in dev
dk promote pkg v1.0.0 --to int
# Test in int
dk promote pkg v1.0.0 --to prod

# Bad: Skipping environments
dk promote pkg v1.0.0 --to prod  # ⚠ Skipped dev and int
```

### 2. Version Pinning

Use exact versions, not `latest`:

```bash
# Good
dk promote my-package v1.2.3 --to prod

# Bad
dk promote my-package latest --to prod
```

### 3. Promote Often

Small, frequent promotions are safer:

```bash
# Good: Small changes, frequently
v1.0.0 → v1.0.1 → v1.0.2 → v1.1.0

# Risky: Large changes, infrequently  
v1.0.0 → v2.0.0
```

### 4. Monitor After Promotion

Always verify after promotion:

```bash
dk promote my-package v1.0.0 --to prod
dk status my-package
dk logs my-package --env prod --follow
```

## Troubleshooting

### Promotion Stuck

If a promotion PR isn't merging:

1. Check CI status in the PR
2. Verify required approvals
3. Check for merge conflicts

```bash
dk promote my-package v1.0.0 --to dev --status
```

### Sync Failed

If ArgoCD sync fails:

```bash
# Check sync status
dk status my-package --env dev

# View sync details
dk logs my-package --env dev --sync
```

### Binding Resolution Errors

If bindings don't resolve:

1. Check binding exists in environment
2. Verify binding key matches manifest
3. Check binding target exists

## Next Steps

- [Data Packages](data-packages.md) - Package structure
- [Governance](governance.md) - Environment-specific policies
- [Troubleshooting](../troubleshooting/common-issues.md) - Common issues
