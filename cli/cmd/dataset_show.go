package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	datasetShowOutput string
)

// datasetShowCmd displays details of a specific dataset.
var datasetShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a dataset",
	Long: `Display the full content of a named dataset, including extension metadata
and configuration.

The dataset is looked up by name across all type directories (sources, sinks, models).

Examples:
  # Show dataset details in YAML format
  dk dataset show aws-security

  # Show dataset details as JSON
  dk dataset show aws-security --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runDataSetShow,
}

func init() {
	datasetCmd.AddCommand(datasetShowCmd)

	datasetShowCmd.Flags().StringVarP(&datasetShowOutput, "output", "o", "yaml",
		"Output format: yaml or json")
}

func runDataSetShow(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()
	name := args[0]

	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	a, err := dataset.FindDataSetByName(projectDir, name)
	if err != nil {
		return fmt.Errorf("dataset %q not found: %w", name, err)
	}

	switch datasetShowOutput {
	case "json":
		return renderDataSetShowJSON(w, a)
	case "yaml":
		return renderDataSetShowYAML(w, a)
	default:
		return fmt.Errorf("unsupported output format: %s (use yaml or json)", datasetShowOutput)
	}
}

func renderDataSetShowJSON(w io.Writer, a *contracts.DataSetManifest) error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dataset as JSON: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func renderDataSetShowYAML(w io.Writer, a *contracts.DataSetManifest) error {
	data, err := yaml.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset as YAML: %w", err)
	}
	fmt.Fprint(w, string(data))
	return nil
}
