package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// HelmChartOptions configures Helm chart generation.
type HelmChartOptions struct {
	// PackageDir is the package directory to chart.
	PackageDir string

	// GitCommit is the source commit SHA.
	GitCommit string

	// GitBranch is the source branch.
	GitBranch string

	// GitTag is the source tag.
	GitTag string

	// Version overrides the chart version (default: from dk.yaml).
	Version string
}

// HelmChartResult is the output of chart generation.
type HelmChartResult struct {
	// ChartPath is the local path to the generated .tgz file.
	ChartPath string

	// ChartName is the chart name (package name).
	ChartName string

	// ChartVersion is the chart version.
	ChartVersion string

	// Size is the chart tarball size in bytes.
	Size int64
}

// GenerateHelmChart creates a Helm chart tarball from a package directory.
// The chart bundles the Transform, Connectors, and Assets — but NOT the
// store/ directory (stores are cell-specific, not part of the package).
//
// Chart structure:
//
//	<name>/
//	├── Chart.yaml
//	├── values.yaml
//	├── templates/
//	│   └── packagedeployment.yaml
//	└── manifests/
//	    ├── dk.yaml
//	    ├── connector/*.yaml
//	    └── asset/*.yaml
func GenerateHelmChart(opts HelmChartOptions) (*HelmChartResult, error) {
	absDir, err := filepath.Abs(opts.PackageDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Parse dk.yaml for metadata.
	dpPath := filepath.Join(absDir, "dk.yaml")
	dpData, err := os.ReadFile(dpPath)
	if err != nil {
		return nil, fmt.Errorf("reading dk.yaml: %w", err)
	}

	m, _, err := manifest.ParseManifest(dpData)
	if err != nil {
		return nil, fmt.Errorf("parsing dk.yaml: %w", err)
	}

	name := m.GetName()
	version := opts.Version
	if version == "" {
		version = m.GetVersion()
	}
	if version == "" {
		version = "0.0.0"
	}

	// Build version tag with git info.
	chartVersion := version
	if opts.GitCommit != "" && !strings.Contains(version, "-g") {
		short := opts.GitCommit
		if len(short) > 8 {
			short = short[:8]
		}
		chartVersion = fmt.Sprintf("%s-g%s", version, short)
	}

	namespace := m.GetNamespace()
	if namespace == "" {
		namespace = "default"
	}

	// Runtime from dk.yaml (for annotation).
	runtime := ""
	if t, ok := m.(interface{ GetRuntime() string }); ok {
		runtime = t.GetRuntime()
	}

	// Generate chart files.
	chartYAML := generateChartYAML(name, chartVersion, version, runtime)
	valuesYAML := generateValuesYAML()
	packageDeploymentTmpl := generatePackageDeploymentTemplate()

	// Create tarball in dist/.
	distDir := filepath.Join(absDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return nil, fmt.Errorf("creating dist/ directory: %w", err)
	}

	chartFileName := fmt.Sprintf("%s-%s.tgz", name, chartVersion)
	chartPath := filepath.Join(distDir, chartFileName)

	f, err := os.Create(chartPath)
	if err != nil {
		return nil, fmt.Errorf("creating chart file: %w", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	now := time.Now()

	// Write Chart.yaml
	if err := writeChartFile(tw, name+"/Chart.yaml", chartYAML, now); err != nil {
		return nil, err
	}

	// Write values.yaml
	if err := writeChartFile(tw, name+"/values.yaml", valuesYAML, now); err != nil {
		return nil, err
	}

	// Write templates/packagedeployment.yaml
	if err := writeChartFile(tw, name+"/templates/packagedeployment.yaml", packageDeploymentTmpl, now); err != nil {
		return nil, err
	}

	// Write manifests/dk.yaml
	if err := writeChartFile(tw, name+"/manifests/dk.yaml", dpData, now); err != nil {
		return nil, err
	}

	// Copy connector/*.yaml → manifests/connector/*.yaml
	if err := copyManifestDir(tw, absDir, "connector", name, now); err != nil {
		return nil, err
	}

	// Copy asset/*.yaml → manifests/asset/*.yaml
	if err := copyManifestDir(tw, absDir, "asset", name, now); err != nil {
		return nil, err
	}

	// NOTE: store/ is intentionally excluded — stores are cell-specific.

	// Flush writers.
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("closing tar: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip: %w", err)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("closing file: %w", err)
	}

	info, _ := os.Stat(chartPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}

	return &HelmChartResult{
		ChartPath:    chartPath,
		ChartName:    name,
		ChartVersion: chartVersion,
		Size:         size,
	}, nil
}

// --- Chart file generators ---

func generateChartYAML(name, version, appVersion, runtime string) []byte {
	var buf bytes.Buffer
	buf.WriteString("apiVersion: v2\n")
	buf.WriteString(fmt.Sprintf("name: %s\n", name))
	buf.WriteString(fmt.Sprintf("version: %s\n", version))
	buf.WriteString(fmt.Sprintf("appVersion: %q\n", appVersion))
	buf.WriteString(fmt.Sprintf("description: %s data package\n", name))
	buf.WriteString("type: application\n")
	buf.WriteString("annotations:\n")
	buf.WriteString("  io.infoblox.dk/kind: package\n")
	if runtime != "" {
		buf.WriteString(fmt.Sprintf("  io.infoblox.dk/runtime: %s\n", runtime))
	}
	return buf.Bytes()
}

func generateValuesYAML() []byte {
	return []byte(`# Cell name — REQUIRED at deploy time.
# The controller resolves Stores from the cell's namespace (dk-<cell>).
cell: ""

# Resource defaults — override per deployment in the CM repo.
resources:
  requests:
    cpu: 100m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 1Gi

# Replicas (streaming mode only).
replicas: 1

# Schedule (batch mode only) — cron expression.
schedule: ""

# OCI registry (default: ghcr.io/infoblox-cto/dk).
registry: ""
`)
}

func generatePackageDeploymentTemplate() []byte {
	return []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: PackageDeployment
metadata:
  name: {{ .Chart.Name }}
  namespace: dk-{{ .Values.cell }}
spec:
  package:
    name: {{ .Chart.Name }}
    version: {{ .Chart.Version }}
    registry: {{ .Values.registry | default "ghcr.io/infoblox-cto/dk" }}
  cell: {{ .Values.cell }}
  mode: {{ .Values.mode | default "batch" }}
  {{- if .Values.schedule }}
  schedule:
    cron: {{ .Values.schedule | quote }}
  {{- end }}
  resources:
    requests:
      cpu: {{ .Values.resources.requests.cpu | quote }}
      memory: {{ .Values.resources.requests.memory | quote }}
    limits:
      cpu: {{ .Values.resources.limits.cpu | quote }}
      memory: {{ .Values.resources.limits.memory | quote }}
`)
}

// --- Tar helpers ---

func writeChartFile(tw *tar.Writer, path string, content []byte, modTime time.Time) error {
	hdr := &tar.Header{
		Name:    path,
		Size:    int64(len(content)),
		Mode:    0644,
		ModTime: modTime,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing header for %s: %w", path, err)
	}
	if _, err := io.Copy(tw, bytes.NewReader(content)); err != nil {
		return fmt.Errorf("writing content for %s: %w", path, err)
	}
	return nil
}

func copyManifestDir(tw *tar.Writer, packageDir, subdir, chartName string, modTime time.Time) error {
	dirPath := filepath.Join(packageDir, subdir)
	entries, err := os.ReadDir(dirPath)
	if os.IsNotExist(err) {
		return nil // optional
	}
	if err != nil {
		return fmt.Errorf("reading %s/: %w", subdir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			return fmt.Errorf("reading %s/%s: %w", subdir, entry.Name(), err)
		}
		tarPath := fmt.Sprintf("%s/manifests/%s/%s", chartName, subdir, entry.Name())
		if err := writeChartFile(tw, tarPath, data, modTime); err != nil {
			return err
		}
	}
	return nil
}
