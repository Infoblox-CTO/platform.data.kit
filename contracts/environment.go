package contracts

import "time"

// Environment represents a deployment environment.
type Environment struct {
	// APIVersion is the API version.
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Environment".
	Kind string `json:"kind" yaml:"kind"`

	// Metadata contains environment metadata.
	Metadata EnvironmentMetadata `json:"metadata" yaml:"metadata"`

	// Spec contains the environment specification.
	Spec EnvironmentSpec `json:"spec" yaml:"spec"`
}

// EnvironmentMetadata contains metadata for an environment.
type EnvironmentMetadata struct {
	// Name is the environment name (e.g., dev, staging, prod).
	Name string `json:"name" yaml:"name"`

	// Labels are key-value pairs for organizing.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// EnvironmentSpec contains the environment specification.
type EnvironmentSpec struct {
	// Description describes the environment.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Tier is the environment tier (development, staging, production).
	Tier string `json:"tier" yaml:"tier"`

	// Cluster is the Kubernetes cluster name.
	Cluster string `json:"cluster,omitempty" yaml:"cluster,omitempty"`

	// Namespace is the Kubernetes namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// Bindings is the name of the bindings to use.
	Bindings string `json:"bindings,omitempty" yaml:"bindings,omitempty"`

	// ApprovalRequired indicates if promotion requires approval.
	ApprovalRequired bool `json:"approvalRequired,omitempty" yaml:"approvalRequired,omitempty"`
}

// PackageDeployment represents a deployed package in an environment.
type PackageDeployment struct {
	// PackageRef references the deployed package.
	PackageRef ArtifactRef `json:"packageRef" yaml:"packageRef"`

	// Environment is the deployment environment.
	Environment string `json:"environment" yaml:"environment"`

	// Status is the deployment status.
	Status DeploymentStatus `json:"status" yaml:"status"`

	// DeployedAt is when the package was deployed.
	DeployedAt time.Time `json:"deployedAt" yaml:"deployedAt"`

	// DeployedBy is who deployed the package.
	DeployedBy string `json:"deployedBy,omitempty" yaml:"deployedBy,omitempty"`

	// Replicas is the number of running instances.
	Replicas int `json:"replicas,omitempty" yaml:"replicas,omitempty"`
}

// DeploymentStatus represents the status of a deployment.
type DeploymentStatus string

const (
	// DeploymentStatusPending means deployment is pending.
	DeploymentStatusPending DeploymentStatus = "pending"

	// DeploymentStatusDeploying means deployment is in progress.
	DeploymentStatusDeploying DeploymentStatus = "deploying"

	// DeploymentStatusRunning means the package is running.
	DeploymentStatusRunning DeploymentStatus = "running"

	// DeploymentStatusFailed means deployment failed.
	DeploymentStatusFailed DeploymentStatus = "failed"

	// DeploymentStatusStopped means the package is stopped.
	DeploymentStatusStopped DeploymentStatus = "stopped"
)

// PromotionRecord records a promotion between environments.
type PromotionRecord struct {
	// ID is the unique identifier for this promotion.
	ID string `json:"id" yaml:"id"`

	// PackageRef references the promoted package.
	PackageRef ArtifactRef `json:"packageRef" yaml:"packageRef"`

	// FromEnvironment is the source environment.
	FromEnvironment string `json:"fromEnvironment" yaml:"fromEnvironment"`

	// ToEnvironment is the target environment.
	ToEnvironment string `json:"toEnvironment" yaml:"toEnvironment"`

	// Status is the promotion status.
	Status PromotionStatus `json:"status" yaml:"status"`

	// InitiatedAt is when the promotion was initiated.
	InitiatedAt time.Time `json:"initiatedAt" yaml:"initiatedAt"`

	// CompletedAt is when the promotion completed.
	CompletedAt *time.Time `json:"completedAt,omitempty" yaml:"completedAt,omitempty"`

	// InitiatedBy is who initiated the promotion.
	InitiatedBy string `json:"initiatedBy,omitempty" yaml:"initiatedBy,omitempty"`

	// ApprovedBy is who approved the promotion.
	ApprovedBy string `json:"approvedBy,omitempty" yaml:"approvedBy,omitempty"`
}

// PromotionStatus represents the status of a promotion.
type PromotionStatus string

const (
	// PromotionStatusPending means promotion is pending.
	PromotionStatusPending PromotionStatus = "pending"

	// PromotionStatusAwaitingApproval means promotion needs approval.
	PromotionStatusAwaitingApproval PromotionStatus = "awaiting-approval"

	// PromotionStatusApproved means promotion is approved.
	PromotionStatusApproved PromotionStatus = "approved"

	// PromotionStatusInProgress means promotion is in progress.
	PromotionStatusInProgress PromotionStatus = "in-progress"

	// PromotionStatusCompleted means promotion completed successfully.
	PromotionStatusCompleted PromotionStatus = "completed"

	// PromotionStatusFailed means promotion failed.
	PromotionStatusFailed PromotionStatus = "failed"

	// PromotionStatusRejected means promotion was rejected.
	PromotionStatusRejected PromotionStatus = "rejected"
)
