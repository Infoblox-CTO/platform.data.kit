package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// PrefixWriter wraps an io.Writer and prepends a prefix to each line.
type PrefixWriter struct {
	out    io.Writer
	prefix string
	mu     sync.Mutex
	buf    []byte
}

// NewPrefixWriter creates a writer that prepends [prefix] to each line.
func NewPrefixWriter(out io.Writer, prefix string) *PrefixWriter {
	return &PrefixWriter{
		out:    out,
		prefix: "[" + prefix + "] ",
	}
}

// Write writes p to the underlying writer with the prefix prepended to each line.
func (pw *PrefixWriter) Write(p []byte) (int, error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	pw.buf = append(pw.buf, p...)

	for {
		idx := indexOf(pw.buf, '\n')
		if idx < 0 {
			break
		}
		line := pw.buf[:idx]
		pw.buf = pw.buf[idx+1:]

		if _, err := fmt.Fprintf(pw.out, "%s%s\n", pw.prefix, string(line)); err != nil {
			return len(p), err
		}
	}

	return len(p), nil
}

// Flush writes any remaining buffered content.
func (pw *PrefixWriter) Flush() error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if len(pw.buf) > 0 {
		_, err := fmt.Fprintf(pw.out, "%s%s\n", pw.prefix, string(pw.buf))
		pw.buf = pw.buf[:0]
		return err
	}
	return nil
}

