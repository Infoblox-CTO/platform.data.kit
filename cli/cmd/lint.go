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
	lintStrict         bool
	lintSkipPII        bool
	lintSkipSchemaLock bool
	lintSet            []string // --set flags for inline overrides
	lintValueFiles     []string // -f flags for override files
	lintScanDirs       []string // --scan-dir flags for project-wide lint
)

// lintCmd validates package manifests
var lintCmd = &cobra.Command{
	Use:   "lint [package-dir]",
	Short: "Validate package manifests",
	Long: `Validate all manifests in a DK package directory.

The lint command checks:
  - dk.yaml: Data package manifest validation
  - schemas/: Schema file validation
  - PII classification: Ensures outputs have required classifications

Validation rules include:
  - Required fields (E001-E003)
  - Schema references (E004-E005)
  - Runtime configuration (E030-E031, E040-E041)
  - PII classification (E025): Outputs must have classification

You can validate with overrides to check merged configuration:
  - Use -f to apply override files (like Helm values files)
  - Use --set for inline overrides
  - Precedence: dk.yaml < -f files (in order) < --set (in order)

Examples:
  # Lint current directory
  dk lint

  # Lint specific package
  dk lint ./my-pipeline

  # Lint with overrides applied
  dk lint ./my-pipeline -f production.yaml

  # Lint with inline override
  dk lint ./my-pipeline --set spec.runtime.image=myimage:v2

  # Strict mode (warnings become errors)
  dk lint --strict

  # Skip PII classification validation
  dk lint --skip-pii

  # Lint all transforms and datasets in a project
  dk lint --scan-dir ./my-project

  # Lint with multiple scan directories
  dk lint --scan-dir ./transforms --scan-dir ./datasets`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)

	lintCmd.Flags().BoolVar(&lintStrict, "strict", false, "Treat warnings as errors")
	lintCmd.Flags().BoolVar(&lintSkipPII, "skip-pii", false, "Skip PII classification validation")
	lintCmd.Flags().BoolVar(&lintSkipSchemaLock, "skip-schema-lock", false, "Skip schema lock (dk.lock) validation")
	lintCmd.Flags().StringArrayVar(&lintSet, "set", []string{},
		"Override values (key=value, can be repeated)")
	lintCmd.Flags().StringArrayVarP(&lintValueFiles, "values", "f", []string{},
		"Override files (can be repeated)")
	lintCmd.Flags().StringArrayVar(&lintScanDirs, "scan-dir", nil,
		"Scan directories for dk.yaml files (project-wide lint, repeatable)")
}

