package charts

import (
	"fmt"
	"time"
)

// ChartDefinition describes a single dev dependency Helm chart.
// All deployment, port-forwarding, health-checking, status, and teardown code
// operates on []ChartDefinition — no service-specific code paths.
type ChartDefinition struct {
	// Name is the unique identifier (matches chart directory name under sdk/localdev/charts/).
	Name string

	// ReleaseName is the Helm release name used with helm upgrade --install.
	// Convention: "dk-" + Name.
	ReleaseName string

	// Namespace is the Kubernetes namespace for deployment.
	Namespace string

	// PortForwards defines the port-forwarding rules for this chart's services.
	PortForwards []PortForwardRule

	// HealthLabels are the pod label selectors used to check health.
	HealthLabels map[string]string

	// HealthTimeout is the maximum duration to wait for pods to become healthy.
	HealthTimeout time.Duration

	// DisplayEndpoints are the human-readable endpoints shown in status output.
	DisplayEndpoints []DisplayEndpoint
}

// PortForwardRule defines a single port-forwarding rule from localhost
// to a Kubernetes service.
type PortForwardRule struct {
	// ServiceName is the Kubernetes service to port-forward to.
	ServiceName string

	// LocalPort is the port on localhost.
	LocalPort int

	// RemotePort is the port on the Kubernetes service.
	RemotePort int
}

// DisplayEndpoint is a human-readable endpoint for status display.
type DisplayEndpoint struct {
	// Label is the display name (e.g., "Kafka", "S3 API").
	Label string

	// URL is the localhost URL (e.g., "localhost:19092").
	URL string
}

// ChartOverride holds user-configurable overrides for a chart.
type ChartOverride struct {
	// Version overrides the embedded chart version.
	// Empty string means use the embedded default.
	Version string `yaml:"version,omitempty"`

	// Values are additional Helm --set values merged at deploy time.
	Values map[string]interface{} `yaml:"values,omitempty"`
}

// DeployResult reports per-chart deployment outcome.
type DeployResult struct {
	// Succeeded lists chart names that deployed successfully.
	Succeeded []string

	// Failed lists charts that failed with reasons.
	Failed []ChartError
}

// ChartError pairs a chart name with its deployment error.
type ChartError struct {
	ChartName string
	Error     error
}

// HasFailures returns true if any chart deployments failed.
func (r *DeployResult) HasFailures() bool {
	return len(r.Failed) > 0
}

// AllLocalPorts returns all local ports used by a slice of chart definitions.
// Useful for port availability checks.
func AllLocalPorts(charts []ChartDefinition) []int {
	var ports []int
	for _, c := range charts {
		for _, pf := range c.PortForwards {
			ports = append(ports, pf.LocalPort)
		}
	}
	return ports
}

// PortsForService returns port forward strings (e.g., "19092:9092") for a
// chart matching the given app label. Used for status display.
func PortsForService(defs []ChartDefinition, appLabel string) []string {
	for _, c := range defs {
		if c.Name == appLabel {
			result := make([]string, 0, len(c.PortForwards))
			for _, pf := range c.PortForwards {
				result = append(result, fmt.Sprintf("%d:%d", pf.LocalPort, pf.RemotePort))
			}
			return result
		}
		// Also check health labels for app label match
		for _, v := range c.HealthLabels {
			if v == appLabel {
				result := make([]string, 0, len(c.PortForwards))
				for _, pf := range c.PortForwards {
					result = append(result, fmt.Sprintf("%d:%d", pf.LocalPort, pf.RemotePort))
				}
				return result
			}
		}
	}
	return nil
}
