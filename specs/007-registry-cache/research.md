# Research: Registry Pull-Through Cache for k3d Local Development

**Feature**: 007-registry-cache  
**Phase**: 0 - Outline & Research  
**Date**: 2026-01-28

## Research Tasks Identified

From Technical Context unknowns and dependencies:
1. Docker registry pull-through cache configuration
2. k3d registry mirror configuration format
3. Host endpoint detection across platforms (macOS/Linux)
4. CI environment detection patterns
5. Docker CLI exec patterns in existing codebase

---

## 1. Docker Registry Pull-Through Cache Configuration

### Decision
Use the official `registry:2` image with proxy configuration to act as a pull-through cache for Docker Hub.

### Rationale
- Official Docker registry image with built-in proxy/mirror support
- Well-documented configuration format
- Lightweight and reliable
- Already used widely in enterprise environments

### Configuration Format
```yaml
# registry-config.yml
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

### Key Points
- `proxy.remoteurl` enables pull-through caching mode
- Storage at `/var/lib/registry` maps to Docker volume
- Port 5000 is the standard registry port
- `delete.enabled: true` allows garbage collection

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|--------------|
| Harbor | Overkill for single-developer use; complex setup |
| Nexus | Heavier footprint; requires Java |
| Artifactory | Commercial; licensing concerns |
| Docker Hub rate limit workarounds | Doesn't solve cluster recreation problem |

---

## 2. k3d Registry Mirror Configuration

### Decision
Use k3d's `--registry-config` flag with a `registries.yaml` file to configure containerd mirrors.

### Rationale
- k3d natively supports registry configuration
- Standard containerd registries.yaml format
- No modifications to k3d cluster after creation needed

### Configuration Format
```yaml
# registries.yaml
mirrors:
  docker.io:
    endpoint:
      - "http://host.k3d.internal:5000"
```

### Key Points
- `mirrors` section defines registry mirrors by registry name
- `docker.io` is the canonical name for Docker Hub
- Endpoint must be accessible from within k3d containers
- HTTP (not HTTPS) since cache is local and trusted

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|--------------|
| k3d embedded registry | Doesn't persist across cluster deletions |
| Modify containerd.toml directly | Requires cluster restart; fragile |
| Use k3d --registry-use | Designed for push, not pull-through |

---

## 3. Host Endpoint Detection (Cross-Platform)

### Decision
Use cascading detection: `host.k3d.internal` → `host.docker.internal` → environment variable override.

### Rationale
- `host.k3d.internal` is k3d's recommended hostname for host access
- `host.docker.internal` is Docker Desktop's hostname (macOS/Windows/WSL2)
- Environment variable allows advanced users to override

### Detection Logic
```go
func detectMirrorHost() string {
    // 1. Check environment override
    if host := os.Getenv("DEV_REGISTRY_MIRROR_HOST"); host != "" {
        return host
    }
    
    // 2. Prefer k3d's special hostname
    if canResolve("host.k3d.internal") {
        return "host.k3d.internal"
    }
    
    // 3. Fall back to Docker Desktop hostname
    return "host.docker.internal"
}
```

### Platform Behavior
| Platform | Primary Hostname | Fallback |
|----------|-----------------|----------|
| macOS (Docker Desktop) | host.k3d.internal | host.docker.internal |
| macOS (Rancher Desktop) | host.k3d.internal | host.docker.internal |
| Linux (Docker native) | host.k3d.internal | host.docker.internal |
| Linux (Docker Desktop) | host.k3d.internal | host.docker.internal |

### Key Points
- Don't test connectivity at startup—just use the hostname
- k3d handles DNS resolution for `host.k3d.internal` internally
- Validation happens when k3d tries to pull images

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|--------------|
| Use container IP directly | Changes on container restart |
| Shared Docker network | Adds complexity; overkill for single container |
| Host networking mode | Security concerns; port conflicts |

---

## 4. CI Environment Detection

### Decision
Check standard CI environment variables to skip cache operations.

### Rationale
- Well-established convention across CI systems
- No false positives in developer environments
- Simple boolean check before cache operations

### Detection Logic
```go
func isCI() bool {
    // CI=true is widely used (GitHub Actions, GitLab, CircleCI, etc.)
    if os.Getenv("CI") == "true" {
        return true
    }
    
    // GitHub Actions specific
    if os.Getenv("GITHUB_ACTIONS") == "true" {
        return true
    }
    
    // Jenkins specific (URL is always set)
    if os.Getenv("JENKINS_URL") != "" {
        return true
    }
    
    return false
}
```

### CI Systems Covered
| System | Detection Variable |
|--------|-------------------|
| GitHub Actions | `GITHUB_ACTIONS=true` or `CI=true` |
| GitLab CI | `CI=true` |
| CircleCI | `CI=true` |
| Jenkins | `JENKINS_URL` non-empty |
| Travis CI | `CI=true` |
| Azure Pipelines | `CI=true` |

### Alternatives Considered
| Alternative | Why Rejected |
|-------------|--------------|
| Check for TTY | Unreliable; scripts may not have TTY |
| Explicit --no-cache flag | Extra burden on CI configuration |
| Check container runtime | CI may use same runtime as local dev |

---

## 5. Docker CLI Exec Patterns (Existing Codebase)

### Decision
Follow the established `exec.CommandContext` pattern used in `k3d.go`.

### Rationale
- Consistency with existing codebase
- No additional dependencies
- Proven reliable in production

### Pattern from k3d.go
```go
// Example from existing codebase
func (m *K3dManager) clusterExists(ctx context.Context) (bool, error) {
    cmd := exec.CommandContext(ctx, "k3d", "cluster", "list", "--output", "json")
    output, err := cmd.Output()
    if err != nil {
        return false, err
    }
    // Parse JSON output...
}
```

### Docker Commands Needed
```go
// Check container exists and state
docker inspect --format '{{.State.Running}}' dev-registry-cache

