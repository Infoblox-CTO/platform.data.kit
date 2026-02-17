package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/cli/internal/output"
	"github.com/Infoblox-CTO/platform.data.kit/cli/internal/templates"
	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/spf13/cobra"
)

var (
	initKind      string // Source, Destination, Model
	initRuntime   string // cloudquery, generic-go, generic-python, dbt
	initMode      string // batch, streaming (model only)
	initNamespace string
	initTeam      string
	initOwner     string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new data package",
	Long: `Initialize a new data package with the required manifest files.

Supported kinds: source, destination, model (default)
Supported runtimes: cloudquery, generic-go, generic-python, dbt

This command creates a new directory with dp.yaml and project
files pre-configured with sensible defaults for the selected kind and runtime.

Examples:
  # Create a new model with CloudQuery runtime (default kind)
  dp init my-model --runtime cloudquery

  # Create a source extension (infra engineer)
  dp init pg-cdc --kind source --runtime cloudquery

  # Create a destination extension in Go
  dp init s3-writer --kind destination --runtime generic-go

  # Create a dbt model
  dp init user-aggregation --kind model --runtime dbt

  # Create a Python model for streaming
  dp init fraud-scorer --kind model --runtime generic-python --mode streaming

  # Create with custom namespace
  dp init my-model --runtime cloudquery --namespace data-team`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initKind, "kind", "k", "model",
		"Manifest kind: source, destination, model")
	initCmd.Flags().StringVarP(&initRuntime, "runtime", "r", "",
		"Runtime: cloudquery, generic-go, generic-python, dbt")
	initCmd.Flags().StringVarP(&initMode, "mode", "m", "batch",
		"Execution mode: batch, streaming (model only)")
	initCmd.Flags().StringVarP(&initNamespace, "namespace", "n", "default",
		"Package namespace")
	initCmd.Flags().StringVar(&initTeam, "team", "my-team",
		"Team label")
	initCmd.Flags().StringVar(&initOwner, "owner", "",
		"Package owner (defaults to current user)")
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if name != "." && !isValidPackageName(name) {
		return fmt.Errorf("invalid package name %q: must be DNS-safe (lowercase, alphanumeric, hyphens, 3-63 chars)", name)
	}

	// Get the target directory
	var targetDir string
	if name == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		targetDir = cwd
		name = filepath.Base(cwd)
	} else {
		targetDir = name
	}

	// Check if directory exists and is not empty
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(targetDir)
		if len(entries) > 0 && targetDir != "." {
			return fmt.Errorf("directory %q already exists and is not empty", targetDir)
		}
	}

	// Normalize and validate kind
	initKind = strings.ToLower(initKind)
	kind := contracts.Kind(titleCase(initKind))
	switch kind {
	case contracts.KindSource, contracts.KindDestination, contracts.KindModel:
		// valid
	default:
		return fmt.Errorf("invalid kind %q: must be source, destination, or model", initKind)
	}

	// Validate runtime is provided
	if initRuntime == "" {
		return fmt.Errorf("--runtime is required: cloudquery, generic-go, generic-python, dbt")
	}

	// Validate runtime
	runtime := contracts.Runtime(initRuntime)
	if !runtime.IsValid() {
		return fmt.Errorf("invalid runtime %q: must be cloudquery, generic-go, generic-python, or dbt", initRuntime)
	}

	// Validate mode (only meaningful for model kind)
	mode := contracts.Mode(initMode)
	if kind == contracts.KindModel {
		if !mode.IsValid() {
			return fmt.Errorf("invalid mode %q: must be batch or streaming", initMode)
		}
	}

	// Validate kind-runtime combinations
	if kind == contracts.KindModel && runtime == contracts.RuntimeDBT && mode == contracts.ModeStreaming {
		return fmt.Errorf("dbt runtime does not support streaming mode")
	}

	// Source and Destination only support cloudquery and generic-go runtimes
	if kind == contracts.KindSource || kind == contracts.KindDestination {
		if runtime != contracts.RuntimeCloudQuery && runtime != contracts.RuntimeGenericGo {
			return fmt.Errorf("%s runtime is only supported for model kind; use cloudquery or generic-go for %s", runtime, kind)
		}
	}

	// Set default owner
	if initOwner == "" {
		initOwner = fmt.Sprintf("%s-team", initNamespace)
	}

	// Create renderer
	renderer, err := templates.NewRenderer()
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	config := &templates.PackageConfig{
		Name:        name,
		Namespace:   initNamespace,
		Team:        initTeam,
		Description: fmt.Sprintf("A %s %s package", initRuntime, strings.ToLower(string(kind))),
		Owner:       initOwner,
		Kind:        strings.ToLower(string(kind)),
		Runtime:     string(runtime),
		Mode:        string(mode.Default()),
		Version:     Version,
	}

	// Use kind-based directory rendering
	if err := renderer.RenderKindDirectory(targetDir, config); err != nil {
		return fmt.Errorf("failed to scaffold %s/%s project: %w", kind, runtime, err)
	}

	output.PrintSuccess(cmd.OutOrStdout(), fmt.Sprintf("Scaffolded %s (%s) in %s", kind, runtime, targetDir))

	// Go runtimes: run go mod tidy + go fmt
	if runtime == contracts.RuntimeGenericGo {
		if err := goPostScaffold(cmd, targetDir); err != nil {
			cmd.PrintErrf("Warning: go post-scaffold failed: %v\n", err)
		}
	}

	cmd.Printf("\nPackage %q initialized successfully!\n", name)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit dp.yaml to configure your package\n")

	switch kind {
	case contracts.KindSource:
		cmd.Printf("  2. Configure your source in dp.yaml (provides, configSchema)\n")
		cmd.Printf("  3. Run 'dp lint' to validate\n")
		cmd.Printf("  4. Run 'dp build' to bundle\n")
		cmd.Printf("  5. Run 'dp publish' to publish to extension registry\n")
	case contracts.KindDestination:
		cmd.Printf("  2. Configure your destination in dp.yaml (accepts, configSchema)\n")
		cmd.Printf("  3. Run 'dp lint' to validate\n")
		cmd.Printf("  4. Run 'dp build' to bundle\n")
		cmd.Printf("  5. Run 'dp publish' to publish to extension registry\n")
	case contracts.KindModel:
		cmd.Printf("  2. Configure source/destination references and schedule\n")
		cmd.Printf("  3. Run 'dp lint' to validate\n")
		cmd.Printf("  4. Run 'dp dev up' to start local environment\n")
		cmd.Printf("  5. Run 'dp run' to execute locally\n")
	}

	return nil
}

