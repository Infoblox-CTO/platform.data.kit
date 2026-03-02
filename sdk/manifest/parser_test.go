package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestNewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Error("NewParser() returned nil")
	}

	_, ok := p.(*DefaultParser)
	if !ok {
		t.Error("NewParser() did not return *DefaultParser")
	}
}

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		wantKind contracts.Kind
		wantName string
	}{
		{
			name: "valid connector",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  capabilities:
    - source
    - destination
`),
			wantErr:  false,
			wantKind: contracts.KindConnector,
			wantName: "postgres",
		},
		{
			name: "valid store",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: data-team
spec:
  connector: postgres
  connection:
    host: db.example.com
`),
			wantErr:  false,
			wantKind: contracts.KindStore,
			wantName: "warehouse",
		},
		{
			name: "valid asset",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: warehouse
  table: public.users
`),
			wantErr:  false,
			wantKind: contracts.KindAsset,
			wantName: "users",
		},
		{
			name: "valid asset group",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: AssetGroup
metadata:
  name: pg-snapshot
spec:
  store: warehouse
  assets:
    - users
    - orders
`),
			wantErr:  false,
			wantKind: contracts.KindAssetGroup,
			wantName: "pg-snapshot",
		},
		{
			name: "valid transform",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  version: 1.0.0
spec:
  runtime: cloudquery
  inputs:
    - asset: users
  outputs:
    - asset: users-parquet
`),
			wantErr:  false,
			wantKind: contracts.KindTransform,
			wantName: "pg-to-s3",
		},
		{
			name: "unsupported kind returns error",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test
`),
			wantErr: true,
		},
		{
			name:    "empty bytes returns error",
			data:    []byte(""),
			wantErr: true,
		},
		{
			name:    "invalid YAML returns error",
			data:    []byte("invalid: yaml: content: ["),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, kind, err := ParseManifest(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if kind != tt.wantKind {
					t.Errorf("kind = %v, want %v", kind, tt.wantKind)
				}
				if m.GetName() != tt.wantName {
					t.Errorf("name = %v, want %v", m.GetName(), tt.wantName)
				}
			}
		})
	}
}

func TestParseManifestFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, dir string) string
		wantErr  bool
		wantName string
		wantKind contracts.Kind
	}{
		{
			name: "valid connector file",
			setup: func(t *testing.T, dir string) string {
				content := `apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: file-test
spec:
  type: postgres
  capabilities:
    - source
`
				path := filepath.Join(dir, "dk.yaml")
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path
			},
			wantErr:  false,
			wantName: "file-test",
			wantKind: contracts.KindConnector,
		},
		{
			name: "file not found",
			setup: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "nonexistent.yaml")
			},
			wantErr: true,
		},
		{
			name: "malformed file",
			setup: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "bad.yaml")
				if err := os.WriteFile(path, []byte("not valid yaml ["), 0644); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
				return path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(t, dir)

			m, kind, err := ParseManifestFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseManifestFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if m.GetName() != tt.wantName {
					t.Errorf("ParseManifestFile() name = %v, want %v", m.GetName(), tt.wantName)
				}
				if kind != tt.wantKind {
					t.Errorf("ParseManifestFile() kind = %v, want %v", kind, tt.wantKind)
				}
			}
		})
	}
}

func TestParser_ParseFromTestdata(t *testing.T) {
	t.Run("malformed yaml", func(t *testing.T) {
		_, _, err := ParseManifestFile("testdata/invalid/malformed.yaml")
		if err == nil {
			t.Error("expected error for malformed YAML")
		}
	})

	t.Run("missing metadata", func(t *testing.T) {
		m, _, err := ParseManifestFile("testdata/invalid/missing-metadata.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() unexpected error = %v", err)
		}
		if m.GetName() != "" {
			t.Errorf("expected empty name for missing metadata, got %v", m.GetName())
		}
	})
}

// Edge case tests for malformed YAML (T030)
func TestParser_MalformedYAML(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "unclosed bracket", data: []byte("key: [value")},
		{name: "unclosed brace", data: []byte("key: {nested: value")},
		{name: "bad indentation", data: []byte("key:\n value\n  nested: bad")},
		{name: "duplicate keys", data: []byte("key: value1\nkey: value2")},
		{name: "tabs instead of spaces", data: []byte("key:\n\t- value")},
		{name: "invalid unicode", data: []byte("key: \xff\xfe")},
		{name: "null bytes", data: []byte("key: \x00value")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseManifest(tt.data)
			// Some malformed YAML might parse, we're just checking no panic
			_ = err
		})
	}
}

