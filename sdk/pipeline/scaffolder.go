package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
)

// ScaffoldOpts configures pipeline scaffolding.
type ScaffoldOpts struct {
	Name       string // Pipeline name (DNS-safe)
	Template   string // Template name (e.g., "sync-transform-test")
	ProjectDir string // Project root directory
	Force      bool   // Overwrite existing pipeline.yaml
}

// ScaffoldPipeline creates a new pipeline.yaml from a template.
func ScaffoldPipeline(opts ScaffoldOpts) (string, error) {
	if opts.Name == "" {
		return "", fmt.Errorf("pipeline name is required")
	}
	if opts.Template == "" {
		return "", fmt.Errorf("template name is required")
	}
	if opts.ProjectDir == "" {
		return "", fmt.Errorf("project directory is required")
	}

	// Check if pipeline.yaml already exists
	outputPath := filepath.Join(opts.ProjectDir, PipelineFileName)
	if _, err := os.Stat(outputPath); err == nil && !opts.Force {
		return "", fmt.Errorf("%s already exists; use --force to overwrite", PipelineFileName)
	}

	// Render the template
	content, err := RenderTemplate(opts.Template, TemplateConfig{Name: opts.Name})
	if err != nil {
		return "", err
	}

	// Write the file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	return outputPath, nil
}
