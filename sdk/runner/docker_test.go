package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerRunner_EnvVarsFromPackage(t *testing.T) {
	// Skip if Docker not available
	if _, err := NewDockerRunner(); err != nil {
		t.Skip("Docker not available:", err)
	}

	tests := []struct {
		name        string
		dpYAML      string
		wantEnvVars map[string]string
	}{
		{
			name: "explicit env vars extracted from transform manifest",
			dpYAML: `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: test-env-mapper
  namespace: data-team
  version: 1.0.0
spec:
  runtime: generic-go
  image: busybox:latest
  mode: batch
  inputs:
    - asset: source-events
  outputs:
    - asset: output-lake
  env:
    - name: LOG_LEVEL
      value: debug
    - name: BATCH_SIZE
      value: "1000"
`,
			wantEnvVars: map[string]string{
				"LOG_LEVEL":  "debug",
				"BATCH_SIZE": "1000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Write dk.yaml
			if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(tt.dpYAML), 0644); err != nil {
				t.Fatalf("failed to write dk.yaml: %v", err)
			}

			// Create runner
			runner, err := NewDockerRunner()
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			dockerRunner := runner.(*DockerRunner)

			// Build env vars from the package
			envVars, err := dockerRunner.buildEnvVarsFromPackage(tmpDir)
			if err != nil {
				t.Fatalf("failed to build env vars: %v", err)
			}

			// Check expected env vars
			for key, wantValue := range tt.wantEnvVars {
				gotValue, ok := envVars[key]
				if !ok {
					t.Errorf("expected env var %s to be set, but it wasn't. Got: %v", key, envVars)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("env var %s = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestDockerRunner_RuntimeFromDP(t *testing.T) {
	// Skip if Docker not available
	if _, err := NewDockerRunner(); err != nil {
		t.Skip("Docker not available:", err)
	}

	tmpDir := t.TempDir()

	dpYAML := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: test-runtime
  namespace: data-team
  version: 1.0.0
spec:
  runtime: generic-go
  image: busybox:latest
  mode: batch
  timeout: 1h
  env:
    - name: LOG_LEVEL
      value: debug
  inputs:
    - asset: source-data
  outputs:
    - asset: output-data
`

	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dpYAML), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	runner, err := NewDockerRunner()
	if err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	// Dry run to validate runtime is read correctly
	result, err := runner.Run(context.Background(), RunOptions{
		PackageDir: tmpDir,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("failed to run: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("expected completed status, got %s", result.Status)
	}
}
