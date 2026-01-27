package localdev

import (
	"fmt"
	"net"
	"time"
)

// DefaultPorts defines the standard port mappings for local development services.
var DefaultPorts = map[string]int{
	"redpanda":   19092, // Kafka protocol
	"localstack": 4566,  // AWS services
	"postgres":   5432,  // PostgreSQL
}

// PortChecker validates port availability.
type PortChecker struct {
	timeout time.Duration
}

// NewPortChecker creates a new port checker with the specified timeout.
func NewPortChecker(timeout time.Duration) *PortChecker {
	if timeout <= 0 {
		timeout = 1 * time.Second
	}
	return &PortChecker{timeout: timeout}
}

// PortCheckResult contains the result of a port availability check.
type PortCheckResult struct {
	// Port is the port number checked.
	Port int
	// Available indicates whether the port is free to use.
	Available bool
	// Service is the service name associated with this port (if known).
	Service string
	// Error contains any error message if the port is in use.
	Error string
}

// CheckPort checks if a specific port is available for binding.
func (c *PortChecker) CheckPort(port int) PortCheckResult {
	result := PortCheckResult{
		Port:      port,
		Available: true,
	}

	// Look up service name
	for svc, p := range DefaultPorts {
		if p == port {
			result.Service = svc
			break
		}
	}

	// Try to listen on the port
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		result.Available = false
		result.Error = fmt.Sprintf("port %d is already in use", port)
		return result
	}
	listener.Close()

	return result
}

// CheckPorts checks multiple ports and returns results for each.
func (c *PortChecker) CheckPorts(ports []int) []PortCheckResult {
	results := make([]PortCheckResult, len(ports))
	for i, port := range ports {
		results[i] = c.CheckPort(port)
	}
	return results
}

// CheckDefaultPorts checks all default service ports.
func (c *PortChecker) CheckDefaultPorts() []PortCheckResult {
	ports := make([]int, 0, len(DefaultPorts))
	for _, port := range DefaultPorts {
		ports = append(ports, port)
	}
	return c.CheckPorts(ports)
}

// CheckAllAvailable checks if all specified ports are available.
// Returns an error listing all ports that are in use.
func (c *PortChecker) CheckAllAvailable(ports []int) error {
	results := c.CheckPorts(ports)

	var inUse []int
	for _, r := range results {
		if !r.Available {
			inUse = append(inUse, r.Port)
		}
	}

	if len(inUse) > 0 {
		return fmt.Errorf("ports already in use: %v", inUse)
	}

	return nil
}

// WaitForPort waits until a port becomes available (listening).
// This is useful for waiting for services to start.
func WaitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for port %d to become available", port)
}
