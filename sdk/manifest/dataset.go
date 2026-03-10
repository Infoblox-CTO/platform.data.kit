package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// DataSetFromBytes parses YAML bytes into a DataSetManifest.
func DataSetFromBytes(data []byte) (*contracts.DataSetManifest, error) {
	var a contracts.DataSetManifest
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("failed to parse DataSet: %w", err)
	}

	if a.Kind != string(contracts.KindDataSet) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindDataSet, a.Kind)
	}

	return &a, nil
}

// DataSetToBytes serializes a DataSetManifest to YAML bytes.
func DataSetToBytes(a *contracts.DataSetManifest) ([]byte, error) {
	return yaml.Marshal(a)
}

// DataSetGroupFromBytes parses YAML bytes into a DataSetGroupManifest.
func DataSetGroupFromBytes(data []byte) (*contracts.DataSetGroupManifest, error) {
	var ag contracts.DataSetGroupManifest
	if err := yaml.Unmarshal(data, &ag); err != nil {
		return nil, fmt.Errorf("failed to parse DataSetGroup: %w", err)
	}

	if ag.Kind != string(contracts.KindDataSetGroup) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindDataSetGroup, ag.Kind)
	}

	return &ag, nil
}

// DataSetGroupToBytes serializes a DataSetGroupManifest to YAML bytes.
func DataSetGroupToBytes(ag *contracts.DataSetGroupManifest) ([]byte, error) {
	return yaml.Marshal(ag)
}
