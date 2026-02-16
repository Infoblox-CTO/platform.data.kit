package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// ModelFromBytes parses a dp.yaml file content into a Model.
func ModelFromBytes(data []byte) (*contracts.Model, error) {
	var model contracts.Model
	if err := yaml.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("failed to parse Model: %w", err)
	}

	if model.Kind != string(contracts.KindModel) {
		return nil, fmt.Errorf("expected kind %q, got %q", contracts.KindModel, model.Kind)
	}

	return &model, nil
}

// ModelToBytes serializes a Model to YAML bytes.
func ModelToBytes(model *contracts.Model) ([]byte, error) {
	return yaml.Marshal(model)
}
