package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Infoblox-CTO/platform.data.kit/cli/internal/output"
	"github.com/Infoblox-CTO/platform.data.kit/cli/internal/templates"
	"github.com/spf13/cobra"
)

var (
	initType      string
	initNamespace string
	initTeam      string
	initOwner     string
	initLanguage  string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new data package",
	Long: `Initialize a new data package with the required manifest files.

This command creates a new directory with dp.yaml and (for pipeline type)
pipeline.yaml files pre-configured with sensible defaults.

Package types:
  pipeline - A data processing pipeline (default)
  model    - A machine learning model package
  dataset  - A curated dataset package

Examples:
  # Create a new pipeline package
  dp init my-pipeline

  # Create a model package with custom namespace
  dp init my-model --type model --namespace ml-team

  # Create in current directory
  dp init . --type pipeline`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initType, "type", "t", "pipeline",
		"Package type: pipeline, model, dataset")
	initCmd.Flags().StringVarP(&initNamespace, "namespace", "n", "default",
		"Package namespace")
	initCmd.Flags().StringVar(&initTeam, "team", "my-team",
		"Team label")
	initCmd.Flags().StringVar(&initOwner, "owner", "",
		"Package owner (defaults to current user)")
	initCmd.Flags().StringVarP(&initLanguage, "language", "l", "go",
		"Pipeline language: go, python")
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if name != "." && !isValidPackageName(name) {
		return fmt.Errorf("invalid package name %q: must be DNS-safe (lowercase, alphanumeric, hyphens, 3-63 chars)", name)
	}

	// Get the target directory
	var targetDir string
	if name == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		targetDir = cwd
		name = filepath.Base(cwd)
	} else {
		targetDir = name
	}

	// Check if directory exists and is not empty
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(targetDir)
		if len(entries) > 0 && targetDir != "." {
			return fmt.Errorf("directory %q already exists and is not empty", targetDir)
		}
	}

	// Validate package type
	if !isValidPackageType(initType) {
		return fmt.Errorf("invalid package type %q: must be pipeline, model, or dataset", initType)
	}

	// Validate language
	if !isValidLanguage(initLanguage) {
		return fmt.Errorf("invalid language %q: must be go or python", initLanguage)
	}

	// Set default owner
	if initOwner == "" {
		initOwner = fmt.Sprintf("%s-team", initNamespace)
	}

	// Create renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	config := &templates.PackageConfig{
		Name:        name,
		Namespace:   initNamespace,
		Team:        initTeam,
		Description: fmt.Sprintf("A %s package", initType),
		Owner:       initOwner,
		Language:    initLanguage,
	}

	// Create dp.yaml
	dpPath := filepath.Join(targetDir, "dp.yaml")
	dpTemplate := templates.GetDPTemplate(initType)
	if err := renderer.RenderToFile(dpPath, dpTemplate, config); err != nil {
		return fmt.Errorf("failed to create dp.yaml: %w", err)
	}
	output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %s", dpPath))

	// Create pipeline.yaml for pipeline type
	if initType == "pipeline" {
		pipelinePath := filepath.Join(targetDir, "pipeline.yaml")
		if err := renderer.RenderToFile(pipelinePath, templates.GetPipelineTemplate(), config); err != nil {
			return fmt.Errorf("failed to create pipeline.yaml: %w", err)
		}
		output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %s", pipelinePath))
	}

	// Create src directory for pipeline
	if initType == "pipeline" {
		srcDir := filepath.Join(targetDir, "src")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			return fmt.Errorf("failed to create src directory: %w", err)
		}

		// Create language-specific files
		switch initLanguage {
		case "python":
			// Create requirements.txt
			reqPath := filepath.Join(srcDir, "requirements.txt")
			reqContent := "# Add your dependencies here\n"
			if err := os.WriteFile(reqPath, []byte(reqContent), 0644); err != nil {
				return fmt.Errorf("failed to create requirements.txt: %w", err)
			}
			output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %s", reqPath))

			// Create main.py
			mainPath := filepath.Join(srcDir, "main.py")
			mainContent := fmt.Sprintf(`#!/usr/bin/env python3
"""
%s pipeline
"""

def main():
    print("Hello from %s pipeline!")

if __name__ == "__main__":
    main()
`, name, name)
			if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
				return fmt.Errorf("failed to create main.py: %w", err)
			}
			output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %s", mainPath))

		default: // go
			// Create go.mod
			goModPath := filepath.Join(srcDir, "go.mod")
			goModContent := fmt.Sprintf("module %s\n\ngo 1.21\n", name)
			if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
				return fmt.Errorf("failed to create go.mod: %w", err)
			}
			output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %s", goModPath))

			// Create main.go
			mainPath := filepath.Join(srcDir, "main.go")
			mainContent := fmt.Sprintf(`package main

import "fmt"

func main() {
	fmt.Println("Hello from %s pipeline!")
}
`, name)
			if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
				return fmt.Errorf("failed to create main.go: %w", err)
			}
			output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %s", mainPath))
		}
	}

	cmd.Printf("\nPackage %q initialized successfully!\n", name)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit dp.yaml to configure your package\n")
	if initType == "pipeline" {
		cmd.Printf("  2. Edit pipeline.yaml to configure runtime settings\n")
		cmd.Printf("  3. Implement your pipeline in src/\n")
		cmd.Printf("  4. Run 'dp lint' to validate\n")
		cmd.Printf("  5. Run 'dp dev up' to start local environment\n")
	}

	return nil
}

// isValidPackageName checks if a name is DNS-safe
func isValidPackageName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	matched, _ := regexp.MatchString("^[a-z][a-z0-9-]*[a-z0-9]$", name)
	return matched
}

// isValidPackageType checks if a package type is valid
func isValidPackageType(t string) bool {
	switch t {
	case "pipeline", "model", "dataset":
		return true
	default:
		return false
	}
}

// isValidLanguage checks if a language is valid
func isValidLanguage(lang string) bool {
	switch lang {
	case "go", "python":
		return true
	default:
		return false
	}
}
