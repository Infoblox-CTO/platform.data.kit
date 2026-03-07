package contracts

// Store is a named instance of a Connector — a specific database, bucket, or cluster
// with its connection details and credentials.
// Secrets live ONLY on the Store — never on Assets or Transforms.
// Created by the team that owns the infrastructure.
type Store struct {
	// APIVersion is the schema version (e.g., "datakit.infoblox.dev/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Store".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains store identification information.
	Metadata StoreMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the store specification.
	Spec StoreSpec `json:"spec" yaml:"spec"`
}

// StoreMetadata contains identification information for a Store.
type StoreMetadata struct {
	// Name is the logical store name (e.g., "warehouse", "lake-raw", "events").
	Name string `json:"name" yaml:"name"`

	// Namespace is the team namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// StoreSpec defines the connection details for a specific infrastructure instance.
type StoreSpec struct {
	// Connector is the provider name of the Connector this store is an instance of.
	// This references spec.provider on the Connector (e.g., "postgres", "s3"),
	// NOT the CR metadata.name.
	Connector string `json:"connector" yaml:"connector"`

	// ConnectorVersion is an optional semver range constraining which connector
	// versions this store is compatible with (e.g., "^1.0.0", ">=1.2.0 <2.0.0").
	// When omitted, the highest available version of the named provider is used.
	ConnectorVersion string `json:"connectorVersion,omitempty" yaml:"connectorVersion,omitempty"`

	// Connection holds technology-specific connection parameters (host, port, bucket, etc.).
	Connection map[string]any `json:"connection" yaml:"connection"`

	// Secrets holds credential references using ${VAR} interpolation syntax.
	// These are resolved from environment variables or a secret store at runtime.
	Secrets map[string]string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

// --- Manifest interface implementation for Store ---

// GetKind returns the manifest kind.
func (s *Store) GetKind() Kind { return KindStore }

// GetName returns the store name.
func (s *Store) GetName() string { return s.Metadata.Name }

// GetNamespace returns the store namespace.
func (s *Store) GetNamespace() string { return s.Metadata.Namespace }

// GetVersion returns an empty string (stores are not versioned).
func (s *Store) GetVersion() string { return "" }

// GetDescription returns an empty string.
func (s *Store) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (s *Store) GetOwner() string { return "" }
