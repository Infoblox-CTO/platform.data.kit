package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineShow(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(dir string)
		wantErr   bool
		errSubstr string
		wantOut   []string // substrings expected in output
		notWant   []string // substrings NOT expected in output
	}{
		{
			name: "table output with steps",
			args: []string{},
			setup: func(dir string) {
				writePipelineShowFile(t, dir)
			},
			wantOut: []string{
				"Pipeline: demo-pipeline",
				"STEP", "TYPE", "DETAILS",
				"sync-data", "sync", "input=aws-source", "output=pg-sink",
				"transform", "asset=dbt-model",
				"custom-step", "custom", "image=alpine:latest",
			},
		},
		{
			name: "json output",
			args: []string{"--output", "json"},
			setup: func(dir string) {
				writePipelineShowFile(t, dir)
			},
			wantOut: []string{
				`"kind": "PipelineWorkflow"`,
				`"name": "demo-pipeline"`,
				`"type": "sync"`,
			},
		},
		{
			name: "yaml output",
			args: []string{"--output", "yaml"},
			setup: func(dir string) {
				writePipelineShowFile(t, dir)
			},
			wantOut: []string{
				"kind: PipelineWorkflow",
				"name: demo-pipeline",
				"type: sync",
			},
		},
		{
			name:      "no pipeline.yaml",
			args:      []string{},
			wantErr:   true,
			errSubstr: "no pipeline.yaml",
		},
		{
			name: "table with schedule",
			args: []string{},
			setup: func(dir string) {
				writePipelineShowFile(t, dir)
				schedContent := `apiVersion: data.infoblox.com/v1alpha1
kind: Schedule
cron: "0 6 * * *"
timezone: America/New_York
`
				if err := os.WriteFile(filepath.Join(dir, "schedule.yaml"), []byte(schedContent), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantOut: []string{
				"Schedule:",
				"0 6 * * *",
				"America/New_York",
				"Active",
			},
		},
		{
			name: "table without schedule",
			args: []string{},
			setup: func(dir string) {
				writePipelineShowFile(t, dir)
			},
			notWant: []string{"Schedule:"},
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

			// Reset flags
			pipelineShowOutput = ""
			pipelineShowAll = false
			pipelineShowDestination = ""
			pipelineShowScanDirs = nil

			buf := new(bytes.Buffer)
			cmd := rootCmd
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(append([]string{"pipeline", "show"}, tt.args...))

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

			output := buf.String()
			for _, want := range tt.wantOut {
				if !strings.Contains(output, want) {
					t.Errorf("output should contain %q, got:\n%s", want, output)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(output, notWant) {
					t.Errorf("output should NOT contain %q, got:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestPipelineShowFlags(t *testing.T) {
	cmd := pipelineShowCmd

	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("--output flag not registered")
	}
	if outputFlag.Shorthand != "o" {
		t.Errorf("output shorthand = %q, want %q", outputFlag.Shorthand, "o")
	}
	if outputFlag.DefValue != "" {
		t.Errorf("output default = %q, want %q", outputFlag.DefValue, "")
	}
}

func writePipelineShowFile(t *testing.T, dir string) {
	t.Helper()
	content := `apiVersion: data.infoblox.com/v1alpha1
kind: PipelineWorkflow
metadata:
  name: demo-pipeline
  description: A demo pipeline
steps:
  - name: sync-data
    type: sync
    input: aws-source
    output: pg-sink
  - name: transform
    type: transform
    asset: dbt-model
  - name: custom-step
    type: custom
    image: alpine:latest
`
	if err := os.WriteFile(filepath.Join(dir, "pipeline.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
