# Connector Tools & Versioned Distribution Proposal

## Problem

Once data lands in an asset (a Postgres table, an S3 prefix, a Kafka topic), there's no way to interact with it through `dp`. Users manually piece together connection strings, look up secrets, and figure out the right CLI flags for `psql`, `aws s3`, etc. This should be a first-class workflow.

Additionally, connectors today are static YAML files duplicated across every package with no versioning, no central source of truth, and no distribution mechanism — even though OCI infrastructure already exists for Transforms.

## Core Ideas

1. **Connectors expose Tools** — technology-specific actions (launch psql, generate DSN, mount S3, etc.)
2. **Connectors are versioned** — semver on `spec.version`, with a `spec.provider` identity separate from `metadata.name`
3. **Connectors are OCI-distributed** — published to and installed from OCI registries, optionally bundling Helm charts for dev infrastructure
4. **Stores pin connector versions** — `spec.connectorVersion` with semver range constraints

## Key Design Decision: Provider vs CR Name

Multiple versions of the same connector can coexist in a cluster. Since k8s doesn't allow two CRs with the same `metadata.name` in the same namespace, we separate:

- **`metadata.name`** — unique CR instance identifier (e.g., `postgres-1-2-0`)
- **`spec.provider`** — logical connector identity that stores reference (e.g., `postgres`)

```yaml
# Two versions coexisting:
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres-1-2-0
  labels:
    dp.infoblox.com/provider: postgres
spec:
  provider: postgres
  version: 1.2.0
  type: postgres
  # ...
---
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres-1-3-0
  labels:
    dp.infoblox.com/provider: postgres
spec:
  provider: postgres
  version: 1.3.0
  type: postgres
  # ...
```

Stores reference the **provider**, not the CR name:

```yaml
spec:
  connector: postgres          # refers to spec.provider
  connectorVersion: "^1.0.0"  # semver range
```

For local files (single version per provider), `metadata.name` can match the provider name for simplicity.

## Contract Changes

### ConnectorSpec — new fields

```go
type ConnectorSpec struct {
    Provider         string                         // logical identity stores reference
    Version          string                         // semver (e.g., "1.2.0")
    Type             string                         // technology identifier (unchanged)
    Protocol         string                         // wire protocol (unchanged)
    Capabilities     []string                       // source/destination (unchanged)
    Plugin           *ConnectorPlugin               // CQ plugin images (unchanged)
    Tools            []ConnectorTool                // NEW: actions this connector exposes
    ConnectionSchema map[string]ConnectionSchemaField // NEW: structured connection field declarations
}
```

### StoreSpec — new field

```go
type StoreSpec struct {
    Connector        string            // now references spec.provider (unchanged field name)
    ConnectorVersion string            // NEW: semver range constraint (e.g., "^1.0.0")
    Connection       map[string]any    // unchanged
    Secrets          map[string]string // unchanged
}
```

### New types

```go
type ConnectorTool struct {
    Name        string   // tool identifier (e.g., "psql", "dsn", "mount")
    Description string   // human-readable summary
    Type        string   // "exec" or "config"
    Requires    []string // binaries that must be on $PATH
    Command     string   // Go template for shell command (type=exec)
    Format      string   // output format for type=config: "text", "file", "env"
    Path        string   // file path for format=file
    Mode        string   // "append" or "overwrite" for format=file
    Template    string   // Go template for output content (type=config)
    PostMessage string   // rendered after execution
    Default     bool     // default tool when none specified
}

type ConnectionSchemaField struct {
    Description string // human-readable explanation
    Field       string // key in Store.spec.connection
    Default     string // fallback value
    Secret      bool   // may be fulfilled from Store.spec.secrets
    Optional    bool   // not required
}
```

### GetProvider() helper

Falls back to `spec.type` when `spec.provider` is not set (backward compatibility):

```go
func (c *Connector) GetProvider() string {
    if c.Spec.Provider != "" {
        return c.Spec.Provider
    }
    return c.Spec.Type
}
```

## Example: Postgres Connector with Tools

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres-1-2-0
  labels:
    dp.infoblox.com/provider: postgres
