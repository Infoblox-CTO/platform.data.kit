// Package localdev provides utilities for local development environment management.
package localdev

import (
	"context"
	"io"
	"time"
)

// RuntimeType represents the type of local development runtime.
type RuntimeType string

const (
	// RuntimeCompose uses Docker Compose for local development.
	RuntimeCompose RuntimeType = "compose"
	// RuntimeK3d uses k3d (k3s in Docker) for local development.
	RuntimeK3d RuntimeType = "k3d"
)

// RuntimeManager defines the interface for managing local development environments.
// Both ComposeManager and K3dManager implement this interface.
type RuntimeManager interface {
	// Up starts the local development stack.
	// If detach is true, the command returns after starting (background mode).
	// Output is written to the provided writer.
	Up(ctx context.Context, detach bool, output io.Writer) error

	// Down stops the local development stack.
	// If removeVolumes is true, persistent data is also deleted.
	// Output is written to the provided writer.
	Down(ctx context.Context, removeVolumes bool, output io.Writer) error

	// Status returns the current status of the stack.
	Status(ctx context.Context) (*StackStatus, error)

	// WaitForHealthy waits for all services to become healthy.
	// Returns an error if the timeout is exceeded.
	WaitForHealthy(ctx context.Context, timeout time.Duration) error

	// Type returns the runtime type for this manager.
	Type() RuntimeType
}

// PortForwardManager defines the interface for managing port forwards (k3d only).
type PortForwardManager interface {
	// Start begins port forwarding for all configured services.
	Start(ctx context.Context) error

	// Stop terminates all active port forwards.
	Stop() error

	// Status returns the status of all port forwards.
	Status() []PortForwardStatus
}

// PortForwardStatus represents the status of a single port forward.
type PortForwardStatus struct {
	// LocalPort is the port on localhost.
	LocalPort int
	// TargetService is the Kubernetes service being forwarded.
	TargetService string
	// Active indicates whether the port forward is running.
	Active bool
	// Error contains any error message if the forward failed.
	Error string
}

// Ensure ComposeManager implements RuntimeManager at compile time.
var _ RuntimeManager = (*ComposeManager)(nil)