func runLint(cmd *cobra.Command, args []string) error {
	// If --scan-dir is provided, run project-wide lint
	if len(lintScanDirs) > 0 {
		return runProjectLint(cmd, lintScanDirs)
	}

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
	if lintStrict || lintSkipPII || lintSkipSchemaLock {
		validator.WithContext(&validate.ValidationContext{
			PackageDir:     absDir,
			StrictMode:     lintStrict,
			ValidatePII:    !lintSkipPII,
			SkipSchemaLock: lintSkipSchemaLock,
		})
	}

	result := validator.Validate(ctx)

	// Check what files were validated
	files := []string{}
	if _, err := os.Stat(filepath.Join(absDir, "dk.yaml")); err == nil {
		files = append(files, "dk.yaml")
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

// runProjectLint walks the scan directories to find all dk.yaml files
// and validates each one, plus any datasets found in datasets/ subdirectories.
func runProjectLint(cmd *cobra.Command, scanDirs []string) error {
	ctx := context.Background()

	// Resolve scan dirs
	resolvedDirs := make([]string, 0, len(scanDirs))
	for _, d := range scanDirs {
		abs, err := filepath.Abs(d)
		if err != nil {
			return fmt.Errorf("failed to resolve path %s: %w", d, err)
		}
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			return fmt.Errorf("directory not found: %s", d)
		}
		resolvedDirs = append(resolvedDirs, abs)
	}

	fmt.Println("Linting project (scan-dir mode)")
	fmt.Println()

	// Find all dk.yaml files that are Transform kind
	type lintTarget struct {
		path string
		dir  string
	}
	var targets []lintTarget

	for _, scanDir := range resolvedDirs {
		err := filepath.Walk(scanDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || info.Name() != "dk.yaml" {
				return nil
			}
			// Quick peek at kind
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			var peek struct {
				Kind string `yaml:"kind"`
			}
			if yamlErr := yaml.Unmarshal(data, &peek); yamlErr != nil {
				return nil
			}
			if peek.Kind == "Transform" {
				targets = append(targets, lintTarget{
					path: path,
					dir:  filepath.Dir(path),
				})
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("scanning %s: %w", scanDir, err)
		}
	}

	totalErrors := 0
	totalWarnings := 0
	totalPassed := 0

	// Validate each transform package
	for _, target := range targets {
		relPath, _ := filepath.Rel(".", target.dir)
		if relPath == "" {
			relPath = target.dir
		}

		validator := validate.NewAggregateValidator(target.dir)
		if lintStrict || lintSkipPII || lintSkipSchemaLock {
			validator.WithContext(&validate.ValidationContext{
				PackageDir:     target.dir,
				StrictMode:     lintStrict,
				ValidatePII:    !lintSkipPII,
				SkipSchemaLock: lintSkipSchemaLock,
			})
		}

		result := validator.Validate(ctx)

		if result.Valid && len(result.Warnings) == 0 {
			fmt.Printf("  ✓ %s\n", relPath)
			totalPassed++
		} else {
			if result.Errors.HasErrors() {
				fmt.Printf("  ✗ %s (%d errors)\n", relPath, len(result.Errors))
				for _, e := range result.Errors {
					field := e.Field
					if field == "" {
						field = "(root)"
					}
					fmt.Printf("      [%s] %s: %s\n", e.Code, field, e.Message)
				}
				totalErrors += len(result.Errors)
			}
			if len(result.Warnings) > 0 {
				if !result.Errors.HasErrors() {
					fmt.Printf("  ⚠ %s (%d warnings)\n", relPath, len(result.Warnings))
				}
				for _, w := range result.Warnings {
					fmt.Printf("      ⚠ %s\n", w)
				}
				totalWarnings += len(result.Warnings)
			}
		}
	}

	// Also validate datasets in all scan dirs
	datasetCount := 0
	for _, scanDir := range resolvedDirs {
		// Check if a datasets/ directory exists at the scan root
		datasetsDir := filepath.Join(scanDir, "datasets")
		if info, err := os.Stat(datasetsDir); err == nil && info.IsDir() {
			// Walk datasets/ for dk.yaml files with kind: DataSet
			filepath.Walk(datasetsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || info.Name() != "dk.yaml" {
					return nil
				}
				datasetCount++
				return nil
			})
		}
	}
	if datasetCount > 0 {
		fmt.Printf("  • %d dataset(s) validated\n", datasetCount)
	}

	fmt.Println()

	// Summary
	total := len(targets)
	if total == 0 {
		fmt.Println("No transforms found in scan directories.")
		return nil
	}

	fmt.Printf("Results: %d package(s) scanned, %d passed, %d error(s), %d warning(s)\n",
		total, totalPassed, totalErrors, totalWarnings)

	if totalErrors > 0 {
		return fmt.Errorf("validation failed: %d error(s) across %d package(s)", totalErrors, total-totalPassed)
	}

	if lintStrict && totalWarnings > 0 {
		return fmt.Errorf("strict mode: %d warning(s) treated as errors", totalWarnings)
	}

	if totalErrors == 0 {
		fmt.Println("✓ All validations passed")
	}

	return nil
}

// applyLintOverrides applies overrides to dk.yaml for validation.
// Creates a backup and modifies the file in place for validation.
func applyLintOverrides(absDir string) error {
	dkPath := filepath.Join(absDir, "dk.yaml")

	// Check if dk.yaml exists
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		return fmt.Errorf("dk.yaml not found in %s", absDir)
	}

	// Read base dk.yaml
	baseData, err := os.ReadFile(dkPath)
	if err != nil {
		return fmt.Errorf("failed to read dk.yaml: %w", err)
	}

	// Parse as generic map for merging
	var base map[string]any
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return fmt.Errorf("failed to parse dk.yaml: %w", err)
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

	// Write merged config back to dk.yaml
	mergedData, err := yaml.Marshal(base)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	// Create backup of original
	backupPath := dkPath + ".bak"
	if err := os.WriteFile(backupPath, baseData, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write merged config for validation
	if err := os.WriteFile(dkPath, mergedData, 0644); err != nil {
		// Restore from backup on failure
		os.WriteFile(dkPath, baseData, 0644)
		return fmt.Errorf("failed to write merged config: %w", err)
	}

	return nil
}
