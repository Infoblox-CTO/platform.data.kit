package asset

import (
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

	// ProjectDir is the root directory of the data package project.
	ProjectDir string

	// Force overwrites an existing asset if true.
	Force bool

	// Store is the store name to pre-fill in the spec.
	Store string
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

// Scaffold creates a new asset.yaml in assets/<name>/asset.yaml.
func Scaffold(opts ScaffoldOpts) (*ScaffoldResult, error) {
	// Validate name
	if !dnsNamePattern.MatchString(opts.Name) {
		return nil, fmt.Errorf("invalid asset name %q: must match %s (DNS-safe, lowercase, 3-63 chars)",
			opts.Name, dnsNamePattern.String())
	}

	// Determine asset directory (flat layout: assets/<name>/)
	assetDir := filepath.Join(opts.ProjectDir, "assets", opts.Name)
	assetPath := filepath.Join(assetDir, "asset.yaml")

	if _, err := os.Stat(assetPath); err == nil && !opts.Force {
		return nil, fmt.Errorf("asset %q already exists at %s (use --force to overwrite)", opts.Name, assetPath)
	}

	// Also check for name uniqueness across all assets
	if !opts.Force {
		existing, _ := LoadAllAssets(opts.ProjectDir)
		for _, a := range existing {
			if a.Metadata.Name == opts.Name {
				return nil, fmt.Errorf("asset with name %q already exists (store: %s)", opts.Name, a.Spec.Store)
			}
		}
	}

	// Build the asset manifest
	store := opts.Store
	asset := &contracts.AssetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "Asset",
		Metadata: contracts.AssetMetadata{
			Name: opts.Name,
		},
		Spec: contracts.AssetSpec{
			Store: store,
		},
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

	b.WriteString("apiVersion: datakit.infoblox.dev/v1alpha1\n")
	b.WriteString("kind: Asset\n")

	// Metadata section
	b.WriteString("metadata:\n")
	b.WriteString(fmt.Sprintf("  name: %s\n", asset.Metadata.Name))
	if asset.Metadata.Namespace != "" {
		b.WriteString(fmt.Sprintf("  namespace: %s\n", asset.Metadata.Namespace))
	}
	if len(asset.Metadata.Labels) > 0 {
		b.WriteString("  labels:\n")
		keys := make([]string, 0, len(asset.Metadata.Labels))
		for k := range asset.Metadata.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("    %s: %s\n", k, asset.Metadata.Labels[k]))
		}
	}

	// Spec section
	b.WriteString("spec:\n")
	if asset.Spec.Store != "" {
		b.WriteString(fmt.Sprintf("  store: %s\n", asset.Spec.Store))
	} else {
		b.WriteString("  store: \"\"              # REQUIRED: set your store name\n")
	}
	if asset.Spec.Table != "" {
		b.WriteString(fmt.Sprintf("  table: %s\n", asset.Spec.Table))
	}
	if asset.Spec.Prefix != "" {
		b.WriteString(fmt.Sprintf("  prefix: %s\n", asset.Spec.Prefix))
	}
	if asset.Spec.Topic != "" {
		b.WriteString(fmt.Sprintf("  topic: %s\n", asset.Spec.Topic))
	}
	if asset.Spec.Format != "" {
		b.WriteString(fmt.Sprintf("  format: %s\n", asset.Spec.Format))
	}
	if asset.Spec.Classification != "" {
		b.WriteString(fmt.Sprintf("  classification: %s\n", asset.Spec.Classification))
	}
	if len(asset.Spec.Schema) > 0 {
		b.WriteString("  schema:\n")
		for _, field := range asset.Spec.Schema {
			b.WriteString(fmt.Sprintf("    - name: %s\n", field.Name))
			b.WriteString(fmt.Sprintf("      type: %s\n", field.Type))
			if field.PII {
				b.WriteString("      pii: true\n")
			}
			if field.From != "" {
				b.WriteString(fmt.Sprintf("      from: %s\n", field.From))
			}
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
