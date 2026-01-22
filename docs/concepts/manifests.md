---
title: Manifests
description: Complete reference for data package manifest files
---

# Manifests

The manifest (`dp.yaml`) is the central configuration file for every data package. It defines metadata, inputs, outputs, and governance requirements.

## Manifest Structure

```yaml title="dp.yaml"
apiVersion: dp.io/v1alpha1    # API version
kind: DataPackage             # Resource type
metadata:                     # Package metadata
  name: my-package
  namespace: default
spec:                         # Package specification
  type: pipeline
  description: Description
  owner: team@example.com
  inputs: []
  outputs: []
```

## Full Schema

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `apiVersion` | string | Always `dp.io/v1alpha1` |
| `kind` | string | Always `DataPackage` |
| `metadata.name` | string | Package name (lowercase, hyphenated) |
| `spec.type` | string | One of: `pipeline`, `producer`, `consumer`, `streaming` |
| `spec.owner` | string | Owner email or team identifier |

### metadata

```yaml
metadata:
  name: my-kafka-pipeline          # Required: unique package name
  namespace: analytics             # Optional: logical grouping
  labels:                          # Optional: key-value labels
    team: data-engineering
    domain: events
    cost-center: analytics
  annotations:                     # Optional: arbitrary metadata
    dp.io/documentation: https://wiki.example.com/my-pipeline
```

#### Naming Rules

- **name**: 1-63 characters, lowercase alphanumeric and hyphens
- **namespace**: 1-63 characters, lowercase alphanumeric and hyphens
- **labels**: Keys up to 63 chars, values up to 253 chars

### spec.inputs

Define what data the package consumes:

```yaml
spec:
  inputs:
    - name: user-events          # Unique name within package
      type: kafka-topic          # Data source type
      binding: input.events      # Reference to bindings.yaml
      description: "Raw user event stream"
      schema: schemas/events.avsc    # Optional: schema file path
      required: true             # Default: true
      config:                    # Type-specific configuration
        format: avro
        consumer-group: my-pipeline-consumer
```

#### Input Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for this input |
| `type` | string | Yes | Type of data source |
| `binding` | string | Yes | Reference to binding key |
| `description` | string | No | Human-readable description |
| `schema` | string | No | Path to schema file |
| `required` | boolean | No | Whether input is required (default: true) |
| `config` | object | No | Type-specific configuration |

### spec.outputs

Define what data the package produces:

```yaml
spec:
  outputs:
    - name: processed-events     # Unique name within package
      type: s3-prefix            # Data destination type
      binding: output.data       # Reference to bindings.yaml
      description: "Processed events in Parquet format"
      schema: schemas/output.avsc
      classification:            # Data classification
        pii: false
        sensitivity: internal
      config:
        format: parquet
        partitioning: date
```

#### Output Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Unique identifier for this output |
| `type` | string | Yes | Type of data destination |
| `binding` | string | Yes | Reference to binding key |
| `description` | string | No | Human-readable description |
| `schema` | string | No | Path to schema file |
| `classification` | object | No | Data classification metadata |
| `config` | object | No | Type-specific configuration |

### spec.classification

Data governance classification:

```yaml
spec:
  outputs:
    - name: customer-data
      type: s3-prefix
      binding: output.customers
      classification:
        pii: true                  # Contains personally identifiable info
        sensitivity: confidential  # internal, confidential, restricted
        retention:
          days: 365
          deletionPolicy: archive
        tags:
          - gdpr
          - customer-data
```

#### Classification Fields

| Field | Type | Description |
|-------|------|-------------|
| `pii` | boolean | Contains personally identifiable information |
| `sensitivity` | string | `internal`, `confidential`, or `restricted` |
| `retention.days` | integer | Retention period in days |
| `retention.deletionPolicy` | string | `delete` or `archive` |
| `tags` | array | Custom classification tags |

### Complete Example

```yaml title="dp.yaml"
apiVersion: dp.io/v1alpha1
kind: DataPackage
metadata:
  name: user-events-processor
  namespace: analytics
  labels:
    team: data-engineering
    domain: user-behavior
    cost-center: CC-ANALYTICS-001
  annotations:
    dp.io/docs: https://wiki.example.com/user-events
    dp.io/runbook: https://wiki.example.com/runbooks/user-events
    
spec:
  type: pipeline
  description: |
    Processes raw user events from Kafka, enriches with user metadata,
    and writes to S3 in Parquet format for analytics consumption.
  owner: data-engineering@example.com
  
  inputs:
    - name: raw-events
      type: kafka-topic
      binding: input.events
      description: Raw user event stream from web application
      schema: schemas/raw-event.avsc
      config:
        format: avro
        consumer-group: user-events-processor
        
    - name: user-metadata
      type: database-table
      binding: input.users
      description: User metadata for enrichment
      required: false
      
  outputs:
    - name: enriched-events
      type: s3-prefix
      binding: output.events
      description: Enriched events in Parquet format
      schema: schemas/enriched-event.avsc
      classification:
        pii: true
        sensitivity: confidential
        retention:
          days: 730
          deletionPolicy: archive
        tags:
          - user-data
          - gdpr
      config:
        format: parquet
        partitioning: date
        compression: snappy
```

## Validation

Validate your manifest:

```bash
dp lint ./my-package
```

The linter checks:

- ✓ Required fields present
- ✓ Valid names (lowercase, hyphenated)
- ✓ Binding references match bindings.yaml
- ✓ Schema files exist if specified
- ✓ Classification is valid for outputs

### Common Validation Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `invalid name` | Uppercase or special chars | Use lowercase and hyphens only |
| `missing binding` | Binding key not in bindings.yaml | Add binding definition |
| `schema not found` | Schema file doesn't exist | Create file or remove reference |
| `pii without sensitivity` | PII true but no sensitivity level | Add sensitivity classification |

## Next Steps

- [Data Packages](data-packages.md) - Package structure overview
- [Lineage](lineage.md) - How manifests enable lineage
- [Configuration Reference](../reference/configuration.md) - Full config options
