# Data Model: Asset Instances (011)

**Date**: 2026-02-14
**Source**: [spec.md](spec.md), [research.md](research.md)

## Entity Relationship Overview

```text
┌─────────────────┐       references        ┌───────────────────┐
│   DataPackage   │──────────────────────────│   ExtensionRef    │
│   (dp.yaml)     │                          │  (FQN + version)  │
│                 │    ┌────────────┐         └───────┬───────────┘
│  assets: [name] │───>│   Asset    │                 │ schema.json
│                 │    │(asset.yaml)│─── validates ───┘
└────────┬────────┘    │            │
         │             │ binding:   │───> ┌──────────┐
         │             └────────────┘     │ Binding  │
         │                                │(bindings │
         └──────── bindings ─────────────>│  .yaml)  │
                                          └──────────┘
```

## Entities

### 1. Asset

The primary new entity. A configured instance of an extension — config-only, no code.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `apiVersion` | `string` | Yes | Schema version: `cdpp.io/v1alpha1` |
| `kind` | `string` | Yes | Always `Asset` |
| `name` | `string` | Yes | Unique asset identifier (DNS-safe: `^[a-z][a-z0-9-]{2,62}$`) |
| `type` | `AssetType` | Yes | `source`, `sink`, or `model-engine` — derived from extension kind |
| `extension` | `string` | Yes | Fully-qualified extension name (e.g., `cloudquery.source.aws`) |
| `version` | `string` | Yes | Extension version (semver, e.g., `v24.0.2`) |
| `owner_team` | `string` | Yes | Team that owns this asset instance |
| `description` | `string` | No | Human-readable description of what this asset does |
| `binding` | `string` | No | Name of the binding entry in bindings.yaml for this asset |
| `config` | `map[string]any` | Yes | Configuration block validated against extension's schema.json |
| `labels` | `map[string]string` | No | Key-value labels for filtering/organizing |

**Example** (`assets/sources/aws-security/asset.yaml`):
```yaml
apiVersion: cdpp.io/v1alpha1
kind: Asset
name: aws-security
type: source
extension: cloudquery.source.aws
version: v24.0.2
owner_team: security-data
description: "AWS security tables for compliance reporting"
binding: aws-raw-output
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
labels:
  domain: security
  sensitivity: internal
```

**Validation rules**:
- `name` must match `^[a-z][a-z0-9-]{2,62}$`
- `type` must be one of: `source`, `sink`, `model-engine`
- `type` must match the extension's kind segment (FQN second segment)
- `extension` must be a valid FQN (`<vendor>.<kind>.<name>`)
- `version` must be a valid semver string
- `config` is validated against the extension's `schema.json`
- `binding` (if present) must reference an existing entry in `bindings.yaml`

**State transitions**: None — assets are static configuration. They don't have lifecycle states.

---

### 2. AssetType (Enum)

| Value | Description | Extension Kind Segment |
|-------|-------------|----------------------|
| `source` | Pulls data into the platform (e.g., CloudQuery source plugin) | `source` |
| `sink` | Pushes data to an external destination (e.g., Snowflake, S3) | `sink` |
| `model-engine` | Transforms data in-place (e.g., dbt, Spark) | `model-engine` |

**Go definition**:
```go
type AssetType string

const (
    AssetTypeSource      AssetType = "source"
    AssetTypeSink        AssetType = "sink"
    AssetTypeModelEngine AssetType = "model-engine"
)
```

---

### 3. ExtensionRef

A reference to an extension published in the OCI registry. Not a standalone entity — embedded in the Asset.

| Field | Type | Description |
|-------|------|-------------|
| FQN | `string` | Fully-qualified name: `<vendor>.<kind>.<name>` |
| Version | `string` | Semver version string |
| Kind | `AssetType` | Derived from FQN second segment |

**FQN parsing rules**:
- Must contain exactly 3 dot-separated segments
- Segment 1 (`vendor`): lowercase alphanumeric + hyphens
- Segment 2 (`kind`): must be `source`, `sink`, or `model-engine`
- Segment 3 (`name`): lowercase alphanumeric + hyphens

