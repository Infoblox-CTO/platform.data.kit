package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/data-platform/contracts"
)

func TestNewBindingsValidator(t *testing.T) {
	bindings := []contracts.Binding{
		{
			Name: "input-data",
			Type: contracts.BindingTypeS3Prefix,
		},
	}

	v := NewBindingsValidator(bindings, "/path/to/bindings")

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if v.Name() != "bindings" {
		t.Errorf("Name() = %s, want bindings", v.Name())
	}
}

func TestBindingsValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		bindings  []contracts.Binding
		wantValid bool
		wantErrs  int
	}{
		{
			name:      "empty bindings",
			bindings:  []contracts.Binding{},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "valid s3 binding",
			bindings: []contracts.Binding{
				{
					Name: "input-data",
					Type: contracts.BindingTypeS3Prefix,
					S3: &contracts.S3PrefixBinding{
						Bucket: "my-bucket",
						Prefix: "data/",
						Region: "us-east-1",
					},
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "valid kafka binding",
			bindings: []contracts.Binding{
				{
					Name: "events",
					Type: contracts.BindingTypeKafkaTopic,
					Kafka: &contracts.KafkaTopicBinding{
						Topic:   "user-events",
						Brokers: []string{"localhost:9092"},
					},
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "valid postgres binding",
			bindings: []contracts.Binding{
				{
					Name: "users-table",
					Type: contracts.BindingTypePostgresTable,
					Postgres: &contracts.PostgresTableBinding{
						Host:     "localhost",
						Port:     5432,
						Database: "mydb",
						Table:    "users",
					},
				},
			},
			wantValid: true,
			wantErrs:  0,
		},
		{
			name: "missing name",
			bindings: []contracts.Binding{
				{
					Type: contracts.BindingTypeS3Prefix,
					S3: &contracts.S3PrefixBinding{
						Bucket: "my-bucket",
					},
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
		{
			name: "duplicate names",
			bindings: []contracts.Binding{
				{
					Name: "data",
					Type: contracts.BindingTypeS3Prefix,
					S3: &contracts.S3PrefixBinding{
						Bucket: "bucket1",
					},
				},
				{
					Name: "data",
					Type: contracts.BindingTypeS3Prefix,
					S3: &contracts.S3PrefixBinding{
						Bucket: "bucket2",
					},
				},
			},
			wantValid: false,
			wantErrs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewBindingsValidator(tt.bindings, "/path")
			errs := v.Validate(context.Background())

			if tt.wantValid && errs.HasErrors() {
				t.Errorf("expected valid, got errors: %v", errs)
			}
			if !tt.wantValid && !errs.HasErrors() {
				t.Error("expected errors, got valid")
			}
			if tt.wantErrs > 0 && len(errs) < tt.wantErrs {
				t.Errorf("len(errs) = %d, want at least %d", len(errs), tt.wantErrs)
			}
		})
	}
}

func TestBindingsValidator_ValidateFromFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "file not found",
			path:    "testdata/nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewBindingsValidatorFromFile(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
