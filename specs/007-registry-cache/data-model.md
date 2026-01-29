# Data Model: Registry Pull-Through Cache

**Feature**: 007-registry-cache  
**Phase**: 1 - Design & Contracts  
**Date**: 2026-01-28

## Entity Definitions

### 1. CacheManager

The central orchestrator for registry cache operations.

```go
// CacheManager manages the Docker registry pull-through cache for local development.
type CacheManager struct {
    containerName string           // Container name: "dev-registry-cache"
    volumeName    string           // Volume name: "dev_registry_cache"
    networkName   string           // Network name: "devcache"
    port          int              // Host port: 5000
    cacheDir      string           // Config directory: ".cache"
    mirrorHost    string           // Computed or overridden host endpoint
}
```

**Fields**:
| Field | Type | Description | Default |
|-------|------|-------------|---------|
| containerName | string | Docker container name | "dev-registry-cache" |
| volumeName | string | Docker volume for cached layers | "dev_registry_cache" |
| networkName | string | Docker network for cache connectivity | "devcache" |
| port | int | Host port to expose registry | 5000 |
| cacheDir | string | Directory for config files | ".cache" |
| mirrorHost | string | Host endpoint for k3d to reach cache | (auto-detected) |

**Relationships**:
- Creates/manages → RegistryConfig
- Creates/manages → RegistriesYAML
- Owns → Docker container, volume, network

---

### 2. RegistryConfig

Configuration for the Docker Distribution registry in pull-through cache mode.

```go
// RegistryConfig represents the registry configuration for pull-through caching.
type RegistryConfig struct {
    Version string        `yaml:"version"`
    Log     LogConfig     `yaml:"log"`
    Storage StorageConfig `yaml:"storage"`
    HTTP    HTTPConfig    `yaml:"http"`
    Proxy   ProxyConfig   `yaml:"proxy"`
}

type LogConfig struct {
    Level string `yaml:"level"`
}

type StorageConfig struct {
    Filesystem FilesystemConfig `yaml:"filesystem"`
    Delete     DeleteConfig     `yaml:"delete"`
}

type FilesystemConfig struct {
    RootDirectory string `yaml:"rootdirectory"`
}

type DeleteConfig struct {
    Enabled bool `yaml:"enabled"`
}

type HTTPConfig struct {
    Addr string `yaml:"addr"`
}

type ProxyConfig struct {
    RemoteURL string `yaml:"remoteurl"`
}
```

**Validation Rules**:
- `Version` must be "0.1"
- `Proxy.RemoteURL` must be valid HTTPS URL
- `HTTP.Addr` must be valid port binding (":5000")

**State Transitions**: N/A (configuration is immutable once written)

---

### 3. RegistriesYAML

k3d/containerd configuration for registry mirrors.

```go
// RegistriesYAML represents the k3d registries configuration.
type RegistriesYAML struct {
    Mirrors map[string]RegistryMirror `yaml:"mirrors"`
}

type RegistryMirror struct {
    Endpoint []string `yaml:"endpoint"`
}
```

**Validation Rules**:
- Must have entry for "docker.io"
- Endpoint URLs must be valid HTTP/HTTPS URLs
- Endpoint must be reachable from k3d containers

**State Transitions**: N/A (configuration is immutable once written)

---

### 4. CacheStatus

Runtime status of the registry cache.

```go
// CacheStatus represents the current state of the registry cache.
type CacheStatus struct {
    Exists     bool   // Container exists
    Running    bool   // Container is running
    ConfigHash string // Current configuration hash
    Endpoint   string // Registry endpoint URL
    VolumeSize string // Approximate cache size (if available)
}
```

**Fields**:
| Field | Type | Description |
|-------|------|-------------|
| Exists | bool | Whether container exists (running or stopped) |
| Running | bool | Whether container is currently running |
| ConfigHash | string | SHA256 hash of current registry-config.yml |
| Endpoint | string | Full endpoint URL (e.g., "http://host.k3d.internal:5000") |
| VolumeSize | string | Human-readable volume size (e.g., "1.2GB") |

**State Transitions**:
```
NOT_EXISTS → STOPPED → RUNNING
    ↑           ↓         ↓
    ←←←←←←←←←←←←←←←←←←←←←←
         (via remove)
```

---

### 5. ContainerLabels

Labels applied to the cache container for identification and state tracking.

```go
// CacheLabels defines the labels applied to the registry cache container.
type CacheLabels struct {
    Capability    string `label:"dev.capability"`         // "cache-registry"
    Backend       string `label:"dev.cache.backend"`      // "filesystem"
    Mode          string `label:"dev.cache.mode"`         // "pull-through"
    Mirror        string `label:"dev.cache.mirror"`       // "docker.io"
    Endpoint      string `label:"dev.cache.endpoint"`     // computed endpoint
    ConfigSHA256  string `label:"dev.cache.config_sha256"` // config hash
}
```

