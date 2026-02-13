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
  
  runtime:                          # Required for pipeline type
    image: string                   # Required: Container image to run
    timeout: string                 # Optional: Execution timeout (e.g., "30m")
    retries: integer                # Optional: Max retry attempts (default: 3)
    env:                            # Optional: Environment variables
      - name: string
        value: string
      - name: string
        valueFrom:
          secretRef:
            name: string
            key: string
    envFrom:                        # Optional: Environment from secrets/configmaps
      - secretRef:
          name: string
      - configMapRef:
          name: string
    resources:                      # Optional: Resource limits
      cpu: string                   # e.g., "500m", "1", "2"
      memory: string                # e.g., "512Mi", "2Gi"
  
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
| Values | `pipeline`, `cloudquery`, `producer`, `consumer`, `streaming` |
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

## spec.cloudquery (CloudQuery Configuration)

For packages with `type: cloudquery`, the `spec.cloudquery` section configures the CloudQuery plugin.

### Full Schema

```yaml
spec:
  type: cloudquery
  cloudquery:
    role: string                    # Required: Plugin role ("source")
    tables:                         # Optional: List of table names
      - string
    grpcPort: integer               # Optional: gRPC server port (default: 7777)
    concurrency: integer            # Optional: Max concurrent resolvers (default: 10000)
  runtime:
    image: string                   # Required: Container image for the plugin
```

### Field Reference

#### spec.cloudquery.role

| Property | Value |
|----------|-------|
| Type | enum |
| Required | Yes |
| Values | `source` |
| Description | Plugin role. Currently only `source` is supported. `destination` is reserved for future use. |

#### spec.cloudquery.tables

| Property | Value |
|----------|-------|
| Type | array of strings |
| Required | No |
| Default | `["*"]` (all tables) |
| Description | List of table names this plugin provides. Used in sync config generation. |

#### spec.cloudquery.grpcPort

| Property | Value |
|----------|-------|
| Type | integer |
| Required | No |
| Default | `7777` |
| Range | 1024–65535 |
| Description | Port the gRPC server listens on. Must not conflict with other services. |

#### spec.cloudquery.concurrency

| Property | Value |
|----------|-------|
| Type | integer |
| Required | No |
| Default | `10000` |
| Minimum | 1 |
| Description | Maximum number of concurrent table resolvers during sync. |

### Validation Rules

| Code | Rule | Severity |
|------|------|----------|
| E060 | `spec.cloudquery` is required when `type: cloudquery` | Error |
| E061 | `spec.cloudquery.role` must be a valid, supported role | Error |
| W060 | `role: destination` is recognized but not yet supported | Warning |
| E062 | `spec.cloudquery.grpcPort` must be 1024–65535 | Error |
| E063 | `spec.cloudquery.concurrency` must be > 0 | Error |

### Example

```yaml title="dp.yaml (CloudQuery source plugin)"
apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: my-source
  namespace: analytics
  version: 0.1.0
  labels:
    team: data-engineering
spec:
  type: cloudquery
  description: CloudQuery source plugin for internal API
  owner: data-engineering@example.com
  cloudquery:
    role: source
    tables:
      - users
      - orders
      - products
    grpcPort: 7777
    concurrency: 10000
  runtime:
    image: analytics/my-source:v0.1.0
```

---

## spec.runtime (Pipeline Configuration)

For packages with `type: pipeline`, the `spec.runtime` section configures container execution.

### Full Schema

```yaml
spec:
  runtime:
    image: string                   # Required: Container image to run
    timeout: string                 # Optional: Execution timeout (e.g., "30m", "1h")
    retries: integer                # Optional: Max retry attempts (default: 3)
    
    env:                            # Optional: Environment variables
      - name: string
        value: string
      - name: string
        valueFrom:
          secretRef:
            name: string
            key: string
            
    envFrom:                        # Optional: Environment from secrets/configmaps
      - secretRef:
          name: string
      - configMapRef:
          name: string
          
    resources:                      # Optional: Resource limits
      cpu: string                   # e.g., "500m", "1", "2"
      memory: string                # e.g., "512Mi", "2Gi"
```

### Field Reference

#### spec.runtime.image

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes (for pipeline type) |
| Examples | `myorg/my-pipeline:v1.0.0`, `python:3.11` |
| Description | Container image to execute |

