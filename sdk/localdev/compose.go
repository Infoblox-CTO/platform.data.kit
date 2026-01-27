// Package localdev provides utilities for local development environment management.
package localdev

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ComposeManager manages Docker Compose operations for local development.
type ComposeManager struct {
	composePath string
	projectName string
	workDir     string
}

// ServiceStatus represents the status of a Docker Compose service.
type ServiceStatus struct {
	Name   string
	Status string
	Ports  []string
	Health string
}

// StackStatus represents the overall status of the local dev stack.
type StackStatus struct {
	Running  bool
	Runtime  RuntimeType
	Services []ServiceStatus
	Errors   []string
}

// NewComposeManager creates a new ComposeManager.
func NewComposeManager(composePath string) (*ComposeManager, error) {
	if composePath == "" {
		searchPaths := []string{
			"hack/compose/docker-compose.yaml",
			"hack/compose/docker-compose.yml",
			"docker-compose.yaml",
			"docker-compose.yml",
		}

		for _, p := range searchPaths {
			if _, err := os.Stat(p); err == nil {
				composePath = p
				break
			}
		}

		if composePath == "" {
			return nil, fmt.Errorf("docker-compose.yaml not found in standard locations")
		}
	}

	absPath, err := filepath.Abs(composePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve compose path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("compose file not found: %s", absPath)
	}

	return &ComposeManager{
		composePath: absPath,
		projectName: "dp",
		workDir:     filepath.Dir(absPath),
	}, nil
}

// Up starts the Docker Compose stack.
func (m *ComposeManager) Up(ctx context.Context, detach bool, output io.Writer) error {
	args := []string{"-f", m.composePath, "-p", m.projectName, "up"}
	if detach {
		args = append(args, "-d")
	}
	args = append(args, "--wait")

	return m.runCommand(ctx, args, output)
}

// Down stops and removes the Docker Compose stack.
func (m *ComposeManager) Down(ctx context.Context, removeVolumes bool, output io.Writer) error {
	args := []string{"-f", m.composePath, "-p", m.projectName, "down"}
	if removeVolumes {
		args = append(args, "-v")
	}

	return m.runCommand(ctx, args, output)
}

// Status returns the current status of the stack.
func (m *ComposeManager) Status(ctx context.Context) (*StackStatus, error) {
	args := []string{"-f", m.composePath, "-p", m.projectName, "ps", "--format", "{{.Name}}|{{.Status}}|{{.Ports}}"}

	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = m.workDir

	out, err := cmd.Output()
	if err != nil {
		cmd = exec.CommandContext(ctx, "docker-compose", args...)
		cmd.Dir = m.workDir
		out, err = cmd.Output()
		if err != nil {
			return &StackStatus{Running: false}, nil
		}
	}

	status := &StackStatus{
		Running:  false,
		Services: []ServiceStatus{},
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}

		svc := ServiceStatus{
			Name:   parts[0],
			Status: parts[1],
		}

		if len(parts) > 2 && parts[2] != "" {
			svc.Ports = strings.Split(parts[2], ", ")
		}

		statusLower := strings.ToLower(svc.Status)
		switch {
		case strings.Contains(statusLower, "healthy"):
			svc.Health = "healthy"
			status.Running = true
		case strings.Contains(statusLower, "unhealthy"):
			svc.Health = "unhealthy"
		case strings.Contains(statusLower, "starting"):
			svc.Health = "starting"
			status.Running = true
		case strings.Contains(statusLower, "up"):
			svc.Health = "running"
			status.Running = true
		case strings.Contains(statusLower, "exited"):
			svc.Health = "exited"
		default:
			svc.Health = "unknown"
		}

		status.Services = append(status.Services, svc)
	}

	return status, nil
}

// Logs streams logs from the specified services.
func (m *ComposeManager) Logs(ctx context.Context, services []string, follow bool, output io.Writer) error {
	args := []string{"-f", m.composePath, "-p", m.projectName, "logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, services...)

	return m.runCommand(ctx, args, output)
}

// WaitForHealthy waits for all services to become healthy.
func (m *ComposeManager) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for services to become healthy")
		case <-ticker.C:
			status, err := m.Status(ctx)
			if err != nil {
				continue
			}

			allHealthy := true
			for _, svc := range status.Services {
				if strings.Contains(svc.Name, "-init") && svc.Health == "exited" {
					continue
				}
				if svc.Health != "healthy" && svc.Health != "running" {
					allHealthy = false
					break
				}
			}

			if allHealthy && len(status.Services) > 0 {
				return nil
			}
		}
	}
}

// Exec runs a command in a running container.
func (m *ComposeManager) Exec(ctx context.Context, service string, command []string, output io.Writer) error {
	args := []string{"-f", m.composePath, "-p", m.projectName, "exec", "-T", service}
	args = append(args, command...)

	return m.runCommand(ctx, args, output)
}

// runCommand executes a docker compose command.
func (m *ComposeManager) runCommand(ctx context.Context, args []string, output io.Writer) error {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = m.workDir
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		cmd = exec.CommandContext(ctx, "docker-compose", args...)
		cmd.Dir = m.workDir
		cmd.Stdout = output
		cmd.Stderr = output

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("docker compose command failed: %w", err)
		}
	}

	return nil
}

// GetComposePath returns the path to the compose file.
func (m *ComposeManager) GetComposePath() string {
	return m.composePath
}

// GetProjectName returns the project name.
func (m *ComposeManager) GetProjectName() string {
	return m.projectName
}

// Type returns the runtime type for this manager.
func (m *ComposeManager) Type() RuntimeType {
	return RuntimeCompose
}
