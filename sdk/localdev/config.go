package localdev

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/charts"
	"gopkg.in/yaml.v3"
)

// ConfigScope represents a configuration file scope.
type ConfigScope string

const (
	// ScopeRepo is the repository-level config (.dk/config.yaml at git root).
	ScopeRepo ConfigScope = "repo"
	// ScopeUser is the user-level config (~/.config/dk/config.yaml).
	ScopeUser ConfigScope = "user"
	// ScopeSystem is the system-level config (/etc/datakit/config.yaml).
	ScopeSystem ConfigScope = "system"
)

const (
	// DefaultPluginRegistry is the default OCI registry for destination plugin images.
	DefaultPluginRegistry = "ghcr.io/infobloxopen"
)

// DefaultPluginVersions maps plugin short names to their default versions.
var DefaultPluginVersions = map[string]string{
	"file":       "v5.5.1",
	"postgresql": "v8.14.1",
	"s3":         "v7.10.1",
}

// Config represents the dk CLI configuration.
type Config struct {
	Dev     DevConfig     `yaml:"dev"`
	Plugins PluginsConfig `yaml:"plugins,omitempty"`
}

// PluginsConfig holds plugin registry and override settings.
type PluginsConfig struct {
	// Registry is the default OCI registry for plugin images.
	Registry string `yaml:"registry,omitempty"`
	// Mirrors is an ordered list of fallback registries.
	Mirrors []string `yaml:"mirrors,omitempty"`
	// Overrides contains per-plugin version or image overrides.
	Overrides map[string]PluginOverride `yaml:"overrides,omitempty"`
	// Destinations contains per-destination plugin configuration (connection strings, etc.).
	Destinations map[string]DestinationConfig `yaml:"destinations,omitempty"`
}

// DestinationConfig holds configuration for a specific destination plugin.
// These values override auto-detected defaults from the k3d cluster.
type DestinationConfig struct {
	// ConnectionString is the database connection string (for postgresql).
	ConnectionString string `yaml:"connection_string,omitempty"`
	// Bucket is the S3 bucket name (for s3).
	Bucket string `yaml:"bucket,omitempty"`
	// Region is the AWS region (for s3).
	Region string `yaml:"region,omitempty"`
	// Endpoint is the S3 endpoint URL (for s3, e.g., LocalStack).
	Endpoint string `yaml:"endpoint,omitempty"`
	// Path is the output path (for file or s3 key prefix).
	Path string `yaml:"path,omitempty"`
}

// PluginOverride allows overriding the default version or entire image reference for a plugin.
type PluginOverride struct {
	// Version overrides the default version tag (e.g., "v8.13.0").
	// Mutually exclusive with Image.
	Version string `yaml:"version,omitempty"`
	// Image overrides the entire image reference, bypassing registry and naming convention.
	// Mutually exclusive with Version.
	Image string `yaml:"image,omitempty"`
}

// DevConfig represents the dev command configuration.
type DevConfig struct {
	// Runtime is the default runtime to use (k3d).
	Runtime string `yaml:"runtime"`
	// Workspace is the path to the DP workspace.
	Workspace string `yaml:"workspace"`
	// K3d contains k3d-specific configuration.
	K3d K3dConfig `yaml:"k3d"`
	// Charts contains per-chart overrides (version, values).
	Charts map[string]charts.ChartOverride `yaml:"charts,omitempty"`
}

// K3dConfig represents k3d-specific configuration.
type K3dConfig struct {
	// ClusterName is the name of the k3d cluster.
	ClusterName string `yaml:"clusterName"`
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "dk", "config.yaml")
}

// LoadConfig loads the configuration from the default path.
func LoadConfig() (*Config, error) {
	return LoadConfigFromPath(DefaultConfigPath())
}

