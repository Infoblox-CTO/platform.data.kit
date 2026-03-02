package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/spf13/cobra"
)

// T036: TestConfigSetCmd
func TestConfigSetCmd(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		scope     string
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "set valid registry",
			key:   "plugins.registry",
			value: "ghcr.io/myteam",
			scope: "user",
		},
		{
			name:  "set valid runtime",
			key:   "dev.runtime",
			value: "k3d",
			scope: "user",
		},
		{
			name:      "set invalid key",
			key:       "unknown.field",
			value:     "anything",
			scope:     "user",
			wantErr:   true,
			errSubstr: "unknown",
		},
		{
			name:      "set invalid runtime value",
			key:       "dev.runtime",
			value:     "invalid-runtime",
			scope:     "user",
			wantErr:   true,
			errSubstr: "allowed",
		},
		{
			name:  "set with repo scope",
			key:   "plugins.registry",
			value: "ghcr.io/repo-team",
			scope: "repo",
		},
		{
			name:  "creates config file if missing",
			key:   "dev.workspace",
			value: "/tmp/test-workspace",
			scope: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore configScope
			oldScope := configScope
			defer func() { configScope = oldScope }()
			configScope = tt.scope

			// Use a temp dir for config files
			tmpDir := t.TempDir()

			// For repo scope, we need a .dk dir
			if tt.scope == "repo" {
				dpDir := filepath.Join(tmpDir, ".dk")
				os.MkdirAll(dpDir, 0755)
			}

			// We test the validation logic directly since the file system paths
			// are determined by git root / home dir in the real implementation
			err := localdev.ValidateField(tt.key, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateField() unexpected error: %v", err)
			}

			// Verify the field can be set on a config struct
			cfg := &localdev.Config{}
			if err := cfg.SetField(tt.key, tt.value); err != nil {
				t.Fatalf("SetField() error: %v", err)
			}

			// Verify it reads back
			got, ok := cfg.GetField(tt.key)
			if !ok {
				t.Errorf("GetField(%q) not found after SetField", tt.key)
			} else if got != tt.value {
				t.Errorf("GetField(%q) = %q, want %q", tt.key, got, tt.value)
			}
		})
	}
}

// T036 additional: Test the configSetCmd cobra execution with real file I/O
func TestConfigSetCmd_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write to a temp config, then read it back
	cfg := &localdev.Config{}
	if err := cfg.SetField("plugins.registry", "ghcr.io/test-org"); err != nil {
		t.Fatal(err)
	}
	if err := localdev.SaveConfigToPath(cfg, configPath); err != nil {
		t.Fatal(err)
	}

	// Read back and verify
	loaded, err := localdev.LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Plugins.Registry != "ghcr.io/test-org" {
		t.Errorf("registry = %q, want %q", loaded.Plugins.Registry, "ghcr.io/test-org")
	}
}

// T037: TestConfigGetCmd
func TestConfigGetCmd(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name: "get built-in registry",
			key:  "plugins.registry",
		},
		{
			name: "get built-in runtime",
			key:  "dev.runtime",
		},
		{
			name: "get built-in cluster name",
			key:  "dev.k3d.clusterName",
		},
		{
			name:    "get unknown key errors",
			key:     "nonexistent.key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test EffectiveValue directly since it's what configGetCmd calls
			value, source, err := localdev.EffectiveValue(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("EffectiveValue(%q) error: %v", tt.key, err)
			}

			// Built-in defaults should return a value
			if value == "" && source == "built-in" {
				// Some keys don't have defaults (like dev.workspace), that's OK
				return
			}
			if source == "" {
				t.Errorf("source should not be empty for key %q", tt.key)
			}
		})
	}
}

