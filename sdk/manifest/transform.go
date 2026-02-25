package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// TransformFromBytes parses YAML bytes into a Transform.
func TransformFromBytes(data []byte) (*contracts.Transform, error) {
	var t contracts.Transform
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to parse Transform: %w", err)
	}

	if t.Kind != string(contracts.KindTransform) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindTransform, t.Kind)
	}

	return &t, nil
}

// TransformToBytes serializes a Transform to YAML bytes.
func TransformToBytes(t *contracts.Transform) ([]byte, error) {
	return yaml.Marshal(t)
}