// Edge case tests for missing required fields (T031)
func TestParser_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		checkErr bool
	}{
		{
			name: "missing apiVersion",
			data: []byte(`kind: Connector
metadata:
  name: test
spec:
  type: postgres
`),
			checkErr: false, // Parses but validation should catch
		},
		{
			name: "missing kind",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
metadata:
  name: test
`),
			checkErr: true, // Should error on empty kind
		},
		{
			name: "missing metadata",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Connector
spec:
  type: postgres
`),
			checkErr: false, // Parses but has empty metadata
		},
		{
			name: "missing spec",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: test
`),
			checkErr: false, // Parses but has empty spec
		},
		{
			name: "empty metadata name",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: ""
`),
			checkErr: false, // Parses but validation should catch empty name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, err := ParseManifest(tt.data)
			if tt.checkErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if m == nil {
					t.Error("expected manifest to be returned")
				}
			}
		})
	}
}

func TestManifestInterface(t *testing.T) {
	// Test new kinds satisfy Manifest interface
	t.Run("Connector", func(t *testing.T) {
		var m Manifest = &contracts.Connector{
			Kind:     string(contracts.KindConnector),
			Metadata: contracts.ConnectorMetadata{Name: "pg"},
		}
		if m.GetKind() != contracts.KindConnector {
			t.Errorf("GetKind() = %v, want Connector", m.GetKind())
		}
		if m.GetName() != "pg" {
			t.Errorf("GetName() = %v, want pg", m.GetName())
		}
	})

	t.Run("Store", func(t *testing.T) {
		var m Manifest = &contracts.Store{
			Kind:     string(contracts.KindStore),
			Metadata: contracts.StoreMetadata{Name: "warehouse", Namespace: "data-team"},
		}
		if m.GetKind() != contracts.KindStore {
			t.Errorf("GetKind() = %v, want Store", m.GetKind())
		}
		if m.GetName() != "warehouse" {
			t.Errorf("GetName() = %v, want warehouse", m.GetName())
		}
		if m.GetNamespace() != "data-team" {
			t.Errorf("GetNamespace() = %v, want data-team", m.GetNamespace())
		}
	})

	t.Run("Asset", func(t *testing.T) {
		var m Manifest = &contracts.AssetManifest{
			Kind:     string(contracts.KindAsset),
			Metadata: contracts.AssetMetadata{Name: "users", Namespace: "data-team"},
		}
		if m.GetKind() != contracts.KindAsset {
			t.Errorf("GetKind() = %v, want Asset", m.GetKind())
		}
		if m.GetName() != "users" {
			t.Errorf("GetName() = %v, want users", m.GetName())
		}
	})

	t.Run("AssetGroup", func(t *testing.T) {
		var m Manifest = &contracts.AssetGroupManifest{
			Kind:     string(contracts.KindAssetGroup),
			Metadata: contracts.AssetGroupMetadata{Name: "pg-snap"},
		}
		if m.GetKind() != contracts.KindAssetGroup {
			t.Errorf("GetKind() = %v, want AssetGroup", m.GetKind())
		}
		if m.GetName() != "pg-snap" {
			t.Errorf("GetName() = %v, want pg-snap", m.GetName())
		}
	})

	t.Run("Transform", func(t *testing.T) {
		var m Manifest = &contracts.Transform{
			Kind:     string(contracts.KindTransform),
			Metadata: contracts.TransformMetadata{Name: "pg-to-s3", Namespace: "ns", Version: "1.0.0"},
		}
		if m.GetKind() != contracts.KindTransform {
			t.Errorf("GetKind() = %v, want Transform", m.GetKind())
		}
		if m.GetName() != "pg-to-s3" {
			t.Errorf("GetName() = %v, want pg-to-s3", m.GetName())
		}
		if m.GetVersion() != "1.0.0" {
			t.Errorf("GetVersion() = %v, want 1.0.0", m.GetVersion())
		}
	})

}

// --- Tests for new kind parser methods ---

