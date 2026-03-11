package schema

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

const (
	// LockFileName is the standard lock file name.
	LockFileName = "dk.lock"

	// LockFileVersion is the current lock file format version.
	LockFileVersion = "1"
)

// ReadLockFile reads and parses a dk.lock file from the given directory.
// Returns nil (no error) if the file does not exist.
func ReadLockFile(dir string) (*contracts.LockFile, error) {
	path := filepath.Join(dir, LockFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", LockFileName, err)
	}

	var lock contracts.LockFile
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", LockFileName, err)
	}
	return &lock, nil
}

// WriteLockFile writes a dk.lock file to the given directory.
func WriteLockFile(dir string, lock *contracts.LockFile) error {
	if lock.Version == "" {
		lock.Version = LockFileVersion
	}

	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", LockFileName, err)
	}

	path := filepath.Join(dir, LockFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", LockFileName, err)
	}
	return nil
}

// ResolveLock resolves a list of schema references to a lock file.
// Each ref is resolved via the SchemaResolver, producing a LockedSchema entry.
func ResolveLock(ctx context.Context, resolver SchemaResolver, refs []string) (*contracts.LockFile, error) {
	lock := &contracts.LockFile{
		Version: LockFileVersion,
	}

	seen := make(map[string]bool)
	for _, ref := range refs {
		module, _ := ParseSchemaRef(ref)
		if seen[module] {
			continue
		}
		seen[module] = true

		resolved, err := resolver.Resolve(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("resolving %q: %w", ref, err)
		}

		lock.Schemas = append(lock.Schemas, contracts.LockedSchema{
			Module:   resolved.Module.ID,
			Version:  resolved.Module.Version,
			Repo:     resolved.Module.Repo,
			Ref:      "", // populated when git-based resolution is available
			Format:   resolved.Module.Format,
			Checksum: resolved.Checksum,
		})
	}

	return lock, nil
}

// FindLockedSchema looks up a module in the lock file by name.
// Returns nil if not found.
func FindLockedSchema(lock *contracts.LockFile, module string) *contracts.LockedSchema {
	if lock == nil {
		return nil
	}
	for i := range lock.Schemas {
		if lock.Schemas[i].Module == module {
			return &lock.Schemas[i]
		}
	}
	return nil
}

// SyntheticDataSet creates a minimal DataSetManifest from a locked schema entry.
// This allows the runtime resolver to treat locked schemas like local datasets.
func SyntheticDataSet(locked contracts.LockedSchema) *contracts.DataSetManifest {
	return &contracts.DataSetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "DataSet",
		Metadata: contracts.DataSetMetadata{
			Name:    locked.Module,
			Version: locked.Version,
			Labels: map[string]string{
				"schema.source": "lock",
			},
		},
		Spec: contracts.DataSetSpec{
			Format:    locked.Format,
			SchemaRef: FormatSchemaRef(locked.Module, locked.Version),
		},
	}
}
