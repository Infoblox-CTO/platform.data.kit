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

	m, kind, err := manifest.ParseManifest(dpData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	var image string
	var runtimeEnv []contracts.EnvVar
	var mode contracts.Mode
	var timeout string
	var runtime contracts.Runtime
	var inputs, outputs []contracts.ArtifactContract

	// Extract kind-specific fields
	switch kind {
	case contracts.KindModel:
		model := m.(*contracts.Model)
		image = model.Spec.Image
		runtimeEnv = model.Spec.Env
		mode = model.Spec.Mode
		timeout = model.Spec.Timeout
		runtime = model.Spec.Runtime
		inputs = model.Spec.Inputs
		outputs = model.Spec.Outputs
	case contracts.KindSource:
		src := m.(*contracts.Source)
		image = src.Spec.Image
		runtime = src.Spec.Runtime
	case contracts.KindDestination:
		dst := m.(*contracts.Destination)
		image = dst.Spec.Image
		runtime = dst.Spec.Runtime
	}

	// Handle cloudquery runtime - run via cloudquery CLI instead of Docker
	if runtime == contracts.RuntimeCloudQuery {
		return r.runCloudQuery(ctx, opts, m)
	}

	if runtime == contracts.RuntimeDBT {
		return r.runDBT(ctx, opts, m)
	}

	// Expand environment variables in image name
	image = os.ExpandEnv(image)

	// If image still contains unexpanded variables (e.g., ${REGISTRY}), treat as empty
	// This allows local development to fall back to building from Dockerfile
	if strings.Contains(image, "${") || strings.Contains(image, "$") {
		image = ""
	}

	// Validate the expanded image reference is usable
	// An image like "/foo1:" or empty segments means env vars weren't set
	if image != "" && !isValidImageReference(image) {
		image = ""
	}

	runID := GenerateRunID(m.GetName())
	jobNamespace := m.GetNamespace()
	jobName := m.GetName()

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
		for _, input := range inputs {
			dataset := lineage.NewDataset(jobNamespace, input.Name)
			event.AddInput(dataset)
		}

		// Add output datasets
		for _, output := range outputs {
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
		// Detect language and generate Dockerfile internally
		lang := detectPipelineLanguage(opts.PackageDir)
		dockerfileContent := generateDockerfile(lang, opts.PackageDir)

		// Create temp directory for build context
		tempDir, err := os.MkdirTemp("", "dp-build-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp build directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Write Dockerfile to temp location
		dockerfilePath := filepath.Join(tempDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write Dockerfile: %w", err)
		}

		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Generated Dockerfile: %s\n", dockerfilePath)
		}

		// Build version tag with git revision
		versionTag := buildVersionTag(m.GetVersion(), opts.PackageDir)
		imageName := fmt.Sprintf("dp/%s:%s", m.GetName(), versionTag)
		if err := r.buildImageWithDockerfile(ctx, opts.PackageDir, dockerfilePath, imageName, opts.Output); err != nil {
			result.Status = contracts.RunStatusFailed
			result.Error = fmt.Sprintf("failed to build image: %v", err)
			emitLineage(lineage.EventTypeFail, err)
			return result, err
		}
		image = imageName
	}

	// Determine execution mode (defaults to batch)
	execMode := mode.Default()

	// Apply timeout if not set in opts
	if opts.Timeout == 0 && timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			opts.Timeout = d
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

	args = append(args, image)

	// Emit START lineage event
	emitLineage(lineage.EventTypeStart, nil)

	// Dispatch based on execution mode
	var runErr error
	if IsStreamingMode(execMode) {
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Running streaming pipeline (mode: streaming)\n")
		}
		runErr = r.RunStreaming(ctx, opts, image, result, args)
	} else {
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Running batch pipeline (mode: batch)\n")
		}
		runErr = r.RunBatch(ctx, opts, image, result, args)
	}

	// Emit completion lineage event
	if runErr != nil {
		emitLineage(lineage.EventTypeFail, runErr)
	} else if result.Status == contracts.RunStatusCompleted {
		emitLineage(lineage.EventTypeComplete, nil)
	}

	return result, runErr
}

