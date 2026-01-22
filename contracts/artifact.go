package contracts

// ArtifactContract describes what a package produces or consumes.
type ArtifactContract struct {
	// Name is the artifact identifier within the package
	Name string `json:"name" yaml:"name"`

	// Type is the artifact type
	Type ArtifactType `json:"type" yaml:"type"`

	// Binding is the abstract binding reference
	Binding string `json:"binding" yaml:"binding"`

	// Schema describes the data schema
	Schema *SchemaSpec `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Classification contains data classification metadata
	Classification *Classification `json:"classification,omitempty" yaml:"classification,omitempty"`
}

// ArtifactType represents the type of artifact.
type ArtifactType string

// Artifact type constants.
const (
	ArtifactTypeS3Prefix      ArtifactType = "s3-prefix"
	ArtifactTypeKafkaTopic    ArtifactType = "kafka-topic"
	ArtifactTypePostgresTable ArtifactType = "postgres-table"
	ArtifactTypeSparkJob      ArtifactType = "spark-job"
)

// ValidArtifactTypes returns all valid artifact types.
func ValidArtifactTypes() []ArtifactType {
	return []ArtifactType{
		ArtifactTypeS3Prefix,
		ArtifactTypeKafkaTopic,
		ArtifactTypePostgresTable,
		ArtifactTypeSparkJob,
	}
}

// IsValid checks if the artifact type is valid.
func (t ArtifactType) IsValid() bool {
	for _, valid := range ValidArtifactTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// SchemaSpec describes the data schema for an artifact.
type SchemaSpec struct {
	// Type is the schema type: "parquet", "avro", "json", "csv"
	Type SchemaType `json:"type,omitempty" yaml:"type,omitempty"`

	// SchemaRef is a path to an external schema file
	SchemaRef string `json:"schemaRef,omitempty" yaml:"schemaRef,omitempty"`

	// Inline is an inline schema definition
	Inline map[string]any `json:"inline,omitempty" yaml:"inline,omitempty"`
}

// SchemaType represents the type of schema.
type SchemaType string

// Schema type constants.
const (
	SchemaTypeParquet SchemaType = "parquet"
	SchemaTypeAvro    SchemaType = "avro"
	SchemaTypeJSON    SchemaType = "json"
	SchemaTypeCSV     SchemaType = "csv"
)

// ValidSchemaTypes returns all valid schema types.
func ValidSchemaTypes() []SchemaType {
	return []SchemaType{
		SchemaTypeParquet,
		SchemaTypeAvro,
		SchemaTypeJSON,
		SchemaTypeCSV,
	}
}

// IsValid checks if the schema type is valid.
func (t SchemaType) IsValid() bool {
	for _, valid := range ValidSchemaTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// Classification contains data classification metadata for governance.
type Classification struct {
	// PII indicates if the artifact contains personally identifiable information
	PII bool `json:"pii" yaml:"pii"`

	// Sensitivity is the data sensitivity level
	Sensitivity Sensitivity `json:"sensitivity,omitempty" yaml:"sensitivity,omitempty"`

	// DataCategory is the business domain category
	DataCategory string `json:"dataCategory,omitempty" yaml:"dataCategory,omitempty"`

	// RetentionDays is the data retention period in days
	RetentionDays int `json:"retentionDays,omitempty" yaml:"retentionDays,omitempty"`
}

// Sensitivity represents the data sensitivity level.
type Sensitivity string

// Sensitivity level constants.
const (
	SensitivityPublic       Sensitivity = "public"
	SensitivityInternal     Sensitivity = "internal"
	SensitivityConfidential Sensitivity = "confidential"
	SensitivityRestricted   Sensitivity = "restricted"
)

// ValidSensitivities returns all valid sensitivity levels.
func ValidSensitivities() []Sensitivity {
	return []Sensitivity{
		SensitivityPublic,
		SensitivityInternal,
		SensitivityConfidential,
		SensitivityRestricted,
	}
}

// IsValid checks if the sensitivity level is valid.
func (s Sensitivity) IsValid() bool {
	for _, valid := range ValidSensitivities() {
		if s == valid {
			return true
		}
	}
	return false
}
