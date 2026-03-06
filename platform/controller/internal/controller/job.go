// Package controller contains Kubernetes workload generators for PackageDeployment.
package controller

import (
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dpv1alpha1 "github.com/Infoblox-CTO/platform.data.kit/platform/controller/api/v1alpha1"
)

// JobGenerator generates Kubernetes Jobs for batch pipelines.
type JobGenerator struct{}

// NewJobGenerator creates a new job generator.
func NewJobGenerator() *JobGenerator {
	return &JobGenerator{}
}

// Generate generates a Kubernetes Job from a PackageDeployment.
func (g *JobGenerator) Generate(deployment *dpv1alpha1.PackageDeployment) (*batchv1.Job, error) {
	if deployment.Spec.Mode == dpv1alpha1.PipelineModeStreaming {
		return nil, fmt.Errorf("cannot generate Job for streaming pipeline")
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       deployment.Spec.Package.Name,
		"app.kubernetes.io/version":    deployment.Spec.Package.Version,
		"app.kubernetes.io/managed-by": "dk-controller",
		"datakit.infoblox.dev/package": deployment.Spec.Package.Name,
		"datakit.infoblox.dev/mode":    "batch",
	}

	// Build container
	container := g.buildContainer(deployment)

	// Build pod template spec
	podSpec := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		ServiceAccountName: deployment.Spec.ServiceAccountName,
		Containers:         []corev1.Container{container},
	}

	// Add image pull secrets
	for _, secret := range deployment.Spec.ImagePullSecrets {
		podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: secret.Name,
		})
	}

	// Calculate active deadline
	var activeDeadlineSeconds *int64
	if deployment.Spec.Timeout != "" {
		duration, err := time.ParseDuration(deployment.Spec.Timeout)
		if err == nil {
			seconds := int64(duration.Seconds())
			activeDeadlineSeconds = &seconds
		}
	}

	// Build job spec
	backoffLimit := int32(3)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", deployment.Name, time.Now().Format("20060102-150405")),
			Namespace: deployment.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dpv1alpha1.GroupVersion.String(),
					Kind:       "PackageDeployment",
					Name:       deployment.Name,
					UID:        deployment.UID,
				},
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          &backoffLimit,
			ActiveDeadlineSeconds: activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podSpec,
			},
		},
	}

	return job, nil
}

// GenerateCronJob generates a Kubernetes CronJob for scheduled batch pipelines.
func (g *JobGenerator) GenerateCronJob(deployment *dpv1alpha1.PackageDeployment) (*batchv1.CronJob, error) {
	if deployment.Spec.Mode == dpv1alpha1.PipelineModeStreaming {
		return nil, fmt.Errorf("cannot generate CronJob for streaming pipeline")
	}

	if deployment.Spec.Schedule == nil || deployment.Spec.Schedule.Cron == "" {
		return nil, fmt.Errorf("schedule cron expression is required for CronJob")
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       deployment.Spec.Package.Name,
		"app.kubernetes.io/version":    deployment.Spec.Package.Version,
		"app.kubernetes.io/managed-by": "dk-controller",
		"datakit.infoblox.dev/package": deployment.Spec.Package.Name,
		"datakit.infoblox.dev/mode":    "batch",
	}

	// Build container
	container := g.buildContainer(deployment)

	// Build pod template spec
	podSpec := corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyNever,
		ServiceAccountName: deployment.Spec.ServiceAccountName,
		Containers:         []corev1.Container{container},
	}

	// Add image pull secrets
	for _, secret := range deployment.Spec.ImagePullSecrets {
		podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: secret.Name,
		})
	}

	// Calculate active deadline
	var activeDeadlineSeconds *int64
	if deployment.Spec.Timeout != "" {
		duration, err := time.ParseDuration(deployment.Spec.Timeout)
		if err == nil {
			seconds := int64(duration.Seconds())
			activeDeadlineSeconds = &seconds
		}
	}

	// Build job spec
	backoffLimit := int32(3)
	suspend := deployment.Spec.Schedule.Suspend

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dpv1alpha1.GroupVersion.String(),
					Kind:       "PackageDeployment",
					Name:       deployment.Name,
					UID:        deployment.UID,
				},
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          deployment.Spec.Schedule.Cron,
			TimeZone:          &deployment.Spec.Schedule.Timezone,
			Suspend:           &suspend,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: batchv1.JobSpec{
					BackoffLimit:          &backoffLimit,
					ActiveDeadlineSeconds: activeDeadlineSeconds,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: labels,
						},
						Spec: podSpec,
					},
				},
			},
		},
	}

	return cronJob, nil
}

// buildContainer builds the container spec for a job.
func (g *JobGenerator) buildContainer(deployment *dpv1alpha1.PackageDeployment) corev1.Container {
	image := fmt.Sprintf("%s/%s/%s:%s",
		deployment.Spec.Package.Registry,
		deployment.Spec.Package.Namespace,
		deployment.Spec.Package.Name,
		deployment.Spec.Package.Version,
	)

	container := corev1.Container{
		Name:            "pipeline",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
	}

	// Add resource requirements
	if deployment.Spec.Resources != nil {
		container.Resources = g.buildResourceRequirements(deployment.Spec.Resources)
	}

	return container
}

// buildResourceRequirements builds Kubernetes resource requirements.
func (g *JobGenerator) buildResourceRequirements(spec *dpv1alpha1.ResourceSpec) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if spec.Requests.CPU != "" {
		requirements.Requests[corev1.ResourceCPU] = resource.MustParse(spec.Requests.CPU)
	}
	if spec.Requests.Memory != "" {
		requirements.Requests[corev1.ResourceMemory] = resource.MustParse(spec.Requests.Memory)
	}

	if spec.Limits.CPU != "" {
		requirements.Limits[corev1.ResourceCPU] = resource.MustParse(spec.Limits.CPU)
	}
	if spec.Limits.Memory != "" {
		requirements.Limits[corev1.ResourceMemory] = resource.MustParse(spec.Limits.Memory)
	}

	return requirements
}
