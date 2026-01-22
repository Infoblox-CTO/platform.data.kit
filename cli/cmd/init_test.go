package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"type", "pipeline"},
		{"namespace", "default"},
		{"team", "my-team"},
		{"owner", ""},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := initCmd.Flags().Lookup(tt.flag)
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

func TestInitCmd_Args(t *testing.T) {
	// Test argument validation
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "one arg is valid",
			args:    []string{"my-package"},
			wantErr: false,
		},
		{
			name:    "no args is invalid",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "two args is invalid",
			args:    []string{"pkg1", "pkg2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := initCmd.Args(initCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitCmd_CreatePackage(t *testing.T) {
	// Test creating a new package
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "test-pkg")

	// Save and restore global flags
	oldType := initType
	oldNamespace := initNamespace
	oldTeam := initTeam
	oldOwner := initOwner
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initTeam = oldTeam
		initOwner = oldOwner
	}()

	initType = "pipeline"
	initNamespace = "data-team"
	initTeam = "analytics"
	initOwner = "test-user"

	// Change to temp dir so relative paths work
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	err := runInit(cmd, []string{"test-pkg"})

	if err != nil {
		t.Errorf("runInit() error = %v, want nil", err)
		return
	}

	// Verify dp.yaml was created
	dpPath := filepath.Join(pkgDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Error("dp.yaml was not created")
	}
}

func TestInitCmd_InvalidName(t *testing.T) {
	// Test that invalid package names are rejected
	invalidNames := []string{
		"My-Package",      // uppercase
		"pkg with spaces", // spaces
		"a",               // too short
		"123-pkg",         // starts with number
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			if isValidPackageName(name) {
				t.Errorf("isValidPackageName(%q) = true, want false", name)
			}
		})
	}
}

func TestInitCmd_ValidName(t *testing.T) {
	// Test that valid package names are accepted
	validNames := []string{
		"my-package",
		"analytics-pipeline",
		"data-team-etl",
		"abc",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			if !isValidPackageName(name) {
				t.Errorf("isValidPackageName(%q) = false, want true", name)
			}
		})
	}
}

func TestInitCmd_CurrentDirectory(t *testing.T) {
	// Test initializing in current directory with "."
	tmpDir := t.TempDir()

	// Create a subdirectory with a valid name
	pkgDir := filepath.Join(tmpDir, "valid-pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Save and restore global flags
	oldType := initType
	oldNamespace := initNamespace
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
	}()

	initType = "dataset"
	initNamespace = "default"

	oldWd, _ := os.Getwd()
	os.Chdir(pkgDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	err := runInit(cmd, []string{"."})

	if err != nil {
		t.Errorf("runInit(.) error = %v, want nil", err)
		return
	}

	// Verify dp.yaml was created
	dpPath := filepath.Join(pkgDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Error("dp.yaml was not created in current directory")
	}
}

func TestInitCmd_PackageTypes(t *testing.T) {
	// Test different package types
	types := []string{"pipeline", "model", "dataset"}

	for _, pkgType := range types {
		t.Run(pkgType, func(t *testing.T) {
			tmpDir := t.TempDir()
			pkgDir := filepath.Join(tmpDir, "test-"+pkgType)

			oldType := initType
			oldNamespace := initNamespace
			defer func() {
				initType = oldType
				initNamespace = oldNamespace
			}()

			initType = pkgType
			initNamespace = "default"

			oldWd, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(oldWd)

			cmd := &cobra.Command{}
			err := runInit(cmd, []string{"test-" + pkgType})

			if err != nil {
				t.Errorf("runInit() for type %s error = %v", pkgType, err)
				return
			}

			// Verify dp.yaml exists
			dpPath := filepath.Join(pkgDir, "dp.yaml")
			if _, err := os.Stat(dpPath); os.IsNotExist(err) {
				t.Errorf("dp.yaml not created for type %s", pkgType)
			}
		})
	}
}
