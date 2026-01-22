package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/data-platform/sdk/validate"
	"github.com/spf13/cobra"
)

var (
	lintStrict  bool
	lintSkipPII bool
)

// lintCmd validates package manifests
var lintCmd = &cobra.Command{
	Use:   "lint [package-dir]",
	Short: "Validate package manifests",
	Long: `Validate all manifests in a DP package directory.

The lint command checks:
  - dp.yaml: Data package manifest validation
  - pipeline.yaml: Pipeline configuration validation
  - bindings.yaml: Binding configuration validation
  - schemas/: Schema file validation
  - PII classification: Ensures outputs have required classifications

Validation rules include:
  - Required fields (E001-E003)
  - Schema references (E004-E005)
  - Binding configuration (E010-E011)
  - Runtime configuration (E030-E031)
  - PII classification (E025): Outputs must have classification

Examples:
  # Lint current directory
  dp lint

  # Lint specific package
  dp lint ./my-pipeline

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

	fmt.Printf("Linting package: %s\n\n", packageDir)

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
