package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	"github.com/spf13/cobra"
)

var (
	assetListOutput string
)

// assetListCmd lists all assets in the project.
var assetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all assets in the project",
	Long: `List all assets under the assets/ directory, showing a summary table
with Name, Type, Extension, Version, and Owner.

Examples:
  # List all assets in table format
  dp asset list

  # List assets as JSON
  dp asset list --output json`,
	Args: cobra.NoArgs,
	RunE: runAssetList,
}

func init() {
	assetCmd.AddCommand(assetListCmd)

	assetListCmd.Flags().StringVarP(&assetListOutput, "output", "o", "table",
		"Output format: table or json")
}

func runAssetList(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()

	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	assets, err := asset.LoadAllAssets(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load assets: %w", err)
	}

	if len(assets) == 0 {
		fmt.Fprintln(w, "No assets found.")
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Get started:")
		fmt.Fprintln(w, "  dp asset create <name> --ext <vendor.kind.name>")
		return nil
	}

	switch assetListOutput {
	case "json":
		return renderAssetListJSON(w, assets)
	case "table":
		return renderAssetListTable(w, assets)
	default:
		return fmt.Errorf("unsupported output format: %s (use table or json)", assetListOutput)
	}
}

// assetListEntry is the JSON representation of an asset in the list.
type assetListEntry struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Extension string `json:"extension"`
	Version   string `json:"version"`
	OwnerTeam string `json:"ownerTeam"`
}

func renderAssetListJSON(w io.Writer, assets []*contracts.AssetManifest) error {
	entries := make([]assetListEntry, 0, len(assets))
	for _, a := range assets {
		entries = append(entries, assetListEntry{
			Name:      a.Name,
			Type:      string(a.Type),
			Extension: a.Extension,
			Version:   a.Version,
			OwnerTeam: a.OwnerTeam,
		})
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal assets as JSON: %w", err)
	}

	fmt.Fprintln(w, string(data))
	return nil
}

func renderAssetListTable(w io.Writer, assets []*contracts.AssetManifest) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tEXTENSION\tVERSION\tOWNER")
	for _, a := range assets {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			a.Name, a.Type, a.Extension, a.Version, a.OwnerTeam)
	}
	return tw.Flush()
}
