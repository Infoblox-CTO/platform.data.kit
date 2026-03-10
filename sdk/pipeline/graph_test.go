package pipeline

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGraph_ThreeChainedTransforms(t *testing.T) {
	dir := setupGraphTestDir(t)

	g, err := BuildGraph(GraphOptions{
		ScanDirs: []string{dir},
		ShowAll:  true,
	})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// Expect 4 datasets + 3 transforms = 7 nodes.
	if len(g.Nodes) != 7 {
		t.Errorf("expected 7 nodes, got %d", len(g.Nodes))
		for _, n := range g.Nodes {
			t.Logf("  node: %s (%s)", n.ID, n.Type)
		}
	}

	// Expect 6 edges (3 input->transform + 3 transform->output).
	if len(g.Edges) != 6 {
		t.Errorf("expected 6 edges, got %d", len(g.Edges))
		for _, e := range g.Edges {
			t.Logf("  edge: %s -> %s", e.From, e.To)
		}
	}

	// Check transform nodes have trigger info.
	nodeMap := make(map[string]GraphNode)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}
	if nodeMap["ingest"].TriggerPolicy != "on-change" {
		t.Errorf("ingest trigger = %q, want on-change", nodeMap["ingest"].TriggerPolicy)
	}
	if nodeMap["aggregate"].TriggerPolicy != "schedule" {
		t.Errorf("aggregate trigger = %q, want schedule", nodeMap["aggregate"].TriggerPolicy)
	}
}

func TestBuildGraph_FilterDestination(t *testing.T) {
	dir := setupGraphTestDir(t)

	g, err := BuildGraph(GraphOptions{
		ScanDirs:    []string{dir},
		Destination: "enriched-events",
	})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// Should only include: raw-events -> ingest -> raw-events-parquet -> enrich -> enriched-events
	nodeIDs := make(map[string]bool)
	for _, n := range g.Nodes {
		nodeIDs[n.ID] = true
	}

	for _, want := range []string{"raw-events", "ingest", "raw-events-parquet", "enrich", "enriched-events"} {
		if !nodeIDs[want] {
			t.Errorf("expected node %q in filtered graph", want)
		}
	}

	// Should NOT include aggregate or event-summary.
	for _, notWant := range []string{"aggregate", "event-summary"} {
		if nodeIDs[notWant] {
			t.Errorf("unexpected node %q in filtered graph", notWant)
		}
	}
}

func TestRenderText(t *testing.T) {
	dir := setupGraphTestDir(t)
	g, err := BuildGraph(GraphOptions{
		ScanDirs:    []string{dir},
		Destination: "event-summary",
	})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	var buf bytes.Buffer
	RenderText(&buf, g, "event-summary")
	output := buf.String()

	for _, want := range []string{"event-summary", "ingest", "enrich", "aggregate", "raw-events"} {
		if !strings.Contains(output, want) {
			t.Errorf("text output should contain %q, got:\n%s", want, output)
		}
	}
}

func TestRenderMermaid(t *testing.T) {
	dir := setupGraphTestDir(t)
	g, err := BuildGraph(GraphOptions{ScanDirs: []string{dir}, ShowAll: true})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	var buf bytes.Buffer
	RenderMermaid(&buf, g)
	output := buf.String()

	if !strings.Contains(output, "graph TD") {
		t.Errorf("mermaid output should start with 'graph TD'")
	}
	if !strings.Contains(output, "-->") {
		t.Errorf("mermaid output should contain edges (-->)")
	}
}

func TestRenderDOT(t *testing.T) {
	dir := setupGraphTestDir(t)
	g, err := BuildGraph(GraphOptions{ScanDirs: []string{dir}, ShowAll: true})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	var buf bytes.Buffer
	RenderDOT(&buf, g)
	output := buf.String()

	if !strings.Contains(output, "digraph pipeline") {
		t.Errorf("DOT output should contain 'digraph pipeline'")
	}
	if !strings.Contains(output, "->") {
		t.Errorf("DOT output should contain edges (->)")
	}
}

func TestRenderJSON(t *testing.T) {
	dir := setupGraphTestDir(t)
	g, err := BuildGraph(GraphOptions{ScanDirs: []string{dir}, ShowAll: true})
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, g); err != nil {
		t.Fatalf("RenderJSON failed: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, `"nodes"`) {
		t.Errorf("JSON output should contain 'nodes'")
	}
	if !strings.Contains(output, `"edges"`) {
		t.Errorf("JSON output should contain 'edges'")
	}
}

// setupGraphTestDir creates a temp directory with three chained transforms.
func setupGraphTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	datasets := map[string]string{
		"raw-events": `apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: raw-events
  version: "1.0.0"
spec:
  store: kafka
  topic: raw.events
`,
		"raw-events-parquet": `apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: raw-events-parquet
  version: "1.0.0"
spec:
  store: lake
  prefix: data/raw/
  format: parquet
`,
		"enriched-events": `apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: enriched-events
  version: "1.0.0"
spec:
  store: lake
  prefix: data/enriched/
  format: parquet
`,
		"event-summary": `apiVersion: datakit.infoblox.dev/v1alpha1
kind: DataSet
metadata:
  name: event-summary
  version: "1.0.0"
spec:
  store: warehouse
  table: analytics.summary
`,
	}

	transforms := map[string]string{
		"ingest": `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: ingest
spec:
  runtime: cloudquery
  inputs:
    - dataset: raw-events
  outputs:
    - dataset: raw-events-parquet
  trigger:
    policy: on-change
`,
		"enrich": `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: enrich
spec:
  runtime: generic-python
  inputs:
    - dataset: raw-events-parquet
  outputs:
    - dataset: enriched-events
  image: enrich:latest
  trigger:
    policy: on-change
`,
		"aggregate": `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: aggregate
spec:
  runtime: dbt
  inputs:
    - dataset: enriched-events
  outputs:
    - dataset: event-summary
  image: dbt:latest
  trigger:
    policy: schedule
    schedule:
      cron: "0 */6 * * *"
`,
	}

	for name, content := range datasets {
		datasetDir := filepath.Join(dir, "datasets", name)
		if err := os.MkdirAll(datasetDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(datasetDir, "dk.yaml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for name, content := range transforms {
		transformDir := filepath.Join(dir, "transforms", name)
		if err := os.MkdirAll(transformDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(transformDir, "dk.yaml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}
