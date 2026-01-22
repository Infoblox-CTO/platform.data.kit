// Package promotion provides services for promoting data packages between environments.
package promotion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitHubClient implements PRClient for GitHub.
type GitHubClient struct {
	// Token is the GitHub API token.
	Token string
	// Owner is the repository owner.
	Owner string
	// Repo is the repository name.
	Repo string
	// BaseURL is the GitHub API base URL (for GitHub Enterprise).
	BaseURL string
	// HTTPClient is the HTTP client to use.
	HTTPClient *http.Client
}

// NewGitHubClient creates a new GitHubClient.
func NewGitHubClient(token, owner, repo string) *GitHubClient {
	return &GitHubClient{
		Token:   token,
		Owner:   owner,
		Repo:    repo,
		BaseURL: "https://api.github.com",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreatePR creates a new pull request on GitHub.
func (c *GitHubClient) CreatePR(ctx context.Context, req *CreatePRRequest) (*PRInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.BaseURL, c.Owner, c.Repo)

	body := map[string]interface{}{
		"title": req.Title,
		"body":  req.Body,
		"head":  req.Head,
		"base":  req.Base,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var pr ghPullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Add labels if specified
	if len(req.Labels) > 0 {
		if err := c.addLabels(ctx, pr.Number, req.Labels); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: failed to add labels: %v\n", err)
		}
	}

	return &PRInfo{
		Number: pr.Number,
		URL:    pr.HTMLURL,
		State:  PRStateOpen,
		Merged: false,
	}, nil
}

// GetPR returns information about a pull request.
func (c *GitHubClient) GetPR(ctx context.Context, number int) (*PRInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.BaseURL, c.Owner, c.Repo, number)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	var pr ghPullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	info := &PRInfo{
		Number: pr.Number,
		URL:    pr.HTMLURL,
		Merged: pr.Merged,
	}

	if pr.State == "closed" && pr.Merged {
		info.State = PRStateMerged
		info.MergedAt = pr.MergedAt
	} else if pr.State == "closed" {
		info.State = PRStateClosed
	} else {
		info.State = PRStateOpen
	}

	return info, nil
}

// EnableAutoMerge enables auto-merge on a pull request.
func (c *GitHubClient) EnableAutoMerge(ctx context.Context, number int) error {
	// GitHub's auto-merge requires GraphQL API
	// For MVP, we'll skip this and rely on PR reviews
	return nil
}

// addLabels adds labels to a pull request.
func (c *GitHubClient) addLabels(ctx context.Context, number int, labels []string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", c.BaseURL, c.Owner, c.Repo, number)

	body := map[string]interface{}{
		"labels": labels,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	return nil
}

// setHeaders sets common headers for GitHub API requests.
func (c *GitHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// ghPullRequest represents a GitHub pull request response.
type ghPullRequest struct {
	Number   int        `json:"number"`
	HTMLURL  string     `json:"html_url"`
	State    string     `json:"state"`
	Merged   bool       `json:"merged"`
	MergedAt *time.Time `json:"merged_at"`
}

// CreateBranch creates a new branch on GitHub.
func (c *GitHubClient) CreateBranch(ctx context.Context, baseBranch, newBranch string) error {
	// First get the SHA of the base branch
	sha, err := c.getBranchSHA(ctx, baseBranch)
	if err != nil {
		return fmt.Errorf("getting base branch SHA: %w", err)
	}

	// Create the new branch reference
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs", c.BaseURL, c.Owner, c.Repo)

	body := map[string]interface{}{
		"ref": fmt.Sprintf("refs/heads/%s", newBranch),
		"sha": sha,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getBranchSHA gets the SHA of a branch.
func (c *GitHubClient) getBranchSHA(ctx context.Context, branch string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", c.BaseURL, c.Owner, c.Repo, branch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API error: status %d", resp.StatusCode)
	}

	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ref); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return ref.Object.SHA, nil
}

// UpdateFile updates a file on GitHub via the API.
func (c *GitHubClient) UpdateFile(ctx context.Context, branch, path, content, message string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.BaseURL, c.Owner, c.Repo, path)

	// Get current file SHA if it exists
	sha, _ := c.getFileSHA(ctx, branch, path)

	body := map[string]interface{}{
		"message": message,
		"content": content, // Must be base64 encoded
		"branch":  branch,
	}
	if sha != "" {
		body["sha"] = sha
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getFileSHA gets the SHA of a file.
func (c *GitHubClient) getFileSHA(ctx context.Context, branch, path string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", c.BaseURL, c.Owner, c.Repo, path, branch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file not found")
	}

	var content struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return "", err
	}

	return content.SHA, nil
}
