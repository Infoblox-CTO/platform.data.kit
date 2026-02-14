package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
	"github.com/spf13/cobra"
)

var (
	pipelineCreateTemplate      string
	pipelineCreateForce         bool
	pipelineCreateListTemplates bool
)

// pipelineCreateCmd scaffolds a new pipeline.yaml from a template.
var pipelineCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new pipeline.yaml from a template",
	Long: `Scaffold a new pipeline.yaml in the current directory from a built-in template.

Available templates:
  sync-transform-test  Full ETL pipeline: sync → transform → test → publish (default)
  sync-only            Simple sync-only pipeline
  custom               Single custom step for bespoke workloads

Examples:
  # Create with default template
  dp pipeline create my-pipeline

  # Create with specific template
  dp pipeline create my-pipeline --template sync-only

  # List available templates
  dp pipeline create --list-templates

  # Force overwrite existing pipeline.yaml
  dp pipeline create my-pipeline --force`,
	Args: func(cmd *cobra.Command, args []string) error {
		if pipelineCreateListTemplates {
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: runPipelineCreate,
}

func init() {
	pipelineCmd.AddCommand(pipelineCreateCmd)

	pipelineCreateCmd.Flags().StringVarP(&pipelineCreateTemplate, "template", "t", "sync-transform-test",
		"Pipeline template name")
	pipelineCreateCmd.Flags().BoolVar(&pipelineCreateForce, "force", false,
		"Overwrite existing pipeline.yaml")
	pipelineCreateCmd.Flags().BoolVar(&pipelineCreateListTemplates, "list-templates", false,
		"List available pipeline templates")
}

func runPipelineCreate(cmd *cobra.Command, args []string) error {
	// Handle --list-templates
	if pipelineCreateListTemplates {
		templates, err := pipeline.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}
		cmd.Println("Available pipeline templates:")
		for _, t := range templates {
			cmd.Printf("  %s\n", t)
		}
		return nil
	}

	name := args[0]

	// Determine project directory
	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Validate template name
	templates, err := pipeline.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}
	validTemplate := false
	for _, t := range templates {
		if t == pipelineCreateTemplate {
			validTemplate = true
			break
		}
	}
	if !validTemplate {
		return fmt.Errorf("unknown template %q; available templates: %s",
			pipelineCreateTemplate, strings.Join(templates, ", "))
	}

	// Scaffold
	outputPath, err := pipeline.ScaffoldPipeline(pipeline.ScaffoldOpts{
		Name:       name,
		Template:   pipelineCreateTemplate,
		ProjectDir: projectDir,
		Force:      pipelineCreateForce,
	})
	if err != nil {
		return err
	}

	relPath, _ := filepath.Rel(projectDir, outputPath)
	if relPath == "" {
		relPath = outputPath
	}

	cmd.Printf("✓ Created pipeline %q at %s\n", name, relPath)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit %s to configure your pipeline steps\n", relPath)
	cmd.Printf("  2. Run 'dp lint' to validate the pipeline\n")
	cmd.Printf("  3. Run 'dp pipeline run' to execute the pipeline\n")

	return nil
}
