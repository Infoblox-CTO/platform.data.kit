package runner

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// ---------------------------------------------------------------------------
// Dev-seed: create tables and load sample data before a CQ sync.
//
// The seed logic reads Asset.Dev.Seed (inline rows or a file), generates
// CREATE TABLE IF NOT EXISTS + INSERT statements, and executes them against
// the backing PostgreSQL instance via kubectl exec.
//
// Features:
//   - Checksum-based skip: a _dp_seed_meta table tracks the SHA-256 of the
//     seed data; if nothing changed the seed is skipped entirely.
//   - Named profiles: dev.seed.profiles.<name> lets developers maintain
//     multiple data sets (e.g. "large", "edge-cases") and switch between
//     them with --profile.
//   - TRUNCATE + INSERT: when data does change (or on --force / --clean)
//     the table is truncated before inserting, so the contents always match
//     the seed spec exactly.
// ---------------------------------------------------------------------------

// SeedOptions controls the seeding behaviour.
type SeedOptions struct {
	// PackageDir is the root of the DataKit package (contains asset/, store/, etc.).
	PackageDir string
	// Clean drops and recreates tables before inserting seed data.
	Clean bool
	// Force re-seeds even when the checksum indicates data is unchanged.
	Force bool
	// Profile selects a named seed profile. Empty means the default profile.
	Profile string
	// AssetFilter limits seeding to a single asset name. Empty means all.
	AssetFilter string
	// Output receives progress messages. May be nil.
	Output io.Writer
	// KubeContext is the kubectl context to use (defaults to k3d-dk-local).
	KubeContext string
	// Namespace is the k8s namespace where the database pod lives.
	Namespace string
}

// SeedResult reports what was seeded.
type SeedResult struct {
	AssetsSeeded  int
	RowsInserted  int
	AssetsSkipped int // assets whose data was unchanged (checksum match)
}

// SeedPackage reads all input assets in a package and seeds any that have a
// dev.seed section.  Only postgres-backed assets are supported today.
//
// Seeding is idempotent by default: a SHA-256 checksum of the resolved rows
// is compared against the _dp_seed_meta table; if it matches the previous
// run the asset is skipped.  Use Force or Clean to override.
func SeedPackage(ctx context.Context, opts SeedOptions) (*SeedResult, error) {
	if opts.KubeContext == "" {
		clusterName := defaultClusterName
		if cfg, err := loadK3dClusterName(); err == nil && cfg != "" {
			clusterName = cfg
		}
		opts.KubeContext = fmt.Sprintf("k3d-%s", clusterName)
	}
	if opts.Namespace == "" {
		opts.Namespace = defaultNamespace
	}

	pm, err := loadPackageManifests(opts.PackageDir)
	if err != nil {
		return nil, fmt.Errorf("loading package manifests: %w", err)
	}

	result := &SeedResult{}

	for name, asset := range pm.Assets {
		if opts.AssetFilter != "" && name != opts.AssetFilter {
			continue
		}
		if asset.Spec.Dev == nil || asset.Spec.Dev.Seed == nil {
			continue
		}
		if asset.Spec.Table == "" {
			// Only relational (table-based) assets can be seeded today.
			continue
		}

		store, ok := pm.Stores[asset.Spec.Store]
		if !ok {
			return nil, fmt.Errorf("asset %q references store %q which was not found", name, asset.Spec.Store)
		}
		conn, ok := pm.Connectors[store.Spec.Connector]
		if !ok {
			return nil, fmt.Errorf("store %q references connector %q which was not found", asset.Spec.Store, store.Spec.Connector)
		}

		// Only postgres connectors are supported for seeding today.
		if conn.Spec.Type != "postgres" {
			if opts.Output != nil {
				fmt.Fprintf(opts.Output, "Skipping %s: seeding only supports postgres connectors (got %s)\n",
					name, conn.Spec.Type)
			}
			continue
		}

		rows, err := resolveSeedRows(asset, opts.PackageDir, opts.Profile)
		if err != nil {
			return nil, fmt.Errorf("asset %q: resolving seed data: %w", name, err)
		}
		if len(rows) == 0 && !opts.Clean {
			continue
		}

		// Compute checksum over the resolved data so we can skip unchanged seeds.
		profile := opts.Profile
		if profile == "" {
			profile = "default"
		}
		checksum := computeSeedChecksum(asset.Spec.Table, profile, rows)

		// Unless forced or cleaning, check whether the data has changed.
		if !opts.Force && !opts.Clean {
			// Ensure the meta table exists (idempotent).
			ensureMetaSQL := seedMetaDDL()
			_ = execPostgresSQL(ctx, opts, store, ensureMetaSQL) // best-effort

			existing, err := querySeedChecksum(ctx, opts, store, asset.Spec.Table, profile)
			if err == nil && existing == checksum {
				if opts.Output != nil {
					fmt.Fprintf(opts.Output, "Skipping %s (profile=%s): data unchanged\n", name, profile)
				}
				result.AssetsSkipped++
				continue
			}
		}

		sql := generateSeedSQL(asset, rows, opts.Clean)

		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Seeding %s (%s, profile=%s): %d row(s)...\n",
				name, asset.Spec.Table, profile, len(rows))
		}

		if err := execPostgresSQL(ctx, opts, store, sql); err != nil {
			return nil, fmt.Errorf("asset %q: executing seed SQL: %w", name, err)
		}

		// Update the checksum so subsequent runs skip this asset.
		upsertSQL := upsertSeedChecksum(asset.Spec.Table, profile, checksum)
		_ = execPostgresSQL(ctx, opts, store, upsertSQL) // best-effort

		result.AssetsSeeded++
		result.RowsInserted += len(rows)
	}

	return result, nil
}

