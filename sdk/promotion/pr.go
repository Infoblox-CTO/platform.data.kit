// Package promotion provides services for promoting data packages between environments.
package promotion

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Promoter implements the Service interface.
type Promoter struct {
	// GitHubClient is the GitHub API client.
	GitHubClient *GitHubClient
	// KustomizeUpdater is the Kustomize overlay updater.
	KustomizeUpdater *FileKustomizeUpdater
	// BaseBranch is the base branch for PRs (default: main).
	BaseBranch string
	// RecordGenerator is the promotion record generator.
	RecordGenerator *RecordGenerator
}

// NewPromoter creates a new Promoter.
func NewPromoter(ghClient *GitHubClient, kustomizeUpdater *FileKustomizeUpdater) *Promoter {
	return &Promoter{
		GitHubClient:     ghClient,
		KustomizeUpdater: kustomizeUpdater,
		BaseBranch:       "main",
		RecordGenerator:  NewRecordGenerator(),
	}
}

// Promote promotes a package to the target environment.
func (p *Promoter) Promote(ctx context.Context, req *PromotionRequest) (*PromotionResult, error) {
	// Validate request
	if err := p.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get current version in target environment
	currentVersion, err := p.KustomizeUpdater.GetCurrentVersion(ctx, req.TargetEnv, req.Package)
	if err != nil {
		return nil, fmt.Errorf("getting current version: %w", err)
	}

	// Generate promotion record
	record := p.RecordGenerator.Generate(req, currentVersion)

	if req.DryRun {
		return &PromotionResult{
			Success: true,
			DryRun:  true,
			Record:  record,
		}, nil
	}

	// Create branch name
	branchName := p.branchName(req)

	// Create branch on GitHub
	if err := p.GitHubClient.CreateBranch(ctx, p.BaseBranch, branchName); err != nil {
		return nil, fmt.Errorf("creating branch: %w", err)
	}

	// Generate and upload version file
	versionContent, err := p.generateVersionFile(req)
	if err != nil {
		return nil, fmt.Errorf("generating version file: %w", err)
	}

	versionPath := p.versionFilePath(req)
	commitMsg := fmt.Sprintf("Promote %s to %s at version %s", req.Package, req.TargetEnv, req.Version)

	if err := p.GitHubClient.UpdateFile(ctx, branchName, versionPath, versionContent, commitMsg); err != nil {
		return nil, fmt.Errorf("updating version file: %w", err)
	}

	// Create pull request
	prReq := p.createPRRequest(req, currentVersion)
	prReq.Head = branchName
	prReq.Base = p.BaseBranch

	prInfo, err := p.GitHubClient.CreatePR(ctx, prReq)
	if err != nil {
		return nil, fmt.Errorf("creating pull request: %w", err)
	}

	// Enable auto-merge if requested
	if req.AutoMerge {
		if err := p.GitHubClient.EnableAutoMerge(ctx, prInfo.Number); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to enable auto-merge: %v\n", err)
		}
	}

	// Update record with PR info
	record.PRNumber = prInfo.Number
	record.PRURL = prInfo.URL

	return &PromotionResult{
		Success:  true,
		PRNumber: prInfo.Number,
		PRURL:    prInfo.URL,
		Branch:   branchName,
		Record:   record,
		DryRun:   false,
	}, nil
}

// GetStatus returns the status of a promotion by PR number.
func (p *Promoter) GetStatus(ctx context.Context, prNumber int) (*PromotionStatus, error) {
	prInfo, err := p.GitHubClient.GetPR(ctx, prNumber)
	if err != nil {
		return nil, fmt.Errorf("getting PR info: %w", err)
	}

	return &PromotionStatus{
		PRNumber: prInfo.Number,
		State:    prInfo.State,
		Merged:   prInfo.Merged,
		MergedAt: prInfo.MergedAt,
	}, nil
}

// ListPromotions lists promotions for a package.
func (p *Promoter) ListPromotions(ctx context.Context, packageName string, limit int) ([]*PromotionRecord, error) {
	// In MVP, we don't persist promotion records
	// This would require a database or file storage
	return nil, nil
}

// validateRequest validates a promotion request.
func (p *Promoter) validateRequest(req *PromotionRequest) error {
	if req.Package == "" {
		return fmt.Errorf("package name is required")
	}
	if req.Version == "" {
		return fmt.Errorf("version is required")
	}
	if !req.TargetEnv.Valid() {
		return fmt.Errorf("invalid target environment: %s", req.TargetEnv)
	}
	return nil
}

// branchName generates the branch name for a promotion.
func (p *Promoter) branchName(req *PromotionRequest) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("promote/%s/%s/%s/%s", req.Package, req.TargetEnv, req.Version, timestamp)
}

// versionFilePath returns the path to the version file in the repository.
func (p *Promoter) versionFilePath(req *PromotionRequest) string {
	return fmt.Sprintf("gitops/environments/%s/packages/%s/version.yaml", req.TargetEnv, req.Package)
}

// generateVersionFile generates the content of the version file.
func (p *Promoter) generateVersionFile(req *PromotionRequest) (string, error) {
	registry := req.Registry
	if registry == "" {
		registry = "ghcr.io/infoblox-cto"
	}

	vf := &VersionFile{
		APIVersion: "dp.io/v1alpha1",
		Kind:       "PackageVersion",
		Metadata: VersionMeta{
			Name: req.Package,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "dp",
				"dp.io/package":              req.Package,
				"dp.io/environment":          req.TargetEnv.String(),
			},
		},
		Spec: VersionSpec{
			Package: PackageRef{
				Name:     req.Package,
				Version:  req.Version,
				Registry: registry,
				Digest:   req.Digest,
			},
		},
	}

	data, err := yaml.Marshal(vf)
	if err != nil {
		return "", fmt.Errorf("marshaling version file: %w", err)
	}

	// Base64 encode for GitHub API
	return base64.StdEncoding.EncodeToString(data), nil
}

// createPRRequest creates the pull request request.
func (p *Promoter) createPRRequest(req *PromotionRequest, currentVersion string) *CreatePRRequest {
	title := fmt.Sprintf("Promote %s to %s: %s", req.Package, req.TargetEnv, req.Version)

	body := fmt.Sprintf(`## Package Promotion

**Package:** %s
**Version:** %s
**Environment:** %s
`, req.Package, req.Version, req.TargetEnv)

	if currentVersion != "" {
		body += fmt.Sprintf("**Previous Version:** %s\n", currentVersion)
	}

	if req.Digest != "" {
		body += fmt.Sprintf("**Digest:** %s\n", req.Digest)
	}

	body += `
## Checklist

- [ ] Version verified in registry
- [ ] Tests passed in CI
- [ ] Ready for deployment

---
*Generated by DP CLI*
`

	labels := []string{
		"promotion",
		fmt.Sprintf("env:%s", req.TargetEnv),
		fmt.Sprintf("package:%s", req.Package),
	}

	return &CreatePRRequest{
		Title:  title,
		Body:   body,
		Labels: labels,
	}
}
