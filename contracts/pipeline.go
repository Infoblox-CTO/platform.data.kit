package contracts

import "fmt"

// PipelineManifest defines the runtime configuration for a pipeline.
type PipelineManifest struct {
	// APIVersion is the API version.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Pipeline".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains pipeline metadata.
	Metadata PipelineMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the pipeline specification.
	Spec PipelineSpec `json:"spec" yaml:"spec"`
}

// PipelineMetadata contains metadata for a pipeline.
type PipelineMetadata struct {
	// Name is the pipeline name.
	Name string `json:"name" yaml:"name"`

	// Labels are key-value pairs for organizing pipelines.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are key-value pairs for additional metadata.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// PipelineSpec contains the pipeline runtime specification.
type PipelineSpec struct {
	// Image is the container image to run.
	Image string `json:"image" yaml:"image"`

	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty" yaml:"command,omitempty"`

	// Args are arguments to pass to the entrypoint.
	Args []string `json:"args,omitempty" yaml:"args,omitempty"`

	// Env contains environment variable definitions.
	Env []EnvVar `json:"env,omitempty" yaml:"env,omitempty"`

	// EnvFrom sources environment variables from external sources.
	EnvFrom []EnvFromSource `json:"envFrom,omitempty" yaml:"envFrom,omitempty"`

	// Replicas is the number of parallel instances.
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`

	// Bindings are references to data bindings.
	Bindings []BindingRef `json:"bindings,omitempty" yaml:"bindings,omitempty"`

	// ServiceAccountName is the Kubernetes service account to use.
	ServiceAccountName string `json:"serviceAccountName,omitempty" yaml:"serviceAccountName,omitempty"`

	// Mode is the pipeline execution mode: batch or streaming.
	Mode PipelineMode `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Timeout is the maximum execution time for batch pipelines (e.g., "30m", "1h").
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Retries is the maximum retry attempts for batch pipelines.
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// BackoffLimit is the Kubernetes Job backoff limit.
	BackoffLimit int `json:"backoffLimit,omitempty" yaml:"backoffLimit,omitempty"`

	// LivenessProbe defines the liveness health check for streaming pipelines.
	LivenessProbe *Probe `json:"livenessProbe,omitempty" yaml:"livenessProbe,omitempty"`

	// ReadinessProbe defines the readiness health check for streaming pipelines.
	ReadinessProbe *Probe `json:"readinessProbe,omitempty" yaml:"readinessProbe,omitempty"`

	// TerminationGracePeriodSeconds is the grace period for streaming pipeline shutdown.
	TerminationGracePeriodSeconds int `json:"terminationGracePeriodSeconds,omitempty" yaml:"terminationGracePeriodSeconds,omitempty"`

	// Lineage configures lineage tracking for the pipeline.
	Lineage *PipelineLineage `json:"lineage,omitempty" yaml:"lineage,omitempty"`
}

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

// PipelineLineage configures lineage tracking for a pipeline.
type PipelineLineage struct {
	// Enabled indicates whether lineage tracking is enabled.
	Enabled bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// HeartbeatInterval is the interval for emitting heartbeat events (streaming only).
	HeartbeatInterval string `json:"heartbeatInterval,omitempty" yaml:"heartbeatInterval,omitempty"`
}

// Validate validates the probe configuration.
func (p *Probe) Validate() error {
	// Count how many probe types are specified
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

	// Validate the specific probe type
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
