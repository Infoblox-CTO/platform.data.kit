package manifest

import (
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestBindingsFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		wantCount int
	}{
		{
			name: "valid s3 binding",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings:
  - name: data-bucket
    type: s3-prefix
    s3:
      bucket: my-bucket
      prefix: data/
      region: us-west-2
`),
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "valid kafka binding",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings:
  - name: events-topic
    type: kafka-topic
    kafka:
      topic: events
      brokers:
        - broker1:9092
        - broker2:9092
`),
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "valid postgres binding",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings:
  - name: users-table
    type: postgres-table
    postgres:
      table: users
      database: app_db
      schema: public
`),
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "multiple bindings",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings:
  - name: input-bucket
    type: s3-prefix
    s3:
      bucket: input
  - name: output-bucket
    type: s3-prefix
    s3:
      bucket: output
  - name: events
    type: kafka-topic
    kafka:
      topic: events
`),
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "empty bindings",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
bindings: []
`),
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:    "invalid YAML returns error",
			data:    []byte("not valid yaml ["),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindings, err := BindingsFromBytes(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("BindingsFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(bindings) != tt.wantCount {
				t.Errorf("count = %v, want %v", len(bindings), tt.wantCount)
			}
		})
	}
}

func TestBindingsToBytes(t *testing.T) {
	tests := []struct {
		name     string
		bindings []contracts.Binding
		wantErr  bool
	}{
		{
			name: "s3 binding",
			bindings: []contracts.Binding{
				{
					Name: "data-bucket",
					Type: contracts.BindingTypeS3Prefix,
					S3: &contracts.S3PrefixBinding{
						Bucket: "my-bucket",
						Prefix: "data/",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "kafka binding",
			bindings: []contracts.Binding{
				{
					Name: "events",
					Type: contracts.BindingTypeKafkaTopic,
					Kafka: &contracts.KafkaTopicBinding{
						Topic:   "events",
						Brokers: []string{"broker1:9092"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "empty bindings",
			bindings: []contracts.Binding{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := BindingsToBytes(tt.bindings)
			if (err != nil) != tt.wantErr {
				t.Errorf("BindingsToBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("BindingsToBytes() returned empty data")
			}
		})
	}
}

func TestBindings_RoundTrip(t *testing.T) {
	original := []contracts.Binding{
		{
			Name: "input-bucket",
			Type: contracts.BindingTypeS3Prefix,
			S3: &contracts.S3PrefixBinding{
				Bucket:   "my-input",
				Prefix:   "data/",
				Region:   "us-east-1",
				Endpoint: "https://s3.amazonaws.com",
			},
		},
		{
			Name: "events-topic",
			Type: contracts.BindingTypeKafkaTopic,
			Kafka: &contracts.KafkaTopicBinding{
				Topic:          "events",
				Brokers:        []string{"broker1:9092", "broker2:9092"},
				SchemaRegistry: "http://schema-registry:8081",
			},
		},
	}

	// Serialize to YAML
	data, err := BindingsToBytes(original)
	if err != nil {
		t.Fatalf("BindingsToBytes() error = %v", err)
	}

	// Parse back
	parsed, err := BindingsFromBytes(data)
	if err != nil {
		t.Fatalf("BindingsFromBytes() error = %v", err)
	}

	// Verify count
	if len(parsed) != len(original) {
		t.Errorf("count = %v, want %v", len(parsed), len(original))
	}

	// Verify S3 binding
	if parsed[0].Name != "input-bucket" {
		t.Errorf("binding[0].name = %v, want input-bucket", parsed[0].Name)
	}
	if parsed[0].S3.Bucket != "my-input" {
		t.Errorf("binding[0].s3.bucket = %v, want my-input", parsed[0].S3.Bucket)
	}

	// Verify Kafka binding
	if parsed[1].Name != "events-topic" {
		t.Errorf("binding[1].name = %v, want events-topic", parsed[1].Name)
	}
	if parsed[1].Kafka.Topic != "events" {
		t.Errorf("binding[1].kafka.topic = %v, want events", parsed[1].Kafka.Topic)
	}
}

func TestGetBinding(t *testing.T) {
	bindings := []contracts.Binding{
		{Name: "bucket-a", Type: contracts.BindingTypeS3Prefix},
		{Name: "bucket-b", Type: contracts.BindingTypeS3Prefix},
		{Name: "topic-c", Type: contracts.BindingTypeKafkaTopic},
	}

	tests := []struct {
		name     string
		find     string
		wantErr  bool
		wantName string
	}{
		{
			name:     "find first binding",
			find:     "bucket-a",
			wantErr:  false,
			wantName: "bucket-a",
		},
		{
			name:     "find middle binding",
			find:     "bucket-b",
			wantErr:  false,
			wantName: "bucket-b",
		},
		{
			name:     "find last binding",
			find:     "topic-c",
			wantErr:  false,
			wantName: "topic-c",
		},
		{
			name:    "binding not found",
			find:    "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding, err := GetBinding(bindings, tt.find)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBinding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && binding.Name != tt.wantName {
				t.Errorf("name = %v, want %v", binding.Name, tt.wantName)
			}
		})
	}
}

func TestGetBindingProperty(t *testing.T) {
	s3Binding := &contracts.Binding{
		Name: "data-bucket",
		Type: contracts.BindingTypeS3Prefix,
		S3: &contracts.S3PrefixBinding{
			Bucket:   "my-bucket",
			Prefix:   "data/",
			Region:   "us-west-2",
			Endpoint: "https://s3.amazonaws.com",
		},
	}

	kafkaBinding := &contracts.Binding{
		Name: "events-topic",
		Type: contracts.BindingTypeKafkaTopic,
		Kafka: &contracts.KafkaTopicBinding{
			Topic:          "events",
			SchemaRegistry: "http://schema-registry:8081",
		},
	}

	postgresBinding := &contracts.Binding{
		Name: "users-table",
		Type: contracts.BindingTypePostgresTable,
		Postgres: &contracts.PostgresTableBinding{
			Table:    "users",
			Database: "app_db",
			Schema:   "public",
		},
	}

	tests := []struct {
		name     string
		binding  *contracts.Binding
		property string
		wantVal  string
		wantErr  bool
	}{
		// Common properties
		{name: "s3 type", binding: s3Binding, property: "type", wantVal: "s3-prefix", wantErr: false},
		{name: "s3 name", binding: s3Binding, property: "name", wantVal: "data-bucket", wantErr: false},

		// S3 properties
		{name: "s3 bucket", binding: s3Binding, property: "bucket", wantVal: "my-bucket", wantErr: false},
		{name: "s3 prefix", binding: s3Binding, property: "prefix", wantVal: "data/", wantErr: false},
		{name: "s3 region", binding: s3Binding, property: "region", wantVal: "us-west-2", wantErr: false},
		{name: "s3 endpoint", binding: s3Binding, property: "endpoint", wantVal: "https://s3.amazonaws.com", wantErr: false},

		// Kafka properties
		{name: "kafka type", binding: kafkaBinding, property: "type", wantVal: "kafka-topic", wantErr: false},
		{name: "kafka topic", binding: kafkaBinding, property: "topic", wantVal: "events", wantErr: false},
		{name: "kafka schemaRegistry", binding: kafkaBinding, property: "schemaRegistry", wantVal: "http://schema-registry:8081", wantErr: false},

		// Postgres properties
		{name: "postgres type", binding: postgresBinding, property: "type", wantVal: "postgres-table", wantErr: false},
		{name: "postgres table", binding: postgresBinding, property: "table", wantVal: "users", wantErr: false},
		{name: "postgres database", binding: postgresBinding, property: "database", wantVal: "app_db", wantErr: false},
		{name: "postgres schema", binding: postgresBinding, property: "schema", wantVal: "public", wantErr: false},

		// Unknown property
		{name: "unknown property", binding: s3Binding, property: "unknown", wantVal: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := GetBindingProperty(tt.binding, tt.property)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBindingProperty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && val != tt.wantVal {
				t.Errorf("value = %v, want %v", val, tt.wantVal)
			}
		})
	}
}

func TestGetBindingProperty_NilConfig(t *testing.T) {
	tests := []struct {
		name     string
		binding  *contracts.Binding
		property string
	}{
		{
			name:     "s3 with nil config",
			binding:  &contracts.Binding{Name: "test", Type: contracts.BindingTypeS3Prefix, S3: nil},
			property: "bucket",
		},
		{
			name:     "kafka with nil config",
			binding:  &contracts.Binding{Name: "test", Type: contracts.BindingTypeKafkaTopic, Kafka: nil},
			property: "topic",
		},
		{
			name:     "postgres with nil config",
			binding:  &contracts.Binding{Name: "test", Type: contracts.BindingTypePostgresTable, Postgres: nil},
			property: "table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetBindingProperty(tt.binding, tt.property)
			if err == nil {
				t.Error("expected error for nil config")
			}
		})
	}
}
