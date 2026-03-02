package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/charts"
	"github.com/spf13/cobra"
)

var (
	devRemoveVolumes bool
	devRuntime       string
)

// devCmd represents the dev command group
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Manage local development environment",
	Long: `Manage the local development environment for DK.

The dev command provides subcommands to start, stop, and monitor the
local development stack which includes:

  - Redpanda (Kafka-compatible message broker)
  - LocalStack (AWS S3 emulation)
  - PostgreSQL (relational database)

Examples:
  # Start the local development stack
  dk dev up

  # Check status of running services
  dk dev status

  # Stop the stack and remove volumes
  dk dev down --volumes`,
}

// devUpCmd starts the local development stack
var devUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the local development stack",
	Long: `Start the local development stack by deploying Helm charts to a k3d cluster.

This command deploys all required dev dependency charts:
  - Redpanda: Kafka-compatible streaming
  - LocalStack: AWS-compatible S3
  - PostgreSQL: Relational database
  - Marquez: Data lineage tracking

Each chart includes init jobs that automatically create topics, buckets,
database schemas, and lineage namespaces. The command waits for all
services to become healthy before returning.

Chart versions and Helm values can be overridden via dk config:
  dk config set dev.charts.redpanda.version 25.2.0
  dk config set dev.charts.postgres.values.primary.resources.limits.memory 1Gi`,
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
	devCmd.PersistentFlags().StringVar(&devRuntime, "runtime", "k3d", "Runtime to use (default: k3d)")
	devDownCmd.Flags().BoolVar(&devRemoveVolumes, "volumes", false, "Remove data volumes when stopping")
}

// getWorkspacePath returns the DK workspace path from environment or config.
func getWorkspacePath() string {
	// Check DK_WORKSPACE_PATH environment variable first
	if envPath := os.Getenv("DK_WORKSPACE_PATH"); envPath != "" {
		return envPath
	}

	// Check config file
	config, err := localdev.LoadConfig()
	if err == nil && config != nil && config.Dev.Workspace != "" {
		return config.Dev.Workspace
	}

	return ""
}

// getRuntime returns the selected runtime type based on the --runtime flag or config.
func getRuntime() (localdev.RuntimeType, error) {
	runtime := strings.ToLower(devRuntime)

	// If flag is empty, check config for default
	if runtime == "" {
		config, err := localdev.LoadConfig()
		if err == nil && config != nil {
			return config.GetDefaultRuntime(), nil
		}
		// No config, default to k3d
		return localdev.RuntimeK3d, nil
	}

	// Handle explicit runtime selection
	switch runtime {
	case "k3d", "kubernetes", "k8s":
		return localdev.RuntimeK3d, nil
	default:
		return "", fmt.Errorf("unsupported runtime %q; use 'k3d'", devRuntime)
	}
}

