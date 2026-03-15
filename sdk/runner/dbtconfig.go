package runner

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// dbtProfilesOutput represents a single output target in profiles.yml.
type dbtProfilesOutput struct {
	Type    string `yaml:"type"`
	Host    string `yaml:"host,omitempty"`
	Port    int    `yaml:"port,omitempty"`
	User    string `yaml:"user,omitempty"`
	Pass    string `yaml:"pass,omitempty"`
	DBName  string `yaml:"dbname,omitempty"`
	Schema  string `yaml:"schema,omitempty"`
	Threads int    `yaml:"threads"`
}

// dbtProfile represents a profile entry in profiles.yml.
type dbtProfile struct {
	Target  string                        `yaml:"target"`
	Outputs map[string]*dbtProfilesOutput `yaml:"outputs"`
}

// GenerateDBTProfiles generates a profiles.yml from the transform's store graph.
// It resolves Transform → DataSet → Store, extracts the DSN, and produces a
// profiles.yml that dbt can use directly (no env_var() needed).
//
// For dbt, all inputs/outputs must reference the same Store (dbt connects to
// one database per run).
func GenerateDBTProfiles(t *contracts.Transform, packageDir string, cellResolver *CellResolver) ([]byte, error) {
	pm, err := loadPackageManifests(packageDir)
	if err != nil {
		return nil, fmt.Errorf("loading package manifests: %w", err)
	}

	// Find the single store for this dbt transform.
	store, err := resolveDBTStore(t, pm, cellResolver)
	if err != nil {
		return nil, err
	}

	dsn, err := BuildDSN(store)
	if err != nil {
		return nil, fmt.Errorf("building DSN for store: %w", err)
	}

	output, err := dsnToDBTOutput(store.Spec.Connector, dsn)
	if err != nil {
		return nil, fmt.Errorf("converting DSN to dbt output: %w", err)
	}

	profileName := strings.ReplaceAll(t.GetName(), "-", "_")
	profiles := map[string]*dbtProfile{
		profileName: {
			Target: "dk",
			Outputs: map[string]*dbtProfilesOutput{
				"dk": output,
			},
		},
	}

	data, err := yaml.Marshal(profiles)
	if err != nil {
		return nil, fmt.Errorf("marshaling profiles.yml: %w", err)
	}

	return data, nil
}

// WriteDBTProfiles generates and writes profiles.yml to the package directory.
func WriteDBTProfiles(t *contracts.Transform, packageDir string, cellResolver *CellResolver) (string, error) {
	data, err := GenerateDBTProfiles(t, packageDir, cellResolver)
	if err != nil {
		return "", err
	}

	profilesPath := filepath.Join(packageDir, "profiles.yml")
	if err := os.WriteFile(profilesPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing profiles.yml: %w", err)
	}

	return profilesPath, nil
}

// resolveDBTStore finds the single store that all datasets in a dbt transform reference.
// dbt can only connect to one database, so all datasets must share the same store.
func resolveDBTStore(t *contracts.Transform, pm *packageManifests, cellResolver *CellResolver) (*contracts.Store, error) {
	storeNames := make(map[string]bool)

	allRefs := append(t.Spec.Inputs, t.Spec.Outputs...)
	for _, ref := range allRefs {
		ds, ok := pm.DataSets[ref.DataSet]
		if !ok {
			continue
		}
		if ds.Spec.Store != "" {
			storeNames[ds.Spec.Store] = true
		}
	}

	if len(storeNames) == 0 {
		return nil, fmt.Errorf("no stores found: dbt transforms require datasets with spec.store set")
	}
	if len(storeNames) > 1 {
		names := make([]string, 0, len(storeNames))
		for n := range storeNames {
			names = append(names, n)
		}
		return nil, fmt.Errorf("dbt transforms must use a single store, found %d: %s", len(storeNames), strings.Join(names, ", "))
	}

	var storeName string
	for n := range storeNames {
		storeName = n
	}

	// Resolve from cell if available, otherwise from package store/ dir.
	if cellResolver != nil {
		return cellResolver.ResolveStore(nil, storeName)
	}

	store, ok := pm.Stores[storeName]
	if !ok {
		return nil, fmt.Errorf("store %q not found in store/ directory", storeName)
	}
	return store, nil
}

// dsnToDBTOutput parses a DSN and converts it to a dbt profiles output config.
func dsnToDBTOutput(connectorType, dsn string) (*dbtProfilesOutput, error) {
	switch connectorType {
	case "postgres", "postgresql":
		return parsePostgresDSN(dsn)
	default:
		return nil, fmt.Errorf("dbt does not support connector type %q — only postgres is supported", connectorType)
	}
}

// parsePostgresDSN parses a postgresql:// DSN into dbt profile fields.
func parsePostgresDSN(dsn string) (*dbtProfilesOutput, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}

	host := u.Hostname()
	port := 5432
	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &port)
	}

	dbname := strings.TrimPrefix(u.Path, "/")
	user := u.User.Username()
	pass, _ := u.User.Password()

	schema := "public"
	if s := u.Query().Get("schema"); s != "" {
		schema = s
	}

	return &dbtProfilesOutput{
		Type:    "postgres",
		Host:    host,
		Port:    port,
		User:    user,
		Pass:    pass,
		DBName:  dbname,
		Schema:  schema,
		Threads: 4,
	}, nil
}