// runCloudQuery executes a CloudQuery pipeline by invoking the cloudquery CLI
// directly instead of building a Docker image. CloudQuery packages are
// config-only (config.yaml) and do not contain source code.
func (r *DockerRunner) runCloudQuery(ctx context.Context, opts RunOptions, m manifest.Manifest) (*RunResult, error) {
	cqBin, err := exec.LookPath("cloudquery")
	if err != nil {
		return nil, fmt.Errorf("cloudquery CLI not found in PATH: install it from https://www.cloudquery.io/docs/quickstart\n  %w", err)
	}

	configPath := filepath.Join(opts.PackageDir, "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("config.yaml not found in %s: %w", opts.PackageDir, err)
	}

	runID := GenerateRunID(m.GetName())
	result := &RunResult{
		RunID:     runID,
		Status:    contracts.RunStatusRunning,
		StartTime: time.Now(),
	}

	r.mu.Lock()
	r.runs[runID] = result
	r.mu.Unlock()

	if opts.DryRun {
		result.Status = contracts.RunStatusCompleted
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Dry run complete. Would run: %s sync config.yaml\n", cqBin)
		}
		return result, nil
	}

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Running CloudQuery sync (%s)...\n", m.GetName())
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, cqBin, "sync", "config.yaml")
	cmd.Dir = opts.PackageDir

	// Merge environment: inherit current env, add binding/opts env vars
	cmd.Env = os.Environ()
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if opts.Output != nil {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output
	}

	runErr := cmd.Run()

	now := time.Now()
	result.EndTime = &now
	result.Duration = now.Sub(result.StartTime)

	if runErr != nil {
		result.Status = contracts.RunStatusFailed
		result.Error = runErr.Error()
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	} else {
		result.Status = contracts.RunStatusCompleted
	}

	return result, runErr
}

// runDBT executes a dbt pipeline by invoking the dbt CLI directly instead of
// building a Docker image. dbt packages contain SQL/YAML transformations.
func (r *DockerRunner) runDBT(ctx context.Context, opts RunOptions, m manifest.Manifest) (*RunResult, error) {
	dbtBin, err := exec.LookPath("dbt")
	if err != nil {
		return nil, fmt.Errorf("dbt CLI not found in PATH: install it from https://docs.getdbt.com/docs/core/installation-overview\n  %w", err)
	}

	runID := GenerateRunID(m.GetName())
	result := &RunResult{
		RunID:     runID,
		Status:    contracts.RunStatusRunning,
		StartTime: time.Now(),
	}

	r.mu.Lock()
	r.runs[runID] = result
	r.mu.Unlock()

	if opts.DryRun {
		result.Status = contracts.RunStatusCompleted
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "Dry run complete. Would run: %s run\n", dbtBin)
		}
		return result, nil
	}

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Running dbt (%s)...\n", m.GetName())
	}

	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, dbtBin, "run")
	cmd.Dir = opts.PackageDir

	// Merge environment: inherit current env, add binding/opts env vars
	cmd.Env = os.Environ()
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if opts.Output != nil {
		cmd.Stdout = opts.Output
		cmd.Stderr = opts.Output
	}

	runErr := cmd.Run()

	now := time.Now()
	result.EndTime = &now
	result.Duration = now.Sub(result.StartTime)

	if runErr != nil {
		result.Status = contracts.RunStatusFailed
		result.Error = runErr.Error()
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
	} else {
		result.Status = contracts.RunStatusCompleted
	}

	return result, runErr
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

	m, kind, err := manifest.ParseManifest(dpData)
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

	// Map bindings to env vars (only Model has inputs/outputs)
	bindingProps, _ := MapBindingsToEnvVars(m, kind, bindings)

	// Get explicit env vars from manifest
	explicitEnvs := EnvVarsFromManifest(m, kind)

	// Merge: explicit env vars override binding-derived ones
	return MergeEnvVars(bindingProps, explicitEnvs), nil
}

// isValidImageReference checks if an image reference is valid for docker run.
// Returns false if the image has empty components (e.g., "/foo:" from unexpanded vars).
func isValidImageReference(image string) bool {
	if image == "" {
		return false
	}

	// Check for leading slash (invalid: /foo:tag)
	if strings.HasPrefix(image, "/") {
		return false
	}

	// Check for empty tag (invalid: foo:)
	if strings.HasSuffix(image, ":") {
		return false
	}

	// Check for double slashes (invalid: registry//image)
	if strings.Contains(image, "//") {
		return false
	}

	// Check for empty segments between colons/slashes
	parts := strings.Split(image, "/")
	for _, part := range parts {
		if part == "" {
			return false
		}
	}

	return true
}

// buildVersionTag creates a version tag that includes git revision information.
// Format: <version>-<short-sha>[-dirty]
func buildVersionTag(baseVersion, packageDir string) string {
	gitVersion := getGitVersion(packageDir)
	if gitVersion == "" {
		return baseVersion
	}
	return fmt.Sprintf("%s-%s", baseVersion, gitVersion)
}

// getGitVersion returns the git short SHA and dirty status.
// Returns empty string if not in a git repository.
func getGitVersion(dir string) string {
	// Get short commit hash
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	sha := strings.TrimSpace(string(output))

	// Check if working directory is dirty
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	output, err = cmd.Output()
	if err != nil {
		return sha
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		return sha + "-dirty"
	}
	return sha
}

