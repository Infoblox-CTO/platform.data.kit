// Package cmd contains all CLI commands for dp.
package cmd

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/spf13/cobra"
)

var configScope string

// configCmd is the parent command for config management.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dp configuration",
	Long: `Manage dp CLI configuration settings.

Configuration is stored in YAML files at three scopes (highest to lowest precedence):
  repo:   {git-root}/.dp/config.yaml
  user:   ~/.config/dp/config.yaml
  system: /etc/datakit/config.yaml

Use subcommands to set, get, unset, and list configuration values.`,
}

// configSetCmd sets a config value.
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value in the specified scope.

Valid keys:
  dev.runtime                            Runtime type (k3d, compose)
  dev.workspace                          Path to DP workspace
  dev.k3d.clusterName                    k3d cluster name
  plugins.registry                       Default OCI registry for plugins
  plugins.overrides.<name>.version       Override version for a plugin
  plugins.overrides.<name>.image         Override image for a plugin`,
	Example: `  # Set default plugin registry
  dp config set plugins.registry ghcr.io/myteam

  # Pin a plugin version
  dp config set plugins.overrides.postgresql.version v8.13.0

  # Set for this project only
  dp config set plugins.registry internal.registry.io --scope repo`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		if err := localdev.ValidateField(key, value); err != nil {
			return err
		}

		scope := localdev.ConfigScope(configScope)
		cfg, err := localdev.LoadConfigForScope(scope)
		if err != nil {
			return err
		}

		if err := cfg.SetField(key, value); err != nil {
			return err
		}

		if err := localdev.SaveConfigForScope(cfg, scope); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s (scope: %s)\n", key, value, scope)
		return nil
	},
}

// configGetCmd gets a config value.
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get the effective value of a configuration key",
	Long: `Get the effective value of a configuration key.

Shows the resolved value and which scope it comes from (repo, user, system, or built-in).`,
	Example: `  dp config get plugins.registry
  dp config get dev.runtime`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		value, source, err := localdev.EffectiveValue(key)
		if err != nil {
			return err
		}

		if value == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "(not set)\n")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%s (source: %s)\n", value, source)
		}
		return nil
	},
}

// configUnsetCmd removes a config value.
var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a configuration value from a scope",
	Long:  `Remove a configuration value from the specified scope, reverting to the next lower scope's value or the built-in default.`,
	Example: `  dp config unset plugins.registry
  dp config unset plugins.overrides.postgresql.version --scope repo`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		scope := localdev.ConfigScope(configScope)
		cfg, err := localdev.LoadConfigForScope(scope)
		if err != nil {
			return err
		}

		if err := cfg.UnsetField(key); err != nil {
			return err
		}

		if err := localdev.SaveConfigForScope(cfg, scope); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Unset %s (scope: %s)\n", key, scope)
		return nil
	},
}

// configListCmd lists all effective settings.
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all effective configuration settings",
	Long:  `List all configuration settings showing the effective value and source scope for each key.`,
	Example: `  dp config list
  dp config list --scope repo`,
	RunE: func(cmd *cobra.Command, args []string) error {
		formatter := GetFormatter()

		// Collect all keys to display
		type configEntry struct {
			Key    string
			Value  string
			Source string
		}
		var entries []configEntry

		if configScope != "" {
			// Show settings from a specific scope only
			scope := localdev.ConfigScope(configScope)
			cfg, err := localdev.LoadConfigForScope(scope)
			if err != nil {
				return err
			}
			for _, key := range localdev.AllConfigKeys() {
				if v, ok := cfg.GetField(key); ok {
					entries = append(entries, configEntry{Key: key, Value: v, Source: string(scope)})
				}
			}
			// Also check overrides
			for name, o := range cfg.Plugins.Overrides {
				if o.Version != "" {
					entries = append(entries, configEntry{
						Key:    fmt.Sprintf("plugins.overrides.%s.version", name),
						Value:  o.Version,
						Source: string(scope),
					})
				}
				if o.Image != "" {
					entries = append(entries, configEntry{
						Key:    fmt.Sprintf("plugins.overrides.%s.image", name),
						Value:  o.Image,
						Source: string(scope),
					})
				}
			}
			// Also check mirrors
			for i, m := range cfg.Plugins.Mirrors {
				entries = append(entries, configEntry{
					Key:    fmt.Sprintf("plugins.mirrors[%d]", i),
					Value:  m,
					Source: string(scope),
				})
			}
		} else {
			// Show effective values from all scopes
			for _, key := range localdev.AllConfigKeys() {
				value, source, err := localdev.EffectiveValue(key)
				if err != nil {
					continue
				}
				if value != "" {
					entries = append(entries, configEntry{Key: key, Value: value, Source: source})
				}
			}
			// Also show overrides and mirrors from hierarchical config
			cfg, err := localdev.LoadHierarchicalConfig()
			if err == nil {
				for name, o := range cfg.Plugins.Overrides {
					if o.Version != "" {
						entries = append(entries, configEntry{
							Key:   fmt.Sprintf("plugins.overrides.%s.version", name),
							Value: o.Version,
						})
					}
					if o.Image != "" {
						entries = append(entries, configEntry{
							Key:   fmt.Sprintf("plugins.overrides.%s.image", name),
							Value: o.Image,
						})
					}
				}
				for i, m := range cfg.Plugins.Mirrors {
					entries = append(entries, configEntry{
						Key:   fmt.Sprintf("plugins.mirrors[%d]", i),
						Value: m,
					})
				}
			}
		}

		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No configuration settings found.")
			return nil
		}

		// Build table output
		headers := []string{"KEY", "VALUE", "SOURCE"}
		var rows [][]string
		for _, e := range entries {
			rows = append(rows, []string{e.Key, e.Value, e.Source})
		}

		return formatter.FormatTable(cmd.OutOrStdout(), headers, rows)
	},
}

