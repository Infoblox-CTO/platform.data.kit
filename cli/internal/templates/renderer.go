package templates

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

//go:embed *.tmpl
var templateFS embed.FS

//go:embed all:cloudquery
var cloudqueryFS embed.FS

// PackageConfig contains the configuration for rendering package templates.
type PackageConfig struct {
	Name        string
	Namespace   string
	Team        string
	Description string
	Owner       string
	Language    string // go, python
	Mode        string // batch, streaming
	Type        string // pipeline, cloudquery
	Role        string // source, destination (cloudquery)
	GRPCPort    int    // gRPC server port (cloudquery, default 7777)
	Concurrency int    // max concurrent resolvers (cloudquery, default 10000)
	Version     string // dp CLI version (for managed file stamps)
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

// splitWords splits a string on dashes, underscores, and camelCase boundaries.
func splitWords(s string) []string {
	var words []string
	var current []rune
	for _, r := range s {
		if r == '-' || r == '_' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
			continue
		}
		if unicode.IsUpper(r) && len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

// snakeCase converts a string to snake_case (e.g. "my-app" -> "my_app").
func snakeCase(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "_")
}

// pascalCase converts a string to PascalCase (e.g. "my-app" -> "MyApp").
func pascalCase(s string) string {
	words := splitWords(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, "")
}

// templateFuncMap provides helper functions available in all templates.
var templateFuncMap = template.FuncMap{
	"snakeCase":  snakeCase,
	"pascalCase": pascalCase,
}

// RenderDirectory renders all templates from a template subdirectory into outputDir.
// It walks the embedded template tree under templateSubDir, creates matching
// subdirectories in outputDir, and renders each .tmpl file stripping the .tmpl suffix.
func (r *Renderer) RenderDirectory(outputDir, templateSubDir string, config *PackageConfig) error {
	return fs.WalkDir(cloudqueryFS, templateSubDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute the relative path from the template subdirectory
		relPath, err := filepath.Rel(templateSubDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		outPath := filepath.Join(outputDir, relPath)

		// Create directories
		if d.IsDir() {
			return os.MkdirAll(outPath, 0755)
		}

		// Only process .tmpl files
		if filepath.Ext(path) != ".tmpl" {
			return nil
		}

		// Strip the .tmpl suffix for the output file
		outPath = strings.TrimSuffix(outPath, ".tmpl")

		// Read and parse the template
		data, err := cloudqueryFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		tmpl, err := template.New(filepath.Base(path)).Funcs(templateFuncMap).Parse(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", outPath, err)
		}

		// Create output file and render
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", outPath, err)
		}
		defer f.Close()

		if err := tmpl.Execute(f, config); err != nil {
			return fmt.Errorf("failed to render template %s: %w", path, err)
		}

		return nil
	})
}

// MakefileCommonConfig holds the values needed to render Makefile.common.
type MakefileCommonConfig struct {
	Version string // dp CLI version that generated the file
}

// RenderMakefileCommon renders the .datakit/Makefile.common template for the
// given language ("go" or "python") and returns the content along with its
// SHA-256 hash. The hash can be stored on disk and compared on subsequent
// build/run invocations to decide whether the file needs updating.
func (r *Renderer) RenderMakefileCommon(language, version string) (content string, hash string, err error) {
	tmplPath := fmt.Sprintf("cloudquery/%s/.datakit/Makefile.common.tmpl", language)
	data, err := cloudqueryFS.ReadFile(tmplPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read Makefile.common template for %s: %w", language, err)
	}

	tmpl, err := template.New("Makefile.common").Parse(string(data))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse Makefile.common template: %w", err)
	}

	var buf bytes.Buffer
	cfg := MakefileCommonConfig{Version: version}
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", "", fmt.Errorf("failed to render Makefile.common: %w", err)
	}

	rendered := buf.String()
	h := sha256.Sum256([]byte(rendered))
	return rendered, hex.EncodeToString(h[:]), nil
}
