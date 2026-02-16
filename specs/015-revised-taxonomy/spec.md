# Data Platform Kit — Revised Design

## Taxonomy

```
┌─────────────────────────────────────────────────────────────────────┐
│                         INFRA ENGINEER                              │
│                    (builds platform extensions)                      │
│                                                                     │
│   kind: Source                       kind: Destination              │
│   ├── runtime: cloudquery            ├── runtime: cloudquery        │
│   ├── runtime: generic-go            ├── runtime: generic-go        │
│   └── runtime: generic-python        └── runtime: generic-python    │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                         DATA ENGINEER                               │
│                    (uses platform to move/transform data)            │
│                                                                     │
│   kind: Model                                                       │
│   ├── runtime: cloudquery        mode: batch | streaming            │
│   ├── runtime: dbt               mode: batch                        │
│   ├── runtime: generic-python    mode: batch | streaming            │
│   └── runtime: generic-go        mode: batch | streaming            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Contracts

### Source (infra engineer)

```go
// Kind constants
const (
    KindSource      = "Source"
    KindDestination = "Destination"
    KindModel       = "Model"
)

// Runtime identifies how the extension or workload executes.
type Runtime string

const (
    RuntimeCloudQuery    Runtime = "cloudquery"
    RuntimeGenericGo     Runtime = "generic-go"
    RuntimeGenericPython Runtime = "generic-python"
    RuntimeDBT           Runtime = "dbt"
)

// Mode identifies the execution pattern.
type Mode string

const (
    ModeBatch     Mode = "batch"
    ModeStreaming Mode = "streaming"
)
```

```go
// Source is a platform extension that ingests data.
// Created by infra engineers. Published to extension registry.
// Referenced by data engineers in Model manifests.
type Source struct {
    APIVersion string         `yaml:"apiVersion"`
    Kind       string         `yaml:"kind"`       // "Source"
    Metadata   ExtMetadata    `yaml:"metadata"`
    Spec       SourceSpec     `yaml:"spec"`
}

type ExtMetadata struct {
    Name        string            `yaml:"name"`
    Namespace   string            `yaml:"namespace"`   // platform team namespace
    Version     string            `yaml:"version"`
    Labels      map[string]string `yaml:"labels,omitempty"`
    Annotations map[string]string `yaml:"annotations,omitempty"`
}

type SourceSpec struct {
    Runtime     Runtime           `yaml:"runtime"`
    Description string            `yaml:"description"`
    Owner       string            `yaml:"owner"`

    // What this source produces (its output contract)
    Provides    ArtifactContract  `yaml:"provides"`

    // Runtime-specific config schema.
    // Infra engineer defines what config knobs data engineers can set.
    ConfigSchema *ConfigSchema    `yaml:"configSchema,omitempty"`

    // Container config for generic runtimes
    Image       string            `yaml:"image,omitempty"`
    Command     []string          `yaml:"command,omitempty"`
}
```

### Destination (infra engineer)

```go
// Destination is a platform extension that writes data.
type Destination struct {
    APIVersion string         `yaml:"apiVersion"`
    Kind       string         `yaml:"kind"`       // "Destination"
    Metadata   ExtMetadata    `yaml:"metadata"`
    Spec       DestSpec       `yaml:"spec"`
}

type DestSpec struct {
    Runtime     Runtime           `yaml:"runtime"`
    Description string            `yaml:"description"`
    Owner       string            `yaml:"owner"`

    // What this destination accepts (its input contract)
    Accepts     ArtifactContract  `yaml:"accepts"`

    ConfigSchema *ConfigSchema    `yaml:"configSchema,omitempty"`
    Image        string           `yaml:"image,omitempty"`
    Command      []string         `yaml:"command,omitempty"`
}
```

### Model (data engineer)

```go
// Model is a data workload. It moves and/or transforms data
// using platform-provided sources and destinations.
type Model struct {
    APIVersion string         `yaml:"apiVersion"`
    Kind       string         `yaml:"kind"`       // "Model"
    Metadata   ModelMetadata  `yaml:"metadata"`
    Spec       ModelSpec      `yaml:"spec"`
}

