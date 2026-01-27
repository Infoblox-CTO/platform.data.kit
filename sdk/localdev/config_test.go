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
