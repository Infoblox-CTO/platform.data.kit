// Package catalog provides types for data catalog records.
package catalog

import (
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// RecordType represents the type of catalog record.
type RecordType string

const (
	// RecordTypeDataset represents a dataset record.
	RecordTypeDataset RecordType = "dataset"
	// RecordTypeJob represents a job/pipeline record.
	RecordTypeJob RecordType = "job"
	// RecordTypeSource represents a data source record.
	RecordTypeSource RecordType = "source"
)

// Record represents a data catalog record.
type Record struct {
	// ID is the unique identifier for this record.
	ID string `json:"id"`
	// Type is the type of record.
	Type RecordType `json:"type"`
	// Namespace is the namespace containing this record.
	Namespace string `json:"namespace"`
	// Name is the name of this record.
	Name string `json:"name"`
	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
	// Tags are labels for categorization.
	Tags []string `json:"tags,omitempty"`
	// Owner is the owner of this record.
	Owner string `json:"owner,omitempty"`
	// Classification is the data classification (e.g., "pii", "internal").
	Classification string `json:"classification,omitempty"`
	// Schema defines the structure of the data.
	Schema *Schema `json:"schema,omitempty"`
	// Source describes where the data comes from.
	Source *Source `json:"source,omitempty"`
	// Lineage tracks data lineage information.
	Lineage *Lineage `json:"lineage,omitempty"`
	// Quality contains data quality metrics.
	Quality *Quality `json:"quality,omitempty"`
	// CreatedAt is when this record was created.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is when this record was last updated.
	UpdatedAt time.Time `json:"updatedAt"`
	// Metadata contains additional metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Schema describes the structure of a dataset.
type Schema struct {
	// Fields is the list of fields in the schema.
	Fields []Field `json:"fields"`
	// Format is the data format (e.g., "json", "avro", "parquet").
	Format string `json:"format,omitempty"`
	// Version is the schema version.
	Version string `json:"version,omitempty"`
}

// Field represents a field in a schema.
type Field struct {
	// Name is the field name.
	Name string `json:"name"`
	// Type is the field type.
	Type string `json:"type"`
	// Description describes the field.
	Description string `json:"description,omitempty"`
	// Required indicates if the field is required.
	Required bool `json:"required,omitempty"`
	// Classification is the PII classification.
	Classification string `json:"classification,omitempty"`
}

// Source describes the origin of data.
type Source struct {
	// Name is a human-readable name for the source.
	Name string `json:"name,omitempty"`
	// Type is the source type (e.g., "kafka", "s3", "postgres").
	Type string `json:"type"`
	// Connection contains connection information.
	Connection string `json:"connection,omitempty"`
	// Location is the specific location within the source.
	Location string `json:"location,omitempty"`
}

// Reference identifies a dataset or job by namespace and name.
type Reference struct {
	// Namespace is the namespace.
	Namespace string `json:"namespace"`
	// Name is the name.
	Name string `json:"name"`
}

// Lineage contains data lineage information.
type Lineage struct {
	// Inputs are the upstream datasets (legacy string format).
	Inputs []string `json:"inputs,omitempty"`
	// Outputs are the downstream datasets (legacy string format).
	Outputs []string `json:"outputs,omitempty"`
	// Upstream are the upstream dataset references.
	Upstream []Reference `json:"upstream,omitempty"`
	// Downstream are the downstream dataset references.
	Downstream []Reference `json:"downstream,omitempty"`
	// Job is the job that produces this dataset.
	Job string `json:"job,omitempty"`
}

// Quality contains data quality metrics.
type Quality struct {
	// RowCount is the number of rows.
	RowCount int64 `json:"rowCount,omitempty"`
	// ByteCount is the size in bytes.
	ByteCount int64 `json:"byteCount,omitempty"`
	// NullRate is the percentage of null values.
	NullRate float64 `json:"nullRate,omitempty"`
	// Freshness is when the data was last updated.
	Freshness time.Time `json:"freshness,omitempty"`
}

// NewRecord creates a new catalog record.
func NewRecord(recordType RecordType, namespace, name string) *Record {
	now := time.Now().UTC()
	return &Record{
		ID:        namespace + "/" + name,
		Type:      recordType,
		Namespace: namespace,
		Name:      name,
		Tags:      []string{},
		Metadata:  make(map[string]string),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// FromManifest creates a catalog record from a manifest (Source, Destination, or Model).
func FromManifest(m manifest.Manifest, kind contracts.Kind) *Record {
	now := time.Now().UTC()

	record := &Record{
		ID:          m.GetNamespace() + "/" + m.GetName(),
		Type:        RecordTypeDataset,
		Namespace:   m.GetNamespace(),
		Name:        m.GetName(),
		Description: m.GetDescription(),
		Tags:        []string{},
		Owner:       m.GetOwner(),
		Metadata:    make(map[string]string),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	record.Metadata["version"] = m.GetVersion()
	record.Metadata["kind"] = string(kind)

	return record
}

// WithSchema adds schema information to the record.
func (r *Record) WithSchema(schema *Schema) *Record {
	r.Schema = schema
	return r
}

// WithSource adds source information to the record.
func (r *Record) WithSource(source *Source) *Record {
	r.Source = source
	return r
}

// WithLineage adds lineage information to the record.
func (r *Record) WithLineage(lineage *Lineage) *Record {
	r.Lineage = lineage
	return r
}

// WithQuality adds quality metrics to the record.
func (r *Record) WithQuality(quality *Quality) *Record {
	r.Quality = quality
	return r
}

// AddTag adds a tag to the record.
func (r *Record) AddTag(tag string) *Record {
	r.Tags = append(r.Tags, tag)
	return r
}

// SetMetadata sets a metadata value.
func (r *Record) SetMetadata(key, value string) *Record {
	r.Metadata[key] = value
	return r
}
