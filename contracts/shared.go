package contracts

import "fmt"

// --------------------------------------------------------------------------
// Shared Kubernetes-compatible types used across manifest kinds.
// --------------------------------------------------------------------------

// EnvVar represents an environment variable.
type EnvVar struct {
	// Name is the environment variable name.
	Name string `json:"name" yaml:"name"`

	// Value is a static value.
	Value string `json:"value,omitempty" yaml:"value,omitempty"`

	// ValueFrom references an external value source.
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty" yaml:"valueFrom,omitempty"`
}

// EnvVarSource specifies a source for an environment variable value.
type EnvVarSource struct {
	// SecretKeyRef references a secret key.
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty" yaml:"secretKeyRef,omitempty"`

	// ConfigMapKeyRef references a configmap key.
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty" yaml:"configMapKeyRef,omitempty"`

	// BindingRef references a binding property.
	BindingRef *BindingPropertyRef `json:"bindingRef,omitempty" yaml:"bindingRef,omitempty"`
}

// SecretKeySelector selects a key from a Kubernetes secret.
type SecretKeySelector struct {
	Name string `json:"name" yaml:"name"`
	Key  string `json:"key" yaml:"key"`
}

// ConfigMapKeySelector selects a key from a Kubernetes configmap.
type ConfigMapKeySelector struct {
	Name string `json:"name" yaml:"name"`
	Key  string `json:"key" yaml:"key"`
}

// BindingPropertyRef references a property from a binding.
type BindingPropertyRef struct {
	Name     string `json:"name" yaml:"name"`
	Property string `json:"property" yaml:"property"`
}

// EnvFromSource specifies a source for multiple environment variables.
type EnvFromSource struct {
	// Prefix to add to all variable names.
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`

	// SecretRef references a secret.
	SecretRef *SecretRef `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`

	// ConfigMapRef references a configmap.
	ConfigMapRef *ConfigMapRef `json:"configMapRef,omitempty" yaml:"configMapRef,omitempty"`
}

// SecretRef references a Kubernetes secret.
type SecretRef struct {
	Name string `json:"name" yaml:"name"`
}

// ConfigMapRef references a Kubernetes configmap.
type ConfigMapRef struct {
	Name string `json:"name" yaml:"name"`
}

// BindingRef references a binding.
type BindingRef struct {
	Name string `json:"name" yaml:"name"`
	Ref  string `json:"ref" yaml:"ref"`
}

// --------------------------------------------------------------------------
// Health-check probes (Kubernetes-compatible).
// --------------------------------------------------------------------------

// Probe defines a health check probe configuration (Kubernetes-compatible).
type Probe struct {
	// HTTPGet specifies an HTTP probe.
	HTTPGet *HTTPGetAction `json:"httpGet,omitempty" yaml:"httpGet,omitempty"`

	// Exec specifies a command-based probe.
	Exec *ExecAction `json:"exec,omitempty" yaml:"exec,omitempty"`

	// TCPSocket specifies a TCP port probe.
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty" yaml:"tcpSocket,omitempty"`

	// InitialDelaySeconds is the delay before starting probes.
	InitialDelaySeconds int `json:"initialDelaySeconds,omitempty" yaml:"initialDelaySeconds,omitempty"`

	// PeriodSeconds is the interval between probes.
	PeriodSeconds int `json:"periodSeconds,omitempty" yaml:"periodSeconds,omitempty"`

	// TimeoutSeconds is the probe timeout.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`

	// SuccessThreshold is the consecutive successes required.
	SuccessThreshold int `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty"`

	// FailureThreshold is the consecutive failures before unhealthy.
	FailureThreshold int `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty"`
}

