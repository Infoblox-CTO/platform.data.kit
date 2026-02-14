package pipeline

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestListTemplates(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}

	if len(templates) < 3 {
		t.Errorf("ListTemplates() returned %d templates, want at least 3", len(templates))
	}

	expected := map[string]bool{
		"sync-transform-test": false,
		"sync-only":           false,
		"custom":              false,
	}
	for _, name := range templates {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected template %q not found in list", name)
		}
	}
}

func TestRenderTemplate_SyncTransformTest(t *testing.T) {
	result, err := RenderTemplate("sync-transform-test", TemplateConfig{Name: "my-pipeline"})
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	if !strings.Contains(result, "name: my-pipeline") {
		t.Error("rendered output should contain pipeline name")
	}
	if !strings.Contains(result, "type: sync") {
		t.Error("rendered output should contain sync step")
	}
	if !strings.Contains(result, "type: transform") {
		t.Error("rendered output should contain transform step")
	}
	if !strings.Contains(result, "type: test") {
		t.Error("rendered output should contain test step")
	}
	if !strings.Contains(result, "type: publish") {
		t.Error("rendered output should contain publish step")
	}

	// Verify the output is valid YAML
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("rendered output is not valid YAML: %v", err)
	}
}

func TestRenderTemplate_SyncOnly(t *testing.T) {
	result, err := RenderTemplate("sync-only", TemplateConfig{Name: "sync-pipe"})
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	if !strings.Contains(result, "name: sync-pipe") {
		t.Error("rendered output should contain pipeline name")
	}
	if !strings.Contains(result, "type: sync") {
		t.Error("rendered output should contain sync step")
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("rendered output is not valid YAML: %v", err)
	}
}

func TestRenderTemplate_Custom(t *testing.T) {
	result, err := RenderTemplate("custom", TemplateConfig{Name: "custom-pipe"})
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	if !strings.Contains(result, "name: custom-pipe") {
		t.Error("rendered output should contain pipeline name")
	}
	if !strings.Contains(result, "type: custom") {
		t.Error("rendered output should contain custom step")
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("rendered output is not valid YAML: %v", err)
	}
}

func TestRenderTemplate_NotFound(t *testing.T) {
	_, err := RenderTemplate("nonexistent", TemplateConfig{Name: "test"})
	if err == nil {
		t.Fatal("expected error for nonexistent template, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}
