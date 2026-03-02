package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	assetShowOutput string
)

// assetShowCmd displays details of a specific asset.
var assetShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of an asset",
	Long: `Display the full content of a named asset, including extension metadata
and configuration.

The asset is looked up by name across all type directories (sources, sinks, models).

Examples:
  # Show asset details in YAML format
  dk asset show aws-security

  # Show asset details as JSON
  dk asset show aws-security --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runAssetShow,
}

func init() {
	assetCmd.AddCommand(assetShowCmd)

	assetShowCmd.Flags().StringVarP(&assetShowOutput, "output", "o", "yaml",
		"Output format: yaml or json")
}

func runAssetShow(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()
	name := args[0]

	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	a, err := asset.FindAssetByName(projectDir, name)
	if err != nil {
		return fmt.Errorf("asset %q not found: %w", name, err)
	}

	switch assetShowOutput {
	case "json":
		return renderAssetShowJSON(w, a)
	case "yaml":
		return renderAssetShowYAML(w, a)
	default:
		return fmt.Errorf("unsupported output format: %s (use yaml or json)", assetShowOutput)
	}
}

func renderAssetShowJSON(w io.Writer, a *contracts.AssetManifest) error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal asset as JSON: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func renderAssetShowYAML(w io.Writer, a *contracts.AssetManifest) error {
	data, err := yaml.Marshal(a)
	if err != nil {
		return fmt.Errorf("failed to marshal asset as YAML: %w", err)
	}
	fmt.Fprint(w, string(data))
	return nil
}
