package cmd

import (
	"github.com/spf13/cobra"
)

// datasetCmd is the parent command for all dataset subcommands.
var datasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Manage data package datasets",
	Long: `Manage data package datasets — configured instances of approved extensions.

DataSets are config-only YAML files that reference an extension by fully-qualified
name (FQN) and version, with a config block validated against the extension's
JSON Schema.

Subcommands:
  create    Scaffold a new dataset from an extension
  validate  Validate dataset config against extension schema
  list      List all datasets in the project
  show      Display details of a specific dataset

Examples:
  # Create a new source dataset
  dk dataset create aws-security --ext cloudquery.source.aws

  # Validate all datasets
  dk dataset validate

  # List datasets
  dk dataset list

  # Show dataset details
  dk dataset show aws-security`,
}

func init() {
	rootCmd.AddCommand(datasetCmd)
}
