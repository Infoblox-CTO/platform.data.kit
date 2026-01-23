// Package cmd contains the CLI commands for DP.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/promotion"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <package>",
	Short: "Rollback a package to a previous version",
	Long: `Rollback a data package to a previous version in an environment.

Rollback is implemented as a promotion of the previous version,
creating a PR to update the environment overlay.

Example:
  # Rollback to previous version in prod
  dp rollback kafka-s3-pipeline --environment prod

  # Rollback to a specific version
  dp rollback kafka-s3-pipeline --environment prod --to-version v1.0.0

  # Dry run to see what would happen
  dp rollback kafka-s3-pipeline --environment prod --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runRollback,
}

var (
	rollbackEnvironment string
	rollbackToVersion   string
	rollbackDryRun      bool
)

func init() {
	rootCmd.AddCommand(rollbackCmd)

	rollbackCmd.Flags().StringVarP(&rollbackEnvironment, "environment", "e", "", "Target environment (dev, int, prod)")
	rollbackCmd.Flags().StringVar(&rollbackToVersion, "to-version", "", "Specific version to rollback to")
	rollbackCmd.Flags().BoolVar(&rollbackDryRun, "dry-run", false, "Simulate the rollback without creating a PR")

	rollbackCmd.MarkFlagRequired("environment")
}

func runRollback(cmd *cobra.Command, args []string) error {
	packageName := args[0]

	// Validate environment
	targetEnv := promotion.Environment(rollbackEnvironment)
	if !targetEnv.Valid() {
		return fmt.Errorf("invalid environment: %s (must be dev, int, or prod)", rollbackEnvironment)
	}

	// Get GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" && !rollbackDryRun {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Determine the version to rollback to
	rollbackVersion := rollbackToVersion
	if rollbackVersion == "" {
		// Get previous version from deployment history
		// For MVP, we'll require explicit version
		return fmt.Errorf("--to-version is required (automatic previous version detection coming soon)")
	}

	fmt.Printf("Rolling back %s in %s to %s...\n", packageName, targetEnv, rollbackVersion)

	if rollbackDryRun {
		fmt.Println("\n[DRY RUN] Would create rollback PR with:")
		fmt.Printf("  Package:     %s\n", packageName)
		fmt.Printf("  Environment: %s\n", targetEnv)
		fmt.Printf("  Version:     %s\n", rollbackVersion)
		fmt.Printf("  Branch:      rollback/%s/%s/%s/...\n", packageName, targetEnv, rollbackVersion)
		fmt.Println("\nNo changes made.")
		return nil
	}

	// Get repository info
	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		owner = "Infoblox-CTO"
	}
	repo := os.Getenv("GITHUB_REPO")
	if repo == "" {
		repo = "data-platform"
	}

	// Create GitHub client
	ghClient := promotion.NewGitHubClient(token, owner, repo)

	// Create promotion request (rollback is just a promotion to an older version)
	req := &promotion.PromotionRequest{
		Package:   packageName,
		Version:   rollbackVersion,
		TargetEnv: targetEnv,
		DryRun:    rollbackDryRun,
	}

	promoter := &promotion.Promoter{
		GitHubClient: ghClient,
		BaseBranch:   "main",
		RecordGenerator: promotion.NewRecordGenerator().WithInitiator(
			os.Getenv("USER"),
		),
	}

	result, err := promoter.Promote(ctx, req)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	if result.Success {
		fmt.Println("\n✓ Rollback PR created successfully!")
		fmt.Printf("  PR #%d: %s\n", result.PRNumber, result.PRURL)
		fmt.Printf("  Branch: %s\n", result.Branch)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Review the rollback PR")
		fmt.Println("  2. Approve and merge")
		fmt.Println("  3. ArgoCD will deploy the previous version")
	}

	return nil
}

// getPreviousVersion would query the deployment history to find the previous version.
// For MVP, this is not implemented - users must specify --to-version.
func getPreviousVersion(ctx context.Context, packageName string, env promotion.Environment) (string, error) {
	// This would:
	// 1. Query the run history for the environment
	// 2. Find the previous successful deployment
	// 3. Return that version
	//
	// For MVP, we require explicit version specification
	return "", fmt.Errorf("automatic previous version detection not yet implemented")
}

// listAvailableVersions would list all available versions for rollback.
func listAvailableVersions(ctx context.Context, packageName string, env promotion.Environment) ([]string, error) {
	// This would:
	// 1. Query the OCI registry for available versions
	// 2. Filter to versions that have been previously deployed
	// 3. Return the list
	return nil, fmt.Errorf("not yet implemented")
}
