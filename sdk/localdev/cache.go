// Package localdev provides utilities for local development environment management.
// This file implements the Docker registry pull-through cache for k3d local development.
package localdev

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Cache-related constants.
const (
	// DefaultContainerName is the name of the cache container.
	DefaultContainerName = "dev-registry-cache"

	// DefaultVolumeName is the name of the Docker volume for cached layers.
	DefaultVolumeName = "dev_registry_cache"

	// DefaultNetworkName is the Docker network for cache connectivity.
	DefaultNetworkName = "devcache"

	// DefaultPort is the host port for the registry.
	// Using 5050 instead of 5000 to avoid conflict with macOS AirPlay Receiver.
	DefaultPort = 5050

	// DefaultCacheDir is the directory for config files.
	DefaultCacheDir = ".cache"

	// DefaultRemoteURL is the upstream registry to proxy.
	DefaultRemoteURL = "https://registry-1.docker.io"

	// RegistryImage is the Docker image for the registry.
	RegistryImage = "registry:2"
)

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
type CacheStatus struct {
	Exists     bool   // Container exists
	Running    bool   // Container is running
	ConfigHash string // Current configuration hash
	Endpoint   string // Registry endpoint URL
	VolumeSize string // Approximate cache size (if available)
}

// RegistryConfig represents the registry configuration for pull-through caching.
type RegistryConfig struct {
	Version string        `yaml:"version"`
	Log     LogConfig     `yaml:"log"`
	Storage StorageConfig `yaml:"storage"`
	HTTP    HTTPConfig    `yaml:"http"`
	Proxy   ProxyConfig   `yaml:"proxy"`
}

// LogConfig configures registry logging.
type LogConfig struct {
	Level string `yaml:"level"`
}

// StorageConfig configures registry storage.
type StorageConfig struct {
	Filesystem FilesystemConfig `yaml:"filesystem"`
	Delete     DeleteConfig     `yaml:"delete"`
}

// FilesystemConfig configures filesystem storage.
type FilesystemConfig struct {
	RootDirectory string `yaml:"rootdirectory"`
}

// DeleteConfig configures layer deletion.
type DeleteConfig struct {
	Enabled bool `yaml:"enabled"`
}

// HTTPConfig configures the registry HTTP server.
type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

// ProxyConfig configures the upstream registry proxy.
type ProxyConfig struct {
	RemoteURL string `yaml:"remoteurl"`
}

// RegistriesYAML represents the k3d registries configuration.
type RegistriesYAML struct {
	Mirrors map[string]RegistryMirror `yaml:"mirrors"`
}

// RegistryMirror represents a registry mirror configuration.
type RegistryMirror struct {
	Endpoint []string `yaml:"endpoint"`
}

// CacheManager manages the Docker registry pull-through cache for local development.
type CacheManager struct {
	containerName string
	volumeName    string
	networkName   string
	port          int
	cacheDir      string
	mirrorHost    string
}

// CacheOption is a functional option for configuring CacheManager.
type CacheOption func(*CacheConfig)

// WithMirrorHost overrides the auto-detected mirror host.
func WithMirrorHost(host string) CacheOption {
	return func(c *CacheConfig) {
		c.MirrorHost = host
	}
}

// WithPort sets a custom port for the registry.
func WithPort(port int) CacheOption {
	return func(c *CacheConfig) {
		c.Port = port
	}
}

// WithCacheDir sets a custom directory for config files.
func WithCacheDir(dir string) CacheOption {
	return func(c *CacheConfig) {
		c.CacheDir = dir
	}
}

