package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestShowCmd_Registered(t *testing.T) {
	// Verify the show command is registered
	cmd := rootCmd.Commands()
	found := false
	for _, c := range cmd {
		if c.Name() == "show" {
			found = true
			break
		}
	}
	if !found {
		t.Error("show command not registered in root")
	}
}

func TestShowCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag      string
		shorthand string
		defValue  string
	}{
		{"set", "", "[]"},
		{"values", "f", "[]"},
		{"output", "o", "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := showCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Errorf("flag --%s not found", tt.flag)
				return
			}
			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("flag --%s shorthand = %q, want %q", tt.flag, flag.Shorthand, tt.shorthand)
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("flag --%s default = %v, want %v", tt.flag, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestShowCmd_OutputYAML(t *testing.T) {
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
	oldSet := showSet
	oldFiles := showValueFiles
	oldOutput := showOutputFormat
	defer func() {
		showSet = oldSet
		showValueFiles = oldFiles
		showOutputFormat = oldOutput
	}()

	showSet = []string{}
	showValueFiles = []string{}
	showOutputFormat = "yaml"

	// Capture output
	var buf bytes.Buffer
	output, err := showManifest(tmpDir, &buf)
	if err != nil {
		t.Fatalf("showManifest() error = %v", err)
	}

	// Verify YAML output
	if output == "" {
		t.Error("expected non-empty output")
	}

	if !containsStr(output, "name: test-pipeline") {
		t.Error("expected output to contain name")
	}
	if !containsStr(output, "image: test:v1") {
		t.Error("expected output to contain image")
	}
}

func TestShowCmd_OutputJSON(t *testing.T) {
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
	oldSet := showSet
	oldFiles := showValueFiles
	oldOutput := showOutputFormat
	defer func() {
		showSet = oldSet
		showValueFiles = oldFiles
		showOutputFormat = oldOutput
	}()

	showSet = []string{}
	showValueFiles = []string{}
	showOutputFormat = "json"

	// Capture output
	var buf bytes.Buffer
	output, err := showManifest(tmpDir, &buf)
	if err != nil {
		t.Fatalf("showManifest() error = %v", err)
	}

	// Verify JSON output (should have braces)
	if !containsStr(output, "{") || !containsStr(output, "}") {
		t.Error("expected JSON output with braces")
	}
	if !containsStr(output, "\"name\":") {
		t.Error("expected JSON output to contain quoted name field")
	}
}

func TestShowCmd_WithOverrides(t *testing.T) {
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
  image: from-file:v2
`
	overridePath := filepath.Join(tmpDir, "overrides.yaml")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write overrides.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := showSet
	oldFiles := showValueFiles
	oldOutput := showOutputFormat
	defer func() {
		showSet = oldSet
		showValueFiles = oldFiles
		showOutputFormat = oldOutput
	}()

	// Apply file override and --set override
	showValueFiles = []string{overridePath}
	showSet = []string{"spec.image=from-set:v3"}
	showOutputFormat = "yaml"

	// Capture output
	var buf bytes.Buffer
	output, err := showManifest(tmpDir, &buf)
	if err != nil {
		t.Fatalf("showManifest() error = %v", err)
	}

	// Verify --set takes precedence
	if !containsStr(output, "from-set:v3") {
		t.Error("expected --set override to be applied")
	}

	// Original timeout should be preserved
	if !containsStr(output, "30m") {
		t.Error("expected timeout to be preserved from original")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Schedule Display Test (T044) ---

func TestShowCmd_DisplaysSchedule(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a minimal dk.yaml
	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
spec:
  runtime: generic-go
  image: myimage:v1
  mode: batch
  inputs:
    - asset: source-data
  outputs:
    - asset: output-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a schedule.yaml
	schedContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Schedule
cron: "0 6 * * *"
timezone: America/Chicago
`
	if err := os.WriteFile(filepath.Join(tmpDir, "schedule.yaml"), []byte(schedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags
	showSet = nil
	showValueFiles = nil
	showOutputFormat = "yaml"

	var buf bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"show", tmpDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("show command error = %v", err)
	}

	output := buf.String()
	if !containsStr(output, "--- Schedule ---") {
		t.Errorf("output should contain schedule section, got:\n%s", output)
	}
	if !containsStr(output, "0 6 * * *") {
		t.Errorf("output should contain cron expression, got:\n%s", output)
	}
	if !containsStr(output, "America/Chicago") {
		t.Errorf("output should contain timezone, got:\n%s", output)
	}
	if !containsStr(output, "Active") {
		t.Errorf("output should contain Active status, got:\n%s", output)
	}
}

func TestShowCmd_NoSchedule(t *testing.T) {
	tmpDir := t.TempDir()

	// Write dk.yaml only, no schedule.yaml
	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
spec:
  runtime: generic-go
  image: myimage:v1
  mode: batch
  inputs:
    - asset: source-data
  outputs:
    - asset: output-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatal(err)
	}

	showSet = nil
	showValueFiles = nil
	showOutputFormat = "yaml"

	var buf bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"show", tmpDir})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("show command error = %v", err)
	}

	output := buf.String()
	if containsStr(output, "--- Schedule ---") {
		t.Errorf("output should NOT contain schedule section when no schedule.yaml exists, got:\n%s", output)
	}
}
