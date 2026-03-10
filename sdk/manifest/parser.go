// Package manifest provides YAML parsing for DataKit manifest files.
package manifest

import (
	"fmt"
	"os"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// kindProbe is used to detect the "kind" field before full parsing.
type kindProbe struct {
	Kind string `yaml:"kind"`
}

// DetectKind reads the kind field from raw YAML without full parsing.
func DetectKind(data []byte) (contracts.Kind, error) {
	var probe kindProbe
	if err := yaml.Unmarshal(data, &probe); err != nil {
		return "", fmt.Errorf("failed to probe kind: %w", err)
	}
	if probe.Kind == "" {
		return "", fmt.Errorf("manifest is missing required 'kind' field")
	}
	return contracts.Kind(probe.Kind), nil
}

// DetectKindFromFile reads the kind from a file.
func DetectKindFromFile(path string) (contracts.Kind, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return DetectKind(data)
}

// Parser is the interface for parsing manifest files.
type Parser interface {
	// ParseConnector parses a manifest with kind: Connector.
	ParseConnector(data []byte) (*contracts.Connector, error)

	// ParseStore parses a manifest with kind: Store.
	ParseStore(data []byte) (*contracts.Store, error)

	// ParseDataSet parses a manifest with kind: DataSet.
	ParseDataSet(data []byte) (*contracts.DataSetManifest, error)

	// ParseDataSetGroup parses a manifest with kind: DataSetGroup.
	ParseDataSetGroup(data []byte) (*contracts.DataSetGroupManifest, error)

	// ParseTransform parses a manifest with kind: Transform.
	ParseTransform(data []byte) (*contracts.Transform, error)
}

// DefaultParser is the default implementation of Parser.
type DefaultParser struct{}

// NewParser creates a new manifest parser.
func NewParser() Parser {
	return &DefaultParser{}
}

// ParseConnector parses a manifest with kind: Connector.
func (p *DefaultParser) ParseConnector(data []byte) (*contracts.Connector, error) {
	return ConnectorFromBytes(data)
}

// ParseStore parses a manifest with kind: Store.
func (p *DefaultParser) ParseStore(data []byte) (*contracts.Store, error) {
	return StoreFromBytes(data)
}

// ParseDataSet parses a manifest with kind: DataSet.
func (p *DefaultParser) ParseDataSet(data []byte) (*contracts.DataSetManifest, error) {
	return DataSetFromBytes(data)
}

// ParseDataSetGroup parses a manifest with kind: DataSetGroup.
func (p *DefaultParser) ParseDataSetGroup(data []byte) (*contracts.DataSetGroupManifest, error) {
	return DataSetGroupFromBytes(data)
}

// ParseTransform parses a manifest with kind: Transform.
func (p *DefaultParser) ParseTransform(data []byte) (*contracts.Transform, error) {
	return TransformFromBytes(data)
}

// ParseConnectorFile parses a Connector manifest file from a path.
func ParseConnectorFile(path string) (*contracts.Connector, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseConnector(data)
}

// ParseStoreFile parses a Store manifest file from a path.
func ParseStoreFile(path string) (*contracts.Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseStore(data)
}

// ParseDataSetFile parses a DataSet manifest file from a path.
func ParseDataSetFile(path string) (*contracts.DataSetManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseDataSet(data)
}

// ParseDataSetGroupFile parses a DataSetGroup manifest file from a path.
func ParseDataSetGroupFile(path string) (*contracts.DataSetGroupManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseDataSetGroup(data)
}

// ParseTransformFile parses a Transform manifest file from a path.
func ParseTransformFile(path string) (*contracts.Transform, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseTransform(data)
}

// Manifest is a generic interface satisfied by all manifest kinds
// (Connector, Store, DataSet, DataSetGroup, Transform).
// It provides access to common metadata.
type Manifest interface {
	// GetKind returns the manifest kind.
	GetKind() contracts.Kind
	// GetName returns the manifest name.
	GetName() string
	// GetNamespace returns the manifest namespace.
	GetNamespace() string
	// GetVersion returns the manifest version.
	GetVersion() string
	// GetDescription returns the spec description.
	GetDescription() string
	// GetOwner returns the spec owner.
	GetOwner() string
}

// ParseManifestFile reads a dk.yaml, detects the kind, and returns the
// parsed manifest as a Manifest interface along with the concrete kind.
func ParseManifestFile(path string) (Manifest, contracts.Kind, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return ParseManifest(data)
}

// ParseManifest detects the kind from raw YAML and parses accordingly.
func ParseManifest(data []byte) (Manifest, contracts.Kind, error) {
	kind, err := DetectKind(data)
	if err != nil {
		return nil, "", err
	}

	parser := NewParser()
	switch kind {
	case contracts.KindConnector:
		m, err := parser.ParseConnector(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	case contracts.KindStore:
		m, err := parser.ParseStore(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	case contracts.KindDataSet:
		m, err := parser.ParseDataSet(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	case contracts.KindDataSetGroup:
		m, err := parser.ParseDataSetGroup(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	case contracts.KindTransform:
		m, err := parser.ParseTransform(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	default:
		return nil, kind, fmt.Errorf("unsupported manifest kind %q: must be Connector, Store, DataSet, DataSetGroup, or Transform", kind)
	}
}