**Validation Rules**:
- All labels must be non-empty
- ConfigSHA256 must be valid hex-encoded SHA256 (64 characters)

---

## Entity Relationship Diagram

```
┌─────────────────┐
│  CacheManager   │
├─────────────────┤
│ containerName   │──────────────────────────────────────────┐
│ volumeName      │──────────────────────────────────────┐   │
│ networkName     │───────────────────────────────────┐  │   │
│ port            │                                   │  │   │
│ cacheDir        │                                   │  │   │
│ mirrorHost      │                                   ▼  ▼   ▼
└─────────────────┘                            ┌─────────────────────┐
        │                                      │   Docker Engine     │
        │ creates                              │  ┌───────────────┐  │
        ▼                                      │  │ Container     │  │
┌─────────────────┐                            │  │ dev-registry- │  │
│ RegistryConfig  │──writes to──────────────→  │  │ cache         │  │
├─────────────────┤                            │  └───────────────┘  │
│ .cache/registry-│                            │  ┌───────────────┐  │
│ config.yml      │                            │  │ Volume        │  │
└─────────────────┘                            │  │ dev_registry_ │  │
                                               │  │ cache         │  │
┌─────────────────┐                            │  └───────────────┘  │
│ RegistriesYAML  │                            │  ┌───────────────┐  │
├─────────────────┤                            │  │ Network       │  │
│ .cache/         │                            │  │ devcache      │  │
│ registries.yaml │                            │  └───────────────┘  │
└─────────────────┘                            └─────────────────────┘
        │
        │ passed to
        ▼
┌─────────────────┐
│   K3dManager    │
├─────────────────┤
│ --registry-     │
│ config flag     │
└─────────────────┘
```

---

## Go Types Summary

### New Types in `sdk/localdev/cache.go`

```go
package localdev

// CacheManager manages the Docker registry pull-through cache.
type CacheManager struct { ... }

// CacheConfig holds configuration options for the cache manager.
type CacheConfig struct {
    ContainerName string
    VolumeName    string
    NetworkName   string
    Port          int
    CacheDir      string
    MirrorHost    string // Optional override
}

// CacheStatus represents the current state of the registry cache.
type CacheStatus struct { ... }

// NewCacheManager creates a new CacheManager with the given options.
func NewCacheManager(opts ...CacheOption) (*CacheManager, error)

// CacheOption is a functional option for configuring CacheManager.
type CacheOption func(*CacheConfig)

// WithMirrorHost overrides the auto-detected mirror host.
func WithMirrorHost(host string) CacheOption

// WithPort sets a custom port for the registry.
func WithPort(port int) CacheOption

// WithCacheDir sets a custom directory for config files.
func WithCacheDir(dir string) CacheOption
```

### Method Signatures

```go
// Up starts the registry cache container.
// Returns nil if cache is already running with matching config.
func (m *CacheManager) Up(ctx context.Context, output io.Writer) error

// Down stops the registry cache container.
// If removeVolume is true, also removes the cache volume.
func (m *CacheManager) Down(ctx context.Context, removeVolume bool, output io.Writer) error

// Status returns the current state of the cache.
func (m *CacheManager) Status(ctx context.Context) (*CacheStatus, error)

// IsCI returns true if running in a CI environment.
func IsCI() bool

// GetRegistriesYAMLPath returns the path to the k3d registries config.
// Returns empty string if CI or cache not configured.
func (m *CacheManager) GetRegistriesYAMLPath() string

// Endpoint returns the registry endpoint URL for k3d configuration.
func (m *CacheManager) Endpoint() string
```

---

## Constants

```go
const (
    // DefaultContainerName is the name of the cache container.
    DefaultContainerName = "dev-registry-cache"
    
    // DefaultVolumeName is the name of the Docker volume for cached layers.
    DefaultVolumeName = "dev_registry_cache"
    
    // DefaultNetworkName is the Docker network for cache connectivity.
    DefaultNetworkName = "devcache"
    
    // DefaultPort is the host port for the registry.
    DefaultPort = 5000
    
    // DefaultCacheDir is the directory for config files.
    DefaultCacheDir = ".cache"
    
    // DefaultRemoteURL is the upstream registry to proxy.
    DefaultRemoteURL = "https://registry-1.docker.io"
    
    // RegistryImage is the Docker image for the registry.
    RegistryImage = "registry:2"
)
```

---

## File Artifacts

### Runtime Generated Files

| File | Location | Purpose |
|------|----------|---------|
| registry-config.yml | `.cache/registry-config.yml` | Docker registry configuration |
| registries.yaml | `.cache/registries.yaml` | k3d mirror configuration |

### Docker Resources

| Resource | Name | Purpose |
|----------|------|---------|
| Container | dev-registry-cache | Registry pull-through cache |
| Volume | dev_registry_cache | Cached image layers |
| Network | devcache | Container connectivity |
