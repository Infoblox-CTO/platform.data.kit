// Package cmd contains the CLI commands for DK.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/promotion"
	"github.com/spf13/cobra"
)

var promoteCmd = &cobra.Command{
	Use:   "promote <package> <version>",
	Short: "Promote a package to an environment",
	Long: `Promote a data package to a target environment via GitOps PR workflow.

This command creates a pull request that updates the version of a package
in the target environment's Kustomize overlay. When the PR is merged,
ArgoCD will automatically deploy the new version.

Example:
  # Promote to dev environment
  dk promote kafka-s3-pipeline v1.0.0 --to dev

  # Promote to integration with digest verification
  dk promote kafka-s3-pipeline v1.0.0 --to int --digest sha256:abc123

  # Dry run to see what would happen
  dk promote kafka-s3-pipeline v1.0.0 --to prod --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runPromote,
}

var (
	promoteToEnv     string
	promoteDigest    string
	promoteRegistry  string
	promoteDryRun    bool
	promoteAutoMerge bool
)

func init() {
	rootCmd.AddCommand(promoteCmd)

	promoteCmd.Flags().StringVar(&promoteToEnv, "to", "", "Target environment (dev, int, prod)")
	promoteCmd.Flags().StringVar(&promoteDigest, "digest", "", "Content digest for verification")
	promoteCmd.Flags().StringVar(&promoteRegistry, "registry", "", "OCI registry URL (default: ghcr.io/infoblox-cto)")
	promoteCmd.Flags().BoolVar(&promoteDryRun, "dry-run", false, "Simulate the promotion without creating a PR")
	promoteCmd.Flags().BoolVar(&promoteAutoMerge, "auto-merge", false, "Enable auto-merge on the PR")

	promoteCmd.MarkFlagRequired("to")
}

func runPromote(cmd *cobra.Command, args []string) error {
	packageName := args[0]
	version := args[1]

	// Validate environment
	targetEnv := promotion.Environment(promoteToEnv)
	if !targetEnv.Valid() {
		return fmt.Errorf("invalid environment: %s (must be dev, int, or prod)", promoteToEnv)
	}

	// Get GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" && !promoteDryRun {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required")
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create promotion request
	req := &promotion.PromotionRequest{
		Package:   packageName,
		Version:   version,
		Digest:    promoteDigest,
		Registry:  promoteRegistry,
		TargetEnv: targetEnv,
		DryRun:    promoteDryRun,
		AutoMerge: promoteAutoMerge,
	}

	fmt.Printf("Promoting %s to %s...\n", packageName, targetEnv)
	fmt.Printf("  Version: %s\n", version)
	if promoteDigest != "" {
		fmt.Printf("  Digest:  %s\n", promoteDigest)
	}

	if promoteDryRun {
		fmt.Println("\n[DRY RUN] Would create promotion PR with:")
		fmt.Printf("  Branch: promote/%s/%s/%s/...\n", packageName, targetEnv, version)
		fmt.Printf("  Title:  Promote %s to %s: %s\n", packageName, targetEnv, version)
		fmt.Println("\nNo changes made.")
		return nil
	}

	// Create GitHub client
	ghClient := promotion.NewGitHubClient(token, owner, repo)

	// Create promoter (Kustomize updater not needed for API-only promotion)
	promoter := &promotion.Promoter{
		GitHubClient: ghClient,
		BaseBranch:   "main",
		RecordGenerator: promotion.NewRecordGenerator().WithInitiator(
			os.Getenv("USER"),
		),
	}

	// Execute promotion
	result, err := promoter.Promote(ctx, req)
	if err != nil {
		return fmt.Errorf("promotion failed: %w", err)
	}

	if result.Success {
		fmt.Println("\n✓ Promotion PR created successfully!")
		fmt.Printf("  PR #%d: %s\n", result.PRNumber, result.PRURL)
		fmt.Printf("  Branch: %s\n", result.Branch)
		if promoteAutoMerge {
			fmt.Println("  Auto-merge: enabled")
		}
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Review the PR")
		fmt.Println("  2. Approve and merge")
		fmt.Println("  3. ArgoCD will deploy automatically")
	}

	return nil
}

// promoteStatusCmd shows the status of a promotion PR.
var promoteStatusCmd = &cobra.Command{
	Use:   "status <pr-number>",
	Short: "Check the status of a promotion PR",
	Args:  cobra.ExactArgs(1),
	RunE:  runPromoteStatus,
}

func init() {
	promoteCmd.AddCommand(promoteStatusCmd)
}

func runPromoteStatus(cmd *cobra.Command, args []string) error {
	var prNumber int
	if _, err := fmt.Sscanf(args[0], "%d", &prNumber); err != nil {
		return fmt.Errorf("invalid PR number: %s", args[0])
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		owner = "Infoblox-CTO"
	}
	repo := os.Getenv("GITHUB_REPO")
	if repo == "" {
		repo = "data-platform"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ghClient := promotion.NewGitHubClient(token, owner, repo)
	promoter := &promotion.Promoter{
		GitHubClient: ghClient,
	}

	status, err := promoter.GetStatus(ctx, prNumber)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Printf("PR #%d Status\n", status.PRNumber)
	fmt.Printf("  State:  %s\n", status.State)
	fmt.Printf("  Merged: %v\n", status.Merged)
	if status.MergedAt != nil {
		fmt.Printf("  Merged At: %s\n", status.MergedAt.Format(time.RFC3339))
	}

	return nil
}
