package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// AssetFromBytes parses YAML bytes into an AssetManifest.
func AssetFromBytes(data []byte) (*contracts.AssetManifest, error) {
	var a contracts.AssetManifest
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("failed to parse Asset: %w", err)
	}

	if a.Kind != string(contracts.KindAsset) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindAsset, a.Kind)
	}

	return &a, nil
}

// AssetToBytes serializes an AssetManifest to YAML bytes.
func AssetToBytes(a *contracts.AssetManifest) ([]byte, error) {
	return yaml.Marshal(a)
}

// AssetGroupFromBytes parses YAML bytes into an AssetGroupManifest.
func AssetGroupFromBytes(data []byte) (*contracts.AssetGroupManifest, error) {
	var ag contracts.AssetGroupManifest
	if err := yaml.Unmarshal(data, &ag); err != nil {
		return nil, fmt.Errorf("failed to parse AssetGroup: %w", err)
	}

	if ag.Kind != string(contracts.KindAssetGroup) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindAssetGroup, ag.Kind)
	}

	return &ag, nil
}

// AssetGroupToBytes serializes an AssetGroupManifest to YAML bytes.
func AssetGroupToBytes(ag *contracts.AssetGroupManifest) ([]byte, error) {
	return yaml.Marshal(ag)
}
