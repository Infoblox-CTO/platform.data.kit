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
	// Test that building a non-existent directory returns an error
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{nonExistent})

	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestBuildCmd_MissingDpYaml(t *testing.T) {
	// Test that building a directory without dp.yaml returns an error
	tmpDir := t.TempDir()

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for missing dp.yaml")
	}
}

func TestBuildCmd_DryRun(t *testing.T) {
	// Test dry-run mode
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: dataset
  description: Test package
  owner: data-team
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldDryRun := buildDryRun
	defer func() { buildDryRun = oldDryRun }()

	buildDryRun = true

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	// Dry run should succeed for valid package
	if err != nil {
		t.Errorf("runBuild() dry-run error = %v, want nil", err)
	}
}

func TestBuildCmd_InvalidPackage(t *testing.T) {
	// Test building an invalid package (missing required fields)
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: ""
spec:
  type: pipeline
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	cmd := &cobra.Command{}
	err := runBuild(cmd, []string{tmpDir})

	// Should return error for invalid package
	if err == nil {
		t.Error("expected error for invalid package")
	}
}

func TestBuildCmd_CustomTag(t *testing.T) {
	// Test building with a custom tag
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: dataset
  description: Test package
  owner: data-team
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
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
