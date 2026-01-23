package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/validate"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	lintStrict     bool
	lintSkipPII    bool
	lintSet        []string // --set flags for inline overrides
	lintValueFiles []string // -f flags for override files
)

// lintCmd validates package manifests
var lintCmd = &cobra.Command{
	Use:   "lint [package-dir]",
	Short: "Validate package manifests",
	Long: `Validate all manifests in a DP package directory.

The lint command checks:
  - dp.yaml: Data package manifest validation
  - bindings.yaml: Binding configuration validation
  - schemas/: Schema file validation
  - PII classification: Ensures outputs have required classifications

Validation rules include:
  - Required fields (E001-E003)
  - Schema references (E004-E005)
  - Binding configuration (E010-E011)
  - Runtime configuration (E030-E031, E040-E041)
  - PII classification (E025): Outputs must have classification

You can validate with overrides to check merged configuration:
  - Use -f to apply override files (like Helm values files)
  - Use --set for inline overrides
  - Precedence: dp.yaml < -f files (in order) < --set (in order)

Examples:
  # Lint current directory
  dp lint

  # Lint specific package
  dp lint ./my-pipeline

  # Lint with overrides applied
  dp lint ./my-pipeline -f production.yaml

  # Lint with inline override
  dp lint ./my-pipeline --set spec.runtime.image=myimage:v2

  # Strict mode (warnings become errors)
  dp lint --strict

  # Skip PII classification validation
  dp lint --skip-pii`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)

	lintCmd.Flags().BoolVar(&lintStrict, "strict", false, "Treat warnings as errors")
	lintCmd.Flags().BoolVar(&lintSkipPII, "skip-pii", false, "Skip PII classification validation")
	lintCmd.Flags().StringArrayVar(&lintSet, "set", []string{},
		"Override values (key=value, can be repeated)")
	lintCmd.Flags().StringArrayVarP(&lintValueFiles, "values", "f", []string{},
		"Override files (can be repeated)")
}

func runLint(cmd *cobra.Command, args []string) error {
	// Determine package directory
	packageDir := "."
	if len(args) > 0 {
		packageDir = args[0]
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", packageDir)
	}

	fmt.Printf("Linting package: %s\n", packageDir)

	// Apply overrides if provided
	if len(lintValueFiles) > 0 || len(lintSet) > 0 {
		if err := applyLintOverrides(absDir); err != nil {
			return err
		}
		fmt.Println()
	} else {
		fmt.Println()
	}

	// Run validation
	ctx := context.Background()
	validator := validate.NewAggregateValidator(absDir)

	// Configure validation context
	if lintStrict {
		validator.WithContext(&validate.ValidationContext{
			PackageDir:  absDir,
			StrictMode:  true,
			ValidatePII: !lintSkipPII,
		})
	} else if lintSkipPII {
		validator.WithContext(&validate.ValidationContext{
			PackageDir:  absDir,
			ValidatePII: false,
		})
	}

	result := validator.Validate(ctx)

	// Check what files were validated
	files := []string{}
	if _, err := os.Stat(filepath.Join(absDir, "dp.yaml")); err == nil {
		files = append(files, "dp.yaml")
	}
	if _, err := os.Stat(filepath.Join(absDir, "pipeline.yaml")); err == nil {
		files = append(files, "pipeline.yaml")
	}
	if _, err := os.Stat(filepath.Join(absDir, "bindings.yaml")); err == nil {
		files = append(files, "bindings.yaml")
	}
	if entries, err := os.ReadDir(filepath.Join(absDir, "schemas")); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, "schemas/"+e.Name())
			}
		}
	}

	fmt.Printf("Validated files:\n")
	for _, f := range files {
		fmt.Printf("  • %s\n", f)
	}
	fmt.Println()

	if result.Valid && len(result.Warnings) == 0 {
		fmt.Println("✓ All validations passed")
		return nil
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings (%d):\n", len(result.Warnings))
		for _, w := range result.Warnings {
			fmt.Printf("  ⚠ %s\n", w)
		}
		fmt.Println()
	}

	// Print errors
	if result.Errors.HasErrors() {
		fmt.Printf("Errors (%d):\n", len(result.Errors))

		for _, e := range result.Errors {
			field := e.Field
			if field == "" {
				field = "(root)"
			}
			fmt.Printf("  ✗ [%s] %s: %s\n", e.Code, field, e.Message)
		}
		fmt.Println()

		// Return error indicating validation failed
		return fmt.Errorf("validation failed with %d errors", len(result.Errors))
	}

	if lintStrict && len(result.Warnings) > 0 {
		return fmt.Errorf("strict mode: %d warnings treated as errors", len(result.Warnings))
	}

	return nil
}

// applyLintOverrides applies overrides to dp.yaml for validation.
// Creates a backup and modifies the file in place for validation.
func applyLintOverrides(absDir string) error {
	dpPath := filepath.Join(absDir, "dp.yaml")

	// Check if dp.yaml exists
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		return fmt.Errorf("dp.yaml not found in %s", absDir)
	}

	// Read base dp.yaml
	baseData, err := os.ReadFile(dpPath)
	if err != nil {
		return fmt.Errorf("failed to read dp.yaml: %w", err)
	}

	// Parse as generic map for merging
	var base map[string]any
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return fmt.Errorf("failed to parse dp.yaml: %w", err)
	}

	mergeOpts := manifest.DefaultMergeOptions()

	// Apply override files in order
	for _, f := range lintValueFiles {
		overrideData, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read override file %s: %w", f, err)
		}

		var override map[string]any
		if err := yaml.Unmarshal(overrideData, &override); err != nil {
			return fmt.Errorf("failed to parse override file %s: %w", f, err)
		}

		base = manifest.DeepMerge(base, override, mergeOpts)
		fmt.Printf("Applied overrides from: %s\n", f)
	}

	// Apply --set values in order
	for _, s := range lintSet {
		path, value, err := manifest.ParseSetFlag(s)
		if err != nil {
			return fmt.Errorf("invalid --set value: %w", err)
		}

		// Validate the path is allowed
		if err := manifest.ValidateOverridePath(path); err != nil {
			return err
		}

		if err := manifest.SetPath(base, path, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", path, err)
		}
		fmt.Printf("Set: %s=%v\n", path, value)
	}

	// Write merged config back to dp.yaml
	mergedData, err := yaml.Marshal(base)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	// Create backup of original
	backupPath := dpPath + ".bak"
	if err := os.WriteFile(backupPath, baseData, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write merged config for validation
	if err := os.WriteFile(dpPath, mergedData, 0644); err != nil {
		// Restore from backup on failure
		os.WriteFile(dpPath, baseData, 0644)
		return fmt.Errorf("failed to write merged config: %w", err)
	}

	return nil
}
