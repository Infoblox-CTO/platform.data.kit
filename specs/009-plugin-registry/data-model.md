# Data Model: Plugin Registry & Configuration Management

**Feature**: 009-plugin-registry
**Date**: 2026-02-13
**Source**: [spec.md](spec.md) Key Entities section

---

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────┐
│                      Config                          │
│  (merged view from all scopes)                       │
├─────────────────────────────────────────────────────┤
│  dev: DevConfig                                      │
│  plugins: PluginsConfig ──────┐                      │
└─────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────┐
│                  PluginsConfig                        │
├─────────────────────────────────────────────────────┤
│  registry: string              ← default registry    │
│  mirrors: []string             ← fallback registries │
│  overrides: map[string] ───────┐ ← per-plugin config│
└─────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────┐
│               PluginOverride                         │
│  (keyed by plugin short name: postgresql, s3, file)  │
├─────────────────────────────────────────────────────┤
│  version: string               ← override version   │
│  image: string                 ← full image ref      │
└─────────────────────────────────────────────────────┘
```

---

## Entities

### Config (root)

The top-level configuration object. Represents the merged view from all config scopes.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `dev` | DevConfig | No | See DevConfig defaults | Development environment settings (existing, unchanged) |
| `plugins` | PluginsConfig | No | See PluginsConfig defaults | Plugin registry and override settings (new) |

**Relationships**: Contains one DevConfig (existing) and one PluginsConfig (new).

**Merge Behavior**: System-scope fields are loaded first, then user-scope fields override, then repo-scope fields override. Missing fields retain values from lower-precedence scopes.

---

### DevConfig (existing — unchanged)

Development environment settings. Already exists in `sdk/localdev/config.go`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `runtime` | string | No | `"k3d"` | Runtime type: `k3d` or `compose` |
| `workspace` | string | No | `""` | Path to DP workspace |
| `k3d` | K3dConfig | No | See K3dConfig | k3d-specific settings |

---

### K3dConfig (existing — unchanged)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `clusterName` | string | No | `"dp-dev"` | k3d cluster name |

---

### PluginsConfig (new)

Plugin registry configuration. Controls where destination plugin images are pulled from and how they're resolved.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `registry` | string | No | `"ghcr.io/infobloxopen"` | Default OCI registry for plugin images |
| `mirrors` | []string | No | `[]` | Ordered list of fallback registries |
| `overrides` | map[string]PluginOverride | No | `{}` | Per-plugin version or image overrides |

**Validation Rules**:
- `registry`: Must be a valid registry URL (e.g., `ghcr.io/org`, `docker.io/org`, `internal.registry.io:5000/org`). No scheme prefix (`https://` is implied).
- `mirrors`: Each entry follows the same format as `registry`. Duplicates are rejected.
- `overrides`: Keys must be valid plugin short names (lowercase alphanumeric + hyphens).

**Merge Behavior**:
- `registry`: Scalar — higher scope wins (standard override).
- `mirrors`: Lists are merged from all scopes, deduplicated, with repo-scope mirrors first.
- `overrides`: Map — higher scope replaces entire map (repo overrides replace all user overrides).

---

### PluginOverride (new)

Per-plugin configuration that allows overriding the default version or the entire image reference.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `version` | string | No | `""` | Override version tag (e.g., `"v8.13.0"`) |
| `image` | string | No | `""` | Full image reference, bypasses registry + naming convention |

**Validation Rules**:
- `version`: Must start with `v` and follow semver pattern (e.g., `v1.2.3`).
- `image`: Must be a valid container image reference (e.g., `registry.io/org/image:tag`).
- `version` and `image` are mutually exclusive — setting `image` ignores `version`.

**Image Resolution Logic**:
1. If `override.image` is set → use it as-is (full image reference).
2. If `override.version` is set → use `{registry}/cloudquery-plugin-{name}:{override.version}`.
3. Otherwise → use `{registry}/cloudquery-plugin-{name}:{built-in-default-version}`.

