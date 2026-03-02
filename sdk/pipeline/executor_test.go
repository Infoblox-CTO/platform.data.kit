package pipeline

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// --- PrefixWriter Tests (T027) ---

func TestPrefixWriter_SingleLine(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixWriter(&buf, "step-1")
	pw.Write([]byte("hello world\n"))
	pw.Flush()

	got := buf.String()
	want := "[step-1] hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixWriter_MultiLine(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixWriter(&buf, "sync")
	pw.Write([]byte("line 1\nline 2\nline 3\n"))
	pw.Flush()

	got := buf.String()
	if !strings.Contains(got, "[sync] line 1\n") {
		t.Errorf("missing prefixed line 1 in %q", got)
	}
	if !strings.Contains(got, "[sync] line 2\n") {
		t.Errorf("missing prefixed line 2 in %q", got)
	}
	if !strings.Contains(got, "[sync] line 3\n") {
		t.Errorf("missing prefixed line 3 in %q", got)
	}
}

func TestPrefixWriter_PartialLines(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixWriter(&buf, "test")

	// Write partial line
	pw.Write([]byte("hel"))
	if buf.String() != "" {
		t.Errorf("expected no output yet, got %q", buf.String())
	}

	// Complete the line
	pw.Write([]byte("lo\n"))
	got := buf.String()
	want := "[test] hello\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixWriter_Empty(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixWriter(&buf, "empty")
	pw.Write([]byte(""))
	pw.Flush()

	if buf.String() != "" {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestPrefixWriter_FlushPartial(t *testing.T) {
	var buf bytes.Buffer
	pw := NewPrefixWriter(&buf, "flush")

	// Write without newline
	pw.Write([]byte("no newline"))
	// Flush should emit the partial line
	pw.Flush()

	got := buf.String()
	want := "[flush] no newline\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- Execute Tests (T030) ---

// fakeCommandRunner returns an exec.Cmd that runs "echo" for success
// or "false" for failure.
func fakeCommandSuccess(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "echo", "step output")
}

func fakeCommandFailure(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "false")
}

// fakeCommandByStep returns different results based on the docker args.
// It inspects args to determine which step is being executed.
func fakeCommandByStep(failStep string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// Look for the step name in the args (via env var pattern)
		argsStr := strings.Join(args, " ")
		if strings.Contains(argsStr, failStep) {
			return exec.CommandContext(ctx, "false")
		}
		return exec.CommandContext(ctx, "echo", "ok")
	}
}

func writePipelineYAML(t *testing.T, dir string, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, PipelineFileName), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestExecute_AllStepsPass(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, `
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: test-pipeline
steps:
  - name: step-custom-one
    type: custom
    image: alpine:latest
  - name: step-custom-two
    type: custom
    image: alpine:latest
`)

	var buf bytes.Buffer
	result, err := Execute(context.Background(), ExecuteOpts{
		PipelineDir:   dir,
		Output:        &buf,
		CommandRunner: fakeCommandSuccess,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Status != contracts.StepStatusCompleted {
		t.Errorf("status = %v, want completed", result.Status)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
	for _, s := range result.Steps {
		if s.Status != contracts.StepStatusCompleted {
			t.Errorf("step %q status = %v, want completed", s.Name, s.Status)
		}
	}
	if result.PipelineName != "test-pipeline" {
		t.Errorf("pipeline name = %q, want %q", result.PipelineName, "test-pipeline")
	}
	if result.Duration == "" {
		t.Error("expected non-empty duration")
	}
}

func TestExecute_StepFailureSkipsRemaining(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, `
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: fail-pipeline
steps:
  - name: step-one
    type: custom
    image: alpine:latest
  - name: step-two
    type: custom
    image: fail-image:latest
  - name: step-three
    type: custom
    image: alpine:latest
`)

	var buf bytes.Buffer
	// step-two will fail because its image name contains "fail"
	runner := fakeCommandByStep("fail-image")
	result, err := Execute(context.Background(), ExecuteOpts{
		PipelineDir:   dir,
		Output:        &buf,
		CommandRunner: runner,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Status != contracts.StepStatusFailed {
		t.Errorf("status = %v, want failed", result.Status)
	}
	if result.FailedStep != "step-two" {
		t.Errorf("failedStep = %q, want %q", result.FailedStep, "step-two")
	}
	if len(result.Steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(result.Steps))
	}
	if result.Steps[0].Status != contracts.StepStatusCompleted {
		t.Errorf("step-one status = %v, want completed", result.Steps[0].Status)
	}
	if result.Steps[1].Status != contracts.StepStatusFailed {
		t.Errorf("step-two status = %v, want failed", result.Steps[1].Status)
	}
	if result.Steps[2].Status != contracts.StepStatusSkipped {
		t.Errorf("step-three status = %v, want skipped", result.Steps[2].Status)
	}
}

func TestExecute_SingleStepFilter(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, `
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: filter-pipeline
steps:
  - name: step-one
    type: custom
    image: alpine:latest
  - name: step-two
    type: custom
    image: alpine:latest
`)

	var buf bytes.Buffer
	result, err := Execute(context.Background(), ExecuteOpts{
		PipelineDir:   dir,
		Output:        &buf,
		StepFilter:    "step-two",
		CommandRunner: fakeCommandSuccess,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(result.Steps))
	}
	if result.Steps[0].Name != "step-two" {
		t.Errorf("step name = %q, want %q", result.Steps[0].Name, "step-two")
	}
}

func TestExecute_StepFilterNotFound(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, `
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: filter-pipeline
steps:
  - name: step-one
    type: custom
    image: alpine:latest
`)

	var buf bytes.Buffer
	_, err := Execute(context.Background(), ExecuteOpts{
		PipelineDir:   dir,
		Output:        &buf,
		StepFilter:    "nonexistent",
		CommandRunner: fakeCommandSuccess,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent step filter, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err)
	}
}

