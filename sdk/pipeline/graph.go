package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// GraphNode represents a node in the pipeline dependency graph.
type GraphNode struct {
	// ID is the unique identifier (transform or dataset name).
	ID string `json:"id"`
	// Type is "transform" or "dataset".
	Type string `json:"type"`
	// Runtime is the transform runtime (empty for datasets).
	Runtime string `json:"runtime,omitempty"`
	// TriggerPolicy is the trigger policy (empty for datasets).
	TriggerPolicy string `json:"triggerPolicy,omitempty"`
	// TriggerDetail is human-readable trigger info (e.g., cron expression).
	TriggerDetail string `json:"triggerDetail,omitempty"`
	// FilePath is the source file path.
	FilePath string `json:"filePath,omitempty"`
}

// GraphEdge represents a directed edge in the pipeline graph.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PipelineGraph is a DAG of transforms and datasets.
type PipelineGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphOptions configures graph building.
type GraphOptions struct {
	// ScanDirs is the list of directories to scan for dk.yaml files.
	ScanDirs []string
	// Destination filters the graph to show only the chain leading to this dataset.
	Destination string
	// ShowAll shows the full graph.
	ShowAll bool
}

// BuildGraph scans directories for Transform and DataSet manifests, then builds
// a dependency graph.
func BuildGraph(opts GraphOptions) (*PipelineGraph, error) {
	transforms, datasets, err := scanManifests(opts.ScanDirs)
	if err != nil {
		return nil, err
	}

	g := &PipelineGraph{}
	nodeSet := make(map[string]bool)

	// Add dataset nodes.
	for name := range datasets {
		g.Nodes = append(g.Nodes, GraphNode{ID: name, Type: "dataset"})
		nodeSet[name] = true
	}

	// Add transform nodes and edges.
	for _, t := range transforms {
		tNode := GraphNode{
			ID:      t.manifest.Metadata.Name,
			Type:    "transform",
			Runtime: string(t.manifest.Spec.Runtime),
		}

		// Determine trigger info.
		if t.manifest.Spec.Trigger != nil {
			tNode.TriggerPolicy = string(t.manifest.Spec.Trigger.Policy)
			if t.manifest.Spec.Trigger.Policy == contracts.TriggerPolicySchedule && t.manifest.Spec.Trigger.Schedule != nil {
				tNode.TriggerDetail = t.manifest.Spec.Trigger.Schedule.Cron
			}
		}

		tNode.FilePath = t.path
		g.Nodes = append(g.Nodes, tNode)
		nodeSet[t.manifest.Metadata.Name] = true

		// Edges: input datasets → transform.
		for _, in := range t.manifest.Spec.Inputs {
			datasetName := in.DataSet
			if datasetName == "" {
				continue
			}
			// Ensure input dataset node exists (even if no manifest found).
			if !nodeSet[datasetName] {
				g.Nodes = append(g.Nodes, GraphNode{ID: datasetName, Type: "dataset"})
				nodeSet[datasetName] = true
			}
			g.Edges = append(g.Edges, GraphEdge{From: datasetName, To: t.manifest.Metadata.Name})
		}

		// Edges: transform → output datasets.
		for _, out := range t.manifest.Spec.Outputs {
			datasetName := out.DataSet
			if datasetName == "" {
				continue
			}
			if !nodeSet[datasetName] {
				g.Nodes = append(g.Nodes, GraphNode{ID: datasetName, Type: "dataset"})
				nodeSet[datasetName] = true
			}
			g.Edges = append(g.Edges, GraphEdge{From: t.manifest.Metadata.Name, To: datasetName})
		}
	}

	// Sort nodes for stable output.
	sort.Slice(g.Nodes, func(i, j int) bool {
		if g.Nodes[i].Type != g.Nodes[j].Type {
			return g.Nodes[i].Type < g.Nodes[j].Type
		}
		return g.Nodes[i].ID < g.Nodes[j].ID
	})

	// Filter to destination if requested.
	if opts.Destination != "" {
		g = filterToDestination(g, opts.Destination)
	}

	return g, nil
}

