package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// StoreFromBytes parses YAML bytes into a Store.
func StoreFromBytes(data []byte) (*contracts.Store, error) {
	var s contracts.Store
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse Store: %w", err)
	}

	if s.Kind != string(contracts.KindStore) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindStore, s.Kind)
	}

	return &s, nil
}

// StoreToBytes serializes a Store to YAML bytes.
func StoreToBytes(s *contracts.Store) ([]byte, error) {
	return yaml.Marshal(s)
}
