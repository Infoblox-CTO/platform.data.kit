package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExtractSubdomain(t *testing.T) {
	tests := []struct {
		host string
		want string
	}{
		{"marquez.localtest.me:54321", "marquez"},
		{"marquez-api.localtest.me:54321", "marquez-api"},
		{"s3.localtest.me:54321", "s3"},
		{"redpanda.localtest.me:54321", "redpanda"},
		{"localtest.me:54321", ""},
		{"localhost:54321", ""},
		{"127.0.0.1:54321", ""},
		{"MARQUEZ.LOCALTEST.ME:54321", "marquez"},
		{"marquez.localtest.me", "marquez"},
		{"localtest.me", ""},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := extractSubdomain(tt.host, "localtest.me")
			if got != tt.want {
				t.Errorf("extractSubdomain(%q) = %q, want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	services := []ServiceProxy{
		{Subdomain: "marquez", Label: "Marquez Web", TargetURL: "http://localhost:3000", Description: "Lineage UI"},
		{Label: "Kafka", TargetURL: "localhost:19092", Description: "Message broker"},
	}

	s, err := New(services)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer s.Shutdown(context.Background())

	if s.Port() == 0 {
		t.Error("expected non-zero port")
	}

	if !strings.HasPrefix(s.URL(), "http://localtest.me:") {
		t.Errorf("unexpected URL: %s", s.URL())
	}
}

func TestNew_InvalidTargetURL(t *testing.T) {
	services := []ServiceProxy{
		{Subdomain: "bad", Label: "Bad", TargetURL: "://invalid", Description: "bad"},
	}

	_, err := New(services)
	if err == nil {
		t.Fatal("expected error for invalid target URL")
	}
}

func TestServer_Dashboard(t *testing.T) {
	// Start a mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	services := []ServiceProxy{
		{Subdomain: "test", Label: "Test Service", TargetURL: backend.URL, Description: "A test"},
		{Label: "TCP Service", TargetURL: "localhost:5432", Description: "Database"},
	}

	s, err := New(services)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	go s.Start()
	defer s.Shutdown(context.Background())

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Request dashboard (bare localtest.me)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", s.Port()))
	if err != nil {
		t.Fatalf("dashboard request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "DataKit Dev Dashboard") {
		t.Error("dashboard HTML missing title")
	}
	if !strings.Contains(html, "Test Service") {
		t.Error("dashboard HTML missing service card")
	}
	if !strings.Contains(html, "TCP Service") {
		t.Error("dashboard HTML missing TCP service")
	}
}

func TestServer_ReverseProxy(t *testing.T) {
	// Start a mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "reached")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("backend response"))
	}))
	defer backend.Close()

	services := []ServiceProxy{
		{Subdomain: "test", Label: "Test", TargetURL: backend.URL, Description: "Test service"},
	}

	s, err := New(services)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	go s.Start()
	defer s.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Request via subdomain Host header
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/some/path", s.Port()), nil)
	req.Host = fmt.Sprintf("test.localtest.me:%d", s.Port())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Backend") != "reached" {
		t.Error("request did not reach backend")
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "backend response" {
		t.Errorf("expected backend response, got %q", string(body))
	}
}

func TestServer_UnknownSubdomain(t *testing.T) {
	services := []ServiceProxy{
		{Subdomain: "known", Label: "Known", TargetURL: "http://localhost:9999", Description: "Known"},
	}

	s, err := New(services)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	go s.Start()
	defer s.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	// Request with unknown subdomain → should serve dashboard
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", s.Port()), nil)
	req.Host = fmt.Sprintf("unknown.localtest.me:%d", s.Port())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "DataKit Dev Dashboard") {
		t.Error("expected dashboard for unknown subdomain")
	}
}

func TestServer_StatusAPI(t *testing.T) {
	// Start a healthy backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	services := []ServiceProxy{
		{Subdomain: "test", Label: "Test", TargetURL: backend.URL, Description: "Healthy service"},
		{Label: "TCP", TargetURL: "localhost:99999", Description: "Not HTTP"},
	}

	s, err := New(services)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	go s.Start()
	defer s.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/_api/status", s.Port()))
	if err != nil {
		t.Fatalf("status API request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected JSON content type, got %q", resp.Header.Get("Content-Type"))
	}

	var statuses []struct {
		Label    string `json:"label"`
		Healthy  bool   `json:"healthy"`
		ProxyURL string `json:"proxyUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		t.Fatalf("failed to decode status: %v", err)
	}

	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}

	// First service (HTTP backend) should be healthy
	if !statuses[0].Healthy {
		t.Error("expected first service to be healthy")
	}
	if statuses[0].ProxyURL == "" {
		t.Error("expected proxyURL for first service")
	}

	// Second service (TCP) should not be healthy (not HTTP)
	if statuses[1].Healthy {
		t.Error("expected TCP service to not be healthy")
	}
	if statuses[1].ProxyURL != "" {
		t.Error("expected empty proxyURL for TCP service")
	}
}

func TestCheckHealth(t *testing.T) {
	// Healthy server
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	if !checkHealth(healthy.URL) {
		t.Error("expected healthy server to be healthy")
	}

	// Non-HTTP URL
	if checkHealth("localhost:5432") {
		t.Error("expected non-HTTP URL to not be healthy")
	}

	// Unreachable URL
	if checkHealth("http://localhost:1") {
		t.Error("expected unreachable URL to not be healthy")
	}

	// Empty URL
	if checkHealth("") {
		t.Error("expected empty URL to not be healthy")
	}
}

func TestRenderDashboardHTML(t *testing.T) {
	services := []ServiceProxy{
		{Subdomain: "marquez", Label: "Marquez Web", TargetURL: "http://localhost:3000", Description: "Lineage UI"},
		{Label: "PostgreSQL", TargetURL: "localhost:5432", Description: "Database"},
	}

	html := renderDashboardHTML(services, 54321, "http")

	// Check basic structure
	if !strings.Contains(html, "DataKit Dev Dashboard") {
		t.Error("missing title")
	}
	if !strings.Contains(html, "Marquez Web") {
		t.Error("missing Marquez card")
	}
	if !strings.Contains(html, "PostgreSQL") {
		t.Error("missing PostgreSQL card")
	}
	if !strings.Contains(html, "marquez.localtest.me:54321") {
		t.Error("missing proxy URL for Marquez")
	}
	if !strings.Contains(html, "localhost:5432") {
		t.Error("missing connection string for PostgreSQL")
	}
	if !strings.Contains(html, "/_api/status") {
		t.Error("missing status API reference in JS")
	}
}
