# Research: Asset Instances (011)

**Date**: 2026-02-14
**Status**: Complete — all NEEDS CLARIFICATION resolved

## R-001: JSON Schema Validation Library for Go

**Question**: Which Go library should validate asset config blocks against extension JSON Schemas (draft 2020-12)?

**Decision**: `github.com/santhosh-tekuri/jsonschema/v6`

**Rationale**:
- Full JSON Schema draft 2020-12 compliance (passes official test suite)
- `schema.Validate(v any)` accepts `map[string]any` directly — no wrapping needed for pre-parsed YAML config
- Rich, structured errors: `*ValidationError` provides `InstanceLocation` (field path as `[]string`), `ErrorKind`, nested `Causes`
- Actively maintained (1.2k stars, 2.1k dependents, Apache-2.0)
- Zero dependencies beyond stdlib

**Alternatives considered**:
- `xeipuuv/gojsonschema`: No draft 2020-12 support, abandoned (6 years stale)
- `qri-io/jsonschema`: No draft 2020-12 support, abandoned (2 years stale)
- `kaptinlin/jsonschema`: Draft 2020-12, good error model, but young project (215 stars)

**Usage pattern**:
```go
import jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

var schemaDoc any
json.Unmarshal(schemaBytes, &schemaDoc)

c := jsonschema.NewCompiler()
c.AddResource("schema.json", schemaDoc)
schema, _ := c.Compile("schema.json")

// Validate parsed YAML config (map[string]any)
if err := schema.Validate(yamlConfig); err != nil {
    ve := err.(*jsonschema.ValidationError)
    // ve.InstanceLocation → field path
    // ve.ErrorKind → constraint type
    // ve.Causes → nested errors
}
```

---

## R-002: Asset Directory Structure

**Question**: How should assets be organized on the filesystem?

**Decision**: Type-based subdirectories under `assets/`

```text
<project-root>/
├── dp.yaml
├── bindings.yaml
└── assets/
    ├── sources/
    │   └── <asset-name>/
    │       └── asset.yaml
    ├── sinks/
    │   └── <asset-name>/
    │       └── asset.yaml
    └── models/
        └── <asset-name>/
            └── asset.yaml
```

**Rationale**:
- Type-based directories (`sources/`, `sinks/`, `models/`) provide visual organization at the filesystem level
- Each asset gets its own subdirectory to allow future co-located files (e.g., README, overrides)
- The asset name is the directory name — enforces uniqueness per type
- Discovery: `filepath.WalkDir("assets/")` finds all asset.yaml files; the parent directory determines the type as a cross-check against the extension's kind
- Consistent with the Go monorepo pattern where each package has its own directory

**Alternatives considered**:
- Flat `assets/` with all asset.yaml files: Doesn't scale, naming collisions between types
- Single `assets.yaml` file: Too monolithic, hard to diff/review individual asset changes

---

## R-003: Backward Compatibility for Bindings

**Question**: How do we extend bindings to be asset-scoped without breaking existing projects?

**Decision**: Additive extension with optional `asset` field on each Binding entry

**Current model** (works without assets):
```yaml
apiVersion: cdpp.io/v1alpha1
kind: Bindings
metadata:
  name: dev-bindings
  environment: dev
bindings:
  - name: raw-output
    type: s3-prefix
    s3:
      bucket: my-bucket
      prefix: raw/
```

**New model** (asset-scoped, backward compatible):
```yaml
apiVersion: cdpp.io/v1alpha1
kind: Bindings
metadata:
  name: dev-bindings
  environment: dev
bindings:
  - name: raw-output
    asset: aws_security    # NEW: optional — associates this binding with an asset
    type: s3-prefix
    s3:
      bucket: my-bucket
      prefix: raw/
```

**Compatibility rules**:
1. If no `asset` field: binding resolves via the existing top-level matching (by binding name referenced in artifact contracts) — **zero changes required**
2. If `asset` field is present: binding is associated with that asset's `binding` reference
3. Validation: if an asset references `binding: raw-output`, at least one binding with `name: raw-output` must exist; if the binding also has `asset: aws_security`, it must match the referencing asset
4. Mixed mode: a project can have both asset-scoped and top-level bindings simultaneously during migration

**Go change**: Add `Asset string` field to `Binding` struct with `omitempty`:
```go
type Binding struct {
    Name  string      `json:"name" yaml:"name"`
    Asset string      `json:"asset,omitempty" yaml:"asset,omitempty"` // NEW
    Type  BindingType `json:"type" yaml:"type"`
    // ... existing fields
}
```

