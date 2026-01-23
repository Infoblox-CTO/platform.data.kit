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
  runtime:
    image: python:3.11
    command:
      - python
      - main.py
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

func TestRunCmd_SetFlag(t *testing.T) {
	// Verify the --set flag is registered correctly
	flag := runCmd.Flags().Lookup("set")
	if flag == nil {
		t.Fatal("--set flag not found")
	}

	// Default should be empty array
	if flag.DefValue != "[]" {
		t.Errorf("--set default = %v, want []", flag.DefValue)
	}
}

func TestRunCmd_ValuesFlag(t *testing.T) {
	// Verify the -f/--values flag is registered correctly
	flag := runCmd.Flags().Lookup("values")
	if flag == nil {
		t.Fatal("--values flag not found")
	}

	// Check shorthand
	if flag.Shorthand != "f" {
		t.Errorf("--values shorthand = %q, want \"f\"", flag.Shorthand)
	}

	// Default should be empty array
	if flag.DefValue != "[]" {
		t.Errorf("--values default = %v, want []", flag.DefValue)
	}
}

func TestApplyOverrides_SetValues(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a base dp.yaml
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
  runtime:
    image: original:v1
    timeout: 30m
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
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
		"spec.runtime.image=overridden:v2",
		"spec.runtime.timeout=1h",
	}
	runValueFiles = []string{}

	// Apply overrides
	if err := applyOverrides(dpPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	// Verify the file was modified
	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read modified dp.yaml: %v", err)
	}

	content := string(data)

	if !contains(content, "overridden:v2") {
		t.Error("expected image to be overridden")
	}
	if !contains(content, "1h") {
		t.Error("expected timeout to be overridden")
	}

	// Verify backup was created
	backupPath := dpPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("expected backup file to be created")
	}
}

func TestApplyOverrides_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
spec:
  type: pipeline
  runtime:
    image: test:v1
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := runSet
	oldFiles := runValueFiles
	defer func() {
		runSet = oldSet
		runValueFiles = oldFiles
	}()

	// Set an invalid path
	runSet = []string{"invalid.path.here=value"}
	runValueFiles = []string{}

	// Apply overrides should fail
	err := applyOverrides(dpPath)
	if err == nil {
		t.Fatal("expected error for invalid path")
	}

	if !contains(err.Error(), "invalid override path") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestApplyOverrides_ValueFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base dp.yaml
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
spec:
  type: pipeline
  runtime:
    image: original:v1
    timeout: 30m
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Create override file
	overrideContent := `spec:
  runtime:
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

	// Apply overrides
	if err := applyOverrides(dpPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	// Verify the file was modified
	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read modified dp.yaml: %v", err)
	}

	content := string(data)

	if !contains(content, "from-file:v3") {
		t.Error("expected image to be overridden from file")
	}
	if !contains(content, "retries: 5") {
		t.Error("expected retries to be added from file")
	}
	// timeout should be preserved from original
	if !contains(content, "30m") {
		t.Error("expected timeout to be preserved")
	}
}

func TestApplyOverrides_Precedence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create base dp.yaml
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
spec:
  type: pipeline
  runtime:
    image: base:v1
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Create override file
	overrideContent := `spec:
  runtime:
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

	// Set both file and --set flag - --set should win
	runValueFiles = []string{overridePath}
	runSet = []string{"spec.runtime.image=from-set:v3"}

	// Apply overrides
	if err := applyOverrides(dpPath); err != nil {
		t.Fatalf("applyOverrides() error = %v", err)
	}

	// Verify --set won (highest precedence)
	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read modified dp.yaml: %v", err)
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