// LoadConfigFromPath loads the configuration from the specified path.
func LoadConfigFromPath(path string) (*Config, error) {
	config := &Config{
		Dev: DevConfig{
			Runtime: "k3d", // Default to k3d
			K3d: K3dConfig{
				ClusterName: DefaultClusterName,
			},
		},
	}

	if path == "" {
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, return defaults
			return config, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	// Apply defaults for empty values
	if config.Dev.Runtime == "" {
		config.Dev.Runtime = "k3d"
	}
	if config.Dev.K3d.ClusterName == "" {
		config.Dev.K3d.ClusterName = DefaultClusterName
	}

	return config, nil
}

// GetDefaultRuntime returns the configured default runtime.
func (c *Config) GetDefaultRuntime() RuntimeType {
	return RuntimeK3d
}

// SaveConfig saves the configuration to the default path.
func SaveConfig(config *Config) error {
	return SaveConfigToPath(config, DefaultConfigPath())
}

// SaveConfigToPath saves the configuration to the specified path.
func SaveConfigToPath(config *Config, path string) error {
	if path == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// gitRepoRoot returns the root directory of the current git repository.
// Returns empty string if not inside a git repository or git is not installed.
func gitRepoRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RepoConfigPath returns the repo-scope config file path (.dk/config.yaml at git root).
// Returns empty string if not inside a git repository.
func RepoConfigPath() string {
	root := gitRepoRoot()
	if root == "" {
		return ""
	}
	return filepath.Join(root, ".dk", "config.yaml")
}

// UserConfigPath returns the user-scope config file path.
// Respects $XDG_CONFIG_HOME if set, otherwise uses ~/.config/dk/config.yaml.
func UserConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dk", "config.yaml")
	}
	return DefaultConfigPath()
}

// SystemConfigPath returns the system-scope config file path.
func SystemConfigPath() string {
	return "/etc/datakit/config.yaml"
}

// ConfigScopePath returns the config file path for the given scope.
func ConfigScopePath(scope ConfigScope) string {
	switch scope {
	case ScopeRepo:
		return RepoConfigPath()
	case ScopeUser:
		return UserConfigPath()
	case ScopeSystem:
		return SystemConfigPath()
	default:
		return ""
	}
}

// applyDefaults sets default values for zero-value fields in the config.
func applyDefaults(config *Config) {
	if config.Dev.Runtime == "" {
		config.Dev.Runtime = "k3d"
	}
	if config.Dev.K3d.ClusterName == "" {
		config.Dev.K3d.ClusterName = DefaultClusterName
	}
	if config.Plugins.Registry == "" {
		config.Plugins.Registry = DefaultPluginRegistry
	}
}

// LoadHierarchicalConfig loads configuration by merging all three scopes:
// system (lowest precedence) → user → repo (highest precedence).
// Missing files are silently skipped. Returns defaults if no files exist.
func LoadHierarchicalConfig() (*Config, error) {
	paths := []string{
		SystemConfigPath(),
		UserConfigPath(),
		RepoConfigPath(),
	}
	return LoadHierarchicalConfigFromPaths(paths)
}

// LoadHierarchicalConfigFromPaths loads and merges config from the given paths in order.
// Later paths override earlier ones. Missing files are silently skipped.
// This is exported for testing purposes.
func LoadHierarchicalConfigFromPaths(paths []string) (*Config, error) {
	config := &Config{}

	for _, p := range paths {
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading config %s: %w", p, err)
		}
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("parsing config %s: %w", p, err)
		}
	}

	applyDefaults(config)
	return config, nil
}

// knownConfigKeys lists all valid dot-separated config keys.
var knownConfigKeys = map[string]bool{
	"dev.runtime":         true,
	"dev.workspace":       true,
	"dev.k3d.clusterName": true,
	"plugins.registry":    true,
}

// isChartKey checks if a key matches the dynamic pattern dev.charts.<name>.version
// or dev.charts.<name>.values.<path>.
func isChartKey(key string) (chartName, field, valuesPath string, ok bool) {
	parts := strings.Split(key, ".")
	if len(parts) < 4 || parts[0] != "dev" || parts[1] != "charts" {
		return "", "", "", false
	}
	chartName = parts[2]
	field = parts[3]
	switch field {
	case "version":
		if len(parts) == 4 {
			return chartName, "version", "", true
		}
	case "values":
		if len(parts) >= 5 {
			valuesPath = strings.Join(parts[4:], ".")
			return chartName, "values", valuesPath, true
		}
	}
	return "", "", "", false
}

// validRuntimes lists valid values for dev.runtime.
var validRuntimes = map[string]bool{
	"k3d": true,
}

