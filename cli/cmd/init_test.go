package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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
		{"mode", "batch"},
		{"role", "source"},
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

	initType = "pipeline"
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
	types := []string{"pipeline"}

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

func TestInitCmd_BatchMode(t *testing.T) {
	// Test initializing with batch mode (default)
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "batch-pipeline")

	// Save and restore global flags
	oldType := initType
	oldNamespace := initNamespace
	oldMode := initMode
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initMode = oldMode
	}()

	initType = "pipeline"
	initNamespace = "default"
	initMode = "batch"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	err := runInit(cmd, []string{"batch-pipeline"})

	if err != nil {
		t.Errorf("runInit() error = %v, want nil", err)
		return
	}

	// Verify dp.yaml was created and contains batch mode
	dpPath := filepath.Join(pkgDir, "dp.yaml")
	content, err := os.ReadFile(dpPath)
	if err != nil {
		t.Errorf("failed to read dp.yaml: %v", err)
		return
	}

	// Check that batch mode is set in spec.runtime
	if !strings.Contains(string(content), "mode: batch") {
		t.Error("dp.yaml should contain 'mode: batch' in spec.runtime")
	}
}

func TestInitCmd_StreamingMode(t *testing.T) {
	// Test initializing with streaming mode
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "stream-pipeline")

	// Save and restore global flags
	oldType := initType
	oldNamespace := initNamespace
	oldMode := initMode
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initMode = oldMode
	}()

	initType = "pipeline"
	initNamespace = "default"
	initMode = "streaming"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	err := runInit(cmd, []string{"stream-pipeline"})

	if err != nil {
		t.Errorf("runInit() error = %v, want nil", err)
		return
	}

	// Verify dp.yaml was created and contains streaming mode
	dpPath := filepath.Join(pkgDir, "dp.yaml")
	content, err := os.ReadFile(dpPath)
	if err != nil {
		t.Errorf("failed to read dp.yaml: %v", err)
		return
	}

	// Check that streaming mode is set in spec.runtime
	if !strings.Contains(string(content), "mode: streaming") {
		t.Error("dp.yaml should contain 'mode: streaming' in spec.runtime")
	}
}

func TestInitCmd_InvalidMode(t *testing.T) {
	// Test that invalid mode is rejected
	tmpDir := t.TempDir()

	// Save and restore global flags
	oldType := initType
	oldNamespace := initNamespace
	oldMode := initMode
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initMode = oldMode
	}()

	initType = "pipeline"
	initNamespace = "default"
	initMode = "invalid-mode"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	err := runInit(cmd, []string{"test-pipeline"})

	if err == nil {
		t.Error("runInit() expected error for invalid mode, got nil")
	}
}

// --- CloudQuery Init Tests (T032) ---

func TestInitCmd_CloudQueryPython(t *testing.T) {
	// Test that dp init --type cloudquery --lang python scaffolds all expected files
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "my-cq-source")

	// Save and restore global flags
	oldType := initType
	oldNamespace := initNamespace
	oldTeam := initTeam
	oldOwner := initOwner
	oldLanguage := initLanguage
	oldMode := initMode
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initTeam = oldTeam
		initOwner = oldOwner
		initLanguage = oldLanguage
		initMode = oldMode
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "acme"
	initTeam = "data-eng"
	initOwner = "test-owner"
	initLanguage = "python"
	initMode = "batch" // should be ignored for cloudquery
	initRole = "source"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := runInit(cmd, []string{"my-cq-source"})

	if err != nil {
		t.Fatalf("runInit() error = %v, want nil", err)
	}

	// Verify all expected files are created (Python CloudQuery scaffold)
	expectedFiles := []string{
		"dp.yaml",
		"main.py",
		"pyproject.toml",
		"requirements.txt",
		"plugin/__init__.py",
		"plugin/plugin.py",
		"plugin/client.py",
		"plugin/spec.py",
		"plugin/tables/__init__.py",
		"plugin/tables/example_resource.py",
		"tests/test_example_resource.py",
		".gitignore",
		"Makefile",
		".datakit/Makefile.common",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(pkgDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}

	// Verify dp.yaml has correct template variables substituted
	dpContent, err := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	if err != nil {
		t.Fatalf("failed to read dp.yaml: %v", err)
	}
	dpStr := string(dpContent)

	if !strings.Contains(dpStr, "my-cq-source") {
		t.Error("dp.yaml should contain package name 'my-cq-source'")
	}
	if !strings.Contains(dpStr, "acme") {
		t.Error("dp.yaml should contain namespace 'acme'")
	}
	if !strings.Contains(dpStr, "cloudquery") {
		t.Error("dp.yaml should contain type 'cloudquery'")
	}
	if !strings.Contains(dpStr, "source") {
		t.Error("dp.yaml should contain role 'source'")
	}

	// T006: Verify pyproject.toml has correct Python version constraint
	pyprojectContent, err := os.ReadFile(filepath.Join(pkgDir, "pyproject.toml"))
	if err != nil {
		t.Fatalf("failed to read pyproject.toml: %v", err)
	}
	pyprojectStr := string(pyprojectContent)

	if !strings.Contains(pyprojectStr, `requires-python = ">=3.12"`) {
		t.Error("pyproject.toml should contain requires-python >= 3.12")
	}
	if strings.Contains(pyprojectStr, "3.13") {
		t.Error("pyproject.toml should NOT contain any 3.13 references")
	}
}

