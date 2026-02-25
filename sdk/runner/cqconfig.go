package runner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// CloudQuery config auto-generation from the manifest graph:
//   Transform → Asset → Store → Connector
// ---------------------------------------------------------------------------

// packageManifests holds all manifests resolved from a package directory.
type packageManifests struct {
	Connectors map[string]*contracts.Connector     // keyed by metadata.name
	Stores     map[string]*contracts.Store         // keyed by metadata.name
	Assets     map[string]*contracts.AssetManifest // keyed by metadata.name
}

// loadPackageManifests scans the standard subdirectories (connector/, store/,
// asset/) under packageDir and parses every *.yaml / *.yml file found.
func loadPackageManifests(packageDir string) (*packageManifests, error) {
	pm := &packageManifests{
		Connectors: make(map[string]*contracts.Connector),
		Stores:     make(map[string]*contracts.Store),
		Assets:     make(map[string]*contracts.AssetManifest),
	}

	subdirs := map[string]func([]byte) error{
		"connector": func(data []byte) error {
			c, err := manifest.NewParser().ParseConnector(data)
			if err != nil {
				return err
			}
			pm.Connectors[c.Metadata.Name] = c
			return nil
		},
		"store": func(data []byte) error {
			s, err := manifest.NewParser().ParseStore(data)
			if err != nil {
				return err
			}
			pm.Stores[s.Metadata.Name] = s
			return nil
		},
		"asset": func(data []byte) error {
			a, err := manifest.NewParser().ParseAsset(data)
			if err != nil {
				return err
			}
			pm.Assets[a.Metadata.Name] = a
			return nil
		},
	}

	for dir, parseFn := range subdirs {
		dirPath := filepath.Join(packageDir, dir)
		entries, err := os.ReadDir(dirPath)
		if os.IsNotExist(err) {
			continue // subdirectory is optional
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s/: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dirPath, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading %s/%s: %w", dir, entry.Name(), err)
			}
			if err := parseFn(data); err != nil {
				return nil, fmt.Errorf("parsing %s/%s: %w", dir, entry.Name(), err)
			}
		}
	}

	return pm, nil
}

// cqConfigDoc mirrors the CloudQuery multi-document YAML structure.
type cqConfigDoc struct {
	Kind string       `yaml:"kind"`
	Spec cqConfigSpec `yaml:"spec"`
}

type cqConfigSpec struct {
	Name         string         `yaml:"name"`
	Registry     string         `yaml:"registry"`
	Path         string         `yaml:"path"`
	Tables       []string       `yaml:"tables,omitempty"`
	Destinations []string       `yaml:"destinations,omitempty"`
	WriteMode    string         `yaml:"write_mode,omitempty"`
	Spec         map[string]any `yaml:"spec,omitempty"`
}

