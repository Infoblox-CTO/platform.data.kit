package localdev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
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
	clusterName    string
	namespace      string
	portForwarder  *PortForwarder
	kubeContext    string
	registriesPath string // Path to k3d registries.yaml config (for registry cache)
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

// SetRegistriesPath sets the path to the k3d registries.yaml config.
// This is used to configure k3d to use a registry mirror (e.g., pull-through cache).
func (m *K3dManager) SetRegistriesPath(path string) {
	m.registriesPath = path
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

// WaitForHealthy waits for all services to become healthy concurrently.
// Each chart's health check runs in its own goroutine. If any service fails,
// all errors are collected and returned together.
func (m *K3dManager) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(charts.DefaultCharts))

	// Launch health checks for all charts concurrently
	for _, def := range charts.DefaultCharts {
		wg.Add(1)
		go func(d charts.ChartDefinition) {
			defer wg.Done()
			if err := m.waitForChartHealthy(ctx, d); err != nil {
				errCh <- err
			}
		}(def)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []string
	for err := range errCh {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf("services not ready: %s", strings.Join(errs, "; "))
	}

	return nil
}

// waitForChartHealthy waits for all pods of a single chart to exist and become ready.
func (m *K3dManager) waitForChartHealthy(ctx context.Context, def charts.ChartDefinition) error {
	for labelKey, labelVal := range def.HealthLabels {
		labelSelector := fmt.Sprintf("%s=%s", labelKey, labelVal)

		// First wait for pod to exist (with polling)
		if err := m.waitForPodToExist(ctx, labelSelector, def.HealthTimeout); err != nil {
			return fmt.Errorf("service %s pod not created: %w", def.Name, err)
		}

		// Wait for running pods to be ready.
		// We use a field selector to exclude Succeeded pods (e.g. completed Jobs
		// like redpanda-configuration) since they will never reach condition=ready.
		cmd := exec.CommandContext(ctx, "kubectl",
			"--context", m.kubeContext,
			"wait", "--for=condition=ready", "pod",
			"-l", labelSelector,
			"--field-selector", "status.phase!=Succeeded",
			"-n", m.namespace,
			"--timeout", fmt.Sprintf("%ds", int(def.HealthTimeout.Seconds())),
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("service %s not ready: %s", def.Name, stderr.String())
		}
	}

	return nil
}

// waitForPodToExist polls until a pod matching the given label selector exists.
func (m *K3dManager) waitForPodToExist(ctx context.Context, labelSelector string, timeout time.Duration) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for pod with %s to exist", labelSelector)
			}

			cmd := exec.CommandContext(ctx, "kubectl",
				"--context", m.kubeContext,
				"get", "pods",
				"-l", labelSelector,
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
	args := []string{"cluster", "create", m.clusterName, "--wait", "--timeout", "120s"}

	// Add registry config if set (for pull-through cache)
	if m.registriesPath != "" {
		args = append(args, "--registry-config", m.registriesPath)
	}

	cmd := exec.CommandContext(ctx, "k3d", args...)
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

// deployCharts installs the embedded Helm charts in parallel using the uniform
// chart deployment mechanism. Config overrides are loaded from the hierarchical
// config system and passed through to helm install.
func (m *K3dManager) deployCharts(ctx context.Context, output io.Writer) error {
	// Load config overrides (best-effort — empty map if config unavailable)
	var overrides map[string]charts.ChartOverride
	if cfg, err := LoadHierarchicalConfig(); err == nil && cfg != nil {
		overrides = cfg.Dev.Charts
	}

	result, err := charts.DeployCharts(ctx, charts.DefaultCharts, overrides, m.kubeContext)
	if err != nil {
		return fmt.Errorf("failed to deploy Helm charts: %w", err)
	}

	// Report successes
	for _, name := range result.Succeeded {
		fmt.Fprintf(output, "  ✓ %s deployed\n", name)
	}

	// Report failures
	if result.HasFailures() {
		var msgs []string
		for _, ce := range result.Failed {
			fmt.Fprintf(output, "  ✗ %s failed: %v\n", ce.ChartName, ce.Error)
			msgs = append(msgs, fmt.Sprintf("%s: %v", ce.ChartName, ce.Error))
		}
		return fmt.Errorf("helm installation errors: %v", msgs)
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

// getPortsForService returns the port mappings for a service by looking up
// the chart definitions. Falls back to empty if no match found.
func getPortsForService(serviceName string) []string {
	return charts.PortsForService(charts.DefaultCharts, serviceName)
}

// startPortForwarding starts port forwarding for all services defined in DefaultCharts.
func (m *K3dManager) startPortForwarding(ctx context.Context, output io.Writer) error {
	m.portForwarder = NewPortForwarder(m.kubeContext, m.namespace)

	// Add port forwards from chart definitions
	m.portForwarder.AddForwardsFromCharts(charts.DefaultCharts)

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
