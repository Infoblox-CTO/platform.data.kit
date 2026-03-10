package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/pipeline"
	"github.com/spf13/cobra"
)

var (
	pipelineShowOutput      string
	pipelineShowDestination string
	pipelineShowScanDirs    []string
)

// pipelineShowCmd displays the pipeline dependency graph.
var pipelineShowCmd = &cobra.Command{
	Use:   "show [dir]",
	Short: "Display pipeline dependency graph",
	Long: `Display the reactive dependency graph derived from Transform and DataSet
manifests (dk.yaml files).

Output formats:
  text      Text tree (default)
  mermaid   Mermaid diagram
  json      JSON adjacency list
  dot       Graphviz DOT format

Examples:
  # Show full dependency graph
  dk pipeline show

  # Show graph leading to a specific dataset
  dk pipeline show --destination event-summary

  # Render as Mermaid
  dk pipeline show --output mermaid

  # Scan specific directories
  dk pipeline show --scan-dir ./transforms --scan-dir ./datasets`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPipelineShow,
}

func init() {
	pipelineCmd.AddCommand(pipelineShowCmd)

	pipelineShowCmd.Flags().StringVarP(&pipelineShowOutput, "output", "o", "text",
		"Output format (text, mermaid, json, dot)")
	pipelineShowCmd.Flags().StringVar(&pipelineShowDestination, "destination", "",
		"Show dependency chain leading to this dataset")
	pipelineShowCmd.Flags().StringArrayVar(&pipelineShowScanDirs, "scan-dir", nil,
		"Directories to scan for dk.yaml files (repeatable)")
}

func runPipelineShow(cmd *cobra.Command, args []string) error {
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
		ShowAll:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	out := cmd.OutOrStdout()
	switch pipelineShowOutput {
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
		return fmt.Errorf("unsupported output format: %s (use text, mermaid, json, dot)", pipelineShowOutput)
	}

	return nil
}
