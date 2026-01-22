---
title: Governance
description: Data governance features in the Data Platform
---

# Governance

The Data Platform provides built-in governance features to ensure data quality, security, and compliance across all data packages.

## Governance Pillars

```
┌─────────────────────────────────────────────────────────────────┐
│                     Data Governance                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐       │
│   │Classification│   │   Lineage    │   │   Policy     │       │
│   │              │   │              │   │              │       │
│   │ • PII        │   │ • Origin     │   │ • Retention  │       │
│   │ • Sensitivity│   │ • Movement   │   │ • Access     │       │
│   │ • Retention  │   │ • Impact     │   │ • Compliance │       │
│   └──────────────┘   └──────────────┘   └──────────────┘       │
│          │                  │                  │                │
│          └──────────────────┼──────────────────┘                │
│                             ▼                                   │
│                    ┌──────────────┐                            │
│                    │   Unified    │                            │
│                    │  Governance  │                            │
│                    │    View      │                            │
│                    └──────────────┘                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Data Classification

Every output in a data package includes classification metadata:

```yaml
outputs:
  - name: customer-records
    type: s3-prefix
    binding: output.customers
    classification:
      pii: true
      sensitivity: confidential
      retention:
        days: 730
        deletionPolicy: archive
      tags:
        - gdpr
        - customer-data
        - eu-region
```

### Sensitivity Levels

| Level | Description | Example |
|-------|-------------|---------|
| `internal` | Internal use only | Operational metrics |
| `confidential` | Limited access | Customer data, PII |
| `restricted` | Highly sensitive | Financial data, credentials |

### PII Handling

When `pii: true` is set:

1. **Lineage tracking** highlights PII data flows
2. **Access controls** may be stricter
3. **Retention policies** are enforced
4. **Audit logging** is enhanced

## Policy Enforcement

### Manifest Validation

The `dp lint` command enforces governance policies:

```bash
dp lint ./my-package
```

Policy checks include:

| Check | Requirement |
|-------|-------------|
| Owner defined | `spec.owner` must be set |
| Classification on PII | `classification` required if `pii: true` |
| Retention specified | `retention.days` for confidential data |
| Valid sensitivity | Must be one of: internal, confidential, restricted |

### Policy Configuration

Define organization-wide policies in `.dp/policies.yaml`:

```yaml
policies:
  # Require classification on all outputs
  require_classification: true
  
  # Require owner email format
  owner_pattern: "^[a-z-]+@example\\.com$"
  
  # Maximum retention for PII data
  max_pii_retention_days: 730
  
  # Required tags for confidential data
  confidential_required_tags:
    - gdpr
```

### Policy Violations

Example validation output:

```
dp lint ./my-package

Errors (blocking):
  ✗ output 'customer-data': pii=true requires sensitivity level
  ✗ output 'customer-data': confidential data requires retention policy

Warnings:
  ⚠ spec.owner: should use email format
  ⚠ output 'logs': consider adding classification

2 errors, 2 warnings
```

## Lineage-Based Governance

Lineage enables impact analysis and compliance:

### Impact Analysis

Understand what's affected by changes:

```bash
dp lineage my-source-package --downstream
```

```
Impact Analysis: my-source-package
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Direct Consumers (3):
  ├─ analytics/dashboard-pipeline
  ├─ reporting/daily-reports
  └─ ml/training-data-prep

Indirect Consumers (7):
  ├─ analytics/executive-dashboard
  ├─ reporting/weekly-summary
  └─ ... and 5 more

PII Data Flow:
  ⚠ customer-data flows to 4 downstream packages
```

### Compliance Reporting

Generate compliance reports:

```bash
dp governance report --namespace analytics
```

```
Governance Report: analytics namespace
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Packages: 12
  ├─ With PII: 4
  ├─ Confidential: 6
  └─ Internal: 2

Policy Compliance:
  ├─ Classification: 12/12 (100%)
  ├─ Retention policies: 10/12 (83%)
  └─ Owner defined: 12/12 (100%)

Data Flow Summary:
  ├─ PII sources: 2
  ├─ PII sinks: 4
  └─ Cross-boundary flows: 1 ⚠
```

## Access Control

### Namespace-Based Access

Packages are organized into namespaces with RBAC:

```yaml
# Role definition
apiVersion: dp.io/v1alpha1
kind: Role
metadata:
  name: analytics-developer
spec:
  namespace: analytics
  rules:
    - resources: ["packages"]
      verbs: ["get", "list", "create", "update"]
    - resources: ["runs"]
      verbs: ["get", "list", "create"]
```

### Environment Promotion

Promotion to production requires approvals:

```bash
dp promote my-package v1.0.0 --to prod
```

```
Promotion Request: my-package v1.0.0 → prod
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Pre-flight Checks:
  ✓ Package exists in registry
  ✓ Version not already in prod
  ✓ Passed lint validation
  ✓ Classification complete

Approval Required:
  This package contains PII data.
  A PR will be created requiring approval from:
    - @data-platform-admins
    - @security-team

Created PR: https://github.com/org/deploys/pull/123
```

## Retention Management

### Defining Retention

Set retention policies in manifests:

```yaml
outputs:
  - name: logs
    classification:
      retention:
        days: 90
        deletionPolicy: delete
        
  - name: customer-data
    classification:
      pii: true
      retention:
        days: 730  # 2 years
        deletionPolicy: archive
```

### Deletion Policies

| Policy | Behavior |
|--------|----------|
| `delete` | Permanently remove after retention period |
| `archive` | Move to cold storage after retention period |
| `notify` | Alert owner, manual deletion required |

### Retention Reporting

View retention status:

```bash
dp governance retention --namespace analytics
```

```
Retention Status: analytics
━━━━━━━━━━━━━━━━━━━━━━━━━━━

Packages approaching retention:
  ⚠ user-events-2023: 15 days remaining (delete)
  ⚠ customer-backup-q1: 30 days remaining (archive)

Packages past retention:
  ✗ old-logs-2022: 45 days overdue (delete) - ACTION REQUIRED
```

## Audit Trail

All operations are logged for audit:

```bash
dp governance audit my-package
```

```
Audit Log: my-package
━━━━━━━━━━━━━━━━━━━━━

2025-01-22 10:00:00  user@example.com  CREATED  v1.0.0
2025-01-22 10:30:00  user@example.com  PROMOTED v1.0.0 → dev
2025-01-22 14:00:00  admin@example.com APPROVED v1.0.0 → prod
2025-01-22 14:05:00  ci-bot           PROMOTED v1.0.0 → prod
2025-01-22 14:10:00  system           RUN      run-abc123 COMPLETE
```

## Best Practices

### 1. Classify Early

Add classification when creating packages:

```bash
dp init my-package --type pipeline
# Immediately update dp.yaml with classification
```

### 2. Use Meaningful Tags

Tags enable filtering and reporting:

```yaml
classification:
  tags:
    - gdpr           # Regulatory
    - customer-data  # Data category
    - eu-region      # Geographic
    - team-analytics # Ownership
```

### 3. Review Lineage Before Changes

Always check downstream impact:

```bash
dp lineage my-package --downstream
# Review affected packages before making changes
```

### 4. Regular Audits

Schedule governance reviews:

```bash
# Weekly: check policy compliance
dp governance report --all

# Monthly: review retention status
dp governance retention --all
```

## Next Steps

- [Environments](environments.md) - Environment-specific governance
- [Lineage](lineage.md) - Deep dive into lineage tracking
- [CLI Reference](../reference/cli.md) - Governance commands
