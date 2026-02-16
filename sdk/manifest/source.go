package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// SourceFromBytes parses a dp.yaml file content into a Source.
func SourceFromBytes(data []byte) (*contracts.Source, error) {
	var src contracts.Source
	if err := yaml.Unmarshal(data, &src); err != nil {
		return nil, fmt.Errorf("failed to parse Source: %w", err)
	}

	if src.Kind != string(contracts.KindSource) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindSource, src.Kind)
	}

	return &src, nil
}

// SourceToBytes serializes a Source to YAML bytes.
func SourceToBytes(src *contracts.Source) ([]byte, error) {
	return yaml.Marshal(src)
}
