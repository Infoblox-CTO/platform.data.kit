package contracts

import (
	"testing"
)

func TestPipelineManifest_Fields(t *testing.T) {
	tests := []struct {
		name     string
		pipeline PipelineManifest
		wantAPI  string
		wantKind string
		wantName string
	}{
		{
			name: "full pipeline",
			pipeline: PipelineManifest{
				APIVersion: string(APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: PipelineSpec{
					Image:   "myorg/pipeline:v1",
					Command: []string{"python", "main.py"},
				},
			},
			wantAPI:  string(APIVersionV1Alpha1),
			wantKind: "Pipeline",
			wantName: "test-pipeline",
		},
		{
			name:     "empty pipeline",
			pipeline: PipelineManifest{},
			wantAPI:  "",
			wantKind: "",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pipeline.APIVersion != tt.wantAPI {
				t.Errorf("APIVersion = %v, want %v", tt.pipeline.APIVersion, tt.wantAPI)
			}
			if tt.pipeline.Kind != tt.wantKind {
				t.Errorf("Kind = %v, want %v", tt.pipeline.Kind, tt.wantKind)
			}
			if tt.pipeline.Metadata.Name != tt.wantName {
				t.Errorf("Metadata.Name = %v, want %v", tt.pipeline.Metadata.Name, tt.wantName)
			}
		})
	}
}

func TestPipelineSpec_Image(t *testing.T) {
	tests := []struct {
		name      string
		spec      PipelineSpec
		wantImage string
	}{
		{
			name:      "with image",
			spec:      PipelineSpec{Image: "myorg/pipeline:latest"},
			wantImage: "myorg/pipeline:latest",
		},
		{
			name:      "empty image",
			spec:      PipelineSpec{},
			wantImage: "",
		},
		{
			name:      "with tag",
			spec:      PipelineSpec{Image: "docker.io/library/python:3.11"},
			wantImage: "docker.io/library/python:3.11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.spec.Image != tt.wantImage {
				t.Errorf("Image = %v, want %v", tt.spec.Image, tt.wantImage)
			}
		})
	}
}

func TestPipelineSpec_CommandAndArgs(t *testing.T) {
	tests := []struct {
		name     string
		spec     PipelineSpec
		wantCmd  []string
		wantArgs []string
	}{
		{
			name: "with command and args",
			spec: PipelineSpec{
				Command: []string{"python", "main.py"},
				Args:    []string{"--verbose", "--config=/etc/config.yaml"},
			},
			wantCmd:  []string{"python", "main.py"},
			wantArgs: []string{"--verbose", "--config=/etc/config.yaml"},
		},
		{
			name:     "empty command and args",
			spec:     PipelineSpec{},
			wantCmd:  nil,
			wantArgs: nil,
		},
		{
			name: "command only",
			spec: PipelineSpec{
				Command: []string{"/bin/sh", "-c", "echo hello"},
			},
			wantCmd:  []string{"/bin/sh", "-c", "echo hello"},
			wantArgs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.spec.Command) != len(tt.wantCmd) {
				t.Errorf("Command length = %v, want %v", len(tt.spec.Command), len(tt.wantCmd))
			}
			for i := range tt.spec.Command {
				if tt.spec.Command[i] != tt.wantCmd[i] {
					t.Errorf("Command[%d] = %v, want %v", i, tt.spec.Command[i], tt.wantCmd[i])
				}
			}
			if len(tt.spec.Args) != len(tt.wantArgs) {
				t.Errorf("Args length = %v, want %v", len(tt.spec.Args), len(tt.wantArgs))
			}
		})
	}
}

func TestEnvVar_Types(t *testing.T) {
	tests := []struct {
		name    string
		env     EnvVar
		wantKey string
		wantVal string
	}{
		{
			name:    "simple value",
			env:     EnvVar{Name: "DATABASE_URL", Value: "postgres://localhost/db"},
			wantKey: "DATABASE_URL",
			wantVal: "postgres://localhost/db",
		},
		{
			name:    "empty value",
			env:     EnvVar{Name: "EMPTY_VAR", Value: ""},
			wantKey: "EMPTY_VAR",
			wantVal: "",
		},
		{
			name:    "with secret ref",
			env:     EnvVar{Name: "SECRET_KEY", ValueFrom: &EnvVarSource{SecretKeyRef: &SecretKeySelector{Name: "my-secret", Key: "api-key"}}},
			wantKey: "SECRET_KEY",
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env.Name != tt.wantKey {
				t.Errorf("Name = %v, want %v", tt.env.Name, tt.wantKey)
			}
			if tt.env.Value != tt.wantVal {
				t.Errorf("Value = %v, want %v", tt.env.Value, tt.wantVal)
			}
		})
	}
}

func TestBindingRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      BindingRef
		wantName string
	}{
		{
			name:     "with name",
			ref:      BindingRef{Name: "data-bucket"},
			wantName: "data-bucket",
		},
		{
			name:     "empty ref",
			ref:      BindingRef{},
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ref.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", tt.ref.Name, tt.wantName)
			}
		})
	}
}

func TestPipelineMetadata_Labels(t *testing.T) {
	tests := []struct {
		name       string
		metadata   PipelineMetadata
		wantLabels map[string]string
	}{
		{
			name: "with labels",
			metadata: PipelineMetadata{
				Name:   "test-pipeline",
				Labels: map[string]string{"env": "prod", "team": "data"},
			},
			wantLabels: map[string]string{"env": "prod", "team": "data"},
		},
		{
			name:       "no labels",
			metadata:   PipelineMetadata{Name: "test"},
			wantLabels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.metadata.Labels) != len(tt.wantLabels) {
				t.Errorf("Labels length = %v, want %v", len(tt.metadata.Labels), len(tt.wantLabels))
			}
			for k, v := range tt.wantLabels {
				if tt.metadata.Labels[k] != v {
					t.Errorf("Labels[%s] = %v, want %v", k, tt.metadata.Labels[k], v)
				}
			}
		})
	}
}

func TestPipelineSpec_Bindings(t *testing.T) {
	tests := []struct {
		name         string
		spec         PipelineSpec
		wantBindings int
	}{
		{
			name: "with bindings",
			spec: PipelineSpec{
				Image: "myorg/pipeline:latest",
				Bindings: []BindingRef{
					{Name: "input-bucket"},
					{Name: "output-bucket"},
				},
			},
			wantBindings: 2,
		},
		{
			name:         "no bindings",
			spec:         PipelineSpec{Image: "myorg/pipeline:latest"},
			wantBindings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.spec.Bindings) != tt.wantBindings {
				t.Errorf("Bindings count = %v, want %v", len(tt.spec.Bindings), tt.wantBindings)
			}
		})
	}
}