func indexOf(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

// CommandRunnerFunc is the signature for creating exec.Cmd instances.
type CommandRunnerFunc func(ctx context.Context, name string, args ...string) *exec.Cmd

// ExecuteOpts configures pipeline execution.
type ExecuteOpts struct {
	// PipelineDir is the directory containing pipeline.yaml.
	PipelineDir string

	// Env is additional environment variables (override step env).
	Env map[string]string

	// StepFilter runs only the named step (empty = all steps).
	StepFilter string

	// Output is where step logs are written (default: os.Stdout).
	Output io.Writer

	// CommandRunner is the function to run external commands.
	// If nil, uses the default exec.CommandContext implementation.
	CommandRunner CommandRunnerFunc
}

// Execute loads and runs the pipeline workflow sequentially.
// It returns a PipelineRunResult with per-step results.
func Execute(ctx context.Context, opts ExecuteOpts) (*contracts.PipelineRunResult, error) {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.CommandRunner == nil {
		opts.CommandRunner = exec.CommandContext
	}

	workflow, err := LoadPipeline(opts.PipelineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load pipeline: %w", err)
	}

	pipelineStart := time.Now()
	result := &contracts.PipelineRunResult{
		PipelineName: workflow.Metadata.Name,
		Status:       contracts.StepStatusCompleted,
		Steps:        make([]contracts.StepResult, 0, len(workflow.Steps)),
	}

	// Set up cancellation via SIGINT/SIGTERM
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	defer signal.Stop(sigCh)

	// Determine which steps to run
	steps := workflow.Steps
	if opts.StepFilter != "" {
		filtered := filterStep(steps, opts.StepFilter)
		if filtered == nil {
			return nil, fmt.Errorf("step %q not found in pipeline %q", opts.StepFilter, workflow.Metadata.Name)
		}
		steps = []contracts.Step{*filtered}
	}

	failed := false
	for _, step := range steps {
		if failed {
			result.Steps = append(result.Steps, contracts.StepResult{
				Name:   step.Name,
				Type:   step.Type,
				Status: contracts.StepStatusSkipped,
			})
			continue
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			result.Steps = append(result.Steps, contracts.StepResult{
				Name:   step.Name,
				Type:   step.Type,
				Status: contracts.StepStatusSkipped,
			})
			failed = true
			result.Status = contracts.StepStatusFailed
			result.FailedStep = step.Name
			continue
		default:
		}

		stepResult := executeStep(ctx, step, opts)
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

// executeStep runs a single pipeline step.
func executeStep(ctx context.Context, step contracts.Step, opts ExecuteOpts) contracts.StepResult {
	start := time.Now()

	result := contracts.StepResult{
		Name:   step.Name,
		Type:   step.Type,
		Status: contracts.StepStatusRunning,
	}

	pw := NewPrefixWriter(opts.Output, step.Name)

	// Build the command based on step type
	args, err := buildStepArgs(step, opts.Env)
	if err != nil {
		result.Status = contracts.StepStatusFailed
		result.Error = err.Error()
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	if len(args) == 0 {
		// No-op step (e.g., publish with no image)
		fmt.Fprintf(pw, "step %q (%s): no-op\n", step.Name, step.Type)
		_ = pw.Flush()
		result.Status = contracts.StepStatusCompleted
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	cmd := opts.CommandRunner(ctx, args[0], args[1:]...)

	// Pipe stdout/stderr through the prefix writer
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Status = contracts.StepStatusFailed
		result.Error = fmt.Sprintf("stdout pipe: %v", err)
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Status = contracts.StepStatusFailed
		result.Error = fmt.Sprintf("stderr pipe: %v", err)
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	if err := cmd.Start(); err != nil {
		result.Status = contracts.StepStatusFailed
		result.Error = fmt.Sprintf("start: %v", err)
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	// Stream output
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamLines(stdout, pw)
	}()
	go func() {
		defer wg.Done()
		streamLines(stderr, pw)
	}()
	wg.Wait()
	_ = pw.Flush()

	// Wait for command completion
	if err := cmd.Wait(); err != nil {
		result.Status = contracts.StepStatusFailed
		result.Error = err.Error()
		result.Duration = time.Since(start).Round(time.Millisecond).String()
		return result
	}

	result.Status = contracts.StepStatusCompleted
	result.Duration = time.Since(start).Round(time.Millisecond).String()
	return result
}

// buildStepArgs builds the command-line arguments for a step execution.
func buildStepArgs(step contracts.Step, extraEnv map[string]string) ([]string, error) {
	var args []string

	switch step.Type {
	case contracts.StepTypeSync:
		// Sync: docker run with input/output env
		args = append(args, "docker", "run", "--rm")
		args = appendEnvArgs(args, step.Env, extraEnv)
		args = append(args, "-e", fmt.Sprintf("DK_INPUT=%s", step.Input))
		args = append(args, "-e", fmt.Sprintf("DK_OUTPUT=%s", step.Output))
		args = append(args, "dk-sync:latest")

	case contracts.StepTypeTransform:
		// Transform: docker run with asset reference
		args = append(args, "docker", "run", "--rm")
		args = appendEnvArgs(args, step.Env, extraEnv)
		args = append(args, "-e", fmt.Sprintf("DK_ASSET=%s", step.Asset))
		args = append(args, "dk-transform:latest")

	case contracts.StepTypeTest:
		// Test: docker run with command override
		args = append(args, "docker", "run", "--rm")
		args = appendEnvArgs(args, step.Env, extraEnv)
		args = append(args, "-e", fmt.Sprintf("DK_ASSET=%s", step.Asset))
		args = append(args, "dk-test:latest")
		if len(step.Command) > 0 {
			args = append(args, step.Command...)
		}

	case contracts.StepTypeCustom:
		// Custom: user-specified image
		if step.Image == "" {
			return nil, fmt.Errorf("custom step %q requires an image", step.Name)
		}
		args = append(args, "docker", "run", "--rm")
		args = appendEnvArgs(args, step.Env, extraEnv)
		args = append(args, step.Image)
		args = append(args, step.Args...)

	case contracts.StepTypePublish:
		// Publish: no-op by default (notification handled externally)
		return nil, nil

	default:
		return nil, fmt.Errorf("unsupported step type: %s", step.Type)
	}

	return args, nil
}

// appendEnvArgs adds -e flags for step env vars and extra env vars.
func appendEnvArgs(args []string, stepEnv []contracts.EnvVar, extraEnv map[string]string) []string {
	for _, e := range stepEnv {
		if e.Value != "" {
			args = append(args, "-e", fmt.Sprintf("%s=%s", e.Name, e.Value))
		}
	}
	for k, v := range extraEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	return args
}

// streamLines reads lines from r and writes them to w.
func streamLines(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintln(w, scanner.Text())
	}
}

// filterStep returns the step with the given name, or nil if not found.
func filterStep(steps []contracts.Step, name string) *contracts.Step {
	for i := range steps {
		if strings.EqualFold(steps[i].Name, name) {
			return &steps[i]
		}
	}
	return nil
}
