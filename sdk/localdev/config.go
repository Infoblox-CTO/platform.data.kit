package localdev

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the dp CLI configuration.
type Config struct {
	Dev DevConfig `yaml:"dev"`
}

// DevConfig represents the dev command configuration.
type DevConfig struct {
	// Runtime is the default runtime to use (compose or k3d).
	Runtime string `yaml:"runtime"`
	// Workspace is the path to the DP workspace.
	Workspace string `yaml:"workspace"`
	// K3d contains k3d-specific configuration.
	K3d K3dConfig `yaml:"k3d"`
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
	return filepath.Join(home, ".config", "dp", "config.yaml")
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
	switch c.Dev.Runtime {
	case "compose", "docker-compose":
		return RuntimeCompose
	default:
		return RuntimeK3d
	}
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
