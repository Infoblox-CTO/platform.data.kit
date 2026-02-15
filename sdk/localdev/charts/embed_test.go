package charts

import (
	"testing"
)

func TestChartNamesNotEmpty(t *testing.T) {
	if len(ChartNames) == 0 {
		t.Error("ChartNames is empty, expected at least one chart")
	}
}

func TestChartNamesContainsExpected(t *testing.T) {
	expected := []string{"redpanda", "localstack", "postgres", "marquez"}

	for _, name := range expected {
		found := false
		for _, chart := range ChartNames {
			if chart == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ChartNames missing expected chart %q", name)
		}
	}
}

func TestEmbeddedChartsExist(t *testing.T) {
	for _, chartName := range ChartNames {
		entries, err := FS.ReadDir(chartName)
		if err != nil {
			t.Errorf("Failed to read embedded chart %q: %v", chartName, err)
			continue
		}

		if len(entries) == 0 {
			t.Errorf("Embedded chart %q is empty", chartName)
		}

		// Verify Chart.yaml exists
		_, err = FS.ReadFile(chartName + "/Chart.yaml")
		if err != nil {
			t.Errorf("Chart %q missing Chart.yaml: %v", chartName, err)
		}

		// Verify values.yaml exists
		_, err = FS.ReadFile(chartName + "/values.yaml")
		if err != nil {
			t.Errorf("Chart %q missing values.yaml: %v", chartName, err)
		}

		// Verify templates directory exists
		_, err = FS.ReadDir(chartName + "/templates")
		if err != nil {
			t.Errorf("Chart %q missing templates directory: %v", chartName, err)
		}
	}
}
