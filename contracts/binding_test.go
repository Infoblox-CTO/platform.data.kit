package contracts

import (
	"testing"
)

func TestBindingType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		binding  BindingType
		wantType string
	}{
		{
			name:     "s3 prefix",
			binding:  BindingTypeS3Prefix,
			wantType: "s3-prefix",
		},
		{
			name:     "kafka topic",
			binding:  BindingTypeKafkaTopic,
			wantType: "kafka-topic",
		},
		{
			name:     "postgres table",
			binding:  BindingTypePostgresTable,
			wantType: "postgres-table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.binding); got != tt.wantType {
				t.Errorf("BindingType = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestBinding_S3Prefix(t *testing.T) {
	tests := []struct {
		name       string
		binding    Binding
		wantBucket string
		wantPrefix string
	}{
		{
			name: "s3 binding with prefix",
			binding: Binding{
				Name: "raw-data",
				Type: BindingTypeS3Prefix,
				S3: &S3PrefixBinding{
					Bucket: "my-bucket",
					Prefix: "data/raw/",
					Region: "us-east-1",
				},
			},
			wantBucket: "my-bucket",
			wantPrefix: "data/raw/",
		},
		{
			name: "s3 binding without prefix",
			binding: Binding{
				Name: "output",
				Type: BindingTypeS3Prefix,
				S3: &S3PrefixBinding{
					Bucket: "output-bucket",
				},
			},
			wantBucket: "output-bucket",
			wantPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.binding.S3 == nil {
				t.Fatal("S3 binding is nil")
			}
			if got := tt.binding.S3.Bucket; got != tt.wantBucket {
				t.Errorf("Bucket = %v, want %v", got, tt.wantBucket)
			}
			if got := tt.binding.S3.Prefix; got != tt.wantPrefix {
				t.Errorf("Prefix = %v, want %v", got, tt.wantPrefix)
			}
		})
	}
}

func TestBinding_Kafka(t *testing.T) {
	tests := []struct {
		name        string
		binding     Binding
		wantTopic   string
		wantBrokers int
	}{
		{
			name: "kafka binding",
			binding: Binding{
				Name: "events",
				Type: BindingTypeKafkaTopic,
				Kafka: &KafkaTopicBinding{
					Topic:          "user-events",
					Brokers:        []string{"kafka-1:9092", "kafka-2:9092"},
					ConsumerGroup:  "my-group",
					SchemaRegistry: "http://schema-registry:8081",
				},
			},
			wantTopic:   "user-events",
			wantBrokers: 2,
		},
		{
			name: "kafka single broker",
			binding: Binding{
				Name: "logs",
				Type: BindingTypeKafkaTopic,
				Kafka: &KafkaTopicBinding{
					Topic:   "app-logs",
					Brokers: []string{"localhost:9092"},
				},
			},
			wantTopic:   "app-logs",
			wantBrokers: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.binding.Kafka == nil {
				t.Fatal("Kafka binding is nil")
			}
			if got := tt.binding.Kafka.Topic; got != tt.wantTopic {
				t.Errorf("Topic = %v, want %v", got, tt.wantTopic)
			}
			if got := len(tt.binding.Kafka.Brokers); got != tt.wantBrokers {
				t.Errorf("len(Brokers) = %v, want %v", got, tt.wantBrokers)
			}
		})
	}
}

func TestBinding_Postgres(t *testing.T) {
	tests := []struct {
		name         string
		binding      Binding
		wantDatabase string
		wantTable    string
	}{
		{
			name: "postgres binding",
			binding: Binding{
				Name: "users-table",
				Type: BindingTypePostgresTable,
				Postgres: &PostgresTableBinding{
					Host:     "localhost",
					Port:     5432,
					Database: "mydb",
					Schema:   "public",
					Table:    "users",
					SSLMode:  "require",
				},
			},
			wantDatabase: "mydb",
			wantTable:    "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.binding.Postgres == nil {
				t.Fatal("Postgres binding is nil")
			}
			if got := tt.binding.Postgres.Database; got != tt.wantDatabase {
				t.Errorf("Database = %v, want %v", got, tt.wantDatabase)
			}
			if got := tt.binding.Postgres.Table; got != tt.wantTable {
				t.Errorf("Table = %v, want %v", got, tt.wantTable)
			}
		})
	}
}
