package localdev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/charts"
)

const (
	// DefaultClusterName is the default k3d cluster name for local development.
	DefaultClusterName = "dp-local"
	// DefaultNamespace is the Kubernetes namespace for local dev services.
	DefaultNamespace = "dp-local"
)

// K3dManager manages k3d cluster operations for local development.
type K3dManager struct {
	clusterName   string
	namespace     string
	portForwarder *PortForwarder
	kubeContext   string
}

// NewK3dManager creates a new K3dManager with the specified cluster name.
func NewK3dManager(clusterName string) (*K3dManager, error) {
	if clusterName == "" {
		clusterName = DefaultClusterName
	}

	return &K3dManager{
		clusterName: clusterName,
		namespace:   DefaultNamespace,
		kubeContext: fmt.Sprintf("k3d-%s", clusterName),
	}, nil
}

// Type returns the runtime type for this manager.
func (m *K3dManager) Type() RuntimeType {
	return RuntimeK3d
}

// Up starts the k3d cluster and deploys services.
func (m *K3dManager) Up(ctx context.Context, detach bool, output io.Writer) error {
	// Check if cluster exists
	exists, err := m.clusterExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check cluster status: %w", err)
	}

	if exists {
		// Check if it's running
		running, err := m.clusterRunning(ctx)
		if err != nil {
			return fmt.Errorf("failed to check cluster state: %w", err)
		}

		if !running {
			fmt.Fprintln(output, "Starting existing k3d cluster...")
			if err := m.startCluster(ctx, output); err != nil {
				return fmt.Errorf("failed to start cluster: %w", err)
			}
		} else {
			fmt.Fprintln(output, "k3d cluster is already running")
		}
	} else {
		fmt.Fprintln(output, "Creating new k3d cluster...")
		if err := m.createCluster(ctx, output); err != nil {
			return fmt.Errorf("failed to create cluster: %w", err)
		}
	}

	// Deploy Helm charts
	fmt.Fprintln(output, "Deploying services via Helm...")
	if err := m.deployCharts(ctx, output); err != nil {
		return fmt.Errorf("failed to deploy Helm charts: %w", err)
	}

	// Start port forwarding
	fmt.Fprintln(output, "Setting up port forwarding...")
	if err := m.startPortForwarding(ctx, output); err != nil {
		return fmt.Errorf("failed to set up port forwarding: %w", err)
	}

	return nil
}

// Down stops the k3d cluster.
func (m *K3dManager) Down(ctx context.Context, removeVolumes bool, output io.Writer) error {
	// Stop port forwarding first
	m.stopPortForwarding()

	exists, err := m.clusterExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check cluster status: %w", err)
	}

	if !exists {
		fmt.Fprintln(output, "k3d cluster does not exist")
		return nil
	}

	if removeVolumes {
		fmt.Fprintln(output, "Deleting k3d cluster and volumes...")
		return m.deleteCluster(ctx, output)
	}

	fmt.Fprintln(output, "Stopping k3d cluster...")
	return m.stopCluster(ctx, output)
}

// Status returns the current status of the k3d cluster.
func (m *K3dManager) Status(ctx context.Context) (*StackStatus, error) {
	status := &StackStatus{
		Running:  false,
		Runtime:  RuntimeK3d,
		Services: []ServiceStatus{},
	}

	exists, err := m.clusterExists(ctx)
	if err != nil {
		return status, nil
	}

	if !exists {
		return status, nil
	}

	running, err := m.clusterRunning(ctx)
	if err != nil {
		return status, nil
	}

	status.Running = running

	if running {
		// Get pod statuses
		pods, err := m.getPodStatuses(ctx)
		if err == nil {
			status.Services = pods
		}
	}

	return status, nil
}

// WaitForHealthy waits for all services to become healthy.
func (m *K3dManager) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	services := []string{"redpanda", "localstack", "postgres"}

	for _, svc := range services {
		// First wait for pod to exist (with polling)
		if err := m.waitForPodToExist(ctx, svc, timeout); err != nil {
			return fmt.Errorf("service %s pod not created: %w", svc, err)
		}

		// Then wait for pod to be ready
		cmd := exec.CommandContext(ctx, "kubectl",
			"--context", m.kubeContext,
			"wait", "--for=condition=ready", "pod",
			"-l", fmt.Sprintf("app=%s", svc),
			"-n", m.namespace,
			"--timeout", fmt.Sprintf("%ds", int(timeout.Seconds())),
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("service %s not ready: %s", svc, stderr.String())
		}
	}

	return nil
}

// waitForPodToExist polls until a pod with the given app label exists.
func (m *K3dManager) waitForPodToExist(ctx context.Context, appLabel string, timeout time.Duration) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for pod with app=%s to exist", appLabel)
			}

			cmd := exec.CommandContext(ctx, "kubectl",
				"--context", m.kubeContext,
				"get", "pods",
				"-l", fmt.Sprintf("app=%s", appLabel),
				"-n", m.namespace,
				"-o", "name",
			)

			output, err := cmd.Output()
			if err == nil && len(bytes.TrimSpace(output)) > 0 {
				// Pod exists
				return nil
			}
		}
	}
}

// clusterExists checks if the k3d cluster exists.
func (m *K3dManager) clusterExists(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	var clusters []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(output, &clusters); err != nil {
		return false, err
	}

	for _, c := range clusters {
		if c.Name == m.clusterName {
			return true, nil
		}
	}

	return false, nil
}

