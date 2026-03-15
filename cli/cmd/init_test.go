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
	tests := []struct {
		flag     string
		defValue string
	}{
		{"runtime", ""},
		{"namespace", "default"},
		{"team", "my-team"},
		{"owner", ""},
		{"mode", "batch"},
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
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"one arg is valid", []string{"my-package"}, false},
		{"no args is valid (interactive)", []string{}, false},
		{"two args is invalid", []string{"pkg1", "pkg2"}, true},
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

func TestInitCmd_InvalidName(t *testing.T) {
	invalidNames := []string{
		"My-Package",
		"pkg with spaces",
		"a",
		"123-pkg",
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

func TestTitleCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"transform", "Transform"},
		{"connector", "Connector"},
		{"store", "Store"},
		{"", ""},
	}
	for _, tt := range tests {
		got := titleCase(tt.input)
		if got != tt.want {
			t.Errorf("titleCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// saveAndRestoreInitFlags saves all init flags and restores them at cleanup.
func saveAndRestoreInitFlags(t *testing.T) {
	t.Helper()
	saved := struct {
		runtime, mode, namespace, team, owner string
	}{initRuntime, initMode, initNamespace, initTeam, initOwner}
	t.Cleanup(func() {
		initRuntime = saved.runtime
		initMode = saved.mode
		initNamespace = saved.namespace
		initTeam = saved.team
		initOwner = saved.owner
	})
}

// runInitWithFlags is a test helper that runs init with specified flags.
func runInitWithFlags(t *testing.T, name string, flags map[string]string) (string, error) {
	t.Helper()
	tmpDir := t.TempDir()

	saveAndRestoreInitFlags(t)

	// Reset to defaults
	initRuntime = ""
	initMode = "batch"
	initNamespace = "default"
	initTeam = "my-team"
	initOwner = ""

	// Apply flags to vars
	for k, v := range flags {
		switch k {
		case "runtime":
			initRuntime = v
		case "mode":
			initMode = v
		case "namespace":
			initNamespace = v
		case "team":
			initTeam = v
		case "owner":
			initOwner = v
		}
	}

	// Change to temp dir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	// Create cobra command with flags matching init.go
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().StringVarP(&initRuntime, "runtime", "r", "", "")
	cmd.Flags().StringVarP(&initMode, "mode", "m", "batch", "")

	// Mark flags as changed
	for k, v := range flags {
		cmd.Flags().Set(k, v)
	}

	err := runInit(cmd, []string{name})
	return filepath.Join(tmpDir, name), err
}

// --- New Taxonomy Init Tests ---

func TestInitCmd_TransformCloudQuery(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "my-transform", map[string]string{
		"runtime": "cloudquery",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(pkgDir, "dk.yaml"))
	if err != nil {
		t.Fatalf("dk.yaml not created: %v", err)
	}
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Transform") {
		t.Error("dk.yaml should contain 'kind: Transform'")
	}
	if !strings.Contains(dpStr, "runtime: cloudquery") {
		t.Error("dk.yaml should contain 'runtime: cloudquery'")
	}
	if !strings.Contains(dpStr, "mode: batch") {
		t.Error("dk.yaml should contain 'mode: batch'")
	}

	// config.yaml should NOT be scaffolded — it is auto-generated at runtime by dk run.
	configPath := filepath.Join(pkgDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("config.yaml should not be scaffolded; it is auto-generated at runtime")
	}
}

func TestInitCmd_TransformDBT(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "user-aggregation", map[string]string{
		"runtime": "dbt",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(pkgDir, "dk.yaml"))
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Transform") {
		t.Error("dk.yaml should contain 'kind: Transform'")
	}
	if !strings.Contains(dpStr, "runtime: dbt") {
		t.Error("dk.yaml should contain 'runtime: dbt'")
	}

	for _, f := range []string{"dbt_project.yml", "profiles.yml", "models/example.sql"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected dbt file %q was not created", f)
		}
	}
}

