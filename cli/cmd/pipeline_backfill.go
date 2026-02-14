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
	pipelineBackfillFrom string
	pipelineBackfillTo   string
	pipelineBackfillEnv  []string
)

// pipelineBackfillCmd re-executes sync steps with a date range.
var pipelineBackfillCmd = &cobra.Command{
	Use:   "backfill [pipeline-dir]",
	Short: "Re-execute sync steps for a date range",
	Long: `Re-execute sync steps in a pipeline workflow with a date range
injected as DP_BACKFILL_FROM and DP_BACKFILL_TO environment variables.

Only sync steps are executed; transform, test, publish, and custom steps
are skipped.

Examples:
  # Backfill January 2026
  dp pipeline backfill --from 2026-01-01 --to 2026-01-31

  # Backfill with extra env vars
  dp pipeline backfill --from 2026-01-01 --to 2026-01-31 --env DEBUG=true`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPipelineBackfill,
}

func init() {
	pipelineCmd.AddCommand(pipelineBackfillCmd)

	pipelineBackfillCmd.Flags().StringVar(&pipelineBackfillFrom, "from", "",
		"Start date (YYYY-MM-DD, required)")
	pipelineBackfillCmd.Flags().StringVar(&pipelineBackfillTo, "to", "",
		"End date (YYYY-MM-DD, required)")
	pipelineBackfillCmd.Flags().StringArrayVarP(&pipelineBackfillEnv, "env", "e", nil,
		"Environment variables (KEY=VALUE)")

	_ = pipelineBackfillCmd.MarkFlagRequired("from")
	_ = pipelineBackfillCmd.MarkFlagRequired("to")
}

func runPipelineBackfill(cmd *cobra.Command, args []string) error {
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
	for _, e := range pipelineBackfillEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid env var format %q, expected KEY=VALUE", e)
		}
		env[parts[0]] = parts[1]
	}

	// Execute backfill
	result, err := pipeline.Backfill(context.Background(), pipeline.BackfillOpts{
		PipelineDir: pipelineDir,
		From:        pipelineBackfillFrom,
		To:          pipelineBackfillTo,
		Env:         env,
		Output:      cmd.OutOrStdout(),
	})
	if err != nil {
		return err
	}

	// Print summary
	cmd.Printf("\n--- Backfill Summary ---\n")
	cmd.Printf("Pipeline: %s\n", result.PipelineName)
	cmd.Printf("Range:    %s → %s\n", pipelineBackfillFrom, pipelineBackfillTo)
	cmd.Printf("Status:   %s\n", result.Status)
	cmd.Printf("Duration: %s\n", result.Duration)
	cmd.Println()

	for _, step := range result.Steps {
		icon := "✓"
		if step.Status == contracts.StepStatusFailed {
			icon = "✗"
		}
		line := fmt.Sprintf("  %s %s (%s)", icon, step.Name, step.Status)
		if step.Duration != "" {
			line += fmt.Sprintf(" [%s]", step.Duration)
		}
		cmd.Println(line)
	}

	if result.Status == contracts.StepStatusFailed {
		return fmt.Errorf("backfill failed at step %q", result.FailedStep)
	}

	return nil
}
