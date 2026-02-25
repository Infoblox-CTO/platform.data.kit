package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConnector_YAMLRoundTrip(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres
  labels:
    managed-by: platform-team
spec:
  type: postgres
  protocol: postgresql
  capabilities:
    - source
    - destination
  plugin:
    source: ghcr.io/infobloxopen/cq-source-postgres:0.1.0
    destination: ghcr.io/cloudquery/cq-destination-postgres:latest
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
	if c.Metadata.Name != "postgres" {
		t.Errorf("Metadata.Name = %q, want %q", c.Metadata.Name, "postgres")
	}
	if c.Metadata.Labels["managed-by"] != "platform-team" {
		t.Errorf("Metadata.Labels[managed-by] = %q, want %q", c.Metadata.Labels["managed-by"], "platform-team")
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
}

func TestConnector_ManifestInterface(t *testing.T) {
	c := &Connector{
		Metadata: ConnectorMetadata{Name: "s3"},
	}
	if c.GetKind() != KindConnector {
		t.Errorf("GetKind() = %v, want %v", c.GetKind(), KindConnector)
	}
	if c.GetName() != "s3" {
		t.Errorf("GetName() = %q, want %q", c.GetName(), "s3")
	}
	if c.GetNamespace() != "" {
		t.Errorf("GetNamespace() = %q, want empty", c.GetNamespace())
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
}
