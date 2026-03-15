package e2e

import (
	"bytes"
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

// giteaInstance holds connection details for a running Gitea container.
type giteaInstance struct {
	URL           string
	Token         string
	ContainerID   string
	Org           string
	Repo          string
}

// startGitea starts a Gitea container, creates an admin user, org, repo, and API token.
// Returns a giteaInstance ready for use in promotion tests.
func startGitea(t *testing.T, org, repo string) *giteaInstance {
	t.Helper()
	skipIfNoDocker(t)

	// Start container
	out, err := exec.Command("docker", "run", "-d",
		"-p", "0:3000",
		"-e", "GITEA__security__INSTALL_LOCK=true",
		"-e", "GITEA__server__ROOT_URL=http://localhost:3000",
		giteaImage,
	).Output()
	if err != nil {
		t.Fatalf("failed to start Gitea: %v", err)
	}
	containerID := strings.TrimSpace(string(out))

	// Get mapped port
	portOut, err := exec.Command("docker", "port", containerID, "3000").Output()
	if err != nil {
		exec.Command("docker", "rm", "-f", containerID).Run()
		t.Fatalf("failed to get Gitea port: %v", err)
	}
	portLine := strings.TrimSpace(string(portOut))
	// Format: 0.0.0.0:XXXXX or [::]:XXXXX
	parts := strings.Split(portLine, ":")
	port := parts[len(parts)-1]
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	gi := &giteaInstance{
		URL:         baseURL,
		ContainerID: containerID,
		Org:         org,
		Repo:        repo,
	}

	// Wait for Gitea to be healthy
	waitForGitea(t, gi)

	// Create admin user
	createGiteaUser(t, containerID)

	// Create API token
	gi.Token = createGiteaToken(t, gi)

	// Create org and repo
	createGiteaOrg(t, gi)
	createGiteaRepo(t, gi)

	t.Cleanup(func() {
		stopGitea(t, containerID)
	})

	return gi
}

// waitForGitea polls the Gitea health endpoint until it's ready.
func waitForGitea(t *testing.T, gi *giteaInstance) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(gi.URL + "/api/v1/version")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("Gitea did not become healthy in time")
}

// createGiteaUser creates the admin user via gitea CLI inside the container.
func createGiteaUser(t *testing.T, containerID string) {
	t.Helper()
	cmd := exec.Command("docker", "exec", containerID,
		"gitea", "admin", "user", "create",
		"--username", giteaUser,
		"--password", giteaPassword,
		"--email", giteaEmail,
		"--admin",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create Gitea user: %v\n%s", err, out)
	}
}

// createGiteaToken creates an API token for the admin user.
func createGiteaToken(t *testing.T, gi *giteaInstance) string {
	t.Helper()
	body := map[string]interface{}{
		"name":   "test-token",
		"scopes": []string{"all"},
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", gi.URL+"/api/v1/users/"+giteaUser+"/tokens", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(giteaUser, giteaPassword)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create token (status %d): %s", resp.StatusCode, respBody)
	}

	var result struct {
		SHA1 string `json:"sha1"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA1
}

// createGiteaOrg creates an organization in Gitea.
func createGiteaOrg(t *testing.T, gi *giteaInstance) {
	t.Helper()
	body := map[string]interface{}{
		"username":   gi.Org,
		"visibility": "public",
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", gi.URL+"/api/v1/orgs", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+gi.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to create org: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create org (status %d): %s", resp.StatusCode, respBody)
	}
}

// createGiteaRepo creates a repository in the organization.
func createGiteaRepo(t *testing.T, gi *giteaInstance) {
	t.Helper()
	body := map[string]interface{}{
		"name":          gi.Repo,
		"auto_init":     true,
		"default_branch": "main",
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", gi.URL+"/api/v1/orgs/"+gi.Org+"/repos", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+gi.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to create repo (status %d): %s", resp.StatusCode, respBody)
	}
}

// seedGiteaRepo pushes initial files (cell layout + shared chart) to the Gitea repo.
func seedGiteaRepo(t *testing.T, gi *giteaInstance, files map[string]string) {
	t.Helper()
	for path, content := range files {
		body := map[string]interface{}{
			"content": content,
			"message": "seed: " + path,
		}
		data, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST",
			fmt.Sprintf("%s/api/v1/repos/%s/%s/contents/%s", gi.URL, gi.Org, gi.Repo, path),
			bytes.NewReader(data),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "token "+gi.Token)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to seed file %s: %v", path, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("failed to seed file %s (status %d)", path, resp.StatusCode)
		}
	}
}

// stopGitea stops and removes the Gitea container.
func stopGitea(t *testing.T, containerID string) {
	t.Helper()
	exec.Command("docker", "rm", "-f", containerID).Run()
}
