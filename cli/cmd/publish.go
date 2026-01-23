package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/data.platform.kit/sdk/manifest"
	"github.com/Infoblox-CTO/data.platform.kit/sdk/registry"
	"github.com/spf13/cobra"
)

var (
	publishRegistry  string
	publishTag       string
	publishInsecure  bool
	publishPlainHTTP bool
	publishDryRun    bool
)

// publishCmd publishes a DP package to an OCI registry
var publishCmd = &cobra.Command{
	Use:   "publish [package-dir]",
	Short: "Publish a DP package to an OCI registry",
	Long: `Publish a DP data package to an OCI-compliant registry.

The publish command builds (if not already built) and pushes the package
artifact to the specified OCI registry.

Tag immutability is enforced - attempting to publish the same version
twice will fail. Use a new version or use --force for development.

Examples:
  # Publish to default registry
  dp publish

  # Publish to specific registry
  dp publish --registry ghcr.io/myorg

  # Publish with custom tag
  dp publish --tag v1.0.0

  # Dry run (build but don't push)
  dp publish --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPublish,
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVar(&publishRegistry, "registry", "", "OCI registry URL (e.g., ghcr.io/myorg)")
	publishCmd.Flags().StringVarP(&publishTag, "tag", "t", "", "Tag for the artifact (default: version from dp.yaml)")
	publishCmd.Flags().BoolVar(&publishInsecure, "insecure", false, "Allow insecure registry connections")
	publishCmd.Flags().BoolVar(&publishPlainHTTP, "plain-http", false, "Use plain HTTP instead of HTTPS")
	publishCmd.Flags().BoolVar(&publishDryRun, "dry-run", false, "Build artifact but don't push")
}

func runPublish(cmd *cobra.Command, args []string) error {
	// Determine package directory
	packageDir := "."
	if len(args) > 0 {
		packageDir = args[0]
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify dp.yaml exists
	dpPath := filepath.Join(absDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		return fmt.Errorf("dp.yaml not found in %s - is this a valid DP package?", packageDir)
	}

	fmt.Printf("Publishing package: %s\n\n", packageDir)

	// Parse dp.yaml to get package info
	dpData, err := os.ReadFile(dpPath)
	if err != nil {
		return fmt.Errorf("failed to read dp.yaml: %w", err)
	}

	parser := manifest.NewParser()
	pkg, err := parser.ParseDataPackage(dpData)
	if err != nil {
		return fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	// Determine version/tag
	version := publishTag
	if version == "" {
		version = pkg.Metadata.Version
	}
	if version == "" {
		return fmt.Errorf("no version specified - use --tag or set metadata.version in dp.yaml")
	}

	// Determine registry
	reg := publishRegistry
	if reg == "" {
		// Check environment
		reg = os.Getenv("DP_REGISTRY")
	}
	if reg == "" {
		// Default to ghcr.io with namespace
		reg = "ghcr.io/" + pkg.Metadata.Namespace
	}

	// Build full reference
	reference := registry.Reference(reg, pkg.Metadata.Namespace, pkg.Metadata.Name, version)

	fmt.Printf("Target: %s\n\n", reference)

	// Step 1: Build artifact
	fmt.Println("Step 1/3: Building artifact...")
	bundler := registry.NewBundler(Version)

	artifact, err := bundler.Bundle(registry.BundleOptions{
		PackageDir: absDir,
		GitCommit:  getGitInfo(absDir).commit,
		GitBranch:  getGitInfo(absDir).branch,
		GitTag:     getGitInfo(absDir).tag,
	})
	if err != nil {
		return fmt.Errorf("failed to bundle artifact: %w", err)
	}

	fmt.Println("✓ Artifact built")

	// Step 2: Check immutability
	fmt.Println("\nStep 2/3: Checking tag availability...")

	client, err := registry.NewOrasClient(registry.ClientConfig{
		Registry:  reg,
		PlainHTTP: publishPlainHTTP,
		Insecure:  publishInsecure,
	})
	if err != nil {
		return fmt.Errorf("failed to create registry client: %w", err)
	}

	ctx := context.Background()
	exists, err := client.Exists(ctx, reference)
	if err != nil {
		fmt.Printf("⚠ Could not check tag availability: %v\n", err)
		// Continue anyway - the push will fail if it exists
	} else if exists {
		digest, _ := client.Resolve(ctx, reference)
		return &registry.ImmutabilityError{
			Reference:      reference,
			ExistingDigest: digest,
			Message:        "tag already exists and cannot be overwritten",
		}
	}

	fmt.Println("✓ Tag is available")

	if publishDryRun {
		fmt.Println("\nDry run complete - artifact not pushed")
		fmt.Printf("\nWould publish: %s\n", reference)
		return nil
	}

	// Step 3: Push to registry
	fmt.Println("\nStep 3/3: Pushing to registry...")

	result, err := client.Push(ctx, reference, artifact)
	if err != nil {
		return fmt.Errorf("failed to push artifact: %w", err)
	}

	fmt.Printf("\n✓ Published successfully!\n")
	fmt.Printf("\nArtifact Details:\n")
	fmt.Printf("  Reference: %s\n", result.Reference)
	fmt.Printf("  Digest:    %s\n", result.Digest)
	fmt.Printf("  Size:      %s\n", formatSize(result.Size))

	// Print next steps
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  dp promote %s %s --to dev  # Deploy to dev environment\n",
		pkg.Metadata.Name, version)
	fmt.Printf("  dp pull %s                       # Pull the artifact\n", reference)

	return nil
}