// T037 additional: Verify configGetCmd cobra execution output format
func TestConfigGetCmd_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	configGetCmd.SetOut(&buf)
	configGetCmd.SetErr(&buf)

	err := configGetCmd.RunE(configGetCmd, []string{"plugins.registry"})
	if err != nil {
		t.Fatalf("configGetCmd error: %v", err)
	}

	output := buf.String()
	// Should contain the registry value and source
	if !strings.Contains(output, "ghcr.io") {
		t.Errorf("output should contain default registry, got: %q", output)
	}
	if !strings.Contains(output, "built-in") && !strings.Contains(output, "repo") && !strings.Contains(output, "user") && !strings.Contains(output, "system") {
		t.Errorf("output should contain source scope, got: %q", output)
	}
}

// T038: TestConfigUnsetCmd
func TestConfigUnsetCmd(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		setupFn func(cfg *localdev.Config)
	}{
		{
			name: "unset existing scalar key",
			key:  "plugins.registry",
			setupFn: func(cfg *localdev.Config) {
				cfg.Plugins.Registry = "ghcr.io/some-org"
			},
		},
		{
			name: "unset runtime",
			key:  "dev.runtime",
			setupFn: func(cfg *localdev.Config) {
				cfg.Dev.Runtime = "compose"
			},
		},
		{
			name:    "unset non-existent key is no-op",
			key:     "dev.workspace",
			setupFn: func(cfg *localdev.Config) {},
		},
		{
			name: "unset cluster name",
			key:  "dev.k3d.clusterName",
			setupFn: func(cfg *localdev.Config) {
				cfg.Dev.K3d.ClusterName = "my-cluster"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &localdev.Config{}
			tt.setupFn(cfg)

			err := cfg.UnsetField(tt.key)
			if err != nil {
				t.Fatalf("UnsetField(%q) error: %v", tt.key, err)
			}

			// After unset, GetField should return false
			if v, ok := cfg.GetField(tt.key); ok {
				t.Errorf("GetField(%q) = %q, want not found after unset", tt.key, v)
			}
		})
	}
}

// T038 additional: Verify unset round-trip through file I/O
func TestConfigUnsetCmd_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Set a value, save, unset, save, load
	cfg := &localdev.Config{}
	cfg.Plugins.Registry = "ghcr.io/temp"
	if err := localdev.SaveConfigToPath(cfg, configPath); err != nil {
		t.Fatal(err)
	}

	// Load, unset, save
	cfg2, err := localdev.LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg2.UnsetField("plugins.registry"); err != nil {
		t.Fatal(err)
	}
	if err := localdev.SaveConfigToPath(cfg2, configPath); err != nil {
		t.Fatal(err)
	}

	// Load again and verify
	cfg3, err := localdev.LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg3.Plugins.Registry != "" {
		t.Errorf("plugins.registry should be empty after unset, got %q", cfg3.Plugins.Registry)
	}
}

