// Package asset provides asset loading, scaffolding, and schema resolution for data package assets.
package asset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		if a.Name == name {
			return a, nil
		}
	}

	return nil, fmt.Errorf("asset %q not found in %s", name, filepath.Join(projectDir, "assets"))
}

// AssetPath returns the expected filesystem path for an asset based on its type and name.
func AssetPath(projectDir string, assetType contracts.AssetType, name string) string {
	typeDir := contracts.AssetTypeDirName(assetType)
	return filepath.Join(projectDir, "assets", typeDir, name, "asset.yaml")
}

// AssetDir returns the expected directory path for an asset based on its type and name.
func AssetDir(projectDir string, assetType contracts.AssetType, name string) string {
	typeDir := contracts.AssetTypeDirName(assetType)
	return filepath.Join(projectDir, "assets", typeDir, name)
}

// validateDirectoryPlacement checks that an asset's type matches the directory it's in.
func validateDirectoryPlacement(asset *contracts.AssetManifest, relPath string) error {
	// relPath is relative to assets/ dir, e.g., "sources/my-source/asset.yaml"
	parts := strings.SplitN(filepath.ToSlash(relPath), "/", 3)
	if len(parts) < 2 {
		return nil // Can't determine placement, skip check
	}

	dirType := parts[0]
	expectedDir := contracts.AssetTypeDirName(asset.Type)

	if expectedDir != "" && dirType != expectedDir {
		return fmt.Errorf("asset %q has type %q but is in %q directory (expected %q)",
			asset.Name, asset.Type, dirType, expectedDir)
	}

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
