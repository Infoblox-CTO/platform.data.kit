package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// PipelineFileName is the default filename for pipeline workflow definitions.
const PipelineFileName = "pipeline.yaml"

// LoadPipeline loads and parses a pipeline.yaml from the given path.
// The path can be a directory containing pipeline.yaml or a direct file path.
func LoadPipeline(path string) (*contracts.PipelineWorkflow, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("pipeline path not found: %w", err)
	}

	pipelinePath := path
	if info.IsDir() {
		pipelinePath = filepath.Join(path, PipelineFileName)
	}

	data, err := os.ReadFile(pipelinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline file %s: %w", pipelinePath, err)
	}

	var pw contracts.PipelineWorkflow
	if err := yaml.Unmarshal(data, &pw); err != nil {
		return nil, fmt.Errorf("failed to parse pipeline file %s: %w", pipelinePath, err)
	}

	return &pw, nil
}

// FindPipeline searches for a pipeline.yaml starting from the given directory
// and walking up the directory tree. Returns the path if found, empty string otherwise.
func FindPipeline(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory: %w", err)
	}

	for {
		candidate := filepath.Join(dir, PipelineFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// HasPipeline checks if a pipeline.yaml file exists in the given directory.
func HasPipeline(dir string) bool {
	pipelinePath := filepath.Join(dir, PipelineFileName)
	_, err := os.Stat(pipelinePath)
	return err == nil
}

// ScheduleFileName is the default filename for schedule definitions.
const ScheduleFileName = "schedule.yaml"

// LoadSchedule loads and parses a schedule.yaml from the given path.
// The path can be a directory containing schedule.yaml or a direct file path.
// Returns nil, nil if the file does not exist (schedule is optional).
func LoadSchedule(path string) (*contracts.ScheduleManifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil // Schedule is optional
	}

	schedulePath := path
	if info.IsDir() {
		schedulePath = filepath.Join(path, ScheduleFileName)
	}

	data, err := os.ReadFile(schedulePath)
	if err != nil {
		return nil, nil // File doesn't exist — schedule is optional
	}

	var sm contracts.ScheduleManifest
	if err := yaml.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("failed to parse schedule file %s: %w", schedulePath, err)
	}

	return &sm, nil
}
