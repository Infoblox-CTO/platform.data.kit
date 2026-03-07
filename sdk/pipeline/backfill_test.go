package pipeline

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackfill_ValidRange(t *testing.T) {
	dir := t.TempDir()
	writePipelineWithSync(t, dir)

	var buf bytes.Buffer
	result, err := Backfill(context.Background(), BackfillOpts{
		PipelineDir:   dir,
		From:          "2026-01-01",
		To:            "2026-01-31",
		Output:        &buf,
		CommandRunner: fakeBackfillSuccess,
	})
	if err != nil {
		t.Fatalf("Backfill() error = %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("status = %v, want completed", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("steps = %d, want 1 (sync step only)", len(result.Steps))
	}
	if result.Steps[0].Name != "sync-data" {
		t.Errorf("step name = %q, want %q", result.Steps[0].Name, "sync-data")
	}
}

func TestBackfill_InvalidFromDate(t *testing.T) {
	dir := t.TempDir()
	writePipelineWithSync(t, dir)

	var buf bytes.Buffer
	_, err := Backfill(context.Background(), BackfillOpts{
		PipelineDir:   dir,
		From:          "not-a-date",
		To:            "2026-01-31",
		Output:        &buf,
		CommandRunner: fakeBackfillSuccess,
	})
	if err == nil {
		t.Fatal("expected error for invalid from date")
	}
	if !strings.Contains(err.Error(), "invalid --from date") {
		t.Errorf("error = %q, want 'invalid --from date'", err)
	}
}

func TestBackfill_InvalidToDate(t *testing.T) {
	dir := t.TempDir()
	writePipelineWithSync(t, dir)

	var buf bytes.Buffer
	_, err := Backfill(context.Background(), BackfillOpts{
		PipelineDir:   dir,
		From:          "2026-01-01",
		To:            "invalid",
		Output:        &buf,
		CommandRunner: fakeBackfillSuccess,
	})
	if err == nil {
		t.Fatal("expected error for invalid to date")
	}
	if !strings.Contains(err.Error(), "invalid --to date") {
		t.Errorf("error = %q, want 'invalid --to date'", err)
	}
}

func TestBackfill_FromAfterTo(t *testing.T) {
	dir := t.TempDir()
	writePipelineWithSync(t, dir)

	var buf bytes.Buffer
	_, err := Backfill(context.Background(), BackfillOpts{
		PipelineDir:   dir,
		From:          "2026-02-01",
		To:            "2026-01-01",
		Output:        &buf,
		CommandRunner: fakeBackfillSuccess,
	})
	if err == nil {
		t.Fatal("expected error for from > to")
	}
	if !strings.Contains(err.Error(), "must be before") {
		t.Errorf("error = %q, want 'must be before'", err)
	}
}

func TestBackfill_FromEqualsTo(t *testing.T) {
	dir := t.TempDir()
	writePipelineWithSync(t, dir)

	var buf bytes.Buffer
	_, err := Backfill(context.Background(), BackfillOpts{
		PipelineDir:   dir,
		From:          "2026-01-15",
		To:            "2026-01-15",
		Output:        &buf,
		CommandRunner: fakeBackfillSuccess,
	})
	if err == nil {
		t.Fatal("expected error for from == to")
	}
}

func TestBackfill_NoSyncSteps(t *testing.T) {
	dir := t.TempDir()
	// Pipeline with only custom steps
	if err := os.WriteFile(filepath.Join(dir, PipelineFileName), []byte(`
apiVersion: datakit.infoblox.dev/v1alpha1
kind: PipelineWorkflow
metadata:
  name: no-sync
steps:
  - name: custom-step
    type: custom
    image: alpine:latest
`), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	_, err := Backfill(context.Background(), BackfillOpts{
		PipelineDir:   dir,
		From:          "2026-01-01",
		To:            "2026-01-31",
		Output:        &buf,
		CommandRunner: fakeBackfillSuccess,
	})
	if err == nil {
		t.Fatal("expected error for no sync steps")
	}
	if !strings.Contains(err.Error(), "no sync steps") {
		t.Errorf("error = %q, want 'no sync steps'", err)
	}
}

func TestBackfill_EnvVarInjection(t *testing.T) {
	dir := t.TempDir()
	writePipelineWithSync(t, dir)

	// Use a command runner that captures args to verify env injection
	var capturedArgs []string
	runner := func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedArgs = append(capturedArgs, args...)
		return exec.CommandContext(ctx, "echo", "ok")
	}

	var buf bytes.Buffer
	_, err := Backfill(context.Background(), BackfillOpts{
		PipelineDir:   dir,
		From:          "2026-01-01",
		To:            "2026-01-31",
		Output:        &buf,
		CommandRunner: runner,
	})
	if err != nil {
		t.Fatalf("Backfill() error = %v", err)
	}

	argsStr := strings.Join(capturedArgs, " ")
	if !strings.Contains(argsStr, "DK_BACKFILL_FROM=2026-01-01") {
		t.Errorf("expected DK_BACKFILL_FROM in args, got: %s", argsStr)
	}
	if !strings.Contains(argsStr, "DK_BACKFILL_TO=2026-01-31") {
		t.Errorf("expected DK_BACKFILL_TO in args, got: %s", argsStr)
	}
}

// Helpers

func writePipelineWithSync(t *testing.T, dir string) {
	t.Helper()
	content := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: PipelineWorkflow
metadata:
  name: backfill-pipeline
steps:
  - name: sync-data
    type: sync
    input: aws-source
    output: postgres-sink
  - name: transform-data
    type: transform
    asset: dbt-transform
`
	if err := os.WriteFile(filepath.Join(dir, PipelineFileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func fakeBackfillSuccess(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "echo", "backfill ok")
}
