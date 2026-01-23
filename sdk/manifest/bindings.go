package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// BindingsFromBytes parses a bindings.yaml file content into a slice of Bindings.
func BindingsFromBytes(data []byte) ([]contracts.Binding, error) {
	var manifest contracts.BindingsManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse Bindings: %w", err)
	}

	return manifest.Bindings, nil
}

// BindingsToBytes serializes bindings to YAML bytes.
func BindingsToBytes(bindings []contracts.Binding) ([]byte, error) {
	manifest := contracts.BindingsManifest{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "Bindings",
		Bindings:   bindings,
	}
	return yaml.Marshal(manifest)
}

// GetBinding finds a binding by name.
func GetBinding(bindings []contracts.Binding, name string) (*contracts.Binding, error) {
	for i := range bindings {
		if bindings[i].Name == name {
			return &bindings[i], nil
		}
	}
	return nil, fmt.Errorf("binding not found: %s", name)
}

// GetBindingProperty returns a specific property from a binding.
func GetBindingProperty(binding *contracts.Binding, property string) (string, error) {
	switch property {
	case "type":
		return string(binding.Type), nil
	case "name":
		return binding.Name, nil
	}

	switch binding.Type {
	case contracts.BindingTypeS3Prefix:
		if binding.S3 == nil {
			return "", fmt.Errorf("s3 config is nil")
		}
		switch property {
		case "bucket":
			return binding.S3.Bucket, nil
		case "prefix":
			return binding.S3.Prefix, nil
		case "endpoint":
			return binding.S3.Endpoint, nil
		case "region":
			return binding.S3.Region, nil
		}
	case contracts.BindingTypeKafkaTopic:
		if binding.Kafka == nil {
			return "", fmt.Errorf("kafka config is nil")
		}
		switch property {
		case "topic":
			return binding.Kafka.Topic, nil
		case "schemaRegistry":
			return binding.Kafka.SchemaRegistry, nil
		}
	case contracts.BindingTypePostgresTable:
		if binding.Postgres == nil {
			return "", fmt.Errorf("postgres config is nil")
		}
		switch property {
		case "table":
			return binding.Postgres.Table, nil
		case "database":
			return binding.Postgres.Database, nil
		case "schema":
			return binding.Postgres.Schema, nil
		}
	}

	return "", fmt.Errorf("unknown property: %s", property)
}