// filterToDestination returns a subgraph containing only nodes on paths
// leading to the destination dataset.
func filterToDestination(g *PipelineGraph, dest string) *PipelineGraph {
	// Build reverse adjacency list.
	reverseAdj := make(map[string][]string)
	for _, e := range g.Edges {
		reverseAdj[e.To] = append(reverseAdj[e.To], e.From)
	}

	// BFS backward from destination.
	keep := make(map[string]bool)
	queue := []string{dest}
	keep[dest] = true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, prev := range reverseAdj[curr] {
			if !keep[prev] {
				keep[prev] = true
				queue = append(queue, prev)
			}
		}
	}

	// Build filtered graph.
	nodeMap := make(map[string]GraphNode)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}

	fg := &PipelineGraph{}
	for id := range keep {
		if n, ok := nodeMap[id]; ok {
			fg.Nodes = append(fg.Nodes, n)
		} else {
			fg.Nodes = append(fg.Nodes, GraphNode{ID: id, Type: "dataset"})
		}
	}
	for _, e := range g.Edges {
		if keep[e.From] && keep[e.To] {
			fg.Edges = append(fg.Edges, e)
		}
	}

	sort.Slice(fg.Nodes, func(i, j int) bool {
		return fg.Nodes[i].ID < fg.Nodes[j].ID
	})

	return fg
}

// RenderText renders the graph as a text tree to the writer.
func RenderText(w io.Writer, g *PipelineGraph, destination string) {
	if len(g.Nodes) == 0 {
		fmt.Fprintln(w, "No transforms or datasets found.")
		return
	}

	if destination != "" {
		fmt.Fprintf(w, "Pipeline Graph → %s\n", destination)
		fmt.Fprintln(w, strings.Repeat("═", 30))
		fmt.Fprintln(w)
		renderChain(w, g, destination)
		return
	}

	fmt.Fprintln(w, "Pipeline Dependency Graph")
	fmt.Fprintln(w, strings.Repeat("═", 30))
	fmt.Fprintln(w)

	// Find root datasets (no incoming edges).
	hasIncoming := make(map[string]bool)
	for _, e := range g.Edges {
		hasIncoming[e.To] = true
	}

	roots := []string{}
	for _, n := range g.Nodes {
		if n.Type == "dataset" && !hasIncoming[n.ID] {
			roots = append(roots, n.ID)
		}
	}
	sort.Strings(roots)

	if len(roots) == 0 {
		// Fall back to showing all nodes.
		for _, n := range g.Nodes {
			fmt.Fprintf(w, "  %s (%s)\n", n.ID, n.Type)
		}
		return
	}

	visited := make(map[string]bool)
	for _, root := range roots {
		renderFromRoot(w, g, root, visited)
	}
}

func renderChain(w io.Writer, g *PipelineGraph, dest string) {
	// Build the forward adjacency and find topological order to dest.
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	reverseAdj := make(map[string][]string)
	for _, e := range g.Edges {
		reverseAdj[e.To] = append(reverseAdj[e.To], e.From)
	}

	// Build ordered chain from roots to destination.
	nodeMap := make(map[string]GraphNode)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}

	// Find roots of this subgraph.
	hasIncoming := make(map[string]bool)
	for _, e := range g.Edges {
		hasIncoming[e.To] = true
	}

	roots := []string{}
	for _, n := range g.Nodes {
		if !hasIncoming[n.ID] {
			roots = append(roots, n.ID)
		}
	}

	// BFS from roots, printing in order.
	visited := make(map[string]bool)
	queue := make([]string, 0)
	for _, r := range roots {
		queue = append(queue, r)
	}

	first := true
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if visited[curr] {
			continue
		}
		visited[curr] = true

		node := nodeMap[curr]
		if !first {
			fmt.Fprintln(w, "    │")
			fmt.Fprintln(w, "    ▼")
		}
		first = false

		if node.Type == "transform" {
			triggerInfo := node.TriggerPolicy
			if node.TriggerDetail != "" {
				triggerInfo += " (" + node.TriggerDetail + ")"
			}
			if triggerInfo == "" {
				triggerInfo = "manual"
			}
			fmt.Fprintf(w, "  ┌%s┐\n", strings.Repeat("─", 20))
			fmt.Fprintf(w, "  │ %-18s │  trigger: %s\n", node.ID, triggerInfo)
			fmt.Fprintf(w, "  │ runtime: %-9s │\n", runtimeShort(node.Runtime))
			fmt.Fprintf(w, "  └%s┘\n", strings.Repeat("─", 20))
		} else {
			fmt.Fprintf(w, "  %s\n", node.ID)
		}

		// Queue children.
		children := adj[curr]
		sort.Strings(children)
		for _, child := range children {
			if !visited[child] {
				queue = append(queue, child)
			}
		}
	}
}