// configAddMirrorCmd adds a fallback registry mirror.
var configAddMirrorCmd = &cobra.Command{
	Use:   "add-mirror <registry>",
	Short: "Add a fallback registry mirror",
	Long:  `Add a fallback registry mirror that is tried when the primary registry is unreachable.`,
	Example: `  dp config add-mirror ghcr.io/backup-org
  dp config add-mirror internal.registry.io --scope repo`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		registry := args[0]

		if !localdev.IsValidRegistry(registry) {
			return fmt.Errorf("invalid registry URL %q", registry)
		}

		scope := localdev.ConfigScope(configScope)
		cfg, err := localdev.LoadConfigForScope(scope)
		if err != nil {
			return err
		}

		// Check for duplicates
		for _, m := range cfg.Plugins.Mirrors {
			if m == registry {
				return fmt.Errorf("mirror %q already exists", registry)
			}
		}

		cfg.Plugins.Mirrors = append(cfg.Plugins.Mirrors, registry)

		if err := localdev.SaveConfigForScope(cfg, scope); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Added mirror %s (scope: %s)\n", registry, scope)
		return nil
	},
}

// configRemoveMirrorCmd removes a fallback registry mirror.
var configRemoveMirrorCmd = &cobra.Command{
	Use:   "remove-mirror <registry>",
	Short: "Remove a fallback registry mirror",
	Long:  `Remove a fallback registry mirror from the specified scope.`,
	Example: `  dp config remove-mirror ghcr.io/backup-org
  dp config remove-mirror internal.registry.io --scope repo`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		registry := args[0]

		scope := localdev.ConfigScope(configScope)
		cfg, err := localdev.LoadConfigForScope(scope)
		if err != nil {
			return err
		}

		found := false
		var newMirrors []string
		for _, m := range cfg.Plugins.Mirrors {
			if m == registry {
				found = true
				continue
			}
			newMirrors = append(newMirrors, m)
		}

		if !found {
			return fmt.Errorf("mirror %q not found in %s config", registry, scope)
		}

		cfg.Plugins.Mirrors = newMirrors

		if err := localdev.SaveConfigForScope(cfg, scope); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Removed mirror %s (scope: %s)\n", registry, scope)
		return nil
	},
}

func init() {
	// Register config command with root
	rootCmd.AddCommand(configCmd)

	// Register subcommands
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configAddMirrorCmd)
	configCmd.AddCommand(configRemoveMirrorCmd)

	// Add --scope flag to commands that need it
	configSetCmd.Flags().StringVar(&configScope, "scope", "user", "Config scope: repo, user, or system")
	configUnsetCmd.Flags().StringVar(&configScope, "scope", "user", "Config scope: repo, user, or system")
	configListCmd.Flags().StringVar(&configScope, "scope", "", "Show settings from a specific scope only")
	configAddMirrorCmd.Flags().StringVar(&configScope, "scope", "user", "Config scope: repo, user, or system")
	configRemoveMirrorCmd.Flags().StringVar(&configScope, "scope", "user", "Config scope: repo, user, or system")
}
