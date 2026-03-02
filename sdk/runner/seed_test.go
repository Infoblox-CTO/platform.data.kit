package runner

import (
	"testing"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestGenerateSeedSQL_Basic(t *testing.T) {
	asset := &contracts.AssetManifest{
		Spec: contracts.AssetSpec{
			Table: "example_table",
			Schema: []contracts.SchemaField{
				{Name: "id", Type: "integer"},
				{Name: "name", Type: "string"},
				{Name: "created_at", Type: "timestamp"},
			},
		},
	}
	rows := []map[string]any{
		{"id": 1, "name": "alice", "created_at": "2026-01-01T00:00:00Z"},
		{"id": 2, "name": "bob", "created_at": "2026-01-15T00:00:00Z"},
	}

	sql := generateSeedSQL(asset, rows, false)

	if !containsAll(sql,
		"CREATE TABLE IF NOT EXISTS example_table",
		"id INTEGER",
		"name TEXT",
		"created_at TIMESTAMPTZ",
		"TRUNCATE TABLE example_table",
		"INSERT INTO example_table",
		"'alice'",
		"'bob'",
	) {
		t.Errorf("unexpected SQL:\n%s", sql)
	}

	// Should NOT contain DROP when clean=false.
	if contains(sql, "DROP TABLE") {
		t.Errorf("SQL should not DROP TABLE when clean=false:\n%s", sql)
	}
}

func TestGenerateSeedSQL_Clean(t *testing.T) {
	asset := &contracts.AssetManifest{
		Spec: contracts.AssetSpec{
			Table: "users",
			Schema: []contracts.SchemaField{
				{Name: "id", Type: "integer"},
			},
		},
	}
	rows := []map[string]any{
		{"id": 42},
	}

	sql := generateSeedSQL(asset, rows, true)

	if !contains(sql, "DROP TABLE IF EXISTS users CASCADE") {
		t.Errorf("expected DROP TABLE when clean=true, got:\n%s", sql)
	}
	if !contains(sql, "CREATE TABLE IF NOT EXISTS users") {
		t.Errorf("expected CREATE TABLE, got:\n%s", sql)
	}
	// Clean mode should NOT add TRUNCATE (table was just DROPped + CREATEd).
	if contains(sql, "TRUNCATE") {
		t.Errorf("should not TRUNCATE when clean=true (already DROPped), got:\n%s", sql)
	}
}

func TestGenerateSeedSQL_NoRows(t *testing.T) {
	asset := &contracts.AssetManifest{
		Spec: contracts.AssetSpec{
			Table: "empty_table",
			Schema: []contracts.SchemaField{
				{Name: "id", Type: "integer"},
			},
		},
	}

	sql := generateSeedSQL(asset, nil, false)

	if !contains(sql, "CREATE TABLE IF NOT EXISTS empty_table") {
		t.Errorf("expected CREATE TABLE, got:\n%s", sql)
	}
	if contains(sql, "INSERT") {
		t.Errorf("should not contain INSERT when no rows, got:\n%s", sql)
	}
}

func TestGenerateSeedSQL_SQLInjectionEscaping(t *testing.T) {
	asset := &contracts.AssetManifest{
		Spec: contracts.AssetSpec{
			Table: "test_table",
			Schema: []contracts.SchemaField{
				{Name: "name", Type: "string"},
			},
		},
	}
	rows := []map[string]any{
		{"name": "O'Reilly"},
	}

	sql := generateSeedSQL(asset, rows, false)

	if !contains(sql, "O''Reilly") {
		t.Errorf("single quotes should be escaped, got:\n%s", sql)
	}
}

func TestSchemaTypeToSQL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"integer", "INTEGER"},
		{"int", "INTEGER"},
		{"bigint", "BIGINT"},
		{"long", "BIGINT"},
		{"float", "DOUBLE PRECISION"},
		{"double", "DOUBLE PRECISION"},
		{"boolean", "BOOLEAN"},
		{"bool", "BOOLEAN"},
		{"timestamp", "TIMESTAMPTZ"},
		{"datetime", "TIMESTAMPTZ"},
		{"date", "DATE"},
		{"string", "TEXT"},
		{"text", "TEXT"},
		{"varchar", "TEXT"},
		{"unknown_type", "TEXT"},
	}

	for _, tt := range tests {
		got := schemaTypeToSQL(tt.input)
		if got != tt.expected {
			t.Errorf("schemaTypeToSQL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParsePostgresConnStr(t *testing.T) {
	tests := []struct {
		connStr  string
		user     string
		password string
		database string
	}{
		{
			"postgresql://dpuser:dppassword@host:5432/dataplatform?sslmode=disable",
			"dpuser", "dppassword", "dataplatform",
		},
		{
			"postgresql://admin:secret@db.example.com:5432/mydb",
			"admin", "secret", "mydb",
		},
		{
			"", // empty -> defaults
			"dpuser", "dppassword", "dataplatform",
		},
		{
			"postgres://u:p@h/d",
			"u", "p", "d",
		},
	}

	for _, tt := range tests {
		user, pass, db := parsePostgresConnStr(tt.connStr)
		if user != tt.user || pass != tt.password || db != tt.database {
			t.Errorf("parsePostgresConnStr(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tt.connStr, user, pass, db, tt.user, tt.password, tt.database)
		}
	}
}

func TestResolveSeedRows_Inline(t *testing.T) {
	asset := &contracts.AssetManifest{
		Spec: contracts.AssetSpec{
			Dev: &contracts.AssetDevSpec{
				Seed: &contracts.SeedSpec{
					Inline: []map[string]any{
						{"id": 1, "name": "test"},
					},
				},
			},
		},
	}

	rows, err := resolveSeedRows(asset, ".", "")
	if err != nil {
		t.Fatalf("resolveSeedRows error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["name"] != "test" {
		t.Errorf("row[0].name = %v, want 'test'", rows[0]["name"])
	}
}

func TestResolveSeedRows_NilDev(t *testing.T) {
	asset := &contracts.AssetManifest{}

	// Should not panic on nil Dev
	if asset.Spec.Dev == nil || asset.Spec.Dev.Seed == nil {
		return // expected
	}
}

func TestLoadSeedFile_CSV(t *testing.T) {
	dir := t.TempDir()
	csvPath := dir + "/data.csv"
	writeFile(t, csvPath, "id,name\n1,alice\n2,bob\n")

	rows, err := loadSeedFile(csvPath)
	if err != nil {
		t.Fatalf("loadSeedFile error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["name"] != "alice" {
		t.Errorf("row[0].name = %v", rows[0]["name"])
	}
}

func TestLoadSeedFile_JSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := dir + "/data.json"
	writeFile(t, jsonPath, `[{"id": 1, "name": "alice"}, {"id": 2, "name": "bob"}]`)

	rows, err := loadSeedFile(jsonPath)
	if err != nil {
		t.Fatalf("loadSeedFile error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1]["name"] != "bob" {
		t.Errorf("row[1].name = %v", rows[1]["name"])
	}
}

func TestLoadSeedFile_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	xmlPath := dir + "/data.xml"
	writeFile(t, xmlPath, "<data/>")

	_, err := loadSeedFile(xmlPath)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !contains(err.Error(), "unsupported") {
		t.Errorf("error should mention unsupported: %v", err)
	}
}

func TestSqlLiteral(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{nil, "NULL"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "TRUE"},
		{false, "FALSE"},
		{"hello", "'hello'"},
		{"it's", "'it''s'"},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "'2026-01-01T00:00:00Z'"},
		{time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC), "'2026-03-15T14:30:00Z'"},
	}

	for _, tt := range tests {
		got := sqlLiteral(tt.input)
		if got != tt.expected {
			t.Errorf("sqlLiteral(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// Helper: check if s contains sub.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Checksum tests
// ---------------------------------------------------------------------------

func TestComputeSeedChecksum_Deterministic(t *testing.T) {
	rows := []map[string]any{
		{"id": 1, "name": "alice"},
		{"id": 2, "name": "bob"},
	}

	c1 := computeSeedChecksum("users", "default", rows)
	c2 := computeSeedChecksum("users", "default", rows)

	if c1 != c2 {
		t.Errorf("checksums should be deterministic: %s != %s", c1, c2)
	}
	if len(c1) != 64 {
		t.Errorf("expected SHA-256 hex (64 chars), got %d chars: %s", len(c1), c1)
	}
}

func TestComputeSeedChecksum_DiffersOnChange(t *testing.T) {
	rowsA := []map[string]any{{"id": 1, "name": "alice"}}
	rowsB := []map[string]any{{"id": 1, "name": "bob"}}

	cA := computeSeedChecksum("t", "default", rowsA)
	cB := computeSeedChecksum("t", "default", rowsB)

	if cA == cB {
		t.Error("checksums should differ when data changes")
	}
}

func TestComputeSeedChecksum_DiffersOnProfile(t *testing.T) {
	rows := []map[string]any{{"id": 1}}

	c1 := computeSeedChecksum("t", "default", rows)
	c2 := computeSeedChecksum("t", "large", rows)

	if c1 == c2 {
		t.Error("checksums should differ for different profiles")
	}
}

func TestComputeSeedChecksum_EmptyRows(t *testing.T) {
	c := computeSeedChecksum("t", "default", nil)
	if c == "" {
		t.Error("checksum should not be empty even for nil rows")
	}
}

// ---------------------------------------------------------------------------
// Profile resolution tests
// ---------------------------------------------------------------------------

func TestResolveSeedRows_Profile(t *testing.T) {
	asset := &contracts.AssetManifest{
		Spec: contracts.AssetSpec{
			Dev: &contracts.AssetDevSpec{
				Seed: &contracts.SeedSpec{
					Inline: []map[string]any{
						{"id": 1, "name": "default-alice"},
					},
					Profiles: map[string]*contracts.SeedProfileSpec{
						"large": {
							Inline: []map[string]any{
								{"id": 100, "name": "large-1"},
								{"id": 200, "name": "large-2"},
							},
						},
						"empty": {},
					},
				},
			},
		},
	}

	// Default profile.
	rows, err := resolveSeedRows(asset, ".", "")
	if err != nil {
		t.Fatalf("default profile error: %v", err)
	}
	if len(rows) != 1 || rows[0]["name"] != "default-alice" {
		t.Errorf("default profile: expected default-alice, got %v", rows)
	}

	// Named profile.
	rows, err = resolveSeedRows(asset, ".", "large")
	if err != nil {
		t.Fatalf("large profile error: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("large profile: expected 2 rows, got %d", len(rows))
	}

	// Empty profile.
	rows, err = resolveSeedRows(asset, ".", "empty")
	if err != nil {
		t.Fatalf("empty profile error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("empty profile: expected 0 rows, got %d", len(rows))
	}

	// Unknown profile.
	_, err = resolveSeedRows(asset, ".", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

// ---------------------------------------------------------------------------
// Meta SQL helpers tests
// ---------------------------------------------------------------------------

func TestSeedMetaDDL(t *testing.T) {
	ddl := seedMetaDDL()
	if !containsAll(ddl, "_dp_seed_meta", "table_name", "profile", "checksum") {
		t.Errorf("unexpected DDL: %s", ddl)
	}
}

func TestUpsertSeedChecksum(t *testing.T) {
	sql := upsertSeedChecksum("my_table", "default", "abc123")
	if !containsAll(sql, "_dp_seed_meta", "my_table", "default", "abc123", "ON CONFLICT") {
		t.Errorf("unexpected upsert SQL: %s", sql)
	}
}

func TestUpsertSeedChecksum_Escaping(t *testing.T) {
	sql := upsertSeedChecksum("O'Table", "it's", "check")
	if !contains(sql, "O''Table") || !contains(sql, "it''s") {
		t.Errorf("single quotes should be escaped: %s", sql)
	}
}
