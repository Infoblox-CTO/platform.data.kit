package localdev

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCacheManager(t *testing.T) {
	tests := []struct {
		name    string
		opts    []CacheOption
		wantErr bool
		check   func(*testing.T, *CacheManager)
	}{
		{
			name:    "default values",
			opts:    nil,
			wantErr: false,
			check: func(t *testing.T, m *CacheManager) {
				if m.containerName != DefaultContainerName {
					t.Errorf("containerName = %q, want %q", m.containerName, DefaultContainerName)
				}
				if m.volumeName != DefaultVolumeName {
					t.Errorf("volumeName = %q, want %q", m.volumeName, DefaultVolumeName)
				}
				if m.networkName != DefaultNetworkName {
					t.Errorf("networkName = %q, want %q", m.networkName, DefaultNetworkName)
				}
				if m.port != DefaultPort {
					t.Errorf("port = %d, want %d", m.port, DefaultPort)
				}
				if m.cacheDir != DefaultCacheDir {
					t.Errorf("cacheDir = %q, want %q", m.cacheDir, DefaultCacheDir)
				}
			},
		},
		{
			name: "with custom port",
			opts: []CacheOption{WithPort(5001)},
			check: func(t *testing.T, m *CacheManager) {
				if m.port != 5001 {
					t.Errorf("port = %d, want 5001", m.port)
				}
			},
		},
		{
			name: "with custom cache dir",
			opts: []CacheOption{WithCacheDir("/tmp/test-cache")},
			check: func(t *testing.T, m *CacheManager) {
				if m.cacheDir != "/tmp/test-cache" {
					t.Errorf("cacheDir = %q, want /tmp/test-cache", m.cacheDir)
				}
			},
		},
		{
			name: "with custom mirror host",
			opts: []CacheOption{WithMirrorHost("custom.host")},
			check: func(t *testing.T, m *CacheManager) {
				if m.mirrorHost != "custom.host" {
					t.Errorf("mirrorHost = %q, want custom.host", m.mirrorHost)
				}
			},
		},
		{
			name: "multiple options",
			opts: []CacheOption{
				WithPort(5002),
				WithCacheDir("/custom/dir"),
				WithMirrorHost("my.host"),
			},
			check: func(t *testing.T, m *CacheManager) {
				if m.port != 5002 {
					t.Errorf("port = %d, want 5002", m.port)
				}
				if m.cacheDir != "/custom/dir" {
					t.Errorf("cacheDir = %q, want /custom/dir", m.cacheDir)
				}
				if m.mirrorHost != "my.host" {
					t.Errorf("mirrorHost = %q, want my.host", m.mirrorHost)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment for consistent tests
			os.Unsetenv("DEV_REGISTRY_MIRROR_HOST")

			m, err := NewCacheManager(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCacheManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, m)
			}
		})
	}
}