// resolveSeedRows returns the rows to insert, reading from inline data or a
// file as configured in the asset's dev.seed section.  If profile is non-empty
// and matches a named profile, that profile's data is used instead of the
// default inline/file.
func resolveSeedRows(asset *contracts.AssetManifest, packageDir string, profile string) ([]map[string]any, error) {
	seed := asset.Spec.Dev.Seed

	// If a profile is requested, look it up.
	if profile != "" && seed.Profiles != nil {
		p, ok := seed.Profiles[profile]
		if !ok {
			available := make([]string, 0, len(seed.Profiles))
			for k := range seed.Profiles {
				available = append(available, k)
			}
			sort.Strings(available)
			return nil, fmt.Errorf("profile %q not found (available: %s)", profile, strings.Join(available, ", "))
		}
		if len(p.Inline) > 0 {
			return p.Inline, nil
		}
		if p.File != "" {
			return loadSeedFile(filepath.Join(packageDir, p.File))
		}
		return nil, nil // empty profile (e.g. "empty" profile for testing)
	}

	// Default profile: use top-level inline / file.
	if len(seed.Inline) > 0 {
		return seed.Inline, nil
	}

	if seed.File != "" {
		return loadSeedFile(filepath.Join(packageDir, seed.File))
	}

	return nil, nil
}

// loadSeedFile reads a CSV or JSON seed file and returns rows as maps.
func loadSeedFile(path string) ([]map[string]any, error) {
	ext := strings.ToLower(filepath.Ext(path))

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	switch ext {
	case ".json":
		var rows []map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			return nil, fmt.Errorf("parsing JSON %s: %w", path, err)
		}
		return rows, nil

	case ".csv":
		return parseCSV(data)

	default:
		return nil, fmt.Errorf("unsupported seed file format %q (use .json or .csv)", ext)
	}
}

