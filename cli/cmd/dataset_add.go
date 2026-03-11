package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/schema"
	"github.com/spf13/cobra"
)

var (
	datasetAddStore string
	datasetAddForce bool
)

// datasetAddCmd references an external dataset via APX schema.
var datasetAddCmd = &cobra.Command{
	Use:   "add <schema-ref>",
	Short: "Reference an external dataset via APX schema",
	Long: `Create a dataset.yaml that references an APX schema module instead of
defining an inline schema. This lets you consume a dataset contract owned by
another team without duplicating the schema definition.

The schema-ref format is "module" or "module@constraint" (e.g., "users@^1.0.0").

After adding, run 'dk lock' to resolve and pin the schema version.

Examples:
  # Reference a schema
  dk dataset add users@^1.0.0

  # Reference with a store
  dk dataset add users@^1.0.0 --store my-pg

  # Overwrite existing dataset
  dk dataset add users@^1.0.0 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDataSetAdd,
}

func init() {
	datasetCmd.AddCommand(datasetAddCmd)

	datasetAddCmd.Flags().StringVar(&datasetAddStore, "store", "",
		"Store name to reference in spec.store")
	datasetAddCmd.Flags().BoolVar(&datasetAddForce, "force", false,
		"Overwrite existing dataset")
}

func runDataSetAdd(cmd *cobra.Command, args []string) error {
	ref := args[0]
	module, _ := schema.ParseSchemaRef(ref)

	// Use the module name as the dataset name.
	name := module

	// Validate name
	if err := dataset.ValidateDataSetName(name); err != nil {
		return err
	}

	// Determine project directory
	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Check if datasets directory exists; create if needed.
	datasetsDir := filepath.Join(projectDir, "datasets", name)
	datasetPath := filepath.Join(datasetsDir, "dataset.yaml")

	if _, err := os.Stat(datasetPath); err == nil && !datasetAddForce {
		return fmt.Errorf("dataset %q already exists at %s (use --force to overwrite)",
			name, filepath.Join("datasets", name, "dataset.yaml"))
	}

	if err := os.MkdirAll(datasetsDir, 0o755); err != nil {
		return fmt.Errorf("creating dataset directory: %w", err)
	}

	// Build the dataset YAML with schemaRef instead of inline schema.
	store := datasetAddStore
	if store == "" {
		store = "# TODO: specify store name"
	}

	yaml := fmt.Sprintf(`apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: %s
spec:
  store: %s
  schemaRef: "%s"
`, name, store, ref)

	if err := os.WriteFile(datasetPath, []byte(yaml), 0o644); err != nil {
		return fmt.Errorf("writing dataset.yaml: %w", err)
	}

	relPath := filepath.Join("datasets", name, "dataset.yaml")
	cmd.Printf("Created dataset %q at %s (schema ref: %s)\n", name, relPath, ref)
	cmd.Printf("\nNext steps:\n")
	if datasetAddStore == "" {
		cmd.Printf("  1. Edit %s to set spec.store and add a locator (table/prefix/topic)\n", relPath)
	} else {
		cmd.Printf("  1. Edit %s to add a locator (table/prefix/topic)\n", relPath)
	}
	cmd.Printf("  2. Run 'dk lock' to resolve and pin the schema version\n")
	cmd.Printf("  3. Run 'dk lint' to validate\n")

	return nil
}
