package cmd

import (
	"github.com/spf13/cobra"
)

// storeCmd is the parent command for store operations.
var storeCmd = &cobra.Command{
	Use:   "store",
	Short: "Manage data stores",
	Long: `Manage data stores — named instances of connectors with connection details.

A Store manifest (stores/<name>/store.yaml) specifies the connector type,
connection parameters, and secret references.

Subcommands:
  create    Scaffold a new store
  list      List all stores in the project

Examples:
  # Create a PostgreSQL store
  dk store create pg-warehouse --connector postgres

  # Create an S3 store
  dk store create s3-raw --connector s3

  # List stores
  dk store list`,
}

func init() {
	rootCmd.AddCommand(storeCmd)
}
