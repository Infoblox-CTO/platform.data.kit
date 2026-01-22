// Package contracts defines the core types for DP data packages.
package contracts

// APIVersion represents the API version of a manifest.
type APIVersion string

const (
	// APIVersionV1Alpha1 is the initial development version.
	APIVersionV1Alpha1 APIVersion = "data.infoblox.com/v1alpha1"

	// APIVersionV1Beta1 is the beta version.
	APIVersionV1Beta1 APIVersion = "data.infoblox.com/v1beta1"

	// APIVersionV1 is the stable version.
	APIVersionV1 APIVersion = "data.infoblox.com/v1"
)

// PackageVersion represents a semantic version.
type PackageVersion struct {
	Major int    `json:"major" yaml:"major"`
	Minor int    `json:"minor" yaml:"minor"`
	Patch int    `json:"patch" yaml:"patch"`
	Pre   string `json:"pre,omitempty" yaml:"pre,omitempty"`
	Build string `json:"build,omitempty" yaml:"build,omitempty"`
}

// String returns the version as a string.
func (v PackageVersion) String() string {
	s := ""
	if v.Major > 0 || v.Minor > 0 || v.Patch > 0 {
		s = string(rune('0'+v.Major)) + "." + string(rune('0'+v.Minor)) + "." + string(rune('0'+v.Patch))
	}
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	if v.Build != "" {
		s += "+" + v.Build
	}
	return s
}

// ArtifactRef references a specific version of an artifact.
type ArtifactRef struct {
	// Name is the artifact name.
	Name string `json:"name" yaml:"name"`

	// Version is the artifact version.
	Version string `json:"version" yaml:"version"`

	// Registry is the OCI registry URL.
	Registry string `json:"registry,omitempty" yaml:"registry,omitempty"`

	// Digest is the content digest for verification.
	Digest string `json:"digest,omitempty" yaml:"digest,omitempty"`
}
