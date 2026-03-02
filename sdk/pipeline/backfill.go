package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

const (
	// BackfillDateFormat is the expected date format for backfill range boundaries.
	BackfillDateFormat = "2006-01-02"

	// EnvBackfillFrom is the env var injected into sync steps for the backfill start date.
	EnvBackfillFrom = "DK_BACKFILL_FROM"

	// EnvBackfillTo is the env var injected into sync steps for the backfill end date.
	EnvBackfillTo = "DK_BACKFILL_TO"
)

// BackfillOpts configures pipeline backfill execution.
type BackfillOpts struct {
	// PipelineDir is the directory containing pipeline.yaml.
	PipelineDir string

	// From is the backfill start date (YYYY-MM-DD).
	From string

	// To is the backfill end date (YYYY-MM-DD).
	To string

	// Env is additional environment variables.
	Env map[string]string

	// Output is where step logs are written (default: os.Stdout).
	Output io.Writer

	// CommandRunner is the function to run external commands.
	// If nil, uses the default implementation.
	CommandRunner CommandRunnerFunc
}

// Backfill re-executes sync steps in a pipeline with date range env vars injected.
func Backfill(ctx context.Context, opts BackfillOpts) (*contracts.PipelineRunResult, error) {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	// Validate dates
	fromDate, err := time.Parse(BackfillDateFormat, opts.From)
	if err != nil {
		return nil, fmt.Errorf("invalid --from date %q: expected format YYYY-MM-DD", opts.From)
	}
	toDate, err := time.Parse(BackfillDateFormat, opts.To)
	if err != nil {
		return nil, fmt.Errorf("invalid --to date %q: expected format YYYY-MM-DD", opts.To)
	}
	if !fromDate.Before(toDate) {
		return nil, fmt.Errorf("--from date %s must be before --to date %s", opts.From, opts.To)
	}

	// Load pipeline
	workflow, err := LoadPipeline(opts.PipelineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load pipeline: %w", err)
	}

	// Find sync steps
	var syncSteps []contracts.Step
	for _, step := range workflow.Steps {
		if step.Type == contracts.StepTypeSync {
			syncSteps = append(syncSteps, step)
		}
	}
	if len(syncSteps) == 0 {
		return nil, fmt.Errorf("pipeline %q has no sync steps to backfill", workflow.Metadata.Name)
	}

	// Build env with backfill dates
	env := make(map[string]string)
	for k, v := range opts.Env {
		env[k] = v
	}
	env[EnvBackfillFrom] = opts.From
	env[EnvBackfillTo] = opts.To

	// Execute sync steps
	pipelineStart := time.Now()
	result := &contracts.PipelineRunResult{
		PipelineName: workflow.Metadata.Name,
		Status:       contracts.StepStatusCompleted,
		Steps:        make([]contracts.StepResult, 0, len(syncSteps)),
	}

	execOpts := ExecuteOpts{
		PipelineDir:   opts.PipelineDir,
		Env:           env,
		Output:        opts.Output,
		CommandRunner: opts.CommandRunner,
	}

	failed := false
	for _, step := range syncSteps {
		if failed {
			result.Steps = append(result.Steps, contracts.StepResult{
				Name:   step.Name,
				Type:   step.Type,
				Status: contracts.StepStatusSkipped,
			})
			continue
		}

		stepResult := executeStep(ctx, step, execOpts)
		result.Steps = append(result.Steps, stepResult)

		if stepResult.Status == contracts.StepStatusFailed {
			failed = true
			result.Status = contracts.StepStatusFailed
			result.FailedStep = step.Name
		}
	}

	result.Duration = time.Since(pipelineStart).Round(time.Millisecond).String()
	return result, nil
}