// semverRegex matches semver version strings like v1.2.3.
var semverRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// dnsNameRegex matches DNS-safe names (lowercase alphanumeric + hyphens).
var dnsNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// registryRegex matches valid registry URLs (host/path, optional port).
var registryRegex = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9.]*[a-zA-Z0-9](:[0-9]+)?(/[a-zA-Z0-9._-]+)*$`)

// Validate checks the entire configuration and returns all validation errors.
func (c *Config) Validate() []error {
	var errs []error

	// Validate dev.runtime
	if c.Dev.Runtime != "" && !validRuntimes[c.Dev.Runtime] {
		errs = append(errs, fmt.Errorf("invalid value %q for dev.runtime (allowed: k3d)", c.Dev.Runtime))
	}

	// Validate dev.k3d.clusterName
	if c.Dev.K3d.ClusterName != "" && !dnsNameRegex.MatchString(c.Dev.K3d.ClusterName) {
		errs = append(errs, fmt.Errorf("invalid value %q for dev.k3d.clusterName (must be DNS-safe)", c.Dev.K3d.ClusterName))
	}

	// Validate plugins.registry
	if c.Plugins.Registry != "" && !registryRegex.MatchString(c.Plugins.Registry) {
		errs = append(errs, fmt.Errorf("invalid value %q for plugins.registry (must be a valid registry URL)", c.Plugins.Registry))
	}

	// Validate plugins.mirrors
	for i, m := range c.Plugins.Mirrors {
		if !registryRegex.MatchString(m) {
			errs = append(errs, fmt.Errorf("invalid mirror[%d] %q (must be a valid registry URL)", i, m))
		}
	}

	// Validate plugins.overrides
	for name, o := range c.Plugins.Overrides {
		if o.Version != "" && o.Image != "" {
			errs = append(errs, fmt.Errorf("plugins.overrides.%s: version and image are mutually exclusive", name))
		}
		if o.Version != "" && !semverRegex.MatchString(o.Version) {
			errs = append(errs, fmt.Errorf("invalid version %q for plugins.overrides.%s (must match v0.0.0)", o.Version, name))
		}
	}

	return errs
}

// isOverrideKey checks if a key matches the dynamic pattern plugins.overrides.<name>.version or plugins.overrides.<name>.image.
func isOverrideKey(key string) (pluginName, field string, ok bool) {
	parts := strings.Split(key, ".")
	if len(parts) == 4 && parts[0] == "plugins" && parts[1] == "overrides" && (parts[3] == "version" || parts[3] == "image") {
		return parts[2], parts[3], true
	}
	return "", "", false
}

// validDestinationFields lists fields allowed under plugins.destinations.<name>.
var validDestinationFields = map[string]bool{
	"connection_string": true,
	"bucket":            true,
	"region":            true,
	"endpoint":          true,
	"path":              true,
}

// isDestinationKey checks if a key matches plugins.destinations.<name>.<field>.
func isDestinationKey(key string) (destName, field string, ok bool) {
	parts := strings.Split(key, ".")
	if len(parts) == 4 && parts[0] == "plugins" && parts[1] == "destinations" && validDestinationFields[parts[3]] {
		return parts[2], parts[3], true
	}
	return "", "", false
}

// ValidateField validates a single key-value pair. Returns an error if the key is unknown
// or the value is invalid for the key's type constraints.
func ValidateField(key, value string) error {
	// Check for dynamic override keys first
	if _, field, ok := isOverrideKey(key); ok {
		switch field {
		case "version":
			if !semverRegex.MatchString(value) {
				return fmt.Errorf("invalid version %q (must match v0.0.0)", value)
			}
		case "image":
			if value == "" {
				return fmt.Errorf("image cannot be empty")
			}
		}
		return nil
	}

	// Check for dynamic destination keys
	if _, _, ok := isDestinationKey(key); ok {
		if value == "" {
			return fmt.Errorf("value cannot be empty")
		}
		return nil
	}

	// Check for dynamic chart keys
	if _, field, _, ok := isChartKey(key); ok {
		if field == "version" && value != "" && !semverRegex.MatchString(value) {
			return fmt.Errorf("invalid version %q (must match v0.0.0)", value)
		}
		return nil
	}

	if !knownConfigKeys[key] {
		return fmt.Errorf("unknown config key %q", key)
	}

	switch key {
	case "dev.runtime":
		if !validRuntimes[value] {
			return fmt.Errorf("invalid value %q for dev.runtime (allowed: k3d)", value)
		}
	case "dev.k3d.clusterName":
		if !dnsNameRegex.MatchString(value) {
			return fmt.Errorf("invalid value %q for dev.k3d.clusterName (must be DNS-safe)", value)
		}
	case "plugins.registry":
		if !registryRegex.MatchString(value) {
			return fmt.Errorf("invalid value %q for plugins.registry (must be a valid registry URL)", value)
		}
	}

	return nil
}

// SetField sets a configuration value at the given dot-separated key path.
func (c *Config) SetField(key, value string) error {
	// Handle dynamic override keys
	if pluginName, field, ok := isOverrideKey(key); ok {
		if c.Plugins.Overrides == nil {
			c.Plugins.Overrides = make(map[string]PluginOverride)
		}
		override := c.Plugins.Overrides[pluginName]
		switch field {
		case "version":
			override.Version = value
		case "image":
			override.Image = value
		}
		c.Plugins.Overrides[pluginName] = override
		return nil
	}

	// Handle dynamic destination keys
	if destName, field, ok := isDestinationKey(key); ok {
		if c.Plugins.Destinations == nil {
			c.Plugins.Destinations = make(map[string]DestinationConfig)
		}
		dest := c.Plugins.Destinations[destName]
		switch field {
		case "connection_string":
			dest.ConnectionString = value
		case "bucket":
			dest.Bucket = value
		case "region":
			dest.Region = value
		case "endpoint":
			dest.Endpoint = value
		case "path":
			dest.Path = value
		}
		c.Plugins.Destinations[destName] = dest
		return nil
	}

	// Handle dynamic chart keys (dev.charts.<name>.version / dev.charts.<name>.values.<path>)
	if chartName, field, valuesPath, ok := isChartKey(key); ok {
		if c.Dev.Charts == nil {
			c.Dev.Charts = make(map[string]charts.ChartOverride)
		}
		override := c.Dev.Charts[chartName]
		switch field {
		case "version":
			override.Version = value
		case "values":
			if override.Values == nil {
				override.Values = make(map[string]interface{})
			}
			override.Values[valuesPath] = value
		}
		c.Dev.Charts[chartName] = override
		return nil
	}

	switch key {
	case "dev.runtime":
		c.Dev.Runtime = value
	case "dev.workspace":
		c.Dev.Workspace = value
	case "dev.k3d.clusterName":
		c.Dev.K3d.ClusterName = value
	case "plugins.registry":
		c.Plugins.Registry = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

// UnsetField removes a configuration value at the given dot-separated key path,
// resetting it to its zero value.
func (c *Config) UnsetField(key string) error {
	// Handle dynamic override keys
	if pluginName, field, ok := isOverrideKey(key); ok {
		if c.Plugins.Overrides == nil {
			return nil
		}
		override, exists := c.Plugins.Overrides[pluginName]
		if !exists {
			return nil
		}
		switch field {
		case "version":
			override.Version = ""
		case "image":
			override.Image = ""
		}
		// Remove the entry entirely if both fields are empty
		if override.Version == "" && override.Image == "" {
			delete(c.Plugins.Overrides, pluginName)
		} else {
			c.Plugins.Overrides[pluginName] = override
		}
		return nil
	}

	// Handle dynamic destination keys
	if destName, field, ok := isDestinationKey(key); ok {
		if c.Plugins.Destinations == nil {
			return nil
		}
		dest, exists := c.Plugins.Destinations[destName]
		if !exists {
			return nil
		}
		switch field {
		case "connection_string":
			dest.ConnectionString = ""
		case "bucket":
			dest.Bucket = ""
		case "region":
			dest.Region = ""
		case "endpoint":
			dest.Endpoint = ""
		case "path":
			dest.Path = ""
		}
		// Remove the entry if all fields are empty
		if dest.ConnectionString == "" && dest.Bucket == "" && dest.Region == "" && dest.Endpoint == "" && dest.Path == "" {
			delete(c.Plugins.Destinations, destName)
		} else {
			c.Plugins.Destinations[destName] = dest
		}
		return nil
	}

	// Handle dynamic chart keys
	if chartName, field, valuesPath, ok := isChartKey(key); ok {
		if c.Dev.Charts == nil {
			return nil
		}
		override, exists := c.Dev.Charts[chartName]
		if !exists {
			return nil
		}
		switch field {
		case "version":
			override.Version = ""
		case "values":
			delete(override.Values, valuesPath)
		}
		// Remove entry if empty
		if override.Version == "" && len(override.Values) == 0 {
			delete(c.Dev.Charts, chartName)
		} else {
			c.Dev.Charts[chartName] = override
		}
		return nil
	}

	switch key {
	case "dev.runtime":
		c.Dev.Runtime = ""
	case "dev.workspace":
		c.Dev.Workspace = ""
	case "dev.k3d.clusterName":
		c.Dev.K3d.ClusterName = ""
	case "plugins.registry":
		c.Plugins.Registry = ""
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

// LoadConfigForScope loads the configuration from a specific scope's file.
// Returns a default config if the file doesn't exist.
func LoadConfigForScope(scope ConfigScope) (*Config, error) {
	path := ConfigScopePath(scope)
	if path == "" {
		if scope == ScopeRepo {
			return nil, fmt.Errorf("cannot read repo scope: not inside a git repository")
		}
		return &Config{}, nil
	}
	return LoadConfigFromPath(path)
}

// SaveConfigForScope saves the configuration to a specific scope's file.
func SaveConfigForScope(config *Config, scope ConfigScope) error {
	path := ConfigScopePath(scope)
	if path == "" {
		if scope == ScopeRepo {
			return fmt.Errorf("cannot write to repo scope: not inside a git repository")
		}
		return fmt.Errorf("cannot determine config path for scope %q", scope)
	}
	return SaveConfigToPath(config, path)
}

// GetField retrieves the value of a config field by dot-separated key path.
// Returns the value and whether it was found.
func (c *Config) GetField(key string) (string, bool) {
	// Handle dynamic override keys
	if pluginName, field, ok := isOverrideKey(key); ok {
		if c.Plugins.Overrides == nil {
			return "", false
		}
		override, exists := c.Plugins.Overrides[pluginName]
		if !exists {
			return "", false
		}
		switch field {
		case "version":
			if override.Version != "" {
				return override.Version, true
			}
		case "image":
			if override.Image != "" {
				return override.Image, true
			}
		}
		return "", false
	}

	// Handle dynamic destination keys
	if destName, field, ok := isDestinationKey(key); ok {
		if c.Plugins.Destinations == nil {
			return "", false
		}
		dest, exists := c.Plugins.Destinations[destName]
		if !exists {
			return "", false
		}
		var val string
		switch field {
		case "connection_string":
			val = dest.ConnectionString
		case "bucket":
			val = dest.Bucket
		case "region":
			val = dest.Region
		case "endpoint":
			val = dest.Endpoint
		case "path":
			val = dest.Path
		}
		if val != "" {
			return val, true
		}
		return "", false
	}

	// Handle dynamic chart keys
	if chartName, field, valuesPath, ok := isChartKey(key); ok {
		if c.Dev.Charts == nil {
			return "", false
		}
		override, exists := c.Dev.Charts[chartName]
		if !exists {
			return "", false
		}
		switch field {
		case "version":
			if override.Version != "" {
				return override.Version, true
			}
		case "values":
			if v, ok := override.Values[valuesPath]; ok {
				return fmt.Sprintf("%v", v), true
			}
		}
		return "", false
	}

	switch key {
	case "dev.runtime":
		if c.Dev.Runtime != "" {
			return c.Dev.Runtime, true
		}
	case "dev.workspace":
		if c.Dev.Workspace != "" {
			return c.Dev.Workspace, true
		}
	case "dev.k3d.clusterName":
		if c.Dev.K3d.ClusterName != "" {
			return c.Dev.K3d.ClusterName, true
		}
	case "plugins.registry":
		if c.Plugins.Registry != "" {
			return c.Plugins.Registry, true
		}
	}
	return "", false
}

// EffectiveValue resolves the effective value and source scope for a config key
// by loading all scopes in precedence order.
func EffectiveValue(key string) (value string, source string, err error) {
	// Check dynamic override keys
	if _, _, ok := isOverrideKey(key); !ok {
		// Check dynamic destination keys
		if _, _, ok := isDestinationKey(key); !ok {
			// Check dynamic chart keys
			if _, _, _, ok := isChartKey(key); !ok && !knownConfigKeys[key] {
				return "", "", fmt.Errorf("unknown config key %q", key)
			}
		}
	}

	// Check scopes in order: repo (highest) → user → system → built-in
	scopes := []struct {
		scope ConfigScope
		label string
	}{
		{ScopeRepo, "repo"},
		{ScopeUser, "user"},
		{ScopeSystem, "system"},
	}

	for _, s := range scopes {
		path := ConfigScopePath(s.scope)
		if path == "" {
			continue
		}
		cfg, err := loadRawConfigFromPath(path)
		if err != nil || cfg == nil {
			continue
		}
		if v, ok := cfg.GetField(key); ok {
			return v, s.label, nil
		}
	}

	// Fall back to built-in defaults
	switch key {
	case "dev.runtime":
		return "k3d", "built-in", nil
	case "dev.k3d.clusterName":
		return DefaultClusterName, "built-in", nil
	case "plugins.registry":
		return DefaultPluginRegistry, "built-in", nil
	default:
		return "", "built-in", nil
	}
}

// loadRawConfigFromPath loads a config file without applying defaults.
func loadRawConfigFromPath(path string) (*Config, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return config, nil
}

// AllConfigKeys returns the list of known static config keys.
func AllConfigKeys() []string {
	return []string{
		"dev.runtime",
		"dev.workspace",
		"dev.k3d.clusterName",
		"plugins.registry",
	}
}

// IsValidRegistry returns true if the registry URL is valid.
func IsValidRegistry(registry string) bool {
	return registryRegex.MatchString(registry)
}