func TestDefaultParser_ParseConnector(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  protocol: postgresql
  capabilities:
    - source
    - destination
  plugin:
    source: ghcr.io/cloudquery/cq-source-postgresql:v8.0.0
    destination: ghcr.io/cloudquery/cq-destination-postgresql:v8.0.0
`)

	p := NewParser()
	c, err := p.ParseConnector(data)
	if err != nil {
		t.Fatalf("ParseConnector() error = %v", err)
	}
	if c.Metadata.Name != "postgres" {
		t.Errorf("name = %v, want postgres", c.Metadata.Name)
	}
	if c.Spec.Type != "postgres" {
		t.Errorf("type = %v, want postgres", c.Spec.Type)
	}
	if len(c.Spec.Capabilities) != 2 {
		t.Errorf("capabilities count = %v, want 2", len(c.Spec.Capabilities))
	}
	if c.Spec.Plugin == nil {
		t.Fatal("plugin is nil")
	}
	if c.Spec.Plugin.Source != "ghcr.io/cloudquery/cq-source-postgresql:v8.0.0" {
		t.Errorf("plugin.source = %v", c.Spec.Plugin.Source)
	}
}

func TestDefaultParser_ParseConnector_WrongKind(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: test
spec:
  connector: postgres
  connection:
    host: localhost
`)

	p := NewParser()
	_, err := p.ParseConnector(data)
	if err == nil {
		t.Error("expected error for wrong kind")
	}
}

func TestDefaultParser_ParseStore(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: warehouse
  namespace: data-team
spec:
  connector: postgres
  connection:
    host: db.example.com
    port: 5432
    database: analytics
  secrets:
    username: ${PG_USER}
    password: ${PG_PASS}
`)

	p := NewParser()
	s, err := p.ParseStore(data)
	if err != nil {
		t.Fatalf("ParseStore() error = %v", err)
	}
	if s.Metadata.Name != "warehouse" {
		t.Errorf("name = %v, want warehouse", s.Metadata.Name)
	}
	if s.Metadata.Namespace != "data-team" {
		t.Errorf("namespace = %v, want data-team", s.Metadata.Namespace)
	}
	if s.Spec.Connector != "postgres" {
		t.Errorf("connector = %v, want postgres", s.Spec.Connector)
	}
	if len(s.Spec.Connection) != 3 {
		t.Errorf("connection fields = %v, want 3", len(s.Spec.Connection))
	}
	if s.Spec.Secrets["username"] != "${PG_USER}" {
		t.Errorf("secrets.username = %v", s.Spec.Secrets["username"])
	}
}

func TestDefaultParser_ParseAsset(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
  namespace: data-team
spec:
  store: warehouse
  table: public.users
  classification: internal
  schema:
    - name: id
      type: integer
    - name: email
      type: string
      pii: true
`)

	p := NewParser()
	a, err := p.ParseAsset(data)
	if err != nil {
		t.Fatalf("ParseAsset() error = %v", err)
	}
	if a.Metadata.Name != "users" {
		t.Errorf("name = %v, want users", a.Metadata.Name)
	}
	if a.Spec.Store != "warehouse" {
		t.Errorf("store = %v, want warehouse", a.Spec.Store)
	}
	if a.Spec.Table != "public.users" {
		t.Errorf("table = %v, want public.users", a.Spec.Table)
	}
	if len(a.Spec.Schema) != 2 {
		t.Errorf("schema fields = %v, want 2", len(a.Spec.Schema))
	}
	if !a.Spec.Schema[1].PII {
		t.Error("expected email field to have PII=true")
	}
}

func TestDefaultParser_ParseAssetGroup(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: AssetGroup
metadata:
  name: pg-snapshot
  namespace: data-team
spec:
  store: warehouse
  assets:
    - users
    - orders
    - products
`)

	p := NewParser()
	ag, err := p.ParseAssetGroup(data)
	if err != nil {
		t.Fatalf("ParseAssetGroup() error = %v", err)
	}
	if ag.Metadata.Name != "pg-snapshot" {
		t.Errorf("name = %v, want pg-snapshot", ag.Metadata.Name)
	}
	if ag.Spec.Store != "warehouse" {
		t.Errorf("store = %v, want warehouse", ag.Spec.Store)
	}
	if len(ag.Spec.Assets) != 3 {
		t.Errorf("assets count = %v, want 3", len(ag.Spec.Assets))
	}
}

func TestDefaultParser_ParseTransform(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  namespace: data-team
  version: 1.0.0
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - asset: users
  outputs:
    - asset: users-parquet
  timeout: 30m
`)

	p := NewParser()
	tr, err := p.ParseTransform(data)
	if err != nil {
		t.Fatalf("ParseTransform() error = %v", err)
	}
	if tr.Metadata.Name != "pg-to-s3" {
		t.Errorf("name = %v, want pg-to-s3", tr.Metadata.Name)
	}
	if tr.Metadata.Version != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", tr.Metadata.Version)
	}
	if tr.Spec.Runtime != contracts.RuntimeCloudQuery {
		t.Errorf("runtime = %v, want cloudquery", tr.Spec.Runtime)
	}
	if tr.Spec.Mode != contracts.ModeBatch {
		t.Errorf("mode = %v, want batch", tr.Spec.Mode)
	}
	if len(tr.Spec.Inputs) != 1 {
		t.Errorf("inputs count = %v, want 1", len(tr.Spec.Inputs))
	}
	if len(tr.Spec.Outputs) != 1 {
		t.Errorf("outputs count = %v, want 1", len(tr.Spec.Outputs))
	}
	if tr.Spec.Timeout != "30m" {
		t.Errorf("timeout = %v, want 30m", tr.Spec.Timeout)
	}
}

