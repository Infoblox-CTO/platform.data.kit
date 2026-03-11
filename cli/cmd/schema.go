package cmd

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/schema"
	"github.com/spf13/cobra"
)

// schemaCmd is the parent command for schema management.
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage dataset schemas via APX catalog",
	Long: `Search, inspect, and check schemas from the APX schema catalog.

Schemas are versioned, format-aware modules (Parquet, Avro, JSON Schema,
OpenAPI, Protobuf) managed by the APX tool. The dk schema commands let you
browse the catalog and verify compatibility without leaving the DK workflow.

Subcommands:
  search    Search the schema catalog
  show      Display schema details and fields
  check     Check for breaking changes against the lock file

Examples:
  # Search for user-related schemas
  dk schema search users

  # Show schema details
  dk schema show users@1.0.0

  # Check for breaking changes
  dk schema check users`,
}

func init() {
	rootCmd.AddCommand(schemaCmd)
}

// --- schema search ---

var schemaSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the schema catalog",
	Long: `Search the APX schema catalog for modules matching the query.

Examples:
  dk schema search users
  dk schema search billing --format parquet`,
	Args: cobra.ExactArgs(1),
	RunE: runSchemaSearch,
}

func init() {
	schemaCmd.AddCommand(schemaSearchCmd)
}

func runSchemaSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// TODO: When APX exports its catalog packages, use SchemaResolver.Search()
	// For now, indicate that the catalog is not yet configured.
	cmd.Printf("Searching schema catalog for %q...\n\n", query)
	cmd.Println("No catalog source configured.")
	cmd.Println("To enable schema search, configure an APX catalog source:")
	cmd.Println("  apx catalog add <repo-url>")
	cmd.Println()
	cmd.Println("Once a catalog is configured, dk schema search will query it")
	cmd.Println("for matching schema modules.")

	return nil
}

// --- schema show ---

var schemaShowCmd = &cobra.Command{
	Use:   "show <module[@version]>",
	Short: "Display schema details and fields",
	Long: `Show the fields, metadata, and version history of a schema module.

Examples:
  dk schema show users
  dk schema show users@1.0.0
  dk schema show orders@^2.0 --output json`,
	Args: cobra.ExactArgs(1),
	RunE: runSchemaShow,
}

func init() {
	schemaCmd.AddCommand(schemaShowCmd)
}

func runSchemaShow(cmd *cobra.Command, args []string) error {
	ref := args[0]
	module, constraint := schema.ParseSchemaRef(ref)

	cmd.Printf("Schema: %s\n", module)
	if constraint != "" {
		cmd.Printf("Constraint: %s\n", constraint)
	}
	cmd.Println()

	// TODO: When APX exports its catalog packages, resolve and display fields.
	cmd.Println("No catalog source configured.")
	cmd.Println("To view schema details, configure an APX catalog source:")
	cmd.Println("  apx catalog add <repo-url>")

	return nil
}

// --- schema check ---

var schemaCheckCmd = &cobra.Command{
	Use:   "check [module]",
	Short: "Check for breaking schema changes",
	Long: `Check for breaking changes between the locked schema version and the
current version in the catalog.

When run without arguments, checks all schemas in dk.lock.
When given a module name, checks only that module.

Examples:
  dk schema check
  dk schema check users`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSchemaCheck,
}

func init() {
	schemaCmd.AddCommand(schemaCheckCmd)
}

func runSchemaCheck(cmd *cobra.Command, args []string) error {
	// Read lock file
	lock, err := schema.ReadLockFile(".")
	if err != nil {
		return err
	}
	if lock == nil {
		return fmt.Errorf("dk.lock not found — run 'dk lock' first")
	}

	if len(lock.Schemas) == 0 {
		cmd.Println("No schemas in dk.lock — nothing to check.")
		return nil
	}

	// Filter to specific module if provided
	var toCheck []string
	if len(args) > 0 {
		module := args[0]
		if schema.FindLockedSchema(lock, module) == nil {
			return fmt.Errorf("module %q not found in dk.lock", module)
		}
		toCheck = []string{module}
	} else {
		for _, s := range lock.Schemas {
			toCheck = append(toCheck, s.Module)
		}
	}

	cmd.Printf("Checking %d schema(s) for breaking changes...\n\n", len(toCheck))

	// TODO: When APX exports its validator packages, run Breaking() checks.
	for _, module := range toCheck {
		locked := schema.FindLockedSchema(lock, module)
		cmd.Printf("  • %s@%s — ", module, locked.Version)
		cmd.Println("skipped (no catalog source configured)")
	}

	cmd.Println()
	cmd.Println("To enable breaking change detection, configure an APX catalog source:")
	cmd.Println("  apx catalog add <repo-url>")

	return nil
}
