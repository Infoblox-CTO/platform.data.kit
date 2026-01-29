package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
	"github.com/spf13/cobra"
)

var (
	devComposePath   string
	devRemoveVolumes bool
	devRuntime       string
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
	devCmd.PersistentFlags().StringVar(&devRuntime, "runtime", "k3d", "Runtime to use: 'k3d' (default) or 'compose'")
	devDownCmd.Flags().BoolVar(&devRemoveVolumes, "volumes", false, "Remove data volumes when stopping")
}

// getWorkspacePath returns the DP workspace path from environment or config.
func getWorkspacePath() string {
	// Check DP_WORKSPACE_PATH environment variable first
	if envPath := os.Getenv("DP_WORKSPACE_PATH"); envPath != "" {
		return envPath
	}

	// Check config file
	config, err := localdev.LoadConfig()
	if err == nil && config != nil && config.Dev.Workspace != "" {
		return config.Dev.Workspace
	}

	return ""
}

// findComposeFile searches for the docker-compose file in standard locations
func findComposeFile() (string, error) {
	// First check DP_WORKSPACE_PATH environment variable
	workspacePath := getWorkspacePath()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Build search paths
	var searchPaths []string

	// If workspace path is set, search there first
	if workspacePath != "" {
		searchPaths = append(searchPaths,
			filepath.Join(workspacePath, "hack", "compose", "docker-compose.yaml"),
			filepath.Join(workspacePath, "hack", "compose", "docker-compose.yml"),
			filepath.Join(workspacePath, "docker-compose.yaml"),
			filepath.Join(workspacePath, "docker-compose.yml"),
		)
	}

	// Search paths relative to current directory
	searchPaths = append(searchPaths,
		filepath.Join(cwd, "hack", "compose", "docker-compose.yaml"),
		filepath.Join(cwd, "hack", "compose", "docker-compose.yml"),
		filepath.Join(cwd, "docker-compose.yaml"),
		filepath.Join(cwd, "docker-compose.yml"),
	)

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

	return "", fmt.Errorf("docker-compose.yaml not found; specify path with --compose, set DP_WORKSPACE_PATH, or run from DP workspace")
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
	case "compose", "docker-compose":
		return localdev.RuntimeCompose, nil
	default:
		return "", fmt.Errorf("unsupported runtime %q; use 'k3d' or 'compose'", devRuntime)
	}
}

// getRuntimeManager creates the appropriate runtime manager based on the selected runtime.
func getRuntimeManager(runtime localdev.RuntimeType) (localdev.RuntimeManager, error) {
	switch runtime {
	case localdev.RuntimeCompose:
		composePath := devComposePath
		var err error

		if composePath == "" {
			composePath, err = findComposeFile()
			if err != nil {
				return nil, err
			}
		}

		return localdev.NewComposeManager(composePath)

	case localdev.RuntimeK3d:
		k3dManager, err := localdev.NewK3dManager("dp-local")
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
		fmt.Println("\nUse 'dp dev down' to stop the stack")
		return nil
	}

	// Check port availability only if not already running
	portChecker := localdev.NewPortChecker(1 * time.Second)
	if err := portChecker.CheckAllAvailable([]int{19092, 4566, 5432}); err != nil {
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
		fmt.Println("Use 'dp dev status' to check service status")
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
	fmt.Println("  Kafka:           localhost:19092")
	fmt.Println("  Schema Registry: localhost:18081")
	fmt.Println("  Redpanda Console: http://localhost:8080")
	fmt.Println("  S3 (LocalStack): localhost:4566")
	fmt.Println("  PostgreSQL:      localhost:5432")

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
		fmt.Println("Run 'dp dev up' to start the stack")
		return nil
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
