package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineRun(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(dir string)
		wantErr   bool
		errSubstr string
		wantOut   string
	}{
		{
			name: "no pipeline.yaml",
			args: []string{},
			// No setup — no pipeline.yaml in tmpDir
			wantErr:   true,
			errSubstr: "failed to load pipeline",
		},
		{
			name: "invalid env format",
			args: []string{"--env", "BADFORMAT"},
			setup: func(dir string) {
				writePipelineFile(t, dir)
			},
			wantErr:   true,
			errSubstr: "invalid env var format",
		},
		{
			name: "step not found",
			args: []string{"--step", "nonexistent"},
			setup: func(dir string) {
				writePipelineFile(t, dir)
			},
			wantErr:   true,
			errSubstr: "not found",
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
			pipelineRunEnv = nil
			pipelineRunStep = ""

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"pipeline", "run"}, tt.args...))

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

			if tt.wantOut != "" {
				output := buf.String()
				if !strings.Contains(output, tt.wantOut) {
					t.Errorf("output %q should contain %q", output, tt.wantOut)
				}
			}
		})
	}
}

func TestPipelineRunFlags(t *testing.T) {
	cmd := pipelineRunCmd

	envFlag := cmd.Flags().Lookup("env")
	if envFlag == nil {
		t.Fatal("--env flag not registered")
	}
	if envFlag.Shorthand != "e" {
		t.Errorf("env shorthand = %q, want %q", envFlag.Shorthand, "e")
	}

	stepFlag := cmd.Flags().Lookup("step")
	if stepFlag == nil {
		t.Fatal("--step flag not registered")
	}
}

func writePipelineFile(t *testing.T, dir string) {
	t.Helper()
	content := `apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: test-pipeline
steps:
  - name: custom-step
    type: custom
    image: alpine:latest
`
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
