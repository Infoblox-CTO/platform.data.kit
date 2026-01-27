package contracts
// Package contracts defines the RuntimeManager interface for local development environments.
// This file is a design contract - the actual implementation will be in sdk/localdev/.
package contracts

import (
	"context"
	"io"
	"time"
)

// RuntimeType represents the type of local development runtime.
















































































}	Error string	// Error contains any error message if the forward failed.	Active bool	// Active indicates whether the port forward is running.	TargetService string	// TargetService is the Kubernetes service being forwarded.	LocalPort int	// LocalPort is the port on localhost.type PortForwardStatus struct {// PortForwardStatus represents the status of a single port forward.}	Status() []PortForwardStatus	// Status returns the status of all port forwards.	Stop() error	// Stop terminates all active port forwards.	Start(ctx context.Context) error	// Start begins port forwarding for all configured services.type PortForwardManager interface {// PortForwardManager defines the interface for managing port forwards (k3d only).}	Type() RuntimeType	// Type returns the runtime type for this manager.	WaitForHealthy(ctx context.Context, timeout time.Duration) error	// Returns an error if the timeout is exceeded.	// WaitForHealthy waits for all services to become healthy.	Status(ctx context.Context) (*StackStatus, error)	// Status returns the current status of the stack.	Down(ctx context.Context, removeVolumes bool, output io.Writer) error	// Output is written to the provided writer.	// If removeVolumes is true, persistent data is also deleted.	// Down stops the local development stack.	Up(ctx context.Context, detach bool, output io.Writer) error	// Output is written to the provided writer.	// If detach is true, the command returns after starting (background mode).	// Up starts the local development stack.type RuntimeManager interface {// Both ComposeManager and K3dManager implement this interface.// RuntimeManager defines the interface for managing local development environments.}	Errors []string	// Errors contains any error messages from the stack.	Services []ServiceStatus	// Services contains the status of each service.	Runtime RuntimeType	// Runtime indicates which runtime is in use ("compose" or "k3d").	Running bool	// Running indicates whether the stack is operational.type StackStatus struct {// StackStatus represents the aggregate status of the local development stack.}	Ports []string	// Ports lists the port mappings for this service.	Health string	// Health indicates if the service is healthy (e.g., "healthy", "unhealthy", "unknown").	Status string	// Status is the current state (e.g., "running", "stopped", "pending").	Name string	// Name is the service identifier (e.g., "redpanda", "localstack", "postgres").type ServiceStatus struct {// ServiceStatus represents the status of a single service in the local dev stack.)	RuntimeK3d RuntimeType = "k3d"	// RuntimeK3d uses k3d (k3s in Docker) for local development.	RuntimeCompose RuntimeType = "compose"	// RuntimeCompose uses Docker Compose for local development.const (type RuntimeType string