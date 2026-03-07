package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestGenerateCQConfig(t *testing.T) {
	// Set up a package directory with connector/, store/, asset/ manifests.
	pkgDir := t.TempDir()

	// --- connectors ---
	connDir := filepath.Join(pkgDir, "connector")
	if err := os.MkdirAll(connDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(connDir, "postgres.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  protocol: postgresql
  capabilities: [source, destination]
  plugin:
    source: ghcr.io/cloudquery/cq-source-postgresql:v8.0.0
    destination: ghcr.io/cloudquery/cq-destination-postgresql:v8.0.0
`)
	writeFile(t, filepath.Join(connDir, "s3.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: s3
spec:
  type: s3
  protocol: s3
  capabilities: [destination]
  plugin:
    destination: ghcr.io/cloudquery/cq-destination-s3:v1.0.0
`)

	// --- stores ---
	storeDir := filepath.Join(pkgDir, "store")
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(storeDir, "warehouse.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: warehouse
spec:
  connector: postgres
  connection:
    connection_string: "postgresql://user:pass@db:5432/mydb"
`)
	writeFile(t, filepath.Join(storeDir, "lake-raw.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: lake-raw
spec:
  connector: s3
  connection:
    bucket: my-bucket
    region: us-east-1
`)

	// --- assets ---
	assetDir := filepath.Join(pkgDir, "asset")
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(assetDir, "users.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: warehouse
  table: public.users
`)
	writeFile(t, filepath.Join(assetDir, "orders.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: orders
spec:
  store: warehouse
  table: public.orders
`)
	writeFile(t, filepath.Join(assetDir, "users-parquet.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: users-parquet
spec:
  store: lake-raw
  prefix: "users/"
  format: parquet
`)

	// Build a transform that reads two PG tables and writes to S3.
	transform := &contracts.Transform{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "Transform",
		Metadata: contracts.TransformMetadata{
			Name:      "pg-to-s3",
			Namespace: "data-team",
			Version:   "1.0.0",
		},
		Spec: contracts.TransformSpec{
			Runtime: contracts.RuntimeCloudQuery,
			Mode:    "batch",
			Inputs: []contracts.AssetRef{
				{Asset: "users"},
				{Asset: "orders"},
			},
			Outputs: []contracts.AssetRef{
				{Asset: "users-parquet"},
			},
		},
	}

	configData, plugins, err := generateCQConfig(transform, pkgDir)
	if err != nil {
		t.Fatalf("generateCQConfig() error: %v", err)
	}

	configStr := string(configData)

	// Verify source plugin.
	if !strings.Contains(configStr, "kind: source") {
		t.Error("config should contain a source document")
	}
	if !strings.Contains(configStr, "ghcr.io/cloudquery/cq-source-postgresql:v8.0.0") {
		t.Error("config should reference postgres source plugin image")
	}
	if !strings.Contains(configStr, "public.users") {
		t.Error("config should contain table public.users")
	}
	if !strings.Contains(configStr, "public.orders") {
		t.Error("config should contain table public.orders")
	}

	// Verify destination plugin.
	if !strings.Contains(configStr, "kind: destination") {
		t.Error("config should contain a destination document")
	}
	if !strings.Contains(configStr, "ghcr.io/cloudquery/cq-destination-s3:v1.0.0") {
		t.Error("config should reference S3 destination plugin image")
	}
	if !strings.Contains(configStr, "my-bucket") {
		t.Error("config should contain store connection bucket")
	}

	// Verify destination gets asset-level overrides.
	if !strings.Contains(configStr, "users/{{TABLE}}/{{UUID}}.{{FORMAT}}") {
		t.Errorf("config destination spec should contain CQ S3 path template built from prefix, got:\n%s", configStr)
	}
	if !strings.Contains(configStr, "parquet") {
		t.Error("config destination spec should contain format parquet")
	}

	// Verify plugins list.
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}

	src := plugins[0]
	if src.Kind != "source" {
		t.Errorf("plugin[0].Kind = %q, want source", src.Kind)
	}
	if src.Name != "postgres" {
		t.Errorf("plugin[0].Name = %q, want postgres", src.Name)
	}
	if src.Image != "ghcr.io/cloudquery/cq-source-postgresql:v8.0.0" {
		t.Errorf("plugin[0].Image = %q", src.Image)
	}
	if src.Port != 7777 {
		t.Errorf("plugin[0].Port = %d, want 7777", src.Port)
	}

	dst := plugins[1]
	if dst.Kind != "destination" {
		t.Errorf("plugin[1].Kind = %q, want destination", dst.Kind)
	}
	if dst.Name != "s3" {
		t.Errorf("plugin[1].Name = %q, want s3", dst.Name)
	}
	if dst.Port != 7778 {
		t.Errorf("plugin[1].Port = %d, want 7778", dst.Port)
	}
}

func TestGenerateCQConfig_NoRotate(t *testing.T) {
	pkgDir := t.TempDir()

	connDir := filepath.Join(pkgDir, "connector")
	os.MkdirAll(connDir, 0755)
	writeFile(t, filepath.Join(connDir, "s3.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: s3
spec:
  type: s3
  capabilities: [destination]
  plugin:
    destination: ghcr.io/cloudquery/cq-destination-s3:v1.0.0
`)
	writeFile(t, filepath.Join(connDir, "postgres.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  type: postgres
  capabilities: [source, destination]
  plugin:
    source: ghcr.io/cloudquery/cq-source-postgresql:v8.0.0
    destination: ghcr.io/cloudquery/cq-destination-postgresql:v8.0.0
`)

	storeDir := filepath.Join(pkgDir, "store")
	os.MkdirAll(storeDir, 0755)
	writeFile(t, filepath.Join(storeDir, "warehouse.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: warehouse
spec:
  connector: postgres
  connection:
    connection_string: "postgresql://user:pass@db:5432/mydb"
`)
	writeFile(t, filepath.Join(storeDir, "lake-raw.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: lake-raw
spec:
  connector: s3
  connection:
    bucket: my-bucket
    region: us-east-1
    no_rotate: true
`)

	assetDir := filepath.Join(pkgDir, "asset")
	os.MkdirAll(assetDir, 0755)
	writeFile(t, filepath.Join(assetDir, "users.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: warehouse
  table: public.users
`)
	writeFile(t, filepath.Join(assetDir, "users-parquet.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: users-parquet
spec:
  store: lake-raw
  prefix: "data/"
  format: parquet
`)

	transform := &contracts.Transform{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "Transform",
		Metadata: contracts.TransformMetadata{
			Name:      "test-no-rotate",
			Namespace: "data-team",
			Version:   "1.0.0",
		},
		Spec: contracts.TransformSpec{
			Runtime: contracts.RuntimeCloudQuery,
			Mode:    "batch",
			Inputs: []contracts.AssetRef{
				{Asset: "users"},
			},
			Outputs: []contracts.AssetRef{
				{Asset: "users-parquet"},
			},
		},
	}

	configData, _, err := generateCQConfig(transform, pkgDir)
	if err != nil {
		t.Fatalf("generateCQConfig() error: %v", err)
	}

	configStr := string(configData)

	// When no_rotate is true, path must NOT contain {{UUID}}.
	if strings.Contains(configStr, "{{UUID}}") {
		t.Errorf("config path should not contain {{UUID}} when no_rotate is true, got:\n%s", configStr)
	}
	// Path should still contain table and format placeholders.
	if !strings.Contains(configStr, "data/{{TABLE}}.{{FORMAT}}") {
		t.Errorf("config path should be prefix/{{TABLE}}.{{FORMAT}} when no_rotate is true, got:\n%s", configStr)
	}
}

func TestGenerateCQConfig_MissingAsset(t *testing.T) {
	pkgDir := t.TempDir()
	// No asset/ directory → should fail gracefully.
	os.MkdirAll(filepath.Join(pkgDir, "connector"), 0755)
	os.MkdirAll(filepath.Join(pkgDir, "store"), 0755)

	transform := &contracts.Transform{
		Spec: contracts.TransformSpec{
			Runtime: contracts.RuntimeCloudQuery,
			Inputs: []contracts.AssetRef{
				{Asset: "nonexistent"},
			},
		},
	}

	_, _, err := generateCQConfig(transform, pkgDir)
	if err == nil {
		t.Fatal("expected error for missing asset, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention asset name: %v", err)
	}
}

func TestGenerateCQConfig_MissingStore(t *testing.T) {
	pkgDir := t.TempDir()

	assetDir := filepath.Join(pkgDir, "asset")
	os.MkdirAll(assetDir, 0755)
	writeFile(t, filepath.Join(assetDir, "users.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: missing-store
  table: public.users
`)

	transform := &contracts.Transform{
		Spec: contracts.TransformSpec{
			Runtime: contracts.RuntimeCloudQuery,
			Inputs: []contracts.AssetRef{
				{Asset: "users"},
			},
		},
	}

	_, _, err := generateCQConfig(transform, pkgDir)
	if err == nil {
		t.Fatal("expected error for missing store, got nil")
	}
	if !strings.Contains(err.Error(), "missing-store") {
		t.Errorf("error should mention store name: %v", err)
	}
}

func TestGenerateCQConfig_MissingConnector(t *testing.T) {
	pkgDir := t.TempDir()

	assetDir := filepath.Join(pkgDir, "asset")
	os.MkdirAll(assetDir, 0755)
	writeFile(t, filepath.Join(assetDir, "users.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: warehouse
  table: public.users
`)

	storeDir := filepath.Join(pkgDir, "store")
	os.MkdirAll(storeDir, 0755)
	writeFile(t, filepath.Join(storeDir, "warehouse.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: warehouse
spec:
  connector: missing-connector
  connection:
    connection_string: "postgresql://localhost/db"
`)

	transform := &contracts.Transform{
		Spec: contracts.TransformSpec{
			Runtime: contracts.RuntimeCloudQuery,
			Inputs: []contracts.AssetRef{
				{Asset: "users"},
			},
		},
	}

	_, _, err := generateCQConfig(transform, pkgDir)
	if err == nil {
		t.Fatal("expected error for missing connector, got nil")
	}
	if !strings.Contains(err.Error(), "missing-connector") {
		t.Errorf("error should mention connector name: %v", err)
	}
}

func TestLoadPackageManifests(t *testing.T) {
	pkgDir := t.TempDir()

	// Create manifests in standard subdirectories.
	connDir := filepath.Join(pkgDir, "connector")
	os.MkdirAll(connDir, 0755)
	writeFile(t, filepath.Join(connDir, "pg.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Connector
metadata:
  name: pg
spec:
  type: postgres
  capabilities: [source]
  plugin:
    source: ghcr.io/cq/pg:v1
`)

	storeDir := filepath.Join(pkgDir, "store")
	os.MkdirAll(storeDir, 0755)
	writeFile(t, filepath.Join(storeDir, "mystore.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Store
metadata:
  name: mystore
spec:
  connector: pg
  connection:
    host: localhost
`)

	assetDir := filepath.Join(pkgDir, "asset")
	os.MkdirAll(assetDir, 0755)
	writeFile(t, filepath.Join(assetDir, "myasset.yaml"), `
apiVersion: datakit.infoblox.dev/v1alpha1
kind: Asset
metadata:
  name: myasset
spec:
  store: mystore
  table: public.users
`)

	pm, err := loadPackageManifests(pkgDir)
	if err != nil {
		t.Fatalf("loadPackageManifests() error: %v", err)
	}

	if len(pm.Connectors) != 1 {
		t.Errorf("expected 1 connector, got %d", len(pm.Connectors))
	}
	if _, ok := pm.Connectors["pg"]; !ok {
		t.Error("expected connector 'pg'")
	}

	if len(pm.Stores) != 1 {
		t.Errorf("expected 1 store, got %d", len(pm.Stores))
	}
	if _, ok := pm.Stores["mystore"]; !ok {
		t.Error("expected store 'mystore'")
	}

	if len(pm.Assets) != 1 {
		t.Errorf("expected 1 asset, got %d", len(pm.Assets))
	}
	if _, ok := pm.Assets["myasset"]; !ok {
		t.Error("expected asset 'myasset'")
	}
}

func TestLoadPackageManifests_EmptyDir(t *testing.T) {
	pkgDir := t.TempDir()

	pm, err := loadPackageManifests(pkgDir)
	if err != nil {
		t.Fatalf("loadPackageManifests() on empty dir error: %v", err)
	}

	if len(pm.Connectors) != 0 || len(pm.Stores) != 0 || len(pm.Assets) != 0 {
		t.Error("expected empty manifests for empty package directory")
	}
}

// writeFile is a test helper that writes content to a file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
