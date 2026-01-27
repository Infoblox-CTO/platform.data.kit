// Package controller contains Kubernetes workload generators for PackageDeployment.
package controller

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	dpv1alpha1 "github.com/Infoblox-CTO/platform.data.kit/platform/controller/api/v1alpha1"
)

// DeploymentGenerator generates Kubernetes Deployments for streaming pipelines.
type DeploymentGenerator struct{}

// NewDeploymentGenerator creates a new deployment generator.
func NewDeploymentGenerator() *DeploymentGenerator {
	return &DeploymentGenerator{}
}

// Generate generates a Kubernetes Deployment from a PackageDeployment.
func (g *DeploymentGenerator) Generate(deployment *dpv1alpha1.PackageDeployment) (*appsv1.Deployment, error) {
	if deployment.Spec.Mode != dpv1alpha1.PipelineModeStreaming {
		return nil, fmt.Errorf("cannot generate Deployment for batch pipeline")
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       deployment.Spec.Package.Name,
		"app.kubernetes.io/version":    deployment.Spec.Package.Version,
		"app.kubernetes.io/managed-by": "dp-controller",
		"dp.io/package":                deployment.Spec.Package.Name,
		"dp.io/mode":                   "streaming",
	}

	// Build container
	container := g.buildContainer(deployment)

	// Add probes
	if deployment.Spec.LivenessProbe != nil {
		container.LivenessProbe = g.buildProbe(deployment.Spec.LivenessProbe)
	}
	if deployment.Spec.ReadinessProbe != nil {
		container.ReadinessProbe = g.buildProbe(deployment.Spec.ReadinessProbe)
	}

	// Build pod template spec
	podSpec := corev1.PodSpec{
		ServiceAccountName: deployment.Spec.ServiceAccountName,
		Containers:         []corev1.Container{container},
	}

	// Set termination grace period
	if deployment.Spec.TerminationGracePeriodSeconds != nil {
		podSpec.TerminationGracePeriodSeconds = deployment.Spec.TerminationGracePeriodSeconds
	}

	// Add image pull secrets
	for _, secret := range deployment.Spec.ImagePullSecrets {
		podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: secret.Name,
		})
	}

	// Set replicas
	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	k8sDeployment := &appsv1.Deployment{
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
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": deployment.Spec.Package.Name,
					"dp.io/package":          deployment.Spec.Package.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podSpec,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
		},
	}

	return k8sDeployment, nil
}

// buildContainer builds the container spec for a deployment.
func (g *DeploymentGenerator) buildContainer(deployment *dpv1alpha1.PackageDeployment) corev1.Container {
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

// buildProbe converts a dp Probe to a Kubernetes Probe.
func (g *DeploymentGenerator) buildProbe(probe *dpv1alpha1.Probe) *corev1.Probe {
	if probe == nil {
		return nil
	}

	k8sProbe := &corev1.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}

	// Set defaults
	if k8sProbe.PeriodSeconds == 0 {
		k8sProbe.PeriodSeconds = 10
	}
	if k8sProbe.TimeoutSeconds == 0 {
		k8sProbe.TimeoutSeconds = 1
	}
	if k8sProbe.SuccessThreshold == 0 {
		k8sProbe.SuccessThreshold = 1
	}
	if k8sProbe.FailureThreshold == 0 {
		k8sProbe.FailureThreshold = 3
	}

	switch {
	case probe.HTTPGet != nil:
		scheme := corev1.URISchemeHTTP
		if probe.HTTPGet.Scheme == "HTTPS" {
			scheme = corev1.URISchemeHTTPS
		}
		k8sProbe.ProbeHandler = corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   probe.HTTPGet.Path,
				Port:   intstr.FromInt32(probe.HTTPGet.Port),
				Scheme: scheme,
			},
		}

	case probe.Exec != nil:
		k8sProbe.ProbeHandler = corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: probe.Exec.Command,
			},
		}

	case probe.TCPSocket != nil:
		k8sProbe.ProbeHandler = corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt32(probe.TCPSocket.Port),
			},
		}
	}

	return k8sProbe
}

// buildResourceRequirements builds Kubernetes resource requirements.
func (g *DeploymentGenerator) buildResourceRequirements(spec *dpv1alpha1.ResourceSpec) corev1.ResourceRequirements {
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
