// Package runner provides local execution capabilities for DP pipelines.
package runner

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// HealthStatus represents the health state of a container.
type HealthStatus string

const (
	HealthStatusUnknown   HealthStatus = "unknown"
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthChecker checks the health of a running container.
type HealthChecker struct {
	containerID string
	probe       *contracts.Probe
	httpClient  *http.Client
}

// NewHealthChecker creates a new health checker for a container.
func NewHealthChecker(containerID string, probe *contracts.Probe) *HealthChecker {
	timeout := 10 * time.Second // default timeout
	if probe != nil && probe.TimeoutSeconds > 0 {
		timeout = time.Duration(probe.TimeoutSeconds) * time.Second
	}

	return &HealthChecker{
		containerID: containerID,
		probe:       probe,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Check performs a single health check and returns the status.
func (h *HealthChecker) Check(ctx context.Context) (HealthStatus, error) {
	if h.probe == nil {
		return HealthStatusUnknown, fmt.Errorf("no probe configured")
	}

	switch {
	case h.probe.HTTPGet != nil:
		return h.checkHTTP(ctx)
	case h.probe.Exec != nil:
		return h.checkExec(ctx)
	case h.probe.TCPSocket != nil:
		return h.checkTCP(ctx)
	default:
		return HealthStatusUnknown, fmt.Errorf("no probe action configured")
	}
}

// checkHTTP performs an HTTP health check.
func (h *HealthChecker) checkHTTP(ctx context.Context) (HealthStatus, error) {
	httpGet := h.probe.HTTPGet

	scheme := httpGet.Scheme
	if scheme == "" {
		scheme = "HTTP"
	}

	url := fmt.Sprintf("%s://localhost:%d%s", strings.ToLower(scheme), httpGet.Port, httpGet.Path)

	return h.checkHTTPWithURL(ctx, url)
}

// checkHTTPWithURL performs an HTTP health check against a specific URL.
// This allows for testing with a mock server.
func (h *HealthChecker) checkHTTPWithURL(ctx context.Context, url string) (HealthStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Drain the body to ensure connection reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	// Status codes 200-399 are considered healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return HealthStatusHealthy, nil
	}

	return HealthStatusUnhealthy, fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
}

// checkExec performs an exec health check by running a command in the container.
func (h *HealthChecker) checkExec(ctx context.Context) (HealthStatus, error) {
	execAction := h.probe.Exec

	if len(execAction.Command) == 0 {
		return HealthStatusUnhealthy, fmt.Errorf("exec probe has no command")
	}

	// Build docker exec command
	args := []string{"exec", h.containerID}
	args = append(args, execAction.Command...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("exec probe failed: %w (output: %s)", err, string(output))
	}

	return HealthStatusHealthy, nil
}

// checkTCP performs a TCP health check by attempting to connect to the port.
func (h *HealthChecker) checkTCP(ctx context.Context) (HealthStatus, error) {
	tcpSocket := h.probe.TCPSocket

	addr := fmt.Sprintf("localhost:%d", tcpSocket.Port)

	dialer := net.Dialer{
		Timeout: time.Duration(h.probe.TimeoutSeconds) * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return HealthStatusUnhealthy, fmt.Errorf("TCP connect failed: %w", err)
	}
	conn.Close()

	return HealthStatusHealthy, nil
}

// WaitForHealthy waits for the container to become healthy.
// It polls the health check at the specified interval until the container
// is healthy or the context is cancelled.
func (h *HealthChecker) WaitForHealthy(ctx context.Context) error {
	if h.probe == nil {
		// No probe configured, assume healthy immediately
		return nil
	}

	// Wait for initial delay
	if h.probe.InitialDelaySeconds > 0 {
		select {
		case <-time.After(time.Duration(h.probe.InitialDelaySeconds) * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	period := time.Duration(h.probe.PeriodSeconds) * time.Second
	if period == 0 {
		period = 10 * time.Second
	}

	failureThreshold := h.probe.FailureThreshold
	if failureThreshold == 0 {
		failureThreshold = 3
	}

	successThreshold := h.probe.SuccessThreshold
	if successThreshold == 0 {
		successThreshold = 1
	}

	consecutiveSuccesses := 0
	consecutiveFailures := 0

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		status, err := h.Check(ctx)

		if status == HealthStatusHealthy {
			consecutiveSuccesses++
			consecutiveFailures = 0

			if consecutiveSuccesses >= successThreshold {
				return nil
			}
		} else {
			consecutiveFailures++
			consecutiveSuccesses = 0

			if consecutiveFailures >= failureThreshold {
				return fmt.Errorf("health check failed after %d attempts: %w", failureThreshold, err)
			}
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// HealthPoller continuously monitors container health and reports state changes.
type HealthPoller struct {
	checker    *HealthChecker
	lastStatus HealthStatus
	onChange   func(status HealthStatus, err error)
	stopCh     chan struct{}
	stoppedCh  chan struct{}
}

// NewHealthPoller creates a new health poller.
func NewHealthPoller(checker *HealthChecker, onChange func(status HealthStatus, err error)) *HealthPoller {
	return &HealthPoller{
		checker:    checker,
		lastStatus: HealthStatusUnknown,
		onChange:   onChange,
		stopCh:     make(chan struct{}),
		stoppedCh:  make(chan struct{}),
	}
}

// Start begins polling for health status changes.
func (p *HealthPoller) Start(ctx context.Context) {
	go p.run(ctx)
}

// Stop stops the health poller.
func (p *HealthPoller) Stop() {
	close(p.stopCh)
	<-p.stoppedCh
}

func (p *HealthPoller) run(ctx context.Context) {
	defer close(p.stoppedCh)

	if p.checker.probe == nil {
		return
	}

	period := time.Duration(p.checker.probe.PeriodSeconds) * time.Second
	if period == 0 {
		period = 10 * time.Second
	}

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			status, err := p.checker.Check(ctx)
			if status != p.lastStatus {
				p.lastStatus = status
				if p.onChange != nil {
					p.onChange(status, err)
				}
			}
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}
