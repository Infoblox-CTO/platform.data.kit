package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// DestinationFromBytes parses a dp.yaml file content into a Destination.
func DestinationFromBytes(data []byte) (*contracts.Destination, error) {
	var dest contracts.Destination
	if err := yaml.Unmarshal(data, &dest); err != nil {
		return nil, fmt.Errorf("failed to parse Destination: %w", err)
	}

	if dest.Kind != string(contracts.KindDestination) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindDestination, dest.Kind)
	}

	return &dest, nil
}

// DestinationToBytes serializes a Destination to YAML bytes.
func DestinationToBytes(dest *contracts.Destination) ([]byte, error) {
	return yaml.Marshal(dest)
}
