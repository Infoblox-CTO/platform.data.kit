package localdev

import (
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev/charts"
)

func TestNewPortForwarder(t *testing.T) {
	pf := NewPortForwarder("k3d-dk-local", "dk-local")

	if pf == nil {
		t.Fatal("NewPortForwarder() returned nil")
	}

	if pf.kubeContext != "k3d-dk-local" {
		t.Errorf("kubeContext = %q, want %q", pf.kubeContext, "k3d-dk-local")
	}

	if pf.namespace != "dk-local" {
		t.Errorf("namespace = %q, want %q", pf.namespace, "dk-local")
	}

	if len(pf.forwards) != 0 {
		t.Errorf("forwards should be empty, got %d", len(pf.forwards))
	}
}

func TestPortForwarder_AddForward(t *testing.T) {
	pf := NewPortForwarder("k3d-dk-local", "dk-local")

	pf.AddForward("redpanda", 19092, 9092)
	pf.AddForward("localstack", 4566, 4566)
	pf.AddForward("postgres", 5432, 5432)

	if len(pf.forwards) != 3 {
		t.Errorf("forwards count = %d, want 3", len(pf.forwards))
	}

	expected := []PortForward{
		{ServiceName: "redpanda", LocalPort: 19092, RemotePort: 9092},
		{ServiceName: "localstack", LocalPort: 4566, RemotePort: 4566},
		{ServiceName: "postgres", LocalPort: 5432, RemotePort: 5432},
	}

	for i, fwd := range pf.forwards {
		if fwd.ServiceName != expected[i].ServiceName {
			t.Errorf("forward[%d].ServiceName = %q, want %q", i, fwd.ServiceName, expected[i].ServiceName)
		}
		if fwd.LocalPort != expected[i].LocalPort {
			t.Errorf("forward[%d].LocalPort = %d, want %d", i, fwd.LocalPort, expected[i].LocalPort)
		}
		if fwd.RemotePort != expected[i].RemotePort {
			t.Errorf("forward[%d].RemotePort = %d, want %d", i, fwd.RemotePort, expected[i].RemotePort)
		}
	}
}

func TestPortForwarder_Status_NoForwards(t *testing.T) {
	pf := NewPortForwarder("k3d-dk-local", "dk-local")

	statuses := pf.Status()

	if len(statuses) != 0 {
		t.Errorf("Status() should return empty slice, got %d items", len(statuses))
	}
}

func TestPortForwarder_Status_WithForwards(t *testing.T) {
	pf := NewPortForwarder("k3d-dk-local", "dk-local")
	pf.AddForward("redpanda", 19092, 9092)
	pf.AddForward("localstack", 4566, 4566)

	statuses := pf.Status()

	if len(statuses) != 2 {
		t.Errorf("Status() returned %d items, want 2", len(statuses))
	}

	// Before Start, processes are empty so Active should be false
	for _, s := range statuses {
		if s.Active {
			t.Errorf("status for %s should not be active before Start()", s.TargetService)
		}
	}
}

// TestPortForwarder_AddForwardsFromCharts tests that port forwards are
// correctly derived from ChartDefinition slices.
func TestPortForwarder_AddForwardsFromCharts(t *testing.T) {
	pf := NewPortForwarder("k3d-dk-local", "dk-local")
	pf.AddForwardsFromCharts(charts.DefaultCharts)

	// Count total expected port forwards from all default charts
	expectedTotal := 0
	for _, def := range charts.DefaultCharts {
		expectedTotal += len(def.PortForwards)
	}

	if len(pf.forwards) != expectedTotal {
		t.Errorf("forwards count = %d, want %d", len(pf.forwards), expectedTotal)
	}

	// Verify first forward matches first chart's first port forward
	if len(pf.forwards) > 0 {
		first := pf.forwards[0]
		expected := charts.DefaultCharts[0].PortForwards[0]
		if first.ServiceName != expected.ServiceName {
			t.Errorf("first forward ServiceName = %q, want %q", first.ServiceName, expected.ServiceName)
		}
		if first.LocalPort != expected.LocalPort {
			t.Errorf("first forward LocalPort = %d, want %d", first.LocalPort, expected.LocalPort)
		}
		if first.RemotePort != expected.RemotePort {
			t.Errorf("first forward RemotePort = %d, want %d", first.RemotePort, expected.RemotePort)
		}
	}
}

func TestPortForward_Struct(t *testing.T) {
	fwd := PortForward{
		ServiceName: "redpanda",
		LocalPort:   19092,
		RemotePort:  9092,
	}

	if fwd.ServiceName != "redpanda" {
		t.Errorf("ServiceName = %q, want 'redpanda'", fwd.ServiceName)
	}
	if fwd.LocalPort != 19092 {
		t.Errorf("LocalPort = %d, want 19092", fwd.LocalPort)
	}
	if fwd.RemotePort != 9092 {
		t.Errorf("RemotePort = %d, want 9092", fwd.RemotePort)
	}
}

func TestPortForwarder_Stop_NoProcesses(t *testing.T) {
	pf := NewPortForwarder("k3d-dk-local", "dk-local")
	pf.AddForward("redpanda", 19092, 9092)

	// Stop should not panic even when no processes are started
	pf.Stop()

	if len(pf.processes) != 0 {
		t.Errorf("processes should be empty after Stop(), got %d", len(pf.processes))
	}
}