---

### ConfigScope (runtime concept — not persisted)

Represents one of three configuration file locations. Used at runtime to determine which file to read/write and for display in `dp config list`.

| Scope | Path | Priority | Use Case |
|-------|------|----------|----------|
| `system` | `/etc/datakit/config.yaml` | Lowest (3) | Organization-wide defaults set by platform admins |
| `user` | `~/.config/dp/config.yaml` | Medium (2) | Personal developer preferences |
| `repo` | `{git-root}/.dp/config.yaml` | Highest (1) | Project-specific settings, checked into version control |

**Path Resolution**:
- `system`: Fixed path. Requires root/admin to write.
- `user`: Uses `os.UserHomeDir()` + `.config/dp/config.yaml`. Respects `$XDG_CONFIG_HOME` if set (use `$XDG_CONFIG_HOME/dp/config.yaml` instead).
- `repo`: Uses `git rev-parse --show-toplevel` + `.dp/config.yaml`. Skipped if not in a git repo.

---

### SupportedPlugin (built-in constant — not configurable)

The built-in map of known destination plugins with their default versions. This replaces the current `supportedDestinations` map in `run.go`.

| Plugin Name | Default Version | Default Image |
|-------------|----------------|---------------|
| `file` | `v5.5.1` | `ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1` |
| `postgresql` | `v8.14.1` | `ghcr.io/infobloxopen/cloudquery-plugin-postgresql:v8.14.1` |
| `s3` | `v7.10.1` | `ghcr.io/infobloxopen/cloudquery-plugin-s3:v7.10.1` |

**Note**: These defaults are hardcoded in Go source. Users override them via config file (`plugins.overrides.<name>.version`) or CLI flags (`--registry`).

---

## Example Config Files

### Minimal (user scope)

```yaml
# ~/.config/dp/config.yaml
dev:
  runtime: k3d
  k3d:
    clusterName: dp-local
```

### With plugin registry (user scope)

```yaml
# ~/.config/dp/config.yaml
dev:
  runtime: k3d
  k3d:
    clusterName: dp-local

plugins:
  registry: ghcr.io/infobloxopen
  mirrors:
    - ghcr.io/backup-org
  overrides:
    postgresql:
      version: v8.13.0
```

### Project-specific (repo scope)

```yaml
# .dp/config.yaml (checked into git)
plugins:
  registry: internal.registry.io/data-team
  overrides:
    postgresql:
      image: internal.registry.io/data-team/custom-pg:v2.0.0
```

### Full example (all fields)

```yaml
# ~/.config/dp/config.yaml
dev:
  runtime: k3d
  workspace: /home/dev/projects
  k3d:
    clusterName: dp-local

plugins:
  registry: ghcr.io/infobloxopen
  mirrors:
    - ghcr.io/backup-org
    - internal.registry.io/mirrors
  overrides:
    postgresql:
      version: v8.13.0
    s3:
      image: internal.registry.io/custom-s3:latest
    file:
      version: v5.5.1
```

---

## State Transitions

### Image Resolution State Machine

```
START
  │
  ├─ override.image set? ──YES──▶ USE override.image AS-IS
  │
  NO
  │
  ├─ override.version set? ──YES──▶ USE {registry}/cq-plugin-{name}:{override.version}
  │
  NO
  │
  └─▶ USE {registry}/cq-plugin-{name}:{built-in-default-version}
```

### Registry Fallback State Machine

```
START: try primary registry
  │
  ├─ docker pull succeeds? ──YES──▶ DONE (use pulled image)
  │
  NO (network error / not found)
  │
  ├─ mirrors configured? ──NO──▶ ERROR: "pull failed, no mirrors configured"
  │
  YES
  │
  ├─ for each mirror in order:
  │    ├─ docker pull from mirror succeeds? ──YES──▶ DONE (log which mirror was used)
  │    └─ NO ──▶ try next mirror
  │
  └─ all mirrors exhausted ──▶ ERROR: "pull failed from all registries"
```
