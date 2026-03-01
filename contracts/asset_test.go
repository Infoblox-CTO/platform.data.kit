package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// New Asset model tests
// ---------------------------------------------------------------------------

func TestAssetManifest_YAMLRoundTrip(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
  namespace: default
spec:
  store: warehouse
  table: public.users
  classification: confidential
  schema:
    - name: id
      type: integer
    - name: email
      type: string
      pii: true
    - name: created_at
      type: timestamp
`

	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if a.APIVersion != "data.infoblox.com/v1alpha1" {
		t.Errorf("APIVersion = %q", a.APIVersion)
	}
	if a.Kind != "Asset" {
		t.Errorf("Kind = %q", a.Kind)
	}
	if a.Metadata.Name != "users" {
		t.Errorf("Metadata.Name = %q", a.Metadata.Name)
	}
	if a.Spec.Store != "warehouse" {
		t.Errorf("Spec.Store = %q", a.Spec.Store)
	}
	if a.Spec.Table != "public.users" {
		t.Errorf("Spec.Table = %q", a.Spec.Table)
	}
	if a.Spec.Classification != "confidential" {
		t.Errorf("Spec.Classification = %q", a.Spec.Classification)
	}
	if len(a.Spec.Schema) != 3 {
		t.Fatalf("Spec.Schema len = %d, want 3", len(a.Spec.Schema))
	}
	if a.Spec.Schema[0].Name != "id" || a.Spec.Schema[0].Type != "integer" {
		t.Errorf("Schema[0] = %+v", a.Spec.Schema[0])
	}
	if !a.Spec.Schema[1].PII {
		t.Error("Schema[1] (email) should have PII=true")
	}

	// Round-trip
	out, err := yaml.Marshal(&a)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var a2 AssetManifest
	if err := yaml.Unmarshal(out, &a2); err != nil {
		t.Fatalf("Unmarshal round-trip failed: %v", err)
	}
	if a2.Spec.Store != a.Spec.Store {
		t.Errorf("round-trip Store mismatch")
	}
}

func TestAssetManifest_OutputWithLineage(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users-parquet
spec:
  store: lake-raw
  prefix: data/users/
  format: parquet
  classification: confidential
  schema:
    - name: id
      type: integer
      from: users.id
    - name: email
      type: string
      pii: true
      from: users.email
    - name: created_at
      type: timestamp
      from: users.created_at
`

	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if a.Spec.Prefix != "data/users/" {
		t.Errorf("Spec.Prefix = %q", a.Spec.Prefix)
	}
	if a.Spec.Format != "parquet" {
		t.Errorf("Spec.Format = %q", a.Spec.Format)
	}
	// Verify lineage from fields
	for _, field := range a.Spec.Schema {
		if field.From == "" {
			t.Errorf("Schema field %q missing 'from' lineage", field.Name)
		}
	}
	if a.Spec.Schema[0].From != "users.id" {
		t.Errorf("Schema[0].From = %q, want %q", a.Spec.Schema[0].From, "users.id")
	}
}

func TestAssetManifest_KafkaTopic(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: user-events
spec:
  store: events
  topic: user.events.v1
  format: json
  classification: internal
`

	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if a.Spec.Topic != "user.events.v1" {
		t.Errorf("Spec.Topic = %q", a.Spec.Topic)
	}
}

func TestAssetManifest_ManifestInterface(t *testing.T) {
	a := &AssetManifest{
		Metadata: AssetMetadata{
			Name:      "users",
			Namespace: "analytics",
		},
	}
	if a.GetKind() != KindAsset {
		t.Errorf("GetKind() = %v, want %v", a.GetKind(), KindAsset)
	}
	if a.GetName() != "users" {
		t.Errorf("GetName() = %q", a.GetName())
	}
	if a.GetNamespace() != "analytics" {
		t.Errorf("GetNamespace() = %q", a.GetNamespace())
	}
}

func TestAssetManifest_Version(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
  version: "1.2.0"
  labels:
    domain: identity
    tier: raw
spec:
  store: warehouse
  table: public.users
`

	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if a.Metadata.Version != "1.2.0" {
		t.Errorf("Metadata.Version = %q, want %q", a.Metadata.Version, "1.2.0")
	}
	if a.GetVersion() != "1.2.0" {
		t.Errorf("GetVersion() = %q, want %q", a.GetVersion(), "1.2.0")
	}
	if a.Metadata.Labels["domain"] != "identity" {
		t.Errorf("Labels[domain] = %q", a.Metadata.Labels["domain"])
	}

	// Round-trip.
	out, err := yaml.Marshal(&a)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var a2 AssetManifest
	if err := yaml.Unmarshal(out, &a2); err != nil {
		t.Fatalf("Round-trip unmarshal failed: %v", err)
	}
	if a2.Metadata.Version != "1.2.0" {
		t.Errorf("Round-trip Version = %q", a2.Metadata.Version)
	}
}

func TestAssetManifest_VersionEmpty(t *testing.T) {
	a := &AssetManifest{
		Metadata: AssetMetadata{Name: "no-version"},
	}
	if a.GetVersion() != "" {
		t.Errorf("GetVersion() = %q, want empty", a.GetVersion())
	}
}

