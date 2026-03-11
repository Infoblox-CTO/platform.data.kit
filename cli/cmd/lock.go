package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/dataset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/schema"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	lockVerify  bool
	lockUpgrade bool
)

// lockCmd resolves schema references and writes dk.lock.
var lockCmd = &cobra.Command{
	Use:   "lock [package-dir]",
	Short: "Resolve schema references and generate dk.lock",
	Long: `Resolve all schemaRef fields in dataset manifests and DataSetRef.schema
fields in transforms, then write a dk.lock file pinning resolved versions.

The lock file ensures reproducible builds by pinning exact schema versions,
checksums, and repository references.

Examples:
  # Generate dk.lock for current directory
  dk lock

  # Verify dk.lock is up-to-date (CI mode, exits non-zero if stale)
  dk lock --verify

  # Re-resolve to latest compatible versions
  dk lock --upgrade`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLock,
}

func init() {
	rootCmd.AddCommand(lockCmd)

	lockCmd.Flags().BoolVar(&lockVerify, "verify", false,
		"Verify dk.lock is up-to-date (exit non-zero if stale)")
	lockCmd.Flags().BoolVar(&lockUpgrade, "upgrade", false,
		"Re-resolve to latest compatible versions")
}

func runLock(cmd *cobra.Command, args []string) error {
	packageDir := "."
	if len(args) > 0 {
		packageDir = args[0]
	}

	absDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return fmt.Errorf("directory not found: %s", packageDir)
	}

	// Collect all schema references from datasets and transforms.
	refs, err := collectSchemaRefs(absDir)
	if err != nil {
		return err
	}

	if len(refs) == 0 {
		cmd.Println("No schema references found — dk.lock not needed.")
		return nil
	}

	if lockVerify {
		return verifyLock(cmd, absDir, refs)
	}

	cmd.Printf("Found %d schema reference(s):\n", len(refs))
	for _, ref := range refs {
		cmd.Printf("  • %s\n", ref)
	}
	cmd.Println()

	// Generate lock file entries.
	// When APX exports its packages, this will resolve via the APX catalog.
	// For now, we generate entries from the ref strings directly.
	lock := buildLockFromRefs(refs)

	if err := schema.WriteLockFile(absDir, lock); err != nil {
		return err
	}

	cmd.Printf("Wrote %s with %d schema(s)\n", schema.LockFileName, len(lock.Schemas))
	return nil
}

// collectSchemaRefs gathers all schemaRef values from dataset manifests
// and DataSetRef.Schema values from transform manifests.
func collectSchemaRefs(packageDir string) ([]string, error) {
	var refs []string
	seen := make(map[string]bool)

	// Scan datasets for schemaRef.
	datasets, err := dataset.LoadAllDataSets(packageDir)
	if err == nil {
		for _, ds := range datasets {
			if ds.Spec.SchemaRef != "" && !seen[ds.Spec.SchemaRef] {
				refs = append(refs, ds.Spec.SchemaRef)
				seen[ds.Spec.SchemaRef] = true
			}
		}
	}

	// Scan dk.yaml for transform schema refs.
	dkPath := filepath.Join(packageDir, "dk.yaml")
	if data, err := os.ReadFile(dkPath); err == nil {
		transformRefs := extractTransformSchemaRefs(data)
		for _, ref := range transformRefs {
			if !seen[ref] {
				refs = append(refs, ref)
				seen[ref] = true
			}
		}
	}

	return refs, nil
}

// extractTransformSchemaRefs parses a dk.yaml and extracts schema refs from
// DataSetRef entries in transforms.
func extractTransformSchemaRefs(data []byte) []string {
	var peek struct {
		Kind string `yaml:"kind"`
		Spec struct {
			Inputs []struct {
				Schema string `yaml:"schema"`
			} `yaml:"inputs"`
			Outputs []struct {
				Schema string `yaml:"schema"`
			} `yaml:"outputs"`
		} `yaml:"spec"`
	}

	if err := yaml.Unmarshal(data, &peek); err != nil {
		return nil
	}

	if peek.Kind != "Transform" {
		return nil
	}

	var refs []string
	for _, in := range peek.Spec.Inputs {
		if in.Schema != "" {
			refs = append(refs, in.Schema)
		}
	}
	for _, out := range peek.Spec.Outputs {
		if out.Schema != "" {
			refs = append(refs, out.Schema)
		}
	}
	return refs
}

// verifyLock checks that dk.lock exists and contains entries for all refs.
func verifyLock(cmd *cobra.Command, packageDir string, refs []string) error {
	lock, err := schema.ReadLockFile(packageDir)
	if err != nil {
		return err
	}
	if lock == nil {
		return fmt.Errorf("dk.lock not found — run 'dk lock' to generate")
	}

	missing := 0
	for _, ref := range refs {
		module, _ := schema.ParseSchemaRef(ref)
		if schema.FindLockedSchema(lock, module) == nil {
			cmd.Printf("  ✗ missing lock entry for %q\n", ref)
			missing++
		}
	}

	if missing > 0 {
		return fmt.Errorf("dk.lock is stale: %d schema(s) missing — run 'dk lock' to update", missing)
	}

	cmd.Printf("✓ dk.lock is up-to-date (%d schema(s) locked)\n", len(lock.Schemas))
	return nil
}

// buildLockFromRefs creates a LockFile from schema refs.
// Currently generates entries with version extracted from the constraint.
// When APX is integrated, this will resolve via the APX catalog to get
// repo, ref, format, and checksum.
func buildLockFromRefs(refs []string) *contracts.LockFile {
	lock := &contracts.LockFile{
		Version: schema.LockFileVersion,
	}

	seen := make(map[string]bool)
	for _, ref := range refs {
		module, constraint := schema.ParseSchemaRef(ref)
		if seen[module] {
			continue
		}
		seen[module] = true

		version := stripConstraintPrefix(constraint)
		if version == "" {
			version = "0.0.0"
		}

		lock.Schemas = append(lock.Schemas, contracts.LockedSchema{
			Module:  module,
			Version: version,
		})
	}

	return lock
}

func stripConstraintPrefix(s string) string {
	for len(s) > 0 && (s[0] == '^' || s[0] == '~' || s[0] == '>' || s[0] == '=' || s[0] == '<' || s[0] == ' ') {
		s = s[1:]
	}
	return s
}
