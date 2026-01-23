// Package controller contains the Kubernetes controller for PackageDeployment.
package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dpv1alpha1 "github.com/Infoblox-CTO/platform.data.kit/platform/controller/api/v1alpha1"
)

// PackageDeploymentReconciler reconciles a PackageDeployment object.
type PackageDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=dp.io,resources=packagedeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dp.io,resources=packagedeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dp.io,resources=packagedeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is the main reconciliation loop for PackageDeployment.
func (r *PackageDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the PackageDeployment instance
	var deployment dpv1alpha1.PackageDeployment
	if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling PackageDeployment",
		"name", deployment.Name,
		"namespace", deployment.Namespace,
		"package", deployment.Spec.Package.Name,
		"version", deployment.Spec.Package.Version,
	)

	// Update observed generation
	if deployment.Status.ObservedGeneration != deployment.Generation {
		deployment.Status.ObservedGeneration = deployment.Generation
	}

	// Handle based on current phase
	switch deployment.Status.Phase {
	case "", dpv1alpha1.PhasePending:
		return r.handlePending(ctx, &deployment)
	case dpv1alpha1.PhasePulling:
		return r.handlePulling(ctx, &deployment)
	case dpv1alpha1.PhaseReady:
		return r.handleReady(ctx, &deployment)
	case dpv1alpha1.PhaseRunning:
		return r.handleRunning(ctx, &deployment)
	case dpv1alpha1.PhaseFailed:
		return r.handleFailed(ctx, &deployment)
	default:
		logger.Info("Unknown phase", "phase", deployment.Status.Phase)
		return ctrl.Result{}, nil
	}
}

// handlePending handles a pending deployment.
func (r *PackageDeploymentReconciler) handlePending(ctx context.Context, deployment *dpv1alpha1.PackageDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling pending deployment")

	// Validate the package reference
	if deployment.Spec.Package.Name == "" || deployment.Spec.Package.Version == "" {
		r.setCondition(deployment, "Ready", metav1.ConditionFalse, "ValidationFailed", "Package name and version are required")
		deployment.Status.Phase = dpv1alpha1.PhaseFailed
		if err := r.Status().Update(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Transition to pulling phase
	deployment.Status.Phase = dpv1alpha1.PhasePulling
	r.setCondition(deployment, "Ready", metav1.ConditionFalse, "Pulling", "Pulling package from registry")

	if err := r.Status().Update(ctx, deployment); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue to continue processing
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// handlePulling handles a deployment in pulling phase.
func (r *PackageDeploymentReconciler) handlePulling(ctx context.Context, deployment *dpv1alpha1.PackageDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling pulling deployment")

	// TODO: Implement actual package pulling from OCI registry
	// For MVP, we simulate successful pull

	// Verify digest if provided
	if deployment.Spec.Package.Digest != "" {
		logger.Info("Verifying package digest", "digest", deployment.Spec.Package.Digest)
		// TODO: Implement digest verification
	}

	// Transition to ready phase
	deployment.Status.Phase = dpv1alpha1.PhaseReady
	r.setCondition(deployment, "Ready", metav1.ConditionTrue, "PackageReady", "Package pulled and ready")

	if err := r.Status().Update(ctx, deployment); err != nil {
		return ctrl.Result{}, err
	}

	// If scheduled, requeue for next run
	if deployment.Spec.Schedule != nil && deployment.Spec.Schedule.Cron != "" && !deployment.Spec.Schedule.Suspend {
		nextRun, err := r.calculateNextRun(deployment.Spec.Schedule.Cron, deployment.Spec.Schedule.Timezone)
		if err != nil {
			logger.Error(err, "Failed to calculate next run")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{RequeueAfter: time.Until(nextRun)}, nil
	}

	return ctrl.Result{}, nil
}

// handleReady handles a ready deployment.
func (r *PackageDeploymentReconciler) handleReady(ctx context.Context, deployment *dpv1alpha1.PackageDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling ready deployment")

	// Check if we should run
	if deployment.Spec.Schedule != nil && deployment.Spec.Schedule.Cron != "" {
		if deployment.Spec.Schedule.Suspend {
			logger.Info("Schedule is suspended")
			return ctrl.Result{}, nil
		}

		// Check if it's time to run
		// TODO: Implement cron parsing and scheduling
	}

	return ctrl.Result{}, nil
}

// handleRunning handles a running deployment.
func (r *PackageDeploymentReconciler) handleRunning(ctx context.Context, deployment *dpv1alpha1.PackageDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling running deployment")

	// TODO: Check job status and update accordingly
	// For MVP, we simulate completion

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// handleFailed handles a failed deployment.
func (r *PackageDeploymentReconciler) handleFailed(ctx context.Context, deployment *dpv1alpha1.PackageDeployment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling failed deployment")

	// Don't requeue failed deployments unless spec changes
	return ctrl.Result{}, nil
}

// setCondition sets a condition on the deployment status.
func (r *PackageDeploymentReconciler) setCondition(deployment *dpv1alpha1.PackageDeployment, condType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: deployment.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	meta.SetStatusCondition(&deployment.Status.Conditions, condition)
}

// calculateNextRun calculates the next run time based on cron expression.
func (r *PackageDeploymentReconciler) calculateNextRun(cron, timezone string) (time.Time, error) {
	// TODO: Implement proper cron parsing
	// For MVP, just return 1 hour from now
	return time.Now().Add(time.Hour), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PackageDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dpv1alpha1.PackageDeployment{}).
		Named("packagedeployment").
		Complete(r)
}

// ReconcileResult represents the result of a reconciliation.
type ReconcileResult struct {
	Requeue      bool
	RequeueAfter time.Duration
	Error        error
}

// String returns a string representation of the result.
func (r ReconcileResult) String() string {
	if r.Error != nil {
		return fmt.Sprintf("error: %v", r.Error)
	}
	if r.Requeue {
		if r.RequeueAfter > 0 {
			return fmt.Sprintf("requeue after %s", r.RequeueAfter)
		}
		return "requeue"
	}
	return "done"
}