spec:
  provider: postgres
  version: 1.2.0
  type: postgres
  protocol: postgresql
  capabilities: [source, destination]
  plugin:
    source: ghcr.io/infobloxopen/cq-source-postgres:v0.1.0
    destination: ghcr.io/infobloxopen/cq-destination-postgresql:v8.14.1

  tools:
    - name: psql
      description: "Launch interactive psql session"
      type: exec
      requires: [psql]
      command: |
        psql "{{ .DSN }}"

    - name: dsn
      description: "Print PostgreSQL connection string"
      type: config
      format: text
      template: |
        {{ .DSN }}

    - name: pgpass
      description: "Configure ~/.pgpass for passwordless psql access"
      type: config
      format: file
      path: "~/.pgpass"
      mode: append
      template: |
        {{ .Host }}:{{ .Port }}:{{ .Database }}:{{ .User }}:{{ .Password }}
      postMessage: |
        ~/.pgpass updated. Connect with:
          psql -h {{ .Host }} -p {{ .Port }} -U {{ .User }} -d {{ .Database }}

    - name: vscode
      description: "Open in VS Code SQLTools extension"
      type: exec
      command: |
        code --open-url "vscode://mtxr.sqltools/connect?server={{ .Host }}&port={{ .Port }}&database={{ .Database }}&driver=PostgreSQL"

  connectionSchema:
    host:
      field: host
      default: localhost
    port:
      field: port
      default: "5432"
    database:
      field: database
      default: postgres
    user:
      field: username
      secret: true
    password:
      field: password
      secret: true
    sslmode:
      field: sslmode
      default: disable
```

## Example: S3 Connector with Tools

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: s3-1-0-0
  labels:
    dp.infoblox.com/provider: s3
spec:
  provider: s3
  version: 1.0.0
  type: s3
  protocol: s3
  capabilities: [destination]
  plugin:
    destination: ghcr.io/infobloxopen/cq-destination-s3:v7.10.2

  tools:
    - name: ls
      description: "List objects in the asset prefix"
      type: exec
      requires: [aws]
      command: |
        aws s3 ls s3://{{ .Bucket }}/{{ .Asset.Prefix }} \
          --endpoint-url {{ .Endpoint }} \
          --region {{ .Region }}

    - name: sync
      description: "Sync asset prefix to local directory"
      type: exec
      requires: [aws]
      command: |
        aws s3 sync s3://{{ .Bucket }}/{{ .Asset.Prefix }} ./{{ .Asset.Name }} \
          --endpoint-url {{ .Endpoint }} \
          --region {{ .Region }}

    - name: mount
      description: "Mount bucket as local filesystem via s3fs-fuse"
      type: exec
      requires: [s3fs]
      command: |
        mkdir -p /tmp/dp-mounts/{{ .Asset.Name }}
        s3fs {{ .Bucket }}:/{{ .Asset.Prefix }} /tmp/dp-mounts/{{ .Asset.Name }} \
          -o url={{ .Endpoint }} \
          -o use_path_request_style
      postMessage: |
        Mounted at /tmp/dp-mounts/{{ .Asset.Name }}
        Unmount with: fusermount -u /tmp/dp-mounts/{{ .Asset.Name }}

    - name: env
      description: "Print AWS environment variables for this store"
      type: config
      format: env
      template: |
        export AWS_ENDPOINT_URL={{ .Endpoint }}
        export AWS_DEFAULT_REGION={{ .Region }}
        export AWS_S3_BUCKET={{ .Bucket }}

  connectionSchema:
    bucket:
      field: bucket
    region:
      field: region
      default: us-east-1
    endpoint:
      field: endpoint
      optional: true
```

## Example: Store with Version Pinning

```yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: source-db
  namespace: default
spec:
  connector: postgres
  connectorVersion: "^1.0.0"
  connection:
    host: dp-postgres-postgresql.dp-local.svc.cluster.local
    port: 5432
    database: dataplatform
    username: dpuser
  secrets:
    password: "${POSTGRES_PASSWORD}"
```

## CLI Commands

### Tool execution

```
dp asset connect <asset-name> [--tool <tool>] [--cell <cell>] [--dry-run]
dp store connect <store-name> [--tool <tool>] [--cell <cell>] [--dry-run]
dp store tools   <store-name> [--cell <cell>]
```

| Flag | Description |
|------|-------------|
| `--tool` | Specific tool to run (defaults to first `default: true` or first `type: exec` tool) |
| `--cell` | Resolve store from a deployment cell's namespace |
| `--dry-run` | Print the resolved command/config without executing |

### Tool types

| Type | Behavior | `--dry-run` output |
|------|----------|-------------------|
| `exec` | Runs a shell command | Prints the command |
| `config` | Generates output (text/file/env) | Prints what would be written |

A `config` tool with `format: env` is designed for shell eval:
```bash
eval $(dp asset connect my-bucket --tool env)
aws s3 ls s3://cdpp-raw/foo/
```

