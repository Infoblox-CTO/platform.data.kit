# Quickstart: Asset Instances

**Audience**: Data engineers
**Prerequisites**: `dp` CLI installed, a project with `dp.yaml`

## Overview

Assets are configured instances of approved extensions. They are **config-only** — you declare *what to run*, not *how to run it*. Assets reference extensions by fully-qualified name (FQN) and version, with a config block validated against the extension's schema.

## 1. Create an Asset

Scaffold a new source asset from the CloudQuery AWS extension:

```bash
dp asset create aws-security --ext cloudquery.source.aws
```

This creates `assets/sources/aws-security/asset.yaml`:

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
name: aws-security
type: source
extension: cloudquery.source.aws
version: v24.0.2
ownerTeam: ""          # REQUIRED: set your team name
config:
  accounts: []         # REQUIRED: List of AWS account IDs to sync
  regions: []          # REQUIRED: AWS regions to sync
  tables: []           # REQUIRED: CloudQuery table names to sync
```

Fill in the config:

```yaml
ownerTeam: security-data
config:
  accounts:
    - "123456789012"
  regions:
    - us-east-1
    - us-west-2
  tables:
    - aws_s3_buckets
    - aws_iam_roles
    - aws_cloudtrail_events
```

> **Tip**: Use `--interactive` to be prompted for each required field:
> ```bash
> dp asset create aws-security --ext cloudquery.source.aws --interactive
> ```

## 2. Validate the Asset

Validate the asset's config against the extension's schema:

```bash
dp asset validate assets/sources/aws-security/
```

Output (success):
```
✓ asset 'aws-security' is valid
```

Output (error — missing required field):
```
✗ asset 'aws-security': config.tables is required by extension cloudquery.source.aws
  — "List of CloudQuery table names to sync"
```

Validate all assets at once:
```bash
dp validate
```

This runs the standard dp.yaml + bindings validation **plus** validates all assets under `assets/`.

## 3. Reference Assets in dp.yaml

Add the asset to your package manifest:

```yaml
# dp.yaml
apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: security-pipeline
  namespace: security
spec:
  type: cloudquery
  description: "Security compliance data pipeline"
  owner: security-data
  assets:
    - aws-security
  # ... rest of spec
```

`dp validate` will now check that `aws-security` exists in `assets/sources/` and is valid.

## 4. Associate a Binding

If your asset needs infrastructure bindings (e.g., an S3 output bucket), add a `binding` field:

```yaml
# assets/sources/aws-security/asset.yaml
binding: aws-raw-output
```

Then define the binding in `bindings.yaml`:

```yaml
# bindings.yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
metadata:
  environment: dev
bindings:
  - name: aws-raw-output
    asset: aws-security       # optional: explicitly scopes this binding
    type: s3-prefix
    properties:
      bucket: my-dev-bucket
      prefix: raw/security/
```

## 5. List and Inspect Assets

List all assets in the project:

```bash
dp asset list
```

```
NAME             TYPE     EXTENSION              VERSION   STATUS
aws-security     source   cloudquery.source.aws   v24.0.2   ✓ valid
```

Inspect a specific asset:

```bash
dp asset show aws-security
```

```yaml
name: aws-security
type: source
extension: cloudquery.source.aws
version: v24.0.2
ownerTeam: security-data
binding: aws-raw-output
config:
  accounts: ["123456789012"]
  regions: [us-east-1, us-west-2]
  tables: [aws_s3_buckets, aws_iam_roles, aws_cloudtrail_events]
status: valid
```

## End-to-End Workflow Summary

```text
dp asset create <name> --ext <fqn>    # 1. Scaffold from extension schema
     ↓
  Edit asset.yaml config               # 2. Fill in your configuration
     ↓
dp asset validate                      # 3. Validate against extension schema
     ↓
  Add to dp.yaml assets: [<name>]      # 4. Wire into your package
     ↓
dp validate                            # 5. Full package validation
     ↓
dp build → dp publish → dp promote    # 6. Normal publish/promote workflow
```

## What's Next

- **Multiple assets**: Create source + sink assets and wire them into a pipeline (feature 013)
- **Environment policies**: Automatic guardrails per environment (feature 014)
- **dbt models**: Use `--ext dbt.model-engine.core` for dbt transform assets (feature 015)
