package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// Bundler creates OCI artifacts from package directories.
type Bundler struct {
	// Version of the CLI/SDK used to build.
	Version string
}

// NewBundler creates a new artifact bundler.
func NewBundler(version string) *Bundler {
	return &Bundler{
		Version: version,
	}
}

// BundleOptions configures the bundling process.
type BundleOptions struct {
	// PackageDir is the package directory to bundle.
	PackageDir string

	// GitCommit is the source commit SHA.
	GitCommit string

	// GitBranch is the source branch.
	GitBranch string

	// GitTag is the source tag.
	GitTag string

	// ExcludePatterns are patterns to exclude from bundling.
	ExcludePatterns []string
}

// Bundle creates an artifact from a package directory.
func (b *Bundler) Bundle(opts BundleOptions) (*Artifact, error) {
	absDir, err := filepath.Abs(opts.PackageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Read and parse dk.yaml
	dkPath := filepath.Join(absDir, "dk.yaml")
	dpData, err := os.ReadFile(dkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dk.yaml: %w", err)
	}

	m, kind, err := manifest.ParseManifest(dpData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dk.yaml: %w", err)
	}

	// Create manifest layer (YAML files)
	manifestLayer, err := b.createManifestLayer(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest layer: %w", err)
	}

	// Create content layer (code, schemas, etc.)
	contentLayer, err := b.createContentLayer(absDir, opts.ExcludePatterns)
	if err != nil {
		return nil, fmt.Errorf("failed to create content layer: %w", err)
	}

	// Create build info
	hostname, _ := os.Hostname()
	buildInfo := &BuildInfo{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Builder:   "dk/" + b.Version,
		GitCommit: opts.GitCommit,
		GitBranch: opts.GitBranch,
		GitTag:    opts.GitTag,
		Host:      hostname,
	}

	// Create artifact config
	config := &ArtifactConfig{
		Manifest:  m,
		Kind:      kind,
		BuildInfo: buildInfo,
	}

	// Create artifact manifest annotations
	annotations := map[string]string{
		"org.opencontainers.image.created":     buildInfo.Timestamp,
		"org.opencontainers.image.version":     m.GetVersion(),
		"org.opencontainers.image.title":       m.GetName(),
		"org.opencontainers.image.description": m.GetDescription(),
		"io.infoblox.dk.namespace":             m.GetNamespace(),
		"io.infoblox.dk.kind":                  string(kind),
	}

	if opts.GitCommit != "" {
		annotations["org.opencontainers.image.revision"] = opts.GitCommit
	}

	return &Artifact{
		Manifest: &ArtifactManifest{
			MediaType:     "application/vnd.oci.image.manifest.v1+json",
			SchemaVersion: 2,
			ArtifactType:  MediaTypeDKPackage,
			Annotations:   annotations,
		},
		Config: config,
		Layers: []Layer{
			manifestLayer,
			contentLayer,
		},
	}, nil
}

// createManifestLayer creates a tar.gz layer containing all YAML manifest files.
func (b *Bundler) createManifestLayer(packageDir string) (Layer, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add manifest files
	manifestFiles := []string{
		"dk.yaml",
	}

	for _, name := range manifestFiles {
		path := filepath.Join(packageDir, name)
		if _, err := os.Stat(path); err != nil {
			continue // Skip if doesn't exist
		}

		if err := b.addFileToTar(tw, path, name); err != nil {
			tw.Close()
			gw.Close()
			return Layer{}, fmt.Errorf("failed to add %s: %w", name, err)
		}
	}

	// Add schemas directory
	schemasDir := filepath.Join(packageDir, "schemas")
	if _, err := os.Stat(schemasDir); err == nil {
		if err := b.addDirToTar(tw, schemasDir, "schemas"); err != nil {
			tw.Close()
			gw.Close()
			return Layer{}, fmt.Errorf("failed to add schemas: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		gw.Close()
		return Layer{}, err
	}
	if err := gw.Close(); err != nil {
		return Layer{}, err
	}

	return Layer{
		MediaType: MediaTypeDKManifest,
		Content:   buf.Bytes(),
		Annotations: map[string]string{
			"io.infoblox.dk.layer.type": "manifests",
		},
	}, nil
}

// createContentLayer creates a tar.gz layer containing source code and other content.
func (b *Bundler) createContentLayer(packageDir string, excludePatterns []string) (Layer, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Default exclude patterns
	defaultExcludes := []string{
		".git",
		".gitignore",
		"node_modules",
		"__pycache__",
		".pytest_cache",
		"*.pyc",
		".DS_Store",
		"Thumbs.db",
	}
	excludes := append(defaultExcludes, excludePatterns...)

	// Walk the package directory
	err := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(packageDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		// Check excludes
		for _, exclude := range excludes {
			if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip manifest files (they're in their own layer)
		if relPath == "dk.yaml" ||
			filepath.HasPrefix(relPath, "schemas") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		return b.addFileToTar(tw, path, relPath)
	})

	if err != nil {
		tw.Close()
		gw.Close()
		return Layer{}, err
	}

	if err := tw.Close(); err != nil {
		gw.Close()
		return Layer{}, err
	}
	if err := gw.Close(); err != nil {
		return Layer{}, err
	}

	return Layer{
		MediaType: MediaTypeTarGz,
		Content:   buf.Bytes(),
		Annotations: map[string]string{
			"io.infoblox.dk.layer.type": "content",
		},
	}, nil
}

// addFileToTar adds a file to a tar archive.
func (b *Bundler) addFileToTar(tw *tar.Writer, path, name string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// addDirToTar adds a directory and its contents to a tar archive.
func (b *Bundler) addDirToTar(tw *tar.Writer, dirPath, baseName string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		name := filepath.Join(baseName, relPath)

		if info.IsDir() {
			header := &tar.Header{
				Name:     name + "/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			}
			return tw.WriteHeader(header)
		}

		return b.addFileToTar(tw, path, name)
	})
}

// Reference creates an OCI reference for a package.
func Reference(registry, namespace, name, version string) string {
	return fmt.Sprintf("%s/%s/%s:%s", registry, namespace, name, version)
}

// DigestReference creates an OCI reference using a digest.
func DigestReference(registry, namespace, name, digest string) string {
	return fmt.Sprintf("%s/%s/%s@%s", registry, namespace, name, digest)
}
