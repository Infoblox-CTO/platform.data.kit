package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestPublishCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"registry", ""},
		{"tag", ""},
		{"insecure", "false"},
		{"plain-http", "false"},
		{"dry-run", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := publishCmd.Flags().Lookup(tt.flag)
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

func TestPublishCmd_Args(t *testing.T) {
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
			err := publishCmd.Args(publishCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPublishCmd_DirectoryNotFound(t *testing.T) {
	// Test that publishing a non-existent directory returns an error
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	cmd := &cobra.Command{}
	err := runPublish(cmd, []string{nonExistent})

	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestPublishCmd_MissingDkYaml(t *testing.T) {
	// Test that publishing a directory without dk.yaml returns an error
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	err := runPublish(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dk.yaml")
	}
}

func TestPublishCmd_DryRun(t *testing.T) {
	// Test dry-run mode (should build but not push)
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
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

	// Save and restore global flags
	oldDryRun := publishDryRun
	defer func() { publishDryRun = oldDryRun }()

	publishDryRun = true

	cmd := &cobra.Command{}
	err := runPublish(cmd, []string{tmpDir})

	// Dry run should succeed for valid package
	if err != nil {
		t.Errorf("runPublish() dry-run error = %v, want nil", err)
	}
}

func TestPublishCmd_CustomRegistry(t *testing.T) {
	// Test publishing with a custom registry
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
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

	// Save and restore global flags
	oldRegistry := publishRegistry
	oldDryRun := publishDryRun
	defer func() {
		publishRegistry = oldRegistry
		publishDryRun = oldDryRun
	}()

	publishRegistry = "ghcr.io/test-org"
	publishDryRun = true

	cmd := &cobra.Command{}
	err := runPublish(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runPublish() with registry error = %v, want nil", err)
	}
}

func TestPublishCmd_InvalidPackage(t *testing.T) {
	// Test publishing an invalid package
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

	// Save and restore global flags
	oldDryRun := publishDryRun
	defer func() { publishDryRun = oldDryRun }()

	publishDryRun = true

	cmd := &cobra.Command{}
	err := runPublish(cmd, []string{tmpDir})

	// Should return error for invalid package
	if err == nil {
		t.Error("expected error for invalid package")
	}
}

func TestPublishCmd_CloudQueryPackage(t *testing.T) {
	// Test that dk publish works for a CloudQuery Transform package in dry-run mode
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: my-cq-source
  namespace: data-team
  version: 0.1.0
spec:
  runtime: cloudquery
  image: my-cq-source:latest
  mode: batch
  inputs:
    - dataset: source-data
  outputs:
    - dataset: example-resource
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := publishDryRun
	defer func() { publishDryRun = oldDryRun }()

	publishDryRun = true

	cmd := &cobra.Command{}
	err := runPublish(cmd, []string{tmpDir})

	// Dry run should succeed for a valid CloudQuery package
	if err != nil {
		t.Errorf("runPublish() dry-run for CloudQuery package error = %v, want nil", err)
	}
}

func TestPublishCmd_CloudQueryInvalidPackage(t *testing.T) {
	// Test that dk publish handles a CloudQuery Transform package with missing fields
	// Note: publish does not run validation - it only builds and pushes the OCI artifact.
	// Validation is the responsibility of dk lint / dk build.
	tmpDir := t.TempDir()

	dkContent := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: bad-cq-source
  namespace: data-team
  version: 0.1.0
spec:
  runtime: cloudquery
  image: bad-source:latest
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dk.yaml"), []byte(dkContent), 0644); err != nil {
		t.Fatalf("failed to write dk.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := publishDryRun
	defer func() { publishDryRun = oldDryRun }()

	publishDryRun = true

	cmd := &cobra.Command{}
	err := runPublish(cmd, []string{tmpDir})

	// Publish dry-run still succeeds because publish does not validate —
	// validation is done by dk lint and dk build
	if err != nil {
		t.Errorf("runPublish() dry-run error = %v, want nil (publish doesn't validate)", err)
	}
}
