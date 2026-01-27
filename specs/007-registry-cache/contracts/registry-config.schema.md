# Registry Configuration Schema

**File**: `registry-config.yml`  
**Location**: `.cache/registry-config.yml`  
**Purpose**: Docker Distribution registry configuration for pull-through cache mode

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://platform.data.kit/schemas/registry-config.schema.json",
  "title": "Docker Registry Pull-Through Cache Configuration",
  "description": "Configuration for Docker Distribution registry in pull-through cache mode",
  "type": "object",
  "required": ["version", "storage", "http", "proxy"],
  "properties": {
    "version": {
      "type": "string",
      "const": "0.1",
      "description": "Configuration version (must be 0.1)"
    },
    "log": {
      "type": "object",
      "properties": {
        "level": {
          "type": "string",
          "enum": ["debug", "info", "warn", "error"],
          "default": "info",
          "description": "Log level for the registry"
        }
      }
    },
    "storage": {
      "type": "object",
      "required": ["filesystem"],
      "properties": {
        "filesystem": {
          "type": "object",
          "required": ["rootdirectory"],
          "properties": {
            "rootdirectory": {
              "type": "string",
              "default": "/var/lib/registry",
              "description": "Root directory for cached data"
            }
          }
        },
        "delete": {
          "type": "object",
          "properties": {
            "enabled": {
              "type": "boolean",
              "default": true,
              "description": "Enable layer deletion for garbage collection"
            }
          }
        }
      }
    },
    "http": {
      "type": "object",
      "required": ["addr"],
      "properties": {
        "addr": {
          "type": "string",
          "pattern": "^:[0-9]+$",
          "default": ":5000",
          "description": "Address to listen on (port only, binds all interfaces)"
        }
      }
    },
    "proxy": {
      "type": "object",
      "required": ["remoteurl"],
      "properties": {
        "remoteurl": {
          "type": "string",
          "format": "uri",
          "default": "https://registry-1.docker.io",
          "description": "Upstream registry URL to proxy (Docker Hub)"
        }
      }
    }
  }
}
```

## Example

```yaml
version: 0.1
log:
  level: info
storage:
  filesystem:
    rootdirectory: /var/lib/registry
  delete:
    enabled: true
http:
  addr: :5000
proxy:
  remoteurl: https://registry-1.docker.io
```

## Notes

- This is the standard Docker Distribution v2 configuration format
- The `proxy` section enables pull-through cache mode
- Storage path `/var/lib/registry` is mounted as a Docker volume
- Port 5000 is the conventional registry port
