package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	"github.com/spf13/cobra"
)

var (
	assetCreateForce bool
	assetCreateStore string
)

// assetCreateCmd scaffolds a new asset.
var assetCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new asset",
	Long: `Scaffold a new asset.yaml in assets/<name>/asset.yaml.

Examples:
  # Create a basic asset
  dk asset create aws-security

  # Create with a pre-filled store reference
  dk asset create aws-security --store my-s3

  # Overwrite existing asset
  dk asset create aws-security --force`,
	Args: cobra.ExactArgs(1),
	RunE: runAssetCreate,
}

func init() {
	assetCmd.AddCommand(assetCreateCmd)

	assetCreateCmd.Flags().BoolVar(&assetCreateForce, "force", false,
		"Overwrite existing asset")
	assetCreateCmd.Flags().StringVar(&assetCreateStore, "store", "",
		"Store name to reference in spec.store")
}

func runAssetCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if err := asset.ValidateAssetName(name); err != nil {
		return err
	}

	// Determine project directory
	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Build scaffold options
	opts := asset.ScaffoldOpts{
		Name:       name,
		ProjectDir: projectDir,
		Force:      assetCreateForce,
		Store:      assetCreateStore,
	}

	result, err := asset.Scaffold(opts)
	if err != nil {
		return err
	}

	relPath, _ := filepath.Rel(projectDir, result.AssetPath)
	if relPath == "" {
		relPath = result.AssetPath
	}

	cmd.Printf("Created asset %q at %s\n", name, relPath)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit %s to fill in spec.store and locators\n", relPath)
	cmd.Printf("  2. Run 'dk asset validate' to validate\n")

	return nil
}
