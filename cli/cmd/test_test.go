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
		{"bindings", ""},
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

func TestTestCmd_SourceDetection(t *testing.T) {
	// Test that Source kind is detected from dp.yaml
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: Source
metadata:
  name: my-source
  namespace: test-team
  version: 0.1.0
spec:
  description: "Test source"
  owner: "test-team"
  runtime: cloudquery
  image: my-source:latest
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldIntegration := testIntegration
	defer func() { testIntegration = oldIntegration }()
	testIntegration = false

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	// Should detect Source kind and not fail with kind-related errors
	if err != nil {
		errMsg := err.Error()
		// Should NOT fail with "unsupported kind" or similar
		if strings.Contains(errMsg, "unsupported kind") {
			t.Errorf("should handle Source kind, got: %s", errMsg)
		}
	}
}

func TestTestCmd_ModelDetection(t *testing.T) {
	// Test that Model kind with batch mode is detected correctly
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: Model
metadata:
  name: test-model
  namespace: test
  version: 1.0.0
spec:
  description: "Test model"
  owner: "test"
  runtime: generic-go
  image: python:3.11
  mode: batch
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
	oldIntegration := testIntegration
	defer func() { testIntegration = oldIntegration }()
	testIntegration = false

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	// Pipeline type should NOT route to CloudQuery path
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "unsupported kind") {
			t.Errorf("should handle Model kind, got: %s", errMsg)
		}
	}
}

func TestTestCmd_MissingDpYaml(t *testing.T) {
	// Test that missing dp.yaml still returns proper error
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dp.yaml")
	}
	if !strings.Contains(err.Error(), "dp.yaml not found") {
		t.Errorf("expected 'dp.yaml not found' error, got: %s", err.Error())
	}
}
