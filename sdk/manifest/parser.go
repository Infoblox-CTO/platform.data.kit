// Package manifest provides YAML parsing for DP manifest files.
package manifest

import (
	"fmt"
	"os"

	"github.com/Infoblox-CTO/data-platform/contracts"
)

// Parser is the interface for parsing manifest files.
type Parser interface {
	// ParseDataPackage parses a dp.yaml file
	ParseDataPackage(data []byte) (*contracts.DataPackage, error)

	// ParsePipeline parses a pipeline.yaml file
	ParsePipeline(data []byte) (*contracts.PipelineManifest, error)

	// ParseBindings parses a bindings.yaml file
	ParseBindings(data []byte) ([]contracts.Binding, error)
}

// DefaultParser is the default implementation of Parser.
type DefaultParser struct{}

// NewParser creates a new manifest parser.
func NewParser() Parser {
	return &DefaultParser{}
}

// ParseDataPackage parses a dp.yaml file from bytes.
func (p *DefaultParser) ParseDataPackage(data []byte) (*contracts.DataPackage, error) {
	return DataPackageFromBytes(data)
}

// ParsePipeline parses a pipeline.yaml file from bytes.
func (p *DefaultParser) ParsePipeline(data []byte) (*contracts.PipelineManifest, error) {
	return PipelineFromBytes(data)
}

// ParseBindings parses a bindings.yaml file from bytes.
func (p *DefaultParser) ParseBindings(data []byte) ([]contracts.Binding, error) {
	return BindingsFromBytes(data)
}

// ParseDataPackageFile parses a dp.yaml file from a path.
func ParseDataPackageFile(path string) (*contracts.DataPackage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return NewParser().ParseDataPackage(data)
}

// ParsePipelineFile parses a pipeline.yaml file from a path.
func ParsePipelineFile(path string) (*contracts.PipelineManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return NewParser().ParsePipeline(data)
}

// ParseBindingsFile parses a bindings.yaml file from a path.
func ParseBindingsFile(path string) ([]contracts.Binding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return NewParser().ParseBindings(data)
}
