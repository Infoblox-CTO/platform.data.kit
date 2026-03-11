package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestReadWriteLockFile(t *testing.T) {
	dir := t.TempDir()

	lock := &contracts.LockFile{
		Version: "1",
		Schemas: []contracts.LockedSchema{
			{
				Module:   "users",
				Version:  "1.2.3",
				Repo:     "https://github.com/example/schemas",
				Ref:      "v1.2.3",
				Format:   "parquet",
				Checksum: "sha256:abc123",
			},
			{
				Module:   "orders",
				Version:  "2.0.0",
				Repo:     "https://github.com/example/schemas",
				Ref:      "v2.0.0",
				Format:   "avro",
				Checksum: "sha256:def456",
			},
		},
	}

	// Write
	if err := WriteLockFile(dir, lock); err != nil {
		t.Fatalf("WriteLockFile: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, LockFileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("lock file not created: %v", err)
	}

	// Read back
	got, err := ReadLockFile(dir)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}

	if got.Version != "1" {
		t.Errorf("version = %q, want %q", got.Version, "1")
	}
	if len(got.Schemas) != 2 {
		t.Fatalf("len(schemas) = %d, want 2", len(got.Schemas))
	}
	if got.Schemas[0].Module != "users" {
		t.Errorf("schemas[0].module = %q, want %q", got.Schemas[0].Module, "users")
	}
	if got.Schemas[1].Checksum != "sha256:def456" {
		t.Errorf("schemas[1].checksum = %q, want %q", got.Schemas[1].Checksum, "sha256:def456")
	}
}

func TestReadLockFile_NotExist(t *testing.T) {
	dir := t.TempDir()
	lock, err := ReadLockFile(dir)
	if err != nil {
		t.Fatalf("ReadLockFile: %v", err)
	}
	if lock != nil {
		t.Errorf("expected nil lock for missing file, got %v", lock)
	}
}

func TestFindLockedSchema(t *testing.T) {
	lock := &contracts.LockFile{
		Version: "1",
		Schemas: []contracts.LockedSchema{
			{Module: "users", Version: "1.0.0"},
			{Module: "orders", Version: "2.0.0"},
		},
	}

	// Found
	s := FindLockedSchema(lock, "orders")
	if s == nil {
		t.Fatal("expected to find 'orders'")
	}
	if s.Version != "2.0.0" {
		t.Errorf("version = %q, want %q", s.Version, "2.0.0")
	}

	// Not found
	s = FindLockedSchema(lock, "missing")
	if s != nil {
		t.Errorf("expected nil for missing module, got %v", s)
	}

	// Nil lock
	s = FindLockedSchema(nil, "users")
	if s != nil {
		t.Errorf("expected nil for nil lock, got %v", s)
	}
}

func TestSyntheticDataSet(t *testing.T) {
	locked := contracts.LockedSchema{
		Module:  "users",
		Version: "1.2.3",
		Format:  "parquet",
	}

	ds := SyntheticDataSet(locked)
	if ds.Metadata.Name != "users" {
		t.Errorf("name = %q, want %q", ds.Metadata.Name, "users")
	}
	if ds.Metadata.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", ds.Metadata.Version, "1.2.3")
	}
	if ds.Spec.Format != "parquet" {
		t.Errorf("format = %q, want %q", ds.Spec.Format, "parquet")
	}
	if ds.Spec.SchemaRef != "users@1.2.3" {
		t.Errorf("schemaRef = %q, want %q", ds.Spec.SchemaRef, "users@1.2.3")
	}
	if ds.Metadata.Labels["schema.source"] != "lock" {
		t.Errorf("missing schema.source=lock label")
	}
}

func TestConverterModuleToSchemaFields(t *testing.T) {
	fields := []FieldDef{
		{Name: "id", Type: "INT64"},
		{Name: "name", Type: "UTF8"},
		{Name: "email", Type: "string", PII: true},
		{Name: "created_at", Type: "TIMESTAMP_MILLIS"},
		{Name: "active", Type: "BOOLEAN"},
		{Name: "score", Type: "DOUBLE"},
	}

	result := ModuleToSchemaFields(fields)
	if len(result) != 6 {
		t.Fatalf("len = %d, want 6", len(result))
	}

	expected := []struct {
		name string
		typ  string
		pii  bool
	}{
		{"id", "integer", false},
		{"name", "string", false},
		{"email", "string", true},
		{"created_at", "timestamp", false},
		{"active", "boolean", false},
		{"score", "float", false},
	}

	for i, exp := range expected {
		if result[i].Name != exp.name {
			t.Errorf("[%d] name = %q, want %q", i, result[i].Name, exp.name)
		}
		if result[i].Type != exp.typ {
			t.Errorf("[%d] type = %q, want %q", i, result[i].Type, exp.typ)
		}
		if result[i].PII != exp.pii {
			t.Errorf("[%d] pii = %v, want %v", i, result[i].PII, exp.pii)
		}
	}
}
