package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	pipelineShowOutput      string
	pipelineShowAll         bool
	pipelineShowDestination string
	pipelineShowScanDirs    []string
)

// pipelineShowCmd displays pipeline definition details.
var pipelineShowCmd = &cobra.Command{
	Use:   "show [pipeline-dir]",
	Short: "Display pipeline workflow or dependency graph",
	Long: `Display the pipeline workflow definition or the reactive dependency graph.

When --all or --destination is used, the command scans for Transform and
Asset manifests (dk.yaml files) and renders the dependency graph.

Without those flags, it falls back to displaying the legacy pipeline.yaml
workflow definition.

Output formats for graph mode:
  text      Text tree (default)
  mermaid   Mermaid diagram
  json      JSON adjacency list
  dot       Graphviz DOT format

Output formats for legacy mode:
  table     Tabular step listing (default)
  json      JSON
  yaml      YAML

Examples:
  # Show dependency graph for all transforms
  dk pipeline show --all

  # Show graph leading to a specific asset
  dk pipeline show --destination event-summary

  # Render as Mermaid
  dk pipeline show --all --output mermaid

  # Scan specific directories
  dk pipeline show --all --scan-dir ./transforms --scan-dir ./assets

  # Legacy: show pipeline.yaml
  dk pipeline show

  # Legacy: show as JSON
  dk pipeline show --output json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPipelineShow,
}

func init() {
	pipelineCmd.AddCommand(pipelineShowCmd)

	pipelineShowCmd.Flags().StringVarP(&pipelineShowOutput, "output", "o", "",
		"Output format (graph: text, mermaid, json, dot; legacy: table, json, yaml)")
	pipelineShowCmd.Flags().BoolVar(&pipelineShowAll, "all", false,
		"Show full dependency graph")
	pipelineShowCmd.Flags().StringVar(&pipelineShowDestination, "destination", "",
		"Show dependency chain leading to this asset")
	pipelineShowCmd.Flags().StringArrayVar(&pipelineShowScanDirs, "scan-dir", nil,
		"Directories to scan for dk.yaml files (repeatable)")
}

func runPipelineShow(cmd *cobra.Command, args []string) error {
	// Graph mode when --all or --destination is specified.
	if pipelineShowAll || pipelineShowDestination != "" {
		return runPipelineShowGraph(cmd, args)
	}

	// Legacy pipeline.yaml mode.
	return runPipelineShowLegacy(cmd, args)
}

func runPipelineShowGraph(cmd *cobra.Command, args []string) error {
	// Determine scan directories.
	scanDirs := pipelineShowScanDirs
	if len(scanDirs) == 0 {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to resolve directory: %w", err)
		}
		scanDirs = []string{absDir}
	}

	g, err := pipeline.BuildGraph(pipeline.GraphOptions{
		ScanDirs:    scanDirs,
		Destination: pipelineShowDestination,
		ShowAll:     pipelineShowAll,
	})
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	outputFmt := pipelineShowOutput
	if outputFmt == "" {
		outputFmt = "text"
	}

	out := cmd.OutOrStdout()
	switch outputFmt {
	case "text":
		pipeline.RenderText(out, g, pipelineShowDestination)
	case "mermaid":
		pipeline.RenderMermaid(out, g)
	case "dot":
		pipeline.RenderDOT(out, g)
	case "json":
		if err := pipeline.RenderJSON(out, g); err != nil {
			return fmt.Errorf("failed to render JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported graph output format: %s (use text, mermaid, json, dot)", outputFmt)
	}

	return nil
}

func runPipelineShowLegacy(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	pipelineDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	workflow, err := pipeline.LoadPipeline(pipelineDir)
	if err != nil {
		return fmt.Errorf("no pipeline.yaml found in %s", dir)
	}

	outputFmt := pipelineShowOutput
	if outputFmt == "" {
		outputFmt = "table"
	}

	switch outputFmt {
	case "json":
		data, err := json.MarshalIndent(workflow, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal as JSON: %w", err)
		}
		cmd.Println(string(data))

	case "yaml":
		data, err := yaml.Marshal(workflow)
		if err != nil {
			return fmt.Errorf("failed to marshal as YAML: %w", err)
		}
		cmd.Print(string(data))

	case "table":
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		cmd.Printf("Pipeline: %s\n", workflow.Metadata.Name)
		if workflow.Metadata.Description != "" {
			cmd.Printf("Description: %s\n", workflow.Metadata.Description)
		}
		cmd.Println()

		fmt.Fprintln(w, "STEP\tTYPE\tDETAILS")
		fmt.Fprintln(w, "----\t----\t-------")
		for _, step := range workflow.Steps {
			details := stepDetails(step)
			fmt.Fprintf(w, "%s\t%s\t%s\n", step.Name, step.Type, details)
		}
		w.Flush()

	default:
		return fmt.Errorf("unsupported output format: %s (use table, json, yaml)", outputFmt)
	}

	return nil
}

// stepDetails returns a human-readable summary of a step's key fields.
func stepDetails(step contracts.Step) string {
	var parts []string

	switch step.Type {
	case "sync":
		if step.Input != "" {
			parts = append(parts, "input="+step.Input)
		}
		if step.Output != "" {
			parts = append(parts, "output="+step.Output)
		}
	case "transform", "test":
		if step.Asset != "" {
			parts = append(parts, "asset="+step.Asset)
		}
		if len(step.Command) > 0 {
			parts = append(parts, "cmd="+strings.Join(step.Command, " "))
		}
	case "custom":
		if step.Image != "" {
			parts = append(parts, "image="+step.Image)
		}
	case "publish":
		if step.Promote {
			parts = append(parts, "promote=true")
		}
		if step.Notify != nil && len(step.Notify.Channels) > 0 {
			parts = append(parts, "channels="+strings.Join(step.Notify.Channels, ","))
		}
	}

	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}
