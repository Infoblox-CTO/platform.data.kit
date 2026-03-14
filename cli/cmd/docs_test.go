package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildReference(t *testing.T) {
	ref := buildReference()

	if ref.Workflow == "" {
		t.Fatal("workflow should not be empty")
	}

	if len(ref.Commands) == 0 {
		t.Fatal("commands should not be empty")
	}

	// Root command should be first.
	if ref.Commands[0].Path != "dk" {
		t.Fatalf("first command should be dk, got %s", ref.Commands[0].Path)
	}

	// Spot-check that key commands exist.
	paths := make(map[string]bool)
	for _, c := range ref.Commands {
		paths[c.Path] = true
	}
	for _, want := range []string{"dk init", "dk lint", "dk build", "dk run", "dk docs"} {
		if !paths[want] {
			t.Errorf("expected command %q in reference", want)
		}
	}

	// Schemas should include all manifest kinds.
	for _, kind := range []string{"Transform", "DataSet", "Store", "Connector", "DataSetGroup"} {
		if _, ok := ref.Schemas[kind]; !ok {
			t.Errorf("expected schema for %q", kind)
		}
	}

	// Errors should be sorted and non-empty.
	if len(ref.Errors) == 0 {
		t.Fatal("errors should not be empty")
	}
	for i := 1; i < len(ref.Errors); i++ {
		if ref.Errors[i].Code < ref.Errors[i-1].Code {
			t.Fatalf("errors not sorted: %s before %s", ref.Errors[i-1].Code, ref.Errors[i].Code)
		}
	}

	// Enums should have key entries.
	for _, name := range []string{"runtime", "mode", "kind", "classification"} {
		if vals, ok := ref.Enums[name]; !ok || len(vals) == 0 {
			t.Errorf("expected non-empty enum %q", name)
		}
	}
}

func TestRenderLLM(t *testing.T) {
	ref := buildReference()
	var buf bytes.Buffer
	if err := renderLLM(&buf, ref); err != nil {
		t.Fatalf("renderLLM: %v", err)
	}
	out := buf.String()

	// Should be valid YAML with key markers.
	for _, want := range []string{"version:", "commands:", "schemas:", "errors:", "enums:"} {
		if !strings.Contains(out, want) {
			t.Errorf("LLM output missing %q", want)
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	ref := buildReference()
	var buf bytes.Buffer
	if err := renderMarkdown(&buf, ref); err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"# dk CLI Reference", "## Commands", "## Manifest Schemas", "## Validation Error Codes", "## Enum Values"} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown output missing %q", want)
		}
	}
}

func TestRenderText(t *testing.T) {
	ref := buildReference()
	var buf bytes.Buffer
	if err := renderText(&buf, ref); err != nil {
		t.Fatalf("renderText: %v", err)
	}
	out := buf.String()

	for _, want := range []string{"COMMANDS", "MANIFEST KINDS", "VALIDATION ERRORS", "ENUM VALUES"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q", want)
		}
	}
}
