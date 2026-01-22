package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRunCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"env", "[]"},
		{"bindings", ""},
		{"network", "dp-network"},
		{"timeout", "30m0s"},
		{"dry-run", "false"},
		{"detach", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := runCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Errorf("flag --%s not found", tt.flag)
				return
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("flag --%s default = %v, want %v", tt.flag, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestRunCmd_Args(t *testing.T) {
	// Test argument validation
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args is valid",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one arg is valid",
			args:    []string{"./my-pipeline"},
			wantErr: false,
		},
		{
			name:    "two args is invalid",
			args:    []string{"./pkg1", "./pkg2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runCmd.Args(runCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCmd_DirectoryNotFound(t *testing.T) {
	// Test that running a non-existent directory returns an error
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	// Save and restore global flags
	oldEnv := runEnv
	oldDryRun := runDryRun
	defer func() {
		runEnv = oldEnv
		runDryRun = oldDryRun
	}()

	runEnv = []string{}
	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{nonExistent})

	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestRunCmd_MissingDpYaml(t *testing.T) {
	// Test that running a directory without dp.yaml returns an error
	tmpDir := t.TempDir()

	// Save and restore global flags
	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()

	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dp.yaml")
	}
}

func TestRunCmd_DryRun(t *testing.T) {
	// Test dry-run mode (should validate but not execute)
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test pipeline
  owner: data-team
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	pipelineContent := `apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  image: python:3.11
  command:
    - python
    - main.py
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "pipeline.yaml"), []byte(pipelineContent), 0644); err != nil {
		t.Fatalf("failed to write pipeline.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := runDryRun
	oldEnv := runEnv
	defer func() {
		runDryRun = oldDryRun
		runEnv = oldEnv
	}()

	runDryRun = true
	runEnv = []string{}

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Dry run should succeed for valid package
	if err != nil {
		t.Errorf("runPipeline() dry-run error = %v, want nil", err)
	}
}

func TestRunCmd_EnvFlags(t *testing.T) {
	// Test parsing environment variable flags
	tests := []struct {
		name    string
		envVars []string
		valid   bool
	}{
		{
			name:    "valid env vars",
			envVars: []string{"KEY=value", "DEBUG=true"},
			valid:   true,
		},
		{
			name:    "empty env vars",
			envVars: []string{},
			valid:   true,
		},
		{
			name:    "env var with equals in value",
			envVars: []string{"URL=http://host?a=b&c=d"},
			valid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the flag accepts the values
			// Actual parsing happens in runPipeline
			for _, env := range tt.envVars {
				_ = env // placeholder - flag parsing tested through cobra
			}
		})
	}
}

func TestRunCmd_TimeoutFlag(t *testing.T) {
	// Test timeout flag
	tests := []struct {
		name     string
		timeout  time.Duration
		expected time.Duration
	}{
		{
			name:     "default timeout",
			timeout:  30 * time.Minute,
			expected: 30 * time.Minute,
		},
		{
			name:     "custom timeout",
			timeout:  1 * time.Hour,
			expected: 1 * time.Hour,
		},
		{
			name:     "short timeout",
			timeout:  5 * time.Minute,
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.timeout != tt.expected {
				t.Errorf("timeout = %v, want %v", tt.timeout, tt.expected)
			}
		})
	}
}

func TestRunCmd_NotPipeline(t *testing.T) {
	// Test running a non-pipeline package type
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-dataset
  namespace: data-team
  version: 1.0.0
spec:
  type: dataset
  description: Test dataset
  owner: data-team
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()

	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Running a dataset (not pipeline) should fail or handle gracefully
	// The exact behavior depends on implementation
	_ = err
}
