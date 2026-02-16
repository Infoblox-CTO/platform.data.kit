package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/registry"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/validate"
	"github.com/spf13/cobra"
)

var (
	buildTag     string
	buildPush    bool
	buildDryRun  bool
	buildNoCache bool
)

// buildCmd builds a DP package artifact
var buildCmd = &cobra.Command{
	Use:   "build [package-dir]",
	Short: "Build a DP package artifact",
	Long: `Build a DP data package into an OCI artifact.

The build command validates the package manifests, bundles all files,
and creates an OCI-compliant artifact ready for publishing.

Examples:
  # Build package in current directory
  dp build

  # Build with custom tag
  dp build --tag v1.0.0

  # Build and push to registry
  dp build --push

  # Dry run (validate only)
  dp build --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringVarP(&buildTag, "tag", "t", "", "Tag for the built artifact (default: version from dp.yaml)")
	buildCmd.Flags().BoolVar(&buildPush, "push", false, "Push artifact to registry after building")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "Validate only, don't build artifact")
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Don't use cache when building")
}

func runBuild(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Building package: %s\n\n", packageDir)

	// Step 1: Validate manifests
	fmt.Println("Step 1/3: Validating manifests...")
	ctx := context.Background()
	result := validate.ValidatePackage(ctx, absDir)

	if !result.Valid {
		fmt.Println("✗ Validation failed:")
		for _, e := range result.Errors {
			field := e.Field
			if field == "" {
				field = "(root)"
			}
			fmt.Printf("  [%s] %s: %s\n", e.Code, field, e.Message)
		}
		return fmt.Errorf("validation failed")
	}

	fmt.Println("✓ Validation passed")

	if buildDryRun {
		fmt.Println("\nDry run complete - no artifact built")
		return nil
	}

	// Parse the manifest for metadata
	m, kind, parseErr := manifest.ParseManifestFile(dpPath)
	if parseErr != nil {
		return fmt.Errorf("failed to parse dp.yaml: %w", parseErr)
	}

	// Step 2: Gather git info
	fmt.Println("\nStep 2/3: Gathering build info...")
	gitInfo := getGitInfo(absDir)
	fmt.Printf("  Git commit: %s\n", gitInfo.commit)
	fmt.Printf("  Git branch: %s\n", gitInfo.branch)
	if gitInfo.tag != "" {
		fmt.Printf("  Git tag: %s\n", gitInfo.tag)
	}

	// Step 3: Bundle artifact
	fmt.Println("\nStep 3/3: Creating artifact bundle...")
	bundler := registry.NewBundler(Version)

	artifact, err := bundler.Bundle(registry.BundleOptions{
		PackageDir: absDir,
		GitCommit:  gitInfo.commit,
		GitBranch:  gitInfo.branch,
		GitTag:     gitInfo.tag,
	})
	if err != nil {
		return fmt.Errorf("failed to bundle artifact: %w", err)
	}

	// Calculate artifact size
	totalSize := int64(0)
	for _, layer := range artifact.Layers {
		totalSize += int64(len(layer.Content))
	}

	// Get version from manifest or flag
	version := buildTag
	if version == "" {
		version = m.GetVersion()
	}
	if version == "" {
		version = "latest"
	}

	fmt.Printf("\n✓ Build complete!\n")
	fmt.Printf("\nArtifact Info:\n")
	fmt.Printf("  Name:      %s\n", m.GetName())
	fmt.Printf("  Namespace: %s\n", m.GetNamespace())
	fmt.Printf("  Version:   %s\n", version)
	fmt.Printf("  Kind:      %s\n", kind)
	fmt.Printf("  Layers:    %d\n", len(artifact.Layers))
	fmt.Printf("  Size:      %s\n", formatSize(totalSize))

	if buildPush {
		fmt.Println("\nPushing to registry...")
		// This would call the publish logic
		fmt.Println("(Push not implemented yet - use 'dp publish' after build)")
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  dp publish              # Push to OCI registry\n")
	fmt.Printf("  dp promote %s %s --to dev  # Promote to dev environment\n",
		m.GetName(), version)

	return nil
}

type gitInfo struct {
	commit string
	branch string
	tag    string
}

func getGitInfo(dir string) gitInfo {
	info := gitInfo{
		commit: "unknown",
		branch: "unknown",
	}

	// Get commit SHA
	if out, err := runGitCommand(dir, "rev-parse", "HEAD"); err == nil {
		info.commit = strings.TrimSpace(out)
		if len(info.commit) > 8 {
			info.commit = info.commit[:8]
		}
	}

	// Get branch name
	if out, err := runGitCommand(dir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		info.branch = strings.TrimSpace(out)
	}

	// Get tag if on a tagged commit
	if out, err := runGitCommand(dir, "describe", "--tags", "--exact-match"); err == nil {
		info.tag = strings.TrimSpace(out)
	}

	return info
}

func runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
