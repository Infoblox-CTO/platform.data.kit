package contracts

// Connector describes a storage technology type (e.g., Postgres, S3, Kafka).
// Multiple versions of the same connector (identified by spec.provider) can
// coexist in a cluster — each as a separate CR with a unique metadata.name
// (e.g., "postgres-1-2-0", "postgres-1-3-0").
// Created by the platform team.
type Connector struct {
	// APIVersion is the schema version (e.g., "datakit.infoblox.dev/v1alpha1").
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
	// Name is the unique CR instance name (e.g., "postgres-1-2-0").
	// For local files this can match the provider name (e.g., "postgres")
	// when only one version is present.
	Name string `json:"name" yaml:"name"`

	// Labels are key-value labels for filtering.
	// Convention: datakit.infoblox.dev/provider and datakit.infoblox.dev/channel
	// are used for indexed lookups in k8s.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// ConnectorSpec defines the technology type and its capabilities.
type ConnectorSpec struct {
	// Provider is the logical connector identity that stores reference
	// (e.g., "postgres", "s3", "kafka"). Multiple CRs can share the same
	// provider — one per version. If empty, defaults to Type.
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`

	// Version is the semantic version of this connector release (e.g., "1.2.0").
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Type is the technology identifier (e.g., "postgres", "s3", "kafka", "snowflake").
	Type string `json:"type" yaml:"type"`

	// Protocol is the wire protocol (e.g., "postgresql", "s3", "kafka").
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`

	// Capabilities lists what roles this connector can serve: "source", "destination", or both.
	Capabilities []string `json:"capabilities" yaml:"capabilities"`

	// Plugin holds optional CloudQuery plugin image references.
	Plugin *ConnectorPlugin `json:"plugin,omitempty" yaml:"plugin,omitempty"`

	// Tools lists technology-specific actions this connector exposes
	// (e.g., launch psql, generate DSN, mount S3 bucket).
	Tools []ConnectorTool `json:"tools,omitempty" yaml:"tools,omitempty"`

	// ConnectionSchema declares the structured connection fields this
	// connector expects. Maps logical field names to Store.spec.connection keys.
	ConnectionSchema map[string]ConnectionSchemaField `json:"connectionSchema,omitempty" yaml:"connectionSchema,omitempty"`
}

// ConnectorPlugin holds CQ plugin image references for the CloudQuery runtime.
type ConnectorPlugin struct {
	// Source is the container image for the source plugin.
	Source string `json:"source,omitempty" yaml:"source,omitempty"`

	// Destination is the container image for the destination plugin.
	Destination string `json:"destination,omitempty" yaml:"destination,omitempty"`
}

// ConnectorTool describes a technology-specific action a connector exposes.
type ConnectorTool struct {
	// Name is the tool identifier (e.g., "psql", "dsn", "mount", "ls").
	Name string `json:"name" yaml:"name"`

	// Description is a human-readable summary of what this tool does.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Type is the tool category: "exec" (run a command) or "config" (generate output).
	Type string `json:"type" yaml:"type"`

	// Requires lists binary names that must be on $PATH (e.g., ["psql"], ["aws"]).
	Requires []string `json:"requires,omitempty" yaml:"requires,omitempty"`

	// Command is a Go template for the shell command to execute (type=exec).
	Command string `json:"command,omitempty" yaml:"command,omitempty"`

	// Format is the output format for type=config: "text", "file", or "env".
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Path is the file path to write to (for format=file).
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Mode is how to write the file: "append" or "overwrite" (for format=file).
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Template is a Go template for the output content (type=config).
	Template string `json:"template,omitempty" yaml:"template,omitempty"`

	// PostMessage is a Go template rendered and displayed after tool execution.
	PostMessage string `json:"postMessage,omitempty" yaml:"postMessage,omitempty"`

	// Default marks this as the default tool when none is specified.
	Default bool `json:"default,omitempty" yaml:"default,omitempty"`
}

// ConnectionSchemaField describes a single connection parameter a connector expects.
type ConnectionSchemaField struct {
	// Description is a human-readable explanation of the field.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Field is the key name in Store.spec.connection that this maps to.
	Field string `json:"field" yaml:"field"`

	// Default is the fallback value when the field is absent from the store.
	Default string `json:"default,omitempty" yaml:"default,omitempty"`

	// Secret indicates this field may also be fulfilled from Store.spec.secrets.
	Secret bool `json:"secret,omitempty" yaml:"secret,omitempty"`

	// Optional indicates the field is not required.
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// GetProvider returns the logical provider identity for this connector.
// Falls back to Type if Provider is not explicitly set.
func (c *Connector) GetProvider() string {
	if c.Spec.Provider != "" {
		return c.Spec.Provider
	}
	return c.Spec.Type
}

// --- Manifest interface implementation for Connector ---

// GetKind returns the manifest kind.
func (c *Connector) GetKind() Kind { return KindConnector }

// GetName returns the CR instance name.
func (c *Connector) GetName() string { return c.Metadata.Name }

// GetNamespace returns an empty string (connectors are cluster-scoped).
func (c *Connector) GetNamespace() string { return "" }

// GetVersion returns the connector's semantic version.
func (c *Connector) GetVersion() string { return c.Spec.Version }

// GetDescription returns an empty string.
func (c *Connector) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (c *Connector) GetOwner() string { return "" }
