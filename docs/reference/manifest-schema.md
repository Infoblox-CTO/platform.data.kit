---
title: Manifest Schema
description: Complete reference for data package manifest schemas
---

# Manifest Schema Reference

This document provides the complete JSON Schema reference for all data package manifest files.

## dp.yaml Schema

The main package manifest file.

### Full Schema

```yaml
# dp.yaml
apiVersion: dp.io/v1alpha1          # Required: API version
kind: DataPackage                   # Required: Resource type

metadata:                           # Required: Package metadata
  name: string                      # Required: Package name (1-63 chars, lowercase, hyphenated)
  namespace: string                 # Optional: Logical grouping
  labels:                           # Optional: Key-value labels
    key: value
  annotations:                      # Optional: Arbitrary metadata
    key: value

spec:                               # Required: Package specification
  type: string                      # Required: pipeline | producer | consumer | streaming
  description: string               # Optional: Human-readable description
  owner: string                     # Required: Owner email or team identifier
  
  inputs:                           # Optional: Input declarations
    - name: string                  # Required: Unique input name
      type: string                  # Required: kafka-topic | s3-prefix | database-table | http-endpoint
      binding: string               # Required: Reference to bindings.yaml key
      description: string           # Optional: Human-readable description
      schema: string                # Optional: Path to schema file
      required: boolean             # Optional: Default true
      config:                       # Optional: Type-specific configuration
        key: value
        
  outputs:                          # Optional: Output declarations
    - name: string                  # Required: Unique output name
      type: string                  # Required: kafka-topic | s3-prefix | database-table | http-endpoint
      binding: string               # Required: Reference to bindings.yaml key
      description: string           # Optional: Human-readable description
      schema: string                # Optional: Path to schema file
      classification:               # Optional: Data classification
        pii: boolean                # Does it contain PII?
        sensitivity: string         # internal | confidential | restricted
        retention:                  # Optional: Retention policy
          days: integer             # Retention period in days
          deletionPolicy: string    # delete | archive | notify
        tags:                       # Optional: Custom classification tags
          - string
      config:                       # Optional: Type-specific configuration
        key: value
```

### Field Reference

#### metadata.name

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Pattern | `^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$` |
| Description | Unique package identifier |

#### metadata.namespace

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Default | `default` |
| Pattern | `^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$` |
| Description | Logical grouping for packages |

#### spec.type

| Property | Value |
|----------|-------|
| Type | enum |
| Required | Yes |
| Values | `pipeline`, `producer`, `consumer`, `streaming` |
| Description | Type of data package |

#### spec.inputs[].type

| Property | Value |
|----------|-------|
| Type | enum |
| Required | Yes |
| Values | `kafka-topic`, `s3-prefix`, `database-table`, `http-endpoint` |
| Description | Type of input data source |

#### spec.outputs[].classification.sensitivity

| Property | Value |
|----------|-------|
| Type | enum |
| Required | When `pii: true` |
| Values | `internal`, `confidential`, `restricted` |
| Description | Data sensitivity level |

---

## pipeline.yaml Schema

Pipeline-specific configuration.

```yaml
# pipeline.yaml
apiVersion: dp.io/v1alpha1
kind: PipelineConfig

spec:
  runtime: string                   # Required: Runtime image (e.g., python:3.11)
  
  schedule:                         # Optional: Scheduling configuration
    cron: string                    # Cron expression
    timezone: string                # Optional: Timezone (default: UTC)
    
  resources:                        # Optional: Resource requirements
    requests:
      memory: string                # e.g., "512Mi"
      cpu: string                   # e.g., "500m"
    limits:
      memory: string                # e.g., "2Gi"
      cpu: string                   # e.g., "2"
      
  retries:                          # Optional: Retry configuration
    maxAttempts: integer            # Default: 3
    backoffMultiplier: number       # Default: 2
    initialDelaySeconds: integer    # Default: 10
    
  timeout: string                   # Optional: Execution timeout (e.g., "30m")
  
  env:                              # Optional: Environment variables
    - name: string
      value: string
    - name: string
      valueFrom:
        secretRef:
          name: string
          key: string
```

