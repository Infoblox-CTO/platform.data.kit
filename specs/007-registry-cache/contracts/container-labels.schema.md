# Container Labels Contract

**Resource**: Docker Container `dev-registry-cache`  
**Purpose**: Identification and state tracking for idempotent operations

## Label Schema

| Label | Type | Required | Description |
|-------|------|----------|-------------|
| `dev.capability` | string | Yes | Identifies container purpose: "cache-registry" |
| `dev.cache.backend` | string | Yes | Storage backend type: "filesystem" |
| `dev.cache.mode` | string | Yes | Cache operation mode: "pull-through" |
| `dev.cache.mirror` | string | Yes | Registry being mirrored: "docker.io" |
| `dev.cache.endpoint` | string | Yes | Full endpoint URL for k3d access |
| `dev.cache.config_sha256` | string | Yes | SHA256 hash of registry-config.yml |

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://platform.data.kit/schemas/cache-container-labels.schema.json",
  "title": "Cache Container Labels",
  "description": "Labels applied to the registry cache container",
  "type": "object",
  "required": [
    "dev.capability",
    "dev.cache.backend",
    "dev.cache.mode",
    "dev.cache.mirror",
    "dev.cache.endpoint",
    "dev.cache.config_sha256"
  ],
  "properties": {
    "dev.capability": {
      "type": "string",
      "const": "cache-registry",
      "description": "Identifies this as a cache registry container"
    },
    "dev.cache.backend": {
      "type": "string",
      "enum": ["filesystem", "s3", "azure", "gcs"],
      "default": "filesystem",
      "description": "Storage backend for cached data"
    },
    "dev.cache.mode": {
      "type": "string",
      "enum": ["pull-through", "push", "standalone"],
      "default": "pull-through",
      "description": "Registry operation mode"
    },
    "dev.cache.mirror": {
      "type": "string",
      "default": "docker.io",
      "description": "Upstream registry being mirrored"
    },
    "dev.cache.endpoint": {
      "type": "string",
      "format": "uri",
      "description": "Full endpoint URL (e.g., http://host.k3d.internal:5000)"
    },
    "dev.cache.config_sha256": {
      "type": "string",
      "pattern": "^[a-f0-9]{64}$",
      "description": "SHA256 hash of the registry configuration file"
    }
  }
}
```

## Example

```bash
docker run -d \
  --name dev-registry-cache \
  --label dev.capability=cache-registry \
  --label dev.cache.backend=filesystem \
  --label dev.cache.mode=pull-through \
  --label dev.cache.mirror=docker.io \
  --label dev.cache.endpoint=http://host.k3d.internal:5000 \
  --label dev.cache.config_sha256=abc123...def456 \
  registry:2
```

## Usage

### Check if container exists with matching config

```bash
# Get existing config hash
docker inspect --format '{{index .Config.Labels "dev.cache.config_sha256"}}' dev-registry-cache

# Compare with new config hash to determine if recreate needed
```

### Find cache container by capability

```bash
docker ps --filter "label=dev.capability=cache-registry" --format "{{.Names}}"
```

## Notes

- Labels persist across container stop/start cycles
- Config hash enables idempotent operations (recreate only if config changed)
- Endpoint label documents runtime configuration for debugging
