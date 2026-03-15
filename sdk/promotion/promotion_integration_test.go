//go:build integration

package promotion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	giteaImage    = "gitea/gitea:latest"
	giteaUser     = "dktest"
	giteaPassword = "dktest1234"
	giteaEmail    = "dktest@test.local"
)

func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("skipping: docker not available")
	}
}

func TestIntegration_PromoteSingleCell(t *testing.T) {
	skipIfNoDocker(t)

	gi := startGitea(t, "test-org", "test-repo")
	defer stopGitea(t, gi.containerID)

	// Seed initial cell layout
	seedFile(t, gi, "cells/us-dev-1/apps/.gitkeep", "")

	// Promote
	ghClient := &GitHubClient{
		Token:      gi.token,
		Owner:      "test-org",
		Repo:       "test-repo",
		BaseURL:    gi.url + "/api/v1",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	promoter := &Promoter{
		GitHubClient:    ghClient,
		BaseBranch:      "main",
		RecordGenerator: NewRecordGenerator().WithInitiator("integration-test"),
	}

	result, err := promoter.Promote(context.Background(), &PromotionRequest{
		Package: "my-pkg",
		Version: "v1.0.0",
		Cell:    "us-dev-1",
	})
	if err != nil {
		t.Fatalf("Promote() error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if result.PRNumber == 0 {
		t.Error("expected non-zero PR number")
	}

	// Verify the PR was created and file content is correct
	content := readFileFromBranch(t, gi, result.Branch, "cells/us-dev-1/apps/my-pkg/values.yaml")
	version := ParseAppVersion(content)
	if version != "v1.0.0" {
		t.Errorf("appVersion = %q, want v1.0.0", version)
	}
}

func TestIntegration_PromotePreserveOverrides(t *testing.T) {
	skipIfNoDocker(t)

	gi := startGitea(t, "test-org2", "test-repo2")
	defer stopGitea(t, gi.containerID)

	// Seed with existing values that have overrides
	existingValues := "appVersion: v0.9.0\nreplicas: 5\nresources:\n  cpu: 500m\n"
	seedFile(t, gi, "cells/us-dev-1/apps/my-pkg/values.yaml", existingValues)

	ghClient := &GitHubClient{
		Token:      gi.token,
		Owner:      "test-org2",
		Repo:       "test-repo2",
		BaseURL:    gi.url + "/api/v1",
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}

	promoter := &Promoter{
		GitHubClient:    ghClient,
		BaseBranch:      "main",
		RecordGenerator: NewRecordGenerator(),
	}

	result, err := promoter.Promote(context.Background(), &PromotionRequest{
		Package: "my-pkg",
		Version: "v1.0.0",
		Cell:    "us-dev-1",
	})
	if err != nil {
		t.Fatalf("Promote() error: %v", err)
	}

	content := readFileFromBranch(t, gi, result.Branch, "cells/us-dev-1/apps/my-pkg/values.yaml")

	// Verify version updated
	version := ParseAppVersion(content)
	if version != "v1.0.0" {
		t.Errorf("appVersion = %q, want v1.0.0", version)
	}

	// Verify overrides preserved
	if !strings.Contains(string(content), "replicas") {
		t.Error("replicas override was lost")
	}
}

// --- Gitea test helpers ---

type giteaTestInstance struct {
	url         string
	token       string
	containerID string
}

func startGitea(t *testing.T, org, repo string) *giteaTestInstance {
	t.Helper()

	out, err := exec.Command("docker", "run", "-d",
		"-p", "0:3000",
		"-e", "GITEA__security__INSTALL_LOCK=true",
		giteaImage,
	).Output()
	if err != nil {
		t.Fatalf("failed to start Gitea: %v", err)
	}
	containerID := strings.TrimSpace(string(out))

	portOut, _ := exec.Command("docker", "port", containerID, "3000").Output()
	parts := strings.Split(strings.TrimSpace(string(portOut)), ":")
	port := parts[len(parts)-1]
	url := fmt.Sprintf("http://localhost:%s", port)

	// Wait for ready
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/api/v1/version")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Create admin user
	exec.Command("docker", "exec", containerID,
		"gitea", "admin", "user", "create",
		"--username", giteaUser, "--password", giteaPassword,
		"--email", giteaEmail, "--admin",
	).Run()

	// Create token
	token := createToken(t, url)

	// Create org + repo
	apiPost(t, url, token, "/api/v1/orgs", map[string]interface{}{"username": org, "visibility": "public"})
	apiPost(t, url, token, fmt.Sprintf("/api/v1/orgs/%s/repos", org),
		map[string]interface{}{"name": repo, "auto_init": true, "default_branch": "main"})

	return &giteaTestInstance{url: url, token: token, containerID: containerID}
}

func stopGitea(t *testing.T, containerID string) {
	exec.Command("docker", "rm", "-f", containerID).Run()
}

func createToken(t *testing.T, url string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{"name": "test", "scopes": []string{"all"}})
	req, _ := http.NewRequest("POST", url+"/api/v1/users/"+giteaUser+"/tokens", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(giteaUser, giteaPassword)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	defer resp.Body.Close()
	var result struct {
		SHA1 string `json:"sha1"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA1
}

func apiPost(t *testing.T, baseURL, token, path string, body map[string]interface{}) {
	t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", baseURL+path, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("API POST %s: %v", path, err)
	}
	resp.Body.Close()
}

func seedFile(t *testing.T, gi *giteaTestInstance, path, content string) {
	t.Helper()
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	body := map[string]interface{}{
		"content": encoded,
		"message": "seed: " + path,
	}
	data, _ := json.Marshal(body)

	owner := "test-org"
	if strings.Contains(gi.url, "test-org2") {
		owner = "test-org2"
	}
	repo := "test-repo"
	if strings.Contains(gi.url, "test-repo2") {
		repo = "test-repo2"
	}

	req, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/repos/%s/%s/contents/%s", gi.url, owner, repo, path),
		strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+gi.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("seed file %s: %v", path, err)
	}
	resp.Body.Close()
}

func readFileFromBranch(t *testing.T, gi *giteaTestInstance, branch, path string) []byte {
	t.Helper()

	owner := "test-org"
	repo := "test-repo"

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/contents/%s?ref=%s", gi.url, owner, repo, path, branch)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+gi.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("read file %s: status %d: %s", path, resp.StatusCode, body)
	}

	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	decoded, _ := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
	return decoded
}
