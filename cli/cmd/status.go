// Package cmd contains the CLI commands for DK.
package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [package]",
	Short: "Show package status across environments",
	Long: `Display the status of a data package across all environments.

Shows deployment versions, last run status, and health indicators
for dev, int, and prod environments.

Example:
  # Show status for a specific package
  dk status kafka-s3-pipeline

  # Show status for all packages
  dk status --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runStatus,
}

var statusAll bool

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVar(&statusAll, "all", false, "Show status for all packages")
}

func runStatus(cmd *cobra.Command, args []string) error {
	if statusAll {
		return showAllPackagesStatus()
	}

	if len(args) == 0 {
		return fmt.Errorf("package name required (or use --all)")
	}

	packageName := args[0]
	return showPackageStatus(packageName)
}

func showPackageStatus(packageName string) error {
	fmt.Printf("Package: %s\n\n", packageName)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENVIRONMENT\tVERSION\tLAST RUN\tSTATUS\tHEALTH")
	fmt.Fprintln(w, "-----------\t-------\t--------\t------\t------")

	// For MVP, show placeholder data
	// In production, this would query the run service
	environments := []struct {
		name    string
		version string
		lastRun string
		status  string
		health  string
	}{
		{"dev", "v1.2.0", "2 hours ago", "success", "✓ healthy"},
		{"int", "v1.1.0", "1 day ago", "success", "✓ healthy"},
		{"prod", "v1.0.0", "3 days ago", "success", "✓ healthy"},
	}

	for _, env := range environments {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			env.name, env.version, env.lastRun, env.status, env.health)
	}

	w.Flush()

	fmt.Println("\nRecent Runs (dev):")
	fmt.Println("  • 20240115-120000 - success (45s, 10,234 records)")
	fmt.Println("  • 20240115-110000 - success (42s, 9,876 records)")
	fmt.Println("  • 20240115-100000 - success (48s, 11,012 records)")

	return nil
}

func showAllPackagesStatus() error {
	fmt.Println("All Packages Status")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PACKAGE\tDEV\tINT\tPROD\tHEALTH")
	fmt.Fprintln(w, "-------\t---\t---\t----\t------")

	// Placeholder data
	packages := []struct {
		name   string
		dev    string
		int_   string
		prod   string
		health string
	}{
		{"kafka-s3-pipeline", "v1.2.0", "v1.1.0", "v1.0.0", "✓ healthy"},
		{"clickstream-etl", "v2.0.0", "v1.9.0", "v1.9.0", "✓ healthy"},
		{"user-events", "v3.1.0", "v3.0.0", "v2.9.0", "⚠ degraded"},
	}

	for _, pkg := range packages {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			pkg.name, pkg.dev, pkg.int_, pkg.prod, pkg.health)
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
