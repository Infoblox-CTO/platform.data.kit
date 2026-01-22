package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/data-platform/contracts"
	"gopkg.in/yaml.v3"
)

// PipelineFromBytes parses a pipeline.yaml file content into a PipelineManifest.
func PipelineFromBytes(data []byte) (*contracts.PipelineManifest, error) {
	var pipeline contracts.PipelineManifest
	if err := yaml.Unmarshal(data, &pipeline); err != nil {
		return nil, fmt.Errorf("failed to parse Pipeline: %w", err)
	}

	if pipeline.Kind != "Pipeline" {
		return nil, fmt.Errorf("expected kind 'Pipeline', got '%s'", pipeline.Kind)
	}

	return &pipeline, nil
}

// PipelineToBytes serializes a PipelineManifest to YAML bytes.
func PipelineToBytes(pipeline *contracts.PipelineManifest) ([]byte, error) {
	return yaml.Marshal(pipeline)
}
