package runner

import (
	"testing"
)

func TestCellResolver_CellNamespace(t *testing.T) {
	tests := []struct {
		cellName string
		want     string
	}{
		{"canary", "dp-canary"},
		{"stable", "dp-stable"},
		{"dev-dgarcia", "dp-dev-dgarcia"},
		{"local", "dp-local"},
	}
	for _, tt := range tests {
		r := NewCellResolver(tt.cellName, "", nil)
		if got := r.cellNamespace(); got != tt.want {
			t.Errorf("cellNamespace(%q) = %q, want %q", tt.cellName, got, tt.want)
		}
	}
}

func TestParseStoreFromJSON(t *testing.T) {
	input := `{
		"apiVersion": "dp.io/v1alpha1",
		"kind": "Store",
		"metadata": {
			"name": "warehouse",
			"namespace": "dp-canary"
		},
		"spec": {
			"connector": "postgres",
			"connection": {
				"connection_string": "postgresql://canary-db:5432/dp_canary"
			},
			"secrets": {
				"password": "${PG_PASS}"
			}
		}
	}`

	store, err := parseStoreFromJSON([]byte(input))
	if err != nil {
		t.Fatalf("parseStoreFromJSON failed: %v", err)
	}

	if store.Metadata.Name != "warehouse" {
		t.Errorf("Name = %q, want %q", store.Metadata.Name, "warehouse")
	}
	if store.Metadata.Namespace != "dp-canary" {
		t.Errorf("Namespace = %q, want %q", store.Metadata.Namespace, "dp-canary")
	}
	if store.Spec.Connector != "postgres" {
		t.Errorf("Connector = %q, want %q", store.Spec.Connector, "postgres")
	}
	connStr, ok := store.Spec.Connection["connection_string"]
	if !ok {
		t.Fatal("connection_string not found in Connection")
	}
	if connStr != "postgresql://canary-db:5432/dp_canary" {
		t.Errorf("connection_string = %v, want %q", connStr, "postgresql://canary-db:5432/dp_canary")
	}
	if store.Spec.Secrets["password"] != "${PG_PASS}" {
		t.Errorf("Secrets[password] = %q, want %q", store.Spec.Secrets["password"], "${PG_PASS}")
	}
}

func TestParseStoreListFromJSON(t *testing.T) {
	input := `{
		"apiVersion": "v1",
		"kind": "List",
		"items": [
			{
				"apiVersion": "dp.io/v1alpha1",
				"kind": "Store",
				"metadata": {"name": "warehouse", "namespace": "dp-canary"},
				"spec": {"connector": "postgres", "connection": {"connection_string": "pg://localhost:5432/db"}}
			},
			{
				"apiVersion": "dp.io/v1alpha1",
				"kind": "Store",
				"metadata": {"name": "lake-raw", "namespace": "dp-canary"},
				"spec": {"connector": "s3", "connection": {"bucket": "dp-canary-raw"}}
			}
		]
	}`

	stores, err := parseStoreListFromJSON([]byte(input))
	if err != nil {
		t.Fatalf("parseStoreListFromJSON failed: %v", err)
	}

	if len(stores) != 2 {
		t.Fatalf("len(stores) = %d, want 2", len(stores))
	}
	if stores[0].Metadata.Name != "warehouse" {
		t.Errorf("stores[0].Name = %q, want %q", stores[0].Metadata.Name, "warehouse")
	}
	if stores[1].Metadata.Name != "lake-raw" {
		t.Errorf("stores[1].Name = %q, want %q", stores[1].Metadata.Name, "lake-raw")
	}
	if stores[1].Spec.Connector != "s3" {
		t.Errorf("stores[1].Connector = %q, want %q", stores[1].Spec.Connector, "s3")
	}
}
