// Package promotion provides services for promoting data packages between environments.
package promotion

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"
)

// Promoter implements the Service interface.
type Promoter struct {
	// GitHubClient is the GitHub API client.
	GitHubClient *GitHubClient
	// BaseBranch is the base branch for PRs (default: main).
	BaseBranch string
	// RecordGenerator is the promotion record generator.
	RecordGenerator *RecordGenerator
}

// NewPromoter creates a new Promoter.
func NewPromoter(ghClient *GitHubClient) *Promoter {
	return &Promoter{
		GitHubClient:    ghClient,
		BaseBranch:      "main",
		RecordGenerator: NewRecordGenerator(),
	}
}

// Promote promotes a package to the target environment and cell.
// TargetEnv is always required. Cell defaults to "c0" if empty.
func (p *Promoter) Promote(ctx context.Context, req *PromotionRequest) (*PromotionResult, error) {
	if err := p.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	cell := ResolveCell(req.Cell)

	// Get current version (for record keeping).
	currentVersion := ""
	if !req.DryRun && p.GitHubClient != nil {
		currentVersion, _ = p.GetCurrentVersion(ctx, req.TargetEnv, cell, req.Package)
	}

	// Generate promotion record.
	record := p.RecordGenerator.Generate(req, currentVersion)

	if req.DryRun {
		return &PromotionResult{
			Success: true,
			DryRun:  true,
			Record:  record,
		}, nil
	}

	// Create branch name.
	branchName := p.branchName(req, cell)

	// Create branch on GitHub.
	if err := p.GitHubClient.CreateBranch(ctx, p.BaseBranch, branchName); err != nil {
		return nil, fmt.Errorf("creating branch: %w", err)
	}

	// Update values.yaml.
	valuesPath := ValuesFilePath(req.TargetEnv, cell, req.Package)

	// Read existing values to preserve overrides.
	existing, _ := p.getFileContent(ctx, p.BaseBranch, valuesPath)

	var content []byte
	var err error
	if len(existing) > 0 {
		content, err = MergeAppVersion(existing, req.Version)
	} else {
		var s string
		s, err = GenerateValuesContent(req.Version)
		content = []byte(s)
	}
	if err != nil {
		return nil, fmt.Errorf("generating values: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(content)
	commitMsg := fmt.Sprintf("Promote %s to %s/%s at version %s", req.Package, req.TargetEnv, cell, req.Version)

	if err := p.GitHubClient.UpdateFile(ctx, branchName, valuesPath, encoded, commitMsg); err != nil {
		return nil, fmt.Errorf("updating values file: %w", err)
	}

	// Create pull request.
	prReq := p.createPRRequest(req, cell, currentVersion)
	prReq.Head = branchName
	prReq.Base = p.BaseBranch

	prInfo, err := p.GitHubClient.CreatePR(ctx, prReq)
	if err != nil {
		return nil, fmt.Errorf("creating pull request: %w", err)
	}

	// Enable auto-merge if requested.
	if req.AutoMerge {
		if err := p.GitHubClient.EnableAutoMerge(ctx, prInfo.Number); err != nil {
			fmt.Printf("Warning: failed to enable auto-merge: %v\n", err)
		}
	}

	// Update record with PR info.
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
	return nil, nil
}

// GetCurrentVersion reads the current appVersion for a package in an env/cell
// via the GitHub Contents API.
func (p *Promoter) GetCurrentVersion(ctx context.Context, env Environment, cell, pkg string) (string, error) {
	path := ValuesFilePath(env, cell, pkg)
	content, err := p.getFileContent(ctx, p.BaseBranch, path)
	if err != nil {
		return "", nil // Not found → no version deployed
	}
	return ParseAppVersion(content), nil
}

// getFileContent reads a file from the repo via GitHub Contents API.
func (p *Promoter) getFileContent(ctx context.Context, ref, path string) ([]byte, error) {
	return p.GitHubClient.GetFileContent(ctx, ref, path)
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
		return fmt.Errorf("--to environment is required (dev, int, prod)")
	}
	return nil
}

// branchName generates the branch name for a promotion.
func (p *Promoter) branchName(req *PromotionRequest, cell string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("promote/%s/%s/%s/%s/%s", req.Package, req.TargetEnv, cell, req.Version, timestamp)
}

// createPRRequest creates the pull request request.
func (p *Promoter) createPRRequest(req *PromotionRequest, cell, currentVersion string) *CreatePRRequest {
	title := fmt.Sprintf("Promote %s to %s/%s: %s", req.Package, req.TargetEnv, cell, req.Version)

	body := fmt.Sprintf(`## Package Promotion

**Package:** %s
**Version:** %s
**Environment:** %s
**Cell:** %s
`, req.Package, req.Version, req.TargetEnv, cell)

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
*Generated by DK CLI*
`

	labels := []string{
		"promotion",
		fmt.Sprintf("env:%s", req.TargetEnv),
		fmt.Sprintf("cell:%s", cell),
		fmt.Sprintf("package:%s", req.Package),
	}

	return &CreatePRRequest{
		Title:  title,
		Body:   body,
		Labels: labels,
	}
}
