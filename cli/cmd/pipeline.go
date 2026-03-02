package cmd

import (
	"github.com/spf13/cobra"
)

// pipelineCmd is the parent command for all pipeline subcommands.
var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage pipeline workflows",
	Long: `Manage pipeline workflows — multi-step execution plans for data pipelines.

A pipeline workflow defines the ordered steps (sync, transform, test, publish,
custom) that compose a data pipeline, along with retry and notification config.

Subcommands:
  create    Scaffold a new pipeline.yaml from a template

Examples:
  # Create a pipeline from the default template
  dk pipeline create my-pipeline

  # Create with a specific template
  dk pipeline create my-pipeline --template sync-only

  # List available templates
  dk pipeline create --list-templates

  # Overwrite existing pipeline.yaml
  dk pipeline create my-pipeline --template custom --force`,
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
}