// NewCacheManager creates a new CacheManager with the given options.
func NewCacheManager(opts ...CacheOption) (*CacheManager, error) {
	cfg := &CacheConfig{
		ContainerName: DefaultContainerName,
		VolumeName:    DefaultVolumeName,
		NetworkName:   DefaultNetworkName,
		Port:          DefaultPort,
		CacheDir:      DefaultCacheDir,
		MirrorHost:    detectMirrorHost(),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &CacheManager{
		containerName: cfg.ContainerName,
		volumeName:    cfg.VolumeName,
		networkName:   cfg.NetworkName,
		port:          cfg.Port,
		cacheDir:      cfg.CacheDir,
		mirrorHost:    cfg.MirrorHost,
	}, nil
}

// detectMirrorHost determines the host endpoint for k3d to reach the cache.
// Priority: DEV_REGISTRY_MIRROR_HOST env var > host.k3d.internal (default).
func detectMirrorHost() string {
	if host := os.Getenv("DEV_REGISTRY_MIRROR_HOST"); host != "" {
		return host
	}
	// k3d handles DNS resolution for host.k3d.internal internally
	return "host.k3d.internal"
}

// configHash computes a SHA256 hash of the given data.
func configHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

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

// Endpoint returns the registry endpoint URL for k3d configuration.
func (m *CacheManager) Endpoint() string {
	return fmt.Sprintf("http://%s:%d", m.mirrorHost, m.port)
}

// GetRegistriesYAMLPath returns the path to the k3d registries config.
// Returns empty string if CI or cache not configured.
func (m *CacheManager) GetRegistriesYAMLPath() string {
	if IsCI() {
		return ""
	}
	return filepath.Join(m.cacheDir, "registries.yaml")
}

// ensureCacheDir creates the .cache directory if it doesn't exist.
func (m *CacheManager) ensureCacheDir() error {
	return os.MkdirAll(m.cacheDir, 0755)
}

// writeRegistryConfig writes the registry configuration file.
func (m *CacheManager) writeRegistryConfig() ([]byte, error) {
	config := RegistryConfig{
		Version: "0.1",
		Log: LogConfig{
			Level: "info",
		},
		Storage: StorageConfig{
			Filesystem: FilesystemConfig{
				RootDirectory: "/var/lib/registry",
			},
			Delete: DeleteConfig{
				Enabled: true,
			},
		},
		HTTP: HTTPConfig{
			// Container internal port is always 5000, host port is m.port
			Addr: ":5000",
		},
		Proxy: ProxyConfig{
			RemoteURL: DefaultRemoteURL,
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal registry config: %w", err)
	}

	configPath := filepath.Join(m.cacheDir, "registry-config.yml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write registry config: %w", err)
	}

	return data, nil
}

// writeRegistriesYAML writes the k3d registries configuration file.
func (m *CacheManager) writeRegistriesYAML() error {
	config := RegistriesYAML{
		Mirrors: map[string]RegistryMirror{
			"docker.io": {
				Endpoint: []string{m.Endpoint()},
			},
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal registries config: %w", err)
	}

	configPath := filepath.Join(m.cacheDir, "registries.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registries config: %w", err)
	}

	return nil
}

// containerExists checks if the cache container exists.
func (m *CacheManager) containerExists(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Name}}", m.containerName)
	if err := cmd.Run(); err != nil {
		// Container doesn't exist
		return false, nil
	}
	return true, nil
}

// containerRunning checks if the cache container is running.
func (m *CacheManager) containerRunning(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", m.containerName)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return bytes.TrimSpace(output) != nil && string(bytes.TrimSpace(output)) == "true", nil
}

// getContainerConfigHash reads the config hash label from the container.
func (m *CacheManager) getContainerConfigHash(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect",
		"--format", `{{index .Config.Labels "dev.cache.config_sha256"}}`,
		m.containerName)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

// ensureNetwork creates the Docker network if it doesn't exist.
func (m *CacheManager) ensureNetwork(ctx context.Context) error {
	// Check if network exists
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", m.networkName)
	if cmd.Run() == nil {
		return nil // Network exists
	}

	// Create network
	cmd = exec.CommandContext(ctx, "docker", "network", "create", m.networkName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create network %s: %s", m.networkName, stderr.String())
	}
	return nil
}

// createContainer creates and starts the registry cache container.
func (m *CacheManager) createContainer(ctx context.Context, configHash string) error {
	configPath, err := filepath.Abs(filepath.Join(m.cacheDir, "registry-config.yml"))
	if err != nil {
		return fmt.Errorf("failed to get absolute config path: %w", err)
	}

	args := []string{
		"run", "-d",
		"--name", m.containerName,
		"-p", fmt.Sprintf("%d:5000", m.port),
		"-v", fmt.Sprintf("%s:/var/lib/registry", m.volumeName),
		"-v", fmt.Sprintf("%s:/etc/docker/registry/config.yml:ro", configPath),
		"--network", m.networkName,
		"--label", "dev.capability=cache-registry",
		"--label", "dev.cache.backend=filesystem",
		"--label", "dev.cache.mode=pull-through",
		"--label", "dev.cache.mirror=docker.io",
		"--label", fmt.Sprintf("dev.cache.endpoint=%s", m.Endpoint()),
		"--label", fmt.Sprintf("dev.cache.config_sha256=%s", configHash),
		"--restart", "unless-stopped",
		RegistryImage,
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create container: %s", stderr.String())
	}
	return nil
}

