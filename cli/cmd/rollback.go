// Package cmd contains the CLI commands for DK.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/promotion"
	"github.com/infobloxopen/apx/pkg/githubauth"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <package>",
	Short: "Rollback a package to a previous version",
	Long: `Rollback a data package to a previous version in an environment.

Rollback is implemented as a promotion of the previous version,
creating a PR to update the cell's values.yaml.

Example:
  # Rollback to a specific version in dev (default cell c0)
  dk rollback kafka-s3-pipeline --to dev --to-version v1.0.0

  # Rollback a specific cell within prod
  dk rollback kafka-s3-pipeline --to prod --cell canary --to-version v1.0.0

  # Dry run to see what would happen
  dk rollback kafka-s3-pipeline --to prod --to-version v1.0.0 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runRollback,
}

var (
	rollbackCell      string
	rollbackToEnv     string
	rollbackToVersion string
	rollbackDryRun    bool
)

func init() {
	rootCmd.AddCommand(rollbackCmd)

	rollbackCmd.Flags().StringVar(&rollbackToEnv, "to", "", "Target environment (dev, int, prod)")
	rollbackCmd.Flags().StringVar(&rollbackCell, "cell", "", "Target cell within the environment (default: c0)")
	rollbackCmd.Flags().StringVar(&rollbackToVersion, "to-version", "", "Specific version to rollback to")
	rollbackCmd.Flags().BoolVar(&rollbackDryRun, "dry-run", false, "Simulate the rollback without creating a PR")

	rollbackCmd.MarkFlagRequired("to")
}

func runRollback(cmd *cobra.Command, args []string) error {
	packageName := args[0]

	// Validate environment
	targetEnv := promotion.Environment(rollbackToEnv)
	if !targetEnv.Valid() {
		return fmt.Errorf("invalid environment: %s (must be dev, int, or prod)", rollbackToEnv)
	}

	cell := promotion.ResolveCell(rollbackCell)

	// Get GitHub token: env var first, then device-flow via githubauth
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" && !rollbackDryRun {
		org, orgErr := githubauth.DetectOrg()
		if orgErr != nil {
			return fmt.Errorf("GITHUB_TOKEN not set and cannot detect org from git remote: %w", orgErr)
		}
		t, tokenErr := githubauth.EnsureToken(org)
		if tokenErr != nil {
			return fmt.Errorf("GitHub authentication failed: %w", tokenErr)
		}
		token = t
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Determine the version to rollback to
	rollbackVersion := rollbackToVersion
	if rollbackVersion == "" {
		return fmt.Errorf("--to-version is required (automatic previous version detection coming soon)")
	}

	fmt.Printf("Rolling back %s in %s/%s to %s...\n", packageName, targetEnv, cell, rollbackVersion)

	if rollbackDryRun {
		fmt.Println("\n[DRY RUN] Would create rollback PR with:")
		fmt.Printf("  Package:     %s\n", packageName)
		fmt.Printf("  Environment: %s\n", targetEnv)
		fmt.Printf("  Cell:        %s\n", cell)
		fmt.Printf("  Version:     %s\n", rollbackVersion)
		fmt.Printf("  File:        envs/%s/cells/%s/apps/%s/values.yaml\n", targetEnv, cell, packageName)
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
		repo = "datakit"
	}

	// Create GitHub client
	ghClient := promotion.NewGitHubClient(token, owner, repo)

	// Support custom GitHub base URL
	if baseURL := os.Getenv("GITHUB_BASE_URL"); baseURL != "" {
		ghClient.BaseURL = baseURL
	}

	// Create promotion request (rollback is just a promotion to an older version)
	req := &promotion.PromotionRequest{
		Package:   packageName,
		Version:   rollbackVersion,
		TargetEnv: targetEnv,
		Cell:      rollbackCell,
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
