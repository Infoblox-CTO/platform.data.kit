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

	// Legacy flags (deprecated but kept for backward compat)
	initType     string
	initLanguage string
	initRole     string
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

	// Legacy flags — hidden, mapped to new flags
	initCmd.Flags().StringVarP(&initType, "type", "t", "",
		"(deprecated) Package type — use --kind and --runtime instead")
	initCmd.Flags().StringVarP(&initLanguage, "language", "l", "",
		"(deprecated) Language — use --runtime instead")
	initCmd.Flags().StringVar(&initRole, "role", "",
		"(deprecated) Plugin role — use --kind instead")
	_ = initCmd.Flags().MarkHidden("type")
	_ = initCmd.Flags().MarkHidden("language")
	_ = initCmd.Flags().MarkHidden("role")
}

// mapLegacyFlags translates legacy --type/--language/--role flags to --kind/--runtime.
func mapLegacyFlags(cmd *cobra.Command) {
	// Map --type cloudquery + --role source → --kind source --runtime cloudquery
	if cmd.Flags().Changed("type") && !cmd.Flags().Changed("kind") {
		switch initType {
		case "cloudquery":
			if !cmd.Flags().Changed("runtime") {
				initRuntime = "cloudquery"
			}
			if cmd.Flags().Changed("role") {
				initKind = initRole // "source" or "destination"
			} else {
				initKind = "source"
			}
		case "pipeline":
			initKind = "model"
			if !cmd.Flags().Changed("runtime") {
				// Map language to runtime
				switch initLanguage {
				case "python":
					initRuntime = "generic-python"
				case "go", "":
					initRuntime = "generic-go"
				}
			}
		}
	}

	// Map --language to --runtime if --runtime wasn't explicitly set
	if cmd.Flags().Changed("language") && !cmd.Flags().Changed("runtime") {
		switch initLanguage {
		case "python":
			initRuntime = "generic-python"
		case "go":
			initRuntime = "generic-go"
		}
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Map legacy flags
	mapLegacyFlags(cmd)

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
	if initKind == "datapackage" {
		kind = contracts.KindModel // treat legacy DataPackage as Model
	}
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
		// Legacy fields for backward compat with old templates
		Language: initLanguage,
		Type:     initType,
		Role:     initRole,
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
func goPostScaffold(cmd *cobra.Command, dir string) error {
	// Require the go toolchain
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go toolchain not found: install Go from https://go.dev/dl/")
	}

	cmd.Printf("Running go mod tidy...\n")
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	cmd.Printf("Running go fmt...\n")
	gofmt := exec.Command("go", "fmt", "./...")
	gofmt.Dir = dir
	gofmt.Stdout = os.Stdout
	gofmt.Stderr = os.Stderr
	if err := gofmt.Run(); err != nil {
		// go fmt is non-critical; warn but don't fail
		cmd.PrintErrf("Warning: go fmt failed: %v\n", err)
	}

	return nil
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
