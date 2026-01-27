# Quickstart: Registry Pull-Through Cache for k3d Local Development

**Feature**: 007-registry-cache  
**Audience**: Developers implementing this feature

## Overview

This feature adds a Docker registry pull-through cache to the local development environment. The cache automatically starts before the k3d cluster and persists images across cluster recreations.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                         Host Machine                              │
│  ┌────────────┐     ┌─────────────────────┐     ┌──────────────┐ │
│  │ dp dev up  │────▶│ dev-registry-cache  │     │  Docker Hub  │ │
│  └────────────┘     │ (registry:2)        │◀───▶│              │ │
│        │            │ port 5000           │     └──────────────┘ │
│        │            └─────────────────────┘                      │
│        │                     ▲                                   │
│        ▼                     │                                   │
│  ┌─────────────────────────────────────────────────┐             │
│  │                   k3d Cluster                    │             │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────────────┐  │             │
│  │  │ redpanda│  │localstk │  │ your-pipeline   │  │             │
│  │  └─────────┘  └─────────┘  └─────────────────┘  │             │
│  │         All images pulled via cache ↑           │             │
│  └─────────────────────────────────────────────────┘             │
└──────────────────────────────────────────────────────────────────┘
```

## Implementation Steps

### Step 1: Create CacheManager

Create `sdk/localdev/cache.go` with the core cache management logic:

```go
package localdev

import (
    "context"
    "crypto/sha256"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    
    "gopkg.in/yaml.v3"
)

const (
    DefaultContainerName = "dev-registry-cache"
    DefaultVolumeName    = "dev_registry_cache"
    DefaultNetworkName   = "devcache"
    DefaultPort          = 5000
    DefaultCacheDir      = ".cache"
    RegistryImage        = "registry:2"
)

type CacheManager struct {
    containerName string
    volumeName    string
    networkName   string
    port          int
    cacheDir      string
    mirrorHost    string
}

func NewCacheManager() (*CacheManager, error) {
    return &CacheManager{
        containerName: DefaultContainerName,
        volumeName:    DefaultVolumeName,
        networkName:   DefaultNetworkName,
        port:          DefaultPort,
        cacheDir:      DefaultCacheDir,
        mirrorHost:    detectMirrorHost(),
    }, nil
}
```

### Step 2: Implement CI Detection

```go
// IsCI returns true if running in a CI environment.
func IsCI() bool {
    if os.Getenv("CI") == "true" {
        return true
    }
    if os.Getenv("GITHUB_ACTIONS") == "true" {
        return true
    }
    if os.Getenv("JENKINS_URL") != "" {
        return true
    }
    return false
}
```

### Step 3: Implement Up/Down Methods

```go
func (m *CacheManager) Up(ctx context.Context, output io.Writer) error {
    if IsCI() {
        fmt.Fprintln(output, "CI environment detected, skipping registry cache")
        return nil
    }
    
    // 1. Ensure .cache directory exists
    // 2. Write registry-config.yml
    // 3. Compute config hash
    // 4. Check if container exists with matching hash
    // 5. Create network if needed
    // 6. Start/create container
    // 7. Write registries.yaml for k3d
    
    return nil
}

func (m *CacheManager) Down(ctx context.Context, removeVolume bool, output io.Writer) error {
    if IsCI() {
        return nil
    }
    
    // 1. Stop container
    // 2. Optionally remove volume
    
    return nil
}
```

### Step 4: Integrate with K3dManager

Modify `sdk/localdev/k3d.go` to use the cache:

```go
func (m *K3dManager) createCluster(ctx context.Context, output io.Writer) error {
    args := []string{"cluster", "create", m.clusterName, "--wait", "--timeout", "120s"}
    
    // Add registry config if cache is running
    if registriesPath := m.getRegistriesYAMLPath(); registriesPath != "" {
        args = append(args, "--registry-config", registriesPath)
    }
    
    cmd := exec.CommandContext(ctx, "k3d", args...)
    cmd.Stdout = output
    cmd.Stderr = output
    return cmd.Run()
}
```

### Step 5: Wire into CLI

Modify `cli/cmd/dev.go`:

```go
func runDevUp(cmd *cobra.Command, args []string) error {
    // Start cache before k3d
    cacheManager, _ := localdev.NewCacheManager()
    if err := cacheManager.Up(ctx, os.Stdout); err != nil {
        return fmt.Errorf("failed to start registry cache: %w", err)
    }
    
    // Continue with existing k3d startup...
}

func runDevDown(cmd *cobra.Command, args []string) error {
    // Stop k3d first, then cache
    // ...
    
    cacheManager, _ := localdev.NewCacheManager()
    if err := cacheManager.Down(ctx, devRemoveVolumes, os.Stdout); err != nil {
        return fmt.Errorf("failed to stop registry cache: %w", err)
    }
}
```

## Testing Locally

### Manual Test Flow

```bash
# 1. Start dev environment
dp dev up

# 2. Verify cache container is running
docker ps | grep dev-registry-cache

# 3. Verify k3d is using the cache
k3d cluster list
cat .cache/registries.yaml

# 4. Pull an image to populate cache
kubectl run nginx --image=nginx:latest
kubectl delete pod nginx

# 5. Delete and recreate cluster
dp dev down
dp dev up

# 6. Pull same image - should be fast (from cache)
kubectl run nginx --image=nginx:latest
# Should complete in <5 seconds vs 30+ first time
```

### CI Test Flow

```bash
# Verify cache is skipped in CI
CI=true dp dev up
# Should see "CI environment detected, skipping registry cache"
docker ps | grep dev-registry-cache  # Should be empty
```

## File Locations

| File | Purpose |
|------|---------|
| `sdk/localdev/cache.go` | CacheManager implementation |
| `sdk/localdev/cache_test.go` | Unit tests |
| `.cache/registry-config.yml` | Registry config (runtime) |
| `.cache/registries.yaml` | k3d mirror config (runtime) |

## Key Design Decisions

1. **Docker CLI exec vs Docker API**: Use exec.Command for consistency with k3d.go
2. **Config hash in labels**: Enables idempotent operations
3. **Separate network**: Clean isolation, easy cleanup
4. **Volume preservation**: Default preserves cache; --volumes flag cleans it

## Common Issues

| Issue | Solution |
|-------|----------|
| Port 5000 in use | Check for other registries; use `lsof -i :5000` |
| k3d can't reach cache | Verify host.k3d.internal resolves; set DEV_REGISTRY_MIRROR_HOST |
| Stale cache | Run `dp dev down --volumes` to clear |
