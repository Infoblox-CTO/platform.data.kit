package runner

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// StoreEnv holds the resolved environment variables for a store.
type StoreEnv struct {
	Name string // logical store name (e.g., "warehouse")
	Type string // connector type (e.g., "postgres", "s3", "kafka")
	DSN  string // connection string
}

// BuildStoreEnvVars builds DK_STORE_DSN_{NAME} and DK_STORE_TYPE_{NAME} env vars
// for each store referenced by the transform's inputs and outputs.
func BuildStoreEnvVars(t *contracts.Transform, stores map[string]*contracts.Store, datasets map[string]*contracts.DataSetManifest) ([]StoreEnv, error) {
	// Collect unique store names from all datasets referenced by the transform.
	storeNames := make(map[string]bool)
	for _, ref := range t.Spec.Inputs {
		ds, ok := datasets[ref.DataSet]
		if !ok {
			continue
		}
		if ds.Spec.Store != "" {
			storeNames[ds.Spec.Store] = true
		}
	}
	for _, ref := range t.Spec.Outputs {
		ds, ok := datasets[ref.DataSet]
		if !ok {
			continue
		}
		if ds.Spec.Store != "" {
			storeNames[ds.Spec.Store] = true
		}
	}

	var result []StoreEnv
	for name := range storeNames {
		store, ok := stores[name]
		if !ok {
			return nil, fmt.Errorf("store %q referenced by dataset but not found", name)
		}

		dsn, err := BuildDSN(store)
		if err != nil {
			return nil, fmt.Errorf("building DSN for store %q: %w", name, err)
		}

		result = append(result, StoreEnv{
			Name: name,
			Type: store.Spec.Connector,
			DSN:  dsn,
		})
	}

	return result, nil
}

// EnvVarName returns the uppercased, underscore-separated env var name for a store.
// e.g., "pg-warehouse" → "PG_WAREHOUSE"
func EnvVarName(storeName string) string {
	s := strings.ToUpper(storeName)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

// ToEnvMap converts StoreEnv slices to a map suitable for RunOptions.Env.
func StoreEnvsToMap(envs []StoreEnv) map[string]string {
	m := make(map[string]string, len(envs)*2)
	for _, e := range envs {
		name := EnvVarName(e.Name)
		m["DK_STORE_DSN_"+name] = e.DSN
		m["DK_STORE_TYPE_"+name] = e.Type
	}
	return m
}

// BuildDSN constructs a connection string from a Store's connection map and secrets.
// If connection_string is already present, it's returned directly (with secret interpolation).
// Otherwise, a DSN is assembled from the connector type and connection fields.
func BuildDSN(store *contracts.Store) (string, error) {
	conn := store.Spec.Connection
	secrets := store.Spec.Secrets

	// If connection_string is provided directly, use it.
	if cs, ok := conn["connection_string"]; ok {
		return interpolateSecrets(fmt.Sprintf("%v", cs), secrets), nil
	}

	switch store.Spec.Connector {
	case "postgres", "postgresql":
		return buildPostgresDSN(conn, secrets), nil
	case "s3":
		return buildS3DSN(conn), nil
	case "kafka":
		return buildKafkaDSN(conn), nil
	default:
		// For unknown connectors, try connection_string or serialize as query params.
		return buildGenericDSN(store.Spec.Connector, conn, secrets), nil
	}
}

func buildPostgresDSN(conn map[string]any, secrets map[string]string) string {
	host := getStr(conn, "host", "localhost")
	port := getStr(conn, "port", "5432")
	dbname := getStr(conn, "dbname", getStr(conn, "database", ""))
	user := getStr(conn, "user", getStr(conn, "username", ""))
	password := getSecretStr(secrets, "password", getStr(conn, "password", ""))
	sslmode := getStr(conn, "sslmode", "disable")

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
	return interpolateSecrets(dsn, secrets)
}

func buildS3DSN(conn map[string]any) string {
	bucket := getStr(conn, "bucket", "")
	region := getStr(conn, "region", "")
	endpoint := getStr(conn, "endpoint", "")

	u := &url.URL{
		Scheme: "s3",
		Host:   bucket,
	}
	q := u.Query()
	if region != "" {
		q.Set("region", region)
	}
	if endpoint != "" {
		q.Set("endpoint", endpoint)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func buildKafkaDSN(conn map[string]any) string {
	brokers := getStr(conn, "bootstrapServers", getStr(conn, "brokers", "localhost:9092"))
	topic := getStr(conn, "topic", "")

	u := &url.URL{
		Scheme: "kafka",
		Host:   brokers,
		Path:   "/" + topic,
	}
	return u.String()
}

func buildGenericDSN(connector string, conn map[string]any, secrets map[string]string) string {
	u := &url.URL{
		Scheme: connector,
	}
	q := u.Query()
	for k, v := range conn {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	u.RawQuery = q.Encode()
	return interpolateSecrets(u.String(), secrets)
}

// interpolateSecrets replaces ${VAR} references in a string with secret values.
func interpolateSecrets(s string, secrets map[string]string) string {
	for k, v := range secrets {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

// getStr retrieves a string value from a map[string]any, with a default.
func getStr(m map[string]any, key, def string) string {
	v, ok := m[key]
	if !ok {
		return def
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

// getSecretStr retrieves a string value from a map[string]string, with a default.
func getSecretStr(m map[string]string, key, def string) string {
	v, ok := m[key]
	if !ok {
		return def
	}
	return v
}
