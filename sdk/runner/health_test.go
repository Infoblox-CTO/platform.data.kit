package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestHealthChecker_CheckHTTP(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus HealthStatus
		wantErr    bool
	}{
		{
			name: "healthy 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			},
			wantStatus: HealthStatusHealthy,
			wantErr:    false,
		},
		{
			name: "healthy 201",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
			wantStatus: HealthStatusHealthy,
			wantErr:    false,
		},
		{
			name: "healthy 302",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusFound)
			},
			wantStatus: HealthStatusHealthy,
			wantErr:    false,
		},
		{
			name: "unhealthy 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: HealthStatusUnhealthy,
			wantErr:    true,
		},
		{
			name: "unhealthy 503",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantStatus: HealthStatusUnhealthy,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			// Extract port from server URL
			probe := &contracts.Probe{
				HTTPGet: &contracts.HTTPGetAction{
					Path:   "/",
					Port:   getPort(server.URL),
					Scheme: "HTTP",
				},
				TimeoutSeconds: 5,
			}

			checker := NewHealthChecker("test-container", probe)
			// Override the URL to use test server
			checker.httpClient = server.Client()

			status, err := checker.checkHTTPWithURL(context.Background(), server.URL+"/")
			if status != tt.wantStatus {
				t.Errorf("Check() status = %v, want %v", status, tt.wantStatus)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHealthChecker_Check_NoProbe(t *testing.T) {
	checker := NewHealthChecker("test-container", nil)
	status, err := checker.Check(context.Background())

	if status != HealthStatusUnknown {
		t.Errorf("Check() status = %v, want %v", status, HealthStatusUnknown)
	}
	if err == nil {
		t.Error("Check() expected error for nil probe")
	}
}

func TestHealthChecker_Check_NoAction(t *testing.T) {
	probe := &contracts.Probe{
		PeriodSeconds: 10,
	}

	checker := NewHealthChecker("test-container", probe)
	status, err := checker.Check(context.Background())

	if status != HealthStatusUnknown {
		t.Errorf("Check() status = %v, want %v", status, HealthStatusUnknown)
	}
	if err == nil {
		t.Error("Check() expected error for no action configured")
	}
}

func TestHealthChecker_WaitForHealthy_NoProbe(t *testing.T) {
	checker := NewHealthChecker("test-container", nil)
	err := checker.WaitForHealthy(context.Background())

	// No probe means assume healthy immediately
	if err != nil {
		t.Errorf("WaitForHealthy() error = %v, want nil for no probe", err)
	}
}

func TestHealthChecker_WaitForHealthy_ContextCancelled(t *testing.T) {
	probe := &contracts.Probe{
		HTTPGet: &contracts.HTTPGetAction{
			Path: "/healthz",
			Port: 9999, // Non-existent port
		},
		InitialDelaySeconds: 0,
		PeriodSeconds:       1,
		TimeoutSeconds:      1,
	}

	checker := NewHealthChecker("test-container", probe)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := checker.WaitForHealthy(ctx)
	if err == nil {
		t.Error("WaitForHealthy() expected error for cancelled context")
	}
}

func TestNewHealthChecker(t *testing.T) {
	probe := &contracts.Probe{
		HTTPGet: &contracts.HTTPGetAction{
			Path: "/healthz",
			Port: 8080,
		},
		TimeoutSeconds: 5,
	}

	checker := NewHealthChecker("my-container", probe)

	if checker == nil {
		t.Fatal("NewHealthChecker() returned nil")
	}
	if checker.containerID != "my-container" {
		t.Errorf("containerID = %v, want my-container", checker.containerID)
	}
	if checker.probe != probe {
		t.Error("probe not set correctly")
	}
	if checker.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestHealthPoller_New(t *testing.T) {
	probe := &contracts.Probe{
		HTTPGet: &contracts.HTTPGetAction{
			Path: "/healthz",
			Port: 8080,
		},
		PeriodSeconds: 5,
	}

	checker := NewHealthChecker("test-container", probe)
	onChange := func(status HealthStatus, err error) {
		// Callback for status changes
	}

	poller := NewHealthPoller(checker, onChange)

	if poller == nil {
		t.Fatal("NewHealthPoller() returned nil")
	}
	if poller.checker != checker {
		t.Error("checker not set correctly")
	}
	if poller.lastStatus != HealthStatusUnknown {
		t.Errorf("lastStatus = %v, want %v", poller.lastStatus, HealthStatusUnknown)
	}
}

// Helper to get port from URL
func getPort(url string) int {
	// Simple extraction - in real tests we'd parse properly
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == ':' {
			port := 0
			for j := i + 1; j < len(url); j++ {
				if url[j] >= '0' && url[j] <= '9' {
					port = port*10 + int(url[j]-'0')
				}
			}
			return port
		}
	}
	return 0
}
