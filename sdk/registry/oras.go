package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// OrasClient implements the Client interface using oras-go.
type OrasClient struct {
	config ClientConfig
}

// NewOrasClient creates a new oras-based OCI client.
func NewOrasClient(config ClientConfig) (*OrasClient, error) {
	return &OrasClient{
		config: config,
	}, nil
}

// Push pushes a package artifact to the registry.
func (c *OrasClient) Push(ctx context.Context, ref string, artifact *Artifact) (*PushResult, error) {
	// Parse the reference
	repo, err := c.getRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}

	// Create in-memory store for content
	store := memory.New()

	// Push layers
	var layerDescs []ocispec.Descriptor
	for _, layer := range artifact.Layers {
		desc := ocispec.Descriptor{
			MediaType:   layer.MediaType,
			Digest:      digest.FromBytes(layer.Content),
			Size:        int64(len(layer.Content)),
			Annotations: layer.Annotations,
		}

		if err := store.Push(ctx, desc, bytes.NewReader(layer.Content)); err != nil {
			return nil, fmt.Errorf("failed to store layer: %w", err)
		}
		layerDescs = append(layerDescs, desc)
	}

	// Push config
	configBytes, err := json.Marshal(artifact.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	configDesc := ocispec.Descriptor{
		MediaType: MediaTypeDPConfig,
		Digest:    digest.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}

	if err := store.Push(ctx, configDesc, bytes.NewReader(configBytes)); err != nil {
		return nil, fmt.Errorf("failed to store config: %w", err)
	}

	// Create and push manifest
	manifest := ocispec.Manifest{
		Versioned: struct {
			SchemaVersion int `json:"schemaVersion"`
		}{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: MediaTypeDPPackage,
		Config:       configDesc,
		Layers:       layerDescs,
		Annotations:  artifact.Manifest.Annotations,
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
	}

	if err := store.Push(ctx, manifestDesc, bytes.NewReader(manifestBytes)); err != nil {
		return nil, fmt.Errorf("failed to store manifest: %w", err)
	}

	// Tag the manifest
	if err := store.Tag(ctx, manifestDesc, ref); err != nil {
		return nil, fmt.Errorf("failed to tag manifest: %w", err)
	}

	// Copy from memory store to registry
	desc, err := oras.Copy(ctx, store, ref, repo, ref, oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to push to registry: %w", err)
	}

	return &PushResult{
		Reference: ref,
		Digest:    string(desc.Digest),
		Size:      desc.Size,
	}, nil
}

// Pull pulls a package artifact from the registry.
func (c *OrasClient) Pull(ctx context.Context, ref string) (*Artifact, error) {
	// Parse the reference
	repo, err := c.getRepository(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference: %w", err)
	}

	// Create in-memory store for content
	store := memory.New()

	// Copy from registry to memory store
	desc, err := oras.Copy(ctx, repo, ref, store, ref, oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to pull from registry: %w", err)
	}

	// Fetch manifest
	manifestReader, err := store.Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer manifestReader.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(manifestReader).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	// Fetch config
	configReader, err := store.Fetch(ctx, manifest.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %w", err)
	}
	defer configReader.Close()

	var config ArtifactConfig
	if err := json.NewDecoder(configReader).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Fetch layers
	var layers []Layer
	for _, layerDesc := range manifest.Layers {
		layerReader, err := store.Fetch(ctx, layerDesc)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch layer: %w", err)
		}

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(layerReader); err != nil {
			layerReader.Close()
			return nil, fmt.Errorf("failed to read layer: %w", err)
		}
		layerReader.Close()

		layers = append(layers, Layer{
			MediaType:   layerDesc.MediaType,
			Content:     buf.Bytes(),
			Annotations: layerDesc.Annotations,
		})
	}

	return &Artifact{
		Manifest: &ArtifactManifest{
			MediaType:     manifest.MediaType,
			SchemaVersion: manifest.Versioned.SchemaVersion,
			ArtifactType:  manifest.ArtifactType,
			Annotations:   manifest.Annotations,
		},
		Config: &config,
		Layers: layers,
	}, nil
}

// Resolve resolves a reference to a digest.
func (c *OrasClient) Resolve(ctx context.Context, ref string) (string, error) {
	repo, err := c.getRepository(ref)
	if err != nil {
		return "", fmt.Errorf("failed to parse reference: %w", err)
	}

	desc, err := repo.Resolve(ctx, ref)
	if err != nil {
		return "", fmt.Errorf("failed to resolve reference: %w", err)
	}

	return string(desc.Digest), nil
}

// Exists checks if a reference exists in the registry.
func (c *OrasClient) Exists(ctx context.Context, ref string) (bool, error) {
	_, err := c.Resolve(ctx, ref)
	if err != nil {
		// Check if error is "not found"
		return false, nil
	}
	return true, nil
}

// Tags lists all tags for a repository.
func (c *OrasClient) Tags(ctx context.Context, repository string) ([]string, error) {
	repo, err := c.getRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository: %w", err)
	}

	var tags []string
	err = repo.Tags(ctx, "", func(t []string) error {
		tags = append(tags, t...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	return tags, nil
}

// Delete removes an artifact from the registry.
func (c *OrasClient) Delete(ctx context.Context, ref string) error {
	repo, err := c.getRepository(ref)
	if err != nil {
		return fmt.Errorf("failed to parse reference: %w", err)
	}

	desc, err := repo.Resolve(ctx, ref)
	if err != nil {
		return fmt.Errorf("failed to resolve reference: %w", err)
	}

	if err := repo.Delete(ctx, desc); err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	return nil
}

// getRepository creates a repository client for the given reference.
func (c *OrasClient) getRepository(ref string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref)
	if err != nil {
		return nil, err
	}

	// Configure authentication
	if c.config.Username != "" || c.config.Token != "" {
		repo.Client = &auth.Client{
			Credential: func(ctx context.Context, hostport string) (auth.Credential, error) {
				return auth.Credential{
					Username:    c.config.Username,
					Password:    c.config.Password,
					AccessToken: c.config.Token,
				}, nil
			},
		}
	}

	// Configure connection options
	repo.PlainHTTP = c.config.PlainHTTP

	return repo, nil
}
