// Package cmd provides the CLI commands for the dk tool.
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
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
	Short: "Show the effective manifest",
	Long: `Show the effective manifest after applying overrides.

This command displays the merged manifest that would be used when running
the pipeline. Use this to preview the effect of override files and --set flags.

Examples:
  # Show the manifest as-is
  dk show ./my-pipeline

  # Show with override file applied
  dk show ./my-pipeline -f production.yaml

  # Show with inline overrides
  dk show ./my-pipeline --set spec.image=myimage:v2

  # Show combined overrides (file first, then --set)
  dk show ./my-pipeline -f base.yaml --set spec.timeout=1h

  # Output as JSON
  dk show ./my-pipeline -o json
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
	// Find dk.yaml
	dkPath := filepath.Join(packageDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		return "", fmt.Errorf("dk.yaml not found in %s", packageDir)
	}

	// Read base dk.yaml
	baseData, err := os.ReadFile(dkPath)
	if err != nil {
		return "", fmt.Errorf("failed to read dk.yaml: %w", err)
	}

	// Parse as generic map for merging
	var base map[string]any
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return "", fmt.Errorf("failed to parse dk.yaml: %w", err)
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

	// Resolve dataset details if datasets are referenced
	resolveDataSetDetails(base, packageDir)

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

// resolveDataSetDetails enriches the datasets section with resolved info from dataset.yaml files.
func resolveDataSetDetails(base map[string]any, packageDir string) {
	spec, ok := base["spec"].(map[string]any)
	if !ok {
		return
	}

	datasetNames, ok := spec["datasets"].([]any)
	if !ok || len(datasetNames) == 0 {
		return
	}

	var resolved []map[string]any
	for _, nameAny := range datasetNames {
		name, ok := nameAny.(string)
		if !ok {
			continue
		}

		entry := map[string]any{"name": name}

		a, err := dataset.FindDataSetByName(packageDir, name)
		if err == nil && a != nil {
			entry["store"] = a.Spec.Store
			if a.Spec.Classification != "" {
				entry["classification"] = a.Spec.Classification
			}
		} else {
			entry["status"] = "not found"
		}

		resolved = append(resolved, entry)
	}

	spec["datasets"] = resolved
}
