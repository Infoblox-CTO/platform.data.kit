package contracts

// Connector describes a storage technology type (e.g., Postgres, S3, Kafka).
// Created by the platform team. Rarely changes.
type Connector struct {
	// APIVersion is the schema version (e.g., "data.infoblox.com/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Connector".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains connector identification information.
	Metadata ConnectorMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the connector specification.
	Spec ConnectorSpec `json:"spec" yaml:"spec"`
}

// ConnectorMetadata contains identification information for a Connector.
type ConnectorMetadata struct {
	// Name is the connector name (e.g., "postgres", "s3", "kafka").
	Name string `json:"name" yaml:"name"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// ConnectorSpec defines the technology type and its capabilities.
type ConnectorSpec struct {
	// Type is the technology identifier (e.g., "postgres", "s3", "kafka", "snowflake").
	Type string `json:"type" yaml:"type"`

	// Protocol is the wire protocol (e.g., "postgresql", "s3", "kafka").
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`

	// Capabilities lists what roles this connector can serve: "source", "destination", or both.
	Capabilities []string `json:"capabilities" yaml:"capabilities"`

	// Plugin holds optional CloudQuery plugin image references.
	Plugin *ConnectorPlugin `json:"plugin,omitempty" yaml:"plugin,omitempty"`
}

// ConnectorPlugin holds CQ plugin image references for the CloudQuery runtime.
type ConnectorPlugin struct {
	// Source is the container image for the source plugin.
	Source string `json:"source,omitempty" yaml:"source,omitempty"`

	// Destination is the container image for the destination plugin.
	Destination string `json:"destination,omitempty" yaml:"destination,omitempty"`
}

// --- Manifest interface implementation for Connector ---

// GetKind returns the manifest kind.
func (c *Connector) GetKind() Kind { return KindConnector }

// GetName returns the connector name.
func (c *Connector) GetName() string { return c.Metadata.Name }

// GetNamespace returns an empty string (connectors are cluster-scoped).
func (c *Connector) GetNamespace() string { return "" }

// GetVersion returns an empty string (connectors are not versioned individually).
func (c *Connector) GetVersion() string { return "" }

// GetDescription returns an empty string.
func (c *Connector) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (c *Connector) GetOwner() string { return "" }