// clusterRunning checks if the k3d cluster is running.
func (m *K3dManager) clusterRunning(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	var clusters []struct {
		Name           string `json:"name"`
		ServersRunning int    `json:"serversRunning"`
	}
	if err := json.Unmarshal(output, &clusters); err != nil {
		return false, err
	}

	for _, c := range clusters {
		if c.Name == m.clusterName {
			return c.ServersRunning > 0, nil
		}
	}

	return false, nil
}

// createCluster creates a new k3d cluster.
func (m *K3dManager) createCluster(ctx context.Context, output io.Writer) error {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "create", m.clusterName,
		"--wait",
		"--timeout", "120s",
	)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

// startCluster starts an existing k3d cluster.
func (m *K3dManager) startCluster(ctx context.Context, output io.Writer) error {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "start", m.clusterName)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

// stopCluster stops the k3d cluster without deleting it.
func (m *K3dManager) stopCluster(ctx context.Context, output io.Writer) error {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "stop", m.clusterName)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

// deleteCluster deletes the k3d cluster including volumes.
func (m *K3dManager) deleteCluster(ctx context.Context, output io.Writer) error {
	cmd := exec.CommandContext(ctx, "k3d", "cluster", "delete", m.clusterName)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

// deployCharts installs the embedded Helm charts.
func (m *K3dManager) deployCharts(ctx context.Context, output io.Writer) error {
	// Create temporary directory for chart extraction
	tempDir, err := os.MkdirTemp("", "dp-charts-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract and install each chart
	for _, chartName := range charts.ChartNames {
		chartDir := filepath.Join(tempDir, chartName)
		if err := extractChart(chartName, chartDir); err != nil {
			return fmt.Errorf("failed to extract chart %s: %w", chartName, err)
		}

		// Install chart using helm upgrade --install
		releaseName := fmt.Sprintf("dp-%s", chartName)
		cmd := exec.CommandContext(ctx, "helm",
			"upgrade", "--install", releaseName, chartDir,
			"--kube-context", m.kubeContext,
			"--namespace", m.namespace,
			"--create-namespace",
			"--wait",
			"--timeout", "120s",
		)
		cmd.Stdout = output
		cmd.Stderr = output

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install chart %s: %w", chartName, err)
		}
	}

	return nil
}

// extractChart extracts an embedded chart to a directory.
func extractChart(chartName string, destDir string) error {
	// Verify the chart exists in embedded FS
	_, err := charts.FS.ReadDir(chartName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	return extractDir(chartName, destDir)
}

// extractDir recursively extracts embedded files to a directory.
func extractDir(srcPath string, destPath string) error {
	entries, err := charts.FS.ReadDir(srcPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcFile := filepath.Join(srcPath, entry.Name())
		destFile := filepath.Join(destPath, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(destFile, 0755); err != nil {
				return err
			}
			if err := extractDir(srcFile, destFile); err != nil {
				return err
			}
		} else {
			content, err := charts.FS.ReadFile(srcFile)
			if err != nil {
				return err
			}
			if err := os.WriteFile(destFile, content, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// getPodStatuses returns the status of all pods in the namespace.
func (m *K3dManager) getPodStatuses(ctx context.Context) ([]ServiceStatus, error) {
	cmd := exec.CommandContext(ctx, "kubectl",
		"--context", m.kubeContext,
		"get", "pods",
		"-n", m.namespace,
		"-o", "json",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var podList struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Phase      string `json:"phase"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &podList); err != nil {
		return nil, err
	}

	services := make([]ServiceStatus, 0, len(podList.Items))
	for _, pod := range podList.Items {
		appName := pod.Metadata.Labels["app"]
		if appName == "" {
			appName = pod.Metadata.Name
		}

		health := "unknown"
		for _, cond := range pod.Status.Conditions {
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					health = "healthy"
				} else {
					health = "unhealthy"
				}
				break
			}
		}

		// Get port info based on app name
		ports := getPortsForService(appName)

		services = append(services, ServiceStatus{
			Name:   appName,
			Status: string(pod.Status.Phase),
			Health: health,
			Ports:  ports,
		})
	}

	return services, nil
}

// getPortsForService returns the port mappings for a service.
func getPortsForService(serviceName string) []string {
	switch serviceName {
	case "redpanda":
		return []string{"19092:9092", "18081:8081"}
	case "localstack":
		return []string{"4566:4566"}
	case "postgres":
		return []string{"5432:5432"}
	default:
		return nil
	}
}

// startPortForwarding starts port forwarding for all services.
func (m *K3dManager) startPortForwarding(ctx context.Context, output io.Writer) error {
	m.portForwarder = NewPortForwarder(m.kubeContext, m.namespace)

	// Add port forwards for each service
	m.portForwarder.AddForward("redpanda", 19092, 9092)
	m.portForwarder.AddForward("localstack", 4566, 4566)
	m.portForwarder.AddForward("postgres", 5432, 5432)

	return m.portForwarder.Start(ctx)
}

// stopPortForwarding stops all port forwards.
func (m *K3dManager) stopPortForwarding() {
	if m.portForwarder != nil {
		m.portForwarder.Stop()
	}
}

// Ensure K3dManager implements RuntimeManager at compile time.
var _ RuntimeManager = (*K3dManager)(nil)
