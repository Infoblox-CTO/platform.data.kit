package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// RunBatch executes a batch pipeline that runs to completion.
// Batch pipelines are expected to start, process data, and exit with a status code.
func (r *DockerRunner) RunBatch(ctx context.Context, opts RunOptions, image string, result *RunResult, args []string) error {
	// Apply timeout if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "docker", args...)

	if opts.Output != nil {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output
		fmt.Fprintf(opts.Output, "Running batch pipeline: docker %s\n\n", strings.Join(args, " "))
	}

	result.Status = contracts.RunStatusRunning

	if err := cmd.Start(); err != nil {
		result.Status = contracts.RunStatusFailed
		result.Error = err.Error()
		return err
	}

	// Wait for completion
	err := cmd.Wait()
	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(result.StartTime)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Status = contracts.RunStatusFailed
			result.Error = fmt.Sprintf("timeout after %s", opts.Timeout)
			return fmt.Errorf("batch pipeline timed out after %s", opts.Timeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Status = contracts.RunStatusFailed
		result.Error = err.Error()
		return err
	}

	result.Status = contracts.RunStatusCompleted
	result.ExitCode = 0
	return nil
}

// IsBatchMode checks if a pipeline is configured for batch mode.
func IsBatchMode(mode contracts.PipelineMode) bool {
	return mode == "" || mode == contracts.PipelineModeBatch
}