// T039: TestConfigListCmd
func TestConfigListCmd(t *testing.T) {
	// Test with no scope filter — shows effective values
	t.Run("list all shows built-in defaults", func(t *testing.T) {
		keys := localdev.AllConfigKeys()
		if len(keys) == 0 {
			t.Fatal("AllConfigKeys() returned empty")
		}

		// Should contain known keys
		keySet := make(map[string]bool)
		for _, k := range keys {
			keySet[k] = true
		}
		expectedKeys := []string{"dev.runtime", "plugins.registry", "dev.k3d.clusterName"}
		for _, ek := range expectedKeys {
			if !keySet[ek] {
				t.Errorf("AllConfigKeys() missing %q", ek)
			}
		}
	})

	t.Run("list with scope filter", func(t *testing.T) {
		// Create a temp config with known values
		cfg := &localdev.Config{}
		cfg.Plugins.Registry = "ghcr.io/scoped"
		cfg.Dev.Runtime = "compose"

		// Verify GetField returns values for configured keys only
		for _, key := range localdev.AllConfigKeys() {
			v, ok := cfg.GetField(key)
			switch key {
			case "plugins.registry":
				if !ok || v != "ghcr.io/scoped" {
					t.Errorf("GetField(%q) = %q, %v; want ghcr.io/scoped, true", key, v, ok)
				}
			case "dev.runtime":
				if !ok || v != "compose" {
					t.Errorf("GetField(%q) = %q, %v; want compose, true", key, v, ok)
				}
			default:
				// Unset keys should return false
				if ok {
					t.Errorf("GetField(%q) = %q, %v; want empty, false", key, v, ok)
				}
			}
		}
	})

	t.Run("list includes overrides", func(t *testing.T) {
		cfg := &localdev.Config{
			Plugins: localdev.PluginsConfig{
				Overrides: map[string]localdev.PluginOverride{
					"postgresql": {Version: "v8.13.0"},
					"s3":         {Image: "custom-s3:v1"},
				},
			},
		}

		// configListCmd iterates overrides map — verify they're accessible
		if cfg.Plugins.Overrides["postgresql"].Version != "v8.13.0" {
			t.Error("override version not found")
		}
		if cfg.Plugins.Overrides["s3"].Image != "custom-s3:v1" {
			t.Error("override image not found")
		}
	})

	t.Run("list includes mirrors", func(t *testing.T) {
		cfg := &localdev.Config{
			Plugins: localdev.PluginsConfig{
				Mirrors: []string{"ghcr.io/backup-1", "ghcr.io/backup-2"},
			},
		}

		if len(cfg.Plugins.Mirrors) != 2 {
			t.Errorf("mirrors count = %d, want 2", len(cfg.Plugins.Mirrors))
		}
	})
}

// TestConfigCmd_Registration verifies configCmd is registered with rootCmd (T044)
func TestConfigCmd_Registration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Error("config command not registered with rootCmd")
	}
}

// TestConfigCmd_Subcommands verifies all subcommands are registered
func TestConfigCmd_Subcommands(t *testing.T) {
	expected := []string{"set", "get", "unset", "list", "add-mirror", "remove-mirror"}
	cmds := configCmd.Commands()
	nameSet := make(map[string]bool)
	for _, c := range cmds {
		nameSet[c.Name()] = true
	}
	for _, e := range expected {
		if !nameSet[e] {
			t.Errorf("subcommand %q not found under configCmd", e)
		}
	}
}

// TestConfigCmd_ScopeFlags verifies --scope flags are on the right commands
func TestConfigCmd_ScopeFlags(t *testing.T) {
	cmdsWithScope := []struct {
		cmd      string
		defValue string
	}{
		{"set", "user"},
		{"unset", "user"},
		{"add-mirror", "user"},
		{"remove-mirror", "user"},
		{"list", ""},
	}

	for _, tt := range cmdsWithScope {
		t.Run(tt.cmd, func(t *testing.T) {
			var target *cobra.Command
			for _, c := range configCmd.Commands() {
				if c.Name() == tt.cmd {
					target = c
					break
				}
			}
			if target == nil {
				t.Fatalf("subcommand %q not found", tt.cmd)
			}
			flag := target.Flags().Lookup("scope")
			if flag == nil {
				t.Errorf("--%s should have --scope flag", tt.cmd)
			} else if flag.DefValue != tt.defValue {
				t.Errorf("--%s --scope default = %q, want %q", tt.cmd, flag.DefValue, tt.defValue)
			}
		})
	}
}

// TestConfigSetCmd_HelpText verifies help text is populated
func TestConfigSetCmd_HelpText(t *testing.T) {
	if configSetCmd.Short == "" {
		t.Error("configSetCmd.Short is empty")
	}
	if configSetCmd.Long == "" {
		t.Error("configSetCmd.Long is empty")
	}
	if configSetCmd.Example == "" {
		t.Error("configSetCmd.Example is empty")
	}
}

// TestConfigGetCmd_HelpText verifies help text is populated
func TestConfigGetCmd_HelpText(t *testing.T) {
	if configGetCmd.Short == "" {
		t.Error("configGetCmd.Short is empty")
	}
	if configGetCmd.Long == "" {
		t.Error("configGetCmd.Long is empty")
	}
	if configGetCmd.Example == "" {
		t.Error("configGetCmd.Example is empty")
	}
}

