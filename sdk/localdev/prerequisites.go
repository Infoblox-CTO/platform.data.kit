package localdev

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ContainerRuntime represents the detected container runtime.
type ContainerRuntime string

const (
	// ContainerRuntimeDocker represents Docker Desktop or standalone Docker.
	ContainerRuntimeDocker ContainerRuntime = "docker"
	// ContainerRuntimeRancher represents Rancher Desktop.
	ContainerRuntimeRancher ContainerRuntime = "rancher"
	// ContainerRuntimeNone represents no container runtime found.
	ContainerRuntimeNone ContainerRuntime = "none"
)

// PrerequisiteChecker validates that required tools are installed.
type PrerequisiteChecker struct {
	// requiredTools maps tool names to their check commands
	requiredTools map[string][]string
	// containerRuntime is the detected container runtime
	containerRuntime ContainerRuntime
}

// PrerequisiteResult contains the result of a prerequisite check.
type PrerequisiteResult struct {
	// Tool is the name of the tool being checked.
	Tool string
	// Available indicates whether the tool is installed and accessible.
	Available bool
	// Version is the version string if available.
	Version string
	// Error contains any error message if the check failed.
	Error string
}

// NewPrerequisiteChecker creates a new checker for the given runtime.
func NewPrerequisiteChecker(runtime RuntimeType) *PrerequisiteChecker {
	checker := &PrerequisiteChecker{
		requiredTools:    make(map[string][]string),
		containerRuntime: ContainerRuntimeNone,
	}

	// Detect container runtime (Rancher Desktop or Docker)
	checker.containerRuntime = detectContainerRuntime()

	// Add container runtime check based on what's detected
	switch checker.containerRuntime {
	case ContainerRuntimeRancher:
		// Rancher Desktop provides docker CLI compatibility
		checker.requiredTools["rancher"] = []string{"rdctl", "version"}
	case ContainerRuntimeDocker:
		checker.requiredTools["docker"] = []string{"docker", "--version"}
	default:
		// Check for either - will show appropriate error
		checker.requiredTools["docker"] = []string{"docker", "--version"}
	}

	switch runtime {
	case RuntimeK3d:
		checker.requiredTools["k3d"] = []string{"k3d", "version"}
		checker.requiredTools["kubectl"] = []string{"kubectl", "version", "--client"}
		checker.requiredTools["helm"] = []string{"helm", "version", "--short"}
	case RuntimeCompose:
		// Docker Compose is bundled with Docker/Rancher, but check anyway
		checker.requiredTools["docker-compose"] = []string{"docker", "compose", "version"}
	}

	return checker
}

// detectContainerRuntime detects which container runtime is available.
func detectContainerRuntime() ContainerRuntime {
	// Check for Rancher Desktop first (rdctl command)
	if isCommandAvailable("rdctl", "version") {
		return ContainerRuntimeRancher
	}

	// Check for Docker
	if isCommandAvailable("docker", "--version") {
		// Check if docker is provided by Rancher Desktop
		cmd := exec.Command("docker", "info")
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), "rancher-desktop") {
			return ContainerRuntimeRancher
		}
		return ContainerRuntimeDocker
	}

	return ContainerRuntimeNone
}

// isCommandAvailable checks if a command is available in PATH.
func isCommandAvailable(name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	return cmd.Run() == nil
}

// GetContainerRuntime returns the detected container runtime.
func (c *PrerequisiteChecker) GetContainerRuntime() ContainerRuntime {
	return c.containerRuntime
}

// Check validates all prerequisites and returns the results.
func (c *PrerequisiteChecker) Check(ctx context.Context) []PrerequisiteResult {
	results := make([]PrerequisiteResult, 0, len(c.requiredTools))

	for tool, cmd := range c.requiredTools {
		result := c.checkTool(ctx, tool, cmd)
		results = append(results, result)
	}

	return results
}

// CheckAll validates all prerequisites and returns an error if any are missing.
func (c *PrerequisiteChecker) CheckAll(ctx context.Context) error {
	results := c.Check(ctx)

	var missing []string
	for _, r := range results {
		if !r.Available {
			missing = append(missing, r.Tool)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	return nil
}

// checkTool checks if a single tool is available.
func (c *PrerequisiteChecker) checkTool(ctx context.Context, name string, cmd []string) PrerequisiteResult {
	result := PrerequisiteResult{
		Tool: name,
	}

	if len(cmd) == 0 {
		result.Error = "no check command defined"
		return result
	}

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	if err := execCmd.Run(); err != nil {
		result.Available = false
		result.Error = fmt.Sprintf("%s not found or not working: %v", name, err)
		return result
	}

	result.Available = true
	// Extract version from output
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		output = strings.TrimSpace(stderr.String())
	}
	result.Version = extractVersion(output)

	return result
}

// extractVersion attempts to extract a version string from command output.
func extractVersion(output string) string {
	// Take first line and clean up
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		line := strings.TrimSpace(lines[0])
		// Remove common prefixes
		line = strings.TrimPrefix(line, "Docker version ")
		line = strings.TrimPrefix(line, "Docker Compose version ")
		line = strings.TrimPrefix(line, "kubectl version ")
		line = strings.TrimPrefix(line, "k3d version ")
		// Take first word if it looks like a version
		parts := strings.Fields(line)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return output
}

// IsDockerRunning checks if the Docker daemon is running (works with both Docker and Rancher Desktop).
func IsDockerRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info")
	return cmd.Run() == nil
}

// IsRancherDesktopRunning checks if Rancher Desktop is running.
func IsRancherDesktopRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "rdctl", "api", "/v1/settings")
	return cmd.Run() == nil
}

// IsContainerRuntimeRunning checks if any container runtime is running.
func IsContainerRuntimeRunning(ctx context.Context) bool {
	return IsDockerRunning(ctx) || IsRancherDesktopRunning(ctx)
}

// GetContainerRuntimeName returns a user-friendly name for the detected runtime.
func GetContainerRuntimeName() string {
	runtime := detectContainerRuntime()
	switch runtime {
	case ContainerRuntimeRancher:
		return "Rancher Desktop"
	case ContainerRuntimeDocker:
		return "Docker"
	default:
		return "Docker"
	}
}
