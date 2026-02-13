# Quickstart: Plugin Registry & Configuration

**Feature**: 009-plugin-registry
**Estimated time**: 5 minutes

---

## Prerequisites

- `dp` CLI installed and on your PATH
- Docker running
- k3d cluster running (`dp dev up` or existing `dp-local` cluster)
- A CloudQuery source plugin project (e.g., from `dp init`)

## 1. Check your current configuration

```bash
dp config list
```

Output:
```
KEY                   VALUE                    SOURCE
dev.runtime           k3d                      built-in
dev.k3d.clusterName   dp-dev                   built-in
plugins.registry      ghcr.io/infobloxopen     built-in
```

All values come from built-in defaults. No config files exist yet.

## 2. Run a sync with a destination plugin

From your source plugin project directory:

```bash
dp run . --sync --destination file
```

This will:
1. Build your source plugin container image
2. Pull `ghcr.io/infobloxopen/cloudquery-plugin-file:v5.5.1` from the public registry
3. Import both images into your k3d cluster
4. Deploy source and destination as pods
5. Run `cloudquery sync` connecting both plugins via gRPC
6. Write output to the file destination (default: `/tmp/cq-output/`)

## 3. Try a different destination

```bash
dp run . --sync --destination postgresql
```

This pulls `ghcr.io/infobloxopen/cloudquery-plugin-postgresql:v8.14.1` and syncs to PostgreSQL.

## 4. Configure a custom registry

If your organization hosts plugins in a private registry:

```bash
# Set the default registry
dp config set plugins.registry ghcr.io/myteam

# Verify it took effect
dp config get plugins.registry
# ghcr.io/myteam (source: user)

# Now dp run will pull from your registry
dp run . --sync --destination file
# Pulls: ghcr.io/myteam/cloudquery-plugin-file:v5.5.1
```

## 5. Pin a plugin version

```bash
# Pin PostgreSQL to a specific version
dp config set plugins.overrides.postgresql.version v8.13.0

# Verify
dp config list
# ...
# plugins.overrides.postgresql.version   v8.13.0   user
```

## 6. Use a custom image

For a completely custom destination plugin:

```bash
dp config set plugins.overrides.postgresql.image internal.registry.io/custom-pg:v2.0.0

# This bypasses the naming convention entirely
dp run . --sync --destination postgresql
# Pulls: internal.registry.io/custom-pg:v2.0.0
```

## 7. Add a fallback mirror

```bash
# Add a mirror for resilience
dp config add-mirror ghcr.io/backup-org

# View mirrors
dp config list
# ...
# plugins.mirrors[0]   ghcr.io/backup-org   user
```

If the primary registry is unreachable, `dp run --sync` will automatically try mirrors in order.

## 8. Project-level configuration

Set config for a specific Git repository (checked into version control):

```bash
# Set registry for this project only
dp config set plugins.registry internal.registry.io/data-team --scope repo

# This creates .dp/config.yaml in your repo root
cat .dp/config.yaml
# plugins:
#   registry: internal.registry.io/data-team
```

Project-level settings override user and system settings.

## 9. One-time registry override

Override the registry for a single invocation without changing config:

```bash
dp run . --sync --destination file --registry ghcr.io/other-org
```

## Configuration file locations

| Scope | Path | Priority |
|-------|------|----------|
| Repo | `{git-root}/.dp/config.yaml` | Highest |
| User | `~/.config/dp/config.yaml` | Medium |
| System | `/etc/datakit/config.yaml` | Lowest |

Higher-priority scopes override lower ones. CLI flags override everything.

---

## Next Steps

- See [Configuration Reference](../../docs/reference/configuration.md) for all config keys
- See [CLI Reference](../../docs/reference/cli.md) for full `dp config` command documentation
- Run `dp config --help` for inline help
