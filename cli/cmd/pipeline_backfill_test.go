package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineBackfill(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(dir string)
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "missing --from flag",
			args:      []string{"--to", "2026-01-31"},
			wantErr:   true,
			errSubstr: "required flag",
		},
		{
			name:      "missing --to flag",
			args:      []string{"--from", "2026-01-01"},
			wantErr:   true,
			errSubstr: "invalid --to date",
		},
		{
			name: "invalid date format",
			args: []string{"--from", "bad-date", "--to", "2026-01-31"},
			setup: func(dir string) {
				writePipelineFileWithSync(t, dir)
			},
			wantErr:   true,
			errSubstr: "invalid --from date",
		},
		{
			name: "from after to",
			args: []string{"--from", "2026-02-01", "--to", "2026-01-01"},
			setup: func(dir string) {
				writePipelineFileWithSync(t, dir)
			},
			wantErr:   true,
			errSubstr: "must be before",
		},
		{
			name: "no sync steps",
			args: []string{"--from", "2026-01-01", "--to", "2026-01-31"},
			setup: func(dir string) {
				content := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: PipelineWorkflow
metadata:
  name: no-sync
steps:
  - name: custom-step
    type: custom
    image: alpine:latest
`
				if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr:   true,
			errSubstr: "no sync steps",
		},
		{
			name: "no pipeline.yaml",
			args: []string{"--from", "2026-01-01", "--to", "2026-01-31"},
			// No setup — no pipeline.yaml
			wantErr:   true,
			errSubstr: "failed to load pipeline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.setup != nil {
				tt.setup(tmpDir)
			}

			origDir, _ := os.Getwd()
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origDir)

			// Reset global flags
			pipelineBackfillFrom = ""
			pipelineBackfillTo = ""
			pipelineBackfillEnv = nil

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"pipeline", "backfill"}, tt.args...))

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPipelineBackfillFlags(t *testing.T) {
	cmd := pipelineBackfillCmd

	fromFlag := cmd.Flags().Lookup("from")
	if fromFlag == nil {
		t.Fatal("--from flag not registered")
	}

	toFlag := cmd.Flags().Lookup("to")
	if toFlag == nil {
		t.Fatal("--to flag not registered")
	}

	envFlag := cmd.Flags().Lookup("env")
	if envFlag == nil {
		t.Fatal("--env flag not registered")
	}
	if envFlag.Shorthand != "e" {
		t.Errorf("env shorthand = %q, want %q", envFlag.Shorthand, "e")
	}
}

func writePipelineFileWithSync(t *testing.T, dir string) {
	t.Helper()
	content := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: PipelineWorkflow
metadata:
  name: backfill-test
steps:
  - name: sync-data
    type: sync
    source: aws-source
    sink: postgres-sink
`
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
