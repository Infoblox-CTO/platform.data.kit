package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/validate"
	"github.com/spf13/cobra"
)

var (
	datasetValidateOffline bool
)

// datasetValidateCmd validates dataset.yaml files.
var datasetValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate dataset configuration",
	Long: `Validate dataset.yaml files against structural rules.

When given a path to a specific dataset.yaml or dataset directory, validates that
single dataset. When no path is given, validates all datasets under datasets/ in the
current directory.

Validation checks:
  - Required fields: apiVersion, kind, name, store
  - Name format: DNS-safe (lowercase, 3-63 chars, starts with letter)
  - Classification: one of public, internal, confidential, restricted
  - Schema fields: unique names, types required

Error codes:
  E070 — Required field missing
  E076 — DataSet reference not found`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDataSetValidate,
}

func init() {
	datasetCmd.AddCommand(datasetValidateCmd)

	datasetValidateCmd.Flags().BoolVar(&datasetValidateOffline, "offline", false,
		"Skip online validation (structural checks only)")
}

func runDataSetValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := cmd.OutOrStdout()

	// Determine what to validate
	var datasetPath string
	if len(args) > 0 {
		datasetPath = args[0]
	}

	// Set up validator
	var validator *validate.DataSetValidator
	if datasetValidateOffline {
		validator = validate.NewOfflineDataSetValidator()
	} else {
		validator = validate.NewDataSetValidator()
	}

	// Determine if we're validating a single dataset or all datasets
	if datasetPath != "" {
		return validateSingleDataSet(ctx, w, validator, datasetPath)
	}
	return validateAllDataSets(ctx, w, validator)
}

func validateSingleDataSet(ctx context.Context, w io.Writer, validator *validate.DataSetValidator, path string) error {
	// Check if path is a directory (load dataset.yaml from it) or a file
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path not found: %s", path)
	}

	var datasetFile string
	if info.IsDir() {
		datasetFile = filepath.Join(path, "dataset.yaml")
	} else {
		datasetFile = path
	}

	a, err := dataset.LoadDataSet(datasetFile)
	if err != nil {
		return fmt.Errorf("failed to load dataset: %w", err)
	}

	errs := validator.ValidateDataSet(ctx, a)
	return reportDataSetErrors(w, datasetFile, errs)
}

func validateAllDataSets(ctx context.Context, w io.Writer, validator *validate.DataSetValidator) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	datasets, err := dataset.LoadAllDataSets(cwd)
	if err != nil {
		return fmt.Errorf("failed to load datasets: %w", err)
	}

	if len(datasets) == 0 {
		fmt.Fprintln(w, "No datasets found in datasets/ directory.")
		return nil
	}

	hasErrors := false
	for _, a := range datasets {
		errs := validator.ValidateDataSet(ctx, a)
		displayPath := fmt.Sprintf("datasets/%s/dataset.yaml", a.Metadata.Name)
		if err := reportDataSetErrors(w, displayPath, errs); err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("dataset validation failed")
	}

	fmt.Fprintf(w, "✓ All %d datasets are valid.\n", len(datasets))
	return nil
}

func reportDataSetErrors(w io.Writer, path string, errs contracts.ValidationErrors) error {
	if len(errs) == 0 {
		fmt.Fprintf(w, "✓ %s is valid.\n", path)
		return nil
	}

	fmt.Fprintf(w, "✗ %s has validation errors:\n", path)
	for _, e := range errs {
		severity := "ERROR"
		if e.Severity == contracts.SeverityWarning {
			severity = "WARN "
		}
		if e.Field != "" {
			fmt.Fprintf(w, "  [%s] %s %s: %s\n", e.Code, severity, e.Field, e.Message)
		} else {
			fmt.Fprintf(w, "  [%s] %s %s\n", e.Code, severity, e.Message)
		}
	}

	return fmt.Errorf("validation failed for %s", path)
}
