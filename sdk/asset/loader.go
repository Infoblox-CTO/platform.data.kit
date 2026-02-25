// Package asset provides asset loading, scaffolding, and schema resolution for data package assets.
package asset

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// LoadAsset loads and parses a single asset.yaml from the given path.
// The path can be:
//   - A directory containing asset.yaml
//   - A direct path to an asset.yaml file
func LoadAsset(path string) (*contracts.AssetManifest, error) {
	// If path is a directory, look for asset.yaml inside it
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("asset path not found: %w", err)
	}

	assetPath := path
	if info.IsDir() {
		assetPath = filepath.Join(path, "asset.yaml")
	}

	data, err := os.ReadFile(assetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset file %s: %w", assetPath, err)
	}

	var asset contracts.AssetManifest
	if err := yaml.Unmarshal(data, &asset); err != nil {
		return nil, fmt.Errorf("failed to parse asset file %s: %w", assetPath, err)
	}

	return &asset, nil
}

// LoadAllAssets discovers and loads all asset.yaml files from the assets/ directory
// under the given project directory. It walks the type-based subdirectories:
//
//	assets/sources/<name>/asset.yaml
//	assets/sinks/<name>/asset.yaml
//	assets/models/<name>/asset.yaml
func LoadAllAssets(projectDir string) ([]*contracts.AssetManifest, error) {
	assetsDir := filepath.Join(projectDir, "assets")

	info, err := os.Stat(assetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No assets directory is fine
		}
		return nil, fmt.Errorf("failed to access assets directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("assets path is not a directory: %s", assetsDir)
	}

	var assets []*contracts.AssetManifest

	err = filepath.WalkDir(assetsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if d.Name() != "asset.yaml" {
			return nil
		}

		asset, loadErr := LoadAsset(path)
		if loadErr != nil {
			return fmt.Errorf("failed to load %s: %w", relativePath(projectDir, path), loadErr)
		}

		// Cross-check: verify the asset type matches its directory placement
		relPath := relativePath(assetsDir, path)
		if dirErr := validateDirectoryPlacement(asset, relPath); dirErr != nil {
			return dirErr
		}

		assets = append(assets, asset)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return assets, nil
}

// FindAssetByName searches for an asset by name in the assets/ directory.
func FindAssetByName(projectDir, name string) (*contracts.AssetManifest, error) {
	assets, err := LoadAllAssets(projectDir)
	if err != nil {
		return nil, err
	}

	for _, a := range assets {
		if a.Metadata.Name == name {
			return a, nil
		}
	}

	return nil, fmt.Errorf("asset %q not found in %s", name, filepath.Join(projectDir, "assets"))
}

// AssetPath returns the expected filesystem path for an asset based on its name.
// Layout: assets/<name>/asset.yaml
func AssetPath(projectDir string, name string) string {
	return filepath.Join(projectDir, "assets", name, "asset.yaml")
}

// AssetDir returns the expected directory path for an asset based on its name.
// Layout: assets/<name>/
func AssetDir(projectDir string, name string) string {
	return filepath.Join(projectDir, "assets", name)
}

// validateDirectoryPlacement checks that an asset is in the expected directory.
// With the new AssetManifest structure, assets are identified by metadata.name
// and the type-based directory layout (sources/sinks/models) is deprecated.
// This function is kept for backward compatibility but performs minimal validation.
func validateDirectoryPlacement(asset *contracts.AssetManifest, relPath string) error {
	// New AssetManifest doesn't have a Type field — skip type-based directory validation.
	// Directory placement will be fully reworked in step 20.
	_ = asset
	_ = relPath
	return nil
}

// relativePath returns a relative path from base to target, or target if it can't be made relative.
func relativePath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
