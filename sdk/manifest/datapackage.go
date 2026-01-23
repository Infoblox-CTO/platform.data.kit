package manifest

import (
	"fmt"

	"github.com/Infoblox-CTO/data-platform/contracts"
	"gopkg.in/yaml.v3"
)

// DataPackageFromBytes parses a dp.yaml file content into a DataPackage.
func DataPackageFromBytes(data []byte) (*contracts.DataPackage, error) {
	var pkg contracts.DataPackage
	if err := yaml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse DataPackage: %w", err)
	}

	if pkg.Kind != "DataPackage" {
		return nil, fmt.Errorf("expected kind 'DataPackage', got '%s'", pkg.Kind)
	}

	return &pkg, nil
}

// DataPackageToBytes serializes a DataPackage to YAML bytes.
func DataPackageToBytes(pkg *contracts.DataPackage) ([]byte, error) {
	return yaml.Marshal(pkg)
}

// ValidateDataPackageVersion checks if the API version is supported.
func ValidateDataPackageVersion(version string) error {
	switch contracts.APIVersion(version) {
	case contracts.APIVersionV1Alpha1, contracts.APIVersionV1Beta1, contracts.APIVersionV1:
		return nil
	default:
		return fmt.Errorf("unsupported API version: %s", version)
	}
}

// ValidateDataPackageRuntime validates that the DataPackage has a valid runtime section.
// Returns an error if runtime is missing or has required fields missing.
func ValidateDataPackageRuntime(pkg *contracts.DataPackage) error {
	if pkg == nil {
		return fmt.Errorf("DataPackage is nil")
	}

	if pkg.Spec.Runtime == nil {
		return fmt.Errorf("spec.runtime is required: DataPackage must include runtime configuration")
	}

	if pkg.Spec.Runtime.Image == "" {
		return fmt.Errorf("spec.runtime.image is required: container image must be specified")
	}

	return nil
}

// HasRuntimeSection returns true if the DataPackage has a runtime section defined.
func HasRuntimeSection(pkg *contracts.DataPackage) bool {
	return pkg != nil && pkg.Spec.Runtime != nil
}

// GetRuntimeImage returns the runtime image or empty string if not set.
func GetRuntimeImage(pkg *contracts.DataPackage) string {
	if pkg == nil || pkg.Spec.Runtime == nil {
		return ""
	}
	return pkg.Spec.Runtime.Image
}
