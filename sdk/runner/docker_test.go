package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerRunner_BindingMapping(t *testing.T) {
	// Skip if Docker not available
	if _, err := NewDockerRunner(); err != nil {
		t.Skip("Docker not available:", err)
	}

	tests := []struct {
		name         string
		dpYAML       string
		bindingsYAML string
		wantEnvVars  map[string]string
	}{
		{
			name: "kafka binding maps to env vars",
			dpYAML: `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-binding-mapper
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test binding mapping
  owner: data-team
  runtime:
    image: busybox:latest
  inputs:
    - name: events
      binding: input.events
      type: kafka-topic
  outputs:
    - name: lake
      binding: output.lake
      type: s3-prefix
      classification: {}
`,
			bindingsYAML: `apiVersion: data.infoblox.com/v1alpha1
kind: Bindings
metadata:
  name: local
  environment: local
bindings:
  - name: input.events
    type: kafka-topic
    kafka:
      topic: events-topic
      brokers:
        - localhost:9092
  - name: output.lake
    type: s3-prefix
    s3:
      bucket: test-bucket
      prefix: data/
`,
			wantEnvVars: map[string]string{
				"INPUT_EVENTS_TOPIC":   "events-topic",
				"INPUT_EVENTS_BROKERS": "localhost:9092",
				"OUTPUT_LAKE_BUCKET":   "test-bucket",
				"OUTPUT_LAKE_PREFIX":   "data/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Write dp.yaml
			if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(tt.dpYAML), 0644); err != nil {
				t.Fatalf("failed to write dp.yaml: %v", err)
			}

			// Write bindings.yaml
			if err := os.WriteFile(filepath.Join(tmpDir, "bindings.yaml"), []byte(tt.bindingsYAML), 0644); err != nil {
				t.Fatalf("failed to write bindings.yaml: %v", err)
			}

			// Create runner
			runner, err := NewDockerRunner()
			if err != nil {
				t.Fatalf("failed to create runner: %v", err)
			}

			dockerRunner := runner.(*DockerRunner)

			// Build env vars using the mapper
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
kind: DataPackage
metadata:
  name: test-runtime
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test runtime from dp.yaml
  owner: data-team
  runtime:
    image: busybox:latest
    timeout: 1h
    retries: 3
    env:
      - name: LOG_LEVEL
        value: debug
  outputs:
    - name: out
      binding: output.data
      type: s3-prefix
      classification: {}
`

	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpYAML), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
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