type ModelMetadata struct {
    Name        string            `yaml:"name"`
    Namespace   string            `yaml:"namespace"`
    Version     string            `yaml:"version"`
    Labels      map[string]string `yaml:"labels,omitempty"`
    Annotations map[string]string `yaml:"annotations,omitempty"`
}

type ModelSpec struct {
    Runtime     Runtime        `yaml:"runtime"`
    Mode        Mode           `yaml:"mode"`
    Description string         `yaml:"description"`
    Owner       string         `yaml:"owner"`

    // References to platform extensions (optional — generic runtimes
    // may not use pre-built sources/destinations)
    Source      *ExtensionRef  `yaml:"source,omitempty"`
    Destination *ExtensionRef  `yaml:"destination,omitempty"`

    // Inputs/outputs for lineage and governance
    Inputs      []ArtifactContract `yaml:"inputs,omitempty"`
    Outputs     []ArtifactContract `yaml:"outputs,omitempty"`

    // Runtime-specific config (validated against extension's configSchema)
    Config      map[string]any `yaml:"config,omitempty"`

    // Execution
    Schedule    *ScheduleSpec  `yaml:"schedule,omitempty"`
    Resources   *ResourceSpec  `yaml:"resources,omitempty"`
    Timeout     string         `yaml:"timeout,omitempty"`
    Retries     int            `yaml:"retries,omitempty"`
    Replicas    int            `yaml:"replicas,omitempty"`

    // Container overrides (for generic runtimes or advanced use)
    Image       string         `yaml:"image,omitempty"`
    Command     []string       `yaml:"command,omitempty"`
    Env         []EnvVar       `yaml:"env,omitempty"`

    // Governance
    Lineage     *LineageSpec   `yaml:"lineage,omitempty"`
}

// ExtensionRef points to a published source or destination.
type ExtensionRef struct {
    Name      string `yaml:"name"`       // e.g. "postgres-cdc"
    Namespace string `yaml:"namespace"`  // e.g. "platform"
    Version   string `yaml:"version"`    // e.g. "1.2.0"
}
```

### Shared types (unchanged from current, mostly)

```go
type ArtifactContract struct {
    Name           string          `yaml:"name"`
    Type           ArtifactType    `yaml:"type"`
    Binding        string          `yaml:"binding"`
    Schema         *SchemaSpec     `yaml:"schema,omitempty"`
    Classification *Classification `yaml:"classification,omitempty"`
}

type ConfigSchema struct {
    // JSON Schema describing what config keys the extension accepts.
    // Used by `dp lint` to validate Model.spec.config against extensions.
    Properties map[string]ConfigProperty `yaml:"properties"`
    Required   []string                  `yaml:"required,omitempty"`
}

type ConfigProperty struct {
    Type        string `yaml:"type"`
    Description string `yaml:"description"`
    Default     any    `yaml:"default,omitempty"`
    Enum        []any  `yaml:"enum,omitempty"`
}
```

---

## Manifest Examples

### Source: PostgreSQL CDC (infra engineer, cloudquery runtime)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Source
metadata:
  name: postgres-cdc
  namespace: platform
  version: 1.0.0
  labels:
    team: infra
spec:
  runtime: cloudquery
  description: "PostgreSQL CDC source using logical replication"
  owner: platform-team

  provides:
    name: pg-changes
    type: kafka-topic
    classification:
      pii: false
      sensitivity: internal

  configSchema:
    properties:
      connection_string:
        type: string
        description: "PostgreSQL connection string"
      tables:
        type: array
        description: "Tables to replicate"
      publication:
        type: string
        description: "Logical replication publication name"
        default: "dp_publication"
    required:
      - connection_string
      - tables
```

### Source: Generic HTTP Poller (infra engineer, generic-go runtime)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Source
metadata:
  name: http-poller
  namespace: platform
  version: 1.0.0
  labels:
    team: infra
spec:
  runtime: generic-go
  description: "Polls HTTP endpoints and publishes to Kafka"
  owner: platform-team
  image: "ghcr.io/infoblox-cto/sources/http-poller:1.0.0"

  provides:
    name: http-events
    type: kafka-topic

  configSchema:
    properties:
      url:
        type: string
        description: "URL to poll"
      interval:
        type: string
        description: "Poll interval"
        default: "5m"
      format:
        type: string
        description: "Response format"
        enum: ["json", "csv", "xml"]
    required:
      - url
