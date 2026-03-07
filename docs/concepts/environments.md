---
title: Environments
description: Understanding deployment environments and the promotion workflow
---

# Environments

The Data Platform uses a structured environment model for promoting data packages from development to production using GitOps principles.

## Environment Model

```
┌─────────────────────────────────────────────────────────────────┐
│                     Environment Pipeline                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────┐      ┌─────────┐      ┌─────────┐                │
│   │   dev   │─────▶│   int   │─────▶│  prod   │                │
│   │         │      │         │      │         │                │
│   └─────────┘      └─────────┘      └─────────┘                │
│        │                │                │                      │
│        ▼                ▼                ▼                      │
│   ┌─────────┐      ┌─────────┐      ┌─────────┐                │
│   │ develop │      │ staging │      │ product │                │
│   │ cluster │      │ cluster │      │ cluster │                │
│   └─────────┘      └─────────┘      └─────────┘                │
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
# Creates PR, auto-merges after CI passes
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

### 2. Promote to Environment

Request promotion to an environment:

```bash
dk promote my-package v1.0.0 --to dev
```

This creates a GitOps PR in the deployment repository:

```
PR: Promote my-package v1.0.0 to dev
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Changes:
  environments/dev/my-package.yaml

+apiVersion: datakit.infoblox.dev/v1alpha1
+kind: DeployedPackage
+metadata:
+  name: my-package
+  namespace: dev
+spec:
+  version: v1.0.0
+  artifact: ghcr.io/org/my-package:v1.0.0
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
│  1. Detect change in environments/dev/my-package.yaml       │
│  2. Pull OCI artifact ghcr.io/org/my-package:v1.0.0        │
│  3. Apply Kubernetes manifests                              │
│  4. Verify deployment health                                │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Environment Stores

Packages use Store manifests to reference infrastructure. Store configurations differ per environment:

### Package Manifest

```yaml title="dk.yaml"
spec:
  store: events-store  # References a Store by name
```

### Environment-Specific Stores

```yaml title="environments/dev/stores/events-store.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: events-store
spec:
  type: kafka-topic
  connection:
    brokers: dev-kafka:9092
    topic: user-events-dev
```

```yaml title="environments/prod/stores/events-store.yaml"
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: events-store
spec:
  type: kafka-topic
  connection:
    brokers: prod-kafka:9092
    topic: user-events
```

Same package, different infrastructure per environment.

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
dk promote my-package v0.9.0 --to dev --rollback
```

The `--rollback` flag:

1. Skips some validation checks
2. Uses expedited approval (if configured)
3. Adds rollback annotation to deployment

## Environment Configuration

### Environment Definition

Environments are defined in the deployment repository:

```yaml title="config/environments.yaml"
environments:
  dev:
    cluster: dev-cluster
    namespace: dev
    approval:
      required: false
      auto_merge: true
    stores: environments/dev/stores/
    
  int:
    cluster: staging-cluster
    namespace: int
    approval:
      required: true
      approvers: ["@team-leads"]
    stores: environments/int/stores/
    
  prod:
    cluster: prod-cluster
    namespace: production
    approval:
      required: true
      approvers: ["@team-leads", "@security"]
      count: 2
    stores: environments/prod/stores/
```

### Custom Environments

Add custom environments for specific needs:

```yaml
environments:
  # Performance testing environment
  perf:
    cluster: perf-cluster
    namespace: perf-test
    approval:
      required: true
      approvers: ["@perf-team"]
    stores: environments/perf/stores/
    
  # Disaster recovery environment
  dr:
    cluster: dr-cluster
    namespace: production
    approval:
      required: true
      approvers: ["@platform-team", "@security"]
      count: 2
    stores: environments/dr/stores/
```

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
