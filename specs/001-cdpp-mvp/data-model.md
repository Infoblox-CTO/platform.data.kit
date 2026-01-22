# Data Model: CDPP MVP

**Feature**: 001-cdpp-mvp  
**Date**: 2026-01-22  
**Status**: Complete

## Overview

This document defines the core entities, their attributes, relationships, and validation rules for the CDPP MVP.

---

## Entity Relationship Diagram

```
┌─────────────────┐         ┌─────────────────┐
│  DataPackage    │────────▶│ PackageVersion  │
│  (dp.yaml)      │ 1    *  │ (immutable)     │
└─────────────────┘         └─────────────────┘
        │                           │
        │ declares                  │ contains
        ▼                           ▼
┌─────────────────┐         ┌─────────────────┐
│ ArtifactContract│         │ PipelineManifest│
│ (inputs/outputs)│         │ (pipeline.yaml) │
└─────────────────┘         └─────────────────┘
        │                           │
        │ references                │ uses
        ▼                           ▼
┌─────────────────┐         ┌─────────────────┐
│    Binding      │◀────────│   Environment   │
│ (abstract→real) │ *    1  │ (dev/int/prod)  │
└─────────────────┘         └─────────────────┘
                                    │
                                    │ deploys
                                    ▼
                            ┌─────────────────┐
                            │   RunRecord     │
                            │ (execution log) │
                            └─────────────────┘
                                    │
                                    │ emits
                                    ▼
                            ┌─────────────────┐
                            │  LineageEvent   │
                            │ (OpenLineage)   │
                            └─────────────────┘
```

---

## 1. DataPackage (dp.yaml)

The root manifest declaring package identity, type, and contracts.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | ✓ | Schema version (e.g., `cdpp.io/v1alpha1`) |
| `kind` | string | ✓ | Always `DataPackage` |
| `metadata.name` | string | ✓ | Package name (DNS-safe, lowercase) |
| `metadata.namespace` | string | ✓ | Team/org namespace |
| `metadata.labels` | map | | Key-value labels for filtering |
| `metadata.annotations` | map | | Arbitrary annotations |
| `spec.type` | enum | ✓ | `pipeline` \| `infra` \| `report` |
| `spec.description` | string | ✓ | Human-readable purpose |
| `spec.owner` | string | ✓ | Team or individual owner |
| `spec.inputs[]` | ArtifactRef[] | | Declared input dependencies |
| `spec.outputs[]` | ArtifactContract[] | ✓* | Declared output artifacts (*required for pipeline) |
| `spec.schedule` | ScheduleSpec | | Scheduling configuration |
| `spec.resources` | ResourceSpec | | CPU/memory requirements |

### Example

```yaml
apiVersion: cdpp.io/v1alpha1
kind: DataPackage
metadata:
  name: kafka-s3-pipeline
  namespace: analytics
  labels:
    team: data-platform
    domain: events
spec:
  type: pipeline
  description: Consumes events from Kafka, transforms, writes to S3
  owner: data-platform-team
  
  inputs:
    - name: events
      type: kafka-topic
      binding: input.events
      
  outputs:
    - name: processed-events
      type: s3-prefix
      binding: output.lake
      schema:
        type: parquet
        schemaRef: ./schemas/processed-events.avsc
      classification:
        pii: true
        sensitivity: confidential
        dataCategory: customer-behavior
        
  schedule:
    cron: "0 */6 * * *"  # Every 6 hours
    
  resources:
    cpu: "2"
    memory: "4Gi"
```

### Validation Rules

- `metadata.name` must match `^[a-z][a-z0-9-]{2,62}$`
- `spec.type` must be one of: `pipeline`, `infra`, `report`
- `spec.outputs[].classification` required when `spec.type == pipeline`
- All `binding` references must exist in target environment

---

## 2. ArtifactContract

Describes what a package produces or consumes.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Artifact identifier within package |
| `type` | enum | ✓ | `s3-prefix` \| `kafka-topic` \| `postgres-table` \| `spark-job` |
| `binding` | string | ✓ | Abstract binding reference |
| `schema.type` | enum | | `parquet` \| `avro` \| `json` \| `csv` |
| `schema.schemaRef` | string | | Path to schema file |
| `schema.inline` | object | | Inline schema definition |
| `classification.pii` | boolean | | Contains PII data |
| `classification.sensitivity` | enum | | `public` \| `internal` \| `confidential` \| `restricted` |
| `classification.dataCategory` | string | | Business domain category |
| `classification.retentionDays` | int | | Data retention period |

