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

var pipelineShowOutput string

// pipelineShowCmd displays pipeline definition details.
var pipelineShowCmd = &cobra.Command{
	Use:   "show [pipeline-dir]",
	Short: "Display pipeline workflow details",
	Long: `Display the pipeline workflow definition with step details.

Outputs a table (default), JSON, or YAML representation of the pipeline
workflow including all steps and their configuration.

Examples:
  # Show pipeline in current directory
  dp pipeline show

  # Show as JSON
  dp pipeline show --output json

  # Show as YAML
  dp pipeline show --output yaml

  # Show pipeline in specific directory
  dp pipeline show ./my-pipeline`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPipelineShow,
}

func init() {
	pipelineCmd.AddCommand(pipelineShowCmd)

	pipelineShowCmd.Flags().StringVarP(&pipelineShowOutput, "output", "o", "table",
		"Output format: table, json, yaml")
}

func runPipelineShow(cmd *cobra.Command, args []string) error {
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

	switch pipelineShowOutput {
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

		// Show schedule if present
		sched, _ := pipeline.LoadSchedule(pipelineDir)
		if sched != nil {
			cmd.Println()
			cmd.Println("Schedule:")
			cmd.Printf("  Cron:     %s\n", sched.Cron)
			tz := sched.Timezone
			if tz == "" {
				tz = "UTC"
			}
			cmd.Printf("  Timezone: %s\n", tz)
			if sched.Suspend {
				cmd.Println("  Status:   SUSPENDED")
			} else {
				cmd.Println("  Status:   Active")
			}
		}

	default:
		return fmt.Errorf("unsupported output format: %s (use table, json, yaml)", pipelineShowOutput)
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
