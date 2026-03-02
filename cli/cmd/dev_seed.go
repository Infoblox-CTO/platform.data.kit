package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
)

var (
	seedClean   bool
	seedForce   bool
	seedAsset   string
	seedProfile string
)

// devSeedCmd seeds development data into local stores.
var devSeedCmd = &cobra.Command{
	Use:   "seed [package-dir]",
	Short: "Load seed data into local dev stores",
	Long: `Create tables and insert sample data defined in asset dev.seed sections.

This command reads each input asset in the package and, for assets that
declare a dev.seed section, generates CREATE TABLE and INSERT statements
and runs them against the backing database in the local k3d cluster.

The dev.seed section supports two data sources:
  - inline: rows defined directly in the asset YAML
  - file:   path to a CSV or JSON file with seed rows

Seed runs are idempotent — a checksum of the seed data is stored in a
_dk_seed_meta table and compared on subsequent runs. If the data hasn't
changed, the seed is skipped entirely.

Named profiles let you maintain multiple data sets for different test
scenarios (e.g. "large", "edge-cases", "empty") in the same asset YAML
under dev.seed.profiles.<name>.

Examples:
  # Seed all input assets (skips if data unchanged)
  dk dev seed

  # Use a named seed profile
  dk dev seed --profile large-dataset

  # Force re-seed even if data unchanged
  dk dev seed --force

  # Seed a specific asset
  dk dev seed --asset foo-source-table

  # Drop and recreate tables before seeding
  dk dev seed --clean

  # Seed from a specific package directory
  dk dev seed ./my-pipeline`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDevSeed,
}

func init() {
	devCmd.AddCommand(devSeedCmd)

	devSeedCmd.Flags().BoolVar(&seedClean, "clean", false, "Drop and recreate tables before seeding")
	devSeedCmd.Flags().BoolVar(&seedForce, "force", false, "Re-seed even when data is unchanged")
	devSeedCmd.Flags().StringVar(&seedAsset, "asset", "", "Seed only a specific asset by name")
	devSeedCmd.Flags().StringVar(&seedProfile, "profile", "", "Use a named seed profile instead of the default")
}

func runDevSeed(cmd *cobra.Command, args []string) error {
	packageDir := "."
	if len(args) > 0 {
		packageDir = args[0]
	}

	absDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify dk.yaml exists.
	dpPath := filepath.Join(absDir, "dk.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		return fmt.Errorf("dk.yaml not found in %s — is this a valid DK package?", packageDir)
	}

	ctx := context.Background()

	opts := runner.SeedOptions{
		PackageDir:  absDir,
		Clean:       seedClean,
		Force:       seedForce,
		Profile:     seedProfile,
		AssetFilter: seedAsset,
		Output:      os.Stdout,
	}

	result, err := runner.SeedPackage(ctx, opts)
	if err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}

	if result.AssetsSeeded == 0 && result.AssetsSkipped == 0 {
		fmt.Println("No assets with dev.seed found. Add a dev.seed section to your asset YAML:")
		fmt.Println()
		fmt.Println("  spec:")
		fmt.Println("    dev:")
		fmt.Println("      seed:")
		fmt.Println("        inline:")
		fmt.Println("          - { id: 1, name: \"alice\" }")
		return nil
	}

	if result.AssetsSkipped > 0 {
		fmt.Printf("\n✓ Seeded %d asset(s), %d row(s) inserted, %d unchanged (skipped)\n",
			result.AssetsSeeded, result.RowsInserted, result.AssetsSkipped)
	} else {
		fmt.Printf("\n✓ Seeded %d asset(s), %d row(s) inserted\n", result.AssetsSeeded, result.RowsInserted)
	}
	return nil
}
