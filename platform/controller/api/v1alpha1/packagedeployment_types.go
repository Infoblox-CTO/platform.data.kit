// Package v1alpha1 contains API Schema definitions for the dp v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PackageDeploymentSpec defines the desired state of PackageDeployment.
type PackageDeploymentSpec struct {
	// Package contains the package reference information.
	Package PackageRef `json:"package"`

	// Mode specifies the execution mode: batch or streaming.
	// +kubebuilder:validation:Enum=batch;streaming
	// +kubebuilder:default=batch
	// +optional
	Mode PipelineMode `json:"mode,omitempty"`

	// Schedule defines when the package should run (batch mode only).
	// +optional
	Schedule *ScheduleSpec `json:"schedule,omitempty"`

	// Replicas is the number of replicas (streaming mode only).
	// +kubebuilder:default=1
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Timeout is the maximum run duration (batch mode only).
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// LivenessProbe configures the liveness probe (streaming mode only).
	// +optional
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`

	// ReadinessProbe configures the readiness probe (streaming mode only).
	// +optional
	ReadinessProbe *Probe `json:"readinessProbe,omitempty"`

	// TerminationGracePeriodSeconds is the grace period for shutdown (streaming mode only).
	// +kubebuilder:default=30
	// +optional
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty"`

	// Resources defines resource requirements for the package run.
	// +optional
	Resources *ResourceSpec `json:"resources,omitempty"`

	// ServiceAccountName is the ServiceAccount to use for runs.
	// +kubebuilder:default=default
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// ImagePullSecrets are the secrets to use for pulling images.
	// +optional
	ImagePullSecrets []ImagePullSecret `json:"imagePullSecrets,omitempty"`
}

// PipelineMode defines the execution mode for a pipeline.
type PipelineMode string

const (
	// PipelineModeBatch runs the pipeline to completion.
	PipelineModeBatch PipelineMode = "batch"
	// PipelineModeStreaming runs the pipeline indefinitely.
	PipelineModeStreaming PipelineMode = "streaming"
)

// Probe describes a health check probe.
type Probe struct {
	// HTTPGet specifies an HTTP GET probe.
	// +optional
	HTTPGet *HTTPGetAction `json:"httpGet,omitempty"`

	// Exec specifies a command execution probe.
	// +optional
	Exec *ExecAction `json:"exec,omitempty"`

	// TCPSocket specifies a TCP socket probe.
	// +optional
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty"`

	// InitialDelaySeconds is the delay before the first probe.
	// +kubebuilder:default=0
	// +optional
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`

	// PeriodSeconds is the probe interval.
	// +kubebuilder:default=10
	// +optional
	PeriodSeconds int32 `json:"periodSeconds,omitempty"`

	// TimeoutSeconds is the probe timeout.
	// +kubebuilder:default=1
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// SuccessThreshold is the consecutive successes required.
	// +kubebuilder:default=1
	// +optional
	SuccessThreshold int32 `json:"successThreshold,omitempty"`

	// FailureThreshold is the consecutive failures required.
	// +kubebuilder:default=3
	// +optional
	FailureThreshold int32 `json:"failureThreshold,omitempty"`
}

// HTTPGetAction describes an HTTP GET probe.
type HTTPGetAction struct {
	// Path is the HTTP path to probe.
	Path string `json:"path"`

	// Port is the port to probe.
	Port int32 `json:"port"`

	// Scheme is HTTP or HTTPS.
	// +kubebuilder:default=HTTP
	// +optional
	Scheme string `json:"scheme,omitempty"`
}

// ExecAction describes a command execution probe.
type ExecAction struct {
	// Command is the command to execute.
	Command []string `json:"command"`
}

// TCPSocketAction describes a TCP socket probe.
type TCPSocketAction struct {
	// Port is the port to probe.
	Port int32 `json:"port"`
}

// PackageRef contains the package reference.
type PackageRef struct {
	// Name is the name of the data package.
	Name string `json:"name"`

	// Namespace is the team/namespace of the package.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Version is the semantic version of the package.
	Version string `json:"version"`

	// Registry is the OCI registry URL.
	Registry string `json:"registry"`

	// Digest is the content digest for verification.
	// +optional
	Digest string `json:"digest,omitempty"`
}

// ScheduleSpec defines scheduling for the package.
type ScheduleSpec struct {
	// Cron is the cron expression for scheduled runs.
	// +optional
	Cron string `json:"cron,omitempty"`

	// Timezone is the timezone for the cron schedule.
	// +kubebuilder:default=UTC
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// Suspend suspends scheduled runs.
	// +optional
	Suspend bool `json:"suspend,omitempty"`
}

// ResourceSpec defines resource requirements.
type ResourceSpec struct {
	// Requests are resource requests.
	// +optional
	Requests ResourceList `json:"requests,omitempty"`

	// Limits are resource limits.
	// +optional
	Limits ResourceList `json:"limits,omitempty"`
}

// ResourceList contains resource quantities.
type ResourceList struct {
	// CPU is the CPU request/limit.
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory is the memory request/limit.
	// +optional
	Memory string `json:"memory,omitempty"`
}

// ImagePullSecret is a reference to an image pull secret.
type ImagePullSecret struct {
	// Name is the secret name.
	Name string `json:"name"`
}

// PackageDeploymentStatus defines the observed state of PackageDeployment.
type PackageDeploymentStatus struct {
	// Phase is the current phase of the deployment.
	// +kubebuilder:validation:Enum=Pending;Pulling;Ready;Running;Failed
	// +optional
	Phase DeploymentPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastRun contains information about the last run.
	// +optional
	LastRun *RunStatus `json:"lastRun,omitempty"`

	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// DeploymentPhase represents the phase of a deployment.
type DeploymentPhase string

const (
	// PhasePending means the deployment is pending.
	PhasePending DeploymentPhase = "Pending"
	// PhasePulling means the package is being pulled.
	PhasePulling DeploymentPhase = "Pulling"
	// PhaseReady means the package is ready to run.
	PhaseReady DeploymentPhase = "Ready"
	// PhaseRunning means the package is currently running.
	PhaseRunning DeploymentPhase = "Running"
	// PhaseFailed means the deployment has failed.
	PhaseFailed DeploymentPhase = "Failed"
)

// RunStatus contains status information about a run.
type RunStatus struct {
	// ID is the unique identifier for this run.
	// +optional
	ID string `json:"id,omitempty"`

	// StartTime is when the run started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the run completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Status is the run status.
	// +optional
	Status string `json:"status,omitempty"`

	// RecordsProcessed is the number of records processed.
	// +optional
	RecordsProcessed int64 `json:"recordsProcessed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Package",type=string,JSONPath=`.spec.package.name`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.package.version`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PackageDeployment is the Schema for the packagedeployments API.
type PackageDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PackageDeploymentSpec   `json:"spec,omitempty"`
	Status PackageDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PackageDeploymentList contains a list of PackageDeployment.
type PackageDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PackageDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PackageDeployment{}, &PackageDeploymentList{})
}
