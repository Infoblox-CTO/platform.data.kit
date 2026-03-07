package registry

import (
	"context"
	"testing"
)

func TestClientInterface(t *testing.T) {
	var _ Client = (*mockClient)(nil)
}

type mockClient struct {
	pushFunc    func(ctx context.Context, ref string, artifact *Artifact) (*PushResult, error)
	pullFunc    func(ctx context.Context, ref string) (*Artifact, error)
	resolveFunc func(ctx context.Context, ref string) (string, error)
	existsFunc  func(ctx context.Context, ref string) (bool, error)
	tagsFunc    func(ctx context.Context, repository string) ([]string, error)
	deleteFunc  func(ctx context.Context, ref string) error
}

func (m *mockClient) Push(ctx context.Context, ref string, artifact *Artifact) (*PushResult, error) {
	if m.pushFunc != nil {
		return m.pushFunc(ctx, ref, artifact)
	}
	return &PushResult{Digest: "sha256:test"}, nil
}

func (m *mockClient) Pull(ctx context.Context, ref string) (*Artifact, error) {
	if m.pullFunc != nil {
		return m.pullFunc(ctx, ref)
	}
	return &Artifact{}, nil
}

func (m *mockClient) Resolve(ctx context.Context, ref string) (string, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, ref)
	}
	return "sha256:test", nil
}

func (m *mockClient) Exists(ctx context.Context, ref string) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, ref)
	}
	return true, nil
}

func (m *mockClient) Tags(ctx context.Context, repository string) ([]string, error) {
	if m.tagsFunc != nil {
		return m.tagsFunc(ctx, repository)
	}
	return []string{"v1.0.0", "latest"}, nil
}

func (m *mockClient) Delete(ctx context.Context, ref string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, ref)
	}
	return nil
}

func TestArtifact(t *testing.T) {
	tests := []struct {
		name     string
		artifact *Artifact
	}{
		{
			name: "empty artifact",
			artifact: &Artifact{
				Manifest: nil,
				Layers:   nil,
				Config:   nil,
			},
		},
		{
			name: "artifact with manifest",
			artifact: &Artifact{
				Manifest: &ArtifactManifest{
					MediaType:     "application/vnd.oci.image.manifest.v1+json",
					SchemaVersion: 2,
					ArtifactType:  "application/vnd.dk.package",
				},
				Layers: []Layer{},
				Config: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.artifact == nil {
				t.Error("artifact should not be nil")
			}
		})
	}
}

func TestArtifactManifest(t *testing.T) {
	manifest := &ArtifactManifest{
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		SchemaVersion: 2,
		ArtifactType:  "application/vnd.dk.package",
		Config: Descriptor{
			MediaType: "application/vnd.dk.config+json",
			Digest:    "sha256:abc123",
			Size:      1024,
		},
		Layers: []Descriptor{
			{
				MediaType: "application/vnd.dk.manifest+tar.gzip",
				Digest:    "sha256:def456",
				Size:      2048,
			},
		},
		Annotations: map[string]string{
			"org.opencontainers.image.title": "test-package",
		},
	}

	if manifest.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", manifest.SchemaVersion)
	}
	if len(manifest.Layers) != 1 {
		t.Errorf("Layers count = %d, want 1", len(manifest.Layers))
	}
}

func TestDescriptor(t *testing.T) {
	tests := []struct {
		name       string
		descriptor Descriptor
		wantSize   int64
	}{
		{
			name: "basic descriptor",
			descriptor: Descriptor{
				MediaType: "application/octet-stream",
				Digest:    "sha256:abc123",
				Size:      1024,
			},
			wantSize: 1024,
		},
		{
			name: "descriptor with annotations",
			descriptor: Descriptor{
				MediaType: "application/octet-stream",
				Digest:    "sha256:def456",
				Size:      2048,
				Annotations: map[string]string{
					"filename": "data.tar.gz",
				},
			},
			wantSize: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.descriptor.Size != tt.wantSize {
				t.Errorf("Size = %d, want %d", tt.descriptor.Size, tt.wantSize)
			}
		})
	}
}

func TestPushResult(t *testing.T) {
	result := &PushResult{
		Digest:    "sha256:abc123def456",
		Size:      4096,
		Reference: "ghcr.io/org/pkg:v1.0.0",
	}

	if result.Digest == "" {
		t.Error("Digest should not be empty")
	}
	if result.Size <= 0 {
		t.Error("Size should be positive")
	}
}

func TestLayer(t *testing.T) {
	layer := Layer{
		MediaType: MediaTypeTarGz,
		Content:   []byte("test content"),
		Annotations: map[string]string{
			"filename": "content.tar.gz",
		},
	}

	if layer.MediaType != MediaTypeTarGz {
		t.Errorf("MediaType = %s, want %s", layer.MediaType, MediaTypeTarGz)
	}
	if len(layer.Content) == 0 {
		t.Error("Content should not be empty")
	}
}

func TestArtifactConfig(t *testing.T) {
	config := &ArtifactConfig{
		BuildInfo: &BuildInfo{
			Timestamp: "2024-01-01T00:00:00Z",
			Builder:   "dk-cli-v1.0.0",
			GitCommit: "abc123",
		},
	}

	if config.BuildInfo == nil {
		t.Error("BuildInfo should not be nil")
	}
	if config.BuildInfo.Builder == "" {
		t.Error("Builder should not be empty")
	}
}

func TestBuildInfo(t *testing.T) {
	info := &BuildInfo{
		Timestamp: "2024-01-01T00:00:00Z",
		Builder:   "dk-cli-v1.0.0",
		GitCommit: "abc123def456",
		GitBranch: "main",
		GitTag:    "v1.0.0",
		Host:      "build-host",
	}

	if info.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if info.Builder == "" {
		t.Error("Builder should not be empty")
	}
}

func TestClientConfig(t *testing.T) {
	config := ClientConfig{
		Registry:  "ghcr.io",
		Insecure:  false,
		PlainHTTP: false,
		Username:  "user",
		Password:  "pass",
	}

	if config.Registry == "" {
		t.Error("Registry should not be empty")
	}
}

func TestMediaTypes(t *testing.T) {
	if MediaTypeDKManifest == "" {
		t.Error("MediaTypeDKManifest should not be empty")
	}
	if MediaTypeDKConfig == "" {
		t.Error("MediaTypeDKConfig should not be empty")
	}
	if MediaTypeDKPackage == "" {
		t.Error("MediaTypeDKPackage should not be empty")
	}
	if MediaTypeTarGz == "" {
		t.Error("MediaTypeTarGz should not be empty")
	}
}