func TestAssetGroupManifest_YAML(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: AssetGroup
metadata:
  name: pg-snapshot
  namespace: default
spec:
  store: lake-raw
  assets:
    - users-parquet
    - orders-parquet
    - products-parquet
`

	var ag AssetGroupManifest
	if err := yaml.Unmarshal([]byte(input), &ag); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if ag.Kind != "AssetGroup" {
		t.Errorf("Kind = %q", ag.Kind)
	}
	if ag.Metadata.Name != "pg-snapshot" {
		t.Errorf("Metadata.Name = %q", ag.Metadata.Name)
	}
	if ag.Spec.Store != "lake-raw" {
		t.Errorf("Spec.Store = %q", ag.Spec.Store)
	}
	if len(ag.Spec.Assets) != 3 {
		t.Fatalf("Spec.Assets len = %d, want 3", len(ag.Spec.Assets))
	}
	if ag.Spec.Assets[0] != "users-parquet" {
		t.Errorf("Spec.Assets[0] = %q", ag.Spec.Assets[0])
	}
}

func TestAssetGroupManifest_ManifestInterface(t *testing.T) {
	ag := &AssetGroupManifest{
		Metadata: AssetGroupMetadata{Name: "pg-snapshot", Namespace: "default"},
	}
	if ag.GetKind() != KindAssetGroup {
		t.Errorf("GetKind() = %v, want %v", ag.GetKind(), KindAssetGroup)
	}
	if ag.GetName() != "pg-snapshot" {
		t.Errorf("GetName() = %q", ag.GetName())
	}
}

func TestAssetManifest_DevSeed_YAMLRoundTrip(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: warehouse
  table: example_table
  schema:
    - name: id
      type: integer
    - name: name
      type: string
  dev:
    seed:
      inline:
        - { id: 1, name: "alice" }
        - { id: 2, name: "bob" }
`
	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if a.Spec.Dev == nil {
		t.Fatal("Spec.Dev should not be nil")
	}
	if a.Spec.Dev.Seed == nil {
		t.Fatal("Spec.Dev.Seed should not be nil")
	}
	if len(a.Spec.Dev.Seed.Inline) != 2 {
		t.Fatalf("Spec.Dev.Seed.Inline len = %d, want 2", len(a.Spec.Dev.Seed.Inline))
	}
	if a.Spec.Dev.Seed.Inline[0]["name"] != "alice" {
		t.Errorf("Inline[0].name = %v, want alice", a.Spec.Dev.Seed.Inline[0]["name"])
	}
}

func TestAssetManifest_DevSeed_File(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: orders
spec:
  store: warehouse
  table: orders
  dev:
    seed:
      file: testdata/orders.csv
`
	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if a.Spec.Dev == nil || a.Spec.Dev.Seed == nil {
		t.Fatal("Spec.Dev.Seed should not be nil")
	}
	if a.Spec.Dev.Seed.File != "testdata/orders.csv" {
		t.Errorf("Spec.Dev.Seed.File = %q, want testdata/orders.csv", a.Spec.Dev.Seed.File)
	}
}

func TestAssetManifest_NoDev(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: warehouse
  table: users
`
	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if a.Spec.Dev != nil {
		t.Errorf("Spec.Dev should be nil when not specified, got %+v", a.Spec.Dev)
	}
}

func TestAssetManifest_DevSeed_Profiles(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: warehouse
  table: example_table
  schema:
    - name: id
      type: integer
    - name: name
      type: string
  dev:
    seed:
      inline:
        - { id: 1, name: "alice" }
      profiles:
        large:
          file: testdata/large.csv
        edge-cases:
          inline:
            - { id: -1, name: "" }
            - { id: 999, name: "O'Reilly" }
        empty:
          inline: []
`
	var a AssetManifest
	if err := yaml.Unmarshal([]byte(input), &a); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	seed := a.Spec.Dev.Seed
	if seed == nil {
		t.Fatal("Spec.Dev.Seed should not be nil")
	}

	// Default inline.
	if len(seed.Inline) != 1 || seed.Inline[0]["name"] != "alice" {
		t.Errorf("default inline unexpected: %v", seed.Inline)
	}

	// Profiles map.
	if len(seed.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(seed.Profiles))
	}

	// "large" profile.
	large := seed.Profiles["large"]
	if large == nil || large.File != "testdata/large.csv" {
		t.Errorf("large profile file: %v", large)
	}

	// "edge-cases" profile.
	edge := seed.Profiles["edge-cases"]
	if edge == nil || len(edge.Inline) != 2 {
		t.Errorf("edge-cases profile: %v", edge)
	}

	// "empty" profile.
	empty := seed.Profiles["empty"]
	if empty == nil || len(empty.Inline) != 0 {
		t.Errorf("empty profile should have 0 inline rows: %v", empty)
	}

	// Round-trip marshal.
	out, err := yaml.Marshal(&a)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	s := string(out)
	if !containsSubstring(s, "profiles") {
		t.Errorf("marshalled YAML should contain 'profiles': %s", s)
	}
}

// containsSubstring is a test helper used across test files in this package.
func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
