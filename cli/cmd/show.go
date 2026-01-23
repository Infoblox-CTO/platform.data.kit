// Package cmd provides the CLI commands for the dp tool.
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/data.platform.kit/sdk/manifest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	showSet          []string // --set flags for inline overrides
	showValueFiles   []string // -f flags for override files
	showOutputFormat string   // --output flag for format selection
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show [path]",
	Short: "Show the effective DataPackage manifest",
	Long: `Show the effective DataPackage manifest after applying overrides.

This command displays the merged manifest that would be used when running
the pipeline. Use this to preview the effect of override files and --set flags.

Examples:
  # Show the manifest as-is
  dp show ./my-pipeline

  # Show with override file applied
  dp show ./my-pipeline -f production.yaml

  # Show with inline overrides
  dp show ./my-pipeline --set spec.runtime.image=myimage:v2

  # Show combined overrides (file first, then --set)
  dp show ./my-pipeline -f base.yaml --set spec.runtime.timeout=1h

  # Output as JSON
  dp show ./my-pipeline -o json
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine the package path
		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		// Resolve to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		output, err := showManifest(absPath, cmd.OutOrStdout())
		if err != nil {
			return err
		}

		fmt.Fprint(cmd.OutOrStdout(), output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)

	showCmd.Flags().StringArrayVar(&showSet, "set", []string{},
		"Override values (key=value, can be repeated)")
	showCmd.Flags().StringArrayVarP(&showValueFiles, "values", "f", []string{},
		"Override files (can be repeated)")
	showCmd.Flags().StringVarP(&showOutputFormat, "output", "o", "yaml",
		"Output format: yaml or json")
}

// showManifest reads the manifest, applies overrides, and returns the output.
// The writer is for any status messages; the return value is the formatted manifest.
func showManifest(packageDir string, w io.Writer) (string, error) {
	// Find dp.yaml
	dpPath := filepath.Join(packageDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dp.yaml not found in %s", packageDir)
	}

	// Read base dp.yaml
	baseData, err := os.ReadFile(dpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read dp.yaml: %w", err)
	}

	// Parse as generic map for merging
	var base map[string]any
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return "", fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	mergeOpts := manifest.DefaultMergeOptions()

	// Apply override files in order
	for _, f := range showValueFiles {
		overrideData, err := os.ReadFile(f)
		if err != nil {
			return "", fmt.Errorf("failed to read override file %s: %w", f, err)
		}

		var override map[string]any
		if err := yaml.Unmarshal(overrideData, &override); err != nil {
			return "", fmt.Errorf("failed to parse override file %s: %w", f, err)
		}

		base = manifest.DeepMerge(base, override, mergeOpts)
	}

	// Apply --set values in order
	for _, s := range showSet {
		path, value, err := manifest.ParseSetFlag(s)
		if err != nil {
			return "", fmt.Errorf("invalid --set value: %w", err)
		}

		// Validate the path is allowed
		if err := manifest.ValidateOverridePath(path); err != nil {
			return "", err
		}

		if err := manifest.SetPath(base, path, value); err != nil {
			return "", fmt.Errorf("failed to set %s: %w", path, err)
		}
	}

	// Format output
	var output []byte
	switch showOutputFormat {
	case "json":
		output, err = json.MarshalIndent(base, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal as JSON: %w", err)
		}
		output = append(output, '\n')
	case "yaml":
		output, err = yaml.Marshal(base)
		if err != nil {
			return "", fmt.Errorf("failed to marshal as YAML: %w", err)
		}
	default:
		return "", fmt.Errorf("unsupported output format: %s (use yaml or json)", showOutputFormat)
	}

	return string(output), nil
}
