package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dpv1alpha1 "github.com/Infoblox-CTO/platform.data.kit/platform/controller/api/v1alpha1"
)

func TestJobGenerator_Generate(t *testing.T) {
	generator := NewJobGenerator()

	tests := []struct {
		name       string
		deployment *dpv1alpha1.PackageDeployment
		wantErr    bool
		check      func(t *testing.T, deployment *dpv1alpha1.PackageDeployment)
	}{
		{
			name: "valid batch deployment",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-batch",
					Namespace: "default",
					UID:       types.UID("test-uid"),
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Mode: dpv1alpha1.PipelineModeBatch,
					Package: dpv1alpha1.PackageRef{
						Name:      "my-pipeline",
						Version:   "v1.0.0",
						Namespace: "data-team",
						Registry:  "registry.example.com",
					},
					Timeout: "30m",
				},
			},
			wantErr: false,
		},
		{
			name: "batch with default mode (empty)",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-batch-default",
					Namespace: "default",
					UID:       types.UID("test-uid-2"),
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Mode: "", // Empty defaults to batch
					Package: dpv1alpha1.PackageRef{
						Name:      "my-pipeline",
						Version:   "v1.0.0",
						Namespace: "data-team",
						Registry:  "registry.example.com",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "streaming mode rejected",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-streaming",
					Namespace: "default",
					UID:       types.UID("test-uid-3"),
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Mode: dpv1alpha1.PipelineModeStreaming,
					Package: dpv1alpha1.PackageRef{
						Name:      "my-pipeline",
						Version:   "v1.0.0",
						Namespace: "data-team",
						Registry:  "registry.example.com",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job, err := generator.Generate(tt.deployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && job != nil {
				// Verify basic job properties
				if job.Namespace != tt.deployment.Namespace {
					t.Errorf("Job namespace = %v, want %v", job.Namespace, tt.deployment.Namespace)
				}

				// Verify labels
				if job.Labels["datakit.infoblox.dev/package"] != tt.deployment.Spec.Package.Name {
					t.Errorf("Job label datakit.infoblox.dev/package = %v, want %v", job.Labels["datakit.infoblox.dev/package"], tt.deployment.Spec.Package.Name)
				}
				if job.Labels["datakit.infoblox.dev/mode"] != "batch" {
					t.Errorf("Job label datakit.infoblox.dev/mode = %v, want batch", job.Labels["datakit.infoblox.dev/mode"])
				}

				// Verify owner reference
				if len(job.OwnerReferences) != 1 {
					t.Errorf("Job owner references count = %d, want 1", len(job.OwnerReferences))
				} else if job.OwnerReferences[0].Name != tt.deployment.Name {
					t.Errorf("Job owner reference name = %v, want %v", job.OwnerReferences[0].Name, tt.deployment.Name)
				}

				// Verify pod spec
				if len(job.Spec.Template.Spec.Containers) != 1 {
					t.Errorf("Job container count = %d, want 1", len(job.Spec.Template.Spec.Containers))
				}
			}
		})
	}
}

func TestJobGenerator_GenerateWithTimeout(t *testing.T) {
	generator := NewJobGenerator()

	deployment := &dpv1alpha1.PackageDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-batch-timeout",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: dpv1alpha1.PackageDeploymentSpec{
			Mode: dpv1alpha1.PipelineModeBatch,
			Package: dpv1alpha1.PackageRef{
				Name:      "my-pipeline",
				Version:   "v1.0.0",
				Namespace: "data-team",
				Registry:  "registry.example.com",
			},
			Timeout: "1h",
		},
	}

	job, err := generator.Generate(deployment)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// 1 hour = 3600 seconds
	if job.Spec.ActiveDeadlineSeconds == nil {
		t.Error("ActiveDeadlineSeconds is nil, expected 3600")
	} else if *job.Spec.ActiveDeadlineSeconds != 3600 {
		t.Errorf("ActiveDeadlineSeconds = %d, want 3600", *job.Spec.ActiveDeadlineSeconds)
	}
}

func TestJobGenerator_GenerateCronJob(t *testing.T) {
	generator := NewJobGenerator()

	tests := []struct {
		name       string
		deployment *dpv1alpha1.PackageDeployment
		wantErr    bool
	}{
		{
			name: "valid cronjob",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cron",
					Namespace: "default",
					UID:       types.UID("test-uid"),
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Mode: dpv1alpha1.PipelineModeBatch,
					Package: dpv1alpha1.PackageRef{
						Name:      "my-pipeline",
						Version:   "v1.0.0",
						Namespace: "data-team",
						Registry:  "registry.example.com",
					},
					Schedule: &dpv1alpha1.ScheduleSpec{
						Cron: "0 * * * *",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing schedule",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-no-schedule",
					Namespace: "default",
					UID:       types.UID("test-uid-2"),
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Mode: dpv1alpha1.PipelineModeBatch,
					Package: dpv1alpha1.PackageRef{
						Name:      "my-pipeline",
						Version:   "v1.0.0",
						Namespace: "data-team",
						Registry:  "registry.example.com",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "streaming mode rejected",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-streaming-cron",
					Namespace: "default",
					UID:       types.UID("test-uid-3"),
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Mode: dpv1alpha1.PipelineModeStreaming,
					Package: dpv1alpha1.PackageRef{
						Name:      "my-pipeline",
						Version:   "v1.0.0",
						Namespace: "data-team",
						Registry:  "registry.example.com",
					},
					Schedule: &dpv1alpha1.ScheduleSpec{
						Cron: "0 * * * *",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronJob, err := generator.GenerateCronJob(tt.deployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateCronJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cronJob != nil {
				// Verify basic cronjob properties
				if cronJob.Namespace != tt.deployment.Namespace {
					t.Errorf("CronJob namespace = %v, want %v", cronJob.Namespace, tt.deployment.Namespace)
				}
				if cronJob.Spec.Schedule != tt.deployment.Spec.Schedule.Cron {
					t.Errorf("CronJob schedule = %v, want %v", cronJob.Spec.Schedule, tt.deployment.Spec.Schedule.Cron)
				}
			}
		})
	}
}
