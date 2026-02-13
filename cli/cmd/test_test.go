package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
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

func TestTestCmd_CloudQueryTypeDetection(t *testing.T) {
	// Test that cloudquery type is detected from dp.yaml and routes to cloudquery test path
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: my-source
  namespace: test-team
  version: 0.1.0
spec:
  type: cloudquery
  description: "Test CloudQuery plugin"
  owner: "test-team"
  cloudquery:
    role: source
    tables:
      - example_resource
    grpcPort: 7777
    concurrency: 10000
  runtime:
    image: my-source:latest
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Create a pyproject.toml so language detection picks Python
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	// Save and restore global flags
	oldIntegration := testIntegration
	defer func() { testIntegration = oldIntegration }()
	testIntegration = false

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	// Should route to cloudquery test path and fail because there are no real
	// test files or the venv bootstrap may fail (but it should NOT fail with
	// pipeline-related errors)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "pipeline") || strings.Contains(errMsg, "runner") {
			t.Errorf("cloudquery type should not route to pipeline test path, got: %s", errMsg)
		}
		// Expected: error about CloudQuery/Python/venv, not about pipeline mode
		isCloudQueryError := strings.Contains(errMsg, "CloudQuery") ||
			strings.Contains(errMsg, "unit tests") ||
			strings.Contains(errMsg, "Python") ||
			strings.Contains(errMsg, "venv")
		if !isCloudQueryError {
			t.Errorf("expected CloudQuery/Python-related error, got: %s", errMsg)
		}
	}
}

func TestTestCmd_CloudQueryLanguageDetectionPython(t *testing.T) {
	// Test Python language detection
	tmpDir := t.TempDir()

	// Create Python indicator files
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	lang := detectCloudQueryLanguage(tmpDir)
	if lang != "python" {
		t.Errorf("detectCloudQueryLanguage() = %q, want %q", lang, "python")
	}
}

func TestTestCmd_CloudQueryLanguageDetectionGo(t *testing.T) {
	// Test Go language detection
	tmpDir := t.TempDir()

	// Create Go indicator file
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	lang := detectCloudQueryLanguage(tmpDir)
	if lang != "go" {
		t.Errorf("detectCloudQueryLanguage() = %q, want %q", lang, "go")
	}
}

func TestTestCmd_CloudQueryLanguageDetectionDefault(t *testing.T) {
	// Test default language detection (no indicators)
	tmpDir := t.TempDir()

	lang := detectCloudQueryLanguage(tmpDir)
	if lang != "python" {
		t.Errorf("detectCloudQueryLanguage() = %q, want %q (default)", lang, "python")
	}
}

func TestTestCmd_CloudQueryLanguageDetectionRequirementsTxt(t *testing.T) {
	// Test Python detection via requirements.txt
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("cloudquery-plugin-sdk>=0.1.52\n"), 0644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}

	lang := detectCloudQueryLanguage(tmpDir)
	if lang != "python" {
		t.Errorf("detectCloudQueryLanguage() = %q, want %q", lang, "python")
	}
}

func TestTestCmd_CloudQueryLanguageDetectionMainPy(t *testing.T) {
	// Test Python detection via main.py
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte("print('hello')\n"), 0644); err != nil {
		t.Fatalf("failed to write main.py: %v", err)
	}

	lang := detectCloudQueryLanguage(tmpDir)
	if lang != "python" {
		t.Errorf("detectCloudQueryLanguage() = %q, want %q", lang, "python")
	}
}

func TestTestCmd_CloudQueryLanguageGoOverPython(t *testing.T) {
	// When both go.mod and pyproject.toml exist, Go should win
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte("[project]\nname = \"test\"\n"), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	lang := detectCloudQueryLanguage(tmpDir)
	if lang != "go" {
		t.Errorf("detectCloudQueryLanguage() = %q, want %q (Go should take priority)", lang, "go")
	}
}

func TestTestCmd_CloudQueryIntegrationFlag(t *testing.T) {
	// Test that --integration flag routes to integration test path
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: my-source
  namespace: test-team
  version: 0.1.0
spec:
  type: cloudquery
  description: "Test CloudQuery plugin"
  owner: "test-team"
  cloudquery:
    role: source
    tables:
      - example_resource
    grpcPort: 7777
    concurrency: 10000
  runtime:
    image: my-source:latest
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldIntegration := testIntegration
	defer func() { testIntegration = oldIntegration }()
	testIntegration = true

	cmd := &cobra.Command{}
	err := runTest(cmd, []string{tmpDir})

	// Should route to integration test path and fail (because cloudquery CLI isn't in test env)
	if err == nil {
		t.Error("expected error from integration test (cloudquery CLI not available in test env)")
	} else {
		errMsg := err.Error()
		// Should fail with cloudquery-related error, not pipeline error
		if strings.Contains(errMsg, "pipeline") || strings.Contains(errMsg, "runner") {
			t.Errorf("--integration should route to CloudQuery path, got: %s", errMsg)
		}
	}
}

