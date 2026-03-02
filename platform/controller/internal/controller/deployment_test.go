package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	dpv1alpha1 "github.com/Infoblox-CTO/platform.data.kit/platform/controller/api/v1alpha1"
)

func TestDeploymentGenerator_Generate(t *testing.T) {
	generator := NewDeploymentGenerator()

	tests := []struct {
		name       string
		deployment *dpv1alpha1.PackageDeployment
		wantErr    bool
	}{
		{
			name: "valid streaming deployment",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-streaming",
					Namespace: "default",
					UID:       types.UID("test-uid"),
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
			wantErr: false,
		},
		{
			name: "batch mode rejected",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-batch",
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
			name: "empty mode rejected (defaults to batch)",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-empty-mode",
					Namespace: "default",
					UID:       types.UID("test-uid-3"),
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploy, err := generator.Generate(tt.deployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && deploy != nil {
				// Verify basic deployment properties
				if deploy.Name != tt.deployment.Name {
					t.Errorf("Deployment name = %v, want %v", deploy.Name, tt.deployment.Name)
				}
				if deploy.Namespace != tt.deployment.Namespace {
					t.Errorf("Deployment namespace = %v, want %v", deploy.Namespace, tt.deployment.Namespace)
				}

				// Verify labels
				if deploy.Labels["datakit.infoblox.dev/package"] != tt.deployment.Spec.Package.Name {
					t.Errorf("Deployment label datakit.infoblox.dev/package = %v, want %v", deploy.Labels["datakit.infoblox.dev/package"], tt.deployment.Spec.Package.Name)
				}
				if deploy.Labels["datakit.infoblox.dev/mode"] != "streaming" {
					t.Errorf("Deployment label datakit.infoblox.dev/mode = %v, want streaming", deploy.Labels["datakit.infoblox.dev/mode"])
				}

				// Verify owner reference
				if len(deploy.OwnerReferences) != 1 {
					t.Errorf("Deployment owner references count = %d, want 1", len(deploy.OwnerReferences))
				} else if deploy.OwnerReferences[0].Name != tt.deployment.Name {
					t.Errorf("Deployment owner reference name = %v, want %v", deploy.OwnerReferences[0].Name, tt.deployment.Name)
				}

				// Verify pod spec
				if len(deploy.Spec.Template.Spec.Containers) != 1 {
					t.Errorf("Deployment container count = %d, want 1", len(deploy.Spec.Template.Spec.Containers))
				}
			}
		})
	}
}

func TestDeploymentGenerator_GenerateWithReplicas(t *testing.T) {
	generator := NewDeploymentGenerator()

	replicas := int32(3)
	deployment := &dpv1alpha1.PackageDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicas",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: dpv1alpha1.PackageDeploymentSpec{
			Mode: dpv1alpha1.PipelineModeStreaming,
			Package: dpv1alpha1.PackageRef{
				Name:      "my-pipeline",
				Version:   "v1.0.0",
				Namespace: "data-team",
				Registry:  "registry.example.com",
			},
			Replicas: &replicas,
		},
	}

	deploy, err := generator.Generate(deployment)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if deploy.Spec.Replicas == nil {
		t.Error("Replicas is nil")
	} else if *deploy.Spec.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", *deploy.Spec.Replicas)
	}
}

func TestDeploymentGenerator_GenerateWithProbes(t *testing.T) {
	generator := NewDeploymentGenerator()

	deployment := &dpv1alpha1.PackageDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-probes",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: dpv1alpha1.PackageDeploymentSpec{
			Mode: dpv1alpha1.PipelineModeStreaming,
			Package: dpv1alpha1.PackageRef{
				Name:      "my-pipeline",
				Version:   "v1.0.0",
				Namespace: "data-team",
				Registry:  "registry.example.com",
			},
			LivenessProbe: &dpv1alpha1.Probe{
				HTTPGet: &dpv1alpha1.HTTPGetAction{
					Path: "/healthz",
					Port: 8080,
				},
				PeriodSeconds: 10,
			},
			ReadinessProbe: &dpv1alpha1.Probe{
				HTTPGet: &dpv1alpha1.HTTPGetAction{
					Path: "/ready",
					Port: 8080,
				},
				PeriodSeconds: 5,
			},
		},
	}

	deploy, err := generator.Generate(deployment)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	container := deploy.Spec.Template.Spec.Containers[0]

	// Verify liveness probe
	if container.LivenessProbe == nil {
		t.Error("LivenessProbe is nil")
	} else {
		if container.LivenessProbe.HTTPGet == nil {
			t.Error("LivenessProbe.HTTPGet is nil")
		} else if container.LivenessProbe.HTTPGet.Path != "/healthz" {
			t.Errorf("LivenessProbe.HTTPGet.Path = %v, want /healthz", container.LivenessProbe.HTTPGet.Path)
		}
	}

	// Verify readiness probe
	if container.ReadinessProbe == nil {
		t.Error("ReadinessProbe is nil")
	} else {
		if container.ReadinessProbe.HTTPGet == nil {
			t.Error("ReadinessProbe.HTTPGet is nil")
		} else if container.ReadinessProbe.HTTPGet.Path != "/ready" {
			t.Errorf("ReadinessProbe.HTTPGet.Path = %v, want /ready", container.ReadinessProbe.HTTPGet.Path)
		}
	}
}

func TestDeploymentGenerator_GenerateWithTerminationGracePeriod(t *testing.T) {
	generator := NewDeploymentGenerator()

	gracePeriod := int64(60)
	deployment := &dpv1alpha1.PackageDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grace-period",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: dpv1alpha1.PackageDeploymentSpec{
			Mode: dpv1alpha1.PipelineModeStreaming,
			Package: dpv1alpha1.PackageRef{
				Name:      "my-pipeline",
				Version:   "v1.0.0",
				Namespace: "data-team",
				Registry:  "registry.example.com",
			},
			TerminationGracePeriodSeconds: &gracePeriod,
		},
	}

	deploy, err := generator.Generate(deployment)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if deploy.Spec.Template.Spec.TerminationGracePeriodSeconds == nil {
		t.Error("TerminationGracePeriodSeconds is nil")
	} else if *deploy.Spec.Template.Spec.TerminationGracePeriodSeconds != 60 {
		t.Errorf("TerminationGracePeriodSeconds = %d, want 60", *deploy.Spec.Template.Spec.TerminationGracePeriodSeconds)
	}
}

func TestDeploymentGenerator_GenerateDefaultReplicas(t *testing.T) {
	generator := NewDeploymentGenerator()

	deployment := &dpv1alpha1.PackageDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-default-replicas",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: dpv1alpha1.PackageDeploymentSpec{
			Mode: dpv1alpha1.PipelineModeStreaming,
			Package: dpv1alpha1.PackageRef{
				Name:      "my-pipeline",
				Version:   "v1.0.0",
				Namespace: "data-team",
				Registry:  "registry.example.com",
			},
			// No replicas specified
		},
	}

	deploy, err := generator.Generate(deployment)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Default should be 1 replica
	if deploy.Spec.Replicas == nil {
		t.Error("Replicas is nil, expected default of 1")
	} else if *deploy.Spec.Replicas != 1 {
		t.Errorf("Replicas = %d, want 1 (default)", *deploy.Spec.Replicas)
	}
}
