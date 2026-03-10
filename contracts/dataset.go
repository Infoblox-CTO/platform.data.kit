package contracts

// ---------------------------------------------------------------------------
// DataSet — a named data contract (table, prefix, or topic) in a Store.
// ---------------------------------------------------------------------------

// DataSetManifest represents a data contract: a named piece of data in a Store
// with schema, classification, and optional column-level lineage.
type DataSetManifest struct {
	// APIVersion is the schema version (e.g., "datakit.infoblox.dev/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "DataSet".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains dataset identification information.
	Metadata DataSetMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the dataset specification.
	Spec DataSetSpec `json:"spec" yaml:"spec"`
}

// DataSetMetadata contains identification information for a DataSet.
type DataSetMetadata struct {
	// Name is the logical dataset name (e.g., "users", "orders-parquet").
	Name string `json:"name" yaml:"name"`

	// Namespace is the team namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Version is the semantic version of the data contract (e.g., "1.2.0").
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// DataSetSpec defines the data contract for a named dataset.
type DataSetSpec struct {
	// Store is the name of the Store where this data lives.
	Store string `json:"store" yaml:"store"`

	// Table is the fully-qualified table name (for relational stores).
	Table string `json:"table,omitempty" yaml:"table,omitempty"`

	// Prefix is the object prefix (for object stores like S3).
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`

	// Topic is the topic name (for streaming stores like Kafka).
	Topic string `json:"topic,omitempty" yaml:"topic,omitempty"`

	// Format is the data format (e.g., "parquet", "json", "csv", "avro").
	Format string `json:"format,omitempty" yaml:"format,omitempty"`

	// Classification is the data classification level (e.g., "public", "internal", "confidential", "restricted").
	Classification string `json:"classification,omitempty" yaml:"classification,omitempty"`

	// Schema defines the fields/columns in this dataset.
	Schema []SchemaField `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Dev contains development-only configuration such as seed data.
	// This section is ignored in production deployments.
	Dev *DataSetDevSpec `json:"dev,omitempty" yaml:"dev,omitempty"`
}

// DataSetDevSpec holds development-time configuration for a DataSet.
type DataSetDevSpec struct {
	// Seed defines mock/sample data to load into the store for local development.
	Seed *SeedSpec `json:"seed,omitempty" yaml:"seed,omitempty"`
}

// SeedSpec defines how to populate a DataSet's backing store with sample data.
type SeedSpec struct {
	// Inline rows to insert. Each entry is a map of column→value.
	// These are the "default" seed rows used when no --profile is specified.
	Inline []map[string]any `json:"inline,omitempty" yaml:"inline,omitempty"`

	// File is a path (relative to the package directory) to a CSV or JSON file
	// containing seed rows for the default profile.
	File string `json:"file,omitempty" yaml:"file,omitempty"`

	// Profiles defines named alternative seed data sets. Each profile can
	// supply its own inline rows or file, enabling different test scenarios
	// (e.g., "large-dataset", "edge-cases", "empty").
	// Use `dk dev seed --profile <name>` to activate a specific profile.
	Profiles map[string]*SeedProfileSpec `json:"profiles,omitempty" yaml:"profiles,omitempty"`
}

// SeedProfileSpec defines a named seed data profile.
type SeedProfileSpec struct {
	// Inline rows to insert for this profile.
	Inline []map[string]any `json:"inline,omitempty" yaml:"inline,omitempty"`

	// File is a path (relative to the package directory) to a CSV or JSON file.
	File string `json:"file,omitempty" yaml:"file,omitempty"`
}

// SchemaField defines a single field in a DataSet's schema.
type SchemaField struct {
	// Name is the field/column name.
	Name string `json:"name" yaml:"name"`

	// Type is the data type (e.g., "integer", "string", "timestamp", "boolean", "float").
	Type string `json:"type" yaml:"type"`

	// PII indicates this field contains personally identifiable information.
	PII bool `json:"pii,omitempty" yaml:"pii,omitempty"`

	// From declares the lineage source for this field (e.g., "users.id").
	// This enables column-level lineage tracking.
	From string `json:"from,omitempty" yaml:"from,omitempty"`
}

// --- Manifest interface implementation for DataSetManifest ---

// GetKind returns the manifest kind.
func (a *DataSetManifest) GetKind() Kind { return KindDataSet }

// GetName returns the dataset name.
func (a *DataSetManifest) GetName() string { return a.Metadata.Name }

// GetNamespace returns the dataset namespace.
func (a *DataSetManifest) GetNamespace() string { return a.Metadata.Namespace }

// GetVersion returns the dataset version.
func (a *DataSetManifest) GetVersion() string { return a.Metadata.Version }

// GetDescription returns an empty string.
func (a *DataSetManifest) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (a *DataSetManifest) GetOwner() string { return "" }

// ---------------------------------------------------------------------------
// DataSetGroup — bundles multiple DataSets from a single materialisation.
// ---------------------------------------------------------------------------

// DataSetGroupManifest bundles multiple DataSets produced by a single operation
// (e.g., a CQ sync that snapshots several tables at once).
type DataSetGroupManifest struct {
	// APIVersion is the schema version.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "DataSetGroup".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains identification information.
	Metadata DataSetGroupMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the dataset group specification.
	Spec DataSetGroupSpec `json:"spec" yaml:"spec"`
}

// DataSetGroupMetadata contains identification information for a DataSetGroup.
type DataSetGroupMetadata struct {
	// Name is the group name (e.g., "pg-snapshot").
	Name string `json:"name" yaml:"name"`

	// Namespace is the team namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// DataSetGroupSpec defines the grouped datasets.
type DataSetGroupSpec struct {
	// Store is the common Store for all datasets in the group.
	Store string `json:"store" yaml:"store"`

	// DataSets is the list of DataSet names in this group.
	DataSets []string `json:"datasets" yaml:"datasets"`
}

// --- Manifest interface implementation for DataSetGroupManifest ---

// GetKind returns the manifest kind.
func (ag *DataSetGroupManifest) GetKind() Kind { return KindDataSetGroup }

// GetName returns the dataset group name.
func (ag *DataSetGroupManifest) GetName() string { return ag.Metadata.Name }

// GetNamespace returns the dataset group namespace.
func (ag *DataSetGroupManifest) GetNamespace() string { return ag.Metadata.Namespace }

// GetVersion returns an empty string.
func (ag *DataSetGroupManifest) GetVersion() string { return "" }

// GetDescription returns an empty string.
func (ag *DataSetGroupManifest) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (ag *DataSetGroupManifest) GetOwner() string { return "" }

// ---------------------------------------------------------------------------
// Media type constants for OCI artifact layers.
// ---------------------------------------------------------------------------

const (
	// MediaTypeDKTransform is the OCI media type for Transform manifest layers.
	MediaTypeDKTransform = "application/vnd.dk.transform.v1+yaml"

	// MediaTypeDKDataSet is the OCI media type for DataSet manifest layers.
	MediaTypeDKDataSet = "application/vnd.dk.dataset.v1+yaml"

	// MediaTypeDKConnector is the OCI media type for Connector manifest layers.
	MediaTypeDKConnector = "application/vnd.dk.connector.v1+yaml"

	// MediaTypeDKStore is the OCI media type for Store manifest layers.
	MediaTypeDKStore = "application/vnd.dk.store.v1+yaml"
)
