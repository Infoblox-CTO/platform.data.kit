package cmd

import (
	"github.com/spf13/cobra"
)

// pipelineCmd is the parent command for pipeline subcommands.
var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Inspect the pipeline dependency graph",
	Long: `Inspect the reactive pipeline dependency graph derived from Transform and
DataSet manifests (dk.yaml files).

Subcommands:
  show    Display pipeline dependency graph

Examples:
  # Show full dependency graph
  dk pipeline show

  # Show graph leading to a specific dataset
  dk pipeline show --destination event-summary

  # Render as Mermaid diagram
  dk pipeline show --output mermaid`,
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
}
