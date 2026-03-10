// Package dataset provides dataset loading and scaffolding for data package datasets.
package dataset

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"gopkg.in/yaml.v3"
)

// LoadDataSet loads and parses a single dataset.yaml from the given path.
// The path can be:
//   - A directory containing dataset.yaml
//   - A direct path to a dataset.yaml file
func LoadDataSet(path string) (*contracts.DataSetManifest, error) {
	// If path is a directory, look for dataset.yaml inside it
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("dataset path not found: %w", err)
	}

	datasetPath := path
	if info.IsDir() {
		datasetPath = filepath.Join(path, "dataset.yaml")
	}

	data, err := os.ReadFile(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dataset file %s: %w", datasetPath, err)
	}

	var ds contracts.DataSetManifest
	if err := yaml.Unmarshal(data, &ds); err != nil {
		return nil, fmt.Errorf("failed to parse dataset file %s: %w", datasetPath, err)
	}

	return &ds, nil
}

// LoadAllDataSets discovers and loads all dataset.yaml files from the datasets/ directory
// under the given project directory.
//
//	datasets/<name>/dataset.yaml
func LoadAllDataSets(projectDir string) ([]*contracts.DataSetManifest, error) {
	datasetsDir := filepath.Join(projectDir, "datasets")

	info, err := os.Stat(datasetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No datasets directory is fine
		}
		return nil, fmt.Errorf("failed to access datasets directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("datasets path is not a directory: %s", datasetsDir)
	}

	var datasets []*contracts.DataSetManifest

	err = filepath.WalkDir(datasetsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if d.Name() != "dataset.yaml" {
			return nil
		}

		ds, loadErr := LoadDataSet(path)
		if loadErr != nil {
			return fmt.Errorf("failed to load %s: %w", relativePath(projectDir, path), loadErr)
		}

		datasets = append(datasets, ds)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return datasets, nil
}

// FindDataSetByName searches for a dataset by name in the datasets/ directory.
func FindDataSetByName(projectDir, name string) (*contracts.DataSetManifest, error) {
	datasets, err := LoadAllDataSets(projectDir)
	if err != nil {
		return nil, err
	}

	for _, ds := range datasets {
		if ds.Metadata.Name == name {
			return ds, nil
		}
	}

	return nil, fmt.Errorf("dataset %q not found in %s", name, filepath.Join(projectDir, "datasets"))
}

// DataSetPath returns the expected filesystem path for a dataset based on its name.
// Layout: datasets/<name>/dataset.yaml
func DataSetPath(projectDir string, name string) string {
	return filepath.Join(projectDir, "datasets", name, "dataset.yaml")
}

// DataSetDir returns the expected directory path for a dataset based on its name.
// Layout: datasets/<name>/
func DataSetDir(projectDir string, name string) string {
	return filepath.Join(projectDir, "datasets", name)
}

// relativePath returns a relative path from base to target, or target if it can't be made relative.
func relativePath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
