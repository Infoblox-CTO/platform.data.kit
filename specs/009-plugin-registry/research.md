# Research: Plugin Registry & Configuration Management

**Feature**: 009-plugin-registry
**Date**: 2026-02-13
**Purpose**: Resolve all technical unknowns identified in plan.md Technical Context

---

## R-001: Hierarchical Config Merge Strategy in Go

**Question**: How to merge YAML config from three file scopes (system → user → repo) with zero new dependencies?

**Decision**: Custom `yaml.v3` unmarshal chain — unmarshal each file into the same struct in precedence order.

**Rationale**: The codebase already uses `gopkg.in/yaml.v3` and has only 4 direct CLI dependencies. `yaml.Unmarshal` into an already-populated struct naturally merges — it only overwrites fields present in the YAML document. Fields absent from higher-precedence files retain values from lower-precedence files.

**Alternatives Considered**:
- **viper `MergeConfigMap`**: Rejected — adds 10+ transitive dependencies (mapstructure, fsnotify, pflag, etc.) for ~30 lines of custom code.
- **koanf merge providers**: Rejected — adds ~5 dependencies; cleaner than viper but still overkill for ~10 config keys.
- **JSON Schema + generic map merge**: Rejected — loses type safety; the struct-based approach gives compile-time checking.

**Implementation Pattern**:
```go
func LoadHierarchicalConfig() (*Config, error) {
    cfg := defaultConfig() // hardcoded defaults

    paths := []string{
        "/etc/datakit/config.yaml",                           // system (lowest)
        filepath.Join(homeDir, ".config", "dp", "config.yaml"), // user
        filepath.Join(repoRoot, ".dp", "config.yaml"),         // repo (highest)
    }

    for _, p := range paths {
        data, err := os.ReadFile(p)
        if errors.Is(err, os.ErrNotExist) { continue }
        if err != nil { return nil, fmt.Errorf("reading %s: %w", p, err) }
        if err := yaml.Unmarshal(data, cfg); err != nil {
            return nil, fmt.Errorf("parsing %s: %w", p, err)
        }
    }
    return cfg, nil
}
```

**Limitation**: Map fields (like `Overrides map[string]PluginOverride`) at a higher scope replace the entire map, not per-key. This is acceptable because plugin overrides at repo scope intentionally override all user-scope overrides. Mirror lists are merged separately with deduplication.

---

## R-002: Git Repository Root Detection

**Question**: How to find the Git repo root for `.dp/config.yaml` lookup?

**Decision**: Shell out to `git rev-parse --show-toplevel`.

**Rationale**: The codebase already shells out to `git` extensively (sparse clones in `run.go`), so there is no philosophical objection. This correctly handles worktrees, submodules, and `$GIT_DIR` overrides.

**Alternatives Considered**:
- **go-git library**: Rejected — adds ~40 transitive dependencies (SSH, crypto libs) for a single `rev-parse` call.
- **Walk up directories looking for `.git/`**: Rejected — misses worktrees (where `.git` is a file, not a directory) and `$GIT_DIR` overrides.

**Edge Cases**:
1. **Not in a git repo**: `git rev-parse` exits code 128. Return empty string — repo-level config is silently skipped with an optional debug log.
2. **Git not installed**: `exec.Command` fails. Treat as "no repo config" — this is a soft dependency (config lookup, not a hard prerequisite).
3. **Bare repository**: `--show-toplevel` errors. Not relevant — users don't run `dp` in bare repos.
4. **Worktrees**: Returns worktree root (correct — config lives with the worktree).

---

## R-003: OCI Image Pull and k3d Import Flow

**Question**: How to get a `ghcr.io` destination plugin image running as a pod in k3d?

**Decision**: `docker pull` → `k3d image import` → `kubectl run` with `imagePullPolicy: Never`.

**Rationale**: This is identical to the existing source plugin flow in `run.go` (`importImageToK3d` function). Consistency reduces cognitive overhead and code complexity.

**Flow**:
```
1. docker pull ghcr.io/infobloxopen/cloudquery-plugin-<name>:<version>
2. k3d image import <full-image-ref> --cluster <cluster-name>
3. kubectl run cq-dest-<name> --image=<full-image-ref> --image-pull-policy=Never \
     --port=7777 --env=CQ_PLUGIN_ADDRESS=[::]:7777
4. kubectl port-forward pod/cq-dest-<name> <local-port>:7777
```

**Alternatives Considered**:
- **k3d registry mirror config**: Rejected for MVP — requires cluster reconfiguration at creation time and auth setup. Can be added later as an optimization.
- **Let k3s pull directly** (no import, `imagePullPolicy: Always`): Rejected — requires network access from inside k3d, may need `imagePullSecret` for private images, and is less deterministic than import.
- **Extract binary from image** (docker create + docker cp): Rejected — more complex, and deploying as a pod is cleaner (consistent with source plugin pattern, enables resource isolation).

**Caching**: After `docker pull`, the image exists in the local Docker daemon. Subsequent `k3d image import` calls for the same tag are fast (k3d checks if the image is already in containerd). We can also check `docker image inspect` before pulling to skip the network call entirely.

---

## R-004: CloudQuery Sync Config for Two gRPC Pods

**Question**: What does the sync YAML look like when both source and destination run as remote gRPC servers (in pods)?

