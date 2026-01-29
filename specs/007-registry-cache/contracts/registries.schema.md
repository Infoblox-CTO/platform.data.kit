# k3d Registries Configuration Schema

**File**: `registries.yaml`  
**Location**: `.cache/registries.yaml`  
**Purpose**: k3d/containerd registry mirror configuration

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://platform.data.kit/schemas/registries.schema.json",
  "title": "k3d Registries Configuration",
  "description": "Containerd registry mirror configuration for k3d clusters",
  "type": "object",
  "required": ["mirrors"],
  "properties": {
    "mirrors": {
      "type": "object",
      "description": "Registry mirrors keyed by registry name",
      "additionalProperties": {
        "$ref": "#/definitions/mirror"
      },
      "properties": {
        "docker.io": {
          "$ref": "#/definitions/mirror",
          "description": "Docker Hub mirror configuration"
        }
      }
    },
    "configs": {
      "type": "object",
      "description": "Registry-specific configurations (auth, TLS, etc.)",
      "additionalProperties": {
        "$ref": "#/definitions/config"
      }
    }
  },
  "definitions": {
    "mirror": {
      "type": "object",
      "required": ["endpoint"],
      "properties": {
        "endpoint": {
          "type": "array",
          "items": {
            "type": "string",
            "format": "uri"
          },
          "minItems": 1,
          "description": "List of mirror endpoints in order of preference"
        }
      }
    },
    "config": {
      "type": "object",
      "properties": {
        "auth": {
          "type": "object",
          "properties": {
            "username": { "type": "string" },
            "password": { "type": "string" },
            "auth": { "type": "string" },
            "identitytoken": { "type": "string" }
          }
        },
        "tls": {
          "type": "object",
          "properties": {
            "ca_file": { "type": "string" },
            "cert_file": { "type": "string" },
            "key_file": { "type": "string" },
            "insecure_skip_verify": { "type": "boolean" }
          }
        }
      }
    }
  }
}
```

## Example (Pull-Through Cache)

```yaml
mirrors:
  docker.io:
    endpoint:
      - "http://host.k3d.internal:5000"
```

## Example (With TLS)

```yaml
mirrors:
  docker.io:
    endpoint:
      - "https://registry.internal:5000"
configs:
  "registry.internal:5000":
    tls:
      insecure_skip_verify: false
      ca_file: "/etc/ssl/certs/ca.crt"
```

## Notes

- This is the standard containerd registries.yaml format
- k3d passes this to containerd via `--registry-config` flag
- Mirrors are tried in order; original registry is fallback
- HTTP endpoints work for local development (no TLS needed)
- `docker.io` is the canonical name for Docker Hub
