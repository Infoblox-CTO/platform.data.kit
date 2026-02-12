package templates

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed *.tmpl
var templateFS embed.FS

// PackageConfig contains the configuration for rendering package templates.
type PackageConfig struct {
	Name        string
	Namespace   string
	Team        string
	Description string
	Owner       string
	Language    string // go, python
	Mode        string // batch, streaming
}

// Renderer renders package templates.
type Renderer struct {
	templates *template.Template
}

// NewRenderer creates a new template renderer.
func NewRenderer() (*Renderer, error) {
	tmpl, err := template.ParseFS(templateFS, "*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return &Renderer{templates: tmpl}, nil
}

// RenderToWriter renders a template to a writer.
func (r *Renderer) RenderToWriter(w io.Writer, templateName string, config *PackageConfig) error {
	return r.templates.ExecuteTemplate(w, templateName, config)
}

// RenderToString renders a template to a string.
func (r *Renderer) RenderToString(templateName string, config *PackageConfig) (string, error) {
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf, templateName, config); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderToFile renders a template to a file.
func (r *Renderer) RenderToFile(path, templateName string, config *PackageConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer f.Close()

	return r.RenderToWriter(f, templateName, config)
}

// GetDPTemplate returns the template name for the given package type.
func GetDPTemplate(packageType string) string {
	return "dp.yaml.tmpl"
}

// GetPipelineTemplate returns the pipeline template name.
func GetPipelineTemplate() string {
	return "pipeline.yaml.tmpl"
}

// GetPipelineTemplateForMode returns the pipeline template for the given mode.
func GetPipelineTemplateForMode(mode string) string {
	switch mode {
	case "streaming":
		return "pipeline.streaming.yaml.tmpl"
	default:
		return "pipeline.batch.yaml.tmpl"
	}
}
