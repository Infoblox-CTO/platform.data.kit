package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	projectInitDescription string
)

// projectInitCmd scaffolds a new multi-transform project.
var projectInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Scaffold a new DataKit project",
	Long: `Scaffold a new DataKit project directory with the standard layout
for a multi-transform data pipeline.

Creates the following directory structure:
  <name>/
    connectors/       Platform-managed connector definitions
    stores/           Named store instances
    datasets/         Data contracts (schema, classification, lineage)
    transforms/       Transform packages (CloudQuery, dbt, Go, Python)
    README.md         Project documentation

Examples:
  # Create a new project
  dk project init k8s-analytics

  # Create with a description
  dk project init k8s-analytics --description "K8s cluster analysis pipeline"

  # Initialize in the current directory
  dk project init .`,
	Args: cobra.ExactArgs(1),
	RunE: runProjectInit,
}

func init() {
	projectCmd.AddCommand(projectInitCmd)

	projectInitCmd.Flags().StringVar(&projectInitDescription, "description", "",
		"Project description for the README")
}

func runProjectInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Determine target directory
	var targetDir string
	if name == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		targetDir = cwd
		name = filepath.Base(cwd)
	} else {
		if !isValidPackageName(name) {
			return fmt.Errorf("invalid project name %q: must be DNS-safe (lowercase, alphanumeric, hyphens, 3-63 chars)", name)
		}
		targetDir = name
	}

	// Check if directory exists and is not empty (unless it's ".")
	if args[0] != "." {
		if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
			entries, _ := os.ReadDir(targetDir)
			if len(entries) > 0 {
				return fmt.Errorf("directory %q already exists and is not empty", targetDir)
			}
		}
	}

	// Create the project directory structure
	dirs := []string{
		"connectors",
		"stores",
		"datasets",
		"transforms",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(targetDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		// Create .gitkeep so empty dirs are tracked
		gitkeep := filepath.Join(dirPath, ".gitkeep")
		if _, err := os.Stat(gitkeep); os.IsNotExist(err) {
			if err := os.WriteFile(gitkeep, []byte(""), 0644); err != nil {
				return fmt.Errorf("failed to create .gitkeep in %s: %w", dir, err)
			}
		}
	}

	// Generate README.md
	description := projectInitDescription
	if description == "" {
		description = fmt.Sprintf("A DataKit data pipeline project.")
	}
	readme := generateProjectReadme(name, description)
	readmePath := filepath.Join(targetDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
			return fmt.Errorf("failed to write README.md: %w", err)
		}
	}

	// Generate .gitignore
	gitignore := generateProjectGitignore()
	gitignorePath := filepath.Join(targetDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
			return fmt.Errorf("failed to write .gitignore: %w", err)
		}
	}

	cmd.Printf("Project %q initialized at %s\n", name, targetDir)
	cmd.Printf("\nCreated:\n")
	cmd.Printf("  %s/\n", targetDir)
	for _, dir := range dirs {
		cmd.Printf("    %s/\n", dir)
	}
	cmd.Printf("    README.md\n")
	cmd.Printf("    .gitignore\n")

	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. cd %s\n", targetDir)
	cmd.Printf("  2. dk connector create <name> --type <type>    # Define connectors\n")
	cmd.Printf("  3. dk store create <name> --connector <type>   # Create stores\n")
	cmd.Printf("  4. dk dataset create <name> --store <store>    # Define datasets\n")
	cmd.Printf("  5. dk init <name> --runtime <runtime>          # Create transforms\n")
	cmd.Printf("  6. dk pipeline show --scan-dir .               # View the pipeline graph\n")

	return nil
}

func generateProjectReadme(name, description string) string {
	title := strings.ReplaceAll(name, "-", " ")
	title = strings.Title(title) //nolint:staticcheck

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", title))
	b.WriteString(fmt.Sprintf("%s\n\n", description))
	b.WriteString("## Project Structure\n\n")
	b.WriteString("```\n")
	b.WriteString(fmt.Sprintf("%s/\n", name))
	b.WriteString("  connectors/     # Platform-managed connector definitions\n")
	b.WriteString("  stores/         # Named store instances with connection details\n")
	b.WriteString("  datasets/       # Data contracts (schema, classification, lineage)\n")
	b.WriteString("  transforms/     # Transform packages (CloudQuery, dbt, Go, Python)\n")
	b.WriteString("```\n\n")
	b.WriteString("## Getting Started\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Validate all manifests\n")
	b.WriteString("dk lint\n\n")
	b.WriteString("# View the pipeline dependency graph\n")
	b.WriteString("dk pipeline show --scan-dir .\n\n")
	b.WriteString("# Start local development environment\n")
	b.WriteString("dk dev up\n\n")
	b.WriteString("# Run a transform locally\n")
	b.WriteString("dk run ./transforms/<name>\n")
	b.WriteString("```\n")

	return b.String()
}

func generateProjectGitignore() string {
	return `# DataKit
*.bak
dk.yaml.bak
.dk/

# Build artifacts
bin/
dist/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Secrets (never commit)
*.env
.env.*
`
}