### Connector management

```
dp connector list                              # list available connectors (local + registry)
dp connector show postgres                     # show connector details, version, tools
dp connector install postgres@^1.0.0           # pull from OCI, cache locally
dp connector publish ./connector/postgres.yaml  # build & push OCI artifact
dp connector tools postgres                    # list tools for a connector
```

## Resolution Flow

```
dp asset connect foo-source-table --tool psql
        │
        ▼
┌─ Load local manifests ────────────────────────┐
│  asset/source.yaml  → AssetManifest           │
│  store/source-db.yaml → Store                 │
│  connector/postgres.yaml → Connector          │
└────────────────────────────────────────────────┘
        │
        ▼
┌─ Store says connector=postgres, ^1.0.0 ───────┐
│  Find connectors where provider=postgres       │
│  Filter by semver range                        │
│  Pick highest match                            │
└────────────────────────────────────────────────┘
        │
        ▼
┌─ Build ToolContext ───────────────────────────┐
│  1. Map Store.connection via connectionSchema │
│  2. Resolve Store.secrets (env/k8s/prompt)    │
│  3. Compute DSN if protocol=postgresql        │
│  4. Attach Asset spec (.Table, .Prefix, etc.) │
└───────────────────────────────────────────────┘
        │
        ▼
┌─ Find tool "psql" on Connector ──────────────┐
│  Check requires: [psql] → which psql          │
│  Render command template with ToolContext      │
│  exec psql "postgresql://..."                 │
└───────────────────────────────────────────────┘
```

## OCI Distribution

Connectors are published as standalone OCI artifacts:

```
ghcr.io/infobloxopen/dp-connector-postgres:1.2.0
ghcr.io/infobloxopen/dp-connector-s3:1.0.0
ghcr.io/infobloxopen/dp-connector-kafka:1.1.0
```

OCI artifact structure:
```
artifact
├── manifest layer:  connector.yaml     (MediaTypeDPConnector)
├── optional layer:  helm/chart.tgz     (for dev infrastructure)
└── config:          artifact-config.json
```

Connectors can bundle a Helm chart for their dev infrastructure (e.g., postgres connector carries Bitnami postgres chart). `dp dev up` becomes connector-driven — the infrastructure deployed is determined by the connectors your stores reference.

## Versioning Contract

| Version bump | When | Store impact |
|-------------|------|-------------|
| **Patch** (1.2.**1**) | Plugin image bumps, tool template fixes | Transparent — `^1.0.0` accepts |
| **Minor** (1.**3**.0) | New tools, new optional connectionSchema fields | Transparent — `^1.0.0` accepts |
| **Major** (**2**.0.0) | Breaking connectionSchema changes, removed tools | Requires store update to `^2.0.0` |

## Backward Compatibility

All changes are additive:
- `spec.provider` — optional, defaults to `spec.type`
- `spec.version` — optional, unversioned connectors still work
- `spec.tools` — optional, connectors without tools work as before
- `spec.connectionSchema` — optional, not required for tool-less connectors
- `StoreSpec.connectorVersion` — optional, omitting means "any version"
- Existing `metadata.name` semantics unchanged for local single-version files

## Implementation Order

1. ✅ Add `Provider`, `Version`, `Tools`, `ConnectionSchema` to `ConnectorSpec`
2. ✅ Add `ConnectorVersion` to `StoreSpec`
3. Update JSON schemas, validation, tests, example YAMLs
4. Implement `dp connector list/show/tools` CLI commands
5. Implement `dp asset connect` / `dp store connect` CLI commands
6. Implement OCI publish/install for connectors (reuse `sdk/registry`)
7. Refactor `dp dev up` to be connector-driven (incremental, keep hardcoded charts as fallback)

Steps 1-5 work without OCI distribution. Steps 6-7 are layered on top.

## Open Questions

| Question | Current lean |
|----------|-------------|
| Should local `connector/` files be gitignored? | No — keep committed for self-containment. Mark as "managed" via annotation so `dp connector install` can update. |
| Connector lock file? | `connector.lock` recording exact resolved versions (like `go.sum`). Good for reproducibility, add later. |
| Should `devInfra` (Helm chart) live in the YAML or OCI layer? | Separate OCI layer. Connector YAML stays portable; Helm chart is a packaging concern. |
| How does this interact with `dp init`? | `dp init --connector postgres@^1.0.0` pulls connector, scaffolds store template, wires up asset. |
| Should stores also be OCI-distributed? | Not yet. Stores are instance-specific. The cell/k8s CRD path handles shared stores. |