func TestDetectMirrorHost(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    string
	}{
		{
			name:    "default to host.k3d.internal",
			envVars: nil,
			want:    "host.k3d.internal",
		},
		{
			name: "override with DEV_REGISTRY_MIRROR_HOST",
			envVars: map[string]string{
				"DEV_REGISTRY_MIRROR_HOST": "custom.registry.host",
			},
			want: "custom.registry.host",
		},
		{
			name: "empty override uses default",
			envVars: map[string]string{
				"DEV_REGISTRY_MIRROR_HOST": "",
			},
			want: "host.k3d.internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set environment
			os.Unsetenv("DEV_REGISTRY_MIRROR_HOST")
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			got := detectMirrorHost()
			if got != tt.want {
				t.Errorf("detectMirrorHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigHash(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		want  int  // Expected length of hash string
		equal bool // Whether two calls with same data should be equal
	}{
		{
			name:  "empty data",
			data:  []byte{},
			want:  64, // SHA256 produces 64 hex characters
			equal: true,
		},
		{
			name:  "simple data",
			data:  []byte("test config data"),
			want:  64,
			equal: true,
		},
		{
			name:  "yaml-like data",
			data:  []byte("version: 0.1\nlog:\n  level: info"),
			want:  64,
			equal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := configHash(tt.data)
			if len(got) != tt.want {
				t.Errorf("configHash() length = %d, want %d", len(got), tt.want)
			}

			if tt.equal {
				got2 := configHash(tt.data)
				if got != got2 {
					t.Errorf("configHash() not deterministic: %q != %q", got, got2)
				}
			}
		})
	}

	// Test that different data produces different hashes
	hash1 := configHash([]byte("data1"))
	hash2 := configHash([]byte("data2"))
	if hash1 == hash2 {
		t.Error("configHash() should produce different hashes for different data")
	}
}

func TestIsCI(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "no CI vars set",
			envVars: nil,
			want:    false,
		},
		{
			name: "CI=true",
			envVars: map[string]string{
				"CI": "true",
			},
			want: true,
		},
		{
			name: "CI=false",
			envVars: map[string]string{
				"CI": "false",
			},
			want: false,
		},
		{
			name: "GITHUB_ACTIONS=true",
			envVars: map[string]string{
				"GITHUB_ACTIONS": "true",
			},
			want: true,
		},
		{
			name: "JENKINS_URL set",
			envVars: map[string]string{
				"JENKINS_URL": "http://jenkins.example.com",
			},
			want: true,
		},
		{
			name: "JENKINS_URL empty",
			envVars: map[string]string{
				"JENKINS_URL": "",
			},
			want: false,
		},
		{
			name: "multiple CI vars",
			envVars: map[string]string{
				"CI":             "true",
				"GITHUB_ACTIONS": "true",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all CI environment variables
			os.Unsetenv("CI")
			os.Unsetenv("GITHUB_ACTIONS")
			os.Unsetenv("JENKINS_URL")

			// Set test environment
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				os.Unsetenv("CI")
				os.Unsetenv("GITHUB_ACTIONS")
				os.Unsetenv("JENKINS_URL")
			}()

			got := IsCI()
			if got != tt.want {
				t.Errorf("IsCI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCacheManager_Endpoint(t *testing.T) {
	tests := []struct {
		name       string
		mirrorHost string
		port       int
		want       string
	}{
		{
			name:       "default values",
			mirrorHost: "host.k3d.internal",
			port:       5000,
			want:       "http://host.k3d.internal:5000",
		},
		{
			name:       "custom host",
			mirrorHost: "custom.host",
			port:       5000,
			want:       "http://custom.host:5000",
		},
		{
			name:       "custom port",
			mirrorHost: "host.k3d.internal",
			port:       5001,
			want:       "http://host.k3d.internal:5001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &CacheManager{
				mirrorHost: tt.mirrorHost,
				port:       tt.port,
			}
			if got := m.Endpoint(); got != tt.want {
				t.Errorf("Endpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCacheManager_GetRegistriesYAMLPath(t *testing.T) {
	tests := []struct {
		name     string
		cacheDir string
		isCI     bool
		want     string
	}{
		{
			name:     "normal environment",
			cacheDir: ".cache",
			isCI:     false,
			want:     ".cache/registries.yaml",
		},
		{
			name:     "CI environment",
			cacheDir: ".cache",
			isCI:     true,
			want:     "",
		},
		{
			name:     "custom cache dir",
			cacheDir: "/tmp/custom",
			isCI:     false,
			want:     "/tmp/custom/registries.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set CI environment
			os.Unsetenv("CI")
			if tt.isCI {
				os.Setenv("CI", "true")
			}
			defer os.Unsetenv("CI")

			m := &CacheManager{
				cacheDir: tt.cacheDir,
			}
			if got := m.GetRegistriesYAMLPath(); got != tt.want {
				t.Errorf("GetRegistriesYAMLPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCacheManager_WriteRegistryConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	m := &CacheManager{
		cacheDir: tmpDir,
		port:     5000,
	}

	data, err := m.writeRegistryConfig()
	if err != nil {
		t.Fatalf("writeRegistryConfig() error = %v", err)
	}

	// Check file was created
	configPath := filepath.Join(tmpDir, "registry-config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("registry-config.yml was not created")
	}

	// Check data is non-empty
	if len(data) == 0 {
		t.Error("writeRegistryConfig() returned empty data")
	}

	// Check content contains expected values
	content, _ := os.ReadFile(configPath)
	contentStr := string(content)
	if !contains(contentStr, "version: \"0.1\"") && !contains(contentStr, "version: 0.1") {
		t.Error("config should contain version: 0.1")
	}
	if !contains(contentStr, "proxy:") {
		t.Error("config should contain proxy section")
	}
	if !contains(contentStr, DefaultRemoteURL) {
		t.Errorf("config should contain remote URL %s", DefaultRemoteURL)
	}
}

func TestCacheManager_WriteRegistriesYAML(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	m := &CacheManager{
		cacheDir:   tmpDir,
		mirrorHost: "host.k3d.internal",
		port:       5000,
	}

	err := m.writeRegistriesYAML()
	if err != nil {
		t.Fatalf("writeRegistriesYAML() error = %v", err)
	}

	// Check file was created
	configPath := filepath.Join(tmpDir, "registries.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("registries.yaml was not created")
	}

	// Check content contains expected values
	content, _ := os.ReadFile(configPath)
	contentStr := string(content)
	if !contains(contentStr, "mirrors:") {
		t.Error("config should contain mirrors section")
	}
	if !contains(contentStr, "docker.io:") {
		t.Error("config should contain docker.io mirror")
	}
	if !contains(contentStr, "http://host.k3d.internal:5000") {
		t.Error("config should contain endpoint URL")
	}
}

func TestCacheManager_Status(t *testing.T) {
	// This test verifies the Status method returns correct defaults
	// when container doesn't exist (unit test, no Docker required)
	m := &CacheManager{
		containerName: "nonexistent-container-test",
		mirrorHost:    "host.k3d.internal",
		port:          5000,
	}

	status, err := m.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.Exists {
		t.Error("Status.Exists should be false for nonexistent container")
	}
	if status.Running {
		t.Error("Status.Running should be false for nonexistent container")
	}
	if status.Endpoint != "http://host.k3d.internal:5000" {
		t.Errorf("Status.Endpoint = %q, want http://host.k3d.internal:5000", status.Endpoint)
	}
}
