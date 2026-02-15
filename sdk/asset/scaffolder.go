package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// dnsNamePattern validates DNS-safe asset names.
var dnsNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,62}$`)

// ScaffoldOpts contains options for scaffolding a new asset.
type ScaffoldOpts struct {
	// Name is the asset name (DNS-safe).
	Name string

	// ExtensionFQN is the fully-qualified extension name (vendor.kind.name).
	ExtensionFQN string

	// ProjectDir is the root directory of the data package project.
	ProjectDir string

	// Force overwrites an existing asset if true.
	Force bool

	// Resolver is used to fetch the extension schema for config placeholders.
	// If nil, the default resolver is used.
	Resolver SchemaResolver

	// Version overrides the extension version. If empty, uses "latest" or "v0.0.0".
	Version string

	// InteractiveConfig is a pre-filled config map (from interactive mode).
	// When set, these values are used instead of schema-derived placeholders.
	InteractiveConfig map[string]any
}

// ScaffoldResult contains the result of scaffolding an asset.
type ScaffoldResult struct {
	// AssetPath is the path to the created asset.yaml file.
	AssetPath string

	// AssetDir is the directory containing the asset.yaml file.
	AssetDir string

	// Asset is the scaffolded asset manifest.
	Asset *contracts.AssetManifest
}

// Scaffold creates a new asset.yaml from an extension schema.
func Scaffold(opts ScaffoldOpts) (*ScaffoldResult, error) {
	// Validate name
	if !dnsNamePattern.MatchString(opts.Name) {
		return nil, fmt.Errorf("invalid asset name %q: must match %s (DNS-safe, lowercase, 3-63 chars)",
			opts.Name, dnsNamePattern.String())
	}

	// Parse extension FQN
	vendor, kind, extName, err := contracts.ParseExtensionFQN(opts.ExtensionFQN)
	if err != nil {
		return nil, fmt.Errorf("invalid extension FQN: %w", err)
	}
	_ = vendor
	_ = extName

	assetType := contracts.AssetType(kind)

	// Determine version
	version := opts.Version
	if version == "" {
		version = "v0.0.0"
	}

	// Determine asset directory and check for duplicates
	assetDir := AssetDir(opts.ProjectDir, assetType, opts.Name)
	assetPath := filepath.Join(assetDir, "asset.yaml")

	if _, err := os.Stat(assetPath); err == nil && !opts.Force {
		return nil, fmt.Errorf("asset %q already exists at %s (use --force to overwrite)", opts.Name, assetPath)
	}

	// Also check for name uniqueness across all asset types
	if !opts.Force {
		existing, _ := LoadAllAssets(opts.ProjectDir)
		for _, a := range existing {
			if a.Name == opts.Name {
				return nil, fmt.Errorf("asset with name %q already exists (type: %s)", opts.Name, a.Type)
			}
		}
	}

	// Resolve config placeholders from extension schema
	config := opts.InteractiveConfig
	if config == nil {
		config, err = resolveConfigPlaceholders(opts)
		if err != nil {
			// Non-fatal: use empty config if schema resolution fails
			config = map[string]any{}
		}
	}

	// Build the asset manifest
	asset := &contracts.AssetManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Asset",
		Name:       opts.Name,
		Type:       assetType,
		Extension:  opts.ExtensionFQN,
		Version:    version,
		OwnerTeam:  "",
		Config:     config,
	}

	// Create directory and write asset.yaml
	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create asset directory: %w", err)
	}

	yamlData, err := marshalAssetWithComments(asset)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal asset: %w", err)
	}

	if err := os.WriteFile(assetPath, yamlData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write asset.yaml: %w", err)
	}

	return &ScaffoldResult{
		AssetPath: assetPath,
		AssetDir:  assetDir,
		Asset:     asset,
	}, nil
}

// resolveConfigPlaceholders builds a config map with placeholder values
// derived from the extension's JSON Schema required fields.
func resolveConfigPlaceholders(opts ScaffoldOpts) (map[string]any, error) {
	resolver := opts.Resolver
	if resolver == nil {
		resolver = DefaultResolver()
	}

	ctx := context.Background()
	schemaBytes, err := resolver.ResolveSchema(ctx, opts.ExtensionFQN, opts.Version)
	if err != nil {
		return nil, err
	}

	return extractConfigFromSchema(schemaBytes)
}

// extractConfigFromSchema parses a JSON Schema and returns a config map
// with placeholder values for required fields.
func extractConfigFromSchema(schemaBytes []byte) (map[string]any, error) {
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse extension schema: %w", err)
	}

	config := make(map[string]any)

	// Get required fields
	required := make(map[string]bool)
	if reqList, ok := schema["required"].([]any); ok {
		for _, r := range reqList {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	// Get properties
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return config, nil
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Generate placeholder values for required properties first, then optional
	for _, key := range keys {
		prop, ok := properties[key].(map[string]any)
		if !ok {
			continue
		}

		// Only include required fields by default (with REQUIRED comment)
		if !required[key] {
			continue
		}

		config[key] = placeholderForType(prop)
	}

	return config, nil
}

// placeholderForType returns a zero-value placeholder based on the JSON Schema type.
func placeholderForType(prop map[string]any) any {
	propType, _ := prop["type"].(string)

	switch propType {
	case "string":
		if enumVals, ok := prop["enum"].([]any); ok && len(enumVals) > 0 {
			return enumVals[0]
		}
		return ""
	case "integer", "number":
		if def, ok := prop["default"]; ok {
			return def
		}
		return 0
	case "boolean":
		if def, ok := prop["default"]; ok {
			return def
		}
		return false
	case "array":
		return []any{}
	case "object":
		return map[string]any{}
	default:
		return nil
	}
}

// SchemaFieldInfo contains metadata about a schema field for interactive mode.
type SchemaFieldInfo struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     any
	Enum        []string
}

// ExtractSchemaFields extracts field metadata from a JSON Schema for interactive prompting.
func ExtractSchemaFields(schemaBytes []byte) ([]SchemaFieldInfo, error) {
	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	required := make(map[string]bool)
	if reqList, ok := schema["required"].([]any); ok {
		for _, r := range reqList {
			if s, ok := r.(string); ok {
				required[s] = true
			}
		}
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil, nil
	}

	var fields []SchemaFieldInfo

	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		prop, ok := properties[key].(map[string]any)
		if !ok {
			continue
		}

		field := SchemaFieldInfo{
			Name:     key,
			Required: required[key],
		}

		if t, ok := prop["type"].(string); ok {
			field.Type = t
		}
		if d, ok := prop["description"].(string); ok {
			field.Description = d
		}
		if def, ok := prop["default"]; ok {
			field.Default = def
		}
		if enumVals, ok := prop["enum"].([]any); ok {
			for _, v := range enumVals {
				if s, ok := v.(string); ok {
					field.Enum = append(field.Enum, s)
				}
			}
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// marshalAssetWithComments creates a YAML representation with helpful inline comments.
func marshalAssetWithComments(asset *contracts.AssetManifest) ([]byte, error) {
	var b strings.Builder

	b.WriteString("apiVersion: data.infoblox.com/v1alpha1\n")
	b.WriteString("kind: Asset\n")
	b.WriteString(fmt.Sprintf("name: %s\n", asset.Name))
	b.WriteString(fmt.Sprintf("type: %s\n", asset.Type))
	b.WriteString(fmt.Sprintf("extension: %s\n", asset.Extension))
	b.WriteString(fmt.Sprintf("version: %s\n", asset.Version))

	if asset.OwnerTeam != "" {
		b.WriteString(fmt.Sprintf("ownerTeam: %s\n", asset.OwnerTeam))
	} else {
		b.WriteString("ownerTeam: \"\"          # REQUIRED: set your team name\n")
	}

	if asset.Description != "" {
		b.WriteString(fmt.Sprintf("description: %q\n", asset.Description))
	}

	if asset.Binding != "" {
		b.WriteString(fmt.Sprintf("binding: %s\n", asset.Binding))
	}

	// Write config section
	b.WriteString("config:\n")
	if len(asset.Config) > 0 {
		// Sort keys for deterministic output
		keys := make([]string, 0, len(asset.Config))
		for k := range asset.Config {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := asset.Config[k]
			writeConfigValue(&b, k, v, 2)
		}
	} else {
		b.WriteString("  {}  # Configure extension-specific settings\n")
	}

	if len(asset.Labels) > 0 {
		b.WriteString("labels:\n")
		keys := make([]string, 0, len(asset.Labels))
		for k := range asset.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, asset.Labels[k]))
		}
	}

	return []byte(b.String()), nil
}

// writeConfigValue writes a YAML config key-value pair with proper indentation.
func writeConfigValue(b *strings.Builder, key string, value any, indent int) {
	prefix := strings.Repeat(" ", indent)

	switch v := value.(type) {
	case []any:
		if len(v) == 0 {
			b.WriteString(fmt.Sprintf("%s%s: []\n", prefix, key))
		} else {
			b.WriteString(fmt.Sprintf("%s%s:\n", prefix, key))
			for _, item := range v {
				// Use yaml.Marshal for complex items
				data, err := yaml.Marshal(item)
				if err != nil {
					b.WriteString(fmt.Sprintf("%s  - %v\n", prefix, item))
				} else {
					b.WriteString(fmt.Sprintf("%s  - %s", prefix, strings.TrimSpace(string(data))))
					b.WriteString("\n")
				}
			}
		}
	case map[string]any:
		if len(v) == 0 {
			b.WriteString(fmt.Sprintf("%s%s: {}\n", prefix, key))
		} else {
			b.WriteString(fmt.Sprintf("%s%s:\n", prefix, key))
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range v {
				writeConfigValue(b, fmt.Sprintf("%v", k), v[fmt.Sprintf("%v", k)], indent+2)
			}
			_ = keys
		}
	case string:
		if v == "" {
			b.WriteString(fmt.Sprintf("%s%s: \"\"\n", prefix, key))
		} else {
			b.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, key, v))
		}
	default:
		data, err := yaml.Marshal(v)
		if err != nil {
			b.WriteString(fmt.Sprintf("%s%s: %v\n", prefix, key, v))
		} else {
			b.WriteString(fmt.Sprintf("%s%s: %s", prefix, key, strings.TrimSpace(string(data))))
			b.WriteString("\n")
		}
	}
}

// ValidateAssetName checks if a name is a valid DNS-safe asset name.
func ValidateAssetName(name string) error {
	if !dnsNamePattern.MatchString(name) {
		return fmt.Errorf("invalid asset name %q: must be lowercase, start with a letter, 3-63 characters, and contain only letters, digits, and hyphens",
			name)
	}
	return nil
}