#### spec.runtime.timeout

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Default | `30m` |
| Pattern | Go duration format (e.g., `30m`, `1h30m`, `2h`) |
| Description | Maximum execution time before timeout |

#### spec.runtime.retries

| Property | Value |
|----------|-------|
| Type | integer |
| Required | No |
| Default | 3 |
| Range | 0-10 |
| Description | Maximum number of retry attempts on failure |

#### spec.runtime.resources

| Property | Value |
|----------|-------|
| Type | object |
| Required | No |
| Description | Kubernetes-style resource limits |

---

## Pipeline Mode Configuration

Pipelines support two execution modes: `batch` (default) and `streaming`. The mode determines how the pipeline is deployed and executed.

### Mode Field

```yaml
spec:
  mode: string                      # Optional: batch | streaming (default: batch)
```

### Batch Mode Fields

For `mode: batch` pipelines (the default):

```yaml
spec:
  mode: batch
  timeout: string                   # Required: Max execution time (e.g., "30m", "1h")
  retries: integer                  # Optional: Retry count on failure (default: 3)
  backoffLimit: integer             # Optional: Kubernetes backoff limit (default: 3)
  schedule:                         # Optional: Cron scheduling
    cron: string                    # Cron expression (e.g., "0 2 * * *")
    timezone: string                # Timezone (default: UTC)
```

### Streaming Mode Fields

For `mode: streaming` pipelines:

```yaml
spec:
  mode: streaming
  replicas: integer                 # Optional: Number of replicas (default: 1)
  terminationGracePeriodSeconds: integer  # Optional: Shutdown grace period (default: 30)
  
  livenessProbe:                    # Optional: Liveness health check
    httpGet:
      path: string                  # Health endpoint path (e.g., "/healthz")
      port: integer                 # Port number
      scheme: string                # HTTP or HTTPS (default: HTTP)
    initialDelaySeconds: integer    # Delay before first probe (default: 0)
    periodSeconds: integer          # Probe interval (default: 10)
    timeoutSeconds: integer         # Probe timeout (default: 1)
    successThreshold: integer       # Successes for healthy (default: 1)
    failureThreshold: integer       # Failures for unhealthy (default: 3)
    
  readinessProbe:                   # Optional: Readiness health check
    httpGet:
      path: string
      port: integer
      scheme: string
    # Same timing fields as livenessProbe
    
  lineage:                          # Optional: Lineage configuration
    enabled: boolean                # Enable OpenLineage events (default: false)
    heartbeatInterval: string       # Heartbeat frequency (default: 30s)
```

### Probe Types

Three types of probes are supported:

#### HTTP Probe

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
    scheme: HTTP
```

#### Exec Probe

```yaml
livenessProbe:
  exec:
    command:
      - /bin/sh
      - -c
      - "exit 0"
```

#### TCP Probe

```yaml
livenessProbe:
  tcpSocket:
    port: 8080
```

### Field Reference

#### spec.mode

| Property | Value |
|----------|-------|
| Type | enum |
| Required | No |
| Values | `batch`, `streaming` |
| Default | `batch` |
| Description | Pipeline execution mode |

#### spec.replicas (streaming)

| Property | Value |
|----------|-------|
| Type | integer |
| Required | No |
| Default | 1 |
| Range | 1-100 |
| Description | Number of concurrent replicas |

#### spec.terminationGracePeriodSeconds (streaming)

| Property | Value |
|----------|-------|
| Type | integer |
| Required | No |
| Default | 30 |
| Range | 0-3600 |
| Description | Seconds to wait for graceful shutdown |

#### spec.lineage.heartbeatInterval (streaming)

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Default | 30s |
| Pattern | Go duration format |
| Description | Interval for RUNNING lineage events |

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

### CloudQuery Errors

| Code | Message | Resolution |
|------|---------|------------|
| E060 | `spec.cloudquery is required when type is cloudquery` | Add `spec.cloudquery` section |
| E061 | `spec.cloudquery.role is required and must be valid` | Set `role: source` |
| W060 | `role: destination is not yet supported` | Use `role: source` (warning in normal mode, error in strict) |
| E062 | `spec.cloudquery.grpcPort must be between 1024 and 65535` | Use a valid port number |
| E063 | `spec.cloudquery.concurrency must be greater than 0` | Set concurrency ≥ 1 |

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
