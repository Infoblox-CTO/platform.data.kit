package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateHelmChart(t *testing.T) {
	// Create a temp package directory.
	tmpDir := t.TempDir()

	// Write dk.yaml
	dpYAML := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  namespace: default
  version: "1.2.0"
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - asset: users
  outputs:
    - asset: users-parquet
`
	os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dpYAML), 0644)

	// Write connector/
	os.MkdirAll(filepath.Join(tmpDir, "connector"), 0755)
	connYAML := `apiVersion: data.infoblox.com/v1alpha1
kind: Connector
metadata:
  name: postgres
spec:
  plugin:
    source: ghcr.io/cloudquery/cq-source-postgresql:v9.0.0
`
	os.WriteFile(filepath.Join(tmpDir, "connector", "postgres.yaml"), []byte(connYAML), 0644)

	// Write asset/
	os.MkdirAll(filepath.Join(tmpDir, "asset"), 0755)
	assetYAML := `apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: users
spec:
  store: source-db
  table: public.users
`
	os.WriteFile(filepath.Join(tmpDir, "asset", "users.yaml"), []byte(assetYAML), 0644)

	// Write store/ (should be excluded from chart)
	os.MkdirAll(filepath.Join(tmpDir, "store"), 0755)
	storeYAML := `apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: source-db
spec:
  connector: postgres
  connection:
    connection_string: "postgresql://localhost:5432/dp_local"
`
	os.WriteFile(filepath.Join(tmpDir, "store", "source-db.yaml"), []byte(storeYAML), 0644)

	// Generate the chart.
	result, err := GenerateHelmChart(HelmChartOptions{
		PackageDir: tmpDir,
		GitCommit:  "abc12345def",
	})
	if err != nil {
		t.Fatalf("GenerateHelmChart() error: %v", err)
	}

	// Verify result metadata.
	if result.ChartName != "pg-to-s3" {
		t.Errorf("ChartName = %q, want %q", result.ChartName, "pg-to-s3")
	}
	if !strings.HasPrefix(result.ChartVersion, "1.2.0-g") {
		t.Errorf("ChartVersion = %q, want prefix %q", result.ChartVersion, "1.2.0-g")
	}
	if result.Size == 0 {
		t.Error("Chart size is 0")
	}

	// Verify file exists.
	if _, err := os.Stat(result.ChartPath); err != nil {
		t.Fatalf("Chart file does not exist: %v", err)
	}

	// Open and inspect the tarball.
	files := listTarFiles(t, result.ChartPath)

	// Expected files.
	expected := []string{
		"pg-to-s3/Chart.yaml",
		"pg-to-s3/values.yaml",
		"pg-to-s3/templates/packagedeployment.yaml",
		"pg-to-s3/manifests/dk.yaml",
		"pg-to-s3/manifests/connector/postgres.yaml",
		"pg-to-s3/manifests/asset/users.yaml",
	}
	for _, exp := range expected {
		if !contains(files, exp) {
			t.Errorf("chart missing expected file: %s", exp)
		}
	}

	// store/ should NOT be in the chart.
	for _, f := range files {
		if strings.Contains(f, "store/") {
			t.Errorf("chart should not contain store files, found: %s", f)
		}
	}
}

func TestGenerateHelmChart_ChartYAMLContent(t *testing.T) {
	tmpDir := t.TempDir()

	dpYAML := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: my-pipeline
  version: "2.0.0"
spec:
  runtime: cloudquery
  inputs:
    - asset: data
  outputs:
    - asset: result
`
	os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dpYAML), 0644)

	result, err := GenerateHelmChart(HelmChartOptions{
		PackageDir: tmpDir,
		GitCommit:  "deadbeef12345",
	})
	if err != nil {
		t.Fatalf("GenerateHelmChart() error: %v", err)
	}

	// Read Chart.yaml from the tarball.
	chartYAML := readTarFile(t, result.ChartPath, "my-pipeline/Chart.yaml")

	if !strings.Contains(string(chartYAML), "name: my-pipeline") {
		t.Errorf("Chart.yaml missing name, got: %s", chartYAML)
	}
	if !strings.Contains(string(chartYAML), "version: 2.0.0-gdeadbeef") {
		t.Errorf("Chart.yaml missing version, got: %s", chartYAML)
	}
	if !strings.Contains(string(chartYAML), "io.infoblox.dk/kind: package") {
		t.Errorf("Chart.yaml missing annotation, got: %s", chartYAML)
	}
}

func TestGenerateValuesYAML(t *testing.T) {
	data := generateValuesYAML()
	s := string(data)
	if !strings.Contains(s, "cell:") {
		t.Error("values.yaml missing cell field")
	}
	if !strings.Contains(s, "resources:") {
		t.Error("values.yaml missing resources field")
	}
	if !strings.Contains(s, "schedule:") {
		t.Error("values.yaml missing schedule field")
	}
}

// --- helpers ---

func listTarFiles(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var files []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		files = append(files, hdr.Name)
	}
	return files
}

func readTarFile(t *testing.T, tarPath, targetFile string) []byte {
	t.Helper()
	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("opening %s: %v", tarPath, err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		if hdr.Name == targetFile {
			var buf bytes.Buffer
			io.Copy(&buf, tr)
			return buf.Bytes()
		}
	}
	t.Fatalf("file %s not found in tarball", targetFile)
	return nil
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
