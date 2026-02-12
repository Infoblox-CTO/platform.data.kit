package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestLintCmd_Flags(t *testing.T) {
	tests := []struct {
		flag     string
		defValue string
	}{
		{"strict", "false"},
		{"skip-pii", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := lintCmd.Flags().Lookup(tt.flag)
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

func TestLintCmd_Args(t *testing.T) {
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
			err := lintCmd.Args(lintCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLintCmd_DirectoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	cmd := &cobra.Command{}
	err := runLint(cmd, []string{nonExistent})

	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestLintCmd_ValidPackage(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test package
  owner: data-team
  runtime:
    image: myimage:v1
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

	lintStrict = false
	lintSkipPII = false

	cmd := &cobra.Command{}
	err := runLint(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runLint() error = %v, want nil for valid package", err)
	}
}

func TestLintCmd_InvalidPackage(t *testing.T) {
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

	lintStrict = false
	lintSkipPII = false

	cmd := &cobra.Command{}
	err := runLint(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for invalid package")
	}
}

func TestLintCmd_StrictMode(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test
  owner: data-team
  runtime:
    image: myimage:v1
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

	lintStrict = true
	lintSkipPII = false

	cmd := &cobra.Command{}
	_ = runLint(cmd, []string{tmpDir})

	lintStrict = false
}

func TestLintCmd_SkipPII(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test
  owner: data-team
  runtime:
    image: myimage:v1
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
`
	if err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	lintStrict = false
	lintSkipPII = true

	cmd := &cobra.Command{}
	err := runLint(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runLint() with skip-pii error = %v, want nil", err)
	}

	lintSkipPII = false
}

func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestLintCmd_OverrideFlags(t *testing.T) {
	// Verify override flags are registered
	tests := []struct {
		flag      string
		shorthand string
		defValue  string
	}{
		{"set", "", "[]"},
		{"values", "f", "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := lintCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Errorf("flag --%s not found", tt.flag)
				return
			}
			if tt.shorthand != "" && flag.Shorthand != tt.shorthand {
				t.Errorf("flag --%s shorthand = %q, want %q", tt.flag, flag.Shorthand, tt.shorthand)
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("flag --%s default = %v, want %v", tt.flag, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestLintCmd_WithOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dp.yaml without runtime (will fail validation for pipelines)
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test pipeline
  owner: data-team
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Create override file that adds runtime
	overrideContent := `spec:
  runtime:
    image: test:v1
`
	overridePath := filepath.Join(tmpDir, "overrides.yaml")
	if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
		t.Fatalf("failed to write overrides.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := lintSet
	oldFiles := lintValueFiles
	oldStrict := lintStrict
	oldSkipPII := lintSkipPII
	defer func() {
		lintSet = oldSet
		lintValueFiles = oldFiles
		lintStrict = oldStrict
		lintSkipPII = oldSkipPII
	}()

	// Apply the override file
	lintValueFiles = []string{overridePath}
	lintSet = []string{}
	lintStrict = false
	lintSkipPII = false

	cmd := &cobra.Command{}
	err := runLint(cmd, []string{tmpDir})

	// With runtime added via override, validation should pass
	if err != nil {
		t.Errorf("runLint() with override error = %v, want nil", err)
	}

	// Verify backup was created
	backupPath := dpPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("expected backup file to be created")
	}
}

func TestLintCmd_WithSetOverride(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid pipeline dp.yaml
	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test pipeline
  owner: data-team
  runtime:
    image: original:v1
  outputs:
    - name: output
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := lintSet
	oldFiles := lintValueFiles
	oldStrict := lintStrict
	oldSkipPII := lintSkipPII
	defer func() {
		lintSet = oldSet
		lintValueFiles = oldFiles
		lintStrict = oldStrict
		lintSkipPII = oldSkipPII
	}()

	// Apply --set override
	lintValueFiles = []string{}
	lintSet = []string{"spec.runtime.image=overridden:v2"}
	lintStrict = false
	lintSkipPII = false

	cmd := &cobra.Command{}
	err := runLint(cmd, []string{tmpDir})

	if err != nil {
		t.Errorf("runLint() with --set error = %v, want nil", err)
	}

	// Read the modified dp.yaml to verify override was applied
	data, err := os.ReadFile(dpPath)
	if err != nil {
		t.Fatalf("failed to read dp.yaml: %v", err)
	}

	if !stringContains(string(data), "overridden:v2") {
		t.Error("expected override to be applied to dp.yaml")
	}
}

func TestLintCmd_InvalidOverridePath(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
spec:
  type: pipeline
  runtime:
    image: test:v1
`
	dpPath := filepath.Join(tmpDir, "dp.yaml")
	if err := os.WriteFile(dpPath, []byte(dpContent), 0644); err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	// Save and restore global flags
	oldSet := lintSet
	oldFiles := lintValueFiles
	oldStrict := lintStrict
	oldSkipPII := lintSkipPII
	defer func() {
		lintSet = oldSet
		lintValueFiles = oldFiles
		lintStrict = oldStrict
		lintSkipPII = oldSkipPII
	}()

	// Use invalid path
	lintValueFiles = []string{}
	lintSet = []string{"invalid.path.here=value"}
	lintStrict = false
	lintSkipPII = false

	cmd := &cobra.Command{}
	err := runLint(cmd, []string{tmpDir})

	if err == nil {
		t.Error("expected error for invalid override path")
	}

	if !stringContains(err.Error(), "invalid override path") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
