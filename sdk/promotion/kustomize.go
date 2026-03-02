// Package promotion provides services for promoting data packages between environments.
package promotion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// VersionFile represents the version.yaml file structure.
type VersionFile struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   VersionMeta `yaml:"metadata"`
	Spec       VersionSpec `yaml:"spec"`
}

// VersionMeta contains metadata for the version file.
type VersionMeta struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels,omitempty"`
}

// VersionSpec contains the version specification.
type VersionSpec struct {
	Package PackageRef `yaml:"package"`
}

// PackageRef contains the package reference.
type PackageRef struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Registry string `yaml:"registry"`
	Digest   string `yaml:"digest,omitempty"`
}

// FileKustomizeUpdater implements KustomizeUpdater using the filesystem.
type FileKustomizeUpdater struct {
	// RepoPath is the path to the GitOps repository root.
	RepoPath string
	// OverlaysDir is the relative path to the environments directory.
	OverlaysDir string
}

// NewFileKustomizeUpdater creates a new FileKustomizeUpdater.
func NewFileKustomizeUpdater(repoPath string) *FileKustomizeUpdater {
	return &FileKustomizeUpdater{
		RepoPath:    repoPath,
		OverlaysDir: "gitops/environments",
	}
}

// UpdateVersion updates the version in the environment overlay.
func (u *FileKustomizeUpdater) UpdateVersion(ctx context.Context, env Environment, pkg, version, digest string) error {
	versionPath := u.versionFilePath(env, pkg)

	// Read existing file or create new one
	vf, err := u.readVersionFile(versionPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading version file: %w", err)
		}
		// Create new version file
		vf = u.newVersionFile(pkg, version, digest)
	} else {
		// Update existing file
		vf.Spec.Package.Version = version
		if digest != "" {
			vf.Spec.Package.Digest = digest
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(versionPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Write updated file
	if err := u.writeVersionFile(versionPath, vf); err != nil {
		return fmt.Errorf("writing version file: %w", err)
	}

	// Update kustomization.yaml to include the version file
	if err := u.ensureKustomizationResource(env, pkg); err != nil {
		return fmt.Errorf("updating kustomization: %w", err)
	}

	return nil
}

// GetCurrentVersion returns the current version in the environment.
func (u *FileKustomizeUpdater) GetCurrentVersion(ctx context.Context, env Environment, pkg string) (string, error) {
	versionPath := u.versionFilePath(env, pkg)

	vf, err := u.readVersionFile(versionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No version deployed
		}
		return "", fmt.Errorf("reading version file: %w", err)
	}

	return vf.Spec.Package.Version, nil
}

// versionFilePath returns the path to the version file for a package.
func (u *FileKustomizeUpdater) versionFilePath(env Environment, pkg string) string {
	return filepath.Join(u.RepoPath, u.OverlaysDir, env.String(), "packages", pkg, "version.yaml")
}

// kustomizationPath returns the path to the kustomization.yaml for an environment.
func (u *FileKustomizeUpdater) kustomizationPath(env Environment) string {
	return filepath.Join(u.RepoPath, u.OverlaysDir, env.String(), "kustomization.yaml")
}

// readVersionFile reads and parses a version file.
func (u *FileKustomizeUpdater) readVersionFile(path string) (*VersionFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var vf VersionFile
	if err := yaml.Unmarshal(data, &vf); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &vf, nil
}

// writeVersionFile writes a version file.
func (u *FileKustomizeUpdater) writeVersionFile(path string, vf *VersionFile) error {
	data, err := yaml.Marshal(vf)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// newVersionFile creates a new version file.
func (u *FileKustomizeUpdater) newVersionFile(pkg, version, digest string) *VersionFile {
	return &VersionFile{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "PackageVersion",
		Metadata: VersionMeta{
			Name: pkg,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":  "dk",
				"datakit.infoblox.dev/package":  pkg,
			},
		},
		Spec: VersionSpec{
			Package: PackageRef{
				Name:     pkg,
				Version:  version,
				Registry: "ghcr.io/infoblox-cto",
				Digest:   digest,
			},
		},
	}
}

// KustomizationFile represents a kustomization.yaml file.
type KustomizationFile struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Resources  []string `yaml:"resources,omitempty"`
}

// ensureKustomizationResource ensures the package is listed in kustomization.yaml.
func (u *FileKustomizeUpdater) ensureKustomizationResource(env Environment, pkg string) error {
	kustomizationPath := u.kustomizationPath(env)

	data, err := os.ReadFile(kustomizationPath)
	if err != nil {
		return fmt.Errorf("reading kustomization.yaml: %w", err)
	}

	var kf KustomizationFile
	if err := yaml.Unmarshal(data, &kf); err != nil {
		return fmt.Errorf("parsing kustomization.yaml: %w", err)
	}

	// Check if package resource is already listed
	resourcePath := fmt.Sprintf("packages/%s/version.yaml", pkg)
	for _, r := range kf.Resources {
		if r == resourcePath {
			return nil // Already present
		}
	}

	// Add the resource
	kf.Resources = append(kf.Resources, resourcePath)

	// Write back
	data, err = yaml.Marshal(&kf)
	if err != nil {
		return fmt.Errorf("marshaling kustomization.yaml: %w", err)
	}

	if err := os.WriteFile(kustomizationPath, data, 0644); err != nil {
		return fmt.Errorf("writing kustomization.yaml: %w", err)
	}

	return nil
}
