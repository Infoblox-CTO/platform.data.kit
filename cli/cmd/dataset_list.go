package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/spf13/cobra"
)

var (
	datasetListOutput string
)

// datasetListCmd lists all datasets in the project.
var datasetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all datasets in the project",
	Long: `List all datasets under the datasets/ directory, showing a summary table
with Name, Type, Extension, Version, and Owner.

Examples:
  # List all datasets in table format
  dk dataset list

  # List datasets as JSON
  dk dataset list --output json`,
	Args: cobra.NoArgs,
	RunE: runDataSetList,
}

func init() {
	datasetCmd.AddCommand(datasetListCmd)

	datasetListCmd.Flags().StringVarP(&datasetListOutput, "output", "o", "table",
		"Output format: table or json")
}

func runDataSetList(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()

	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	datasets, err := dataset.LoadAllDataSets(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load datasets: %w", err)
	}

	if len(datasets) == 0 {
		fmt.Fprintln(w, "No datasets found.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Get started:")
		fmt.Fprintln(w, "  dk dataset create <name>")
		return nil
	}

	switch datasetListOutput {
	case "json":
		return renderDataSetListJSON(w, datasets)
	case "table":
		return renderDataSetListTable(w, datasets)
	default:
		return fmt.Errorf("unsupported output format: %s (use table or json)", datasetListOutput)
	}
}

// datasetListEntry is the JSON representation of a dataset in the list.
type datasetListEntry struct {
	Name           string `json:"name"`
	Store          string `json:"store"`
	Classification string `json:"classification,omitempty"`
}

func renderDataSetListJSON(w io.Writer, datasets []*contracts.DataSetManifest) error {
	entries := make([]datasetListEntry, 0, len(datasets))
	for _, a := range datasets {
		entries = append(entries, datasetListEntry{
			Name:           a.Metadata.Name,
			Store:          a.Spec.Store,
			Classification: a.Spec.Classification,
		})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal datasets as JSON: %w", err)
	}

	fmt.Fprintln(w, string(data))
	return nil
}

func renderDataSetListTable(w io.Writer, datasets []*contracts.DataSetManifest) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTORE\tCLASSIFICATION")
	for _, a := range datasets {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			a.Metadata.Name, a.Spec.Store, a.Spec.Classification)
	}
	return tw.Flush()
}