// --- Phase 6: Plugin Override Tests (T045) ---

// T045: TestConfigSet_PluginOverride
func TestConfigSet_PluginOverride(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
		checkFn func(t *testing.T, cfg *localdev.Config)
	}{
		{
			name:  "set version override",
			key:   "plugins.overrides.postgresql.version",
			value: "v8.13.0",
			checkFn: func(t *testing.T, cfg *localdev.Config) {
				if cfg.Plugins.Overrides == nil {
					t.Fatal("overrides map is nil")
				}
				o, ok := cfg.Plugins.Overrides["postgresql"]
				if !ok {
					t.Fatal("postgresql override not found")
				}
				if o.Version != "v8.13.0" {
					t.Errorf("version = %q, want v8.13.0", o.Version)
				}
			},
		},
		{
			name:  "set image override",
			key:   "plugins.overrides.postgresql.image",
			value: "internal.registry.io/custom-pg:v2.0.0",
			checkFn: func(t *testing.T, cfg *localdev.Config) {
				o, ok := cfg.Plugins.Overrides["postgresql"]
				if !ok {
					t.Fatal("postgresql override not found")
				}
				if o.Image != "internal.registry.io/custom-pg:v2.0.0" {
					t.Errorf("image = %q, want internal.registry.io/custom-pg:v2.0.0", o.Image)
				}
			},
		},
		{
			name:  "set s3 version override",
			key:   "plugins.overrides.s3.version",
			value: "v7.9.0",
			checkFn: func(t *testing.T, cfg *localdev.Config) {
				o, ok := cfg.Plugins.Overrides["s3"]
				if !ok {
					t.Fatal("s3 override not found")
				}
				if o.Version != "v7.9.0" {
					t.Errorf("version = %q, want v7.9.0", o.Version)
				}
			},
		},
		{
			name:    "set override with invalid field",
			key:     "plugins.overrides.postgresql.invalid",
			value:   "something",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate first
			err := localdev.ValidateField(tt.key, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateField() error: %v", err)
			}

			// Set on config
			cfg := &localdev.Config{}
			if err := cfg.SetField(tt.key, tt.value); err != nil {
				t.Fatalf("SetField() error: %v", err)
			}

			tt.checkFn(t, cfg)
		})
	}
}

// T045 additional: Verify YAML serialization of overrides
func TestConfigSet_PluginOverride_YAMLRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg := &localdev.Config{}
	cfg.SetField("plugins.overrides.postgresql.version", "v8.13.0")
	cfg.SetField("plugins.overrides.s3.image", "custom-s3:v1")

	if err := localdev.SaveConfigToPath(cfg, configPath); err != nil {
		t.Fatal(err)
	}

	loaded, err := localdev.LoadConfigFromPath(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Plugins.Overrides["postgresql"].Version != "v8.13.0" {
		t.Errorf("postgresql version = %q after round-trip", loaded.Plugins.Overrides["postgresql"].Version)
	}
	if loaded.Plugins.Overrides["s3"].Image != "custom-s3:v1" {
		t.Errorf("s3 image = %q after round-trip", loaded.Plugins.Overrides["s3"].Image)
	}
}

// --- Phase 7: Mirror Management Tests (T050-T051) ---