func TestTestCmd_PipelineTypeNotAffected(t *testing.T) {
	// Test that pipeline type still goes through the pipeline test path
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: test
  version: 1.0.0
spec:
  type: pipeline
  description: "Test pipeline"
  owner: "test"
  runtime:
    image: python:3.11
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
		if strings.Contains(errMsg, "CloudQuery") || strings.Contains(errMsg, "cloudquery") {
			t.Errorf("pipeline type should not route to CloudQuery test path, got: %s", errMsg)
		}
	}
}

func TestTestCmd_CloudQueryMissingDpYaml(t *testing.T) {
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

func TestFindPython3(t *testing.T) {
	// findPython3 should locate a python3 binary on the system
	p, err := findPython3()
	if err != nil {
		t.Skip("python3 not available on this system")
	}
	if p == "" {
		t.Error("findPython3() returned empty path")
	}
	// Verify it's executable
	cmd := exec.Command(p, "--version")
	if err := cmd.Run(); err != nil {
		t.Errorf("python3 at %s is not executable: %v", p, err)
	}
}

func TestEnsurePythonVenv_CreatesVenv(t *testing.T) {
	// ensurePythonVenv should create a .venv directory with pytest
	if _, err := findPython3(); err != nil {
		t.Skip("python3 not available on this system")
	}

	tmpDir := t.TempDir()

	// Create a minimal pyproject.toml with pytest as dev dep
	pyproject := `[build-system]
requires = ["setuptools>=69.0"]
build-backend = "setuptools.build_meta"

[project]
name = "test-plugin"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = []

[project.optional-dependencies]
dev = ["pytest>=8.0"]

[tool.pytest.ini_options]
pythonpath = ["."]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644); err != nil {
		t.Fatalf("failed to write pyproject.toml: %v", err)
	}

	venvDir, err := ensurePythonVenv(tmpDir)
	if err != nil {
		t.Fatalf("ensurePythonVenv() error: %v", err)
	}

	// Verify .venv was created
	if _, err := os.Stat(venvDir); os.IsNotExist(err) {
		t.Error(".venv directory was not created")
	}

	// Verify pytest is installed in venv
	pytestBin := filepath.Join(venvDir, "bin", "pytest")
	if _, err := os.Stat(pytestBin); os.IsNotExist(err) {
		t.Error("pytest not found in .venv/bin/")
	}
}

func TestEnsurePythonVenv_SkipsIfExists(t *testing.T) {
	// ensurePythonVenv should skip creation if .venv/bin/pytest already exists
	if _, err := findPython3(); err != nil {
		t.Skip("python3 not available on this system")
	}

	tmpDir := t.TempDir()

	// Create a fake venv with pytest binary
	venvBinDir := filepath.Join(tmpDir, ".venv", "bin")
	if err := os.MkdirAll(venvBinDir, 0755); err != nil {
		t.Fatalf("failed to create .venv/bin: %v", err)
	}
	pytestBin := filepath.Join(venvBinDir, "pytest")
	if err := os.WriteFile(pytestBin, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
		t.Fatalf("failed to create fake pytest: %v", err)
	}

	// ensurePythonVenv should detect the existing pytest and return immediately
	venvDir, err := ensurePythonVenv(tmpDir)
	if err != nil {
		t.Fatalf("ensurePythonVenv() error: %v", err)
	}

	expected := filepath.Join(tmpDir, ".venv")
	if venvDir != expected {
		t.Errorf("ensurePythonVenv() = %q, want %q", venvDir, expected)
	}

	// Verify the fake pytest was NOT overwritten (no pip install ran)
	content, _ := os.ReadFile(pytestBin)
	if !strings.Contains(string(content), "fake") {
		t.Error("ensurePythonVenv should have skipped creation for existing venv")
	}
}

func TestTestCmd_CloudQuerySourceConfigForIntegration(t *testing.T) {
	// Verify source config generation (reused from run path) works for test integration
	dp := &contracts.DataPackage{
		Metadata: contracts.PackageMetadata{
			Name:      "test-source",
			Namespace: "test",
		},
		Spec: contracts.DataPackageSpec{
			Type: contracts.PackageTypeCloudQuery,
			CloudQuery: &contracts.CloudQuerySpec{
				Role:   "source",
				Tables: []string{"users", "orders"},
			},
		},
	}

	configPath, err := generateSourceConfig(dp, 7777)
	if err != nil {
		t.Fatalf("generateSourceConfig() error: %v", err)
	}
	defer os.Remove(configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read source config: %v", err)
	}

	content := string(data)
	// Verify source config contains expected content (no destination)
	if !strings.Contains(content, `name: "test-source"`) {
		t.Error("source config should contain plugin name")
	}
	if !strings.Contains(content, "localhost:7777") {
		t.Error("source config should contain gRPC address")
	}
	if !strings.Contains(content, `"users"`) {
		t.Error("source config should contain tables")
	}
	if !strings.Contains(content, `"orders"`) {
		t.Error("source config should contain tables")
	}
	if strings.Contains(content, "destination") {
		t.Error("source config should not contain any destination")
	}
}
