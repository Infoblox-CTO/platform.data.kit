package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

//go:embed all:transform
var transformFS embed.FS

// templateFuncMap provides helper functions available in templates.
var templateFuncMap = template.FuncMap{
	"lower":      strings.ToLower,
	"upper":      strings.ToUpper,
	"title":      strings.Title,
	"snake":      snakeCase,
	"snakeCase":  snakeCase,
	"pascal":     pascalCase,
	"pascalCase": pascalCase,
	"contains":   strings.Contains,
	"hasPrefix":  strings.HasPrefix,
	"hasSuffix":  strings.HasSuffix,
	"replace":    strings.ReplaceAll,
	"trimSpace":  strings.TrimSpace,
}

// snakeCase converts a string to snake_case.
func snakeCase(s string) string {
	words := splitWords(s)
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	return strings.Join(words, "_")
}

// pascalCase converts a string to PascalCase.
func pascalCase(s string) string {
	words := splitWords(s)
	for i := range words {
		if len(words[i]) > 0 {
			words[i] = strings.ToUpper(words[i][:1]) + strings.ToLower(words[i][1:])
		}
	}
	return strings.Join(words, "")
}

// splitWords splits a string into words on hyphens, underscores, and camelCase boundaries.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder

	for i, r := range s {
		if r == '-' || r == '_' || r == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}
		if i > 0 && unicode.IsUpper(r) && current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// PackageConfig contains the configuration for rendering package templates.
type PackageConfig struct {
	Name        string
	Namespace   string
	Team        string
	Description string
	Owner       string
	Mode        string // batch, streaming
	Kind        string // Transform, Connector, Store, DataSet, DataSetGroup
	Runtime     string // cloudquery, generic-go, generic-python, dbt (new taxonomy)
	GRPCPort    int    // gRPC server port (cloudquery, default 7777)
	Concurrency int    // max concurrent resolvers (cloudquery, default 10000)
	Version     string // dk CLI version (for managed file stamps)
}

// Renderer renders package templates.
type Renderer struct{}

// NewRenderer creates a new template renderer.
func NewRenderer() (*Renderer, error) {
	return &Renderer{}, nil
}

// kindFS returns the embedded filesystem for the given kind.
func kindFS(kind string) (embed.FS, error) {
	switch strings.ToLower(kind) {
	case "transform":
		return transformFS, nil
	default:
		return embed.FS{}, fmt.Errorf("unknown kind %q; supported kinds: transform", kind)
	}
}

// RenderKindDirectory renders templates from a kind/runtime subdirectory into outputDir.
// It uses the Kind field in config to select the correct embedded filesystem,
// then walks the runtime subdirectory (e.g., "cloudquery", "generic-go").
func (r *Renderer) RenderKindDirectory(outputDir string, config *PackageConfig) error {
	efs, err := kindFS(config.Kind)
	if err != nil {
		return err
	}

	// Template subdirectory is kind/runtime within the embedded FS.
	kindDir := strings.ToLower(config.Kind)
	templateSubDir := fmt.Sprintf("%s/%s", kindDir, config.Runtime)

	return fs.WalkDir(efs, templateSubDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(templateSubDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path: %w", err)
		}

		if relPath == "." {
			return nil
		}

		outPath := filepath.Join(outputDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(outPath, 0755)
		}

		if filepath.Ext(path) != ".tmpl" {
			return nil
		}

		outPath = strings.TrimSuffix(outPath, ".tmpl")

		data, readErr := efs.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("failed to read template %s: %w", path, readErr)
		}

		tmpl, parseErr := template.New(filepath.Base(path)).Funcs(templateFuncMap).Parse(string(data))
		if parseErr != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, parseErr)
		}

		if mkErr := os.MkdirAll(filepath.Dir(outPath), 0755); mkErr != nil {
			return fmt.Errorf("failed to create directory for %s: %w", outPath, mkErr)
		}

		f, createErr := os.Create(outPath)
		if createErr != nil {
			return fmt.Errorf("failed to create file %s: %w", outPath, createErr)
		}
		defer f.Close()

		if execErr := tmpl.Execute(f, config); execErr != nil {
			return fmt.Errorf("failed to render template %s: %w", path, execErr)
		}

		return nil
	})
}
