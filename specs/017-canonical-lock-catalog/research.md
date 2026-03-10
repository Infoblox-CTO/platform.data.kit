# Phase 0: Research Findings

**Feature**: Canonical Lock & Catalog Model
**Date**: 2026-03-07

## 1. Duplicate PackageRef / ArtifactRef Types

### Decision
Unify into a single `PackageRef` in `contracts/version.go`. Deprecate `ArtifactRef`.

### Rationale
Three nearly-identical types exist:
1. **`contracts.ArtifactRef`** — `{Name, Version, Registry, Digest}` — used by CLI publish, lockfile references.
2. **`controller.PackageRef`** — `{Name, Namespace, Version, Registry, Digest}` — used by PackageDeployment CRD in `platform/controller/api/v1alpha1/packagedeployment_types.go`.
3. **`promotion.PackageRef`** — `{Name, Version, Registry, Digest}` inside `VersionFile` in `sdk/promotion/kustomize.go` — used by promotion workflow.

All three represent "a specific version of an OCI package, resolved to a digest." The only field difference is `Namespace` (present only in controller type). Unifying eliminates drift and satisfies FR-015.

### Alternatives Considered
- **Keep three types, add interface** — rejected: interface doesn't help YAML serialisation; still three schemas to maintain.
- **Make `ArtifactRef` the canonical type** — rejected: name "Artifact" is overloaded with OCI concepts. "Package" aligns with DK terminology.

### Migration Path
- Add `Namespace` to `contracts.PackageRef` (new unified type).
- Type-alias `ArtifactRef = PackageRef` with deprecation comment (pre-production exemption, no backward-compat obligation).
- Update controller import to use `contracts.PackageRef`.
- Update promotion to embed `contracts.PackageRef`.

---

## 2. Semver Library Selection

### Decision
Add `github.com/Masterminds/semver/v3` as a direct dependency in `sdk/go.mod`.

### Rationale
- **Current state**: Hand-rolled `isSemVerValid()` in `sdk/validate/manifest.go` — validates format only, no range resolution.
- **Need**: Lock resolution requires evaluating constraints like `^1.2.0`, `>=1.0.0`, `~2.1` against a list of published tags.
- **`blang/semver/v4`**: Available transitively via `k8s.io/apimachinery` but its constraint syntax differs from npm/cargo conventions that dk.yaml users expect.
- **`Masterminds/semver/v3`**: Supports `^`, `~`, `>=`, `||` constraints natively. Used by Helm (familiar to Kubernetes ecosystem). Well-maintained.

### Alternatives Considered
- **`blang/semver/v4`** — mature but constraint API requires manual construction; syntax unfamiliar to most users.
- **Hand-rolled resolver** — rejected: constraint resolution is complex and error-prone.
- **`hashicorp/go-version`** — supports constraints but lacks pre-release handling.

---

## 3. Lockfile Storage Format

### Decision
YAML file named `dk.lock` at package root (alongside `dk.yaml`), deterministically serialized.

### Rationale
- Users already work with `dk.yaml`; YAML is consistent.
- Deterministic output achieved by: sorting dependencies alphabetically, using fixed field ordering from Go struct tags, and writing through `yaml.Marshal` with sorted map keys.
- Schema versioned via `lockVersion: 1` field for future migration.
- `manifestHash` field captures SHA-256 of `dk.yaml` to detect when the lockfile is stale.

### Alternatives Considered
- **JSON** — rejected: less readable in diffs, inconsistent with dk.yaml.
- **TOML** — rejected: not used elsewhere in DK.
- **Binary format** — rejected: not diffable, violates constitution Article II.

### Determinism Strategy
1. Dependencies sorted alphabetically by name.
2. YAML output through `yaml.Marshal` (struct field order is deterministic from Go struct definition).
3. Trailing newline always present.
4. No comments in generated file (machine-authored).

---

## 4. Catalog Metadata Storage in OCI Registry

### Decision
Store catalog metadata as OCI manifest annotations on the existing package artifact. No separate artifact or manifest needed.

### Rationale
- `ArtifactManifest.Annotations` is a `map[string]string` passed through to OCI manifest during `Push`.
- ORAS `oras.PackManifest()` writes annotations from `packOptions.ManifestAnnotations` directly.
- Catalog fields (name, version, kind, namespace, owner, description, tags, created timestamp) can be stored as annotations with `dev.datakit.catalog.*` keys.
- Complex structured data (inputs/outputs/schema fingerprints) stored in `ArtifactConfig` — already serialized as JSON config blob.
- Querying: `Tags()` enumerates versions; `Pull()` retrieves annotations for filtering.

### Alternatives Considered
- **Separate OCI artifact per catalog entry** — rejected: doubles registry storage, complicates garbage collection.
- **External catalog service/database** — rejected: violates Article VIII (pragmatism — adds infrastructure dependency).
- **OCI Referrers API** — rejected: not all registries support it; GHCR support is experimental.

### Annotation Key Design
```
dev.datakit.catalog.kind       = "Connector"
dev.datakit.catalog.namespace  = "infoblox"
dev.datakit.catalog.owner      = "platform-team"
dev.datakit.catalog.created    = "2026-03-07T12:00:00Z"
dev.datakit.catalog.tags       = "production,certified"
dev.datakit.catalog.description = "PostgreSQL CDC connector"
```

