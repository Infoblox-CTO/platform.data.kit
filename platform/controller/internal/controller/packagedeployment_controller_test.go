package controller

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dpv1alpha1 "github.com/Infoblox-CTO/data.platform.kit/platform/controller/api/v1alpha1"
)

func TestPackageDeploymentReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := dpv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	tests := []struct {
		name       string
		deployment *dpv1alpha1.PackageDeployment
		wantErr    bool
	}{
		{
			name: "valid deployment - pending phase",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Package: dpv1alpha1.PackageRef{
						Name:     "test-pkg",
						Version:  "v1.0.0",
						Registry: "registry.example.com",
					},
				},
				Status: dpv1alpha1.PackageDeploymentStatus{
					Phase: dpv1alpha1.PhasePending,
				},
			},
			wantErr: false,
		},
		{
			name: "valid deployment - empty phase",
			deployment: &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Package: dpv1alpha1.PackageRef{
						Name:     "test-pkg",
						Version:  "v1.0.0",
						Registry: "registry.example.com",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.deployment).
				WithStatusSubresource(tt.deployment).
				Build()

			reconciler := &PackageDeploymentReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tt.deployment.Name,
					Namespace: tt.deployment.Namespace,
				},
			}

			_, err := reconciler.Reconcile(context.Background(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPackageDeploymentReconciler_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := dpv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	reconciler := &PackageDeploymentReconciler{
		Client: client,
		Scheme: scheme,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(context.Background(), req)
	if err != nil {
		t.Errorf("Reconcile() should not return error for not found: %v", err)
	}
}

func TestPackageDeploymentReconciler_Phases(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := dpv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add scheme: %v", err)
	}

	phases := []dpv1alpha1.DeploymentPhase{
		dpv1alpha1.PhasePending,
		dpv1alpha1.PhasePulling,
		dpv1alpha1.PhaseReady,
		dpv1alpha1.PhaseRunning,
		dpv1alpha1.PhaseFailed,
	}

	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			deployment := &dpv1alpha1.PackageDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: dpv1alpha1.PackageDeploymentSpec{
					Package: dpv1alpha1.PackageRef{
						Name:     "test-pkg",
						Version:  "v1.0.0",
						Registry: "registry.example.com",
					},
				},
				Status: dpv1alpha1.PackageDeploymentStatus{
					Phase: phase,
				},
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(deployment).
				WithStatusSubresource(deployment).
				Build()

			reconciler := &PackageDeploymentReconciler{
				Client: client,
				Scheme: scheme,
			}

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      deployment.Name,
					Namespace: deployment.Namespace,
				},
			}

			_, err := reconciler.Reconcile(context.Background(), req)
			if err != nil {
				t.Errorf("Reconcile() error = %v for phase %s", err, phase)
			}
		})
	}
}
