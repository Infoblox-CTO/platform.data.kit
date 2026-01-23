package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Infoblox-CTO/data.platform.kit/sdk/localdev"
	"github.com/spf13/cobra"
)

var (
	devComposePath   string
	devRemoveVolumes bool
)

// devCmd represents the dev command group
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Manage local development environment",
	Long: `Manage the local development environment for DP.

The dev command provides subcommands to start, stop, and monitor the
local development stack which includes:

  - Redpanda (Kafka-compatible message broker)
  - LocalStack (AWS S3 emulation)
  - PostgreSQL (relational database)

Examples:
  # Start the local development stack
  dp dev up

  # Check status of running services
  dp dev status

  # Stop the stack and remove volumes
  dp dev down --volumes`,
}

// devUpCmd starts the local development stack
var devUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the local development stack",
	Long: `Start the local development stack using Docker Compose.

This command starts all required services for local development:
  - Redpanda: Kafka-compatible streaming at localhost:19092
  - LocalStack: S3 at localhost:4566
  - PostgreSQL: Database at localhost:5432

The command waits for all services to become healthy before returning.`,
	RunE: runDevUp,
}

// devDownCmd stops the local development stack
var devDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the local development stack",
	Long: `Stop and remove the local development stack.

By default, data volumes are preserved. Use --volumes to remove them.`,
	RunE: runDevDown,
}

// devStatusCmd shows the status of the local development stack
var devStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of local development services",
	Long:  `Display the current status of all local development services.`,
	RunE:  runDevStatus,
}

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.AddCommand(devUpCmd)
	devCmd.AddCommand(devDownCmd)
	devCmd.AddCommand(devStatusCmd)

	// Flags for dev commands
	devCmd.PersistentFlags().StringVar(&devComposePath, "compose", "", "Path to docker-compose.yaml (auto-detected if not specified)")
	devDownCmd.Flags().BoolVar(&devRemoveVolumes, "volumes", false, "Remove data volumes when stopping")
}

// findComposeFile searches for the docker-compose file in standard locations
func findComposeFile() (string, error) {
	// First check if we're in a DP workspace
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search paths relative to current directory
	searchPaths := []string{
		filepath.Join(cwd, "hack", "compose", "docker-compose.yaml"),
		filepath.Join(cwd, "hack", "compose", "docker-compose.yml"),
		filepath.Join(cwd, "docker-compose.yaml"),
		filepath.Join(cwd, "docker-compose.yml"),
	}

	// Also search in parent directories (up to 3 levels)
	for i := 0; i < 3; i++ {
		parent := cwd
		for j := 0; j <= i; j++ {
			parent = filepath.Dir(parent)
		}
		searchPaths = append(searchPaths,
			filepath.Join(parent, "hack", "compose", "docker-compose.yaml"),
		)
	}

	for _, p := range searchPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("docker-compose.yaml not found; specify path with --compose or run from DP workspace")
}

func runDevUp(cmd *cobra.Command, args []string) error {
	composePath := devComposePath
	var err error

	if composePath == "" {
		composePath, err = findComposeFile()
		if err != nil {
			return err
		}
	}

	fmt.Printf("Starting local development stack...\n")
	fmt.Printf("Using compose file: %s\n\n", composePath)

	manager, err := localdev.NewComposeManager(composePath)
	if err != nil {
		return fmt.Errorf("failed to initialize compose manager: %w", err)
	}

	ctx := context.Background()

	// Start the stack
	if err := manager.Up(ctx, true, os.Stdout); err != nil {
		return fmt.Errorf("failed to start stack: %w", err)
	}

	fmt.Println("\nWaiting for services to become healthy...")

	// Wait for healthy with timeout
	if err := manager.WaitForHealthy(ctx, 2*time.Minute); err != nil {
		fmt.Println("Warning: Some services may not be fully healthy yet")
		fmt.Println("Use 'dp dev status' to check service status")
	}

	// Show status
	status, err := manager.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	fmt.Println("\n✓ Local development stack is running!")
	fmt.Println("\nServices:")

	formatter := GetFormatter()
	data := formatServiceStatus(status)

	if err := formatter.Format(os.Stdout, data); err != nil {
		return err
	}

	fmt.Println("\nEndpoints:")
	fmt.Println("  Kafka:           localhost:19092")
	fmt.Println("  Schema Registry: localhost:18081")
	fmt.Println("  Redpanda Console: http://localhost:8080")
	fmt.Println("  S3 (LocalStack): localhost:4566")
	fmt.Println("  PostgreSQL:      localhost:5432")

	return nil
}

func runDevDown(cmd *cobra.Command, args []string) error {
	composePath := devComposePath
	var err error

	if composePath == "" {
		composePath, err = findComposeFile()
		if err != nil {
			return err
		}
	}

	fmt.Printf("Stopping local development stack...\n")

	manager, err := localdev.NewComposeManager(composePath)
	if err != nil {
		return fmt.Errorf("failed to initialize compose manager: %w", err)
	}

	ctx := context.Background()

	if err := manager.Down(ctx, devRemoveVolumes, os.Stdout); err != nil {
		return fmt.Errorf("failed to stop stack: %w", err)
	}

	fmt.Println("\n✓ Local development stack stopped")
	if devRemoveVolumes {
		fmt.Println("  Data volumes removed")
	} else {
		fmt.Println("  Data volumes preserved (use --volumes to remove)")
	}

	return nil
}

func runDevStatus(cmd *cobra.Command, args []string) error {
	composePath := devComposePath
	var err error

	if composePath == "" {
		composePath, err = findComposeFile()
		if err != nil {
			// Not an error for status - just means stack isn't set up
			fmt.Println("Local development stack is not configured")
			fmt.Println("Run 'dp dev up' to start the stack")
			return nil
		}
	}

	manager, err := localdev.NewComposeManager(composePath)
	if err != nil {
		return fmt.Errorf("failed to initialize compose manager: %w", err)
	}

	ctx := context.Background()
	status, err := manager.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if !status.Running || len(status.Services) == 0 {
		fmt.Println("Local development stack is not running")
		fmt.Println("Run 'dp dev up' to start the stack")
		return nil
	}

	fmt.Println("Local development stack status:")

	formatter := GetFormatter()
	data := formatServiceStatus(status)

	return formatter.Format(os.Stdout, data)
}

// formatServiceStatus converts status to a format suitable for output
func formatServiceStatus(status *localdev.StackStatus) interface{} {
	type serviceInfo struct {
		Name   string `json:"name" yaml:"name"`
		Status string `json:"status" yaml:"status"`
		Health string `json:"health" yaml:"health"`
		Ports  string `json:"ports,omitempty" yaml:"ports,omitempty"`
	}

	services := make([]serviceInfo, 0, len(status.Services))
	for _, svc := range status.Services {
		ports := ""
		if len(svc.Ports) > 0 {
			for i, p := range svc.Ports {
				if i > 0 {
					ports += ", "
				}
				ports += p
			}
		}

		healthIcon := getHealthIcon(svc.Health)

		services = append(services, serviceInfo{
			Name:   svc.Name,
			Status: svc.Status,
			Health: healthIcon + " " + svc.Health,
			Ports:  ports,
		})
	}

	return services
}

func getHealthIcon(health string) string {
	switch health {
	case "healthy", "running":
		return "✓"
	case "starting":
		return "◌"
	case "unhealthy":
		return "✗"
	case "exited":
		return "○"
	default:
		return "?"
	}
}
