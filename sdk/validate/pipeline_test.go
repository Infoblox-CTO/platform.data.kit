package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestPipelineValidator_Mode(t *testing.T) {
	tests := []struct {
		name     string
		pipeline contracts.PipelineManifest
		wantErrs int
	}{
		{
			name: "valid batch mode",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:    contracts.PipelineModeBatch,
					Image:   "myorg/pipeline:v1",
					Timeout: "30m",
				},
			},
			wantErrs: 0,
		},
		{
			name: "valid streaming mode",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:     contracts.PipelineModeStreaming,
					Image:    "myorg/pipeline:v1",
					Replicas: 3,
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid mode",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  "invalid",
					Image: "myorg/pipeline:v1",
				},
			},
			wantErrs: 1,
		},
		{
			name: "batch without timeout - warning",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeBatch,
					Image: "myorg/pipeline:v1",
					// No timeout - should generate warning, not error
				},
			},
			wantErrs: 0, // Warning, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPipelineValidator(&tt.pipeline, "pipeline.yaml")
			errs := v.Validate(context.Background())
			errorCount := len(errs.Errors())
			if errorCount != tt.wantErrs {
				t.Errorf("Validate() errors = %d, want %d. Errors: %v", errorCount, tt.wantErrs, errs)
			}
		})
	}
}

func TestPipelineValidator_StreamingProbes(t *testing.T) {
	tests := []struct {
		name     string
		pipeline contracts.PipelineManifest
		wantErrs int
	}{
		{
			name: "valid streaming with probes",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeStreaming,
					Image: "myorg/pipeline:v1",
					LivenessProbe: &contracts.Probe{
						HTTPGet: &contracts.HTTPGetAction{
							Path: "/healthz",
							Port: 8080,
						},
						PeriodSeconds: 10,
					},
					ReadinessProbe: &contracts.Probe{
						HTTPGet: &contracts.HTTPGetAction{
							Path: "/ready",
							Port: 8080,
						},
						PeriodSeconds: 5,
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "streaming with invalid probe port",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeStreaming,
					Image: "myorg/pipeline:v1",
					LivenessProbe: &contracts.Probe{
						HTTPGet: &contracts.HTTPGetAction{
							Path: "/healthz",
							Port: 0, // Invalid
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "streaming with exec probe",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeStreaming,
					Image: "myorg/pipeline:v1",
					LivenessProbe: &contracts.Probe{
						Exec: &contracts.ExecAction{
							Command: []string{"/bin/sh", "-c", "exit 0"},
						},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "streaming with empty exec command",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeStreaming,
					Image: "myorg/pipeline:v1",
					LivenessProbe: &contracts.Probe{
						Exec: &contracts.ExecAction{
							Command: []string{}, // Invalid
						},
					},
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPipelineValidator(&tt.pipeline, "pipeline.yaml")
			errs := v.Validate(context.Background())
			errorCount := len(errs.Errors())
			if errorCount != tt.wantErrs {
				t.Errorf("Validate() errors = %d, want %d. Errors: %v", errorCount, tt.wantErrs, errs)
			}
		})
	}
}

func TestPipelineValidator_Lineage(t *testing.T) {
	tests := []struct {
		name     string
		pipeline contracts.PipelineManifest
		wantErrs int
	}{
		{
			name: "valid lineage config",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeStreaming,
					Image: "myorg/pipeline:v1",
					Lineage: &contracts.PipelineLineage{
						Enabled:           true,
						HeartbeatInterval: "30s",
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid heartbeat interval",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeStreaming,
					Image: "myorg/pipeline:v1",
					Lineage: &contracts.PipelineLineage{
						Enabled:           true,
						HeartbeatInterval: "invalid",
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "heartbeat with batch mode - warning",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:    contracts.PipelineModeBatch,
					Image:   "myorg/pipeline:v1",
					Timeout: "30m",
					Lineage: &contracts.PipelineLineage{
						Enabled:           true,
						HeartbeatInterval: "30s",
					},
				},
			},
			wantErrs: 0, // Warning, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPipelineValidator(&tt.pipeline, "pipeline.yaml")
			errs := v.Validate(context.Background())
			errorCount := len(errs.Errors())
			if errorCount != tt.wantErrs {
				t.Errorf("Validate() errors = %d, want %d. Errors: %v", errorCount, tt.wantErrs, errs)
			}
		})
	}
}

func TestPipelineValidator_Warnings(t *testing.T) {
	tests := []struct {
		name         string
		pipeline     contracts.PipelineManifest
		wantWarnings int
	}{
		{
			name: "batch without timeout",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeBatch,
					Image: "myorg/pipeline:v1",
				},
			},
			wantWarnings: 1,
		},
		{
			name: "streaming without probes",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:  contracts.PipelineModeStreaming,
					Image: "myorg/pipeline:v1",
				},
			},
			wantWarnings: 1, // No probes warning
		},
		{
			name: "heartbeat on batch mode",
			pipeline: contracts.PipelineManifest{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "Pipeline",
				Metadata: contracts.PipelineMetadata{
					Name: "test-pipeline",
				},
				Spec: contracts.PipelineSpec{
					Mode:    contracts.PipelineModeBatch,
					Image:   "myorg/pipeline:v1",
					Timeout: "30m",
					Lineage: &contracts.PipelineLineage{
						Enabled:           true,
						HeartbeatInterval: "30s",
					},
				},
			},
			wantWarnings: 1, // Heartbeat ignored in batch mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewPipelineValidator(&tt.pipeline, "pipeline.yaml")
			errs := v.Validate(context.Background())
			warningCount := len(errs.Warnings())
			if warningCount != tt.wantWarnings {
				t.Errorf("Validate() warnings = %d, want %d. Warnings: %v", warningCount, tt.wantWarnings, errs.Warnings())
			}
		})
	}
}
