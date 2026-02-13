package cmd

import (
	"fmt"
	"os"
	"os/exec"
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
	initMode      string
	initRole      string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new data package",
	Long: `Initialize a new data package with the required manifest files.

Supported package types: pipeline, cloudquery

This command creates a new directory with dp.yaml and project
files pre-configured with sensible defaults for the selected type.

Examples:
  # Create a new pipeline package (default)
  dp init my-pipeline

  # Create a CloudQuery source plugin (Python, default)
  dp init my-source --type cloudquery

  # Create a CloudQuery destination plugin in Go
  dp init my-dest --type cloudquery --role destination --language go

  # Create with custom namespace
  dp init my-pipeline --namespace data-team

  # Create in current directory
  dp init . --type pipeline`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initType, "type", "t", "pipeline",
		"Package type: pipeline, cloudquery")
	initCmd.Flags().StringVarP(&initNamespace, "namespace", "n", "default",
		"Package namespace")
	initCmd.Flags().StringVar(&initTeam, "team", "my-team",
		"Team label")
	initCmd.Flags().StringVar(&initOwner, "owner", "",
		"Package owner (defaults to current user)")
	initCmd.Flags().StringVarP(&initLanguage, "language", "l", "go",
		"Language: go, python")
	initCmd.Flags().StringVarP(&initMode, "mode", "m", "batch",
		"Pipeline mode: batch, streaming (pipeline only)")
	initCmd.Flags().StringVar(&initRole, "role", "source",
		"Plugin role: source, destination (cloudquery only)")
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
		return fmt.Errorf("invalid package type %q: must be pipeline or cloudquery", initType)
	}

	// For cloudquery type, default language to python if not explicitly set
	if initType == "cloudquery" && !cmd.Flags().Changed("language") {
		initLanguage = "python"
	}

	// Validate language
	if !isValidLanguage(initLanguage) {
		return fmt.Errorf("invalid language %q: must be go or python", initLanguage)
	}

	// Validate mode (only for pipeline type)
	if initType == "pipeline" && !isValidMode(initMode) {
		return fmt.Errorf("invalid mode %q: must be batch or streaming", initMode)
	}

	// Validate role for cloudquery type
	if initType == "cloudquery" {
		if initRole == "destination" {
			return fmt.Errorf("destination plugins are not yet supported; only 'source' is currently available")
		}
		if initRole != "source" {
			return fmt.Errorf("invalid role %q: must be source", initRole)
		}
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
		Mode:        initMode,
		Type:        initType,
		Role:        initRole,
		GRPCPort:    7777,
		Concurrency: 10000,
		Version:     Version,
	}

	// CloudQuery packages use directory-based scaffolding
	if initType == "cloudquery" {
		templateSubDir := fmt.Sprintf("cloudquery/%s", initLanguage)
		if err := renderer.RenderDirectory(targetDir, templateSubDir, config); err != nil {
			return fmt.Errorf("failed to scaffold cloudquery project: %w", err)
		}
		output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Scaffolded CloudQuery %s plugin in %s", initLanguage, targetDir))

		// Write the Makefile.common hash so build/run can detect freshness
		if _, err := syncMakefileCommon(targetDir); err != nil {
			cmd.PrintErrf("Warning: failed to write Makefile.common hash: %v\n", err)
		}

		// Go projects: resolve dependencies and format source
		if initLanguage == "go" {
			if err := goPostScaffold(cmd, targetDir); err != nil {
				cmd.PrintErrf("Warning: go post-scaffold failed: %v\n", err)
			}
		}

		cmd.Printf("\nPackage %q initialized successfully!\n", name)
		cmd.Printf("\nNext steps:\n")
		cmd.Printf("  1. Edit dp.yaml to configure your package\n")
		cmd.Printf("  2. Implement your tables in plugin/tables/\n")
		cmd.Printf("  3. Run 'dp lint' to validate\n")
		cmd.Printf("  4. Run 'dp dev up' to start local environment\n")
		cmd.Printf("  5. Run 'dp run' to sync data\n")
		return nil
	}

	// Pipeline packages use single-file templates
	dpPath := filepath.Join(targetDir, "dp.yaml")
	dpTemplate := templates.GetDPTemplate(initType)
	if err := renderer.RenderToFile(dpPath, dpTemplate, config); err != nil {
		return fmt.Errorf("failed to create dp.yaml: %w", err)
	}
	output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created %s", dpPath))

	// Create pipeline.yaml for pipeline type
	if initType == "pipeline" {
		pipelinePath := filepath.Join(targetDir, "pipeline.yaml")
		pipelineTemplate := templates.GetPipelineTemplateForMode(initMode)
		if err := renderer.RenderToFile(pipelinePath, pipelineTemplate, config); err != nil {
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

	// Go projects: resolve dependencies and format source
	if initLanguage == "go" {
		goDir := targetDir
		if initType == "pipeline" {
			goDir = filepath.Join(targetDir, "src")
		}
		if err := goPostScaffold(cmd, goDir); err != nil {
			cmd.PrintErrf("Warning: go post-scaffold failed: %v\n", err)
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

// goPostScaffold runs go mod tidy and go fmt on a scaffolded Go project so
// the generated code compiles immediately (go.sum present, source formatted).
func goPostScaffold(cmd *cobra.Command, dir string) error {
	// Require the go toolchain
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go toolchain not found: install Go from https://go.dev/dl/")
	}

	cmd.Printf("Running go mod tidy...\n")
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	cmd.Printf("Running go fmt...\n")
	gofmt := exec.Command("go", "fmt", "./...")
	gofmt.Dir = dir
	gofmt.Stdout = os.Stdout
	gofmt.Stderr = os.Stderr
	if err := gofmt.Run(); err != nil {
		// go fmt is non-critical; warn but don't fail
		cmd.PrintErrf("Warning: go fmt failed: %v\n", err)
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
	case "pipeline", "cloudquery":
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

// isValidMode checks if a pipeline mode is valid
func isValidMode(mode string) bool {
	switch mode {
	case "batch", "streaming":
		return true
	default:
		return false
	}
}
