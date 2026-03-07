package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBundler(t *testing.T) {
	bundler := NewBundler("v1.0.0")
	if bundler == nil {
		t.Error("NewBundler should not return nil")
	}
	if bundler.Version != "v1.0.0" {
		t.Errorf("Version = %s, want v1.0.0", bundler.Version)
	}
}

func TestBundleOptions_Defaults(t *testing.T) {
	opts := BundleOptions{
		PackageDir: "/path/to/package",
	}

	if opts.PackageDir == "" {
		t.Error("PackageDir should not be empty")
	}
	if opts.ExcludePatterns == nil {
		// nil is acceptable for ExcludePatterns
	}
}

func TestBundleOptions_WithAllFields(t *testing.T) {
	opts := BundleOptions{
		PackageDir:      "/path/to/package",
		GitCommit:       "abc123",
		GitBranch:       "main",
		GitTag:          "v1.0.0",
		ExcludePatterns: []string{".git", "node_modules"},
	}

	if opts.GitCommit != "abc123" {
		t.Errorf("GitCommit = %s, want abc123", opts.GitCommit)
	}
	if opts.GitBranch != "main" {
		t.Errorf("GitBranch = %s, want main", opts.GitBranch)
	}
	if opts.GitTag != "v1.0.0" {
		t.Errorf("GitTag = %s, want v1.0.0", opts.GitTag)
	}
	if len(opts.ExcludePatterns) != 2 {
		t.Errorf("ExcludePatterns count = %d, want 2", len(opts.ExcludePatterns))
	}
}

func TestBundle_MissingDkYaml(t *testing.T) {
	tmpDir := t.TempDir()
	bundler := NewBundler("v1.0.0")

	opts := BundleOptions{
		PackageDir: tmpDir,
	}

	_, err := bundler.Bundle(opts)
	if err == nil {
		t.Error("expected error for missing dk.yaml")
	}
}

func TestBundle_InvalidDkYaml(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid YAML
	invalidYAML := "invalid: [yaml: content"
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	bundler := NewBundler("v1.0.0")
	opts := BundleOptions{
		PackageDir: tmpDir,
	}

	_, err := bundler.Bundle(opts)
	if err == nil {
		t.Error("expected error for invalid dk.yaml")
	}
}

func TestBundle_ValidPackage(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  description: Test package
  owner: data-team
  runtime: generic-go
  image: myimage:v1
  mode: batch
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	bundler := NewBundler("v1.0.0")
	opts := BundleOptions{
		PackageDir: tmpDir,
		GitCommit:  "abc123",
		GitBranch:  "main",
	}

	artifact, err := bundler.Bundle(opts)
	if err != nil {
		t.Fatalf("Bundle() error = %v", err)
	}

	if artifact == nil {
		t.Error("artifact should not be nil")
	}
}

func TestBundle_NonExistentDirectory(t *testing.T) {
	bundler := NewBundler("v1.0.0")
	opts := BundleOptions{
		PackageDir: "/non/existent/path",
	}

	_, err := bundler.Bundle(opts)
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestBundle_WithExcludePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  description: Test package
  owner: data-team
  runtime: generic-go
  image: myimage:v1
  mode: batch
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Create a directory to exclude
	if err := os.MkdirAll(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "node_modules", "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	bundler := NewBundler("v1.0.0")
	opts := BundleOptions{
		PackageDir:      tmpDir,
		ExcludePatterns: []string{"node_modules"},
	}

	artifact, err := bundler.Bundle(opts)
	if err != nil {
		t.Fatalf("Bundle() error = %v", err)
	}

	if artifact == nil {
		t.Error("artifact should not be nil")
	}
}
