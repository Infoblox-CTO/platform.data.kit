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

  assets:                           # Optional: Asset references
    - string                        # Asset name (DNS-safe, must match an asset under assets/)
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

## asset.yaml Schema

Asset configuration file for extension instances. Located at `assets/<type>/<name>/asset.yaml`.

### Full Schema

```yaml
# asset.yaml
apiVersion: cdpp.io/v1alpha1        # Required: API version
kind: Asset                          # Required: Always "Asset"
name: string                         # Required: Asset name (DNS-safe, 3-63 chars)
type: string                         # Required: source | sink | model-engine
extension: string                    # Required: Extension FQN (vendor.kind.name)
version: string                      # Required: Extension version (semver, e.g., v24.0.2)
ownerTeam: string                    # Required: Owning team name
description: string                  # Optional: Human-readable description
binding: string                      # Optional: Reference to bindings.yaml entry name
config:                              # Required: Extension-specific configuration
  key: value                         #   Validated against the extension's JSON Schema
labels:                              # Optional: Key-value labels
  key: value
```

### Field Reference

#### name

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Pattern | `^[a-z][a-z0-9-]{2,62}$` |
| Description | DNS-safe asset identifier. Must start with a lowercase letter. |

#### type

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Enum | `source`, `sink`, `model-engine` |
| Description | Asset type, must match the `kind` segment of the extension FQN. |

#### extension

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Format | `<vendor>.<kind>.<name>` |
| Description | Fully-qualified extension name identifying the runtime extension. |

#### version

| Property | Value |
|----------|-------|
| Type | string |
| Required | Yes |
| Format | Semantic version (e.g., `v24.0.2`) |
| Description | Pinned extension version. |

#### config

| Property | Value |
|----------|-------|
| Type | object |
| Required | Yes |
| Description | Extension-specific configuration validated against the extension's JSON Schema. |

#### binding

| Property | Value |
|----------|-------|
| Type | string |
| Required | No |
| Description | Name of a binding entry in `bindings.yaml`. Cross-validated by `dp validate`. |

### Directory Structure

Assets are placed in type-based directories under `assets/`:

| Type | Directory |
|------|-----------|
| `source` | `assets/sources/<name>/asset.yaml` |
| `sink` | `assets/sinks/<name>/asset.yaml` |
| `model-engine` | `assets/models/<name>/asset.yaml` |

### Asset Validation Error Codes

| Code | Message | Resolution |
|------|---------|------------|
| E070 | Required field missing | Add the required field to asset.yaml |
| E071 | Invalid extension FQN | Use `vendor.kind.name` format |
| E072 | Invalid version format | Use semver format (e.g., `v1.0.0`) |
| E073 | Type does not match extension kind | Ensure `type` matches the kind segment of the FQN |
| E074 | Config fails schema validation | Fix config per the extension's JSON Schema |
| E075 | Extension schema not found | Check extension FQN and version |
| E076 | Asset referenced in dp.yaml not found | Run `dp asset create <name> --ext <extension>` |
| E077 | Asset binding not found in bindings.yaml | Add the binding entry to bindings.yaml |

---

## pipeline.yaml Schema

The pipeline workflow manifest defines ordered execution steps for multi-step data pipelines.

### Full Schema

```yaml
# pipeline.yaml
apiVersion: data.infoblox.com/v1alpha1  # Required: API version
kind: PipelineWorkflow                  # Required: Resource type

metadata:                               # Required: Pipeline metadata
  name: string                          # Required: Pipeline name (3-63 chars, lowercase)
  description: string                   # Optional: Human-readable description

steps:                                  # Required: Ordered list of steps
  - name: string                        # Required: Step name (3-63 chars, DNS-safe)
    type: string                        # Required: sync | transform | test | publish | custom
    description: string                 # Optional: Step description

    # Sync step fields
    source: string                      # Required for sync: Source asset name
    sink: string                        # Required for sync: Sink asset name

    # Transform step fields
    asset: string                       # Required for transform: Asset name

    # Test step fields
    asset: string                       # Required for test: Asset to test
    command: [string]                   # Required for test: Command and args

    # Publish step fields
    promote: boolean                    # Optional: Trigger promotion
    notify:                             # Optional: Notification config
      channels: [string]               # Notification channels
      recipients: [string]             # Direct recipients

    # Custom step fields
    image: string                       # Required for custom: Container image
    command: [string]                   # Optional: Override entrypoint
    args: [string]                      # Optional: Container arguments

    # Common optional fields
    env:                                # Optional: Environment variables
      - name: string
        value: string
      - name: string
        valueFrom:
          secretRef:
            name: string
            key: string
```

### Step Type Requirements

| Type | Required Fields | Description |
|------|----------------|-------------|
| `sync` | `source`, `sink` | Data ingestion from source to sink |
| `transform` | `asset` | Transformation engine execution |
| `test` | `asset`, `command` | Validation and assertions |
| `publish` | — | Notification and promotion |
| `custom` | `image` | Arbitrary container execution |

### Validation Error Codes

| Code | Message | Resolution |
|------|---------|------------|
| E080 | `metadata.name is required` | Add name to metadata |
| E081 | `steps list is required` | Add at least one step |
| E082 | `step name is required` | Give each step a name |
| E083 | `step type is required` | Set type: sync/transform/test/publish/custom |
| E084 | `invalid step name format` | Use 3-63 lowercase chars, hyphens allowed |
| E085 | `duplicate step name` | Make each step name unique |
| E086 | `invalid step type` | Use a valid step type |
| E087 | `sync step requires source` | Add source field to sync step |
| E088 | `sync step requires sink` | Add sink field to sync step |
| E089 | `transform step requires asset` | Add asset field to transform step |
| E090 | `test step requires asset` | Add asset field to test step |
| E091 | `custom step requires image` | Add image field to custom step |

---

## schedule.yaml Schema

Optional cron-based schedule for pipeline execution.

### Full Schema

```yaml
# schedule.yaml
apiVersion: data.infoblox.com/v1alpha1  # Required: API version
kind: Schedule                          # Required: Resource type

cron: string                            # Required: 5-field cron expression
timezone: string                        # Optional: IANA timezone (default: UTC)
suspend: boolean                        # Optional: Pause execution (default: false)
```

### Cron Expression Format

```
┌───────────── minute (0-59)
│ ┌───────────── hour (0-23)
│ │ ┌───────────── day of month (1-31)
│ │ │ ┌───────────── month (1-12)
│ │ │ │ ┌───────────── day of week (0-6, Sun=0)
│ │ │ │ │
* * * * *
```

### Examples

| Expression | Description |
|-----------|-------------|
| `0 6 * * *` | Every day at 6:00 AM |
| `*/15 * * * *` | Every 15 minutes |
| `0 0 * * 1` | Every Monday at midnight |
| `0 8 1 * *` | First day of each month at 8:00 AM |

### Validation Error Codes

| Code | Message | Resolution |
|------|---------|------------|
| E100 | `cron expression is required` | Add a cron field |
| E101 | `invalid cron expression` | Use valid 5-field cron syntax |
| E102 | `invalid timezone` | Use a valid IANA timezone (e.g., America/New_York) |
| E103 | `schedule has no actionable fields` | Add cron expression |

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
