package cmd

import (
	"github.com/spf13/cobra"
)

// datasetCmd is the parent command for all dataset subcommands.
var datasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Manage data package datasets",
	Long: `Manage data package datasets — data contracts that declare where data lives,
what it looks like (schema), and how it is classified.

A DataSet manifest (datasets/<name>/dataset.yaml) specifies the store, table/
prefix/topic, format, classification, and optionally an inline schema or an APX
schema reference (schemaRef).

Subcommands:
  create    Scaffold a new dataset with inline schema
  add       Reference an external dataset via APX schema
  validate  Validate dataset manifests
  list      List all datasets in the project
  show      Display details of a specific dataset

Examples:
  # Create a new dataset with inline schema
  dk dataset create aws-security --store my-s3

  # Reference an external schema (APX)
  dk dataset add users@^1.0.0 --store my-pg

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
