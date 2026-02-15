package localdev

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()

	if path == "" {
		t.Skip("could not determine home directory")
	}

	// Should contain .config/dp/config.yaml
	if !contains(path, ".config") || !contains(path, "dp") || !contains(path, "config.yaml") {
		t.Errorf("DefaultConfigPath() = %q, expected to contain .config/dp/config.yaml", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestLoadConfigFromPath_NonExistent(t *testing.T) {
	config, err := LoadConfigFromPath("/nonexistent/path/config.yaml")

	if err != nil {
		t.Errorf("LoadConfigFromPath() error = %v, want nil for nonexistent file", err)
	}

	if config == nil {
		t.Fatal("LoadConfigFromPath() returned nil config")
	}

	// Should have defaults
	if config.Dev.Runtime != "k3d" {
		t.Errorf("Dev.Runtime = %q, want 'k3d'", config.Dev.Runtime)
	}

	if config.Dev.K3d.ClusterName != DefaultClusterName {
		t.Errorf("Dev.K3d.ClusterName = %q, want %q", config.Dev.K3d.ClusterName, DefaultClusterName)
	}
}

func TestLoadConfigFromPath_EmptyPath(t *testing.T) {
	config, err := LoadConfigFromPath("")

	if err != nil {
		t.Errorf("LoadConfigFromPath('') error = %v, want nil", err)
	}

	if config == nil {
		t.Fatal("LoadConfigFromPath('') returned nil config")
	}

	// Should have defaults
	if config.Dev.Runtime != "k3d" {
		t.Errorf("Dev.Runtime = %q, want 'k3d'", config.Dev.Runtime)
	}
}

func TestLoadConfigFromPath_ValidConfig(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `dev:
  runtime: k3d
  workspace: /path/to/workspace
  k3d:
    clusterName: my-cluster
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	config, err := LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromPath() error = %v", err)
	}

	if config.Dev.Runtime != "k3d" {
		t.Errorf("Dev.Runtime = %q, want 'k3d'", config.Dev.Runtime)
	}

	if config.Dev.Workspace != "/path/to/workspace" {
		t.Errorf("Dev.Workspace = %q, want '/path/to/workspace'", config.Dev.Workspace)
	}

	if config.Dev.K3d.ClusterName != "my-cluster" {
		t.Errorf("Dev.K3d.ClusterName = %q, want 'my-cluster'", config.Dev.K3d.ClusterName)
	}
}

func TestConfig_GetDefaultRuntime(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
		expect  RuntimeType
	}{
		{
			name:    "compose",
			runtime: "compose",
			expect:  RuntimeCompose,
		},
		{
			name:    "k3d",
			runtime: "k3d",
			expect:  RuntimeK3d,
		},
		{
			name:    "kubernetes alias",
			runtime: "kubernetes",
			expect:  RuntimeK3d,
		},
		{
			name:    "k8s alias",
			runtime: "k8s",
			expect:  RuntimeK3d,
		},
		{
			name:    "empty defaults to k3d",
			runtime: "",
			expect:  RuntimeK3d,
		},
		{
			name:    "unknown defaults to k3d",
			runtime: "unknown",
			expect:  RuntimeK3d,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Dev: DevConfig{
					Runtime: tt.runtime,
				},
			}

			result := config.GetDefaultRuntime()
			if result != tt.expect {
				t.Errorf("GetDefaultRuntime() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestSaveConfigToPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.yaml")

	config := &Config{
		Dev: DevConfig{
			Runtime:   "k3d",
			Workspace: "/my/workspace",
			K3d: K3dConfig{
				ClusterName: "test-cluster",
			},
		},
	}

	err := SaveConfigToPath(config, configPath)
	if err != nil {
		t.Fatalf("SaveConfigToPath() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	// Load and verify
	loaded, err := LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromPath() error = %v", err)
	}

	if loaded.Dev.Runtime != config.Dev.Runtime {
		t.Errorf("loaded Runtime = %q, want %q", loaded.Dev.Runtime, config.Dev.Runtime)
	}

	if loaded.Dev.K3d.ClusterName != config.Dev.K3d.ClusterName {
		t.Errorf("loaded ClusterName = %q, want %q", loaded.Dev.K3d.ClusterName, config.Dev.K3d.ClusterName)
	}
}

func TestSaveConfigToPath_EmptyPath(t *testing.T) {
	config := &Config{}

	err := SaveConfigToPath(config, "")
	if err != nil {
		t.Errorf("SaveConfigToPath('') should not error, got %v", err)
	}
}

func TestLoadConfig_Integration(t *testing.T) {
	// This test uses the real default path but should not fail
	// if config doesn't exist (returns defaults)
	config, err := LoadConfig()

	if err != nil {
		// Only fail if it's not a "file doesn't exist" error
		if !os.IsNotExist(err) {
			t.Errorf("LoadConfig() error = %v", err)
		}
	}

	if config == nil {
		t.Error("LoadConfig() returned nil config")
	}
}

// T004: TestGitRepoRoot
func TestGitRepoRoot(t *testing.T) {
	// We're running inside this git repo, so gitRepoRoot should return a non-empty path
	root := gitRepoRoot()
	if root == "" {
		t.Skip("not running inside a git repository")
	}

	// The root should be a valid directory
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("gitRepoRoot() returned %q which doesn't exist: %v", root, err)
	}
	if !info.IsDir() {
		t.Errorf("gitRepoRoot() returned %q which is not a directory", root)
	}

	// Should contain a .git directory
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Errorf("gitRepoRoot() returned %q which doesn't contain a .git directory", root)
	}
}

// T005: TestLoadHierarchicalConfig
func TestLoadHierarchicalConfig(t *testing.T) {
	tests := []struct {
		name         string
		systemYAML   string
		userYAML     string
		repoYAML     string
		wantRuntime  string
		wantRegistry string
		wantCluster  string
		wantErr      bool
	}{
		{
			name:         "no files use defaults",
			wantRuntime:  "k3d",
			wantRegistry: DefaultPluginRegistry,
			wantCluster:  DefaultClusterName,
		},
		{
			name:         "system only",
			systemYAML:   "dev:\n  runtime: compose\nplugins:\n  registry: ghcr.io/system-org\n",
			wantRuntime:  "compose",
			wantRegistry: "ghcr.io/system-org",
			wantCluster:  DefaultClusterName,
		},
		{
			name:         "user only",
			userYAML:     "dev:\n  runtime: k3d\n  k3d:\n    clusterName: my-cluster\nplugins:\n  registry: ghcr.io/user-org\n",
			wantRuntime:  "k3d",
			wantRegistry: "ghcr.io/user-org",
			wantCluster:  "my-cluster",
		},
		{
			name:         "repo only",
			repoYAML:     "plugins:\n  registry: internal.registry.io/team\n",
			wantRuntime:  "k3d",
			wantRegistry: "internal.registry.io/team",
			wantCluster:  DefaultClusterName,
		},
		{
			name:         "merge all three - repo wins",
			systemYAML:   "dev:\n  runtime: compose\nplugins:\n  registry: ghcr.io/system-org\n",
			userYAML:     "dev:\n  runtime: k3d\n  k3d:\n    clusterName: user-cluster\nplugins:\n  registry: ghcr.io/user-org\n",
			repoYAML:     "plugins:\n  registry: internal.registry.io/repo\n",
			wantRuntime:  "k3d",
			wantRegistry: "internal.registry.io/repo",
			wantCluster:  "user-cluster",
		},
		{
			name:     "invalid yaml returns error",
			userYAML: "this is not: valid: yaml: [",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory structure to simulate scopes
			tmpDir := t.TempDir()
			systemDir := filepath.Join(tmpDir, "system")
			userDir := filepath.Join(tmpDir, "user", "dp")
			repoDir := filepath.Join(tmpDir, "repo", ".dp")

			os.MkdirAll(systemDir, 0755)
			os.MkdirAll(userDir, 0755)
			os.MkdirAll(repoDir, 0755)

			systemPath := filepath.Join(systemDir, "config.yaml")
			userPath := filepath.Join(userDir, "config.yaml")
			repoPath := filepath.Join(repoDir, "config.yaml")

			if tt.systemYAML != "" {
				os.WriteFile(systemPath, []byte(tt.systemYAML), 0644)
			}
			if tt.userYAML != "" {
				os.WriteFile(userPath, []byte(tt.userYAML), 0644)
			}
			if tt.repoYAML != "" {
				os.WriteFile(repoPath, []byte(tt.repoYAML), 0644)
			}

			// Build the paths list in order
			var paths []string
			if tt.systemYAML != "" {
				paths = append(paths, systemPath)
			} else {
				paths = append(paths, filepath.Join(systemDir, "nonexistent.yaml"))
			}
			if tt.userYAML != "" {
				paths = append(paths, userPath)
			} else {
				paths = append(paths, filepath.Join(userDir, "nonexistent.yaml"))
			}
			if tt.repoYAML != "" {
				paths = append(paths, repoPath)
			}

			// Test using LoadHierarchicalConfigFromPaths
			config, err := LoadHierarchicalConfigFromPaths(paths)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if config.Dev.Runtime != tt.wantRuntime {
				t.Errorf("Dev.Runtime = %q, want %q", config.Dev.Runtime, tt.wantRuntime)
			}
			if config.Plugins.Registry != tt.wantRegistry {
				t.Errorf("Plugins.Registry = %q, want %q", config.Plugins.Registry, tt.wantRegistry)
			}
			if config.Dev.K3d.ClusterName != tt.wantCluster {
				t.Errorf("Dev.K3d.ClusterName = %q, want %q", config.Dev.K3d.ClusterName, tt.wantCluster)
			}
		})
	}
}

// T006: TestConfigScopePaths
func TestConfigScopePaths(t *testing.T) {
	// UserConfigPath should return a non-empty path
	userPath := UserConfigPath()
	if userPath == "" {
		t.Error("UserConfigPath() returned empty string")
	}
	if !containsStr(userPath, ".config") || !containsStr(userPath, "dp") || !containsStr(userPath, "config.yaml") {
		t.Errorf("UserConfigPath() = %q, expected to contain .config/dp/config.yaml", userPath)
	}

	// SystemConfigPath should return the fixed path
	sysPath := SystemConfigPath()
	if sysPath != "/etc/datakit/config.yaml" {
		t.Errorf("SystemConfigPath() = %q, want /etc/datakit/config.yaml", sysPath)
	}

	// RepoConfigPath - in a git repo should return non-empty
	repoPath := RepoConfigPath()
	if repoPath != "" {
		if !containsStr(repoPath, ".dp") || !containsStr(repoPath, "config.yaml") {
			t.Errorf("RepoConfigPath() = %q, expected to contain .dp/config.yaml", repoPath)
		}
	}

	// ConfigScopePath should return the right path for each scope
	if ConfigScopePath(ScopeUser) != userPath {
		t.Errorf("ConfigScopePath(ScopeUser) = %q, want %q", ConfigScopePath(ScopeUser), userPath)
	}
	if ConfigScopePath(ScopeSystem) != sysPath {
		t.Errorf("ConfigScopePath(ScopeSystem) = %q, want %q", ConfigScopePath(ScopeSystem), sysPath)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

// T007: TestValidate and TestValidateField
func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantErrs int
	}{
		{
			name: "valid config",
			config: Config{
				Dev:     DevConfig{Runtime: "k3d", K3d: K3dConfig{ClusterName: "dp-local"}},
				Plugins: PluginsConfig{Registry: "ghcr.io/infobloxopen"},
			},
			wantErrs: 0,
		},
		{
			name: "invalid runtime",
			config: Config{
				Dev: DevConfig{Runtime: "docker"},
			},
			wantErrs: 1,
		},
		{
			name: "invalid cluster name",
			config: Config{
				Dev: DevConfig{Runtime: "k3d", K3d: K3dConfig{ClusterName: "INVALID_NAME!"}},
			},
			wantErrs: 1,
		},
		{
			name: "invalid registry",
			config: Config{
				Plugins: PluginsConfig{Registry: "https://not-valid"},
			},
			wantErrs: 1,
		},
		{
			name: "mutually exclusive version and image",
			config: Config{
				Plugins: PluginsConfig{
					Overrides: map[string]PluginOverride{
						"postgresql": {Version: "v8.13.0", Image: "custom:latest"},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "invalid version in override",
			config: Config{
				Plugins: PluginsConfig{
					Overrides: map[string]PluginOverride{
						"postgresql": {Version: "8.13.0"}, // missing v prefix
					},
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.config.Validate()
			if len(errs) != tt.wantErrs {
				t.Errorf("Validate() returned %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
			}
		})
	}
}

func TestValidateField(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{name: "valid runtime k3d", key: "dev.runtime", value: "k3d", wantErr: false},
		{name: "valid runtime compose", key: "dev.runtime", value: "compose", wantErr: false},
		{name: "invalid runtime", key: "dev.runtime", value: "docker", wantErr: true},
		{name: "valid registry", key: "plugins.registry", value: "ghcr.io/myteam", wantErr: false},
		{name: "invalid registry", key: "plugins.registry", value: "https://bad", wantErr: true},
		{name: "valid cluster name", key: "dev.k3d.clusterName", value: "dp-local", wantErr: false},
		{name: "invalid cluster name", key: "dev.k3d.clusterName", value: "BAD!", wantErr: true},
		{name: "valid workspace", key: "dev.workspace", value: "/path/to/workspace", wantErr: false},
		{name: "unknown key", key: "foo.bar", value: "whatever", wantErr: true},
		{name: "valid override version", key: "plugins.overrides.postgresql.version", value: "v8.13.0", wantErr: false},
		{name: "invalid override version", key: "plugins.overrides.postgresql.version", value: "8.13.0", wantErr: true},
		{name: "valid override image", key: "plugins.overrides.postgresql.image", value: "custom:latest", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateField(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateField(%q, %q) error = %v, wantErr %v", tt.key, tt.value, err, tt.wantErr)
			}
		})
	}
}

// T008: TestConfigSetField and TestConfigUnsetField
func TestConfigSetField(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		check   func(*Config) bool
		wantErr bool
	}{
		{
			name:  "set dev.runtime",
			key:   "dev.runtime",
			value: "compose",
			check: func(c *Config) bool { return c.Dev.Runtime == "compose" },
		},
		{
			name:  "set dev.workspace",
			key:   "dev.workspace",
			value: "/my/workspace",
			check: func(c *Config) bool { return c.Dev.Workspace == "/my/workspace" },
		},
		{
			name:  "set dev.k3d.clusterName",
			key:   "dev.k3d.clusterName",
			value: "my-cluster",
			check: func(c *Config) bool { return c.Dev.K3d.ClusterName == "my-cluster" },
		},
		{
			name:  "set plugins.registry",
			key:   "plugins.registry",
			value: "ghcr.io/myteam",
			check: func(c *Config) bool { return c.Plugins.Registry == "ghcr.io/myteam" },
		},
		{
			name:  "set override version",
			key:   "plugins.overrides.postgresql.version",
			value: "v8.13.0",
			check: func(c *Config) bool {
				return c.Plugins.Overrides != nil && c.Plugins.Overrides["postgresql"].Version == "v8.13.0"
			},
		},
		{
			name:  "set override image",
			key:   "plugins.overrides.postgresql.image",
			value: "custom-pg:v2.0.0",
			check: func(c *Config) bool {
				return c.Plugins.Overrides != nil && c.Plugins.Overrides["postgresql"].Image == "custom-pg:v2.0.0"
			},
		},
		{
			name:    "unknown key",
			key:     "foo.bar",
			value:   "baz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			err := config.SetField(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetField(%q, %q) error = %v, wantErr %v", tt.key, tt.value, err, tt.wantErr)
			}
			if tt.check != nil && !tt.check(config) {
				t.Errorf("SetField(%q, %q) did not set the expected value", tt.key, tt.value)
			}
		})
	}
}

func TestConfigUnsetField(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Config)
		key     string
		check   func(*Config) bool
		wantErr bool
	}{
		{
			name:  "unset dev.runtime",
			setup: func(c *Config) { c.Dev.Runtime = "compose" },
			key:   "dev.runtime",
			check: func(c *Config) bool { return c.Dev.Runtime == "" },
		},
		{
			name:  "unset dev.workspace",
			setup: func(c *Config) { c.Dev.Workspace = "/some/path" },
			key:   "dev.workspace",
			check: func(c *Config) bool { return c.Dev.Workspace == "" },
		},
		{
			name:  "unset plugins.registry",
			setup: func(c *Config) { c.Plugins.Registry = "ghcr.io/myteam" },
			key:   "plugins.registry",
			check: func(c *Config) bool { return c.Plugins.Registry == "" },
		},
		{
			name: "unset override version removes entry when both empty",
			setup: func(c *Config) {
				c.Plugins.Overrides = map[string]PluginOverride{
					"postgresql": {Version: "v8.13.0"},
				}
			},
			key: "plugins.overrides.postgresql.version",
			check: func(c *Config) bool {
				_, exists := c.Plugins.Overrides["postgresql"]
				return !exists
			},
		},
		{
			name: "unset override version keeps image",
			setup: func(c *Config) {
				c.Plugins.Overrides = map[string]PluginOverride{
					"postgresql": {Version: "v8.13.0", Image: "custom:latest"},
				}
			},
			key: "plugins.overrides.postgresql.version",
			check: func(c *Config) bool {
				o, exists := c.Plugins.Overrides["postgresql"]
				return exists && o.Version == "" && o.Image == "custom:latest"
			},
		},
		{
			name:    "unknown key",
			key:     "foo.bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			if tt.setup != nil {
				tt.setup(config)
			}
			err := config.UnsetField(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnsetField(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
			if tt.check != nil && !tt.check(config) {
				t.Errorf("UnsetField(%q) did not produce expected result", tt.key)
			}
		})
	}
}

// T009: TestBackwardCompatibility
func TestBackwardCompatibility(t *testing.T) {
	// Verify that loading an existing dev-only YAML config into the new
	// Config struct works correctly — Plugins gets zero value
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a config that only has the dev section (old format)
	oldConfig := `dev:
  runtime: k3d
  workspace: /path/to/workspace
  k3d:
    clusterName: dp-local
`
	if err := os.WriteFile(configPath, []byte(oldConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	config, err := LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromPath() error = %v", err)
	}

	// Dev section should be loaded correctly
	if config.Dev.Runtime != "k3d" {
		t.Errorf("Dev.Runtime = %q, want 'k3d'", config.Dev.Runtime)
	}
	if config.Dev.Workspace != "/path/to/workspace" {
		t.Errorf("Dev.Workspace = %q, want '/path/to/workspace'", config.Dev.Workspace)
	}
	if config.Dev.K3d.ClusterName != "dp-local" {
		t.Errorf("Dev.K3d.ClusterName = %q, want 'dp-local'", config.Dev.K3d.ClusterName)
	}

	// Plugins section should be zero value
	if config.Plugins.Registry != "" {
		t.Errorf("Plugins.Registry = %q, want empty (zero value)", config.Plugins.Registry)
	}
	if len(config.Plugins.Mirrors) != 0 {
		t.Errorf("Plugins.Mirrors = %v, want empty", config.Plugins.Mirrors)
	}
	if len(config.Plugins.Overrides) != 0 {
		t.Errorf("Plugins.Overrides = %v, want empty", config.Plugins.Overrides)
	}

	// Round-trip: save and reload
	err = SaveConfigToPath(config, configPath)
	if err != nil {
		t.Fatalf("SaveConfigToPath() error = %v", err)
	}

	reloaded, err := LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromPath() after save error = %v", err)
	}

	// Should still match
	if reloaded.Dev.Runtime != config.Dev.Runtime {
		t.Errorf("Round-trip Runtime = %q, want %q", reloaded.Dev.Runtime, config.Dev.Runtime)
	}
	if reloaded.Dev.K3d.ClusterName != config.Dev.K3d.ClusterName {
		t.Errorf("Round-trip ClusterName = %q, want %q", reloaded.Dev.K3d.ClusterName, config.Dev.K3d.ClusterName)
	}
}

// --- Destination Config Tests ---

func TestDestinationConfig_SetGetUnset(t *testing.T) {
	cfg := &Config{}

	// Set
	if err := cfg.SetField("plugins.destinations.postgresql.connection_string", "postgresql://admin:pass@db:5432/mydb"); err != nil {
		t.Fatalf("SetField() error: %v", err)
	}

	// Get
	val, ok := cfg.GetField("plugins.destinations.postgresql.connection_string")
	if !ok || val != "postgresql://admin:pass@db:5432/mydb" {
		t.Errorf("GetField() = (%q, %v), want (postgresql://admin:pass@db:5432/mydb, true)", val, ok)
	}

	// Unset
	if err := cfg.UnsetField("plugins.destinations.postgresql.connection_string"); err != nil {
		t.Fatalf("UnsetField() error: %v", err)
	}
	val, ok = cfg.GetField("plugins.destinations.postgresql.connection_string")
	if ok {
		t.Errorf("GetField() after unset = (%q, %v), want ('', false)", val, ok)
	}

	// Destination entry should be removed entirely when all fields are empty
	if len(cfg.Plugins.Destinations) != 0 {
		t.Errorf("Destinations map should be empty after unsetting all fields, got %v", cfg.Plugins.Destinations)
	}
}

func TestDestinationConfig_S3Fields(t *testing.T) {
	cfg := &Config{}

	fields := map[string]string{
		"plugins.destinations.s3.bucket":   "my-bucket",
		"plugins.destinations.s3.region":   "eu-west-1",
		"plugins.destinations.s3.endpoint": "http://minio:9000",
		"plugins.destinations.s3.path":     "data/{{TABLE}}/{{UUID}}.json",
	}

	for key, val := range fields {
		if err := cfg.SetField(key, val); err != nil {
			t.Fatalf("SetField(%q) error: %v", key, err)
		}
	}

	for key, want := range fields {
		got, ok := cfg.GetField(key)
		if !ok || got != want {
			t.Errorf("GetField(%q) = (%q, %v), want (%q, true)", key, got, ok, want)
		}
	}

	// Verify struct values directly
	s3 := cfg.Plugins.Destinations["s3"]
	if s3.Bucket != "my-bucket" {
		t.Errorf("Bucket = %q, want 'my-bucket'", s3.Bucket)
	}
	if s3.Endpoint != "http://minio:9000" {
		t.Errorf("Endpoint = %q, want 'http://minio:9000'", s3.Endpoint)
	}
}

func TestDestinationConfig_ValidateField(t *testing.T) {
	// Valid destination keys should pass validation
	validKeys := []string{
		"plugins.destinations.postgresql.connection_string",
		"plugins.destinations.s3.bucket",
		"plugins.destinations.s3.region",
		"plugins.destinations.s3.endpoint",
		"plugins.destinations.file.path",
	}
	for _, key := range validKeys {
		if err := ValidateField(key, "some-value"); err != nil {
			t.Errorf("ValidateField(%q) should pass, got: %v", key, err)
		}
	}

	// Empty value should fail
	if err := ValidateField("plugins.destinations.postgresql.connection_string", ""); err == nil {
		t.Error("ValidateField with empty value should fail")
	}

	// Invalid field name should fail
	if err := ValidateField("plugins.destinations.postgresql.invalid_field", "val"); err == nil {
		t.Error("ValidateField with invalid destination field should fail")
	}
}

func TestDestinationConfig_YAMLRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &Config{
		Plugins: PluginsConfig{
			Destinations: map[string]DestinationConfig{
				"postgresql": {ConnectionString: "postgresql://admin:pass@mydb:5432/analytics"},
				"s3":         {Bucket: "prod-data", Region: "us-west-2", Endpoint: "http://localstack:4566"},
			},
		},
	}

	if err := SaveConfigToPath(cfg, configPath); err != nil {
		t.Fatalf("SaveConfigToPath() error: %v", err)
	}

	loaded, err := LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigFromPath() error: %v", err)
	}

	pg := loaded.Plugins.Destinations["postgresql"]
	if pg.ConnectionString != "postgresql://admin:pass@mydb:5432/analytics" {
		t.Errorf("postgresql connection_string = %q, want 'postgresql://admin:pass@mydb:5432/analytics'", pg.ConnectionString)
	}

	s3 := loaded.Plugins.Destinations["s3"]
	if s3.Bucket != "prod-data" {
		t.Errorf("s3 bucket = %q, want 'prod-data'", s3.Bucket)
	}
	if s3.Endpoint != "http://localstack:4566" {
		t.Errorf("s3 endpoint = %q, want 'http://localstack:4566'", s3.Endpoint)
	}
}

func TestDestinationConfig_EffectiveValue(t *testing.T) {
	// EffectiveValue should accept destination keys without returning "unknown key" error
	_, _, err := EffectiveValue("plugins.destinations.postgresql.connection_string")
	if err != nil {
		t.Errorf("EffectiveValue() should accept destination key, got: %v", err)
	}
}

// TestChartOverride_SetGetUnset tests round-trip of chart override config keys.
func TestChartOverride_SetGetUnset(t *testing.T) {
	config := &Config{}

	// Set chart version
	if err := config.SetField("dev.charts.redpanda.version", "v25.2.0"); err != nil {
		t.Fatalf("SetField(dev.charts.redpanda.version) error: %v", err)
	}

	// Get chart version
	val, ok := config.GetField("dev.charts.redpanda.version")
	if !ok || val != "v25.2.0" {
		t.Errorf("GetField(dev.charts.redpanda.version) = %q, %v; want %q, true", val, ok, "v25.2.0")
	}

	// Set chart value override
	if err := config.SetField("dev.charts.postgres.values.primary.resources.limits.memory", "512Mi"); err != nil {
		t.Fatalf("SetField(dev.charts.postgres.values...) error: %v", err)
	}

	// Get chart value override
	val, ok = config.GetField("dev.charts.postgres.values.primary.resources.limits.memory")
	if !ok || val != "512Mi" {
		t.Errorf("GetField(dev.charts.postgres.values...) = %q, %v; want %q, true", val, ok, "512Mi")
	}

	// Unset chart version
	if err := config.UnsetField("dev.charts.redpanda.version"); err != nil {
		t.Fatalf("UnsetField(dev.charts.redpanda.version) error: %v", err)
	}
	_, ok = config.GetField("dev.charts.redpanda.version")
	if ok {
		t.Error("GetField should return false after UnsetField")
	}

	// Unset chart value
	if err := config.UnsetField("dev.charts.postgres.values.primary.resources.limits.memory"); err != nil {
		t.Fatalf("UnsetField(dev.charts.postgres.values...) error: %v", err)
	}
	_, ok = config.GetField("dev.charts.postgres.values.primary.resources.limits.memory")
	if ok {
		t.Error("GetField should return false after UnsetField")
	}
}

// TestChartOverride_ValidateField tests validation of chart config keys.
func TestChartOverride_ValidateField(t *testing.T) {
	// Valid version
	err := ValidateField("dev.charts.redpanda.version", "v25.3.2")
	if err != nil {
		t.Errorf("ValidateField valid version error: %v", err)
	}

	// Invalid version
	err = ValidateField("dev.charts.redpanda.version", "not-semver")
	if err == nil {
		t.Error("ValidateField should reject non-semver version")
	}

	// Values path — any value is valid
	err = ValidateField("dev.charts.postgres.values.some.path", "anything")
	if err != nil {
		t.Errorf("ValidateField values path error: %v", err)
	}
}

// TestChartOverride_YAMLRoundTrip tests YAML marshaling of chart overrides.
func TestChartOverride_YAMLRoundTrip(t *testing.T) {
	config := &Config{}
	config.SetField("dev.charts.redpanda.version", "v25.2.0")
	config.SetField("dev.charts.postgres.values.memory", "512Mi")

	// Save and reload
	tmpDir := t.TempDir()
	path := tmpDir + "/config.yaml"
	if err := SaveConfigToPath(config, path); err != nil {
		t.Fatalf("SaveConfigToPath error: %v", err)
	}

	loaded, err := LoadConfigFromPath(path)
	if err != nil {
		t.Fatalf("LoadConfigFromPath error: %v", err)
	}

	// Verify chart overrides survived round-trip
	if loaded.Dev.Charts == nil {
		t.Fatal("Charts map is nil after round-trip")
	}
	if loaded.Dev.Charts["redpanda"].Version != "v25.2.0" {
		t.Errorf("redpanda version = %q, want %q", loaded.Dev.Charts["redpanda"].Version, "v25.2.0")
	}
}

// TestChartOverride_EffectiveValue tests EffectiveValue accepts chart keys.
func TestChartOverride_EffectiveValue(t *testing.T) {
	_, _, err := EffectiveValue("dev.charts.redpanda.version")
	if err != nil {
		t.Errorf("EffectiveValue should accept chart key, got: %v", err)
	}

	_, _, err = EffectiveValue("dev.charts.postgres.values.some.path")
	if err != nil {
		t.Errorf("EffectiveValue should accept chart values key, got: %v", err)
	}
}
