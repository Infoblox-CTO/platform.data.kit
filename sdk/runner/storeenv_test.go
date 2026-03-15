package runner

import (
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestBuildDSN_ConnectionString(t *testing.T) {
	store := &contracts.Store{
		Spec: contracts.StoreSpec{
			Connector: "postgres",
			Connection: map[string]any{
				"connection_string": "postgresql://user:pass@host:5432/db",
			},
		},
	}

	dsn, err := BuildDSN(store)
	if err != nil {
		t.Fatalf("BuildDSN() error: %v", err)
	}
	if dsn != "postgresql://user:pass@host:5432/db" {
		t.Errorf("DSN = %q, want postgresql://user:pass@host:5432/db", dsn)
	}
}

func TestBuildDSN_PostgresFields(t *testing.T) {
	store := &contracts.Store{
		Spec: contracts.StoreSpec{
			Connector: "postgres",
			Connection: map[string]any{
				"host":   "prod-db",
				"port":   "5432",
				"dbname": "mydb",
				"user":   "app",
			},
			Secrets: map[string]string{
				"password": "secret123",
			},
		},
	}

	dsn, err := BuildDSN(store)
	if err != nil {
		t.Fatalf("BuildDSN() error: %v", err)
	}
	if dsn != "postgresql://app:secret123@prod-db:5432/mydb?sslmode=disable" {
		t.Errorf("DSN = %q", dsn)
	}
}

func TestBuildDSN_S3(t *testing.T) {
	store := &contracts.Store{
		Spec: contracts.StoreSpec{
			Connector: "s3",
			Connection: map[string]any{
				"bucket":   "my-bucket",
				"region":   "us-east-1",
				"endpoint": "http://localhost:4566",
			},
		},
	}

	dsn, err := BuildDSN(store)
	if err != nil {
		t.Fatalf("BuildDSN() error: %v", err)
	}
	if dsn != "s3://my-bucket?endpoint=http%3A%2F%2Flocalhost%3A4566&region=us-east-1" {
		t.Errorf("DSN = %q", dsn)
	}
}

func TestBuildDSN_Kafka(t *testing.T) {
	store := &contracts.Store{
		Spec: contracts.StoreSpec{
			Connector: "kafka",
			Connection: map[string]any{
				"bootstrapServers": "broker1:9092,broker2:9092",
				"topic":            "events",
			},
		},
	}

	dsn, err := BuildDSN(store)
	if err != nil {
		t.Fatalf("BuildDSN() error: %v", err)
	}
	if dsn != "kafka://broker1:9092,broker2:9092/events" {
		t.Errorf("DSN = %q", dsn)
	}
}

func TestEnvVarName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"warehouse", "WAREHOUSE"},
		{"pg-warehouse", "PG_WAREHOUSE"},
		{"lake.raw", "LAKE_RAW"},
		{"s3-raw-data", "S3_RAW_DATA"},
	}
	for _, tt := range tests {
		got := EnvVarName(tt.input)
		if got != tt.want {
			t.Errorf("EnvVarName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStoreEnvsToMap(t *testing.T) {
	envs := []StoreEnv{
		{Name: "warehouse", Type: "postgres", DSN: "postgresql://host/db"},
		{Name: "lake-raw", Type: "s3", DSN: "s3://bucket"},
	}

	m := StoreEnvsToMap(envs)

	if m["DK_STORE_DSN_WAREHOUSE"] != "postgresql://host/db" {
		t.Errorf("DSN_WAREHOUSE = %q", m["DK_STORE_DSN_WAREHOUSE"])
	}
	if m["DK_STORE_TYPE_WAREHOUSE"] != "postgres" {
		t.Errorf("TYPE_WAREHOUSE = %q", m["DK_STORE_TYPE_WAREHOUSE"])
	}
	if m["DK_STORE_DSN_LAKE_RAW"] != "s3://bucket" {
		t.Errorf("DSN_LAKE_RAW = %q", m["DK_STORE_DSN_LAKE_RAW"])
	}
	if m["DK_STORE_TYPE_LAKE_RAW"] != "s3" {
		t.Errorf("TYPE_LAKE_RAW = %q", m["DK_STORE_TYPE_LAKE_RAW"])
	}
}

func TestConnectionStringWithSecretInterpolation(t *testing.T) {
	store := &contracts.Store{
		Spec: contracts.StoreSpec{
			Connector: "postgres",
			Connection: map[string]any{
				"connection_string": "postgresql://app:${PG_PASS}@host:5432/db",
			},
			Secrets: map[string]string{
				"PG_PASS": "s3cret",
			},
		},
	}

	dsn, err := BuildDSN(store)
	if err != nil {
		t.Fatalf("BuildDSN() error: %v", err)
	}
	if dsn != "postgresql://app:s3cret@host:5432/db" {
		t.Errorf("DSN = %q", dsn)
	}
}
