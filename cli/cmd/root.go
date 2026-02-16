// Package cmd contains all CLI commands for dp.
package cmd

import (
	"os"

	"github.com/Infoblox-CTO/platform.data.kit/cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	// outputFormat is the global output format flag
	outputFormat string

	// Version is set at build time
	Version = "dev"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dp",
	Short: "DP - Data Platform CLI",
	Long: `DP (Data Platform) is a Kubernetes-native data pipeline platform
enabling teams to contribute reusable, versioned "data packages" with
a complete developer workflow.

Workflow: init -> dev -> run -> lint -> test -> build -> publish -> promote

Example:
  # Create a new model package
  dp init my-model --kind model --runtime cloudquery

  # Start local development environment
  dp dev up

  # Validate manifest files
  dp lint

  # Run pipeline locally
  dp run

  # Build and publish package
  dp build
  dp publish

  # Promote to next environment
  dp promote my-pipeline v1.0.0 --to int`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table",
		"Output format: table, json, yaml")
	rootCmd.AddCommand(versionCmd)
}

// GetOutputFormat returns the current output format.
func GetOutputFormat() output.Format {
	return output.ParseFormat(outputFormat)
}

// GetFormatter returns a formatter for the current output format.
func GetFormatter() output.Formatter {
	return output.NewFormatter(GetOutputFormat())
}

// versionCmd prints the CLI version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("dp version %s\n", Version)
	},
}
