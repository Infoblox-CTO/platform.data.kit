package dataset

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// dnsNamePattern validates DNS-safe dataset names.
var dnsNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,62}$`)

// ScaffoldOpts contains options for scaffolding a new dataset.
type ScaffoldOpts struct {
	// Name is the dataset name (DNS-safe).
	Name string

	// ProjectDir is the root directory of the data package project.
	ProjectDir string

	// Force overwrites an existing dataset if true.
	Force bool

	// Store is the store name to pre-fill in the spec.
	Store string
}

// ScaffoldResult contains the result of scaffolding a dataset.
type ScaffoldResult struct {
	// DataSetPath is the path to the created dataset.yaml file.
	DataSetPath string

	// DataSetDir is the directory containing the dataset.yaml file.
	DataSetDir string

	// DataSet is the scaffolded dataset manifest.
	DataSet *contracts.DataSetManifest
}

// Scaffold creates a new dataset.yaml in datasets/<name>/dataset.yaml.
func Scaffold(opts ScaffoldOpts) (*ScaffoldResult, error) {
	// Validate name
	if !dnsNamePattern.MatchString(opts.Name) {
		return nil, fmt.Errorf("invalid dataset name %q: must match %s (DNS-safe, lowercase, 3-63 chars)",
			opts.Name, dnsNamePattern.String())
	}

	// Determine dataset directory (flat layout: datasets/<name>/)
	datasetDir := filepath.Join(opts.ProjectDir, "datasets", opts.Name)
	datasetPath := filepath.Join(datasetDir, "dataset.yaml")

	if _, err := os.Stat(datasetPath); err == nil && !opts.Force {
		return nil, fmt.Errorf("dataset %q already exists at %s (use --force to overwrite)", opts.Name, datasetPath)
	}

	// Also check for name uniqueness across all datasets
	if !opts.Force {
		existing, _ := LoadAllDataSets(opts.ProjectDir)
		for _, ds := range existing {
			if ds.Metadata.Name == opts.Name {
				return nil, fmt.Errorf("dataset with name %q already exists (store: %s)", opts.Name, ds.Spec.Store)
			}
		}
	}

	// Build the dataset manifest
	store := opts.Store
	ds := &contracts.DataSetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "DataSet",
		Metadata: contracts.DataSetMetadata{
			Name: opts.Name,
		},
		Spec: contracts.DataSetSpec{
			Store: store,
		},
	}

	// Create directory and write dataset.yaml
	if err := os.MkdirAll(datasetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create dataset directory: %w", err)
	}

	yamlData, err := marshalDataSetWithComments(ds)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dataset: %w", err)
	}

	if err := os.WriteFile(datasetPath, yamlData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write dataset.yaml: %w", err)
	}

	return &ScaffoldResult{
		DataSetPath: datasetPath,
		DataSetDir:  datasetDir,
		DataSet:     ds,
	}, nil
}

// marshalDataSetWithComments creates a YAML representation with helpful inline comments.
func marshalDataSetWithComments(ds *contracts.DataSetManifest) ([]byte, error) {
	var b strings.Builder

	b.WriteString("apiVersion: datakit.infoblox.dev/v1alpha1\n")
	b.WriteString("kind: DataSet\n")

	// Metadata section
	b.WriteString("metadata:\n")
	b.WriteString(fmt.Sprintf("  name: %s\n", ds.Metadata.Name))
	if ds.Metadata.Namespace != "" {
		b.WriteString(fmt.Sprintf("  namespace: %s\n", ds.Metadata.Namespace))
	}
	if len(ds.Metadata.Labels) > 0 {
		b.WriteString("  labels:\n")
		keys := make([]string, 0, len(ds.Metadata.Labels))
		for k := range ds.Metadata.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(fmt.Sprintf("    %s: %s\n", k, ds.Metadata.Labels[k]))
		}
	}

	// Spec section
	b.WriteString("spec:\n")
	if ds.Spec.Store != "" {
		b.WriteString(fmt.Sprintf("  store: %s\n", ds.Spec.Store))
	} else {
		b.WriteString("  store: \"\"              # REQUIRED: set your store name\n")
	}
	if ds.Spec.Table != "" {
		b.WriteString(fmt.Sprintf("  table: %s\n", ds.Spec.Table))
	}
	if ds.Spec.Prefix != "" {
		b.WriteString(fmt.Sprintf("  prefix: %s\n", ds.Spec.Prefix))
	}
	if ds.Spec.Topic != "" {
		b.WriteString(fmt.Sprintf("  topic: %s\n", ds.Spec.Topic))
	}
	if ds.Spec.Format != "" {
		b.WriteString(fmt.Sprintf("  format: %s\n", ds.Spec.Format))
	}
	if ds.Spec.Classification != "" {
		b.WriteString(fmt.Sprintf("  classification: %s\n", ds.Spec.Classification))
	}
	if len(ds.Spec.Schema) > 0 {
		b.WriteString("  schema:\n")
		for _, field := range ds.Spec.Schema {
			b.WriteString(fmt.Sprintf("    - name: %s\n", field.Name))
			b.WriteString(fmt.Sprintf("      type: %s\n", field.Type))
			if field.PII {
				b.WriteString("      pii: true\n")
			}
			if field.From != "" {
				b.WriteString(fmt.Sprintf("      from: %s\n", field.From))
			}
		}
	}

	return []byte(b.String()), nil
}

// ValidateDataSetName checks if a name is a valid DNS-safe dataset name.
func ValidateDataSetName(name string) error {
	if !dnsNamePattern.MatchString(name) {
		return fmt.Errorf("invalid dataset name %q: must be lowercase, start with a letter, 3-63 characters, and contain only letters, digits, and hyphens",
			name)
	}
	return nil
}
