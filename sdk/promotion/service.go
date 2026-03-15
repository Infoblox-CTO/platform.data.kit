// Package promotion provides services for promoting data packages between environments.
package promotion

import (
	"context"
	"time"
)

// Environment represents a deployment environment.
type Environment string

const (
	// EnvDev is the development environment.
	EnvDev Environment = "dev"
	// EnvInt is the integration environment.
	EnvInt Environment = "int"
	// EnvProd is the production environment.
	EnvProd Environment = "prod"
)

// Valid returns true if the environment is valid.
func (e Environment) Valid() bool {
	switch e {
	case EnvDev, EnvInt, EnvProd:
		return true
	}
	return false
}

// String returns the string representation of the environment.
func (e Environment) String() string {
	return string(e)
}

// PromotionRequest represents a request to promote a package.
type PromotionRequest struct {
	// Package is the name of the package to promote.
	Package string
	// Namespace is the team/namespace of the package.
	Namespace string
	// Version is the version to promote.
	Version string
	// Digest is the content digest for verification.
	Digest string
	// Registry is the OCI registry URL.
	Registry string
	// TargetEnv is the target environment (e.g., dev, int, prod). Required.
	TargetEnv Environment
	// Cell is the target cell within the environment (e.g., "canary", "c0"). Defaults to "c0".
	Cell string
	// DryRun if true, only simulate the promotion.
	DryRun bool
	// AutoMerge if true, enable auto-merge on the PR.
	AutoMerge bool
}

// PromotionResult represents the result of a promotion operation.
type PromotionResult struct {
	// Success indicates if the promotion was successful.
	Success bool
	// PRNumber is the pull request number created.
	PRNumber int
	// PRURL is the URL of the created pull request.
	PRURL string
	// Branch is the name of the branch created.
	Branch string
	// Record is the promotion record.
	Record *PromotionRecord
	// DryRun indicates if this was a dry run.
	DryRun bool
}

// PromotionRecord captures details about a promotion.
type PromotionRecord struct {
	// ID is the unique identifier for this promotion.
	ID string
	// Package is the package name.
	Package string
	// Namespace is the team/namespace.
	Namespace string
	// Version is the promoted version.
	Version string
	// Digest is the content digest.
	Digest string
	// FromEnv is the source environment (or "registry" for first deploy).
	FromEnv string
	// ToEnv is the target environment.
	ToEnv Environment
	// Timestamp is when the promotion was initiated.
	Timestamp time.Time
	// InitiatedBy is the user who initiated the promotion.
	InitiatedBy string
	// PRNumber is the associated pull request number.
	PRNumber int
	// PRURL is the URL of the pull request.
	PRURL string
}

// Service is the interface for promotion operations.
type Service interface {
	// Promote promotes a package to the target environment.
	Promote(ctx context.Context, req *PromotionRequest) (*PromotionResult, error)
	// GetStatus returns the status of a promotion by PR number.
	GetStatus(ctx context.Context, prNumber int) (*PromotionStatus, error)
	// ListPromotions lists promotions for a package.
	ListPromotions(ctx context.Context, packageName string, limit int) ([]*PromotionRecord, error)
}

// PromotionStatus represents the status of a promotion.
type PromotionStatus struct {
	// PRNumber is the pull request number.
	PRNumber int
	// State is the current state of the PR.
	State PRState
	// Merged indicates if the PR was merged.
	Merged bool
	// MergedAt is when the PR was merged.
	MergedAt *time.Time
	// Checks contains the status of CI checks.
	Checks []CheckStatus
}

// PRState represents the state of a pull request.
type PRState string

const (
	// PRStateOpen means the PR is open.
	PRStateOpen PRState = "open"
	// PRStateClosed means the PR was closed without merging.
	PRStateClosed PRState = "closed"
	// PRStateMerged means the PR was merged.
	PRStateMerged PRState = "merged"
)

// CheckStatus represents a CI check status.
type CheckStatus struct {
	// Name is the check name.
	Name string
	// Status is the check status.
	Status string
	// Conclusion is the check conclusion.
	Conclusion string
}

// PRClient provides pull request operations.
type PRClient interface {
	// CreatePR creates a new pull request.
	CreatePR(ctx context.Context, req *CreatePRRequest) (*PRInfo, error)
	// GetPR returns information about a pull request.
	GetPR(ctx context.Context, number int) (*PRInfo, error)
	// EnableAutoMerge enables auto-merge on a pull request.
	EnableAutoMerge(ctx context.Context, number int) error
}

// CreatePRRequest represents a request to create a PR.
type CreatePRRequest struct {
	// Title is the PR title.
	Title string
	// Body is the PR body.
	Body string
	// Head is the source branch.
	Head string
	// Base is the target branch.
	Base string
	// Labels are labels to add to the PR.
	Labels []string
}

// PRInfo contains information about a pull request.
type PRInfo struct {
	// Number is the PR number.
	Number int
	// URL is the PR URL.
	URL string
	// State is the PR state.
	State PRState
	// Merged indicates if merged.
	Merged bool
	// MergedAt is the merge time.
	MergedAt *time.Time
}
