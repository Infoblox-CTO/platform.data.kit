package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
	"github.com/spf13/cobra"
)

var (
	pipelineRunEnv  []string
	pipelineRunStep string
)

// pipelineRunCmd executes a pipeline workflow.
var pipelineRunCmd = &cobra.Command{
	Use:   "run [pipeline-dir]",
	Short: "Execute a pipeline workflow",
	Long: `Execute a pipeline workflow defined in pipeline.yaml.

Steps are executed sequentially. If any step fails, remaining steps are
skipped and the pipeline is marked as failed.

Each step's output is prefixed with [step-name] for easy identification.

Examples:
  # Run pipeline in current directory
  dk pipeline run

  # Run pipeline in a specific directory
  dk pipeline run ./my-pipeline

  # Pass environment variables
  dk pipeline run --env KEY=VALUE --env OTHER=VAL

  # Run a single step
  dk pipeline run --step sync-data`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPipelineRun,
}

func init() {
	pipelineCmd.AddCommand(pipelineRunCmd)

	pipelineRunCmd.Flags().StringArrayVarP(&pipelineRunEnv, "env", "e", nil,
		"Environment variables (KEY=VALUE)")
	pipelineRunCmd.Flags().StringVar(&pipelineRunStep, "step", "",
		"Run only the named step")
}

func runPipelineRun(cmd *cobra.Command, args []string) error {
	// Determine pipeline directory
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	pipelineDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	// Parse env vars
	env := make(map[string]string)
	for _, e := range pipelineRunEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid env var format %q, expected KEY=VALUE", e)
		}
		env[parts[0]] = parts[1]
	}

	// Execute
	result, err := pipeline.Execute(context.Background(), pipeline.ExecuteOpts{
		PipelineDir: pipelineDir,
		Env:         env,
		StepFilter:  pipelineRunStep,
		Output:      cmd.OutOrStdout(),
	})
	if err != nil {
		return err
	}

	// Print summary
	cmd.Printf("\n--- Pipeline Run Summary ---\n")
	cmd.Printf("Pipeline: %s\n", result.PipelineName)
	cmd.Printf("Status:   %s\n", result.Status)
	cmd.Printf("Duration: %s\n", result.Duration)
	cmd.Println()

	for _, step := range result.Steps {
		icon := "✓"
		switch step.Status {
		case contracts.StepStatusFailed:
			icon = "✗"
		case contracts.StepStatusSkipped:
			icon = "⊘"
		}
		line := fmt.Sprintf("  %s %s (%s)", icon, step.Name, step.Status)
		if step.Duration != "" {
			line += fmt.Sprintf(" [%s]", step.Duration)
		}
		if step.Error != "" {
			line += fmt.Sprintf(" — %s", step.Error)
		}
		cmd.Println(line)
	}

	if result.Status == contracts.StepStatusFailed {
		return fmt.Errorf("pipeline failed at step %q", result.FailedStep)
	}

	return nil
}