```

### Destination: S3 Parquet Writer (infra engineer, cloudquery runtime)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Destination
metadata:
  name: s3-parquet
  namespace: platform
  version: 2.1.0
  labels:
    team: infra
spec:
  runtime: cloudquery
  description: "Writes data to S3 in Parquet format with partitioning"
  owner: platform-team

  accepts:
    name: input
    type: s3-prefix
    schema:
      type: parquet

  configSchema:
    properties:
      bucket:
        type: string
        description: "S3 bucket name"
      prefix:
        type: string
        description: "Key prefix"
      partition_by:
        type: string
        description: "Partition column"
        default: "date"
      compression:
        type: string
        enum: ["snappy", "gzip", "zstd"]
        default: "snappy"
    required:
      - bucket
```

### Model: CloudQuery sync (data engineer, batch)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: postgres-to-s3
  namespace: analytics
  version: 1.0.0
  labels:
    team: data-engineering
    domain: events
spec:
  runtime: cloudquery
  mode: batch
  description: "Syncs user events from PostgreSQL to S3 for analytics"
  owner: data-engineering

  source:
    name: postgres-cdc
    namespace: platform
    version: ">=1.0.0"

  destination:
    name: s3-parquet
    namespace: platform
    version: ">=2.0.0"

  # Runtime-specific config — validated against source/dest configSchemas
  config:
    source:
      connection_string: "${PG_CONNECTION_STRING}"
      tables:
        - public.user_events
        - public.page_views
    destination:
      bucket: "analytics-lake"
      prefix: "raw/user-events/"
      partition_by: "event_date"

  inputs:
    - name: user-events
      type: postgres-table
      binding: input.events

  outputs:
    - name: user-events-parquet
      type: s3-prefix
      binding: output.lake
      classification:
        pii: true
        sensitivity: confidential
        dataCategory: customer-behavior
        retentionDays: 365

  schedule:
    cron: "0 */6 * * *"

  resources:
    cpu: "2"
    memory: "4Gi"

  timeout: 2h
  retries: 2
```

### Model: dbt transformation (data engineer, batch)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: user-aggregation
  namespace: analytics
  version: 1.0.0
  labels:
    team: data-engineering
spec:
  runtime: dbt
  mode: batch
  description: "Aggregates raw user events into daily summaries"
  owner: data-engineering

  # No source/destination refs — dbt talks directly to the engine
  config:
    target: production
    project_dir: ./dbt
    profiles_dir: ./dbt
    select: "tag:daily"

  inputs:
    - name: raw-events
      type: postgres-table
      binding: input.raw_events

  outputs:
    - name: daily-user-summary
      type: postgres-table
      binding: output.summaries
      classification:
        pii: false
        sensitivity: internal

  schedule:
    cron: "30 6 * * *"

  resources:
    cpu: "1"
    memory: "2Gi"
```

### Model: Generic Python streaming (data engineer)

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: fraud-detection
  namespace: risk
  version: 2.0.0
spec:
  runtime: generic-python
  mode: streaming
  description: "Real-time fraud scoring on transaction events"
  owner: risk-engineering
  image: "ghcr.io/infoblox-cto/risk/fraud-detection:2.0.0"

  inputs:
    - name: transactions
      type: kafka-topic
      binding: input.transactions

  outputs:
    - name: fraud-scores
      type: kafka-topic
      binding: output.scores
      classification:
        pii: true
        sensitivity: restricted
        retentionDays: 90

  resources:
    cpu: "4"
    memory: "8Gi"

  replicas: 3
```

---

## CLI Flow

### dp init

```
INFRA ENGINEER:

  dp init my-source --kind source --runtime cloudquery
  dp init my-source --kind source --runtime generic-go
  dp init my-dest   --kind destination --runtime cloudquery
  dp init my-dest   --kind destination --runtime generic-go

DATA ENGINEER:

  dp init my-model --kind model --runtime cloudquery
  dp init my-model --kind model --runtime dbt
  dp init my-model --kind model --runtime generic-python
  dp init my-model --kind model --runtime generic-go
