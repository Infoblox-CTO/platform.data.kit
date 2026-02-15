package contracts

// BindingType represents the type of data binding.
type BindingType string

const (
	// BindingTypeS3Prefix binds to an S3 prefix.
	BindingTypeS3Prefix BindingType = "s3-prefix"

	// BindingTypeKafkaTopic binds to a Kafka topic.
	BindingTypeKafkaTopic BindingType = "kafka-topic"

	// BindingTypePostgresTable binds to a PostgreSQL table.
	BindingTypePostgresTable BindingType = "postgres-table"
)

// Binding represents a data source or sink binding.
type Binding struct {
	// Name is the logical name of this binding.
	Name string `json:"name" yaml:"name"`

	// Asset is the optional asset name this binding is associated with.
	// When present, this binding is scoped to a specific asset instance.
	Asset string `json:"asset,omitempty" yaml:"asset,omitempty"`

	// Type is the binding type.
	Type BindingType `json:"type" yaml:"type"`

	// S3 contains S3-specific binding configuration.
	S3 *S3PrefixBinding `json:"s3,omitempty" yaml:"s3,omitempty"`

	// Kafka contains Kafka-specific binding configuration.
	Kafka *KafkaTopicBinding `json:"kafka,omitempty" yaml:"kafka,omitempty"`

	// Postgres contains PostgreSQL-specific binding configuration.
	Postgres *PostgresTableBinding `json:"postgres,omitempty" yaml:"postgres,omitempty"`
}

// S3PrefixBinding contains S3 binding configuration.
type S3PrefixBinding struct {
	// Bucket is the S3 bucket name.
	Bucket string `json:"bucket" yaml:"bucket"`

	// Prefix is the S3 key prefix.
	Prefix string `json:"prefix,omitempty" yaml:"prefix,omitempty"`

	// Region is the AWS region.
	Region string `json:"region,omitempty" yaml:"region,omitempty"`

	// Endpoint is a custom S3 endpoint (for LocalStack).
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

	// Format is the file format (parquet, json, csv, avro).
	Format string `json:"format,omitempty" yaml:"format,omitempty"`
}

// KafkaTopicBinding contains Kafka binding configuration.
type KafkaTopicBinding struct {
	// Topic is the Kafka topic name.
	Topic string `json:"topic" yaml:"topic"`

	// Brokers is the list of Kafka broker addresses.
	Brokers []string `json:"brokers" yaml:"brokers"`

	// ConsumerGroup is the consumer group ID.
	ConsumerGroup string `json:"consumerGroup,omitempty" yaml:"consumerGroup,omitempty"`

	// SchemaRegistry is the schema registry URL.
	SchemaRegistry string `json:"schemaRegistry,omitempty" yaml:"schemaRegistry,omitempty"`

	// SecurityProtocol is the security protocol (PLAINTEXT, SSL, SASL_SSL).
	SecurityProtocol string `json:"securityProtocol,omitempty" yaml:"securityProtocol,omitempty"`
}

// PostgresTableBinding contains PostgreSQL binding configuration.
type PostgresTableBinding struct {
	// Host is the database host.
	Host string `json:"host,omitempty" yaml:"host,omitempty"`

	// Port is the database port.
	Port int `json:"port,omitempty" yaml:"port,omitempty"`

	// Database is the database name.
	Database string `json:"database" yaml:"database"`

	// Schema is the database schema.
	Schema string `json:"schema,omitempty" yaml:"schema,omitempty"`

	// Table is the table name.
	Table string `json:"table" yaml:"table"`

	// SSLMode is the SSL mode (disable, require, verify-ca, verify-full).
	SSLMode string `json:"sslMode,omitempty" yaml:"sslMode,omitempty"`
}

// BindingsManifest contains multiple bindings.
type BindingsManifest struct {
	// APIVersion is the API version.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Bindings".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains binding metadata.
	Metadata BindingsMetadata `json:"metadata" yaml:"metadata"`

	// Bindings is the list of bindings.
	Bindings []Binding `json:"bindings" yaml:"bindings"`
}

// BindingsMetadata contains metadata for bindings.
type BindingsMetadata struct {
	// Name is the bindings collection name.
	Name string `json:"name" yaml:"name"`

	// Environment is the target environment.
	Environment string `json:"environment,omitempty" yaml:"environment,omitempty"`
}
