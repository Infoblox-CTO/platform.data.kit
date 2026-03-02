package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConnector_YAMLRoundTrip(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres-1-2-0
  labels:
    managed-by: platform-team
    datakit.infoblox.dev/provider: postgres
spec:
  provider: postgres
  version: 1.2.0
  type: postgres
  protocol: postgresql
  capabilities:
    - source
    - destination
  plugin:
    source: ghcr.io/infobloxopen/cq-source-postgres:0.1.0
    destination: ghcr.io/cloudquery/cq-destination-postgres:latest
  tools:
    - name: psql
      description: Launch interactive psql session
      type: exec
      requires:
        - psql
      command: psql "{{ .DSN }}"
    - name: dsn
      description: Print connection string
      type: config
      format: text
      template: "{{ .DSN }}"
  connectionSchema:
    host:
      field: host
      default: localhost
    port:
      field: port
      default: "5432"
    database:
      field: database
      default: postgres
    user:
      field: username
      secret: true
    password:
      field: password
      secret: true
`

	var c Connector
	if err := yaml.Unmarshal([]byte(input), &c); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if c.APIVersion != "data.infoblox.com/v1alpha1" {
		t.Errorf("APIVersion = %q, want %q", c.APIVersion, "data.infoblox.com/v1alpha1")
	}
	if c.Kind != "Connector" {
		t.Errorf("Kind = %q, want %q", c.Kind, "Connector")
	}
	if c.Metadata.Name != "postgres-1-2-0" {
		t.Errorf("Metadata.Name = %q, want %q", c.Metadata.Name, "postgres-1-2-0")
	}
	if c.Metadata.Labels["managed-by"] != "platform-team" {
		t.Errorf("Metadata.Labels[managed-by] = %q, want %q", c.Metadata.Labels["managed-by"], "platform-team")
	}
	if c.Metadata.Labels["datakit.infoblox.dev/provider"] != "postgres" {
		t.Errorf("Metadata.Labels[datakit.infoblox.dev/provider] = %q, want %q", c.Metadata.Labels["datakit.infoblox.dev/provider"], "postgres")
	}
	if c.Spec.Provider != "postgres" {
		t.Errorf("Spec.Provider = %q, want %q", c.Spec.Provider, "postgres")
	}
	if c.Spec.Version != "1.2.0" {
		t.Errorf("Spec.Version = %q, want %q", c.Spec.Version, "1.2.0")
	}
	if c.Spec.Type != "postgres" {
		t.Errorf("Spec.Type = %q, want %q", c.Spec.Type, "postgres")
	}
	if c.Spec.Protocol != "postgresql" {
		t.Errorf("Spec.Protocol = %q, want %q", c.Spec.Protocol, "postgresql")
	}
	if len(c.Spec.Capabilities) != 2 {
		t.Fatalf("Spec.Capabilities len = %d, want 2", len(c.Spec.Capabilities))
	}
	if c.Spec.Capabilities[0] != "source" || c.Spec.Capabilities[1] != "destination" {
		t.Errorf("Spec.Capabilities = %v, want [source destination]", c.Spec.Capabilities)
	}
	if c.Spec.Plugin == nil {
		t.Fatal("Spec.Plugin is nil")
	}
	if c.Spec.Plugin.Source != "ghcr.io/infobloxopen/cq-source-postgres:0.1.0" {
		t.Errorf("Spec.Plugin.Source = %q", c.Spec.Plugin.Source)
	}
	if c.Spec.Plugin.Destination != "ghcr.io/cloudquery/cq-destination-postgres:latest" {
		t.Errorf("Spec.Plugin.Destination = %q", c.Spec.Plugin.Destination)
	}

	// Tools
	if len(c.Spec.Tools) != 2 {
		t.Fatalf("Spec.Tools len = %d, want 2", len(c.Spec.Tools))
	}
	if c.Spec.Tools[0].Name != "psql" {
		t.Errorf("Spec.Tools[0].Name = %q, want %q", c.Spec.Tools[0].Name, "psql")
	}
	if c.Spec.Tools[0].Type != "exec" {
		t.Errorf("Spec.Tools[0].Type = %q, want %q", c.Spec.Tools[0].Type, "exec")
	}
	if len(c.Spec.Tools[0].Requires) != 1 || c.Spec.Tools[0].Requires[0] != "psql" {
		t.Errorf("Spec.Tools[0].Requires = %v, want [psql]", c.Spec.Tools[0].Requires)
	}
	if c.Spec.Tools[1].Name != "dsn" {
		t.Errorf("Spec.Tools[1].Name = %q, want %q", c.Spec.Tools[1].Name, "dsn")
	}
	if c.Spec.Tools[1].Format != "text" {
		t.Errorf("Spec.Tools[1].Format = %q, want %q", c.Spec.Tools[1].Format, "text")
	}

	// ConnectionSchema
	if len(c.Spec.ConnectionSchema) != 5 {
		t.Fatalf("Spec.ConnectionSchema len = %d, want 5", len(c.Spec.ConnectionSchema))
	}
	hostField := c.Spec.ConnectionSchema["host"]
	if hostField.Field != "host" || hostField.Default != "localhost" {
		t.Errorf("ConnectionSchema[host] = %+v", hostField)
	}
	userField := c.Spec.ConnectionSchema["user"]
	if !userField.Secret {
		t.Error("ConnectionSchema[user].Secret should be true")
	}

	// Round-trip: marshal and unmarshal
	out, err := yaml.Marshal(&c)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var c2 Connector
	if err := yaml.Unmarshal(out, &c2); err != nil {
		t.Fatalf("Unmarshal round-trip failed: %v", err)
	}
	if c2.Metadata.Name != c.Metadata.Name {
		t.Errorf("round-trip Name mismatch: %q vs %q", c2.Metadata.Name, c.Metadata.Name)
	}
	if c2.Spec.Type != c.Spec.Type {
		t.Errorf("round-trip Type mismatch: %q vs %q", c2.Spec.Type, c.Spec.Type)
	}
	if c2.Spec.Provider != c.Spec.Provider {
		t.Errorf("round-trip Provider mismatch: %q vs %q", c2.Spec.Provider, c.Spec.Provider)
	}
	if c2.Spec.Version != c.Spec.Version {
		t.Errorf("round-trip Version mismatch: %q vs %q", c2.Spec.Version, c.Spec.Version)
	}
	if len(c2.Spec.Tools) != len(c.Spec.Tools) {
		t.Errorf("round-trip Tools len mismatch: %d vs %d", len(c2.Spec.Tools), len(c.Spec.Tools))
	}
	if len(c2.Spec.ConnectionSchema) != len(c.Spec.ConnectionSchema) {
		t.Errorf("round-trip ConnectionSchema len mismatch: %d vs %d", len(c2.Spec.ConnectionSchema), len(c.Spec.ConnectionSchema))
	}
}

func TestConnector_ManifestInterface(t *testing.T) {
	c := &Connector{
		Metadata: ConnectorMetadata{Name: "s3-1-0-0"},
		Spec: ConnectorSpec{
			Provider: "s3",
			Version:  "1.0.0",
			Type:     "s3",
		},
	}
	if c.GetKind() != KindConnector {
		t.Errorf("GetKind() = %v, want %v", c.GetKind(), KindConnector)
	}
	if c.GetName() != "s3-1-0-0" {
		t.Errorf("GetName() = %q, want %q", c.GetName(), "s3-1-0-0")
	}
	if c.GetNamespace() != "" {
		t.Errorf("GetNamespace() = %q, want empty", c.GetNamespace())
	}
	if c.GetVersion() != "1.0.0" {
		t.Errorf("GetVersion() = %q, want %q", c.GetVersion(), "1.0.0")
	}
	if c.GetProvider() != "s3" {
		t.Errorf("GetProvider() = %q, want %q", c.GetProvider(), "s3")
	}
}

func TestConnector_GetProviderFallback(t *testing.T) {
	// When provider is not set, GetProvider falls back to Type.
	c := &Connector{
		Spec: ConnectorSpec{Type: "kafka"},
	}
	if c.GetProvider() != "kafka" {
		t.Errorf("GetProvider() = %q, want %q (fallback to Type)", c.GetProvider(), "kafka")
	}
}

func TestConnector_MinimalYAML(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: s3
spec:
  type: s3
  capabilities:
    - destination
`
	var c Connector
	if err := yaml.Unmarshal([]byte(input), &c); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if c.Metadata.Name != "s3" {
		t.Errorf("Name = %q, want %q", c.Metadata.Name, "s3")
	}
	if c.Spec.Plugin != nil {
		t.Error("Plugin should be nil for minimal connector")
	}
	if c.Spec.Protocol != "" {
		t.Errorf("Protocol should be empty, got %q", c.Spec.Protocol)
	}
	// Provider and version are optional — should be zero-value.
	if c.Spec.Provider != "" {
		t.Errorf("Provider should be empty, got %q", c.Spec.Provider)
	}
	if c.Spec.Version != "" {
		t.Errorf("Version should be empty, got %q", c.Spec.Version)
	}
	// GetProvider should fall back to Type.
	if c.GetProvider() != "s3" {
		t.Errorf("GetProvider() = %q, want %q (fallback)", c.GetProvider(), "s3")
	}
	// GetVersion should return empty for unversioned.
	if c.GetVersion() != "" {
		t.Errorf("GetVersion() = %q, want empty", c.GetVersion())
	}
	if c.Spec.Tools != nil {
		t.Error("Tools should be nil for minimal connector")
	}
	if c.Spec.ConnectionSchema != nil {
		t.Error("ConnectionSchema should be nil for minimal connector")
	}
}