### Field Reference

#### spec.runtime

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Examples | `python:3.11`, `python:3.10-slim`, `node:20` |
| Description | Base container image for pipeline execution |

#### spec.schedule.cron

| Property | Value |
|----------|-------|
| Type | string |
| Required | When schedule is defined |
| Pattern | Standard cron expression (5 fields) |
| Examples | `0 0 * * *` (daily), `0 */6 * * *` (every 6 hours) |

#### spec.resources

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Description | Kubernetes-style resource requirements |

---

## bindings.yaml Schema

Infrastructure binding references.

```yaml
# bindings.yaml
apiVersion: dp.io/v1alpha1
kind: Bindings

spec:
  bindings:
    <binding-key>:                  # Key referenced in dp.yaml
      type: string                  # Required: Same as input/output type
      ref: string                   # Required: Infrastructure reference
      config:                       # Optional: Connection configuration
        key: value
```

### Binding Types

#### kafka-topic

```yaml
bindings:
  input.events:
    type: kafka-topic
    ref: cluster-name/topic-name
    config:
      bootstrap-servers: kafka:9092
      consumer-group: my-pipeline-consumer
      format: avro
```

#### s3-prefix

```yaml
bindings:
  output.data:
    type: s3-prefix
    ref: bucket-name/path/prefix/
    config:
      endpoint: http://minio:9000
      region: us-east-1
```

#### database-table

```yaml
bindings:
  input.users:
    type: database-table
    ref: database-name/schema/table
    config:
      host: postgres:5432
      driver: postgresql
```

---

## Validation Rules

### Required Field Errors

| Code | Message | Resolution |
|------|---------|------------|
| E001 | `metadata.name is required` | Add name field |
| E002 | `spec.type is required` | Add type field |
| E003 | `spec.owner is required` | Add owner field |

### Format Errors

| Code | Message | Resolution |
|------|---------|------------|
| E004 | `invalid name format` | Use lowercase and hyphens only |
| E005 | `name too long` | Maximum 63 characters |

### Reference Errors

| Code | Message | Resolution |
|------|---------|------------|
| E010 | `binding not found: <key>` | Add binding to bindings.yaml |
| E011 | `schema file not found: <path>` | Create schema file |

### Classification Errors

| Code | Message | Resolution |
|------|---------|------------|
| E025 | `pii=true requires sensitivity` | Add sensitivity level |
| E026 | `confidential requires retention` | Add retention policy |

---

## Example: Complete Package

### dp.yaml

```yaml
apiVersion: dp.io/v1alpha1
kind: DataPackage
metadata:
  name: user-events-processor
  namespace: analytics
  labels:
    team: data-engineering
    domain: events
spec:
  type: pipeline
  description: Processes user events from Kafka to S3
  owner: data-engineering@example.com
  
  inputs:
    - name: events
      type: kafka-topic
      binding: input.events
      schema: schemas/event.avsc
      
  outputs:
    - name: processed
      type: s3-prefix
      binding: output.data
      schema: schemas/output.avsc
      classification:
        pii: false
        sensitivity: internal
```

### pipeline.yaml

```yaml
apiVersion: dp.io/v1alpha1
kind: PipelineConfig
spec:
  runtime: python:3.11
  schedule:
    cron: "0 */4 * * *"
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "2Gi"
      cpu: "2"
  retries:
    maxAttempts: 3
  timeout: "30m"
```

### bindings.yaml

```yaml
apiVersion: dp.io/v1alpha1
kind: Bindings
spec:
  bindings:
    input.events:
      type: kafka-topic
      ref: analytics/user-events
      config:
        format: avro
        consumer-group: user-events-processor
        
    output.data:
      type: s3-prefix
      ref: analytics-bucket/processed/events/
      config:
        format: parquet
        compression: snappy
```

---

## See Also

- [CLI Reference](cli.md) - dp lint command
- [Concepts: Manifests](../concepts/manifests.md) - Conceptual overview
- [Concepts: Data Packages](../concepts/data-packages.md) - Package structure