**Rationale**: The `omitempty` tag ensures existing YAML without `asset` parses correctly. Existing validation logic doesn't reference the field, so it's purely additive.

---

## R-004: Extension Schema Resolution from OCI Registry

**Question**: How does the CLI fetch an extension's `schema.json` at validation time?

**Decision**: Three-tier resolution with embedded fallback

**Resolution order**:
1. **Local cache**: `~/.cache/dp/schemas/<extension-fqn>/<version>/schema.json`
2. **OCI registry**: Fetch the schema layer from the extension's OCI artifact
3. **Embedded fallback**: Built-in schemas for known extensions (CloudQuery source)

**OCI artifact structure for extensions**:
```
OCI Manifest (artifactType: application/vnd.infoblox.dp.extension.v1)
├── Config:  extension metadata (FQN, kind, version, description)
├── Layer 0: plugin binary/archive
└── Layer 1: schema.json (application/vnd.infoblox.dp.extension.schema.v1+json)
```

**Single-layer fetch** (avoids pulling the entire plugin):
```go
repo, _ := c.getRepository(ref)
desc, _ := repo.Resolve(ctx, ref)
manifestReader, _ := repo.Fetch(ctx, desc)
// Decode manifest, find layer with MediaType == MediaTypeExtensionSchema
schemaReader, _ := repo.Fetch(ctx, schemaLayerDesc) // fetches only the schema blob
```

**Cache key**: `<extension-fqn>/<version>/schema.json`
- Schemas are immutable per semver version — no TTL, no re-fetch
- OCI layer digest stored in `.digest` file for integrity verification
- `--offline` flag reads from cache only, errors if not cached

**Bootstrap (MVP)**:
```go
package schemas

import "embed"

//go:embed cloudquery.source.schema.json
var CloudQuerySourceSchema []byte
```

**Rationale**: Embedded schemas guarantee day-one functionality without a populated extension registry. As the extension registry matures, embedded schemas become a fallback. Cache ensures fast repeated validation.

---

## R-005: Asset Name Validation Rules

**Question**: What naming rules should assets follow?

**Decision**: Same DNS-safe rules as package names

**Rules**:
- Lowercase alphanumeric characters and hyphens only
- Must start with a letter
- 3–63 characters
- Pattern: `^[a-z][a-z0-9-]{2,62}$`
- Underscores are normalized to hyphens (with a warning) for ergonomics

**Rationale**: Consistent with `PackageMetadata.Name` validation in the existing `dp-manifest.schema.json`. DNS-safe names ensure assets can be referenced in any context (filenames, env vars, labels).

**Note**: The spec examples use underscores (`aws_security`), but the implementation will accept underscores and normalize to hyphens with a deprecation warning, matching the pattern used by Kubernetes for label names.

---

## R-006: Asset Type Determination

**Question**: How is the asset type (source, sink, model-engine) determined?

**Decision**: Derived from the extension's `kind` field, not independently declared

**Flow**:
1. User runs `dp asset create my-source --ext cloudquery.source.aws`
2. CLI resolves the extension from registry (or embedded catalog)
3. Extension metadata includes `kind: source` (derived from FQN: `cloudquery.source.aws` → kind=source)
4. Asset is placed in `assets/sources/my-source/` and `type: source` is set automatically
5. If the user manually sets a different `type` in asset.yaml, validation reports an error

**FQN convention**: `<vendor>.<kind>.<name>` (e.g., `cloudquery.source.aws`)
- Second segment determines the kind: `source`, `sink`, `model-engine`
- This is validated during `dp asset create` and `dp asset validate`

**Rationale**: Deriving the type from the extension FQN prevents mismatches between the extension kind and the asset type. The data engineer doesn't need to know or declare the type — it's inherited from the extension contract.

---

## Summary

| Research Item | Decision | Status |
|---|---|---|
| R-001: JSON Schema library | `santhosh-tekuri/jsonschema/v6` | ✅ Resolved |
| R-002: Directory structure | Type-based subdirs under `assets/` | ✅ Resolved |
| R-003: Binding compatibility | Additive `asset` field with `omitempty` | ✅ Resolved |
| R-004: Schema resolution | 3-tier: cache → registry → embedded | ✅ Resolved |
| R-005: Name validation | DNS-safe, `^[a-z][a-z0-9-]{2,62}$` | ✅ Resolved |
| R-006: Type determination | Derived from extension FQN kind segment | ✅ Resolved |