### Example

```yaml
name: processed-events
type: s3-prefix
binding: output.lake
schema:
  type: parquet
  schemaRef: ./schemas/processed-events.avsc
classification:
  pii: true
  sensitivity: confidential
  dataCategory: customer-behavior
  retentionDays: 365
```

---

## 3. PipelineManifest (pipeline.yaml)

Runtime configuration for pipeline execution.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | string | ✓ | Schema version |
| `kind` | string | ✓ | Always `Pipeline` |
| `spec.image` | string | ✓ | Container image reference |
| `spec.command` | string[] | | Override entrypoint |
| `spec.args` | string[] | | Command arguments |
| `spec.env[]` | EnvVar[] | | Environment variables |
| `spec.envFrom[]` | EnvFromSource[] | | Env from ConfigMap/Secret |
| `spec.timeout` | duration | | Max execution time |
| `spec.retries` | int | | Retry count on failure |
| `spec.successfulJobsHistoryLimit` | int | | Jobs to retain |
| `spec.failedJobsHistoryLimit` | int | | Failed jobs to retain |

### Example

```yaml
apiVersion: cdpp.io/v1alpha1
kind: Pipeline
spec:
  image: "${REGISTRY}/kafka-s3-pipeline:${VERSION}"
  
  env:
    - name: LOG_LEVEL
      value: info
    - name: KAFKA_BROKERS
      valueFrom:
        bindingRef: input.events.brokers
    - name: S3_BUCKET
      valueFrom:
        bindingRef: output.lake.bucket
        
  envFrom:
    - secretRef:
        name: pipeline-credentials
        
  timeout: 2h
  retries: 2
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 5
```

---

## 4. Binding

Environment-specific mapping from abstract references to concrete resources.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Binding name (matches package reference) |
| `type` | enum | ✓ | Resource type |
| `properties` | map | ✓ | Type-specific configuration |

### Binding Types

**S3 Prefix**:
```yaml
name: output.lake
type: s3-prefix
properties:
  bucket: my-data-lake-dev
  prefix: analytics/processed-events/
  region: us-west-2
```

**Kafka Topic**:
```yaml
name: input.events
type: kafka-topic
properties:
  brokers: kafka-dev.internal:9092
  topic: raw-events
  consumerGroup: kafka-s3-pipeline
```

**PostgreSQL Table**:
```yaml
name: output.catalog
type: postgres-table
properties:
  host: postgres-dev.internal
  port: 5432
  database: catalog
  table: artifacts
  # Credentials via ExternalSecret
  credentialsSecretRef: postgres-credentials
```

---

## 5. Environment

Deployment target with bindings and package versions.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✓ | Environment name |
| `clusterRef` | string | ✓ | Kubernetes cluster reference |
| `namespace` | string | ✓ | Target namespace |
| `bindings` | Binding[] | ✓ | Environment-specific bindings |
| `packages` | PackageDeployment[] | | Deployed package versions |

### Example (Kustomize overlay)

```yaml
# environments/dev/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: cdpp-dev

resources:
  - ../../base
  - bindings.yaml
  - packages/kafka-s3-pipeline.yaml

configMapGenerator:
  - name: environment-config
    literals:
      - ENVIRONMENT=dev
```

---

## 6. PackageVersion

Immutable snapshot of a package at a point in time.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `package` | string | ✓ | Package name |
| `version` | string | ✓ | SemVer version |
| `digest` | string | ✓ | OCI manifest digest |
| `createdAt` | timestamp | ✓ | Publication timestamp |
| `createdBy` | string | ✓ | Publisher identity |
| `artifacts` | ArtifactRef[] | ✓ | OCI artifact references |

### Version Reference Format

```
registry.example.com/cdpp/kafka-s3-pipeline:v1.2.3
registry.example.com/cdpp/kafka-s3-pipeline@sha256:abc123...
```

### Deployment Manifest

```yaml
# environments/dev/packages/kafka-s3-pipeline.yaml
apiVersion: cdpp.io/v1alpha1
kind: PackageDeployment
metadata:
  name: kafka-s3-pipeline
spec:
  packageRef:
    name: kafka-s3-pipeline
    namespace: analytics
    version: v1.2.3  # ← Promotion changes only this field
```

---

## 7. PromotionRecord

