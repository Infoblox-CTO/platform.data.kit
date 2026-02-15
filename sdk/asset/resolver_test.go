package asset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestEmbeddedResolver_KnownExtension(t *testing.T) {
	r := NewEmbeddedResolver()

	data, err := r.ResolveSchema(context.Background(), "cloudquery.source.aws", "v24.0.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("schema data should not be empty")
	}

	// Verify it contains expected JSON Schema markers
	s := string(data)
	if !containsString(s, "$schema") {
		t.Error("schema should contain $schema field")
	}
	if !containsString(s, "accounts") {
		t.Error("schema should contain accounts property")
	}
}

func TestEmbeddedResolver_UnknownExtension(t *testing.T) {
	r := NewEmbeddedResolver()

	_, err := r.ResolveSchema(context.Background(), "unknown.source.test", "v1.0.0")
	if err == nil {
		t.Fatal("expected error for unknown extension")
	}
	if !containsString(err.Error(), "no embedded schema") {
		t.Errorf("error should mention no embedded schema, got: %v", err)
	}
}

func TestCachingResolver_CacheHit(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-populate cache
	cacheDir := filepath.Join(tmpDir, "cloudquery.source.aws", "v1.0.0")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	cachedData := []byte(`{"cached": true}`)
	if err := os.WriteFile(filepath.Join(cacheDir, "schema.json"), cachedData, 0644); err != nil {
		t.Fatal(err)
	}

	// Inner resolver should NOT be called
	inner := &mockResolver{err: fmt.Errorf("should not be called")}
	r := NewCachingResolver(inner, tmpDir)

	data, err := r.ResolveSchema(context.Background(), "cloudquery.source.aws", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != string(cachedData) {
		t.Errorf("got %q, want %q", string(data), string(cachedData))
	}
}

func TestCachingResolver_CacheMiss(t *testing.T) {
	tmpDir := t.TempDir()

	innerData := []byte(`{"from": "inner"}`)
	inner := &mockResolver{data: innerData}
	r := NewCachingResolver(inner, tmpDir)

	data, err := r.ResolveSchema(context.Background(), "cloudquery.source.aws", "v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != string(innerData) {
		t.Errorf("got %q, want %q", string(data), string(innerData))
	}

	// Verify it was cached
	cached, err := os.ReadFile(r.cachePath("cloudquery.source.aws", "v2.0.0"))
	if err != nil {
		t.Fatalf("cache file should exist: %v", err)
	}
	if string(cached) != string(innerData) {
		t.Errorf("cached = %q, want %q", string(cached), string(innerData))
	}
}

func TestFallbackResolver(t *testing.T) {
	first := &mockResolver{err: fmt.Errorf("first failed")}
	second := &mockResolver{data: []byte(`{"ok": true}`)}

	r := NewFallbackResolver(first, second)

	data, err := r.ResolveSchema(context.Background(), "test.source.x", "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data) != `{"ok": true}` {
		t.Errorf("got %q, want %q", string(data), `{"ok": true}`)
	}
}

func TestFallbackResolver_AllFail(t *testing.T) {
	first := &mockResolver{err: fmt.Errorf("first failed")}
	second := &mockResolver{err: fmt.Errorf("second failed")}

	r := NewFallbackResolver(first, second)

	_, err := r.ResolveSchema(context.Background(), "test.source.x", "v1.0.0")
	if err == nil {
		t.Fatal("expected error when all resolvers fail")
	}
}

// mockResolver is a test double for SchemaResolver.
type mockResolver struct {
	data []byte
	err  error
}

func (m *mockResolver) ResolveSchema(_ context.Context, _, _ string) ([]byte, error) {
	return m.data, m.err
}