// goPostScaffold runs go mod tidy and go fmt on a scaffolded Go project so
// the generated code compiles immediately (go.sum present, source formatted).
// It also detects parent go.work files and creates a local go.work to isolate
// the new module so `go build` works without GOWORK=off.
func goPostScaffold(cmd *cobra.Command, dir string) error {
	// Require the go toolchain
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go toolchain not found: install Go from https://go.dev/dl/")
	}

	// If a parent go.work exists, create a local go.work to isolate this module.
	// Without this, `go build` would fail because the new module isn't listed
	// in the parent workspace.
	if parentGoWork := findParentGoWork(dir); parentGoWork != "" {
		cmd.Printf("Detected parent go.work at %s, creating local go.work...\n", parentGoWork)
		localWork := filepath.Join(dir, "go.work")
		// Read go directive from go.mod to keep versions consistent.
		goVersion := readGoDirective(filepath.Join(dir, "go.mod"))
		content := fmt.Sprintf("go %s\n\nuse .\n", goVersion)
		if err := os.WriteFile(localWork, []byte(content), 0644); err != nil {
			cmd.PrintErrf("Warning: failed to create local go.work: %v\n", err)
		}
	}

	// Use GOWORK=off for tidy/fmt so they succeed even inside a parent workspace
	// that doesn't list this module yet.
	env := append(os.Environ(), "GOWORK=off")

	cmd.Printf("Running go mod tidy...\n")
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Env = env
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	cmd.Printf("Running go fmt...\n")
	gofmt := exec.Command("go", "fmt", "./...")
	gofmt.Dir = dir
	gofmt.Env = env
	gofmt.Stdout = os.Stdout
	gofmt.Stderr = os.Stderr
	if err := gofmt.Run(); err != nil {
		// go fmt is non-critical; warn but don't fail
		cmd.PrintErrf("Warning: go fmt failed: %v\n", err)
	}

	return nil
}

// findParentGoWork walks up from dir looking for a go.work file in an ancestor
// directory. Returns the path to go.work if found, or empty string.
func findParentGoWork(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	// Start from the parent of the scaffolded directory.
	current := filepath.Dir(abs)
	for {
		candidate := filepath.Join(current, "go.work")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

// readGoDirective reads the "go X.Y" directive from a go.mod file.
// Returns "1.21" as a fallback if the file can't be read.
func readGoDirective(goModPath string) string {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "1.21"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	return "1.21"
}

// isValidPackageName checks if a name is DNS-safe
func isValidPackageName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	matched, _ := regexp.MatchString("^[a-z][a-z0-9-]*[a-z0-9]$", name)
	return matched
}

// titleCase capitalizes the first letter of a string.
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