func TestInitCmd_CloudQueryDefaultLanguage(t *testing.T) {
	// Test that cloudquery type defaults language to python when not explicitly set
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "auto-lang-pkg")

	oldType := initType
	oldNamespace := initNamespace
	oldLanguage := initLanguage
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initLanguage = oldLanguage
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "default"
	initLanguage = "go" // will be overridden to python since --language flag not "Changed"
	initRole = "source"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Use a cobra command where the language flag is NOT marked as changed
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	// Simulate: the --language flag was NOT explicitly set by user
	// In runInit, cmd.Flags().Changed("language") returns false → defaults to python
	err := runInit(cmd, []string{"auto-lang-pkg"})

	if err != nil {
		t.Fatalf("runInit() error = %v, want nil", err)
	}

	// Should have created Python files (main.py), not Go files
	mainPy := filepath.Join(pkgDir, "main.py")
	if _, err := os.Stat(mainPy); os.IsNotExist(err) {
		t.Error("expected main.py to be created (python is default for cloudquery)")
	}
}

func TestInitCmd_CloudQueryRoleDestinationRejected(t *testing.T) {
	// Test that --role destination is rejected with helpful message
	tmpDir := t.TempDir()

	oldType := initType
	oldNamespace := initNamespace
	oldLanguage := initLanguage
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initLanguage = oldLanguage
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "default"
	initLanguage = "python"
	initRole = "destination"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := runInit(cmd, []string{"dest-plugin"})

	if err == nil {
		t.Fatal("runInit() expected error for destination role, got nil")
	}

	if !strings.Contains(err.Error(), "not yet supported") {
		t.Errorf("error should mention 'not yet supported', got: %v", err)
	}
}

func TestInitCmd_CloudQueryInvalidRole(t *testing.T) {
	// Test that invalid role values are rejected
	tmpDir := t.TempDir()

	oldType := initType
	oldNamespace := initNamespace
	oldLanguage := initLanguage
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initLanguage = oldLanguage
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "default"
	initLanguage = "python"
	initRole = "transformer"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := runInit(cmd, []string{"bad-role-pkg"})

	if err == nil {
		t.Fatal("runInit() expected error for invalid role, got nil")
	}

	if !strings.Contains(err.Error(), "invalid role") {
		t.Errorf("error should mention 'invalid role', got: %v", err)
	}
}

func TestInitCmd_CloudQueryTypeAccepted(t *testing.T) {
	// Test that "cloudquery" is accepted by isValidPackageType
	if !isValidPackageType("cloudquery") {
		t.Error("isValidPackageType('cloudquery') = false, want true")
	}
	// Pipeline should still be valid
	if !isValidPackageType("pipeline") {
		t.Error("isValidPackageType('pipeline') = false, want true")
	}
	// Invalid types should still be rejected
	if isValidPackageType("invalid") {
		t.Error("isValidPackageType('invalid') = true, want false")
	}
}

func TestInitCmd_CloudQueryModeSkipped(t *testing.T) {
	// Test that invalid mode does NOT cause error for cloudquery type
	// (mode is pipeline-specific, should be skipped for cloudquery)
	tmpDir := t.TempDir()

	oldType := initType
	oldNamespace := initNamespace
	oldLanguage := initLanguage
	oldMode := initMode
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initLanguage = oldLanguage
		initMode = oldMode
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "default"
	initLanguage = "python"
	initMode = "invalid-mode" // should be ignored for cloudquery
	initRole = "source"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := runInit(cmd, []string{"mode-test-pkg"})

	if err != nil {
		t.Errorf("runInit() error = %v; mode validation should be skipped for cloudquery", err)
	}
}

func TestInitCmd_CloudQueryNextSteps(t *testing.T) {
	// Test that cloudquery init prints CloudQuery-specific next steps
	tmpDir := t.TempDir()

	oldType := initType
	oldNamespace := initNamespace
	oldLanguage := initLanguage
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initLanguage = oldLanguage
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "default"
	initLanguage = "python"
	initRole = "source"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	err := runInit(cmd, []string{"steps-test-pkg"})

	if err != nil {
		t.Fatalf("runInit() error = %v, want nil", err)
	}

	output := buf.String()
	if !strings.Contains(output, "plugin/tables/") {
		t.Error("next steps should mention plugin/tables/ for cloudquery")
	}
	if !strings.Contains(output, "dp run") {
		t.Error("next steps should mention 'dp run' for cloudquery")
	}
}

