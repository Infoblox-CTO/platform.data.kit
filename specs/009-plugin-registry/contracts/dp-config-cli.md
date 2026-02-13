# CLI Contract: `dp config` Subcommand

**Feature**: 009-plugin-registry
**Date**: 2026-02-13
**Purpose**: Define the command interface for `dp config` and its subcommands

---

## Command Tree

```
dp config
├── set <key> <value> [--scope repo|user|system]
├── get <key>
├── unset <key> [--scope repo|user|system]
├── list [--scope repo|user|system]
├── add-mirror <registry> [--scope repo|user|system]
└── remove-mirror <registry> [--scope repo|user|system]
```

---

## Commands

### `dp config set`

Set a configuration value.

**Synopsis**:
```
dp config set <key> <value> [flags]
```

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| `key` | Yes | Dot-separated config key (e.g., `plugins.registry`, `dev.runtime`) |
| `value` | Yes | Value to set |

**Flags**:
| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | `user` | Config scope to write to: `repo`, `user`, or `system` |

**Behavior**:
1. Validate `key` is a known config key (error if unknown).
2. Validate `value` against the key's type and constraints (error if invalid).
3. Load the config file for the target scope.
4. Set the value at the specified key path.
5. Save the config file.
6. Print confirmation: `Set {key} = {value} (scope: {scope})`.

**Examples**:
```bash
# Set default plugin registry (user scope)
dp config set plugins.registry ghcr.io/myteam

# Set plugin registry for this project
dp config set plugins.registry internal.registry.io --scope repo

# Pin a plugin version
dp config set plugins.overrides.postgresql.version v8.13.0

# Override a plugin image entirely
dp config set plugins.overrides.postgresql.image internal.registry.io/custom-pg:v2.0.0

# Set dev runtime
dp config set dev.runtime compose
```

**Errors**:
- `unknown config key "foo.bar"` — key not in schema
- `invalid value "docker" for dev.runtime (allowed: k3d, compose)` — validation failure
- `cannot write to repo scope: not inside a git repository` — no git root found

---

### `dp config get`

Get the effective value of a configuration key.

**Synopsis**:
```
dp config get <key>
```

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| `key` | Yes | Dot-separated config key |

**Behavior**:
1. Load all config scopes (system → user → repo).
2. Resolve the effective value for the key.
3. Print the value and its source scope.

**Output Format**:
```
{value} (source: {scope})
```

**Examples**:
```bash
$ dp config get plugins.registry
ghcr.io/infobloxopen (source: built-in)

$ dp config get plugins.registry
ghcr.io/myteam (source: user)

$ dp config get dev.runtime
k3d (source: repo)
```

**Errors**:
- `unknown config key "foo.bar"` — key not in schema

---

### `dp config unset`

Remove a configuration value from a specific scope.

**Synopsis**:
```
dp config unset <key> [flags]
```

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| `key` | Yes | Dot-separated config key to remove |

**Flags**:
| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | `user` | Config scope to remove from |

**Behavior**:
1. Load the config file for the target scope.
2. Remove the key from the config.
3. Save the config file.
4. Print: `Unset {key} (scope: {scope})`.

**Examples**:
```bash
# Remove custom registry from user config
dp config unset plugins.registry

# Remove a plugin override
dp config unset plugins.overrides.postgresql.version
```

---

### `dp config list`

List all effective configuration settings.

**Synopsis**:
```
dp config list [flags]
```

**Flags**:
| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | (all) | Show settings from a specific scope only |

**Output Format** (table):
```
KEY                                    VALUE                              SOURCE
dev.runtime                            k3d                                built-in
dev.k3d.clusterName                    dp-local                           user
plugins.registry                       internal.registry.io               repo
plugins.mirrors[0]                     ghcr.io/backup-org                 user
plugins.overrides.postgresql.version   v8.13.0                            repo
```

**Examples**:
```bash
# Show all effective settings
dp config list

# Show only repo-level settings
dp config list --scope repo
```

---

### `dp config add-mirror`

Add a fallback registry mirror.

**Synopsis**:
```
dp config add-mirror <registry> [flags]
```

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| `registry` | Yes | Registry URL to add as a mirror |

**Flags**:
| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | `user` | Config scope to add the mirror to |

**Behavior**:
1. Validate the registry URL.
2. Load the config file for the target scope.
3. Append to the mirrors list (reject duplicates).
4. Save the config file.
5. Print: `Added mirror {registry} (scope: {scope})`.

**Examples**:
```bash
dp config add-mirror ghcr.io/backup-org
dp config add-mirror internal.registry.io --scope repo
```

**Errors**:
- `mirror "ghcr.io/backup-org" already exists` — duplicate
- `invalid registry URL "not a url"` — validation failure

---

### `dp config remove-mirror`

Remove a fallback registry mirror.

**Synopsis**:
```
dp config remove-mirror <registry> [flags]
```

**Arguments**:
| Argument | Required | Description |
|----------|----------|-------------|
| `registry` | Yes | Registry URL to remove |

**Flags**:
| Flag | Default | Description |
|------|---------|-------------|
| `--scope` | `user` | Config scope to remove the mirror from |

**Behavior**:
1. Load the config file for the target scope.
2. Remove the registry from the mirrors list.
3. Save the config file.
4. Print: `Removed mirror {registry} (scope: {scope})`.

**Errors**:
- `mirror "ghcr.io/backup-org" not found in {scope} config` — mirror not in the specified scope

---

## `dp run` Flag Changes

### New Flag

| Flag | Default | Description |
|------|---------|-------------|
| `--registry` | (from config) | Override the plugin registry for this invocation only |

**Behavior**: When `--registry` is set, it overrides `plugins.registry` from all config scopes for this single invocation. Does not persist.

**Example**:
```bash
dp run . --sync --destination postgresql --registry ghcr.io/myteam
```

---

## Valid Config Keys

The following dot-separated keys are recognized by `dp config set/get/unset`:

| Key | Type | Validation |
|-----|------|------------|
| `dev.runtime` | string | Must be `k3d` or `compose` |
| `dev.workspace` | string | Must be a valid path |
| `dev.k3d.clusterName` | string | Must be a valid DNS name |
| `plugins.registry` | string | Must be a valid registry URL |
| `plugins.overrides.<name>.version` | string | Must match `v\d+\.\d+\.\d+` |
| `plugins.overrides.<name>.image` | string | Must be a valid image reference |

**Note**: `plugins.mirrors` is managed via `add-mirror`/`remove-mirror`, not via `set`/`unset`.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (invalid key, validation failure, file I/O error) |
