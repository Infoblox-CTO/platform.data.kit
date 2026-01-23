package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/lineage"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

func init() {
	RegisterRunner("docker", NewDockerRunner)
}

// DockerRunner executes pipelines using Docker.
type DockerRunner struct {
	mu   sync.RWMutex
	runs map[string]*RunResult
}

// NewDockerRunner creates a new Docker-based runner.
func NewDockerRunner() (Runner, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("docker not found in PATH: %w", err)
	}

	return &DockerRunner{
		runs: make(map[string]*RunResult),
	}, nil
}

// Run executes a pipeline using Docker.
func (r *DockerRunner) Run(ctx context.Context, opts RunOptions) (*RunResult, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	dpPath := filepath.Join(opts.PackageDir, "dp.yaml")
	dpData, err := os.ReadFile(dpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dp.yaml: %w", err)
	}

	parser := manifest.NewParser()
	pkg, err := parser.ParseDataPackage(dpData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	var image string
	var runtimeEnv []contracts.EnvVar

	// Get runtime configuration from dp.yaml
	if pkg.Spec.Runtime != nil && pkg.Spec.Runtime.Image != "" {
		image = pkg.Spec.Runtime.Image
		runtimeEnv = pkg.Spec.Runtime.Env
	}

	runID := GenerateRunID(pkg.Metadata.Name)
	jobNamespace := pkg.Metadata.Namespace
	jobName := pkg.Metadata.Name

	result := &RunResult{
		RunID:     runID,
		Status:    contracts.RunStatusPending,
		StartTime: time.Now(),
	}

	r.mu.Lock()
	r.runs[runID] = result
	r.mu.Unlock()

	// Helper to emit lineage events
	emitLineage := func(eventType lineage.EventType, runErr error) {
		if opts.LineageEmitter == nil {
			return
		}
		event := lineage.NewEvent(eventType, runID, jobNamespace, jobName)

		// Add input datasets
		for _, input := range pkg.Spec.Inputs {
			dataset := lineage.NewDataset(jobNamespace, input.Name)
			event.AddInput(dataset)
		}

		// Add output datasets
		for _, output := range pkg.Spec.Outputs {
			dataset := lineage.NewDataset(jobNamespace, output.Name)
			event.AddOutput(dataset)
		}

		// Add error facet for failures
		if runErr != nil && eventType == lineage.EventTypeFail {
			event.WithErrorFacet(runErr.Error(), string(debug.Stack()))
		}

		if err := opts.LineageEmitter.Emit(ctx, event); err != nil && opts.Output != nil {
			fmt.Fprintf(opts.Output, "Warning: failed to emit lineage event: %v\n", err)
		}
	}

	if image == "" {
		dockerfile := filepath.Join(opts.PackageDir, "Dockerfile")
		if _, err := os.Stat(dockerfile); err == nil {
			imageName := fmt.Sprintf("dp/%s:%s", pkg.Metadata.Name, pkg.Metadata.Version)
			if err := r.buildImage(ctx, opts.PackageDir, imageName, opts.Output); err != nil {
				result.Status = contracts.RunStatusFailed
				result.Error = fmt.Sprintf("failed to build image: %v", err)
				emitLineage(lineage.EventTypeFail, err)
				return result, err
			}
			image = imageName
		} else {
			return nil, fmt.Errorf("no image specified and no Dockerfile found")
		}
	}

	if opts.DryRun {
		result.Status = contracts.RunStatusCompleted
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Dry run complete. Would run image: %s\n", image)
		}
		return result, nil
	}

	args := []string{"run", "--rm"}
	args = append(args, "--name", runID)

	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}

	// Add binding-derived env vars
	bindingEnvs, err := r.buildEnvVarsFromPackage(opts.PackageDir)
	if err != nil && opts.Output != nil {
		fmt.Fprintf(opts.Output, "Warning: failed to map bindings to env vars: %v\n", err)
	}
	for k, v := range bindingEnvs {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add runtime env vars (override bindings)
	for _, env := range runtimeEnv {
		if env.Value != "" {
			args = append(args, "-e", fmt.Sprintf("%s=%s", env.Name, env.Value))
		}
	}

	// Add opts env vars (override all)
	for k, v := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	absPackageDir, _ := filepath.Abs(opts.PackageDir)
	args = append(args, "-v", fmt.Sprintf("%s:/app/package:ro", absPackageDir))

	if opts.Detach {
		args = append(args, "-d")
	}

	args = append(args, image)

	// Emit START lineage event
	emitLineage(lineage.EventTypeStart, nil)

	result.Status = contracts.RunStatusRunning

	cmd := exec.CommandContext(ctx, "docker", args...)

	if !opts.Detach && opts.Output != nil {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output
	}

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Running: docker %s\n\n", strings.Join(args, " "))
	}

	if err := cmd.Start(); err != nil {
		result.Status = contracts.RunStatusFailed
		result.Error = err.Error()
		emitLineage(lineage.EventTypeFail, err)
		return result, err
	}

	if opts.Detach {
		result.ContainerID = runID
		result.Status = contracts.RunStatusRunning
		return result, nil
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Status = contracts.RunStatusFailed
		result.Error = err.Error()
		emitLineage(lineage.EventTypeFail, err)
	} else {
		result.Status = contracts.RunStatusCompleted
		result.ExitCode = 0
		emitLineage(lineage.EventTypeComplete, nil)
	}

	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = endTime.Sub(result.StartTime)

	return result, nil
}

