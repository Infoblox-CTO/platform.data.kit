// Package cmd contains the CLI commands for DK.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <package>",
	Short: "Stream logs from a running or completed run",
	Long: `Stream logs from a data package run.

By default, shows logs from the most recent run in the dev environment.
Use --run to specify a specific run ID, or --environment to change the target.

Example:
  # Show logs from last run in dev
  dk logs kafka-s3-pipeline

  # Follow logs in real-time
  dk logs kafka-s3-pipeline --follow

  # Show logs from specific run
  dk logs kafka-s3-pipeline --run 20240115-120000

  # Show logs from production
  dk logs kafka-s3-pipeline --environment prod`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

var (
	logsFollow      bool
	logsRunID       string
	logsEnvironment string
	logsTail        int
	logsSince       string
	logsTimestamps  bool
)

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().StringVar(&logsRunID, "run", "", "Specific run ID to show logs for")
	logsCmd.Flags().StringVarP(&logsEnvironment, "environment", "e", "dev", "Environment (dev, int, prod)")
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show from the end")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since timestamp (e.g., 2024-01-15T10:00:00Z)")
	logsCmd.Flags().BoolVarP(&logsTimestamps, "timestamps", "t", false, "Show timestamps")
}

func runLogs(cmd *cobra.Command, args []string) error {
	packageName := args[0]

	// Validate environment
	if logsEnvironment != "dev" && logsEnvironment != "int" && logsEnvironment != "prod" {
		return fmt.Errorf("invalid environment: %s (must be dev, int, or prod)", logsEnvironment)
	}

	if logsEnvironment == "dev" {
		// For local dev, use Docker logs
		return streamDockerLogs(packageName)
	}

	// For remote environments, use kubectl
	return streamKubernetesLogs(packageName, logsEnvironment)
}

func streamDockerLogs(packageName string) error {
	containerName := fmt.Sprintf("dk-%s-runner", packageName)

	// Check if container exists
	checkCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check container: %w", err)
	}

	if strings.TrimSpace(string(output)) == "" {
		// No container found, show sample output
		fmt.Printf("No recent runs found for %s in local environment.\n", packageName)
		fmt.Println("\nTo run the package locally:")
		fmt.Printf("  dk run %s\n", packageName)
		return nil
	}

	// Build docker logs command
	args := []string{"logs"}
	if logsFollow {
		args = append(args, "-f")
	}
	if logsTail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", logsTail))
	}
	if logsTimestamps {
		args = append(args, "-t")
	}
	if logsSince != "" {
		args = append(args, "--since", logsSince)
	}
	args = append(args, containerName)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func streamKubernetesLogs(packageName, environment string) error {
	namespace := fmt.Sprintf("dk-%s", environment)
	labelSelector := fmt.Sprintf("datakit.infoblox.dev/package=%s", packageName)

	// Check for kubeconfig
	if os.Getenv("KUBECONFIG") == "" {
		home, _ := os.UserHomeDir()
		defaultKubeconfig := home + "/.kube/config"
		if _, err := os.Stat(defaultKubeconfig); os.IsNotExist(err) {
			return fmt.Errorf("no kubeconfig found. Set KUBECONFIG or ensure ~/.kube/config exists")
		}
	}

	// Get pods for the package
	getPods := exec.Command("kubectl", "get", "pods",
		"-n", namespace,
		"-l", labelSelector,
		"--sort-by=.status.startTime",
		"-o", "jsonpath={.items[-1].metadata.name}")

	podName, err := getPods.Output()
	if err != nil {
		return fmt.Errorf("failed to get pod: %w (is kubectl configured for %s?)", err, environment)
	}

	if len(podName) == 0 {
		fmt.Printf("No pods found for %s in %s environment.\n", packageName, environment)
		return nil
	}

	fmt.Printf("Streaming logs from pod %s in %s...\n\n", string(podName), namespace)

	// Build kubectl logs command
	args := []string{"logs", "-n", namespace}
	if logsFollow {
		args = append(args, "-f")
	}
	if logsTail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", logsTail))
	}
	if logsTimestamps {
		args = append(args, "--timestamps")
	}
	if logsSince != "" {
		args = append(args, "--since-time", logsSince)
	}
	args = append(args, string(podName))

	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// streamWithContext streams logs with context cancellation support.
func streamWithContext(ctx context.Context, cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				cmd.Process.Kill()
				return
			default:
				fmt.Println(scanner.Text())
			}
		}
	}()

	return cmd.Wait()
}

// formatLogLine formats a log line with optional timestamp.
func formatLogLine(line string, showTimestamp bool) string {
	if showTimestamp {
		return fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), line)
	}
	return line
}
