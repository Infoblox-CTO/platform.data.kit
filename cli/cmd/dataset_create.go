package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/spf13/cobra"
)

var (
	datasetCreateForce bool
	datasetCreateStore string
)

// datasetCreateCmd scaffolds a new dataset.
var datasetCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new dataset",
	Long: `Scaffold a new dataset.yaml in datasets/<name>/dataset.yaml.

Examples:
  # Create a basic dataset
  dk dataset create aws-security

  # Create with a pre-filled store reference
  dk dataset create aws-security --store my-s3

  # Overwrite existing dataset
  dk dataset create aws-security --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDataSetCreate,
}

func init() {
	datasetCmd.AddCommand(datasetCreateCmd)

	datasetCreateCmd.Flags().BoolVar(&datasetCreateForce, "force", false,
		"Overwrite existing dataset")
	datasetCreateCmd.Flags().StringVar(&datasetCreateStore, "store", "",
		"Store name to reference in spec.store")
}

func runDataSetCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if err := dataset.ValidateDataSetName(name); err != nil {
		return err
	}

	// Determine project directory
	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Build scaffold options
	opts := dataset.ScaffoldOpts{
		Name:       name,
		ProjectDir: projectDir,
		Force:      datasetCreateForce,
		Store:      datasetCreateStore,
	}

	result, err := dataset.Scaffold(opts)
	if err != nil {
		return err
	}

	relPath, _ := filepath.Rel(projectDir, result.DataSetPath)
	if relPath == "" {
		relPath = result.DataSetPath
	}

	cmd.Printf("Created dataset %q at %s\n", name, relPath)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit %s to fill in spec.store and locators\n", relPath)
	cmd.Printf("  2. Run 'dk dataset validate' to validate\n")

	return nil
}
