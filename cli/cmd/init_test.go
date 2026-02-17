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
		{"kind", "model"},
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
		{"no args is invalid", []string{}, true},
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
		{"source", "Source"},
		{"destination", "Destination"},
		{"model", "Model"},
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
		kind, runtime, mode, namespace, team, owner string
	}{initKind, initRuntime, initMode, initNamespace, initTeam, initOwner}
	t.Cleanup(func() {
		initKind = saved.kind
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
	initKind = "model"
	initRuntime = ""
	initMode = "batch"
	initNamespace = "default"
	initTeam = "my-team"
	initOwner = ""

	// Apply flags to vars
	for k, v := range flags {
		switch k {
		case "kind":
			initKind = v
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
	cmd.Flags().StringVarP(&initKind, "kind", "k", "model", "")
	cmd.Flags().StringVarP(&initRuntime, "runtime", "r", "", "")
	cmd.Flags().StringVarP(&initMode, "mode", "m", "batch", "")

	// Mark flags as changed so mapLegacyFlags detects them
	for k, v := range flags {
		cmd.Flags().Set(k, v)
	}

	err := runInit(cmd, []string{name})
	return filepath.Join(tmpDir, name), err
}

// --- New Taxonomy Init Tests ---

func TestInitCmd_ModelCloudQuery(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "my-model", map[string]string{
		"kind": "model", "runtime": "cloudquery",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	if err != nil {
		t.Fatalf("dp.yaml not created: %v", err)
	}
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Model") {
		t.Error("dp.yaml should contain 'kind: Model'")
	}
	if !strings.Contains(dpStr, "runtime: cloudquery") {
		t.Error("dp.yaml should contain 'runtime: cloudquery'")
	}
	if !strings.Contains(dpStr, "mode: batch") {
		t.Error("dp.yaml should contain 'mode: batch'")
	}

	configPath := filepath.Join(pkgDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.yaml should be created for model/cloudquery")
	}
}

func TestInitCmd_ModelDBT(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "user-aggregation", map[string]string{
		"kind": "model", "runtime": "dbt",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Model") {
		t.Error("dp.yaml should contain 'kind: Model'")
	}
	if !strings.Contains(dpStr, "runtime: dbt") {
		t.Error("dp.yaml should contain 'runtime: dbt'")
	}

	for _, f := range []string{"dbt_project.yml", "profiles.yml", "models/example.sql"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected dbt file %q was not created", f)
		}
	}
}

func TestInitCmd_ModelGenericPython(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "fraud-scorer", map[string]string{
		"kind": "model", "runtime": "generic-python", "mode": "streaming",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Model") {
		t.Error("dp.yaml should contain 'kind: Model'")
	}
	if !strings.Contains(dpStr, "mode: streaming") {
		t.Error("dp.yaml should contain 'mode: streaming'")
	}

	for _, f := range []string{"main.py", "requirements.txt"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestInitCmd_ModelGenericGo(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "data-worker", map[string]string{
		"kind": "model", "runtime": "generic-go",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	for _, f := range []string{"dp.yaml", "go.mod", "main.go", "cmd/root.go", ".gitignore", ".dockerignore"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

func TestInitCmd_SourceCloudQuery(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "pg-cdc-source", map[string]string{
		"kind": "source", "runtime": "cloudquery",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Source") {
		t.Error("dp.yaml should contain 'kind: Source'")
	}
	if !strings.Contains(dpStr, "runtime: cloudquery") {
		t.Error("dp.yaml should contain 'runtime: cloudquery'")
	}
	if !strings.Contains(dpStr, "provides:") {
		t.Error("dp.yaml should contain 'provides:' for source")
	}
}

func TestInitCmd_SourceGenericGo(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "http-poller", map[string]string{
		"kind": "source", "runtime": "generic-go",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	for _, f := range []string{"dp.yaml", "go.mod", "main.go", "cmd/root.go", ".gitignore", ".dockerignore"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}

	content, _ := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	if !strings.Contains(string(content), "kind: Source") {
		t.Error("dp.yaml should contain 'kind: Source'")
	}
}

func TestInitCmd_DestinationCloudQuery(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "s3-parquet-dest", map[string]string{
		"kind": "destination", "runtime": "cloudquery",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	dpStr := string(content)
	if !strings.Contains(dpStr, "kind: Destination") {
		t.Error("dp.yaml should contain 'kind: Destination'")
	}
	if !strings.Contains(dpStr, "accepts:") {
		t.Error("dp.yaml should contain 'accepts:' for destination")
	}
}

func TestInitCmd_DestinationGenericGo(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "s3-writer-dest", map[string]string{
		"kind": "destination", "runtime": "generic-go",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}

	for _, f := range []string{"dp.yaml", "go.mod", "main.go", "cmd/root.go", ".gitignore", ".dockerignore"} {
		if _, err := os.Stat(filepath.Join(pkgDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", f)
		}
	}
}

// --- Validation Tests ---

func TestInitCmd_MissingRuntime(t *testing.T) {
	_, err := runInitWithFlags(t, "test-no-runtime", map[string]string{
		"kind": "model",
	})
	if err == nil {
		t.Fatal("expected error when --runtime is not provided")
	}
	if !strings.Contains(err.Error(), "--runtime is required") {
		t.Errorf("error should mention --runtime is required, got: %v", err)
	}
}

func TestInitCmd_InvalidKind(t *testing.T) {
	_, err := runInitWithFlags(t, "test-bad-kind", map[string]string{
		"kind": "widget", "runtime": "cloudquery",
	})
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
	if !strings.Contains(err.Error(), "invalid kind") {
		t.Errorf("error should mention 'invalid kind', got: %v", err)
	}
}

func TestInitCmd_InvalidRuntime(t *testing.T) {
	_, err := runInitWithFlags(t, "test-bad-runtime", map[string]string{
		"kind": "model", "runtime": "spark",
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
		"kind": "model", "runtime": "cloudquery", "mode": "real-time",
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
		"kind": "model", "runtime": "dbt", "mode": "streaming",
	})
	if err == nil {
		t.Fatal("expected error for dbt + streaming")
	}
	if !strings.Contains(err.Error(), "streaming") {
		t.Errorf("error should mention streaming, got: %v", err)
	}
}

func TestInitCmd_DBTForSourceRejected(t *testing.T) {
	_, err := runInitWithFlags(t, "source-dbt", map[string]string{
		"kind": "source", "runtime": "dbt",
	})
	if err == nil {
		t.Fatal("expected error for source + dbt")
	}
	if !strings.Contains(err.Error(), "only supported for model") {
		t.Errorf("error should mention 'only supported for model', got: %v", err)
	}
}

func TestInitCmd_DefaultKindIsModel(t *testing.T) {
	pkgDir, err := runInitWithFlags(t, "default-kind-test", map[string]string{
		"runtime": "cloudquery",
	})
	if err != nil {
		t.Fatalf("runInit() error = %v", err)
	}
	content, _ := os.ReadFile(filepath.Join(pkgDir, "dp.yaml"))
	if !strings.Contains(string(content), "kind: Model") {
		t.Error("dp.yaml should default to 'kind: Model'")
	}
}

func TestInitCmd_CurrentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "valid-pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	saveAndRestoreInitFlags(t)

	initKind = "model"
	initRuntime = "cloudquery"
	initMode = "batch"
	initNamespace = "default"

	oldWd, _ := os.Getwd()
	os.Chdir(pkgDir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().StringVarP(&initKind, "kind", "k", "model", "")
	cmd.Flags().StringVarP(&initRuntime, "runtime", "r", "", "")
	cmd.Flags().Set("runtime", "cloudquery")

	err := runInit(cmd, []string{"."})
	if err != nil {
		t.Errorf("runInit(.) error = %v", err)
		return
	}

	dpPath := filepath.Join(pkgDir, "dp.yaml")
	if _, err := os.Stat(dpPath); os.IsNotExist(err) {
		t.Error("dp.yaml was not created in current directory")
	}
}
