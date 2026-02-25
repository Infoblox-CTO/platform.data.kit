package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// ConnectorFromBytes parses YAML bytes into a Connector.
func ConnectorFromBytes(data []byte) (*contracts.Connector, error) {
	var c contracts.Connector
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("failed to parse Connector: %w", err)
	}

	if c.Kind != string(contracts.KindConnector) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindConnector, c.Kind)
	}

	return &c, nil
}

// ConnectorToBytes serializes a Connector to YAML bytes.
func ConnectorToBytes(c *contracts.Connector) ([]byte, error) {
	return yaml.Marshal(c)
}