func TestExecute_Cancellation(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, `
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: cancel-pipeline
steps:
  - name: step-one
    type: custom
    image: alpine:latest
  - name: step-two
    type: custom
    image: alpine:latest
`)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	result, err := Execute(ctx, ExecuteOpts{
		PipelineDir:   dir,
		Output:        &buf,
		CommandRunner: fakeCommandSuccess,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// All steps should be skipped due to cancellation
	if result.Status != contracts.StepStatusFailed {
		t.Errorf("status = %v, want failed", result.Status)
	}
	for _, s := range result.Steps {
		if s.Status != contracts.StepStatusSkipped {
			t.Errorf("step %q status = %v, want skipped", s.Name, s.Status)
		}
	}
}

func TestExecute_MissingPipeline(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	_, err := Execute(context.Background(), ExecuteOpts{
		PipelineDir:   dir,
		Output:        &buf,
		CommandRunner: fakeCommandSuccess,
	})
	if err == nil {
		t.Fatal("expected error for missing pipeline.yaml, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load pipeline") {
		t.Errorf("error = %q, want 'failed to load pipeline'", err)
	}
}

func TestExecute_PublishNoOp(t *testing.T) {
	dir := t.TempDir()
	writePipelineYAML(t, dir, `
apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: publish-pipeline
steps:
  - name: notify-team
    type: publish
    promote: true
`)

	var buf bytes.Buffer
	result, err := Execute(context.Background(), ExecuteOpts{
		PipelineDir:   dir,
		Output:        &buf,
		CommandRunner: fakeCommandSuccess,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Status != contracts.StepStatusCompleted {
		t.Errorf("status = %v, want completed", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(result.Steps))
	}
	if result.Steps[0].Status != contracts.StepStatusCompleted {
		t.Errorf("step status = %v, want completed", result.Steps[0].Status)
	}
}

// --- buildStepArgs Tests ---

func TestBuildStepArgs_CustomStep(t *testing.T) {
	step := contracts.Step{
		Name:  "custom-step",
		Type:  contracts.StepTypeCustom,
		Image: "my-image:v1",
		Args:  []string{"--flag", "value"},
		Env: []contracts.EnvVar{
			{Name: "FOO", Value: "bar"},
		},
	}

	args, err := buildStepArgs(step, nil)
	if err != nil {
		t.Fatalf("buildStepArgs error = %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "docker run --rm") {
		t.Errorf("expected docker run --rm, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "FOO=bar") {
		t.Errorf("expected env var FOO=bar, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "my-image:v1") {
		t.Errorf("expected image name, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "--flag value") {
		t.Errorf("expected args, got %q", argsStr)
	}
}

func TestBuildStepArgs_CustomStep_ImageOnly(t *testing.T) {
	step := contracts.Step{
		Name:  "image-only",
		Type:  contracts.StepTypeCustom,
		Image: "alpine:3.18",
	}

	args, err := buildStepArgs(step, nil)
	if err != nil {
		t.Fatalf("buildStepArgs error = %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "alpine:3.18") {
		t.Errorf("expected image, got %q", argsStr)
	}
	// No args should follow the image
	lastArg := args[len(args)-1]
	if lastArg != "alpine:3.18" {
		t.Errorf("last arg = %q, want image name", lastArg)
	}
}

func TestBuildStepArgs_CustomStep_WithCommandAndArgs(t *testing.T) {
	step := contracts.Step{
		Name:  "cmd-step",
		Type:  contracts.StepTypeCustom,
		Image: "python:3.12",
		Args:  []string{"python", "-c", "print('hello')"},
	}

	args, err := buildStepArgs(step, nil)
	if err != nil {
		t.Fatalf("buildStepArgs error = %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "python:3.12 python -c print('hello')") {
		t.Errorf("expected image followed by args, got %q", argsStr)
	}
}

func TestBuildStepArgs_CustomStep_EnvVarsPassedThrough(t *testing.T) {
	step := contracts.Step{
		Name:  "env-step",
		Type:  contracts.StepTypeCustom,
		Image: "img:latest",
		Env: []contracts.EnvVar{
			{Name: "DB_HOST", Value: "localhost"},
			{Name: "DB_PORT", Value: "5432"},
		},
	}

	args, err := buildStepArgs(step, nil)
	if err != nil {
		t.Fatalf("buildStepArgs error = %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "DB_HOST=localhost") {
		t.Errorf("expected DB_HOST, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "DB_PORT=5432") {
		t.Errorf("expected DB_PORT, got %q", argsStr)
	}
}

func TestBuildStepArgs_CustomStepMissingImage(t *testing.T) {
	step := contracts.Step{
		Name: "no-image",
		Type: contracts.StepTypeCustom,
	}

	_, err := buildStepArgs(step, nil)
	if err == nil {
		t.Fatal("expected error for missing image")
	}
}

func TestBuildStepArgs_SyncStep(t *testing.T) {
	step := contracts.Step{
		Name:   "sync-data",
		Type:   contracts.StepTypeSync,
		Input:  "aws-source",
		Output: "postgres-sink",
	}

	args, err := buildStepArgs(step, nil)
	if err != nil {
		t.Fatalf("buildStepArgs error = %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "DK_INPUT=aws-source") {
		t.Errorf("expected DK_INPUT, got %q", argsStr)
	}
	if !strings.Contains(argsStr, "DK_OUTPUT=postgres-sink") {
		t.Errorf("expected DK_OUTPUT, got %q", argsStr)
	}
}

func TestBuildStepArgs_ExtraEnv(t *testing.T) {
	step := contracts.Step{
		Name:  "env-step",
		Type:  contracts.StepTypeCustom,
		Image: "img:latest",
	}

	extra := map[string]string{"EXTRA": "val"}
	args, err := buildStepArgs(step, extra)
	if err != nil {
		t.Fatalf("buildStepArgs error = %v", err)
	}

	argsStr := strings.Join(args, " ")
	if !strings.Contains(argsStr, "EXTRA=val") {
		t.Errorf("expected EXTRA=val, got %q", argsStr)
	}
}

func TestBuildStepArgs_PublishReturnsNil(t *testing.T) {
	step := contracts.Step{
		Name: "publish",
		Type: contracts.StepTypePublish,
	}

	args, err := buildStepArgs(step, nil)
	if err != nil {
		t.Fatalf("buildStepArgs error = %v", err)
	}
	if args != nil {
		t.Errorf("expected nil args for publish, got %v", args)
	}
}

func TestFilterStep_Found(t *testing.T) {
	steps := []contracts.Step{
		{Name: "step-a"},
		{Name: "step-b"},
	}

	found := filterStep(steps, "step-b")
	if found == nil {
		t.Fatal("expected to find step-b")
	}
	if found.Name != "step-b" {
		t.Errorf("name = %q, want %q", found.Name, "step-b")
	}
}

func TestFilterStep_NotFound(t *testing.T) {
	steps := []contracts.Step{
		{Name: "step-a"},
	}

	found := filterStep(steps, "nonexistent")
	if found != nil {
		t.Error("expected nil for nonexistent step")
	}
}
