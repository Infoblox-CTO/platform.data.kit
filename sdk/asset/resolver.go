package asset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset/schemas"
)

// SchemaResolver resolves extension JSON schemas by FQN and version.
type SchemaResolver interface {
	// ResolveSchema retrieves the JSON Schema bytes for a given extension FQN and version.
	ResolveSchema(ctx context.Context, fqn, version string) ([]byte, error)
}

// EmbeddedResolver resolves schemas from embedded Go files.
// This is the fallback resolver that ships with the CLI for known extensions.
type EmbeddedResolver struct{}

// NewEmbeddedResolver creates a new EmbeddedResolver.
func NewEmbeddedResolver() *EmbeddedResolver {
	return &EmbeddedResolver{}
}

// ResolveSchema returns the embedded schema for a known extension FQN.
// Version is ignored for embedded schemas (they are pinned to the CLI build).
func (r *EmbeddedResolver) ResolveSchema(_ context.Context, fqn, _ string) ([]byte, error) {
	filename, ok := schemas.KnownExtensions[fqn]
	if !ok {
		return nil, fmt.Errorf("no embedded schema for extension %q", fqn)
	}

	data, err := schemas.EmbeddedSchemas.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schema for %q: %w", fqn, err)
	}

	return data, nil
}

// CachingResolver wraps another resolver and caches schemas on the local filesystem.
// Cache location: ~/.cache/dp/schemas/<fqn>/<version>/schema.json
type CachingResolver struct {
	inner    SchemaResolver
	cacheDir string
	mu       sync.Mutex
}

// NewCachingResolver creates a resolver that caches schemas locally.
// If cacheDir is empty, it defaults to ~/.cache/dp/schemas/.
func NewCachingResolver(inner SchemaResolver, cacheDir string) *CachingResolver {
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			cacheDir = filepath.Join(os.TempDir(), "dp", "schemas")
		} else {
			cacheDir = filepath.Join(home, ".cache", "dp", "schemas")
		}
	}
	return &CachingResolver{
		inner:    inner,
		cacheDir: cacheDir,
	}
}

// ResolveSchema checks the cache first, then falls back to the inner resolver.
func (r *CachingResolver) ResolveSchema(ctx context.Context, fqn, version string) ([]byte, error) {
	// Try cache first
	cached, err := r.readCache(fqn, version)
	if err == nil {
		return cached, nil
	}

	// Resolve from inner
	data, err := r.inner.ResolveSchema(ctx, fqn, version)
	if err != nil {
		return nil, err
	}

	// Write to cache (best-effort)
	r.writeCache(fqn, version, data)

	return data, nil
}

// cachePath returns the filesystem path for a cached schema.
func (r *CachingResolver) cachePath(fqn, version string) string {
	return filepath.Join(r.cacheDir, fqn, version, "schema.json")
}

// readCache reads a schema from the local cache.
func (r *CachingResolver) readCache(fqn, version string) ([]byte, error) {
	return os.ReadFile(r.cachePath(fqn, version))
}

// writeCache writes a schema to the local cache.
func (r *CachingResolver) writeCache(fqn, version string, data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	path := r.cachePath(fqn, version)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return // best-effort
	}
	_ = os.WriteFile(path, data, 0644)
}

// FallbackResolver tries multiple resolvers in order, returning the first success.
type FallbackResolver struct {
	resolvers []SchemaResolver
}

// NewFallbackResolver creates a resolver that tries resolvers in order.
func NewFallbackResolver(resolvers ...SchemaResolver) *FallbackResolver {
	return &FallbackResolver{resolvers: resolvers}
}

// ResolveSchema tries each resolver in order until one succeeds.
func (r *FallbackResolver) ResolveSchema(ctx context.Context, fqn, version string) ([]byte, error) {
	var lastErr error
	for _, resolver := range r.resolvers {
		data, err := resolver.ResolveSchema(ctx, fqn, version)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all resolvers failed for %s@%s: %w", fqn, version, lastErr)
	}
	return nil, fmt.Errorf("no resolvers configured for %s@%s", fqn, version)
}

// DefaultResolver creates the standard three-tier resolution chain:
// cache → embedded fallback
func DefaultResolver() SchemaResolver {
	embedded := NewEmbeddedResolver()
	return NewCachingResolver(embedded, "")
}