**Resolution**: FQN + version resolves to an OCI artifact reference:
`<registry>/<vendor>/<kind>/<name>:<version>`
Example: `registry.example.com/cloudquery/source/aws:v24.0.2`

---

### 4. DataPackageSpec (Modified)

Add optional `Assets` field to the existing `DataPackageSpec`.

| Field (new) | Type | Required | Description |
|-------------|------|----------|-------------|
| `assets` | `[]string` | No | List of asset names that are part of this package |

**Go change**:
```go
type DataPackageSpec struct {
    // ... existing fields ...
    
    // Assets are asset names included in this package.
    // Each name must correspond to an asset.yaml in the assets/ directory.
    Assets []string `json:"assets,omitempty" yaml:"assets,omitempty"`
}
```

**Backward compatibility**: Field is optional with `omitempty`. Existing dp.yaml files without `assets` parse and validate correctly.

---

### 5. Binding (Modified)

Add optional `Asset` field to associate a binding with a specific asset.

| Field (new) | Type | Required | Description |
|-------------|------|----------|-------------|
| `asset` | `string` | No | Asset name this binding is associated with |

**Go change**:
```go
type Binding struct {
    Name  string      `json:"name" yaml:"name"`
    Asset string      `json:"asset,omitempty" yaml:"asset,omitempty"` // NEW
    Type  BindingType `json:"type" yaml:"type"`
    // ... existing fields ...
}
```

**Resolution rules**:
1. Asset has `binding: raw-output` → look for binding with `name: raw-output`
2. If binding also has `asset: aws-security`, it's an explicit association (validates match)
3. If binding has no `asset` field, it's a top-level binding (existing behavior)
4. Both modes can coexist in the same bindings.yaml

---

### 6. AssetManifest

A container for parsing `asset.yaml` files, analogous to `DataPackage` for `dp.yaml`.

| Field | Type | Description |
|-------|------|-------------|
| `APIVersion` | `string` | Schema version |
| `Kind` | `string` | Always `Asset` |
| `Name` | `string` | Asset identifier |
| `Type` | `AssetType` | Asset type |
| `Extension` | `string` | Extension FQN |
| `Version` | `string` | Extension version |
| `OwnerTeam` | `string` | Owning team |
| `Description` | `string` | Optional description |
| `Binding` | `string` | Optional binding reference |
| `Config` | `map[string]any` | Configuration block |
| `Labels` | `map[string]string` | Optional labels |

**Go definition**:
```go
type AssetManifest struct {
    APIVersion  string            `json:"apiVersion" yaml:"apiVersion"`
    Kind        string            `json:"kind" yaml:"kind"`
    Name        string            `json:"name" yaml:"name"`
    Type        AssetType         `json:"type" yaml:"type"`
    Extension   string            `json:"extension" yaml:"extension"`
    Version     string            `json:"version" yaml:"version"`
    OwnerTeam   string            `json:"ownerTeam" yaml:"ownerTeam"`
    Description string            `json:"description,omitempty" yaml:"description,omitempty"`
    Binding     string            `json:"binding,omitempty" yaml:"binding,omitempty"`
    Config      map[string]any    `json:"config" yaml:"config"`
    Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}
```

---

## Relationship Summary

| From | To | Relationship | Cardinality |
|------|----|-------------|-------------|
| DataPackage | Asset | references by name via `assets` list | 1:N (optional) |
| Asset | ExtensionRef | references by FQN + version | N:1 |
| Asset | Binding | references by name via `binding` field | 1:1 (optional) |
| Binding | Asset | optionally scoped via `asset` field | 1:1 (optional) |
| ExtensionRef | schema.json | provides validation schema | 1:1 |

## Cross-Reference to Functional Requirements

| Entity/Change | Functional Requirements |
|--------------|------------------------|
| Asset (new) | FR-001, FR-002, FR-003, FR-014 |
| AssetType (new) | FR-001, FR-002 |
| ExtensionRef (embedded) | FR-005, FR-015, FR-016 |
| DataPackageSpec.Assets (modified) | FR-007, FR-008 |
| Binding.Asset (modified) | FR-010, FR-011 |
| AssetManifest (new) | FR-002, FR-012, FR-013 |