// --- FromBytes/ToBytes round-trip tests ---

func TestConnectorFromBytes_RoundTrip(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: s3
spec:
  type: s3
  protocol: s3
  capabilities:
    - destination
`)

	c, err := ConnectorFromBytes(data)
	if err != nil {
		t.Fatalf("ConnectorFromBytes() error = %v", err)
	}

	out, err := ConnectorToBytes(c)
	if err != nil {
		t.Fatalf("ConnectorToBytes() error = %v", err)
	}

	c2, err := ConnectorFromBytes(out)
	if err != nil {
		t.Fatalf("re-parse error = %v", err)
	}
	if c2.Metadata.Name != "s3" {
		t.Errorf("round-trip name = %v, want s3", c2.Metadata.Name)
	}
	if c2.Spec.Type != "s3" {
		t.Errorf("round-trip type = %v, want s3", c2.Spec.Type)
	}
}

func TestStoreFromBytes_RoundTrip(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: lake-raw
spec:
  connector: s3
  connection:
    bucket: my-data-lake
    region: us-east-1
`)

	s, err := StoreFromBytes(data)
	if err != nil {
		t.Fatalf("StoreFromBytes() error = %v", err)
	}

	out, err := StoreToBytes(s)
	if err != nil {
		t.Fatalf("StoreToBytes() error = %v", err)
	}

	s2, err := StoreFromBytes(out)
	if err != nil {
		t.Fatalf("re-parse error = %v", err)
	}
	if s2.Metadata.Name != "lake-raw" {
		t.Errorf("round-trip name = %v, want lake-raw", s2.Metadata.Name)
	}
}

func TestAssetFromBytes_RoundTrip(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: orders
spec:
  store: warehouse
  table: public.orders
  format: parquet
  classification: confidential
  schema:
    - name: id
      type: integer
    - name: amount
      type: float
`)

	a, err := AssetFromBytes(data)
	if err != nil {
		t.Fatalf("AssetFromBytes() error = %v", err)
	}

	out, err := AssetToBytes(a)
	if err != nil {
		t.Fatalf("AssetToBytes() error = %v", err)
	}

	a2, err := AssetFromBytes(out)
	if err != nil {
		t.Fatalf("re-parse error = %v", err)
	}
	if a2.Metadata.Name != "orders" {
		t.Errorf("round-trip name = %v, want orders", a2.Metadata.Name)
	}
	if len(a2.Spec.Schema) != 2 {
		t.Errorf("round-trip schema = %v, want 2", len(a2.Spec.Schema))
	}
}

func TestAssetGroupFromBytes_RoundTrip(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: AssetGroup
metadata:
  name: snapshot
spec:
  store: warehouse
  assets:
    - users
    - orders
`)

	ag, err := AssetGroupFromBytes(data)
	if err != nil {
		t.Fatalf("AssetGroupFromBytes() error = %v", err)
	}

	out, err := AssetGroupToBytes(ag)
	if err != nil {
		t.Fatalf("AssetGroupToBytes() error = %v", err)
	}

	ag2, err := AssetGroupFromBytes(out)
	if err != nil {
		t.Fatalf("re-parse error = %v", err)
	}
	if ag2.Metadata.Name != "snapshot" {
		t.Errorf("round-trip name = %v, want snapshot", ag2.Metadata.Name)
	}
	if len(ag2.Spec.Assets) != 2 {
		t.Errorf("round-trip assets = %v, want 2", len(ag2.Spec.Assets))
	}
}

