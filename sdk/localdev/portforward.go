package localdev

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/charts"
)

// PortForward represents a single port forward configuration.
type PortForward struct {
	// ServiceName is the Kubernetes service to forward to.
	ServiceName string
	// LocalPort is the port on localhost.
	LocalPort int
	// RemotePort is the port on the Kubernetes service.
	RemotePort int
}

// PortForwarder manages kubectl port-forward processes.
type PortForwarder struct {
	kubeContext string
	namespace   string
	forwards    []PortForward
	processes   []*exec.Cmd
	mu          sync.Mutex
}

// NewPortForwarder creates a new port forwarder.
func NewPortForwarder(kubeContext, namespace string) *PortForwarder {
	return &PortForwarder{
		kubeContext: kubeContext,
		namespace:   namespace,
		forwards:    make([]PortForward, 0),
		processes:   make([]*exec.Cmd, 0),
	}
}

// AddForward adds a port forward configuration.
func (p *PortForwarder) AddForward(serviceName string, localPort, remotePort int) {
	p.forwards = append(p.forwards, PortForward{
		ServiceName: serviceName,
		LocalPort:   localPort,
		RemotePort:  remotePort,
	})
}

// AddForwardsFromCharts adds port forwards for all services defined in the
// given chart definitions. This replaces hardcoded AddForward calls.
func (p *PortForwarder) AddForwardsFromCharts(defs []charts.ChartDefinition) {
	for _, def := range defs {
		for _, pf := range def.PortForwards {
			p.AddForward(pf.ServiceName, pf.LocalPort, pf.RemotePort)
		}
	}
}

// Start begins all configured port forwards.
func (p *PortForwarder) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, fwd := range p.forwards {
		cmd := exec.CommandContext(ctx, "kubectl",
			"--context", p.kubeContext,
			"port-forward",
			"-n", p.namespace,
			fmt.Sprintf("svc/%s", fwd.ServiceName),
			fmt.Sprintf("%d:%d", fwd.LocalPort, fwd.RemotePort),
		)

		// Redirect to /dev/null to run silently in background
		cmd.Stdout = nil
		cmd.Stderr = nil

		if err := cmd.Start(); err != nil {
			// Clean up already started processes
			p.stopLocked()
			return fmt.Errorf("failed to start port-forward for %s: %w", fwd.ServiceName, err)
		}

		p.processes = append(p.processes, cmd)
	}

	return nil
}

// Stop terminates all port forward processes.
func (p *PortForwarder) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
}

// stopLocked stops all processes (must be called with lock held).
func (p *PortForwarder) stopLocked() {
	for _, cmd := range p.processes {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
	p.processes = make([]*exec.Cmd, 0)
}

// Status returns the status of all port forwards.
func (p *PortForwarder) Status() []PortForwardStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	statuses := make([]PortForwardStatus, len(p.forwards))
	for i, fwd := range p.forwards {
		statuses[i] = PortForwardStatus{
			LocalPort:     fwd.LocalPort,
			TargetService: fwd.ServiceName,
			Active:        i < len(p.processes) && p.processes[i].Process != nil,
		}
	}

	return statuses
}
