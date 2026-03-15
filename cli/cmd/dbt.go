package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
	"github.com/spf13/cobra"
)

var (
	dbtPackageDir string
	dbtCell       string
)

var dbtCmd = &cobra.Command{
	Use:   "dbt [dbt-args...]",
	Short: "Run dbt with DataKit store resolution",
	Long: `Run any dbt command with automatic store resolution and profiles.yml generation.

dk dbt resolves the Store graph from your dk.yaml manifests, injects
DK_STORE_DSN_* / DK_STORE_TYPE_* environment variables, generates
profiles.yml via the Python SDK (dk-profiles), and then executes dbt
with those settings. All extra arguments are passed through to dbt.

Examples:
  dk dbt run                          # build models
  dk dbt test                         # run dbt tests
  dk dbt run --select my_model        # run a specific model
  dk dbt debug                        # check connection
  dk dbt run --cell canary            # resolve stores from a cell
  dk dbt -- run --full-refresh        # explicit separator for dbt flags`,
	DisableFlagParsing: true,
	RunE:               runDBTCmd,
}

func init() {
	rootCmd.AddCommand(dbtCmd)
}

func runDBTCmd(cmd *cobra.Command, args []string) error {
	// Parse our flags from the front of args before passing the rest to dbt.
	dbtArgs, packageDir, cell := parseDbtFlags(args)

	if len(dbtArgs) == 0 || dbtArgs[0] == "--help" || dbtArgs[0] == "-h" {
		return cmd.Help()
	}

	// Resolve package directory.
	absDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Verify dk.yaml exists and is a dbt transform.
	dkPath := filepath.Join(absDir, "dk.yaml")
	if _, err := os.Stat(dkPath); os.IsNotExist(err) {
		return fmt.Errorf("dk.yaml not found in %s — is this a DK package?", packageDir)
	}

	m, _, err := manifest.ParseManifestFile(dkPath)
	if err != nil {
		return fmt.Errorf("parsing dk.yaml: %w", err)
	}

	transform, ok := m.(*contracts.Transform)
	if !ok {
		return fmt.Errorf("dk.yaml is not a Transform manifest")
	}

	if transform.Spec.Runtime != contracts.RuntimeDBT {
		return fmt.Errorf("dk dbt requires runtime: dbt (got %s)", transform.Spec.Runtime)
	}

	// Find dbt and dk-profiles binaries.
	dbtBin, err := exec.LookPath("dbt")
	if err != nil {
		return fmt.Errorf("dbt not found in PATH: install it from https://docs.getdbt.com/docs/core/installation-overview")
	}

	dkProfilesBin, err := exec.LookPath("dk-profiles")
	if err != nil {
		return fmt.Errorf("dk-profiles not found in PATH: install it with: pip install datakit-sdk")
	}

	// Resolve stores and build env vars.
	fmt.Printf("Resolving stores for %s...\n", transform.GetName())

	pm, err := loadPackageManifestsFromDir(absDir)
	if err != nil {
		return fmt.Errorf("loading manifests: %w", err)
	}

	if cell != "" {
		cellRes := runner.NewCellResolver(cell, "", nil)
		for name := range pm.stores {
			if s, err := cellRes.ResolveStore(nil, name); err == nil {
				pm.stores[name] = s
			}
		}
	}

	storeEnvs, err := runner.BuildStoreEnvVars(transform, pm.stores, pm.datasets)
	if err != nil {
		return fmt.Errorf("resolving stores: %w", err)
	}

	storeEnvMap := runner.StoreEnvsToMap(storeEnvs)

	// Build environment: OS + store env vars.
	env := os.Environ()
	for k, v := range storeEnvMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	env = append(env, fmt.Sprintf("DBT_PROFILES_DIR=%s", absDir))

	// Step 1: Generate profiles.yml.
	fmt.Println("Generating profiles.yml via dk-profiles...")

	profilesCmd := exec.Command(dkProfilesBin, "generate", "-o", absDir)
	profilesCmd.Dir = absDir
	profilesCmd.Env = env
	profilesCmd.Stdout = os.Stdout
	profilesCmd.Stderr = os.Stderr
	if err := profilesCmd.Run(); err != nil {
		return fmt.Errorf("dk-profiles generate failed: %w", err)
	}

	// Step 2: Run dbt with all remaining args.
	fmt.Printf("Running: dbt %s\n\n", joinArgs(dbtArgs))

	dbtExec := exec.Command(dbtBin, dbtArgs...)
	dbtExec.Dir = absDir
	dbtExec.Env = env
	dbtExec.Stdout = os.Stdout
	dbtExec.Stderr = os.Stderr
	dbtExec.Stdin = os.Stdin

	return dbtExec.Run()
}

// parseDbtFlags extracts --dir and --cell from the front of args,
// returning the remaining args to pass to dbt.
func parseDbtFlags(args []string) (dbtArgs []string, packageDir, cell string) {
	packageDir = "."
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--dir":
			if i+1 < len(args) {
				packageDir = args[i+1]
				i += 2
				continue
			}
		case "--cell":
			if i+1 < len(args) {
				cell = args[i+1]
				i += 2
				continue
			}
		case "--":
			// Everything after -- goes to dbt.
			return args[i+1:], packageDir, cell
		}
		dbtArgs = append(dbtArgs, args[i])
		i++
	}
	return dbtArgs, packageDir, cell
}

func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}

// packageManifestsLocal is a simplified manifest loader for the dbt command.
type packageManifestsLocal struct {
	stores   map[string]*contracts.Store
	datasets map[string]*contracts.DataSetManifest
}

func loadPackageManifestsFromDir(dir string) (*packageManifestsLocal, error) {
	pm := &packageManifestsLocal{
		stores:   make(map[string]*contracts.Store),
		datasets: make(map[string]*contracts.DataSetManifest),
	}

	// Load stores.
	storeDir := filepath.Join(dir, "store")
	if entries, err := os.ReadDir(storeDir); err == nil {
		parser := manifest.NewParser()
		for _, e := range entries {
			if e.IsDir() || !isYAML(e.Name()) {
				continue
			}
			data, err := os.ReadFile(filepath.Join(storeDir, e.Name()))
			if err != nil {
				continue
			}
			s, err := parser.ParseStore(data)
			if err != nil {
				continue
			}
			pm.stores[s.Metadata.Name] = s
		}
	}

	// Load datasets.
	dsDir := filepath.Join(dir, "dataset")
	if entries, err := os.ReadDir(dsDir); err == nil {
		parser := manifest.NewParser()
		for _, e := range entries {
			if e.IsDir() || !isYAML(e.Name()) {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dsDir, e.Name()))
			if err != nil {
				continue
			}
			ds, err := parser.ParseDataSet(data)
			if err != nil {
				continue
			}
			pm.datasets[ds.Metadata.Name] = ds
		}
	}

	return pm, nil
}

func isYAML(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}