Auditable record of package promotion between environments.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | ✓ | Unique promotion ID |
| `package` | string | ✓ | Package name |
| `version` | string | ✓ | Version promoted |
| `sourceEnv` | string | ✓ | Source environment |
| `targetEnv` | string | ✓ | Target environment |
| `promotedAt` | timestamp | ✓ | Promotion timestamp |
| `promotedBy` | string | ✓ | Promoter identity |
| `approvedBy` | string | | Approver (if required) |
| `prUrl` | string | | Associated PR URL |
| `status` | enum | ✓ | `pending` \| `approved` \| `merged` \| `rolled-back` |

---

## 8. RunRecord

Record of a single pipeline execution.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | ✓ | Unique run ID (UUID) |
| `package` | string | ✓ | Package name |
| `version` | string | ✓ | Package version |
| `environment` | string | ✓ | Execution environment |
| `status` | enum | ✓ | `pending` \| `running` \| `succeeded` \| `failed` \| `timeout` |
| `startedAt` | timestamp | | Execution start time |
| `completedAt` | timestamp | | Execution end time |
| `duration` | duration | | Total execution time |
| `trigger` | enum | ✓ | `scheduled` \| `manual` \| `dependency` |
| `logsUrl` | string | | URL to execution logs |
| `errorMessage` | string | | Error details if failed |
| `lineageEvents` | LineageEvent[] | | Emitted lineage events |

---

## 9. LineageEvent (OpenLineage)

Lineage event following OpenLineage specification.

### Attributes

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `eventType` | enum | ✓ | `START` \| `RUNNING` \| `COMPLETE` \| `FAIL` \| `ABORT` |
| `eventTime` | timestamp | ✓ | Event timestamp (ISO 8601) |
| `run.runId` | string | ✓ | Unique run identifier |
| `job.namespace` | string | ✓ | Job namespace |
| `job.name` | string | ✓ | Job name |
| `inputs[]` | Dataset[] | | Input datasets |
| `outputs[]` | Dataset[] | | Output datasets |
| `producer` | string | ✓ | Producer URI |

### Example

```json
{
  "eventType": "COMPLETE",
  "eventTime": "2026-01-22T10:30:00Z",
  "run": {
    "runId": "run-abc123"
  },
  "job": {
    "namespace": "analytics",
    "name": "kafka-s3-pipeline",
    "facets": {
      "sourceCode": {
        "type": "git",
        "url": "https://github.com/org/kafka-s3-pipeline"
      }
    }
  },
  "inputs": [
    {
      "namespace": "kafka",
      "name": "raw-events",
      "facets": {
        "dataSource": {
          "name": "kafka-dev",
          "uri": "kafka://kafka-dev.internal:9092/raw-events"
        }
      }
    }
  ],
  "outputs": [
    {
      "namespace": "s3",
      "name": "processed-events",
      "facets": {
        "dataSource": {
          "name": "data-lake-dev",
          "uri": "s3://my-data-lake-dev/analytics/processed-events/"
        },
        "schema": {
          "fields": [
            {"name": "event_id", "type": "STRING"},
            {"name": "user_id", "type": "STRING", "description": "PII"},
            {"name": "timestamp", "type": "TIMESTAMP"}
          ]
        },
        "dataQuality": {
          "rowCount": 150000
        }
      }
    }
  ],
  "producer": "https://cdpp.io/producer"
}
```

---

## State Transitions

### Package Lifecycle

```
[Draft] → [Validated] → [Published] → [Deployed:dev] → [Deployed:int] → [Deployed:prod]
                                            ↓
                                       [Rolled-back]
```

### Run Lifecycle

```
[Pending] → [Running] → [Succeeded]
               ↓
           [Failed]
               ↓
           [Timeout]
```

---

## Validation Rules Summary

| Entity | Rule | Error Code |
|--------|------|------------|
| DataPackage | Name must be DNS-safe | `E001` |
| DataPackage | Type must be valid enum | `E002` |
| DataPackage | Outputs required for pipeline type | `E003` |
| ArtifactContract | Classification required for outputs | `E004` |
| ArtifactContract | Schema type must be valid enum | `E005` |
| Binding | All package bindings must exist in environment | `E010` |
| Binding | Type must match artifact contract type | `E011` |
| PackageVersion | Version must be valid SemVer | `E020` |
| PackageVersion | Cannot overwrite existing version | `E021` |
| PipelineManifest | Image must be valid reference | `E030` |
| PipelineManifest | Timeout must be positive duration | `E031` |
