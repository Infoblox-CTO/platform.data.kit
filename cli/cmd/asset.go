package cmd

import (
	"github.com/spf13/cobra"
)

// assetCmd is the parent command for all asset subcommands.
var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Manage data package assets",
	Long: `Manage data package assets — configured instances of approved extensions.

Assets are config-only YAML files that reference an extension by fully-qualified
name (FQN) and version, with a config block validated against the extension's
JSON Schema.

Subcommands:
  create    Scaffold a new asset from an extension
  validate  Validate asset config against extension schema
  list      List all assets in the project
  show      Display details of a specific asset

Examples:
  # Create a new source asset
  dp asset create aws-security --ext cloudquery.source.aws

  # Validate all assets
  dp asset validate

  # List assets
  dp asset list

  # Show asset details
  dp asset show aws-security`,
}

func init() {
	rootCmd.AddCommand(assetCmd)
}