// Check container labels
docker inspect --format '{{index .Config.Labels "dev.cache.config_sha256"}}' dev-registry-cache

// Create/start container
docker run -d --name dev-registry-cache \
    -p 5000:5000 \
    -v dev_registry_cache:/var/lib/registry \
    -v .cache/registry-config.yml:/etc/docker/registry/config.yml:ro \
    --network devcache \
    --label dev.capability=cache-registry \
    --label dev.cache.config_sha256=<hash> \
    registry:2

// Stop container
docker stop dev-registry-cache

// Remove container
docker rm dev-registry-cache

// Create network
docker network create devcache

// Remove volume
docker volume rm dev_registry_cache
```

### Error Handling Pattern
```go
var stderr bytes.Buffer
cmd := exec.CommandContext(ctx, "docker", args...)
cmd.Stderr = &stderr

if err := cmd.Run(); err != nil {
    return fmt.Errorf("docker command failed: %s: %w", stderr.String(), err)
}
```

---

## 6. Container Labels for State Management

### Decision
Use container labels to track configuration state and enable idempotent operations.

### Rationale
- Labels persist with container
- Can be queried via `docker inspect`
- Enables config-hash-based change detection

### Label Schema
```yaml
dev.capability: cache-registry      # Identifies this as the cache container
dev.cache.backend: filesystem       # Storage backend type
dev.cache.mode: pull-through        # Cache mode
dev.cache.mirror: docker.io         # Registry being mirrored
dev.cache.endpoint: host.k3d.internal:5000  # Computed endpoint
dev.cache.config_sha256: <hash>     # SHA256 of registry-config.yml
```

### Config Hash Calculation
```go
import "crypto/sha256"

func configHash(configData []byte) string {
    h := sha256.Sum256(configData)
    return fmt.Sprintf("%x", h)
}
```

### Idempotency Logic
```go
func needsRecreate(ctx context.Context, newConfigHash string) (bool, error) {
    existingHash, err := getContainerLabel(ctx, "dev-registry-cache", "dev.cache.config_sha256")
    if err != nil {
        return true, nil // Container doesn't exist
    }
    return existingHash != newConfigHash, nil
}
```

---

## Summary of Decisions

| Topic | Decision | Key Artifact |
|-------|----------|--------------|
| Cache Image | `registry:2` with proxy config | registry-config.yml |
| k3d Integration | `--registry-config` flag | registries.yaml |
| Host Detection | Cascading: k3d → docker → env | `detectMirrorHost()` |
| CI Detection | Standard env vars (CI, GITHUB_ACTIONS, JENKINS_URL) | `isCI()` |
| Docker Exec | Follow k3d.go patterns | exec.CommandContext |
| State Management | Container labels with config hash | docker inspect |

## Unresolved Items

None. All NEEDS CLARIFICATION items have been resolved through research.