// detectPipelineLanguage detects the programming language of a pipeline package.
// It checks both the legacy src/ directory layout and the new root-level layout.
func detectPipelineLanguage(packageDir string) string {
	srcDir := filepath.Join(packageDir, "src")

	// Check for Python (src/ layout first, then root)
	if _, err := os.Stat(filepath.Join(srcDir, "main.py")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(srcDir, "requirements.txt")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "main.py")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "requirements.txt")); err == nil {
		return "python"
	}

	// Check for Go (src/ layout first, then cmd/ layout, then root)
	if _, err := os.Stat(filepath.Join(srcDir, "main.go")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(srcDir, "go.mod")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "cmd", "main.go")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "main.go")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(packageDir, "go.mod")); err == nil {
		return "go"
	}

	// Default to Go
	return "go"
}

// hasSrcDir checks if the package directory has a legacy src/ subdirectory.
func hasSrcDir(packageDir string) bool {
	info, err := os.Stat(filepath.Join(packageDir, "src"))
	return err == nil && info.IsDir()
}

// readModulePath reads the module path from go.mod in the given directory.
// Returns the module path (e.g. "my-model") or empty string if go.mod is missing.
func readModulePath(packageDir string) string {
	data, err := os.ReadFile(filepath.Join(packageDir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// detectGoBuildTarget determines the correct `go build` target path for a Go
// project. It inspects the directory structure to distinguish between:
//   - Cobra-style layout:  main.go at root, cmd/ is a library package → "."
//   - cmd/ entrypoint:     cmd/main.go is package main, root has no main.go → "./cmd"
//   - flat layout:         main.go at root, no cmd/ → "."
func detectGoBuildTarget(packageDir string) string {
	// Root main.go always wins — this is the standard Go / Cobra convention.
	if _, err := os.Stat(filepath.Join(packageDir, "main.go")); err == nil {
		return "."
	}
	// Check for a main.go inside cmd/ as the entrypoint.
	if _, err := os.Stat(filepath.Join(packageDir, "cmd", "main.go")); err == nil {
		return "./cmd"
	}
	// Fallback: build from root.
	return "."
}

// generateDockerfile generates a Dockerfile for the given language.
// It detects the project layout (src/, cmd/, or flat) and reads go.mod
// to determine the correct build target so imports resolve properly.
func generateDockerfile(lang, packageDir string) string {
	useSrcLayout := hasSrcDir(packageDir)

	switch lang {
	case "python":
		if useSrcLayout {
			return `# DP Pipeline Image (auto-generated)
ARG DP_BASE_IMAGE=python:3.11-slim

FROM python:3.11-slim AS builder
WORKDIR /build
COPY src/requirements.txt ./
RUN pip install --no-cache-dir --target=/deps -r requirements.txt || true

FROM ${DP_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /deps /app/deps
ENV PYTHONPATH=/app/deps
COPY src/ /app/src/
COPY dp.yaml /app/
ENTRYPOINT ["python", "/app/src/main.py"]
`
		}
		return `# DP Pipeline Image (auto-generated)
ARG DP_BASE_IMAGE=python:3.11-slim

FROM python:3.11-slim AS builder
WORKDIR /build
COPY requirements.txt ./
RUN pip install --no-cache-dir --target=/deps -r requirements.txt || true

FROM ${DP_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /deps /app/deps
ENV PYTHONPATH=/app/deps
COPY . /app/
COPY dp.yaml /app/
ENTRYPOINT ["python", "/app/main.py"]
`
	default: // go
		if useSrcLayout {
			return `# DP Pipeline Image (auto-generated)
ARG DP_BASE_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY src/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pipeline .

FROM ${DP_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /pipeline /app/pipeline
COPY dp.yaml /app/
ENTRYPOINT ["/app/pipeline"]
`
		}
		buildTarget := detectGoBuildTarget(packageDir)
		return fmt.Sprintf(`# DP Pipeline Image (auto-generated)
ARG DP_BASE_IMAGE=gcr.io/distroless/static-debian12:nonroot

FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pipeline %s

FROM ${DP_BASE_IMAGE}
WORKDIR /app
COPY --from=builder /pipeline /app/pipeline
COPY dp.yaml /app/
ENTRYPOINT ["/app/pipeline"]
`, buildTarget)
	}
}

// buildImageWithDockerfile builds a Docker image using an external Dockerfile path.
func (r *DockerRunner) buildImageWithDockerfile(ctx context.Context, contextDir, dockerfilePath, imageName string, output io.Writer) error {
	args := []string{"build", "-t", imageName, "-f", dockerfilePath, "."}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = contextDir

	if output != nil {
		cmd.Stdout = output
		cmd.Stderr = output
	}

	return cmd.Run()
}
