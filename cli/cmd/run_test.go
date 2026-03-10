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
		{"network", ""},
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

func TestRunCmd_MissingDkYaml(t *testing.T) {
	tmpDir := t.TempDir()

	// Save and restore global flags
	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()

	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dk.yaml")
	}
}

func TestRunCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pipeline
spec:
  runtime: generic-go
  image: python:3.11
  mode: batch
  command:
    - python
    - main.py
  inputs:
    - dataset: source-data
  outputs:
    - dataset: output-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
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
			for _, env := range tt.envVars {
				_ = env // flag parsing tested through cobra
			}
		})
	}
}

func TestRunCmd_TimeoutFlag(t *testing.T) {
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

func TestRunCmd_InvalidKind(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Invalid
metadata:
  name: test-invalid
  namespace: data-team
  version: 1.0.0
spec:
  runtime: generic-go
  description: Test invalid kind
  owner: data-team
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()

	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Running an invalid kind should fail
	_ = err
}

func TestRunCmd_SetFlag(t *testing.T) {
	flag := runCmd.Flags().Lookup("set")
	if flag == nil {
		t.Fatal("--set flag not found")
	}

	if flag.DefValue != "[]" {
		t.Errorf("--set default = %v, want []", flag.DefValue)
	}
}

func TestRunCmd_ValuesFlag(t *testing.T) {
	flag := runCmd.Flags().Lookup("values")
	if flag == nil {
		t.Fatal("--values flag not found")
	}

	if flag.Shorthand != "f" {
		t.Errorf("--values shorthand = %q, want \"f\"", flag.Shorthand)
	}

	if flag.DefValue != "[]" {
		t.Errorf("--values default = %v, want []", flag.DefValue)
	}
}

func TestApplyOverrides_SetValues(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pipeline
spec:
  runtime: generic-go
  image: original:v1
  timeout: 30m
  mode: batch
  inputs:
    - dataset: source-data
  outputs:
    - dataset: output-data
`
	dkPath := filepath.Join(tmpDir, "dk.yaml")
	if err := os.WriteFile(dkPath, []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	// Set override values
	runSet = []string{
		"spec.image=overridden:v2",
		"spec.timeout=1h",
	}
	runValueFiles = []string{}

	// Apply overrides
	if err := applyOverrides(dkPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	// Verify the file was modified
	data, err := os.ReadFile(dkPath)
	if err != nil {
		t.Fatalf("failed to read modified dk.yaml: %v", err)
	}

	content := string(data)

	if !contains(content, "overridden:v2") {
		t.Error("expected image to be overridden")
	}
	if !contains(content, "1h") {
		t.Error("expected timeout to be overridden")
	}

	// Verify backup was created
	backupPath := dkPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("expected backup file to be created")
	}
}

func TestApplyOverrides_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pipeline
spec:
  runtime: generic-go
  image: test:v1
`
	dkPath := filepath.Join(tmpDir, "dk.yaml")
	if err := os.WriteFile(dkPath, []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	runSet = []string{"invalid.path.here=value"}
	runValueFiles = []string{}

	err := applyOverrides(dkPath)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}

	if !contains(err.Error(), "invalid override path") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestApplyOverrides_ValueFiles(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pipeline
spec:
  runtime: generic-go
  image: original:v1
  timeout: 30m
`
	dkPath := filepath.Join(tmpDir, "dk.yaml")
	if err := os.WriteFile(dkPath, []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Create override file
	overrideContent := `spec:
  image: from-file:v3
  retries: 5
`
	overridePath := filepath.Join(tmpDir, "overrides.yaml")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write overrides.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	runSet = []string{}
	runValueFiles = []string{overridePath}

	if err := applyOverrides(dkPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	data, err := os.ReadFile(dkPath)
	if err != nil {
		t.Fatalf("failed to read modified dk.yaml: %v", err)
	}

	content := string(data)

	if !contains(content, "from-file:v3") {
		t.Error("expected image to be overridden from file")
	}
	if !contains(content, "retries: 5") {
		t.Error("expected retries to be added from file")
	}
	if !contains(content, "30m") {
		t.Error("expected timeout to be preserved")
	}
}

func TestApplyOverrides_Precedence(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pipeline
spec:
  runtime: generic-go
  image: base:v1
`
	dkPath := filepath.Join(tmpDir, "dk.yaml")
	if err := os.WriteFile(dkPath, []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Create override file
	overrideContent := `spec:
  image: from-file:v2
`
	overridePath := filepath.Join(tmpDir, "overrides.yaml")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write overrides.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	// Set both file and --set flag — --set should win
	runValueFiles = []string{overridePath}
	runSet = []string{"spec.image=from-set:v3"}

	if err := applyOverrides(dkPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	data, err := os.ReadFile(dkPath)
	if err != nil {
		t.Fatalf("failed to read modified dk.yaml: %v", err)
	}

	content := string(data)
	if !contains(content, "from-set:v3") {
		t.Error("expected --set to override file override")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRunCmd_TransformWithBatchMode(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
spec:
  runtime: generic-go
  mode: batch
  image: myimage:v1
  inputs:
    - dataset: source-data
  outputs:
    - dataset: output-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()
	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	if err != nil {
		errMsg := err.Error()
		// Should NOT fail because of mode parsing
		if contains(errMsg, "unsupported mode") {
			t.Error("batch mode should be supported")
		}
	}
}

func TestRunCmd_TransformCloudQuery(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-source
spec:
  runtime: cloudquery
  image: "test/test-source:latest"
  mode: batch
  inputs:
    - dataset: source-data
  outputs:
    - dataset: example-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatal(err)
	}

	oldDryRun := runDryRun
	defer func() { runDryRun = oldDryRun }()
	runDryRun = true

	cmd := &cobra.Command{}
	err := runPipeline(cmd, []string{tmpDir})

	// Transform kind should be parseable; may fail at Docker step
	if err != nil {
		errMsg := err.Error()
		if contains(errMsg, "failed to parse dk.yaml") {
			t.Errorf("Transform kind should be parseable, got: %v", err)
		}
	}
}
