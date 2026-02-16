// Package manifest provides YAML parsing for DP manifest files.
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
	// ParseSource parses a dp.yaml file with kind: Source
	ParseSource(data []byte) (*contracts.Source, error)

	// ParseDestination parses a dp.yaml file with kind: Destination
	ParseDestination(data []byte) (*contracts.Destination, error)

	// ParseModel parses a dp.yaml file with kind: Model
	ParseModel(data []byte) (*contracts.Model, error)

	// ParseBindings parses a bindings.yaml file
	ParseBindings(data []byte) ([]contracts.Binding, error)
}

// DefaultParser is the default implementation of Parser.
type DefaultParser struct{}

// NewParser creates a new manifest parser.
func NewParser() Parser {
	return &DefaultParser{}
}

// ParseSource parses a dp.yaml with kind: Source.
func (p *DefaultParser) ParseSource(data []byte) (*contracts.Source, error) {
	return SourceFromBytes(data)
}

// ParseDestination parses a dp.yaml with kind: Destination.
func (p *DefaultParser) ParseDestination(data []byte) (*contracts.Destination, error) {
	return DestinationFromBytes(data)
}

// ParseModel parses a dp.yaml with kind: Model.
func (p *DefaultParser) ParseModel(data []byte) (*contracts.Model, error) {
	return ModelFromBytes(data)
}

// ParseBindings parses a bindings.yaml file from bytes.
func (p *DefaultParser) ParseBindings(data []byte) ([]contracts.Binding, error) {
	return BindingsFromBytes(data)
}

// ParseSourceFile parses a Source manifest file from a path.
func ParseSourceFile(path string) (*contracts.Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseSource(data)
}

// ParseDestinationFile parses a Destination manifest file from a path.
func ParseDestinationFile(path string) (*contracts.Destination, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseDestination(data)
}

// ParseModelFile parses a Model manifest file from a path.
func ParseModelFile(path string) (*contracts.Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return NewParser().ParseModel(data)
}

// ParseBindingsFile parses a bindings.yaml file from a path.
func ParseBindingsFile(path string) ([]contracts.Binding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return NewParser().ParseBindings(data)
}

// Manifest is a generic interface satisfied by all manifest kinds
// (Source, Destination, Model). It provides access to common metadata.
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

// ParseManifestFile reads a dp.yaml, detects the kind, and returns the
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
	case contracts.KindSource:
		m, err := parser.ParseSource(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	case contracts.KindDestination:
		m, err := parser.ParseDestination(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	case contracts.KindModel:
		m, err := parser.ParseModel(data)
		if err != nil {
			return nil, kind, err
		}
		return m, kind, nil
	default:
		return nil, kind, fmt.Errorf("unsupported manifest kind %q: must be Source, Destination, or Model", kind)
	}
}
