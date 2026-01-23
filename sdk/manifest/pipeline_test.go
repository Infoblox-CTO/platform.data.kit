package manifest

import (
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestPipelineFromBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		wantName  string
		wantImage string
	}{
		{
			name: "valid pipeline",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: test-pipeline
spec:
  image: myorg/pipeline:v1
`),
			wantErr:   false,
			wantName:  "test-pipeline",
			wantImage: "myorg/pipeline:v1",
		},
		{
			name: "pipeline with command",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: cmd-pipeline
spec:
  image: python:3.11
  command:
    - python
    - main.py
`),
			wantErr:   false,
			wantName:  "cmd-pipeline",
			wantImage: "python:3.11",
		},
		{
			name: "pipeline with args",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: args-pipeline
spec:
  image: myorg/app:latest
  args:
    - --verbose
    - --config=/etc/config.yaml
`),
			wantErr:   false,
			wantName:  "args-pipeline",
			wantImage: "myorg/app:latest",
		},
		{
			name: "wrong kind returns error",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test
`),
			wantErr: true,
		},
		{
			name:    "empty bytes returns error",
			data:    []byte(""),
			wantErr: true,
		},
		{
			name:    "invalid YAML returns error",
			data:    []byte("invalid: yaml: [broken"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline, err := PipelineFromBytes(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("PipelineFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if pipeline.Metadata.Name != tt.wantName {
					t.Errorf("name = %v, want %v", pipeline.Metadata.Name, tt.wantName)
				}
				if pipeline.Spec.Image != tt.wantImage {
					t.Errorf("image = %v, want %v", pipeline.Spec.Image, tt.wantImage)
				}
			}
		})
	}
}

func TestPipelineToBytes(t *testing.T) {
	tests := []struct {
		name     string
		pipeline *contracts.PipelineManifest
		wantErr  bool
	}{
		{
			name: "valid pipeline",
			pipeline: &contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Image: "myorg/app:v1",
				},
			},
			wantErr: false,
		},
		{
			name:     "empty pipeline",
			pipeline: &contracts.PipelineManifest{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := PipelineToBytes(tt.pipeline)
			if (err != nil) != tt.wantErr {
				t.Errorf("PipelineToBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("PipelineToBytes() returned empty data")
			}
		})
	}
}

func TestPipeline_RoundTrip(t *testing.T) {
	original := &contracts.PipelineManifest{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "Pipeline",
		Metadata: contracts.PipelineMetadata{
			Name:   "roundtrip-pipeline",
			Labels: map[string]string{"team": "analytics"},
		},
		Spec: contracts.PipelineSpec{
			Image:   "myorg/etl:v2",
			Command: []string{"python", "main.py"},
			Args:    []string{"--input=/data", "--output=/results"},
			Env: []contracts.EnvVar{
				{Name: "LOG_LEVEL", Value: "DEBUG"},
			},
			Replicas: 2,
		},
	}

	// Serialize to YAML
	data, err := PipelineToBytes(original)
	if err != nil {
		t.Fatalf("PipelineToBytes() error = %v", err)
	}

	// Parse back
	parsed, err := PipelineFromBytes(data)
	if err != nil {
		t.Fatalf("PipelineFromBytes() error = %v", err)
	}

	// Verify fields
	if parsed.Metadata.Name != original.Metadata.Name {
		t.Errorf("name = %v, want %v", parsed.Metadata.Name, original.Metadata.Name)
	}
	if parsed.Spec.Image != original.Spec.Image {
		t.Errorf("image = %v, want %v", parsed.Spec.Image, original.Spec.Image)
	}
	if len(parsed.Spec.Command) != len(original.Spec.Command) {
		t.Errorf("command count = %v, want %v", len(parsed.Spec.Command), len(original.Spec.Command))
	}
	if len(parsed.Spec.Args) != len(original.Spec.Args) {
		t.Errorf("args count = %v, want %v", len(parsed.Spec.Args), len(original.Spec.Args))
	}
	if parsed.Spec.Replicas != original.Spec.Replicas {
		t.Errorf("replicas = %v, want %v", parsed.Spec.Replicas, original.Spec.Replicas)
	}
	if parsed.Metadata.Labels["team"] != "analytics" {
		t.Errorf("label team = %v, want analytics", parsed.Metadata.Labels["team"])
	}
}

func TestPipelineFromBytes_WithEnv(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: env-pipeline
spec:
  image: myorg/app:latest
  env:
    - name: DATABASE_URL
      value: postgres://localhost/db
    - name: SECRET_KEY
      valueFrom:
        secretKeyRef:
          name: app-secrets
          key: api-key
`)

	pipeline, err := PipelineFromBytes(data)
	if err != nil {
		t.Fatalf("PipelineFromBytes() error = %v", err)
	}

	if len(pipeline.Spec.Env) != 2 {
		t.Errorf("env count = %v, want 2", len(pipeline.Spec.Env))
	}

	// Check first env var (direct value)
	if pipeline.Spec.Env[0].Name != "DATABASE_URL" {
		t.Errorf("env[0].name = %v, want DATABASE_URL", pipeline.Spec.Env[0].Name)
	}
	if pipeline.Spec.Env[0].Value != "postgres://localhost/db" {
		t.Errorf("env[0].value = %v, want postgres://localhost/db", pipeline.Spec.Env[0].Value)
	}

	// Check second env var (secret ref)
	if pipeline.Spec.Env[1].Name != "SECRET_KEY" {
		t.Errorf("env[1].name = %v, want SECRET_KEY", pipeline.Spec.Env[1].Name)
	}
	if pipeline.Spec.Env[1].ValueFrom == nil {
		t.Error("env[1].valueFrom should not be nil")
	} else if pipeline.Spec.Env[1].ValueFrom.SecretKeyRef == nil {
		t.Error("env[1].valueFrom.secretKeyRef should not be nil")
	} else {
		if pipeline.Spec.Env[1].ValueFrom.SecretKeyRef.Name != "app-secrets" {
			t.Errorf("secret name = %v, want app-secrets", pipeline.Spec.Env[1].ValueFrom.SecretKeyRef.Name)
		}
	}
}

func TestPipelineFromBytes_WithBindings(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: binding-pipeline
spec:
  image: myorg/etl:v1
  bindings:
    - name: input-data
    - name: output-data
`)

	pipeline, err := PipelineFromBytes(data)
	if err != nil {
		t.Fatalf("PipelineFromBytes() error = %v", err)
	}

	if len(pipeline.Spec.Bindings) != 2 {
		t.Errorf("bindings count = %v, want 2", len(pipeline.Spec.Bindings))
	}

	if pipeline.Spec.Bindings[0].Name != "input-data" {
		t.Errorf("binding[0].name = %v, want input-data", pipeline.Spec.Bindings[0].Name)
	}
	if pipeline.Spec.Bindings[1].Name != "output-data" {
		t.Errorf("binding[1].name = %v, want output-data", pipeline.Spec.Bindings[1].Name)
	}
}

func TestPipelineFromBytes_WithServiceAccount(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: sa-pipeline
spec:
  image: myorg/app:v1
  serviceAccountName: pipeline-runner
`)

	pipeline, err := PipelineFromBytes(data)
	if err != nil {
		t.Fatalf("PipelineFromBytes() error = %v", err)
	}

	if pipeline.Spec.ServiceAccountName != "pipeline-runner" {
		t.Errorf("serviceAccountName = %v, want pipeline-runner", pipeline.Spec.ServiceAccountName)
	}
}
