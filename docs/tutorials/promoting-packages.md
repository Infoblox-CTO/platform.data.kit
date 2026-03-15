---
title: Promoting Packages
description: Learn how to promote data packages through environments using GitOps
---

# Tutorial: Promoting Packages

This tutorial walks through the complete promotion workflow, from building a package to deploying it in production using GitOps.

**Prerequisites**: 

- Complete the [Quickstart](../getting-started/quickstart.md)
- A built and published data package
- Access to a GitOps repository

**Time**: ~25 minutes

## What You'll Learn

- Build and publish packages
- Promote through environments (dev → int → prod)
- Review GitOps PRs
- Rollback deployments
- Monitor package status

## Environment Overview

DataKit uses three standard environments:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Promotion Pipeline                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────┐      ┌─────────┐      ┌─────────┐                │
│   │   dev   │─────▶│   int   │─────▶│  prod   │                │
│   │         │      │         │      │         │                │
│   │ Auto    │      │ 1       │      │ 2       │                │
│   │ merge   │      │ approval│      │approvals│                │
│   └─────────┘      └─────────┘      └─────────┘                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

| Environment | Approval | Purpose |
|-------------|----------|---------|
| **dev** | Auto-merge | Rapid iteration |
| **int** | 1 approval | Integration testing |
| **prod** | 2 approvals | Production deployment |

## Step 1: Prepare Your Package

First, make sure your package is ready:

```bash
cd my-pipeline

# Validate manifests
dk lint

# Run locally to verify
dk dev up
dk run
dk dev down
```

## Step 2: Build the Package

Create an OCI artifact:

```bash
dk build --tag v1.0.0
```

Output:

```
▶ Building package: my-pipeline
  → Validating manifest...
  → Bundling files (5 files, 12 KB)...
  → Creating OCI artifact...
✓ Built: my-pipeline:v1.0.0

Artifact details:
  Name: my-pipeline
  Version: v1.0.0
  Digest: sha256:abc123...
  Size: 2.3 MB
```

## Step 3: Publish to Registry

Push the artifact to your registry:

```bash
dk publish
```

!!! note "Authentication"
    You may need to authenticate first:
    ```bash
    # GitHub Container Registry
    echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin
    ```

Output:

```
▶ Publishing: my-pipeline:v1.0.0
  → Checking registry authentication...
  → Pushing layers...
  → Pushing manifest...
✓ Published: ghcr.io/myorg/my-pipeline:v1.0.0

Registry: ghcr.io/myorg
Image: my-pipeline:v1.0.0
Digest: sha256:abc123...
```

## Step 4: Promote to a Cell

Deploy to the dev environment (uses the default cell `c0`):

```bash
dk promote my-pipeline v1.0.0 --to dev
```

Or promote to a specific cell within the environment:

```bash
dk promote my-pipeline v1.0.0 --to dev --cell canary
```

Output:

```
▶ Promoting: my-pipeline v1.0.0 → dev/c0

Creating PR:
  Repository: github.com/myorg/datakit
  Branch: promote/my-pipeline/dev/c0/v1.0.0/...
  File: envs/dev/cells/c0/apps/my-pipeline/values.yaml

✓ Created PR #123
  URL: https://github.com/myorg/datakit/pull/123
```

### Understanding the PR

The PR creates/updates a values.yaml in the environment + cell layout:

```yaml title="envs/dev/cells/c0/apps/my-pipeline/values.yaml"
appVersion: v1.0.0
```

ArgoCD uses a git generator to discover `envs/*/cells/*/apps/*` directories and renders the shared `dk-app` chart with each app's `values.yaml`.

### Auto-Merge

For dev environment, PR auto-merges after CI passes:

```
PR #123: Promote my-pipeline v1.0.0 to dev

✓ CI: Lint passed
✓ CI: Schema validation passed
✓ CI: Policy check passed
→ Auto-merging...
✓ Merged
```

## Step 5: Verify Deployment

Check the deployment status:

```bash
dk status my-pipeline
```

```
Package: my-pipeline
━━━━━━━━━━━━━━━━━━━

Environment  Version   Status     Last Sync
───────────  ───────   ──────     ─────────
dev          v1.0.0    Synced     2 min ago
int          -         -          -
prod         -         -          -
```

Watch logs for the first run:

```bash
dk logs my-pipeline --env dev --follow
```

## Step 6: Promote to Int

After testing in dev, promote to the integration cell:

```bash
dk promote my-pipeline v1.0.0 --to int
```

This creates a PR requiring 1 approval:

```
▶ Promoting: my-pipeline v1.0.0 → int

Pre-flight checks:
  ✓ Package exists in registry
  ✓ Currently deployed in dev (v1.0.0)
  ✓ Version not already deployed to int

Creating PR:
  Repository: github.com/myorg/gitops
  Branch: promote/my-pipeline-v1.0.0-int
  
✓ Created PR #124
  URL: https://github.com/myorg/gitops/pull/124
  Status: Awaiting approval from @team-leads
```

### Reviewing the PR

The PR includes:

1. **Changes**: Environment manifest update
2. **Checks**: CI validation results
3. **Lineage**: Link to Marquez showing data flows
4. **Changelog**: Differences from current int version

Request review from your team lead:

```bash
# In the GitHub PR
# Add reviewers: @team-leads
# Approval required before merge
```

## Step 7: Promote to Production

After int testing, promote to the production cell:

```bash
dk promote my-pipeline v1.0.0 --to prod
```

Production requires more scrutiny:

```
▶ Promoting: my-pipeline v1.0.0 → prod

Pre-flight checks:
  ✓ Package exists in registry
  ✓ Currently deployed in int (v1.0.0) for 3 days
  ✓ No failures in int runs (15 successful runs)
  ✓ Classification review passed

⚠ This package handles INTERNAL data
  Review classification before approving.

Creating PR:
  Repository: github.com/myorg/gitops

✓ Created PR #125
  URL: https://github.com/myorg/gitops/pull/125
  Status: Awaiting 2 approvals from @team-leads, @security
```

### Production PR Requirements

1. **Two approvals**: Team lead + Security
2. **Staging time**: Must have run in int for minimum period
3. **Success rate**: No failures in recent int runs
4. **Classification**: Data classification reviewed

## Step 8: Monitor After Promotion

Once promoted, monitor the deployment:

### Check Status

```bash
dk status my-pipeline
```

```
Package: my-pipeline
━━━━━━━━━━━━━━━━━━━

Environment  Version   Status    Last Run         Result
───────────  ───────   ──────    ────────         ──────
dev          v1.0.0    Synced    10 min ago       ✓ success
int          v1.0.0    Synced    2 hours ago      ✓ success
prod         v1.0.0    Synced    5 min ago        ✓ success
```

### View Logs

```bash
dk logs my-pipeline --env prod --tail 100
```

### View Lineage

<!-- dk lineage is not yet implemented -->
Open the Marquez UI at http://localhost:3000 to view the lineage graph.

Check Marquez for production lineage graph.

## Handling Rollbacks

If something goes wrong, rollback to the previous version:

### Quick Rollback

```bash
dk rollback my-pipeline --to prod --to-version v0.9.0
```

This promotes the previous version:

```
▶ Rolling back: my-pipeline in prod

Current version: v1.0.0
Previous version: v0.9.0

Creating expedited PR:
  ✓ PR #130 created
  Status: Expedited approval (1 required)
```

### Rollback to Specific Version

```bash
dk rollback my-pipeline --to prod --to-version v0.8.0
```

### Rollback a Specific Cell

```bash
dk rollback my-pipeline --to prod --cell canary --to-version v0.9.0
```

## Dry Run Mode

Preview what would happen:

```bash
dk promote my-pipeline v1.0.0 --to prod --dry-run
```

```
DRY RUN: Promoting my-pipeline v1.0.0 → prod

Would perform:
  1. Validate package in registry
  2. Check current prod version (v0.9.0)
  3. Create PR with these changes:

  + apiVersion: datakit.infoblox.dev/v1alpha1
  + kind: DeployedPackage
  + metadata:
  +   name: my-pipeline
  + spec:
  -   version: v0.9.0
  +   version: v1.0.0
```

## Version Management

### Listing Versions

See all published versions:

```bash
dk versions my-pipeline
```

```
Versions: my-pipeline
━━━━━━━━━━━━━━━━━━━━

Version   Published          Deployed
───────   ─────────          ────────
v1.0.0    2 hours ago        dev, int
v0.9.0    1 week ago         prod
v0.8.0    2 weeks ago        -
v0.7.0    3 weeks ago        -
```

### Version Comparison

```bash
dk diff my-pipeline v0.9.0 v1.0.0
```

## Best Practices

### 1. Progressive Promotion

Always follow: dev → int → prod

```bash
# Good
dk promote pkg v1.0.0 --to dev
# Test...
dk promote pkg v1.0.0 --to int
# Test...
dk promote pkg v1.0.0 --to prod

# Bad: Skipping environments
dk promote pkg v1.0.0 --to prod  # ⚠️
```

### 2. Small, Frequent Releases

Promote smaller changes more often:

```bash
# Good: Incremental versions
v1.0.0 → v1.0.1 → v1.0.2 → v1.1.0

# Risky: Large version jumps
v1.0.0 → v2.0.0
```

### 3. Monitor After Each Promotion

```bash
# Immediately after promotion
dk status my-pipeline --watch
dk logs my-pipeline --env prod --follow
```

### 4. Document Changes

Use meaningful version messages:

```bash
dk build --tag v1.0.1 --message "Fix: Handle null user IDs"
```

## Summary

You've learned how to:

- [x] Build and publish OCI artifacts
- [x] Promote through dev → int → prod
- [x] Review and approve GitOps PRs
- [x] Rollback deployments
- [x] Monitor package status
- [x] Use dry run mode

## Next Steps

- [Kafka to S3](kafka-to-s3.md) - Build a complete pipeline
- [Local Development](local-development.md) - Development workflow
- [Environments](../concepts/environments.md) - Environment configuration
- [Troubleshooting](../troubleshooting/common-issues.md) - Common issues
