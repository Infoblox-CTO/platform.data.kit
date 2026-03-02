package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestTestCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"data", ""},
		{"timeout", "5m0s"},
		{"duration", "30s"},
		{"startup-timeout", "1m0s"},
		{"integration", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := testCmd.Flags().Lookup(tt.flag)
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

func TestTestCmd_TransformDetection(t *testing.T) {
	// Test that Transform kind is detected from dk.yaml
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: my-source
spec:
  runtime: cloudquery
  image: my-source:latest
  mode: batch
  inputs:
    - asset: source-data
  outputs:
    - asset: output-data
`
	dpPath := filepath.Join(tmpDir, "dk.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Save and restore global flags
	oldIntegration := testIntegration
	defer func() { testIntegration = oldIntegration }()
	testIntegration = false

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	// Should detect Transform kind and not fail with kind-related errors
	if err != nil {
		errMsg := err.Error()
		// Should NOT fail with "unsupported kind" or similar
		if strings.Contains(errMsg, "unsupported kind") {
			t.Errorf("should handle Transform kind, got: %s", errMsg)
		}
	}
}

func TestTestCmd_TransformBatchDetection(t *testing.T) {
	// Test that Transform kind with batch mode is detected correctly
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: test-model
spec:
  runtime: generic-go
  image: python:3.11
  mode: batch
  inputs:
    - asset: source-data
  outputs:
    - asset: output-data
`
	dpPath := filepath.Join(tmpDir, "dk.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Save and restore global flags
	oldIntegration := testIntegration
	defer func() { testIntegration = oldIntegration }()
	testIntegration = false

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	// Pipeline type should NOT route to CloudQuery path
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "unsupported kind") {
			t.Errorf("should handle Transform kind, got: %s", errMsg)
		}
	}
}

func TestTestCmd_MissingDpYaml(t *testing.T) {
	// Test that missing dk.yaml still returns proper error
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dk.yaml")
	}
	if !strings.Contains(err.Error(), "dk.yaml not found") {
		t.Errorf("expected 'dk.yaml not found' error, got: %s", err.Error())
	}
}
