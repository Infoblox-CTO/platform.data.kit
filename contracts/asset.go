package contracts

// ---------------------------------------------------------------------------
// Asset — a named data contract (table, prefix, or topic) in a Store.
// ---------------------------------------------------------------------------

// AssetManifest represents a data contract: a named piece of data in a Store
// with schema, classification, and optional column-level lineage.
type AssetManifest struct {
	// APIVersion is the schema version (e.g., "data.infoblox.com/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Asset".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains asset identification information.
	Metadata AssetMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the asset specification.
	Spec AssetSpec `json:"spec" yaml:"spec"`
}

// AssetMetadata contains identification information for an Asset.
type AssetMetadata struct {
	// Name is the logical asset name (e.g., "users", "orders-parquet").
	Name string `json:"name" yaml:"name"`

	// Namespace is the team namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`

	// Annotations are arbitrary annotations.
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// AssetSpec defines the data contract for a named dataset.
type AssetSpec struct {
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
}

// SchemaField defines a single field in an Asset's schema.
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

// --- Manifest interface implementation for AssetManifest ---

// GetKind returns the manifest kind.
func (a *AssetManifest) GetKind() Kind { return KindAsset }

// GetName returns the asset name.
func (a *AssetManifest) GetName() string { return a.Metadata.Name }

// GetNamespace returns the asset namespace.
func (a *AssetManifest) GetNamespace() string { return a.Metadata.Namespace }

// GetVersion returns an empty string (assets are not versioned individually).
func (a *AssetManifest) GetVersion() string { return "" }

// GetDescription returns an empty string.
func (a *AssetManifest) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (a *AssetManifest) GetOwner() string { return "" }

// ---------------------------------------------------------------------------
// AssetGroup — bundles multiple Assets from a single materialisation.
// ---------------------------------------------------------------------------

// AssetGroupManifest bundles multiple Assets produced by a single operation
// (e.g., a CQ sync that snapshots several tables at once).
type AssetGroupManifest struct {
	// APIVersion is the schema version.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "AssetGroup".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains identification information.
	Metadata AssetGroupMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the asset group specification.
	Spec AssetGroupSpec `json:"spec" yaml:"spec"`
}

// AssetGroupMetadata contains identification information for an AssetGroup.
type AssetGroupMetadata struct {
	// Name is the group name (e.g., "pg-snapshot").
	Name string `json:"name" yaml:"name"`

	// Namespace is the team namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Labels are key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// AssetGroupSpec defines the grouped assets.
type AssetGroupSpec struct {
	// Store is the common Store for all assets in the group.
	Store string `json:"store" yaml:"store"`

	// Assets is the list of Asset names in this group.
	Assets []string `json:"assets" yaml:"assets"`
}

// --- Manifest interface implementation for AssetGroupManifest ---

// GetKind returns the manifest kind.
func (ag *AssetGroupManifest) GetKind() Kind { return KindAssetGroup }

// GetName returns the asset group name.
func (ag *AssetGroupManifest) GetName() string { return ag.Metadata.Name }

// GetNamespace returns the asset group namespace.
func (ag *AssetGroupManifest) GetNamespace() string { return ag.Metadata.Namespace }

// GetVersion returns an empty string.
func (ag *AssetGroupManifest) GetVersion() string { return "" }

// GetDescription returns an empty string.
func (ag *AssetGroupManifest) GetDescription() string { return "" }

// GetOwner returns an empty string.
func (ag *AssetGroupManifest) GetOwner() string { return "" }

// ---------------------------------------------------------------------------
// Media type constants for OCI artifact layers.
// ---------------------------------------------------------------------------

const (
	// MediaTypeDPTransform is the OCI media type for Transform manifest layers.
	MediaTypeDPTransform = "application/vnd.dp.transform.v1+yaml"

	// MediaTypeDPAsset is the OCI media type for Asset manifest layers.
	MediaTypeDPAsset = "application/vnd.dp.asset.v1+yaml"

	// MediaTypeDPConnector is the OCI media type for Connector manifest layers.
	MediaTypeDPConnector = "application/vnd.dp.connector.v1+yaml"

	// MediaTypeDPStore is the OCI media type for Store manifest layers.
	MediaTypeDPStore = "application/vnd.dp.store.v1+yaml"
)