```

`--kind` defaults to `model` (most common persona).
`--runtime` is required (no sane default — you must know how you're running).

### Template directory structure

```
cli/internal/templates/
├── source/
│   ├── cloudquery/
│   │   ├── dp.yaml.tmpl          # Source manifest
│   │   └── config.yaml.tmpl      # CloudQuery source spec
│   └── generic-go/
│       ├── dp.yaml.tmpl
│       ├── main.go.tmpl
│       └── Dockerfile.tmpl
├── destination/
│   ├── cloudquery/
│   │   ├── dp.yaml.tmpl
│   │   └── config.yaml.tmpl
│   └── generic-go/
│       ├── dp.yaml.tmpl
│       ├── main.go.tmpl
│       └── Dockerfile.tmpl
└── model/
    ├── cloudquery/
    │   ├── dp.yaml.tmpl          # Model manifest with source/dest refs
    │   └── config.yaml.tmpl      # CloudQuery sync config
    ├── dbt/
    │   ├── dp.yaml.tmpl
    │   ├── dbt_project.yml.tmpl
    │   ├── profiles.yml.tmpl
    │   └── models/
    │       └── example.sql.tmpl
    ├── generic-python/
    │   ├── dp.yaml.tmpl
    │   ├── main.py.tmpl
    │   ├── requirements.txt.tmpl
    │   └── Dockerfile.tmpl
    └── generic-go/
        ├── dp.yaml.tmpl
        ├── main.go.tmpl
        ├── go.mod.tmpl
        └── Dockerfile.tmpl
```

### Scaffolded output examples

```
$ dp init user-sync --kind model --runtime cloudquery

user-sync/
├── dp.yaml              # kind: Model, runtime: cloudquery
└── config.yaml          # cloudquery sync spec (source + dest tables)


$ dp init fraud-scorer --kind model --runtime generic-python

fraud-scorer/
├── dp.yaml              # kind: Model, runtime: generic-python
├── src/
│   ├── main.py          # entrypoint with SDK hooks
│   └── requirements.txt
└── Dockerfile


$ dp init user-aggregation --kind model --runtime dbt

user-aggregation/
├── dp.yaml              # kind: Model, runtime: dbt
└── dbt/
    ├── dbt_project.yml
    ├── profiles.yml
    └── models/
        └── example.sql


$ dp init pg-cdc --kind source --runtime cloudquery

pg-cdc/
├── dp.yaml              # kind: Source, runtime: cloudquery
└── config.yaml          # cloudquery source plugin spec


$ dp init s3-writer --kind destination --runtime generic-go

s3-writer/
├── dp.yaml              # kind: Destination, runtime: generic-go
├── main.go
├── go.mod
└── Dockerfile
```

### Full workflow: infra engineer

```bash
# Build a new source extension
dp init pg-cdc --kind source --runtime cloudquery
cd pg-cdc
# edit dp.yaml (set configSchema, provides contract)
# edit config.yaml (cloudquery source plugin config)

dp lint                    # validates Source manifest + configSchema
dp dev up                  # start local infra (redpanda, localstack, postgres)
dp run                     # test the source locally
dp test                    # run against test fixtures

dp build                   # bundle as OCI artifact
dp publish                 # push to extension registry

# Data engineers can now reference:
#   source:
#     name: pg-cdc
#     namespace: platform
#     version: ">=1.0.0"
```

### Full workflow: data engineer

```bash
# Build a new model
dp init user-sync --kind model --runtime cloudquery
cd user-sync
# edit dp.yaml (set source/dest refs, config, schedule, classification)

dp lint                    # validates Model manifest
                           # checks source/dest extensions exist in registry
                           # validates config against extension configSchemas

dp dev up                  # start local infra
dp run                     # run model locally (pulls source/dest extensions)
dp test --data ./fixtures  # run with test data