**Decision**: Both use `registry: grpc` with `path: localhost:<port>`.

**Rationale**: The existing code already uses `registry: grpc` for the source plugin. Extending this to the destination is a drop-in change. The `path` field for `registry: grpc` is a gRPC address (not a filesystem path).

**Sync Config Example**:
```yaml
kind: source
spec:
  name: "my-source"
  registry: grpc
  path: "localhost:7777"
  tables: ["*"]
  destinations: ["my-destination"]
  spec: {}
---
kind: destination
spec:
  name: "my-destination"
  registry: grpc
  path: "localhost:7778"
  spec:
    connection_string: "postgresql://user:pass@localhost:5432/mydb"
```

**Key Details**:
- `---` YAML document separator between source and destination (existing pattern in `generateSyncConfig`).
- `destinations: ["my-destination"]` in source must match `name: "my-destination"` in destination exactly.
- `spec: {}` under destination holds plugin-specific config (connection_string for PG, bucket for S3, etc.).
- Port 7777 for source, 7778 for destination (avoid collision when both are port-forwarded).

**Change from Current**: Current code uses `registry: local` with a filesystem path for destination binary. New code uses `registry: grpc` with a localhost address for the port-forwarded pod.

---

## R-005: Config Validation Pattern

**Question**: How to validate config values — at write time, read time, or both?

**Decision**: Validate at both write (`dp config set`) and read (`LoadHierarchicalConfig`) time.

**Rationale**: Write-time validation gives instant feedback when the user sets a value. Read-time validation catches hand-edited configs and cross-field constraints.

**Alternatives Considered**:
- **Write-only validation**: Rejected — doesn't catch hand-edited configs or configs from other scopes.
- **Read-only validation**: Rejected — bad UX — user types `dp config set` and gets no feedback until they run `dp run` minutes later.
- **JSON Schema validation**: Rejected — project uses JSON Schema for manifests but for ~10 config keys, a `Validate()` method gives better error messages with less complexity.
- **go-playground/validator tags**: Rejected — adds a large dependency with reflection overhead for a simple case.

**Implementation Pattern**:
```go
// Full struct validation (called after LoadHierarchicalConfig)
func (c *Config) Validate() error {
    var errs []string
    if c.Plugins.Registry != "" {
        if !isValidRegistryURL(c.Plugins.Registry) {
            errs = append(errs, fmt.Sprintf("plugins.registry: invalid registry URL %q", c.Plugins.Registry))
        }
    }
    // ... more field checks ...
    if len(errs) > 0 {
        return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(errs, "\n  - "))
    }
    return nil
}

// Single-field validation (called by dp config set)
func ValidateField(key, value string) error { ... }
```

**Error Message Pattern**: Always include field path (`plugins.registry`), invalid value (`"not://valid"`), and allowed format ("must be a valid registry URL like `ghcr.io/org`").

---

## R-006: Backward Compatibility with Existing Config

**Question**: How to extend the existing `Config` struct without breaking `dp dev`?

**Decision**: Add a new `Plugins` section alongside the existing `Dev` section. The `yaml` struct tags ensure existing config files are parsed without error (unknown fields are ignored by `yaml.v3`, and missing fields get zero values).

**Current Config Struct** (sdk/localdev/config.go):
```go
type Config struct {
    Dev DevConfig `yaml:"dev"`
}
```

**Extended Config Struct**:
```go
type Config struct {
    Dev     DevConfig     `yaml:"dev"`
    Plugins PluginsConfig `yaml:"plugins"`
}

type PluginsConfig struct {
    Registry  string                     `yaml:"registry"`
    Mirrors   []string                   `yaml:"mirrors"`
    Overrides map[string]PluginOverride  `yaml:"overrides"`
}

type PluginOverride struct {
    Version string `yaml:"version"`
    Image   string `yaml:"image"`
}
```

**Backward Compatibility Proof**:
- Existing config files have only a `dev:` section. When parsed into the new struct, `Plugins` gets its zero value (`PluginsConfig{}`). Code checks for zero values and falls back to built-in defaults.
- Existing `LoadConfig()` / `SaveConfig()` functions continue to work. `SaveConfig` will write the `plugins:` section only if it's non-zero (yaml.v3 omits zero-valued fields when using `omitempty`).
- The existing `DefaultConfigPath()`, `LoadConfigFromPath()`, and `SaveConfigToPath()` functions are preserved. New `LoadHierarchicalConfig()` wraps the existing load pattern with multi-scope support.

---

## Summary

| Research Item | Status | Decision | New Dependencies |
|---------------|--------|----------|-----------------|
| R-001: Hierarchical merge | ✅ Resolved | Custom yaml.v3 unmarshal chain | None |
| R-002: Git repo root | ✅ Resolved | `git rev-parse --show-toplevel` | None |
| R-003: OCI pull + k3d import | ✅ Resolved | `docker pull` → `k3d image import` → pod with `imagePullPolicy: Never` | None |
| R-004: Dual gRPC sync config | ✅ Resolved | Both `registry: grpc`, different ports | None |
| R-005: Config validation | ✅ Resolved | `Validate()` method + `ValidateField()` for single keys | None |
| R-006: Backward compatibility | ✅ Resolved | Additive `Plugins` section; yaml.v3 ignores unknown/missing fields | None |

**All NEEDS CLARIFICATION items resolved. Zero new dependencies required.**
