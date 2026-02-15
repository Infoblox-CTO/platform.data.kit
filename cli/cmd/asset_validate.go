package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/validate"
	"github.com/spf13/cobra"
)

var (
	assetValidateOffline bool
)

// assetValidateCmd validates asset.yaml files against their extension schemas.
var assetValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate asset configuration",
	Long: `Validate asset.yaml files against their extension JSON Schemas.

When given a path to a specific asset.yaml or asset directory, validates that
single asset. When no path is given, validates all assets under assets/ in the
current directory.

Validation checks:
  - Required fields: apiVersion, kind, name, type, extension, version, ownerTeam, config
  - Name format: DNS-safe (lowercase, 3-63 chars, starts with letter)
  - Extension FQN: valid vendor.kind.name format
  - Version: valid semver format
  - Type/kind match: asset type matches extension kind from FQN
  - Config schema: config block matches the extension's JSON Schema

Error codes:
  E070 — Required field missing
  E071 — Invalid extension FQN format
  E072 — Invalid version format
  E073 — Asset type does not match extension kind
  E074 — Config block fails schema validation
  E075 — Extension schema not found`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAssetValidate,
}

func init() {
	assetCmd.AddCommand(assetValidateCmd)

	assetValidateCmd.Flags().BoolVar(&assetValidateOffline, "offline", false,
		"Skip schema validation (structural checks only)")
}

func runAssetValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	w := cmd.OutOrStdout()

	// Determine what to validate
	var assetPath string
	if len(args) > 0 {
		assetPath = args[0]
	}

	// Set up validator
	var validator *validate.AssetValidator
	if assetValidateOffline {
		validator = validate.NewOfflineAssetValidator()
	} else {
		validator = validate.NewAssetValidator(asset.DefaultResolver())
	}

	// Determine if we're validating a single asset or all assets
	if assetPath != "" {
		return validateSingleAsset(ctx, w, validator, assetPath)
	}
	return validateAllAssets(ctx, w, validator)
}

func validateSingleAsset(ctx context.Context, w io.Writer, validator *validate.AssetValidator, path string) error {
	// Check if path is a directory (load asset.yaml from it) or a file
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path not found: %s", path)
	}

	var assetFile string
	if info.IsDir() {
		assetFile = filepath.Join(path, "asset.yaml")
	} else {
		assetFile = path
	}

	a, err := asset.LoadAsset(assetFile)
	if err != nil {
		return fmt.Errorf("failed to load asset: %w", err)
	}

	errs := validator.ValidateAsset(ctx, a)
	return reportAssetErrors(w, assetFile, errs)
}

func validateAllAssets(ctx context.Context, w io.Writer, validator *validate.AssetValidator) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	assets, err := asset.LoadAllAssets(cwd)
	if err != nil {
		return fmt.Errorf("failed to load assets: %w", err)
	}

	if len(assets) == 0 {
		fmt.Fprintln(w, "No assets found in assets/ directory.")
		return nil
	}

	hasErrors := false
	for _, a := range assets {
		errs := validator.ValidateAsset(ctx, a)
		displayPath := fmt.Sprintf("assets/%s/%s/asset.yaml",
			contracts.AssetTypeDirName(a.Type), a.Name)
		if err := reportAssetErrors(w, displayPath, errs); err != nil {
			hasErrors = true
		}
	}

	if hasErrors {
		return fmt.Errorf("asset validation failed")
	}

	fmt.Fprintf(w, "✓ All %d assets are valid.\n", len(assets))
	return nil
}

func reportAssetErrors(w io.Writer, path string, errs contracts.ValidationErrors) error {
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
