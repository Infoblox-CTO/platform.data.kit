package charts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractChart(t *testing.T) {
	for _, chartName := range ChartNames {
		t.Run(chartName, func(t *testing.T) {
			destDir := t.TempDir()
			chartDir := filepath.Join(destDir, chartName)

			err := ExtractChart(chartName, chartDir)
			if err != nil {
				t.Fatalf("ExtractChart(%q) error: %v", chartName, err)
			}

			// Verify Chart.yaml exists
			if _, err := os.Stat(filepath.Join(chartDir, "Chart.yaml")); os.IsNotExist(err) {
				t.Errorf("Chart.yaml not extracted for %q", chartName)
			}

			// Verify values.yaml exists
			if _, err := os.Stat(filepath.Join(chartDir, "values.yaml")); os.IsNotExist(err) {
				t.Errorf("values.yaml not extracted for %q", chartName)
			}

			// Verify templates directory exists
			if _, err := os.Stat(filepath.Join(chartDir, "templates")); os.IsNotExist(err) {
				t.Errorf("templates/ not extracted for %q", chartName)
			}
		})
	}
}

func TestExtractChart_NotFound(t *testing.T) {
	destDir := t.TempDir()
	err := ExtractChart("nonexistent-chart", filepath.Join(destDir, "nonexistent"))
	if err == nil {
		t.Error("ExtractChart should return error for nonexistent chart")
	}
}

func TestApplyOverrides_Empty(t *testing.T) {
	args := []string{"upgrade", "--install", "test", "/tmp/chart"}
	override := ChartOverride{}

	result := ApplyOverrides(args, override)
	if len(result) != len(args) {
		t.Errorf("ApplyOverrides with empty override should not change args length: got %d, want %d", len(result), len(args))
	}
}

func TestApplyOverrides_WithValues(t *testing.T) {
	args := []string{"upgrade", "--install", "test", "/tmp/chart"}
	override := ChartOverride{
		Values: map[string]interface{}{
			"replicaCount":                2,
			"resources.limits.memory":     "512Mi",
			"primary.persistence.enabled": false,
		},
	}

	result := ApplyOverrides(args, override)

	// Should have original args + 2 * len(Values) additional args (--set key=val)
	expectedLen := len(args) + 2*len(override.Values)
	if len(result) != expectedLen {
		t.Errorf("ApplyOverrides result length = %d, want %d", len(result), expectedLen)
	}

	// Verify --set flags are present
	setCount := 0
	for _, a := range result {
		if a == "--set" {
			setCount++
		}
	}
	if setCount != len(override.Values) {
		t.Errorf("expected %d --set flags, got %d", len(override.Values), setCount)
	}
}

func TestDefaultCharts_Registry(t *testing.T) {
	if len(DefaultCharts) != 5 {
		t.Fatalf("DefaultCharts should have 5 entries, got %d", len(DefaultCharts))
	}

	expectedNames := []string{"redpanda", "redpanda-console", "localstack", "postgres", "marquez"}
	for i, expected := range expectedNames {
		if DefaultCharts[i].Name != expected {
			t.Errorf("DefaultCharts[%d].Name = %q, want %q", i, DefaultCharts[i].Name, expected)
		}
	}
}

func TestDefaultCharts_UniqueNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, def := range DefaultCharts {
		if seen[def.Name] {
			t.Errorf("duplicate chart name: %q", def.Name)
		}
		seen[def.Name] = true
	}
}

func TestDefaultCharts_UniquePorts(t *testing.T) {
	seen := make(map[int]string)
	for _, def := range DefaultCharts {
		for _, pf := range def.PortForwards {
			if existing, ok := seen[pf.LocalPort]; ok {
				t.Errorf("duplicate local port %d: used by %q and %q", pf.LocalPort, existing, def.Name)
			}
			seen[pf.LocalPort] = def.Name
		}
	}
}

func TestDefaultCharts_AllHaveRequired(t *testing.T) {
	for _, def := range DefaultCharts {
		if def.Name == "" {
			t.Error("chart definition has empty Name")
		}
		if def.ReleaseName == "" {
			t.Errorf("chart %q has empty ReleaseName", def.Name)
		}
		if def.Namespace == "" {
			t.Errorf("chart %q has empty Namespace", def.Name)
		}
		if len(def.PortForwards) == 0 {
			t.Errorf("chart %q has no PortForwards", def.Name)
		}
		if len(def.HealthLabels) == 0 {
			t.Errorf("chart %q has no HealthLabels", def.Name)
		}
		if def.HealthTimeout == 0 {
			t.Errorf("chart %q has zero HealthTimeout", def.Name)
		}
		if len(def.DisplayEndpoints) == 0 {
			t.Errorf("chart %q has no DisplayEndpoints", def.Name)
		}
	}
}

func TestAllLocalPorts(t *testing.T) {
	ports := AllLocalPorts(DefaultCharts)

	expectedPorts := map[int]bool{
		19092: true, 18081: true, // redpanda
		4566: true,                         // localstack
		5432: true,                         // postgres
		18080: true,                         // redpanda-console
		5000: true, 5001: true, 3000: true,  // marquez
	}

	if len(ports) != len(expectedPorts) {
		t.Errorf("AllLocalPorts returned %d ports, want %d", len(ports), len(expectedPorts))
	}

	for _, p := range ports {
		if !expectedPorts[p] {
			t.Errorf("unexpected port %d in AllLocalPorts", p)
		}
	}
}

func TestDeployResult_HasFailures(t *testing.T) {
	tests := []struct {
		name     string
		result   DeployResult
		expected bool
	}{
		{
			name:     "no failures",
			result:   DeployResult{Succeeded: []string{"a", "b"}},
			expected: false,
		},
		{
			name:     "with failures",
			result:   DeployResult{Failed: []ChartError{{ChartName: "a"}}},
			expected: true,
		},
		{
			name:     "empty result",
			result:   DeployResult{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasFailures(); got != tt.expected {
				t.Errorf("HasFailures() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestChartDefinition_HealthTimeout(t *testing.T) {
	for _, def := range DefaultCharts {
		if def.HealthTimeout < 30*time.Second {
			t.Errorf("chart %q HealthTimeout %v is too short (min 30s)", def.Name, def.HealthTimeout)
		}
		if def.HealthTimeout > 5*time.Minute {
			t.Errorf("chart %q HealthTimeout %v is too long (max 5m)", def.Name, def.HealthTimeout)
		}
	}
}

func TestPortsForService(t *testing.T) {
	tests := []struct {
		appLabel string
		expected int
	}{
		{"redpanda", 2},
		{"localstack", 1},
		{"postgres", 1},   // matches by chart name
		{"postgresql", 1}, // matches via health label
		{"marquez", 3},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.appLabel, func(t *testing.T) {
			result := PortsForService(DefaultCharts, tt.appLabel)
			if len(result) != tt.expected {
				t.Errorf("PortsForService(%q) returned %d ports, want %d", tt.appLabel, len(result), tt.expected)
			}
		})
	}
}
