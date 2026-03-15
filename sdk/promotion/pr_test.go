package promotion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPromoter_PromoteDefaultCell(t *testing.T) {
	// Mock GitHub API
	mux := http.NewServeMux()
	var updatedPath string

	mux.HandleFunc("GET /repos/test-owner/test-repo/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": map[string]string{"sha": "abc123"},
		})
	})
	mux.HandleFunc("POST /repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"ref": "refs/heads/test"})
	})
	mux.HandleFunc("GET /repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("PUT /repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		updatedPath = r.URL.Path
		// Verify content has appVersion
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if content, ok := body["content"].(string); ok {
			decoded, _ := base64.StdEncoding.DecodeString(content)
			if !strings.Contains(string(decoded), "appVersion") {
				t.Errorf("file content missing appVersion: %s", decoded)
			}
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"content": map[string]string{"sha": "newsha"}})
	})
	mux.HandleFunc("POST /repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":   42,
			"html_url": "https://github.com/test-owner/test-repo/pull/42",
		})
	})
	mux.HandleFunc("POST /repos/test-owner/test-repo/issues/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	ghClient := &GitHubClient{
		Token:      "test-token",
		Owner:      "test-owner",
		Repo:       "test-repo",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	promoter := &Promoter{
		GitHubClient:    ghClient,
		BaseBranch:      "main",
		RecordGenerator: NewRecordGenerator().WithInitiator("test-user"),
	}

	result, err := promoter.Promote(context.Background(), &PromotionRequest{
		Package:   "my-pkg",
		Version:   "v1.0.0",
		TargetEnv: EnvDev,
		// Cell empty → defaults to c0
	})
	if err != nil {
		t.Fatalf("Promote() error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if result.PRNumber != 42 {
		t.Errorf("PRNumber = %d, want 42", result.PRNumber)
	}
	// Verify path uses envs/{env}/cells/c0/apps/{pkg}
	if !strings.Contains(updatedPath, "envs/dev/cells/c0/apps/my-pkg") {
		t.Errorf("updated path = %q, expected to contain envs/dev/cells/c0/apps/my-pkg", updatedPath)
	}
	// Verify branch contains env and cell
	if !strings.Contains(result.Branch, "promote/my-pkg/dev/c0/v1.0.0") {
		t.Errorf("Branch = %q, expected to contain promote/my-pkg/dev/c0/v1.0.0", result.Branch)
	}
}

func TestPromoter_PromoteNamedCell(t *testing.T) {
	var updatedPath string
	mux := http.NewServeMux()

	mux.HandleFunc("GET /repos/test-owner/test-repo/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": map[string]string{"sha": "abc123"},
		})
	})
	mux.HandleFunc("POST /repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"ref": "refs/heads/test"})
	})
	mux.HandleFunc("GET /repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("PUT /repos/test-owner/test-repo/contents/", func(w http.ResponseWriter, r *http.Request) {
		updatedPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"content": map[string]string{"sha": "newsha"}})
	})
	mux.HandleFunc("POST /repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":   43,
			"html_url": "https://github.com/test-owner/test-repo/pull/43",
		})
	})
	mux.HandleFunc("POST /repos/test-owner/test-repo/issues/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	ghClient := &GitHubClient{
		Token:      "test-token",
		Owner:      "test-owner",
		Repo:       "test-repo",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	promoter := &Promoter{
		GitHubClient:    ghClient,
		BaseBranch:      "main",
		RecordGenerator: NewRecordGenerator(),
	}

	result, err := promoter.Promote(context.Background(), &PromotionRequest{
		Package:   "my-pkg",
		Version:   "v1.0.0",
		TargetEnv: EnvProd,
		Cell:      "canary",
	})
	if err != nil {
		t.Fatalf("Promote() error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	// Verify path uses envs/prod/cells/canary/apps/my-pkg
	if !strings.Contains(updatedPath, "envs/prod/cells/canary/apps/my-pkg") {
		t.Errorf("updated path = %q, expected to contain envs/prod/cells/canary/apps/my-pkg", updatedPath)
	}
	if !strings.Contains(result.Branch, "promote/my-pkg/prod/canary/v1.0.0") {
		t.Errorf("Branch = %q, expected to contain promote/my-pkg/prod/canary/v1.0.0", result.Branch)
	}
}

func TestPromoter_DryRun(t *testing.T) {
	promoter := &Promoter{
		RecordGenerator: NewRecordGenerator(),
	}

	result, err := promoter.Promote(context.Background(), &PromotionRequest{
		Package:   "my-pkg",
		Version:   "v1.0.0",
		TargetEnv: EnvDev,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("Promote() dry-run error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}
	if !result.DryRun {
		t.Error("expected DryRun = true")
	}
	if result.Record == nil {
		t.Error("expected Record to be set")
	}
}

func TestPromoter_GetCurrentVersion(t *testing.T) {
	mux := http.NewServeMux()

	valuesContent := "appVersion: v0.9.0\nreplicas: 3\n"
	encodedContent := base64.StdEncoding.EncodeToString([]byte(valuesContent))

	mux.HandleFunc("GET /repos/test-owner/test-repo/contents/envs/dev/cells/c0/apps/my-pkg/values.yaml", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content":  encodedContent,
			"encoding": "base64",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	ghClient := &GitHubClient{
		Token:      "test-token",
		Owner:      "test-owner",
		Repo:       "test-repo",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	promoter := &Promoter{
		GitHubClient: ghClient,
		BaseBranch:   "main",
	}

	version, err := promoter.GetCurrentVersion(context.Background(), EnvDev, "c0", "my-pkg")
	if err != nil {
		t.Fatalf("GetCurrentVersion() error: %v", err)
	}
	if version != "v0.9.0" {
		t.Errorf("version = %q, want %q", version, "v0.9.0")
	}
}

func TestPromoter_GetCurrentVersion_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	ghClient := &GitHubClient{
		Token:      "test-token",
		Owner:      "test-owner",
		Repo:       "test-repo",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	promoter := &Promoter{
		GitHubClient: ghClient,
		BaseBranch:   "main",
	}

	version, err := promoter.GetCurrentVersion(context.Background(), EnvDev, "c0", "my-pkg")
	if err != nil {
		t.Fatalf("GetCurrentVersion() error: %v", err)
	}
	if version != "" {
		t.Errorf("version = %q, want empty", version)
	}
}

func TestPromoter_ValidateRequest(t *testing.T) {
	p := &Promoter{}

	tests := []struct {
		name    string
		req     *PromotionRequest
		wantErr bool
	}{
		{"valid default cell", &PromotionRequest{Package: "pkg", Version: "v1", TargetEnv: EnvDev}, false},
		{"valid named cell", &PromotionRequest{Package: "pkg", Version: "v1", TargetEnv: EnvProd, Cell: "canary"}, false},
		{"missing package", &PromotionRequest{Version: "v1", TargetEnv: EnvDev}, true},
		{"missing version", &PromotionRequest{Package: "pkg", TargetEnv: EnvDev}, true},
		{"missing env", &PromotionRequest{Package: "pkg", Version: "v1"}, true},
		{"invalid env", &PromotionRequest{Package: "pkg", Version: "v1", TargetEnv: Environment("bad")}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.validateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
