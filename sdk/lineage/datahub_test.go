package lineage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Verify DataHubEmitter implements Emitter interface.
var _ Emitter = (*DataHubEmitter)(nil)

func TestDefaultDataHubConfig(t *testing.T) {
	config := DefaultDataHubConfig()

	if config.Namespace != "dk" {
		t.Errorf("expected namespace %q, got %q", "dk", config.Namespace)
	}
	if config.TimeoutSeconds != 30 {
		t.Errorf("expected timeout 30, got %d", config.TimeoutSeconds)
	}
	if config.Endpoint != "" {
		t.Errorf("expected empty endpoint, got %q", config.Endpoint)
	}
}

func TestNewDataHubEmitter_RequiresEndpoint(t *testing.T) {
	_, err := NewDataHubEmitter(DataHubConfig{})
	if err == nil {
		t.Fatal("expected error when endpoint is empty")
	}
}

func TestDataHubEmitter_Emit(t *testing.T) {
	var receivedPath string
	var receivedContentType string
	var receivedAuth string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedContentType = r.Header.Get("Content-Type")
		receivedAuth = r.Header.Get("Authorization")
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	emitter, err := NewDataHubEmitter(DataHubConfig{
		Endpoint:       server.URL,
		Namespace:      "test-ns",
		APIToken:       "test-token",
		TimeoutSeconds: 5,
	})
	if err != nil {
		t.Fatalf("failed to create emitter: %v", err)
	}

	event := NewEvent(EventTypeStart, "run-123", "test-ns", "test-job")

	err = emitter.Emit(context.Background(), event)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}

	// Verify URL path
	if receivedPath != "/openapi/v1/lineage" {
		t.Errorf("expected path %q, got %q", "/openapi/v1/lineage", receivedPath)
	}

	// Verify content type
	if receivedContentType != "application/json" {
		t.Errorf("expected content type %q, got %q", "application/json", receivedContentType)
	}

	// Verify auth header
	if receivedAuth != "Bearer test-token" {
		t.Errorf("expected auth %q, got %q", "Bearer test-token", receivedAuth)
	}

	// Verify body is valid JSON event
	var decoded Event
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if decoded.Job.Name != "test-job" {
		t.Errorf("expected job name %q, got %q", "test-job", decoded.Job.Name)
	}
	if decoded.Run.RunID != "run-123" {
		t.Errorf("expected run ID %q, got %q", "run-123", decoded.Run.RunID)
	}
}

func TestDataHubEmitter_EmitNilEvent(t *testing.T) {
	emitter, err := NewDataHubEmitter(DataHubConfig{
		Endpoint:  "http://localhost:8080",
		Namespace: "dk",
	})
	if err != nil {
		t.Fatalf("failed to create emitter: %v", err)
	}

	err = emitter.Emit(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil event")
	}
}

func TestDataHubEmitter_EmitSetsNamespace(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	emitter, err := NewDataHubEmitter(DataHubConfig{
		Endpoint:  server.URL,
		Namespace: "my-namespace",
	})
	if err != nil {
		t.Fatalf("failed to create emitter: %v", err)
	}

	// Create event with empty namespace
	event := NewEvent(EventTypeComplete, "run-456", "", "job-name")

	err = emitter.Emit(context.Background(), event)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if decoded.Job.Namespace != "my-namespace" {
		t.Errorf("expected namespace %q, got %q", "my-namespace", decoded.Job.Namespace)
	}
}

func TestDataHubEmitter_EmitNoAuth(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	emitter, err := NewDataHubEmitter(DataHubConfig{
		Endpoint:  server.URL,
		Namespace: "dk",
	})
	if err != nil {
		t.Fatalf("failed to create emitter: %v", err)
	}

	event := NewEvent(EventTypeStart, "run-789", "dk", "test-job")
	err = emitter.Emit(context.Background(), event)
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}

	if receivedAuth != "" {
		t.Errorf("expected no auth header, got %q", receivedAuth)
	}
}

func TestDataHubEmitter_EmitServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	emitter, err := NewDataHubEmitter(DataHubConfig{
		Endpoint:  server.URL,
		Namespace: "dk",
	})
	if err != nil {
		t.Fatalf("failed to create emitter: %v", err)
	}

	event := NewEvent(EventTypeStart, "run-err", "dk", "test-job")
	err = emitter.Emit(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for server 500")
	}
}

func TestDataHubEmitter_Close(t *testing.T) {
	emitter, err := NewDataHubEmitter(DataHubConfig{
		Endpoint:  "http://localhost:8080",
		Namespace: "dk",
	})
	if err != nil {
		t.Fatalf("failed to create emitter: %v", err)
	}

	if err := emitter.Close(); err != nil {
		t.Errorf("expected nil error from Close, got %v", err)
	}
}
