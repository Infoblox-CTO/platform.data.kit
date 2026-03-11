package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
)

var (
	statusAll         bool
	statusCell        string
	statusKubeContext string
	statusScanDirs    []string
)

var statusCmd = &cobra.Command{
	Use:   "status [package]",
	Short: "Show package status across environments",
	Long: `Display the deployment status of data packages.

When --cell is specified, queries the Kubernetes cluster for Store resources
in the cell's namespace (dk-<cell>). Without --cell, shows a summary of
the local project's transforms and datasets.

Examples:
  # Show local project summary
  dk status --scan-dir .

  # Show stores in a specific cell
  dk status --cell dev

  # Show stores in prod cell with specific kube context
  dk status --cell prod --kube-context prod-cluster

  # Show status for a specific package
  dk status my-pipeline`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Show status for all packages")
	statusCmd.Flags().StringVar(&statusCell, "cell", "", "Cell name to query for deployment status")
	statusCmd.Flags().StringVar(&statusKubeContext, "kube-context", "", "Kubernetes context to use")
	statusCmd.Flags().StringArrayVar(&statusScanDirs, "scan-dir", nil, "Directories to scan for manifests")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Cell mode — query Kubernetes for store status
	if statusCell != "" {
		return runCellStatus(cmd, statusCell)
	}

	// Single-package mode (with or without scan-dir)
	if len(args) > 0 {
		return showPackageStatus(cmd, args[0])
	}

	// Scan mode — show local project summary
	return runProjectStatus(cmd)
}

// runCellStatus queries a Kubernetes cell for deployment information.
func runCellStatus(cmd *cobra.Command, cellName string) error {
	ctx := context.Background()
	resolver := runner.NewCellResolver(cellName, statusKubeContext, cmd.OutOrStdout())

	// Check if cell exists
	exists, err := resolver.CellExists(ctx)
	if err != nil {
		cmd.Printf("Cell: %s (unable to verify — kubectl may not be configured)\n\n", cellName)
		cmd.Println("To use cell-based status, ensure:")
		cmd.Println("  1. kubectl is installed and configured")
		cmd.Println("  2. DataKit CRDs are installed in the cluster")
		cmd.Printf("  3. Cell %q exists in the cluster\n", cellName)
		return nil
	}
	if !exists {
		return fmt.Errorf("cell %q not found in cluster", cellName)
	}

	cmd.Printf("Cell: %s (namespace: dk-%s)\n\n", cellName, cellName)

	// List stores in the cell
	stores, err := resolver.ListStores(ctx)
	if err != nil {
		cmd.Printf("Warning: failed to list stores: %v\n", err)
		return nil
	}

	if len(stores) == 0 {
		cmd.Println("No stores found in this cell.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STORE\tCONNECTOR\tVERSION\tNAMESPACE")
	fmt.Fprintln(w, "-----\t---------\t-------\t---------")

	for _, store := range stores {
		version := store.Spec.ConnectorVersion
		if version == "" {
			version = "latest"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			store.Metadata.Name,
			store.Spec.Connector,
			version,
			store.Metadata.Namespace,
		)
	}
	w.Flush()

	return nil
}

// runProjectStatus shows a summary of the local project.
func runProjectStatus(cmd *cobra.Command) error {
	scanDirs := statusScanDirs
	if len(scanDirs) == 0 {
		absDir, err := filepath.Abs(".")
		if err != nil {
			return fmt.Errorf("failed to resolve current directory: %w", err)
		}
		scanDirs = []string{absDir}
	}

	// Build pipeline graph to discover transforms and datasets
	g, err := pipeline.BuildGraph(pipeline.GraphOptions{
		ScanDirs: scanDirs,
		ShowAll:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to scan project: %w", err)
	}

	if len(g.Nodes) == 0 {
		cmd.Println("No transforms or datasets found.")
		cmd.Println("Use --scan-dir to specify the project directory.")
		return nil
	}

	// Count transforms and datasets
	var transforms, datasets []pipeline.GraphNode
	for _, n := range g.Nodes {
		if n.Type == "transform" {
			transforms = append(transforms, n)
		} else {
			datasets = append(datasets, n)
		}
	}

	cmd.Printf("Project Status\n")
	cmd.Printf("══════════════\n\n")
	cmd.Printf("Transforms: %d\n", len(transforms))
	cmd.Printf("DataSets:   %d\n", len(datasets))
	cmd.Printf("Edges:      %d\n\n", len(g.Edges))

	if len(transforms) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TRANSFORM\tRUNTIME\tTRIGGER\tINPUTS\tOUTPUTS")
		fmt.Fprintln(w, "---------\t-------\t-------\t------\t-------")

		for _, t := range transforms {
			// Count inputs and outputs from edges
			inputs := 0
			outputs := 0
			for _, e := range g.Edges {
				if e.To == t.ID {
					inputs++
				}
				if e.From == t.ID {
					outputs++
				}
			}

			trigger := t.TriggerPolicy
			if t.TriggerDetail != "" {
				trigger = fmt.Sprintf("%s (%s)", trigger, t.TriggerDetail)
			}
			if trigger == "" {
				trigger = "manual"
			}

			runtime := t.Runtime
			if runtime == "" {
				runtime = "unknown"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n",
				t.ID, runtime, trigger, inputs, outputs)
		}
		w.Flush()
	}

	return nil
}

// showPackageStatus shows status for a single package (legacy mode).
func showPackageStatus(cmd *cobra.Command, packageName string) error {
	// Try to find the package in the current directory
	scanDirs := statusScanDirs
	if len(scanDirs) == 0 {
		absDir, err := filepath.Abs(".")
		if err != nil {
			return fmt.Errorf("failed to resolve directory: %w", err)
		}
		scanDirs = []string{absDir}
	}

	g, err := pipeline.BuildGraph(pipeline.GraphOptions{
		ScanDirs: scanDirs,
		ShowAll:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to scan for package: %w", err)
	}

	// Find the transform node
	var found *pipeline.GraphNode
	for i, n := range g.Nodes {
		if n.ID == packageName && n.Type == "transform" {
			found = &g.Nodes[i]
			break
		}
	}

	if found == nil {
		return fmt.Errorf("transform %q not found (scanned %d directories)", packageName, len(scanDirs))
	}

	cmd.Printf("Package: %s\n", found.ID)
	cmd.Printf("Runtime: %s\n", found.Runtime)
	if found.TriggerPolicy != "" {
		cmd.Printf("Trigger: %s", found.TriggerPolicy)
		if found.TriggerDetail != "" {
			cmd.Printf(" (%s)", found.TriggerDetail)
		}
		cmd.Println()
	}
	cmd.Println()

	// Show inputs and outputs
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DIRECTION\tDATASET")
	fmt.Fprintln(w, "---------\t-------")
	for _, e := range g.Edges {
		if e.To == found.ID {
			fmt.Fprintf(w, "input\t%s\n", e.From)
		}
	}
	for _, e := range g.Edges {
		if e.From == found.ID {
			fmt.Fprintf(w, "output\t%s\n", e.To)
		}
	}
	w.Flush()

	return nil
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}

// formatRelativeTime formats a time relative to now.
func formatRelativeTime(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}

// healthIcon returns an icon for health status.
func healthIcon(status string) string {
	switch strings.ToLower(status) {
	case "healthy":
		return "✓"
	case "degraded":
		return "⚠"
	case "unhealthy":
		return "✗"
	default:
		return "?"
	}
}