---

## 5. Lockfile Validation Error Codes

### Decision
Reserve error codes E300–E319 for lockfile validation errors.

### Rationale
- Current error codes go up to E241 (`ErrCodeAssetGroupAssetsRequired`).
- Range E250–E299 left as buffer for future kind-specific errors.
- Lockfile errors are a distinct category deserving their own block.

### Error Codes
| Code | Constant | Description |
|------|----------|-------------|
| E300 | `ErrCodeLockfileStale` | `dk.lock` manifest hash does not match current `dk.yaml` |
| E301 | `ErrCodeLockfileOrphan` | Lockfile entry for dependency not declared in `dk.yaml` |
| E302 | `ErrCodeLockfileMissing` | Dependency declared in `dk.yaml` not found in `dk.lock` |
| E303 | `ErrCodeLockfileConstraintViolation` | Locked version does not satisfy `dk.yaml` version constraint |
| E304 | `ErrCodeLockfileInvalidDigest` | Digest format is invalid |
| E305 | `ErrCodeLockfileInvalidVersion` | Locked version is not valid semver |
| W300 | `WarnCodeLockfileNotFound` | No `dk.lock` file found (warning, not error) |

---

## 6. Integration Points in Existing Code

### `dk publish` (cli/cmd/publish.go)
- **Where**: After `client.Push()` succeeds (~line 194), populate OCI annotations from manifest metadata.
- **How**: Before calling `Push`, add catalog annotation entries to `artifact.Manifest.Annotations`. No signature change to `Client.Push()`.
- **What**: Extract name, version, kind, namespace, owner, tags, inputs, outputs from the parsed dk.yaml manifest; set annotation keys.

### `dk lint` (cli/cmd/lint.go → sdk/validate/)
- **Where**: In `AggregateValidator.Validate()`, after existing `validateManifest` + `validateSchemas` + `validateAssets` steps.
- **How**: Add new `validateLockfile()` method that reads `dk.lock`, parses it, and cross-references against dk.yaml dependencies.
- **What**: Check manifest hash, orphans, missing entries, constraint violations.

### `dk lock` (NEW — cli/cmd/lock.go)
- **Dependency**: On `sdk/lock` package (new) and `sdk/registry.Client` (existing).
- **Flow**: Parse dk.yaml → extract dependency constraints → for each, query registry tags → resolve best match via semver → write dk.lock.

### `dk catalog search` (NEW — cli/cmd/catalog.go)
- **Dependency**: On `sdk/registry.Client` (existing) for `Tags()` + `Pull()`.
- **Flow**: List tags for repository → Pull each to read annotations → filter by kind/namespace/owner/tag → display table.

---

## 7. `ArtifactConfig` Extension for Catalog Data

### Decision
Add catalog-specific fields to `ArtifactConfig` for structured data that doesn't fit in string annotations.

### Rationale
- `ArtifactConfig` is already serialised as the OCI config blob (JSON).
- Adding fields like `Inputs []AssetRef`, `Outputs []AssetRef`, `SchemaFingerprints map[string]string` is natural.
- No change to Push signature; config blob is already marshalled from `ArtifactConfig`.

### New Fields
```go
type ArtifactConfig struct {
    Manifest        interface{}            `json:"manifest"`
    Kind            contracts.Kind         `json:"kind"`
    BuildInfo       *BuildInfo             `json:"buildInfo"`
    // New catalog fields:
    Inputs          []contracts.AssetRef   `json:"inputs,omitempty"`
    Outputs         []contracts.AssetRef   `json:"outputs,omitempty"`
    SchemaFingerprints map[string]string   `json:"schemaFingerprints,omitempty"`
    Classification  string                 `json:"classification,omitempty"`
}
```

---

## 8. `dk.lock` File Format

### Decision
Use the following YAML structure:

```yaml
# Auto-generated by dk lock. DO NOT EDIT.
lockVersion: 1
manifestHash: "sha256:abc123..."
generatedAt: "2026-03-07T12:00:00Z"
dependencies:
  - name: postgres-cdc
    kind: Connector
    version: "1.2.3"
    registry: ghcr.io/infoblox-cto
    digest: "sha256:def456..."
    constraint: ">=1.0.0"
  - name: s3-store
    kind: Store
    version: "2.0.1"
    registry: ghcr.io/infoblox-cto
    digest: "sha256:789abc..."
    constraint: "^2.0.0"
```

### Field Descriptions
- `lockVersion`: Schema version for future migration (integer).
- `manifestHash`: SHA-256 of the `dk.yaml` file content at lock time. Detects drift.
- `generatedAt`: RFC3339 timestamp when the lockfile was generated.
- `dependencies`: Sorted list of `LockedDependency` entries.
  - `name`: Dependency name (matches dk.yaml reference).
  - `kind`: Package kind (`Connector`, `Store`, `Asset`).
  - `version`: Exact resolved version (no ranges).
  - `registry`: OCI registry where the artifact is stored.
  - `digest`: OCI manifest digest (content-addressable).
  - `constraint`: Original version constraint from dk.yaml (for informational/validation purposes).
