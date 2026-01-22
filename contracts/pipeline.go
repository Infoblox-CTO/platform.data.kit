package contracts

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