// parseCSV reads CSV bytes into a slice of column→value maps.
func parseCSV(data []byte) ([]map[string]any, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	var rows []map[string]any
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}
		row := make(map[string]any, len(headers))
		for i, h := range headers {
			if i < len(record) {
				row[h] = record[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// schemaTypeToSQL maps Asset schema types to PostgreSQL column types.
func schemaTypeToSQL(t string) string {
	switch strings.ToLower(t) {
	case "integer", "int":
		return "INTEGER"
	case "bigint", "long":
		return "BIGINT"
	case "float", "double", "number":
		return "DOUBLE PRECISION"
	case "boolean", "bool":
		return "BOOLEAN"
	case "timestamp", "datetime":
		return "TIMESTAMPTZ"
	case "date":
		return "DATE"
	case "string", "text", "varchar":
		return "TEXT"
	default:
		return "TEXT"
	}
}

// generateSeedSQL produces DDL + DML SQL for creating/populating a table.
// When clean=true, the table is DROPped first.  Otherwise the table is
// TRUNCATEd before inserting so that re-runs are idempotent and the table
// contents always match the seed spec exactly (no stale rows, no duplicates).
func generateSeedSQL(asset *contracts.AssetManifest, rows []map[string]any, clean bool) string {
	table := asset.Spec.Table
	var sb strings.Builder

	if clean {
		fmt.Fprintf(&sb, "DROP TABLE IF EXISTS %s CASCADE;\n", table)
	}

	// CREATE TABLE IF NOT EXISTS from the asset schema.
	if len(asset.Spec.Schema) > 0 {
		fmt.Fprintf(&sb, "CREATE TABLE IF NOT EXISTS %s (\n", table)
		for i, col := range asset.Spec.Schema {
			sqlType := schemaTypeToSQL(col.Type)
			fmt.Fprintf(&sb, "  %s %s", col.Name, sqlType)
			if i < len(asset.Spec.Schema)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString(");\n")
	}

	if len(rows) == 0 {
		return sb.String()
	}

	// TRUNCATE so re-runs never produce duplicates or leave stale data.
	if !clean {
		fmt.Fprintf(&sb, "TRUNCATE TABLE %s;\n", table)
	}

	// Determine column order from the first row.
	columns := sortedKeys(rows[0])

	fmt.Fprintf(&sb, "INSERT INTO %s (%s) VALUES\n", table, strings.Join(columns, ", "))
	for i, row := range rows {
		sb.WriteString("  (")
		for j, col := range columns {
			if j > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(sqlLiteral(row[col]))
		}
		sb.WriteString(")")
		if i < len(rows)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(";\n")

	return sb.String()
}

// sqlLiteral formats a Go value as a SQL literal.
func sqlLiteral(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	case time.Time:
		// Format as ISO 8601 so PostgreSQL TIMESTAMPTZ columns parse
		// the value unambiguously (date + time + timezone).
		return fmt.Sprintf("'%s'", val.Format(time.RFC3339))
	default:
		// Escape single quotes for string values.
		s := fmt.Sprintf("%v", val)
		s = strings.ReplaceAll(s, "'", "''")
		return fmt.Sprintf("'%s'", s)
	}
}

// sortedKeys returns the keys of a map in sorted order (deterministic SQL).
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// execPostgresSQL runs a SQL string against the PostgreSQL pod in the k3d
// cluster via kubectl exec.
func execPostgresSQL(ctx context.Context, opts SeedOptions, store *contracts.Store, sql string) error {
	// Extract database and credentials from the store's connection_string.
	connStr, _ := store.Spec.Connection["connection_string"].(string)

	// Default pod label selector for the postgresql statefulset.
	podSelector := "app.kubernetes.io/name=postgresql"

	// Parse user/password/database from the connection string.
	user, password, database := parsePostgresConnStr(connStr)

	// Find the pod.
	podCmd := exec.CommandContext(ctx, "kubectl", "--context", opts.KubeContext,
		"get", "pod", "-n", opts.Namespace,
		"-l", podSelector,
		"-o", "jsonpath={.items[0].metadata.name}")
	podOut, err := podCmd.Output()
	if err != nil {
		return fmt.Errorf("finding postgres pod: %w", err)
	}
	podName := strings.TrimSpace(string(podOut))
	if podName == "" {
		return fmt.Errorf("no postgres pod found in namespace %s", opts.Namespace)
	}

	// Execute SQL via kubectl exec + psql.
	args := []string{
		"--context", opts.KubeContext,
		"exec", "-n", opts.Namespace, podName, "--",
		"env", fmt.Sprintf("PGPASSWORD=%s", password),
		"psql", "-U", user, "-d", database, "-c", sql,
	}
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql exec failed: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// parsePostgresConnStr extracts user, password, and database from a
// postgresql:// connection string.  Returns defaults if parsing fails.
func parsePostgresConnStr(connStr string) (user, password, database string) {
	user = "dkuser"
	password = "dkpassword"
	database = "datakit"

	if connStr == "" {
		return
	}

	// postgresql://user:pass@host:port/dbname?params
	s := connStr
	s = strings.TrimPrefix(s, "postgresql://")
	s = strings.TrimPrefix(s, "postgres://")

	// Split on @ to separate credentials from host.
	if idx := strings.Index(s, "@"); idx >= 0 {
		creds := s[:idx]
		rest := s[idx+1:]

		parts := strings.SplitN(creds, ":", 2)
		if len(parts) >= 1 && parts[0] != "" {
			user = parts[0]
		}
		if len(parts) >= 2 && parts[1] != "" {
			password = parts[1]
		}

		// Extract database from the path after host:port.
		if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
			dbPart := rest[slashIdx+1:]
			if qIdx := strings.Index(dbPart, "?"); qIdx >= 0 {
				dbPart = dbPart[:qIdx]
			}
			if dbPart != "" {
				database = dbPart
			}
		}
	}

	return
}

// ---------------------------------------------------------------------------
// Checksum tracking: _dp_seed_meta table
// ---------------------------------------------------------------------------

// seedMetaDDL returns the DDL to create the checksum-tracking table.
func seedMetaDDL() string {
	return `CREATE TABLE IF NOT EXISTS _dp_seed_meta (
  table_name TEXT NOT NULL,
  profile    TEXT NOT NULL,
  checksum   TEXT NOT NULL,
  seeded_at  TIMESTAMP DEFAULT NOW(),
  PRIMARY KEY (table_name, profile)
);`
}

// computeSeedChecksum produces a deterministic SHA-256 hex digest of the
// resolved seed data so that unchanged seeds can be skipped.
func computeSeedChecksum(table, profile string, rows []map[string]any) string {
	h := sha256.New()
	fmt.Fprintf(h, "table=%s\nprofile=%s\n", table, profile)
	for _, row := range rows {
		keys := sortedKeys(row)
		for _, k := range keys {
			fmt.Fprintf(h, "%s=%v\n", k, row[k])
		}
		h.Write([]byte("---\n"))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// querySeedChecksum retrieves the stored checksum for a table+profile pair.
// Returns ("", error) when no row exists or the query fails.
func querySeedChecksum(ctx context.Context, opts SeedOptions, store *contracts.Store, table, profile string) (string, error) {
	sql := fmt.Sprintf(
		"SELECT checksum FROM _dp_seed_meta WHERE table_name = '%s' AND profile = '%s';",
		strings.ReplaceAll(table, "'", "''"),
		strings.ReplaceAll(profile, "'", "''"),
	)

	connStr, _ := store.Spec.Connection["connection_string"].(string)
	user, password, database := parsePostgresConnStr(connStr)
	podSelector := "app.kubernetes.io/name=postgresql"

	podCmd := exec.CommandContext(ctx, "kubectl", "--context", opts.KubeContext,
		"get", "pod", "-n", opts.Namespace,
		"-l", podSelector,
		"-o", "jsonpath={.items[0].metadata.name}")
	podOut, err := podCmd.Output()
	if err != nil {
		return "", fmt.Errorf("finding postgres pod: %w", err)
	}
	podName := strings.TrimSpace(string(podOut))
	if podName == "" {
		return "", fmt.Errorf("no postgres pod found")
	}

	args := []string{
		"--context", opts.KubeContext,
		"exec", "-n", opts.Namespace, podName, "--",
		"env", fmt.Sprintf("PGPASSWORD=%s", password),
		"psql", "-U", user, "-d", database, "-t", "-A", "-c", sql,
	}
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("query checksum failed: %w", err)
	}
	checksum := strings.TrimSpace(string(out))
	if checksum == "" {
		return "", fmt.Errorf("no checksum found")
	}
	return checksum, nil
}

// upsertSeedChecksum returns SQL that inserts or updates the checksum for a
// table+profile pair.
func upsertSeedChecksum(table, profile, checksum string) string {
	return fmt.Sprintf(
		`INSERT INTO _dp_seed_meta (table_name, profile, checksum, seeded_at)
VALUES ('%s', '%s', '%s', NOW())
ON CONFLICT (table_name, profile) DO UPDATE SET checksum = EXCLUDED.checksum, seeded_at = NOW();`,
		strings.ReplaceAll(table, "'", "''"),
		strings.ReplaceAll(profile, "'", "''"),
		strings.ReplaceAll(checksum, "'", "''"),
	)
}