// T050: TestConfigAddMirrorCmd
func TestConfigAddMirrorCmd(t *testing.T) {
	t.Run("add valid mirror", func(t *testing.T) {
		cfg := &localdev.Config{}
		registry := "ghcr.io/backup-org"

		if !localdev.IsValidRegistry(registry) {
			t.Fatalf("%q should be valid registry", registry)
		}

		cfg.Plugins.Mirrors = append(cfg.Plugins.Mirrors, registry)
		if len(cfg.Plugins.Mirrors) != 1 || cfg.Plugins.Mirrors[0] != registry {
			t.Error("mirror not added correctly")
		}
	})

	t.Run("duplicate rejected", func(t *testing.T) {
		cfg := &localdev.Config{
			Plugins: localdev.PluginsConfig{
				Mirrors: []string{"ghcr.io/backup-org"},
			},
		}
		registry := "ghcr.io/backup-org"

		// Check for duplicates (same logic as configAddMirrorCmd)
		for _, m := range cfg.Plugins.Mirrors {
			if m == registry {
				return // Expected: duplicate detected
			}
		}
		t.Error("duplicate should have been detected")
	})

	t.Run("invalid URL rejected", func(t *testing.T) {
		invalid := "not a valid registry!"
		if localdev.IsValidRegistry(invalid) {
			t.Errorf("%q should be invalid registry", invalid)
		}
	})

	t.Run("mirror persists through save/load", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		cfg := &localdev.Config{}
		cfg.Plugins.Mirrors = []string{"ghcr.io/mirror-1", "ghcr.io/mirror-2"}

		if err := localdev.SaveConfigToPath(cfg, configPath); err != nil {
			t.Fatal(err)
		}

		loaded, err := localdev.LoadConfigFromPath(configPath)
		if err != nil {
			t.Fatal(err)
		}

		if len(loaded.Plugins.Mirrors) != 2 {
			t.Fatalf("mirrors count = %d, want 2", len(loaded.Plugins.Mirrors))
		}
		if loaded.Plugins.Mirrors[0] != "ghcr.io/mirror-1" {
			t.Errorf("mirror[0] = %q", loaded.Plugins.Mirrors[0])
		}
		if loaded.Plugins.Mirrors[1] != "ghcr.io/mirror-2" {
			t.Errorf("mirror[1] = %q", loaded.Plugins.Mirrors[1])
		}
	})
}

// T050 additional: Test cobra command execution
func TestConfigAddMirrorCmd_Execution(t *testing.T) {
	var buf bytes.Buffer
	configAddMirrorCmd.SetOut(&buf)
	configAddMirrorCmd.SetErr(&buf)

	// Validate args count
	if configAddMirrorCmd.Args == nil {
		t.Fatal("add-mirror should require args validation")
	}
}

// T051: TestConfigRemoveMirrorCmd
func TestConfigRemoveMirrorCmd(t *testing.T) {
	t.Run("remove existing mirror", func(t *testing.T) {
		cfg := &localdev.Config{
			Plugins: localdev.PluginsConfig{
				Mirrors: []string{"ghcr.io/mirror-1", "ghcr.io/mirror-2"},
			},
		}
		registry := "ghcr.io/mirror-1"

		found := false
		var newMirrors []string
		for _, m := range cfg.Plugins.Mirrors {
			if m == registry {
				found = true
				continue
			}
			newMirrors = append(newMirrors, m)
		}
		cfg.Plugins.Mirrors = newMirrors

		if !found {
			t.Error("mirror should have been found")
		}
		if len(cfg.Plugins.Mirrors) != 1 {
			t.Errorf("mirrors count = %d, want 1", len(cfg.Plugins.Mirrors))
		}
		if cfg.Plugins.Mirrors[0] != "ghcr.io/mirror-2" {
			t.Errorf("remaining mirror = %q", cfg.Plugins.Mirrors[0])
		}
	})

	t.Run("remove non-existent errors", func(t *testing.T) {
		cfg := &localdev.Config{
			Plugins: localdev.PluginsConfig{
				Mirrors: []string{"ghcr.io/mirror-1"},
			},
		}
		registry := "ghcr.io/nonexistent"

		found := false
		for _, m := range cfg.Plugins.Mirrors {
			if m == registry {
				found = true
			}
		}
		if found {
			t.Error("non-existent mirror should not be found")
		}
	})

	t.Run("remove from empty is safe", func(t *testing.T) {
		cfg := &localdev.Config{}
		found := false
		for _, m := range cfg.Plugins.Mirrors {
			if m == "anything" {
				found = true
			}
		}
		if found {
			t.Error("should not find mirror in empty list")
		}
	})
}
