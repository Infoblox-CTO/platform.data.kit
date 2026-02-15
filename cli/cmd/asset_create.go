package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	"github.com/spf13/cobra"
)

var (
	assetCreateExt         string
	assetCreateForce       bool
	assetCreateInteractive bool
	assetCreateVersion     string
)

// assetCreateCmd scaffolds a new asset from an extension.
var assetCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new asset from an extension",
	Long: `Scaffold a new asset.yaml from an extension's JSON Schema.

The asset is created in assets/<type>/<name>/asset.yaml with placeholder
config values derived from the extension schema's required fields.

Examples:
  # Create a source asset from CloudQuery AWS extension
  dp asset create aws-security --ext cloudquery.source.aws

  # Create with specific version
  dp asset create aws-security --ext cloudquery.source.aws --version v24.0.2

  # Overwrite existing asset
  dp asset create aws-security --ext cloudquery.source.aws --force

  # Interactive mode — prompted for each required config field
  dp asset create aws-security --ext cloudquery.source.aws --interactive`,
	Args: cobra.ExactArgs(1),
	RunE: runAssetCreate,
}

func init() {
	assetCmd.AddCommand(assetCreateCmd)

	assetCreateCmd.Flags().StringVar(&assetCreateExt, "ext", "",
		"Extension FQN (required, e.g., cloudquery.source.aws)")
	assetCreateCmd.Flags().BoolVar(&assetCreateForce, "force", false,
		"Overwrite existing asset")
	assetCreateCmd.Flags().BoolVarP(&assetCreateInteractive, "interactive", "i", false,
		"Prompt for each required config field")
	assetCreateCmd.Flags().StringVar(&assetCreateVersion, "version", "",
		"Extension version (default: latest known)")

	_ = assetCreateCmd.MarkFlagRequired("ext")
}

func runAssetCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if err := asset.ValidateAssetName(name); err != nil {
		return err
	}

	// Determine project directory
	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Build scaffold options
	opts := asset.ScaffoldOpts{
		Name:         name,
		ExtensionFQN: assetCreateExt,
		ProjectDir:   projectDir,
		Force:        assetCreateForce,
		Version:      assetCreateVersion,
	}

	// Interactive mode: resolve schema and prompt for fields
	if assetCreateInteractive {
		config, err := runInteractiveConfig(cmd, opts)
		if err != nil {
			return fmt.Errorf("interactive mode failed: %w", err)
		}
		opts.InteractiveConfig = config
	}

	result, err := asset.Scaffold(opts)
	if err != nil {
		return err
	}

	relPath, _ := filepath.Rel(projectDir, result.AssetPath)
	if relPath == "" {
		relPath = result.AssetPath
	}

	cmd.Printf("✓ Created asset %q at %s\n", name, relPath)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit %s to configure your asset\n", relPath)
	cmd.Printf("  2. Set ownerTeam to your team name\n")
	cmd.Printf("  3. Run 'dp asset validate' to validate the config\n")
	cmd.Printf("  4. Add '%s' to the assets section in dp.yaml\n", name)

	return nil
}

// runInteractiveConfig prompts the user for each required config field.
func runInteractiveConfig(cmd *cobra.Command, opts asset.ScaffoldOpts) (map[string]any, error) {
	// Resolve the schema
	resolver := opts.Resolver
	if resolver == nil {
		resolver = asset.DefaultResolver()
	}

	ctx := context.Background()
	schemaBytes, err := resolver.ResolveSchema(ctx, opts.ExtensionFQN, opts.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve extension schema: %w", err)
	}

	fields, err := asset.ExtractSchemaFields(schemaBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to extract schema fields: %w", err)
	}

	config := make(map[string]any)
	scanner := bufio.NewScanner(os.Stdin)

	for _, field := range fields {
		if !field.Required {
			continue
		}

		prompt := fmt.Sprintf("%s", field.Name)
		if field.Description != "" {
			prompt += fmt.Sprintf(" (%s)", field.Description)
		}
		if len(field.Enum) > 0 {
			prompt += fmt.Sprintf(" [%s]", strings.Join(field.Enum, ", "))
		}

		label := "required"
		if field.Default != nil {
			label = fmt.Sprintf("default: %v", field.Default)
		}

		cmd.Printf("  %s [%s]: ", prompt, label)

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "" && field.Default != nil {
			config[field.Name] = field.Default
		} else if input != "" {
			config[field.Name] = parseInteractiveValue(field.Type, input)
		} else {
			// Empty input for required field with no default
			config[field.Name] = placeholderForFieldType(field.Type)
		}
	}

	return config, nil
}

// parseInteractiveValue parses a user-provided string into the appropriate type.
func parseInteractiveValue(fieldType, input string) any {
	switch fieldType {
	case "array":
		// Split comma-separated values
		parts := strings.Split(input, ",")
		result := make([]any, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	case "integer":
		// Try to parse as int
		var n int
		if _, err := fmt.Sscanf(input, "%d", &n); err == nil {
			return n
		}
		return input
	case "boolean":
		return strings.EqualFold(input, "true") || strings.EqualFold(input, "yes") || input == "1"
	default:
		return input
	}
}

// placeholderForFieldType returns a zero-value placeholder based on the field type.
func placeholderForFieldType(fieldType string) any {
	switch fieldType {
	case "array":
		return []any{}
	case "object":
		return map[string]any{}
	case "integer", "number":
		return 0
	case "boolean":
		return false
	default:
		return ""
	}
}