func renderFromRoot(w io.Writer, g *PipelineGraph, root string, visited map[string]bool) {
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	nodeMap := make(map[string]GraphNode)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}

	var walk func(id string, depth int)
	walk = func(id string, depth int) {
		if visited[id] {
			return
		}
		visited[id] = true

		node := nodeMap[id]
		indent := strings.Repeat("  ", depth)

		if node.Type == "transform" {
			triggerInfo := node.TriggerPolicy
			if node.TriggerDetail != "" {
				triggerInfo += " (" + node.TriggerDetail + ")"
			}
			if triggerInfo == "" {
				triggerInfo = "manual"
			}
			fmt.Fprintf(w, "%s  [%s] runtime=%s trigger=%s\n", indent, node.ID, runtimeShort(node.Runtime), triggerInfo)
		} else {
			fmt.Fprintf(w, "%s  %s\n", indent, node.ID)
		}

		children := adj[id]
		sort.Strings(children)
		for _, child := range children {
			walk(child, depth+1)
		}
	}

	walk(root, 0)
}

func runtimeShort(r string) string {
	switch r {
	case "cloudquery":
		return "cq"
	case "generic-go":
		return "go"
	case "generic-python":
		return "python"
	default:
		return r
	}
}

// RenderMermaid renders the graph as a Mermaid diagram.
func RenderMermaid(w io.Writer, g *PipelineGraph) {
	fmt.Fprintln(w, "graph TD")

	nodeMap := make(map[string]GraphNode)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}

	// Use safe IDs for Mermaid (replace hyphens).
	safeID := func(id string) string {
		return strings.ReplaceAll(id, "-", "_")
	}

	// Print node definitions.
	for _, n := range g.Nodes {
		sid := safeID(n.ID)
		if n.Type == "transform" {
			trigger := n.TriggerPolicy
			if trigger == "" {
				trigger = "manual"
			}
			fmt.Fprintf(w, "  %s[%s<br/>%s / %s]\n", sid, n.ID, runtimeShort(n.Runtime), trigger)
		} else {
			fmt.Fprintf(w, "  %s[%s]\n", sid, n.ID)
		}
	}

	// Print edges.
	for _, e := range g.Edges {
		fmt.Fprintf(w, "  %s --> %s\n", safeID(e.From), safeID(e.To))
	}
}

// RenderDOT renders the graph in Graphviz DOT format.
func RenderDOT(w io.Writer, g *PipelineGraph) {
	fmt.Fprintln(w, "digraph pipeline {")
	fmt.Fprintln(w, "  rankdir=TD;")
	fmt.Fprintln(w, "  node [shape=box];")
	fmt.Fprintln(w)

	nodeMap := make(map[string]GraphNode)
	for _, n := range g.Nodes {
		nodeMap[n.ID] = n
	}

	dotID := func(id string) string {
		return `"` + id + `"`
	}

	for _, n := range g.Nodes {
		if n.Type == "transform" {
			trigger := n.TriggerPolicy
			if trigger == "" {
				trigger = "manual"
			}
			label := fmt.Sprintf("%s\\n%s / %s", n.ID, runtimeShort(n.Runtime), trigger)
			fmt.Fprintf(w, "  %s [label=%q shape=box style=filled fillcolor=lightblue];\n", dotID(n.ID), label)
		} else {
			fmt.Fprintf(w, "  %s [label=%q shape=ellipse];\n", dotID(n.ID), n.ID)
		}
	}

	fmt.Fprintln(w)
	for _, e := range g.Edges {
		fmt.Fprintf(w, "  %s -> %s;\n", dotID(e.From), dotID(e.To))
	}

	fmt.Fprintln(w, "}")
}

// RenderJSON renders the graph as JSON.
func RenderJSON(w io.Writer, g *PipelineGraph) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(g)
}

// --- internal helpers ---

type scannedTransform struct {
	manifest *contracts.Transform
	path     string
}

func scanManifests(dirs []string) ([]scannedTransform, map[string]*contracts.DataSetManifest, error) {
	var transforms []scannedTransform
	datasets := make(map[string]*contracts.DataSetManifest)

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip inaccessible
			}
			if info.IsDir() {
				return nil
			}
			if info.Name() != "dk.yaml" {
				return nil
			}

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}

			// Quick peek at kind.
			var peek struct {
				Kind string `yaml:"kind"`
			}
			if err := yaml.Unmarshal(data, &peek); err != nil {
				return nil
			}

			switch peek.Kind {
			case "Transform":
				var t contracts.Transform
				if err := yaml.Unmarshal(data, &t); err == nil {
					transforms = append(transforms, scannedTransform{manifest: &t, path: path})
				}
			case "DataSet":
				var a contracts.DataSetManifest
				if err := yaml.Unmarshal(data, &a); err == nil {
					datasets[a.Metadata.Name] = &a
				}
			}

			return nil
		})
		if err != nil {
			return nil, nil, fmt.Errorf("scanning %s: %w", dir, err)
		}
	}

	return transforms, datasets, nil
}