func TestTransformFromBytes_RoundTrip(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: enrich
  version: 2.0.0
spec:
  runtime: generic-python
  mode: batch
  image: myorg/enrich:v2
  command:
    - python
    - -m
    - enrich
  inputs:
    - asset: raw-events
  outputs:
    - asset: enriched-events
  env:
    - name: LOG_LEVEL
      value: debug
  timeout: 1h
`)

	tr, err := TransformFromBytes(data)
	if err != nil {
		t.Fatalf("TransformFromBytes() error = %v", err)
	}

	out, err := TransformToBytes(tr)
	if err != nil {
		t.Fatalf("TransformToBytes() error = %v", err)
	}

	tr2, err := TransformFromBytes(out)
	if err != nil {
		t.Fatalf("re-parse error = %v", err)
	}
	if tr2.Metadata.Name != "enrich" {
		t.Errorf("round-trip name = %v, want enrich", tr2.Metadata.Name)
	}
	if tr2.Spec.Image != "myorg/enrich:v2" {
		t.Errorf("round-trip image = %v, want myorg/enrich:v2", tr2.Spec.Image)
	}
	if len(tr2.Spec.Command) != 3 {
		t.Errorf("round-trip command = %v, want 3", len(tr2.Spec.Command))
	}
}

// --- File parsing tests for new kinds ---

func TestParseConnectorFile(t *testing.T) {
	c, err := ParseConnectorFile("testdata/valid/connector.yaml")
	if err != nil {
		t.Fatalf("ParseConnectorFile() error = %v", err)
	}
	if c.Metadata.Name != "postgres" {
		t.Errorf("name = %v, want postgres", c.Metadata.Name)
	}
}

func TestParseStoreFile(t *testing.T) {
	s, err := ParseStoreFile("testdata/valid/store.yaml")
	if err != nil {
		t.Fatalf("ParseStoreFile() error = %v", err)
	}
	if s.Metadata.Name != "warehouse" {
		t.Errorf("name = %v, want warehouse", s.Metadata.Name)
	}
}

func TestParseAssetFile(t *testing.T) {
	a, err := ParseAssetFile("testdata/valid/asset.yaml")
	if err != nil {
		t.Fatalf("ParseAssetFile() error = %v", err)
	}
	if a.Metadata.Name != "users" {
		t.Errorf("name = %v, want users", a.Metadata.Name)
	}
}

func TestParseAssetGroupFile(t *testing.T) {
	ag, err := ParseAssetGroupFile("testdata/valid/asset-group.yaml")
	if err != nil {
		t.Fatalf("ParseAssetGroupFile() error = %v", err)
	}
	if ag.Metadata.Name != "pg-snapshot" {
		t.Errorf("name = %v, want pg-snapshot", ag.Metadata.Name)
	}
}

func TestParseTransformFile(t *testing.T) {
	tr, err := ParseTransformFile("testdata/valid/transform.yaml")
	if err != nil {
		t.Fatalf("ParseTransformFile() error = %v", err)
	}
	if tr.Metadata.Name != "pg-to-s3" {
		t.Errorf("name = %v, want pg-to-s3", tr.Metadata.Name)
	}
}

func TestParser_ParseFromTestdata_NewKinds(t *testing.T) {
	t.Run("valid connector", func(t *testing.T) {
		m, kind, err := ParseManifestFile("testdata/valid/connector.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() error = %v", err)
		}
		if m.GetName() != "postgres" {
			t.Errorf("name = %v, want postgres", m.GetName())
		}
		if kind != contracts.KindConnector {
			t.Errorf("kind = %v, want Connector", kind)
		}
	})

	t.Run("valid store", func(t *testing.T) {
		m, kind, err := ParseManifestFile("testdata/valid/store.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() error = %v", err)
		}
		if m.GetName() != "warehouse" {
			t.Errorf("name = %v, want warehouse", m.GetName())
		}
		if kind != contracts.KindStore {
			t.Errorf("kind = %v, want Store", kind)
		}
	})

	t.Run("valid asset", func(t *testing.T) {
		m, kind, err := ParseManifestFile("testdata/valid/asset.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() error = %v", err)
		}
		if m.GetName() != "users" {
			t.Errorf("name = %v, want users", m.GetName())
		}
		if kind != contracts.KindAsset {
			t.Errorf("kind = %v, want Asset", kind)
		}
	})

	t.Run("valid asset group", func(t *testing.T) {
		m, kind, err := ParseManifestFile("testdata/valid/asset-group.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() error = %v", err)
		}
		if m.GetName() != "pg-snapshot" {
			t.Errorf("name = %v, want pg-snapshot", m.GetName())
		}
		if kind != contracts.KindAssetGroup {
			t.Errorf("kind = %v, want AssetGroup", kind)
		}
	})

	t.Run("valid transform", func(t *testing.T) {
		m, kind, err := ParseManifestFile("testdata/valid/transform.yaml")
		if err != nil {
			t.Fatalf("ParseManifestFile() error = %v", err)
		}
		if m.GetName() != "pg-to-s3" {
			t.Errorf("name = %v, want pg-to-s3", m.GetName())
		}
		if kind != contracts.KindTransform {
			t.Errorf("kind = %v, want Transform", kind)
		}
	})
}
