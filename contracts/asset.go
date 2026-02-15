package contracts

import (
	"fmt"
	"strings"
)

// AssetType represents the type of asset.
type AssetType string

const (
	// AssetTypeSource is a data source asset (pulls data into the platform).
	AssetTypeSource AssetType = "source"

	// AssetTypeSink is a data sink asset (pushes data to an external destination).
	AssetTypeSink AssetType = "sink"

	// AssetTypeModelEngine is a model-engine asset (transforms data in-place).
	AssetTypeModelEngine AssetType = "model-engine"
)

// ValidAssetTypes returns all valid asset types.
func ValidAssetTypes() []AssetType {
	return []AssetType{
		AssetTypeSource,
		AssetTypeSink,
		AssetTypeModelEngine,
	}
}

// IsValid checks if the asset type is valid.
func (t AssetType) IsValid() bool {
	for _, valid := range ValidAssetTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// AssetManifest represents a parsed asset.yaml file.
type AssetManifest struct {
	// APIVersion is the schema version (e.g., "cdpp.io/v1alpha1").
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`

	// Kind is always "Asset".
	Kind string `json:"kind" yaml:"kind"`

	// Name is the unique asset identifier (DNS-safe, lowercase, 3-63 characters).
	Name string `json:"name" yaml:"name"`

	// Type is the asset type: source, sink, or model-engine.
	// Derived from the extension's kind segment.
	Type AssetType `json:"type" yaml:"type"`

	// Extension is the fully-qualified extension name (e.g., "cloudquery.source.aws").
	Extension string `json:"extension" yaml:"extension"`

	// Version is the extension version (semver, e.g., "v24.0.2").
	Version string `json:"version" yaml:"version"`

	// OwnerTeam is the team that owns this asset instance.
	OwnerTeam string `json:"ownerTeam" yaml:"ownerTeam"`

	// Description is an optional human-readable description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Binding is the optional name of the binding entry in bindings.yaml.
	Binding string `json:"binding,omitempty" yaml:"binding,omitempty"`

	// Config is the configuration block validated against the extension's schema.json.
	Config map[string]any `json:"config" yaml:"config"`

	// Labels are optional key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// Media type constants for extension packages.
const (
	// MediaTypeExtensionSchema is the media type for extension JSON Schema files.
	MediaTypeExtensionSchema = "application/vnd.infoblox.dp.extension.schema.v1+json"

	// MediaTypeExtensionPackage is the artifact type for extension packages.
	MediaTypeExtensionPackage = "application/vnd.infoblox.dp.extension.v1"
)

// ParseExtensionFQN parses a fully-qualified extension name into its components.
// FQN format: <vendor>.<kind>.<name> (e.g., "cloudquery.source.aws")
func ParseExtensionFQN(fqn string) (vendor, kind, name string, err error) {
	parts := strings.SplitN(fqn, ".", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid extension FQN %q: expected vendor.kind.name", fqn)
	}
	vendor, kind, name = parts[0], parts[1], parts[2]

	if vendor == "" {
		return "", "", "", fmt.Errorf("invalid extension FQN %q: vendor segment is empty", fqn)
	}
	if kind == "" {
		return "", "", "", fmt.Errorf("invalid extension FQN %q: kind segment is empty", fqn)
	}
	if name == "" {
		return "", "", "", fmt.Errorf("invalid extension FQN %q: name segment is empty", fqn)
	}

	// Validate kind is a known asset type
	assetType := AssetType(kind)
	if !assetType.IsValid() {
		return "", "", "", fmt.Errorf("invalid extension kind %q in FQN %q: must be one of source, sink, model-engine", kind, fqn)
	}

	return vendor, kind, name, nil
}

// AssetTypeFromFQN extracts the asset type from an extension FQN.
func AssetTypeFromFQN(fqn string) (AssetType, error) {
	_, kind, _, err := ParseExtensionFQN(fqn)
	if err != nil {
		return "", err
	}
	return AssetType(kind), nil
}

// AssetTypeDirName returns the directory name for the given asset type.
func AssetTypeDirName(t AssetType) string {
	switch t {
	case AssetTypeSource:
		return "sources"
	case AssetTypeSink:
		return "sinks"
	case AssetTypeModelEngine:
		return "models"
	default:
		return ""
	}
}
