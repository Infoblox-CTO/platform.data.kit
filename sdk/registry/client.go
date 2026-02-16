// Package registry provides OCI registry integration for DP packages.
package registry

import (
	"context"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// Client defines the interface for OCI registry operations.
type Client interface {
	// Push pushes a package artifact to the registry.
	Push(ctx context.Context, ref string, artifact *Artifact) (*PushResult, error)

	// Pull pulls a package artifact from the registry.
	Pull(ctx context.Context, ref string) (*Artifact, error)

	// Resolve resolves a reference to a digest.
	Resolve(ctx context.Context, ref string) (string, error)

	// Exists checks if a reference exists in the registry.
	Exists(ctx context.Context, ref string) (bool, error)

	// Tags lists all tags for a repository.
	Tags(ctx context.Context, repository string) ([]string, error)

	// Delete removes an artifact from the registry.
	Delete(ctx context.Context, ref string) error
}

// Artifact represents a packaged DP artifact ready for push/pull.
type Artifact struct {
	// Manifest is the OCI manifest for this artifact.
	Manifest *ArtifactManifest

	// Layers contains the artifact content layers.
	Layers []Layer

	// Config is the artifact configuration.
	Config *ArtifactConfig
}

// ArtifactManifest represents the OCI manifest structure.
type ArtifactManifest struct {
	// MediaType is the manifest media type.
	MediaType string `json:"mediaType"`

	// SchemaVersion is the OCI schema version.
	SchemaVersion int `json:"schemaVersion"`

	// ArtifactType is the DP artifact type.
	ArtifactType string `json:"artifactType,omitempty"`

	// Config is the config descriptor.
	Config Descriptor `json:"config"`

	// Layers are the layer descriptors.
	Layers []Descriptor `json:"layers"`

	// Annotations are the manifest annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Descriptor describes a content blob.
type Descriptor struct {
	// MediaType is the blob media type.
	MediaType string `json:"mediaType"`

	// Digest is the content digest.
	Digest string `json:"digest"`

	// Size is the blob size in bytes.
	Size int64 `json:"size"`

	// Annotations are descriptor annotations.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Layer represents a content layer in the artifact.
type Layer struct {
	// MediaType is the layer media type.
	MediaType string

	// Content is the layer content.
	Content []byte

	// Annotations are layer annotations.
	Annotations map[string]string
}

// ArtifactConfig contains metadata about the artifact.
type ArtifactConfig struct {
	// Manifest is the parsed manifest (Source, Destination, or Model).
	Manifest interface{} `json:"manifest"`

	// Kind is the manifest kind.
	Kind contracts.Kind `json:"kind"`

	// BuildInfo contains build metadata.
	BuildInfo *BuildInfo `json:"buildInfo"`
}

// BuildInfo contains build-time metadata.
type BuildInfo struct {
	// Timestamp is when the artifact was built.
	Timestamp string `json:"timestamp"`

	// Builder is the build tool version.
	Builder string `json:"builder"`

	// GitCommit is the source commit SHA.
	GitCommit string `json:"gitCommit,omitempty"`

	// GitBranch is the source branch.
	GitBranch string `json:"gitBranch,omitempty"`

	// GitTag is the source tag.
	GitTag string `json:"gitTag,omitempty"`

	// Host is the build host.
	Host string `json:"host,omitempty"`
}

// PushResult contains the result of a push operation.
type PushResult struct {
	// Reference is the full reference that was pushed.
	Reference string `json:"reference"`

	// Digest is the manifest digest.
	Digest string `json:"digest"`

	// Size is the total artifact size.
	Size int64 `json:"size"`
}

// ClientConfig contains configuration for the OCI client.
type ClientConfig struct {
	// Registry is the registry hostname.
	Registry string

	// Insecure allows insecure registry connections.
	Insecure bool

	// PlainHTTP uses plain HTTP instead of HTTPS.
	PlainHTTP bool

	// Username for authentication.
	Username string

	// Password for authentication.
	Password string

	// Token for token-based authentication.
	Token string
}

// Media types for DP artifacts.
const (
	// MediaTypeDPManifest is the media type for DP manifests.
	MediaTypeDPManifest = "application/vnd.infoblox.dp.manifest.v1+yaml"

	// MediaTypeDPConfig is the media type for DP artifact config.
	MediaTypeDPConfig = "application/vnd.infoblox.dp.config.v1+json"

	// MediaTypeDPPackage is the artifact type for DP packages.
	MediaTypeDPPackage = "application/vnd.infoblox.dp.package.v1"

	// MediaTypeTarGz is the media type for tar.gz archives.
	MediaTypeTarGz = "application/vnd.oci.image.layer.v1.tar+gzip"
)
