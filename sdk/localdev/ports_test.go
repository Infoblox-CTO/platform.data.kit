package localdev

import (
	"testing"
	"time"
)

func TestDefaultPorts(t *testing.T) {
	// DefaultPorts is now derived from charts.DefaultCharts.
	// Verify key services are present and have expected primary ports.
	expectedPorts := map[string]int{
		"redpanda":   19092,
		"localstack": 4566,
		"postgres":   5432,
		"marquez":    5000,
	}

	for service, expected := range expectedPorts {
		if port, ok := DefaultPorts[service]; !ok {
			t.Errorf("DefaultPorts missing service %q", service)
		} else if port != expected {
			t.Errorf("DefaultPorts[%q] = %d, want %d", service, port, expected)
		}
	}

	// Verify count matches number of charts
	if len(DefaultPorts) != 4 {
		t.Errorf("DefaultPorts has %d entries, want 4", len(DefaultPorts))
	}
}

func TestNewPortChecker(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "valid timeout",
			timeout:         5 * time.Second,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "zero timeout defaults to 1 second",
			timeout:         0,
			expectedTimeout: 1 * time.Second,
		},
		{
			name:            "negative timeout defaults to 1 second",
			timeout:         -1 * time.Second,
			expectedTimeout: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewPortChecker(tt.timeout)
			if checker == nil {
				t.Fatal("NewPortChecker returned nil")
			}
			if checker.timeout != tt.expectedTimeout {
				t.Errorf("timeout = %v, want %v", checker.timeout, tt.expectedTimeout)
			}
		})
	}
}

func TestPortChecker_CheckPort(t *testing.T) {
	checker := NewPortChecker(1 * time.Second)

	// Test a port that should be available (high port number)
	result := checker.CheckPort(59999)
	if !result.Available {
		t.Skip("Port 59999 is in use, skipping test")
	}
	if result.Port != 59999 {
		t.Errorf("Port = %d, want 59999", result.Port)
	}
}

func TestPortChecker_CheckDefaultPorts(t *testing.T) {
	checker := NewPortChecker(1 * time.Second)
	results := checker.CheckDefaultPorts()

	// Should return results for all default ports
	if len(results) != len(DefaultPorts) {
		t.Errorf("CheckDefaultPorts returned %d results, want %d", len(results), len(DefaultPorts))
	}

	// Each result should have a port from DefaultPorts
	for _, r := range results {
		found := false
		for _, port := range DefaultPorts {
			if r.Port == port {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected port %d in results", r.Port)
		}
	}
}

func TestPortChecker_CheckPorts(t *testing.T) {
	checker := NewPortChecker(1 * time.Second)
	ports := []int{8080, 8081, 8082}

	results := checker.CheckPorts(ports)

	if len(results) != len(ports) {
		t.Errorf("CheckPorts returned %d results, want %d", len(results), len(ports))
	}

	for i, r := range results {
		if r.Port != ports[i] {
			t.Errorf("result[%d].Port = %d, want %d", i, r.Port, ports[i])
		}
	}
}

func TestPortCheckResult_Struct(t *testing.T) {
	result := PortCheckResult{
		Port:      19092,
		Available: true,
		Service:   "redpanda",
		Error:     "",
	}

	if result.Port != 19092 {
		t.Errorf("Port = %d, want 19092", result.Port)
	}
	if !result.Available {
		t.Error("Available = false, want true")
	}
	if result.Service != "redpanda" {
		t.Errorf("Service = %q, want 'redpanda'", result.Service)
	}
}

func TestPortChecker_CheckAllAvailable_NoPortsInUse(t *testing.T) {
	checker := NewPortChecker(1 * time.Second)
	// Use high ports that are unlikely to be in use
	ports := []int{59997, 59998, 59999}

	err := checker.CheckAllAvailable(ports)
	if err != nil {
		t.Skip("Some ports are in use, skipping test")
	}
}
