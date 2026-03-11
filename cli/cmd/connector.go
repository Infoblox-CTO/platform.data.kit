package cmd

import (
	"github.com/spf13/cobra"
)

// connectorCmd is the parent command for connector operations.
var connectorCmd = &cobra.Command{
	Use:   "connector",
	Short: "Manage technology connectors",
	Long: `Manage technology connectors — definitions of storage technology types.

A Connector manifest (connectors/<name>/connector.yaml) declares a storage
technology type (Postgres, S3, Kafka, etc.), its capabilities, and which
CloudQuery plugin images to use.

Connectors are typically managed by the platform team and referenced by
Stores that data engineers create.

Subcommands:
  create    Scaffold a new connector
  list      List all connectors in the project

Examples:
  # Create a PostgreSQL connector
  dk connector create postgres-analytics --type postgres

  # Create an S3 connector
  dk connector create s3-datalake --type s3

  # List connectors
  dk connector list`,
}

func init() {
	rootCmd.AddCommand(connectorCmd)
}
