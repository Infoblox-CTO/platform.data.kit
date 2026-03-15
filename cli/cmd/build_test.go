package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"tag", ""},
		{"push", "false"},
		{"dry-run", "false"},
		{"no-cache", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := buildCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Errorf("flag --%s not found", tt.flag)
				return
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("flag --%s default = %v, want %v", tt.flag, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestBuildCmd_Args(t *testing.T) {
	// Test argument validation
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no args is valid",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one arg is valid",
			args:    []string{"./my-package"},
			wantErr: false,
		},
		{
			name:    "two args is invalid",
			args:    []string{"./pkg1", "./pkg2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildCmd.Args(buildCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildCmd_DirectoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{nonExistent})

	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestBuildCmd_MissingDkYaml(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dk.yaml")
	}
}

func TestBuildCmd_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
spec:
  runtime: generic-go
  image: myimage:v1
  mode: batch
  inputs:
    - dataset: source-data
  outputs:
    - dataset: output-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	oldDryRun := buildDryRun
	defer func() { buildDryRun = oldDryRun }()

	buildDryRun = true

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runBuild() dry-run error = %v, want nil", err)
	}
}

func TestBuildCmd_InvalidPackage(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: ""
spec:
  runtime: generic-go
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for invalid package")
	}
}

func TestBuildCmd_CustomTag(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
spec:
  runtime: generic-go
  image: myimage:v1
  mode: batch
  inputs:
    - dataset: source-data
  outputs:
    - dataset: output-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	oldTag := buildTag
	oldDryRun := buildDryRun
	defer func() {
		buildTag = oldTag
		buildDryRun = oldDryRun
	}()

	buildTag = "v2.0.0"
	buildDryRun = true

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runBuild() with tag error = %v, want nil", err)
	}
}

func TestBuildCmd_CloudQueryPackage(t *testing.T) {
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: my-cq-source
spec:
  runtime: cloudquery
  image: my-cq-source:latest
  mode: batch
  inputs:
    - dataset: source-data
  outputs:
    - dataset: cloud-resources
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	oldDryRun := buildDryRun
	defer func() { buildDryRun = oldDryRun }()

	buildDryRun = true

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runBuild() dry-run for CloudQuery package error = %v, want nil", err)
	}
}

func TestBuildCmd_NoHelmChart(t *testing.T) {
	// Verify build no longer produces a Helm chart
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
  version: "1.0.0"
spec:
  runtime: generic-go
  image: myimage:v1
  mode: batch
  inputs:
    - dataset: source-data
  outputs:
    - dataset: output-data
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	oldDryRun := buildDryRun
	defer func() { buildDryRun = oldDryRun }()

	buildDryRun = false

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runBuild() error = %v, want nil", err)
	}

	// Verify no .tgz files in dist/
	distDir := filepath.Join(tmpDir, "dist")
	entries, err := os.ReadDir(distDir)
	if err == nil {
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".tgz" {
				t.Errorf("build should not produce .tgz files, found: %s", e.Name())
			}
		}
	}
	// dist/ not existing is also acceptable (no chart generated)
}