// generateCQConfig resolves the Transform → Asset → Store → Connector graph
// and produces CloudQuery config.yaml bytes plus the list of plugins that
// need to run as sidecar containers.
//
// The generated config uses `registry: docker` with OCI image paths from
// the Connector's Plugin field. The caller (runCloudQuery) rewrites these
// to `registry: grpc` / `localhost:<port>` before deploying.
func generateCQConfig(t *contracts.Transform, packageDir string) ([]byte, []cqPlugin, error) {
	pm, err := loadPackageManifests(packageDir)
	if err != nil {
		return nil, nil, fmt.Errorf("loading package manifests: %w", err)
	}

	// Resolve input chain: Asset → Store → Connector.
	// Group inputs by (Connector, Store) pair → one CQ source doc each.
	type sourceKey struct {
		connectorName string
		storeName     string
	}
	type sourceInfo struct {
		connector *contracts.Connector
		store     *contracts.Store
		tables    []string // from Asset.Table / Asset.Topic
	}
	sources := make(map[sourceKey]*sourceInfo)
	var sourceOrder []sourceKey // preserve deterministic order

	for _, ref := range t.Spec.Inputs {
		asset, ok := pm.Assets[ref.Asset]
		if !ok {
			return nil, nil, fmt.Errorf("input asset %q not found in asset/ directory", ref.Asset)
		}
		store, ok := pm.Stores[asset.Spec.Store]
		if !ok {
			return nil, nil, fmt.Errorf("store %q (referenced by asset %q) not found in store/ directory", asset.Spec.Store, ref.Asset)
		}
		conn, ok := pm.Connectors[store.Spec.Connector]
		if !ok {
			return nil, nil, fmt.Errorf("connector %q (referenced by store %q) not found in connector/ directory", store.Spec.Connector, asset.Spec.Store)
		}
		if conn.Spec.Plugin == nil || conn.Spec.Plugin.Source == "" {
			return nil, nil, fmt.Errorf("connector %q has no source plugin image", conn.Metadata.Name)
		}

		key := sourceKey{connectorName: conn.Metadata.Name, storeName: store.Metadata.Name}
		si, exists := sources[key]
		if !exists {
			si = &sourceInfo{connector: conn, store: store}
			sources[key] = si
			sourceOrder = append(sourceOrder, key)
		}

		// Determine the "table" identifier from the Asset.
		table := asset.Spec.Table
		if table == "" {
			table = asset.Spec.Topic
		}
		if table == "" {
			table = asset.Spec.Prefix
		}
		if table == "" {
			table = asset.Metadata.Name
		}
		si.tables = append(si.tables, table)
	}

	// Resolve output chain: Asset → Store → Connector.
	// Group outputs by (Connector, Store) pair → one CQ destination doc each.
	type destKey struct {
		connectorName string
		storeName     string
	}
	type destInfo struct {
		connector *contracts.Connector
		store     *contracts.Store
		assets    []*contracts.AssetManifest
	}
	destinations := make(map[destKey]*destInfo)
	var destOrder []destKey

	for _, ref := range t.Spec.Outputs {
		asset, ok := pm.Assets[ref.Asset]
		if !ok {
			return nil, nil, fmt.Errorf("output asset %q not found in asset/ directory", ref.Asset)
		}
		store, ok := pm.Stores[asset.Spec.Store]
		if !ok {
			return nil, nil, fmt.Errorf("store %q (referenced by asset %q) not found in store/ directory", asset.Spec.Store, ref.Asset)
		}
		conn, ok := pm.Connectors[store.Spec.Connector]
		if !ok {
			return nil, nil, fmt.Errorf("connector %q (referenced by store %q) not found in connector/ directory", store.Spec.Connector, asset.Spec.Store)
		}
		if conn.Spec.Plugin == nil || conn.Spec.Plugin.Destination == "" {
			return nil, nil, fmt.Errorf("connector %q has no destination plugin image", conn.Metadata.Name)
		}

		key := destKey{connectorName: conn.Metadata.Name, storeName: store.Metadata.Name}
		di, exists := destinations[key]
		if !exists {
			di = &destInfo{connector: conn, store: store}
			destinations[key] = di
			destOrder = append(destOrder, key)
		}
		di.assets = append(di.assets, asset)
	}

	// Build destination names list for source docs' `destinations` field.
	destNames := make([]string, 0, len(destOrder))
	for _, key := range destOrder {
		destNames = append(destNames, key.connectorName)
	}

	// Assemble CloudQuery config documents and plugin list.
	var docs []cqConfigDoc
	var plugins []cqPlugin
	port := 7777

	// Source documents.
	for _, key := range sourceOrder {
		si := sources[key]
		specMap := buildPluginSpec(si.store)

		doc := cqConfigDoc{
			Kind: "source",
			Spec: cqConfigSpec{
				Name:         si.connector.Metadata.Name,
				Registry:     "docker",
				Path:         si.connector.Spec.Plugin.Source,
				Tables:       si.tables,
				Destinations: destNames,
				Spec:         specMap,
			},
		}
		docs = append(docs, doc)
		plugins = append(plugins, cqPlugin{
			Kind:  "source",
			Name:  si.connector.Metadata.Name,
			Image: si.connector.Spec.Plugin.Source,
			Port:  port,
		})
		port++
	}

	// Destination documents.
	for _, key := range destOrder {
		di := destinations[key]
		specMap := buildPluginSpec(di.store)

		// Merge asset-level overrides into the destination spec.
		// Use the first asset's properties for path/format since CQ
		// destination plugins apply these globally.
		if len(di.assets) > 0 {
			a := di.assets[0]
			if a.Spec.Prefix != "" {
				specMap["path"] = a.Spec.Prefix
			}
			if a.Spec.Format != "" {
				specMap["format"] = a.Spec.Format
			}
		}

		doc := cqConfigDoc{
			Kind: "destination",
			Spec: cqConfigSpec{
				Name:      di.connector.Metadata.Name,
				Registry:  "docker",
				Path:      di.connector.Spec.Plugin.Destination,
				WriteMode: "append",
				Spec:      specMap,
			},
		}
		docs = append(docs, doc)
		plugins = append(plugins, cqPlugin{
			Kind:  "destination",
			Name:  di.connector.Metadata.Name,
			Image: di.connector.Spec.Plugin.Destination,
			Port:  port,
		})
		port++
	}

	// Encode to multi-document YAML.
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	for _, doc := range docs {
		if err := encoder.Encode(doc); err != nil {
			return nil, nil, fmt.Errorf("encoding CQ config: %w", err)
		}
	}
	encoder.Close()

	return buf.Bytes(), plugins, nil
}

// buildPluginSpec merges a Store's Connection and Secrets maps into a single
// map suitable for the CloudQuery plugin `spec` section.
func buildPluginSpec(store *contracts.Store) map[string]any {
	specMap := make(map[string]any, len(store.Spec.Connection)+len(store.Spec.Secrets))
	for k, v := range store.Spec.Connection {
		specMap[k] = v
	}
	// Secrets overlay connection values. They may use ${VAR} interpolation
	// which will be resolved by the k8s env injection at runtime.
	for k, v := range store.Spec.Secrets {
		specMap[k] = v
	}
	return specMap
}
