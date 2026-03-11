package cmd

import (
	"github.com/spf13/cobra"
)

// projectCmd is the parent command for project-level operations.
var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage DataKit projects",
	Long: `Manage multi-transform DataKit projects.

A project is a directory containing connectors, stores, datasets, and transforms
that together form a data pipeline.

Subcommands:
  init    Scaffold a new project directory

Examples:
  # Create a new project
  dk project init k8s-analytics

  # Create a project in the current directory
  dk project init .`,
}

func init() {
	rootCmd.AddCommand(projectCmd)
}