// getRuntimeManager creates the appropriate runtime manager based on the selected runtime.
func getRuntimeManager(runtime localdev.RuntimeType) (localdev.RuntimeManager, error) {
	switch runtime {
	case localdev.RuntimeK3d:
		k3dManager, err := localdev.NewK3dManager("dk-local")
		if err != nil {
			return nil, err
		}
		return k3dManager, nil

	default:
		return nil, fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

func runDevUp(cmd *cobra.Command, args []string) error {
	runtime, err := getRuntime()
	if err != nil {
		return err
	}

	// Check prerequisites
	checker := localdev.NewPrerequisiteChecker(runtime)
	ctx := context.Background()
	if err := checker.CheckAll(ctx); err != nil {
		return fmt.Errorf("prerequisites check failed: %w", err)
	}

	// Start registry cache for k3d runtime (before getting runtime manager)
	var cacheManager *localdev.CacheManager
	var registriesPath string
	if runtime == localdev.RuntimeK3d {
		var err error
		cacheManager, err = localdev.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to initialize cache manager: %w", err)
		}

		if err := cacheManager.Up(ctx, os.Stdout); err != nil {
			return fmt.Errorf("failed to start registry cache: %w", err)
		}

		registriesPath = cacheManager.GetRegistriesYAMLPath()
	}

	manager, err := getRuntimeManager(runtime)
	if err != nil {
		return fmt.Errorf("failed to initialize runtime manager: %w", err)
	}

	// Set registries path for k3d manager
	if runtime == localdev.RuntimeK3d && registriesPath != "" {
		if k3dMgr, ok := manager.(*localdev.K3dManager); ok {
			k3dMgr.SetRegistriesPath(registriesPath)
		}
	}

	// Check if already running - if so, just report status
	status, err := manager.Status(ctx)
	if err == nil && status.Running {
		fmt.Printf("Local development stack is already running (runtime: %s)\n\n", runtime)
		fmt.Println("Services:")
		formatter := GetFormatter()
		data := formatServiceStatus(status)
		if err := formatter.Format(os.Stdout, data); err != nil {
			return err
		}
		fmt.Println("\nUse 'dk dev down' to stop the stack")
		return nil
	}

	// Check port availability only if not already running
	portChecker := localdev.NewPortChecker(1 * time.Second)
	if err := portChecker.CheckAllAvailable(charts.AllLocalPorts(charts.DefaultCharts)); err != nil {
		return fmt.Errorf("port availability check failed: %w", err)
	}

	fmt.Printf("Starting local development stack (runtime: %s)...\n\n", runtime)

	// Start the stack
	if err := manager.Up(ctx, true, os.Stdout); err != nil {
		return fmt.Errorf("failed to start stack: %w", err)
	}

	fmt.Println("\nWaiting for services to become healthy...")

	// Wait for healthy with timeout
	if err := manager.WaitForHealthy(ctx, 2*time.Minute); err != nil {
		fmt.Println("Warning: Some services may not be fully healthy yet")
		fmt.Println("Use 'dk dev status' to check service status")
	}

	// Show status
	status, err = manager.Status(ctx)
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
	for _, def := range charts.DefaultCharts {
		for _, ep := range def.DisplayEndpoints {
			fmt.Printf("  %-18s %s\n", ep.Label+":", ep.URL)
		}
	}

	return nil
}

func runDevDown(cmd *cobra.Command, args []string) error {
	runtime, err := getRuntime()
	if err != nil {
		return err
	}

	manager, err := getRuntimeManager(runtime)
	if err != nil {
		return fmt.Errorf("failed to initialize runtime manager: %w", err)
	}

	fmt.Printf("Stopping local development stack (runtime: %s)...\n", runtime)

	ctx := context.Background()
	if err := manager.Down(ctx, devRemoveVolumes, os.Stdout); err != nil {
		return fmt.Errorf("failed to stop stack: %w", err)
	}

	// Stop registry cache for k3d runtime (after k3d is stopped)
	if runtime == localdev.RuntimeK3d {
		cacheManager, err := localdev.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to initialize cache manager: %w", err)
		}

		if err := cacheManager.Down(ctx, devRemoveVolumes, os.Stdout); err != nil {
			return fmt.Errorf("failed to stop registry cache: %w", err)
		}
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
	runtime, err := getRuntime()
	if err != nil {
		return err
	}

	manager, err := getRuntimeManager(runtime)
	if err != nil {
		// Not an error for status - just means stack isn't set up
		fmt.Println("Local development stack is not configured")
		fmt.Println("Run 'dk dev up' to start the stack")
		return nil
	}

	ctx := context.Background()
	status, err := manager.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	if !status.Running || len(status.Services) == 0 {
		fmt.Println("Local development stack is not running")
		fmt.Println("Run 'dk dev up' to start the stack")
		return nil
	}

	fmt.Printf("Local development stack status (runtime: %s):\n", runtime)

	formatter := GetFormatter()
	data := formatServiceStatus(status)

	if err := formatter.Format(os.Stdout, data); err != nil {
		return err
	}

	// Show registry cache status for k3d runtime
	if runtime == localdev.RuntimeK3d {
		cacheManager, err := localdev.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to initialize cache manager: %w", err)
		}

		cacheStatus, err := cacheManager.Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to get cache status: %w", err)
		}

		fmt.Println("\nRegistry Cache:")
		if cacheStatus.Running {
			fmt.Printf("  Status:    running\n")
			fmt.Printf("  Endpoint:  %s\n", cacheStatus.Endpoint)
			if cacheStatus.VolumeSize != "" {
				fmt.Printf("  Cache Size: %s\n", cacheStatus.VolumeSize)
			}
		} else if cacheStatus.Exists {
			fmt.Println("  Status:    stopped")
		} else {
			fmt.Println("  Status:    not created")
		}
	}

	return nil
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