dp build
dp publish
dp promote user-sync 1.0.0 --to dev
dp promote user-sync 1.0.0 --to int
dp promote user-sync 1.0.0 --to prod
```

---

## Validation Rules

### Source / Destination

| Code | Field | Rule |
|------|-------|------|
| E100 | kind | Must be "Source" or "Destination" |
| E101 | spec.runtime | Must be a known runtime |
| E102 | spec.provides / spec.accepts | Required — extensions must declare their contract |
| E103 | spec.image | Required for generic-* runtimes |
| E104 | spec.configSchema | Recommended (warning if missing) |
| E105 | metadata.version | Required, valid semver |

### Model

| Code | Field | Rule |
|------|-------|------|
| E200 | kind | Must be "Model" |
| E201 | spec.runtime | Must be a known runtime |
| E202 | spec.mode | Must be "batch" or "streaming" |
| E203 | spec.outputs | Required, at least one |
| E204 | spec.outputs[].classification | Required on all outputs |
| E205 | spec.source | If set, must resolve to a published Source extension |
| E206 | spec.destination | If set, must resolve to a published Destination extension |
| E207 | spec.config | If source/dest have configSchemas, config must validate against them |
| E208 | spec.image | Required for generic-* runtimes (unless source/dest provide it) |
| E209 | spec.schedule | Required for batch mode (warning if missing) |
| E210 | metadata.version | Required, valid semver |

### Cross-cutting

| Code | Field | Rule |
|------|-------|------|
| E001 | metadata.name | DNS-safe |
| E004 | outputs[].classification | Required on all outputs |
| E020 | metadata.version | Valid semver |

---

## What to Delete from Current Codebase

```
REMOVE (legacy / dead):

  contracts/pipeline.go              # PipelineManifest — replaced by Model
  contracts/schemas/pipeline-manifest.schema.json
  cli/internal/templates/pipeline.yaml.tmpl
  cli/internal/templates/dp.yaml.tmpl           # replace with model/*/dp.yaml.tmpl
  cli/internal/templates/model.dp.yaml.tmpl     # replace with model/*/dp.yaml.tmpl
  cli/internal/templates/dataset.dp.yaml.tmpl   # datasets are just Models with no transform

RENAME / RESTRUCTURE:

  contracts/datapackage.go    →  contracts/model.go  (Model struct)
                              +  contracts/source.go  (Source struct)
                              +  contracts/destination.go (Destination struct)
  contracts/types.go          →  update PackageType → remove, replace with Kind + Runtime + Mode

KEEP (still useful):

  contracts/artifact.go       # ArtifactContract, Classification
  contracts/binding.go        # Bindings stay the same
  contracts/environment.go    # Environments stay the same
  contracts/errors.go         # ValidationErrors framework
  contracts/version.go        # APIVersion (but unify the string!)

UPDATE:

  cli/cmd/init.go             # --kind + --runtime flags, template lookup
  cli/cmd/lint.go             # dispatch to Source/Dest/Model validator
  cli/cmd/build.go            # handle all three kinds
  cli/cmd/run.go              # runtime-aware execution
  sdk/validate/               # new validators per kind
  sdk/manifest/               # new parsers per kind
  docs/                       # rewrite around two personas
```

---

## Open Questions

1. **Should `mode` default?** Probably `batch` — streaming is the explicit opt-in.

2. **Extension versioning constraints.** The model references `version: ">=1.0.0"`. Do you want semver ranges, or pinned versions only? Ranges are more ergonomic but harder to reproduce.

3. **Does `dp dev up` pull extensions locally?** When a data engineer runs `dp dev up`, should it look at their `dp.yaml`, see the source/dest refs, and pull those extension images into the local dev environment? That would make `dp run` work seamlessly, yes

4. **Where do extensions live?** Same OCI registry as models, just different `kind` in the manifest? Or a separate catalog/registry? Same registry with kind-based filtering is simpler.

5. **What about `dataset` type?** A static, versioned dataset (CSV, parquet file) with no runtime. It should be `kind: Model, runtime: none, mode: batch` with just outputs and no inputs. This is basically a glorified asset, but it fits the mental model of "data workloads are all Models" and can be managed with the same CLI commands.

6. **dbt: source/dest or engine?** dbt doesn't use your source/destination extensions — it talks to a database directly. So `source`/`destination` fields are optional on Model. But y want an `engine` field instead for dbt models (`engine: postgres`, `engine: databricks`). This would be purely informational (no runtime behavior change) but would make it clearer that dbt models don't use source/dest extensions. 