// startContainer starts an existing stopped container.
func (m *CacheManager) startContainer(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "start", m.containerName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container: %s", stderr.String())
	}
	return nil
}

// stopContainer stops the container.
func (m *CacheManager) stopContainer(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", m.containerName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %s", stderr.String())
	}
	return nil
}

// removeContainer removes the container.
func (m *CacheManager) removeContainer(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", m.containerName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container: %s", stderr.String())
	}
	return nil
}

// removeVolume removes the cache volume.
func (m *CacheManager) removeVolume(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "volume", "rm", m.volumeName)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove volume: %s", stderr.String())
	}
	return nil
}

// Up starts the registry cache container.
// Returns nil if cache is already running with matching config.
func (m *CacheManager) Up(ctx context.Context, output io.Writer) error {
	if IsCI() {
		fmt.Fprintln(output, "CI environment detected, skipping registry cache")
		return nil
	}

	// Ensure cache directory exists
	if err := m.ensureCacheDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write registry config and compute hash
	configData, err := m.writeRegistryConfig()
	if err != nil {
		return err
	}
	newHash := configHash(configData)

	// Write k3d registries config
	if err := m.writeRegistriesYAML(); err != nil {
		return err
	}

	// Check if container exists
	exists, err := m.containerExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if exists {
		// Check if it's running
		running, err := m.containerRunning(ctx)
		if err != nil {
			return fmt.Errorf("failed to check container state: %w", err)
		}

		// Check config hash
		existingHash, _ := m.getContainerConfigHash(ctx)

		if existingHash == newHash {
			// Config matches
			if running {
				fmt.Fprintln(output, "Registry cache is already running")
				return nil
			}
			// Start stopped container
			fmt.Fprintln(output, "Starting existing registry cache...")
			return m.startContainer(ctx)
		}

		// Config changed, recreate container
		fmt.Fprintln(output, "Registry cache configuration changed, recreating...")
		if err := m.removeContainer(ctx); err != nil {
			return err
		}
	}

	// Ensure network exists
	if err := m.ensureNetwork(ctx); err != nil {
		return err
	}

	// Create new container
	fmt.Fprintln(output, "Starting registry cache...")
	if err := m.createContainer(ctx, newHash); err != nil {
		return err
	}

	fmt.Fprintf(output, "Registry cache started at %s\n", m.Endpoint())
	return nil
}

// Down stops the registry cache container.
// If removeVolume is true, also removes the cache volume.
func (m *CacheManager) Down(ctx context.Context, removeVol bool, output io.Writer) error {
	if IsCI() {
		return nil
	}

	exists, err := m.containerExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w", err)
	}

	if !exists {
		fmt.Fprintln(output, "Registry cache is not running")
		return nil
	}

	fmt.Fprintln(output, "Stopping registry cache...")
	if err := m.stopContainer(ctx); err != nil {
		// Try to force remove if stop fails
		_ = m.removeContainer(ctx)
	}

	if removeVol {
		fmt.Fprintln(output, "Removing registry cache volume...")
		// Remove container first to release volume
		_ = m.removeContainer(ctx)
		if err := m.removeVolume(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Status returns the current state of the cache.
func (m *CacheManager) Status(ctx context.Context) (*CacheStatus, error) {
	status := &CacheStatus{
		Exists:   false,
		Running:  false,
		Endpoint: m.Endpoint(),
	}

	exists, err := m.containerExists(ctx)
	if err != nil {
		return status, nil
	}
	status.Exists = exists

	if !exists {
		return status, nil
	}

	running, err := m.containerRunning(ctx)
	if err != nil {
		return status, nil
	}
	status.Running = running

	hash, _ := m.getContainerConfigHash(ctx)
	status.ConfigHash = hash

	return status, nil
}
