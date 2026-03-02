package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStore_YAMLRoundTrip(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: default
  labels:
    team: data-platform
spec:
  connector: postgres
  connectorVersion: "^1.0.0"
  connection:
    host: dk-postgres-postgresql.dk-local.svc.cluster.local
    port: 5432
    database: dataplatform
    schema: public
    sslmode: disable
  secrets:
    username: ${PG_USER}
    password: ${PG_PASSWORD}
`

	var s Store
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if s.APIVersion != "data.infoblox.com/v1alpha1" {
		t.Errorf("APIVersion = %q", s.APIVersion)
	}
	if s.Kind != "Store" {
		t.Errorf("Kind = %q", s.Kind)
	}
	if s.Metadata.Name != "warehouse" {
		t.Errorf("Metadata.Name = %q, want %q", s.Metadata.Name, "warehouse")
	}
	if s.Metadata.Namespace != "default" {
		t.Errorf("Metadata.Namespace = %q, want %q", s.Metadata.Namespace, "default")
	}
	if s.Spec.Connector != "postgres" {
		t.Errorf("Spec.Connector = %q, want %q", s.Spec.Connector, "postgres")
	}
	if s.Spec.ConnectorVersion != "^1.0.0" {
		t.Errorf("Spec.ConnectorVersion = %q, want %q", s.Spec.ConnectorVersion, "^1.0.0")
	}
	if s.Spec.Connection["host"] != "dk-postgres-postgresql.dk-local.svc.cluster.local" {
		t.Errorf("Spec.Connection[host] = %v", s.Spec.Connection["host"])
	}
	// YAML numeric values decode as int
	if port, ok := s.Spec.Connection["port"].(int); !ok || port != 5432 {
		t.Errorf("Spec.Connection[port] = %v (type %T)", s.Spec.Connection["port"], s.Spec.Connection["port"])
	}
	if s.Spec.Secrets["username"] != "${PG_USER}" {
		t.Errorf("Spec.Secrets[username] = %q", s.Spec.Secrets["username"])
	}
	if s.Spec.Secrets["password"] != "${PG_PASSWORD}" {
		t.Errorf("Spec.Secrets[password] = %q", s.Spec.Secrets["password"])
	}

	// Round-trip
	out, err := yaml.Marshal(&s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var s2 Store
	if err := yaml.Unmarshal(out, &s2); err != nil {
		t.Fatalf("Unmarshal round-trip failed: %v", err)
	}
	if s2.Spec.Connector != s.Spec.Connector {
		t.Errorf("round-trip Connector mismatch")
	}
	if s2.Spec.ConnectorVersion != s.Spec.ConnectorVersion {
		t.Errorf("round-trip ConnectorVersion mismatch: %q vs %q", s2.Spec.ConnectorVersion, s.Spec.ConnectorVersion)
	}
}

func TestStore_WithoutConnectorVersion(t *testing.T) {
	// ConnectorVersion is optional — existing stores without it should still work.
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: legacy-store
spec:
  connector: postgres
  connection:
    host: localhost
`
	var s Store
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if s.Spec.ConnectorVersion != "" {
		t.Errorf("ConnectorVersion should be empty for legacy store, got %q", s.Spec.ConnectorVersion)
	}
}

func TestStore_ManifestInterface(t *testing.T) {
	s := &Store{
		Metadata: StoreMetadata{
			Name:      "lake-raw",
			Namespace: "analytics",
		},
	}
	if s.GetKind() != KindStore {
		t.Errorf("GetKind() = %v, want %v", s.GetKind(), KindStore)
	}
	if s.GetName() != "lake-raw" {
		t.Errorf("GetName() = %q", s.GetName())
	}
	if s.GetNamespace() != "analytics" {
		t.Errorf("GetNamespace() = %q", s.GetNamespace())
	}
}

func TestStore_S3(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: lake-raw
spec:
  connector: s3
  connectorVersion: ">=1.0.0"
  connection:
    bucket: cdpp-raw
    region: us-east-1
    endpoint: http://dk-localstack-localstack.dk-local.svc.cluster.local:4566
  secrets:
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
`
	var s Store
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if s.Spec.Connector != "s3" {
		t.Errorf("Spec.Connector = %q, want %q", s.Spec.Connector, "s3")
	}
	if s.Spec.ConnectorVersion != ">=1.0.0" {
		t.Errorf("Spec.ConnectorVersion = %q, want %q", s.Spec.ConnectorVersion, ">=1.0.0")
	}
	if s.Spec.Connection["bucket"] != "cdpp-raw" {
		t.Errorf("Spec.Connection[bucket] = %v", s.Spec.Connection["bucket"])
	}
}
