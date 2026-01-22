package mocks

import (
	"context"

	"github.com/Infoblox-CTO/data-platform/sdk/registry"
)

// MockRegistryClient is a mock implementation of registry.Client for testing.
type MockRegistryClient struct {
	PushFunc    func(ctx context.Context, ref string, artifact *registry.Artifact) (*registry.PushResult, error)
	PullFunc    func(ctx context.Context, ref string) (*registry.Artifact, error)
	ResolveFunc func(ctx context.Context, ref string) (string, error)
	ExistsFunc  func(ctx context.Context, ref string) (bool, error)
	TagsFunc    func(ctx context.Context, repository string) ([]string, error)
	DeleteFunc  func(ctx context.Context, ref string) error
}

func (m *MockRegistryClient) Push(ctx context.Context, ref string, artifact *registry.Artifact) (*registry.PushResult, error) {
	if m.PushFunc != nil {
		return m.PushFunc(ctx, ref, artifact)
	}
	return &registry.PushResult{Digest: "sha256:mock"}, nil
}

func (m *MockRegistryClient) Pull(ctx context.Context, ref string) (*registry.Artifact, error) {
	if m.PullFunc != nil {
		return m.PullFunc(ctx, ref)
	}
	return &registry.Artifact{}, nil
}

func (m *MockRegistryClient) Resolve(ctx context.Context, ref string) (string, error) {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(ctx, ref)
	}
	return "sha256:mock", nil
}

func (m *MockRegistryClient) Exists(ctx context.Context, ref string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, ref)
	}
	return true, nil
}

func (m *MockRegistryClient) Tags(ctx context.Context, repository string) ([]string, error) {
	if m.TagsFunc != nil {
		return m.TagsFunc(ctx, repository)
	}
	return []string{"v1.0.0", "latest"}, nil
}

func (m *MockRegistryClient) Delete(ctx context.Context, ref string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, ref)
	}
	return nil
}

var _ registry.Client = (*MockRegistryClient)(nil)