// Stop stops a running pipeline.
func (r *DockerRunner) Stop(ctx context.Context, runID string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", runID)
	return cmd.Run()
}

// Logs streams logs from a pipeline run.
func (r *DockerRunner) Logs(ctx context.Context, runID string, follow bool, output io.Writer) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, runID)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}

// Status returns the status of a pipeline run.
func (r *DockerRunner) Status(ctx context.Context, runID string) (*RunResult, error) {
	r.mu.RLock()
	result, ok := r.runs[runID]
	r.mu.RUnlock()

	if !ok {
		result = &RunResult{
			RunID: runID,
		}
	}

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", runID)
	out, err := cmd.Output()
	if err != nil {
		if result.Status == "" {
			result.Status = contracts.RunStatus("unknown")
		}
		return result, nil
	}

	status := strings.TrimSpace(string(out))
	switch status {
	case "running":
		result.Status = contracts.RunStatusRunning
	case "exited":
		result.Status = contracts.RunStatusCompleted
		exitCmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.ExitCode}}", runID)
		exitOut, _ := exitCmd.Output()
		if strings.TrimSpace(string(exitOut)) != "0" {
			result.Status = contracts.RunStatusFailed
		}
	case "created":
		result.Status = contracts.RunStatusPending
	default:
		result.Status = contracts.RunStatus(status)
	}

	return result, nil
}

// buildImage builds a Docker image from a Dockerfile.
func (r *DockerRunner) buildImage(ctx context.Context, dir, imageName string, output io.Writer) error {
	args := []string{"build", "-t", imageName, "."}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = dir

	if output != nil {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		go func() {
			scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
			for scanner.Scan() {
				fmt.Fprintln(output, scanner.Text())
			}
		}()

		return cmd.Wait()
	}

	return cmd.Run()
}

// buildEnvVarsFromPackage reads the package and bindings, then maps binding properties
// to environment variables automatically.
func (r *DockerRunner) buildEnvVarsFromPackage(packageDir string) (map[string]string, error) {
	dpPath := filepath.Join(packageDir, "dp.yaml")
	dpData, err := os.ReadFile(dpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dp.yaml: %w", err)
	}

	parser := manifest.NewParser()
	pkg, err := parser.ParseDataPackage(dpData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	// Read bindings if they exist
	bindingsPath := filepath.Join(packageDir, "bindings.yaml")
	var bindings []contracts.Binding
	if _, err := os.Stat(bindingsPath); err == nil {
		bindings, err = manifest.ParseBindingsFile(bindingsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bindings.yaml: %w", err)
		}
	}

	// Map bindings to env vars
	bindingProps, _ := MapBindingsToEnvVars(pkg, bindings)

	// Get explicit env vars from runtime
	explicitEnvs := EnvVarsFromRuntime(pkg.Spec.Runtime)

	// Merge: explicit env vars override binding-derived ones
	return MergeEnvVars(bindingProps, explicitEnvs), nil
}
