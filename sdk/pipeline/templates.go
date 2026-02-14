package pipeline

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// TemplateConfig holds the configuration values for rendering a pipeline template.
type TemplateConfig struct {
	Name string
}

// ListTemplates returns the names of all available pipeline templates.
func ListTemplates() ([]string, error) {
	entries, err := fs.ReadDir(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read template directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmpl") {
			names = append(names, strings.TrimSuffix(name, ".tmpl"))
		}
	}
	return names, nil
}

// RenderTemplate renders a named pipeline template with the given configuration.
// Returns the rendered YAML content.
func RenderTemplate(templateName string, config TemplateConfig) (string, error) {
	tmplPath := "templates/" + templateName + ".tmpl"

	data, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		available, _ := ListTemplates()
		return "", fmt.Errorf("template %q not found; available templates: %v", templateName, available)
	}

	tmpl, err := template.New(templateName).Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %q: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to render template %q: %w", templateName, err)
	}

	return buf.String(), nil
}
