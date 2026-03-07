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
	"github.com/spf13/cobra"
)

var (
	publishRegistry  string
	publishTag       string
	publishInsecure  bool
	publishPlainHTTP bool
	publishDryRun    bool
)

// publishCmd publishes a DK package to an OCI registry
var publishCmd = &cobra.Command{
	Use:   "publish [package-dir]",
	Short: "Publish a DK package to an OCI registry",
	Long: `Publish a DK data package to an OCI-compliant registry.

The publish command builds (if not already built) and pushes the package
artifact to the specified OCI registry.

Tag immutability is enforced - attempting to publish the same version
twice will fail. Use a new version or use --force for development.

Examples:
  # Publish to default registry
  dk publish

  # Publish to specific registry
  dk publish --registry ghcr.io/myorg

  # Publish with custom tag
  dk publish --tag v1.0.0

  # Dry run (build but don't push)
  dk publish --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPublish,
}

func init() {
	rootCmd.AddCommand(publishCmd)

	publishCmd.Flags().StringVar(&publishRegistry, "registry", "", "OCI registry URL (e.g., ghcr.io/myorg)")
	publishCmd.Flags().StringVarP(&publishTag, "tag", "t", "", "Tag for the artifact (default: version from dk.yaml)")
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

	// Verify dk.yaml exists
	dkPath := filepath.Join(absDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		return fmt.Errorf("dk.yaml not found in %s - is this a valid DK package?", packageDir)
	}

	fmt.Printf("Publishing package: %s\n\n", packageDir)

	// Parse dk.yaml to get package info
	dpData, err := os.ReadFile(dkPath)
	if err != nil {
		return fmt.Errorf("failed to read dk.yaml: %w", err)
	}

	parser := manifest.NewParser()
	m, kind, err := manifest.ParseManifest(dpData)
	_ = parser // parser still used elsewhere; keep import
	_ = kind
	if err != nil {
		return fmt.Errorf("failed to parse dk.yaml: %w", err)
	}

	// Determine version/tag
	version := publishTag
	if version == "" {
		version = m.GetVersion()
	}
	if version == "" {
		return fmt.Errorf("no version specified - use --tag or set metadata.version in dk.yaml")
	}

	// Determine registry
	reg := publishRegistry
	if reg == "" {
		// Check environment
		reg = os.Getenv("DK_REGISTRY")
	}
	if reg == "" {
		// Default to ghcr.io with namespace
		reg = "ghcr.io/" + m.GetNamespace()
	}

	// Build full reference
	reference := registry.Reference(reg, m.GetNamespace(), m.GetName(), version)

	fmt.Printf("Target: %s\n\n", reference)

	// Step 1: Build artifact
	fmt.Println("Step 1/4: Building artifact...")
	bundler := registry.NewBundler(Version)

	gitInfo := getGitInfo(absDir)
	artifact, err := bundler.Bundle(registry.BundleOptions{
		PackageDir: absDir,
		GitCommit:  gitInfo.commit,
		GitBranch:  gitInfo.branch,
		GitTag:     gitInfo.tag,
	})
	if err != nil {
		return fmt.Errorf("failed to bundle artifact: %w", err)
	}

	fmt.Println("✓ Artifact built")

	// Step 2: Generate Helm chart (if not already in dist/)
	fmt.Println("\nStep 2/4: Preparing Helm chart...")
	chartPath := findHelmChart(absDir, m.GetName())
	if chartPath == "" {
		// Generate chart on the fly
		chartResult, chartErr := registry.GenerateHelmChart(registry.HelmChartOptions{
			PackageDir: absDir,
			GitCommit:  gitInfo.commit,
			GitBranch:  gitInfo.branch,
			GitTag:     gitInfo.tag,
			Version:    version,
		})
		if chartErr != nil {
			return fmt.Errorf("failed to generate Helm chart: %w", chartErr)
		}
		chartPath = chartResult.ChartPath
		fmt.Printf("✓ Helm chart generated: %s\n", chartPath)
	} else {
		fmt.Printf("✓ Using existing chart: %s\n", chartPath)
	}

	// Step 3: Check immutability
	fmt.Println("\nStep 3/4: Checking tag availability...")

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
		fmt.Printf("\nWould publish:\n")
		fmt.Printf("  OCI artifact: %s\n", reference)
		fmt.Printf("  Helm chart:   oci://%s/%s/%s\n", reg, m.GetNamespace(), m.GetName())
		return nil
	}

	// Step 4: Push to registry
	fmt.Println("\nStep 4/4: Pushing to registry...")

	// Push OCI artifact
	result, err := client.Push(ctx, reference, artifact)
	if err != nil {
		return fmt.Errorf("failed to push artifact: %w", err)
	}
	fmt.Printf("✓ OCI artifact pushed\n")

	// Push Helm chart via helm push
	helmRef := fmt.Sprintf("oci://%s/%s", reg, m.GetNamespace())
	helmErr := pushHelmChart(chartPath, helmRef, publishPlainHTTP, publishInsecure)
	if helmErr != nil {
		fmt.Printf("⚠ Helm chart push failed: %v\n", helmErr)
		fmt.Println("  (OCI artifact was pushed successfully)")
		fmt.Println("  Install helm and run: helm push", chartPath, helmRef)
	} else {
		fmt.Printf("✓ Helm chart pushed to %s\n", helmRef)
	}

	fmt.Printf("\n✓ Published successfully!\n")
	fmt.Printf("\nArtifact Details:\n")
	fmt.Printf("  Reference: %s\n", result.Reference)
	fmt.Printf("  Digest:    %s\n", result.Digest)
	fmt.Printf("  Size:      %s\n", formatSize(result.Size))
	fmt.Printf("  Chart:     %s\n", chartPath)

	// Print next steps
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  dk promote %s %s --to dev  # Deploy to dev environment\n",
		m.GetName(), version)
	fmt.Printf("  helm install %s oci://%s/%s/%s --version %s --set cell=<cell>  # Deploy to cell\n",
		m.GetName(), reg, m.GetNamespace(), m.GetName(), version)

	return nil
}

// findHelmChart looks for an existing Helm chart in dist/ directory.
func findHelmChart(packageDir, name string) string {
	distDir := filepath.Join(packageDir, "dist")
	entries, err := os.ReadDir(distDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), name) && strings.HasSuffix(e.Name(), ".tgz") {
			return filepath.Join(distDir, e.Name())
		}
	}
	return ""
}

// pushHelmChart pushes a Helm chart to an OCI registry using the helm CLI.
func pushHelmChart(chartPath, ociRef string, plainHTTP, insecure bool) error {
	args := []string{"push", chartPath, ociRef}
	if plainHTTP {
		args = append(args, "--plain-http")
	}
	if insecure {
		args = append(args, "--insecure-skip-tls-verify")
	}

	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