// HTTPGetAction describes an HTTP probe action.
type HTTPGetAction struct {
	// Path is the HTTP path to probe.
	Path string `json:"path" yaml:"path"`

	// Port is the port to probe.
	Port int `json:"port" yaml:"port"`

	// Scheme is the protocol scheme (HTTP or HTTPS).
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`

	// Host is the hostname (defaults to pod IP).
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
}

// ExecAction describes a command-based probe action.
type ExecAction struct {
	// Command is the command to execute.
	Command []string `json:"command" yaml:"command"`
}

// TCPSocketAction describes a TCP port probe action.
type TCPSocketAction struct {
	// Port is the TCP port to probe.
	Port int `json:"port" yaml:"port"`

	// Host is the hostname (defaults to pod IP).
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
}

// Validate validates the probe configuration.
func (p *Probe) Validate() error {
	count := 0
	if p.HTTPGet != nil {
		count++
	}
	if p.Exec != nil {
		count++
	}
	if p.TCPSocket != nil {
		count++
	}

	if count == 0 {
		return fmt.Errorf("probe must specify exactly one of httpGet, exec, or tcpSocket")
	}
	if count > 1 {
		return fmt.Errorf("probe must specify exactly one of httpGet, exec, or tcpSocket, got %d", count)
	}

	if p.HTTPGet != nil {
		if err := p.HTTPGet.Validate(); err != nil {
			return fmt.Errorf("httpGet: %w", err)
		}
	}
	if p.Exec != nil {
		if err := p.Exec.Validate(); err != nil {
			return fmt.Errorf("exec: %w", err)
		}
	}
	if p.TCPSocket != nil {
		if err := p.TCPSocket.Validate(); err != nil {
			return fmt.Errorf("tcpSocket: %w", err)
		}
	}

	return nil
}

// Validate validates the HTTP GET action.
func (h *HTTPGetAction) Validate() error {
	if h.Path == "" {
		return fmt.Errorf("path is required")
	}
	if h.Port <= 0 || h.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", h.Port)
	}
	return nil
}

// Validate validates the exec action.
func (e *ExecAction) Validate() error {
	if len(e.Command) == 0 {
		return fmt.Errorf("command is required")
	}
	return nil
}

// Validate validates the TCP socket action.
func (t *TCPSocketAction) Validate() error {
	if t.Port <= 0 || t.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", t.Port)
	}
	return nil
}

// --------------------------------------------------------------------------
// Pipeline lineage (used by pipeline workflow and runner).
// --------------------------------------------------------------------------

// PipelineLineage configures lineage tracking for a pipeline.
type PipelineLineage struct {
	// Enabled indicates whether lineage tracking is enabled.
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// HeartbeatInterval is the interval for emitting heartbeat events (streaming only).
	HeartbeatInterval string `json:"heartbeatInterval,omitempty" yaml:"heartbeatInterval,omitempty"`
}

// --------------------------------------------------------------------------
// Spec sub-types shared across manifest kinds.
// --------------------------------------------------------------------------

// ScheduleSpec defines the scheduling configuration for a package.
type ScheduleSpec struct {
	// Cron is a cron expression for scheduling (e.g., "0 */6 * * *")
	Cron string `json:"cron,omitempty" yaml:"cron,omitempty"`

	// Timezone is the timezone for the cron schedule
	Timezone string `json:"timezone,omitempty" yaml:"timezone,omitempty"`

	// Suspend indicates if scheduling is suspended
	Suspend bool `json:"suspend,omitempty" yaml:"suspend,omitempty"`
}

// ResourceSpec defines resource requirements for a package.
type ResourceSpec struct {
	// CPU is the CPU request/limit (e.g., "2", "500m")
	CPU string `json:"cpu,omitempty" yaml:"cpu,omitempty"`

	// Memory is the memory request/limit (e.g., "4Gi", "512Mi")
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`

	// EphemeralStorage is the ephemeral storage request/limit
	EphemeralStorage string `json:"ephemeralStorage,omitempty" yaml:"ephemeralStorage,omitempty"`
}

// LineageSpec defines the lineage tracking configuration for a package.
type LineageSpec struct {
	// Enabled indicates if lineage tracking is enabled
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// Emitter is the lineage emitter type (e.g., "marquez", "openlineage")
	Emitter string `json:"emitter,omitempty" yaml:"emitter,omitempty"`

	// Namespace is the lineage namespace
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}