func TestInitCmd_CloudQueryPackageTypes(t *testing.T) {
	// Test that cloudquery type works in the PackageTypes test pattern
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "test-cloudquery")

	oldType := initType
	oldNamespace := initNamespace
	oldLanguage := initLanguage
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initLanguage = oldLanguage
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "default"
	initLanguage = "python"
	initRole = "source"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	err := runInit(cmd, []string{"test-cloudquery"})

	if err != nil {
		t.Errorf("runInit() for type cloudquery error = %v", err)
		return
	}

	// Verify dp.yaml exists
	dpPath := filepath.Join(pkgDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Errorf("dp.yaml not created for type cloudquery")
	}
}

// --- CloudQuery Go Init Tests (T043) ---

func TestInitCmd_CloudQueryGo(t *testing.T) {
	// Test that dp init --type cloudquery --lang go scaffolds all expected files
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "my-go-source")

	// Save and restore global flags
	oldType := initType
	oldNamespace := initNamespace
	oldTeam := initTeam
	oldOwner := initOwner
	oldLanguage := initLanguage
	oldMode := initMode
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initTeam = oldTeam
		initOwner = oldOwner
		initLanguage = oldLanguage
		initMode = oldMode
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "acme"
	initTeam = "data-eng"
	initOwner = "test-owner"
	initLanguage = "go"
	initMode = "batch" // should be ignored for cloudquery
	initRole = "source"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	// Register and set the language flag so Changed("language") returns true
	cmd.Flags().StringVarP(&initLanguage, "language", "l", "go", "Pipeline language")
	cmd.Flags().Set("language", "go")
	err := runInit(cmd, []string{"my-go-source"})

	if err != nil {
		t.Fatalf("runInit() error = %v, want nil", err)
	}

	// Verify all expected files are created (Go CloudQuery scaffold)
	expectedFiles := []string{
		"dp.yaml",
		"main.go",
		"go.mod",
		"resources/plugin/plugin.go",
		"internal/client/client.go",
		"internal/client/spec.go",
		"internal/tables/example_resource.go",
		"internal/tables/example_resource_test.go",
		".gitignore",
		"Makefile",
		".datakit/Makefile.common",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(pkgDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}

	// Verify dp.yaml has correct template variables substituted
	dpContent, err := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	if err != nil {
		t.Fatalf("failed to read dp.yaml: %v", err)
	}
	dpStr := string(dpContent)

	if !strings.Contains(dpStr, "my-go-source") {
		t.Error("dp.yaml should contain package name 'my-go-source'")
	}
	if !strings.Contains(dpStr, "acme") {
		t.Error("dp.yaml should contain namespace 'acme'")
	}
	if !strings.Contains(dpStr, "cloudquery") {
		t.Error("dp.yaml should contain type 'cloudquery'")
	}

	// Verify go.mod has correct module name
	goModContent, err := os.ReadFile(filepath.Join(pkgDir, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if !strings.Contains(string(goModContent), "module my-go-source") {
		t.Error("go.mod should contain module name 'my-go-source'")
	}

	// Verify main.go has correct import
	mainContent, err := os.ReadFile(filepath.Join(pkgDir, "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}
	if !strings.Contains(string(mainContent), "my-go-source/resources/plugin") {
		t.Error("main.go should import 'my-go-source/resources/plugin'")
	}
}

func TestInitCmd_CloudQueryGoExplicitLang(t *testing.T) {
	// Test that explicit --lang go works for cloudquery type
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "explicit-go-pkg")

	oldType := initType
	oldNamespace := initNamespace
	oldLanguage := initLanguage
	oldRole := initRole
	defer func() {
		initType = oldType
		initNamespace = oldNamespace
		initLanguage = oldLanguage
		initRole = oldRole
	}()

	initType = "cloudquery"
	initNamespace = "default"
	initLanguage = "go"
	initRole = "source"

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	// Register and set the language flag so Changed("language") returns true
	cmd.Flags().StringVarP(&initLanguage, "language", "l", "go", "Pipeline language")
	cmd.Flags().Set("language", "go")
	err := runInit(cmd, []string{"explicit-go-pkg"})

	if err != nil {
		t.Fatalf("runInit() error = %v, want nil", err)
	}

	// Verify Go-specific files exist (main.go, go.mod, not main.py)
	mainGo := filepath.Join(pkgDir, "main.go")
	if _, err := os.Stat(mainGo); os.IsNotExist(err) {
		t.Error("expected main.go to be created for --lang go")
	}

	goMod := filepath.Join(pkgDir, "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		t.Error("expected go.mod to be created for --lang go")
	}

	// Verify Python files do NOT exist
	mainPy := filepath.Join(pkgDir, "main.py")
	if _, err := os.Stat(mainPy); err == nil {
		t.Error("main.py should not exist for --lang go")
	}
}