func TestInitCmd_TransformGenericPython(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "fraud-scorer", map[string]string{
		"runtime": "generic-python", "mode": "streaming",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(pkgDir, "dk.yaml"))
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Transform") {
		t.Error("dk.yaml should contain 'kind: Transform'")
	}
	if !strings.Contains(dpStr, "mode: streaming") {
		t.Error("dk.yaml should contain 'mode: streaming'")
	}

	for _, f := range []string{"main.py", "requirements.txt"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestInitCmd_TransformGenericGo(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "data-worker", map[string]string{
		"runtime": "generic-go",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	for _, f := range []string{"dk.yaml", "go.mod", "main.go", "cmd/root.go", ".gitignore", ".dockerignore"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

// --- Validation Tests ---

func TestInitCmd_MissingRuntime(t *testing.T) {
	_, err := runInitWithFlags(t, "test-no-runtime", map[string]string{})
	if err == nil {
		t.Fatal("expected error when --runtime is not provided")
	}
	if !strings.Contains(err.Error(), "--runtime is required") {
		t.Errorf("error should mention --runtime is required, got: %v", err)
	}
}

func TestInitCmd_InvalidRuntime(t *testing.T) {
	_, err := runInitWithFlags(t, "test-bad-runtime", map[string]string{
		"runtime": "spark",
	})
	if err == nil {
		t.Fatal("expected error for invalid runtime")
	}
	if !strings.Contains(err.Error(), "invalid runtime") {
		t.Errorf("error should mention 'invalid runtime', got: %v", err)
	}
}

func TestInitCmd_InvalidMode(t *testing.T) {
	_, err := runInitWithFlags(t, "test-bad-mode", map[string]string{
		"runtime": "cloudquery", "mode": "real-time",
	})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("error should mention 'invalid mode', got: %v", err)
	}
}

func TestInitCmd_DBTStreamingRejected(t *testing.T) {
	_, err := runInitWithFlags(t, "dbt-streaming", map[string]string{
		"runtime": "dbt", "mode": "streaming",
	})
	if err == nil {
		t.Fatal("expected error for dbt + streaming")
	}
	if !strings.Contains(err.Error(), "streaming") {
		t.Errorf("error should mention streaming, got: %v", err)
	}
}

func TestInitCmd_DefaultKindIsTransform(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "default-kind-test", map[string]string{
		"runtime": "cloudquery",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}
	content, _ := os.ReadFile(filepath.Join(pkgDir, "dk.yaml"))
	if !strings.Contains(string(content), "kind: Transform") {
		t.Error("dk.yaml should contain 'kind: Transform'")
	}
}

func TestInitCmd_CurrentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "valid-pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	saveAndRestoreInitFlags(t)

	initRuntime = "cloudquery"
	initMode = "batch"
	initNamespace = "default"

	oldWd, _ := os.Getwd()
	os.Chdir(pkgDir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().StringVarP(&initRuntime, "runtime", "r", "", "")
	cmd.Flags().Set("runtime", "cloudquery")

	err := runInit(cmd, []string{"."})
	if err != nil {
		t.Errorf("runInit(.) error = %v", err)
		return
	}

	dkPath := filepath.Join(pkgDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		t.Error("dk.yaml was not created in current directory")
	}
}

func TestInitCmd_ProjectContextTransformsDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate a project context by creating a transforms/ directory.
	if err := os.MkdirAll(filepath.Join(tmpDir, "transforms"), 0755); err != nil {
		t.Fatalf("failed to create transforms dir: %v", err)
	}

	saveAndRestoreInitFlags(t)
	initRuntime = "cloudquery"
	initMode = "batch"
	initNamespace = "default"
	initTeam = "my-team"
	initOwner = ""

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().StringVarP(&initRuntime, "runtime", "r", "", "")
	cmd.Flags().Set("runtime", "cloudquery")

	err := runInit(cmd, []string{"my-transform"})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	dkPath := filepath.Join(tmpDir, "transforms", "my-transform", "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		t.Error("dk.yaml should be created under transforms/my-transform/, got created elsewhere")
	}

	// Should NOT exist at project root
	rootDkPath := filepath.Join(tmpDir, "my-transform", "dk.yaml")
	if _, err := os.Stat(rootDkPath); err == nil {
		t.Error("dk.yaml should not be created at project root when transforms/ dir exists")
	}
}

func TestInitCmd_NoNameNonInteractive(t *testing.T) {
	// When stdin is not a terminal (CI, tests), omitting the name arg should
	// produce a clear error rather than blocking on a prompt.
	saveAndRestoreInitFlags(t)
	initRuntime = "cloudquery"
	initMode = "batch"

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().StringVarP(&initRuntime, "runtime", "r", "", "")
	cmd.Flags().Set("runtime", "cloudquery")

	err := runInit(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when name omitted in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "package name is required") {
		t.Errorf("error should mention package name, got: %v", err)
	}
}
