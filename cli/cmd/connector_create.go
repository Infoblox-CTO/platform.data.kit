package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	connectorCreateType  string
	connectorCreateForce bool
)

// connectorCreateCmd scaffolds a new connector.
var connectorCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Scaffold a new connector",
	Long: `Scaffold a new connector.yaml in connectors/<name>/connector.yaml.

A connector defines a storage technology type, its capabilities (source,
destination), and which CloudQuery plugin images to use.

Examples:
  # Create a PostgreSQL connector
  dk connector create postgres-analytics --type postgres

  # Create an S3 connector
  dk connector create s3-datalake --type s3

  # Create a Kubernetes connector
  dk connector create k8s-cluster --type kubernetes

  # Overwrite existing connector
  dk connector create postgres-analytics --type postgres --force`,
	Args: cobra.ExactArgs(1),
	RunE: runConnectorCreate,
}

func init() {
	connectorCmd.AddCommand(connectorCreateCmd)

	connectorCreateCmd.Flags().StringVar(&connectorCreateType, "type", "",
		"Technology type (e.g., postgres, s3, kafka, kubernetes)")
	connectorCreateCmd.Flags().BoolVar(&connectorCreateForce, "force", false,
		"Overwrite existing connector")
}

func runConnectorCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name
	if !isValidPackageName(name) {
		return fmt.Errorf("invalid connector name %q: must be DNS-safe (lowercase, alphanumeric, hyphens, 3-63 chars)", name)
	}

	// Determine project directory
	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Create connector directory and file
	connDir := filepath.Join(projectDir, "connectors", name)
	connPath := filepath.Join(connDir, "connector.yaml")

	if _, err := os.Stat(connPath); err == nil && !connectorCreateForce {
		return fmt.Errorf("connector %q already exists at %s (use --force to overwrite)",
			name, filepath.Join("connectors", name, "connector.yaml"))
	}

	if err := os.MkdirAll(connDir, 0755); err != nil {
		return fmt.Errorf("creating connector directory: %w", err)
	}

	// Generate connector YAML based on type
	yaml := generateConnectorYAML(name, connectorCreateType)
	if err := os.WriteFile(connPath, []byte(yaml), 0644); err != nil {
		return fmt.Errorf("writing connector.yaml: %w", err)
	}

	relPath := filepath.Join("connectors", name, "connector.yaml")
	cmd.Printf("Created connector %q at %s\n", name, relPath)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit %s to configure capabilities and plugins\n", relPath)
	cmd.Printf("  2. Create a store that references this connector:\n")
	cmd.Printf("     dk store create <store-name> --connector %s\n", connectorCreateType)
	cmd.Printf("  3. Run 'dk lint' to validate\n")

	return nil
}

func generateConnectorYAML(name, connType string) string {
	var b strings.Builder

	b.WriteString("apiVersion: datakit.infoblox.dev/v1alpha1\n")
	b.WriteString("kind: Connector\n")
	b.WriteString("metadata:\n")
	b.WriteString(fmt.Sprintf("  name: %s\n", name))
	b.WriteString("spec:\n")

	if connType != "" {
		b.WriteString(fmt.Sprintf("  type: %s\n", connType))
	} else {
		b.WriteString("  type: \"\"                # REQUIRED: technology type (postgres, s3, kafka, etc.)\n")
	}

	// Generate type-specific templates
	switch connType {
	case "postgres", "postgresql":
		b.WriteString("  protocol: postgresql\n")
		b.WriteString("  capabilities: [source, destination]\n")
		b.WriteString("  plugin:\n")
		b.WriteString("    source: ghcr.io/cloudquery/cq-source-postgres:latest\n")
		b.WriteString("    destination: ghcr.io/cloudquery/cq-destination-postgres:latest\n")

	case "s3":
		b.WriteString("  protocol: s3\n")
		b.WriteString("  capabilities: [source, destination]\n")
		b.WriteString("  plugin:\n")
		b.WriteString("    source: ghcr.io/cloudquery/cq-source-s3:latest\n")
		b.WriteString("    destination: ghcr.io/cloudquery/cq-destination-s3:latest\n")

	case "kafka":
		b.WriteString("  protocol: kafka\n")
		b.WriteString("  capabilities: [source, destination]\n")
		b.WriteString("  plugin:\n")
		b.WriteString("    source: ghcr.io/cloudquery/cq-source-kafka:latest\n")
		b.WriteString("    destination: ghcr.io/cloudquery/cq-destination-kafka:latest\n")

	case "kubernetes", "k8s":
		b.WriteString("  protocol: kubernetes\n")
		b.WriteString("  capabilities: [source]\n")
		b.WriteString("  plugin:\n")
		b.WriteString("    source: ghcr.io/cloudquery/cq-source-k8s:latest\n")

	case "snowflake":
		b.WriteString("  protocol: snowflake\n")
		b.WriteString("  capabilities: [source, destination]\n")
		b.WriteString("  plugin:\n")
		b.WriteString("    source: ghcr.io/cloudquery/cq-source-snowflake:latest\n")
		b.WriteString("    destination: ghcr.io/cloudquery/cq-destination-snowflake:latest\n")

	default:
		b.WriteString("  protocol: \"\"             # Wire protocol (e.g., postgresql, s3, kafka)\n")
		b.WriteString("  capabilities: [source]   # source, destination, or both\n")
		b.WriteString("  plugin:\n")
		b.WriteString("    source: \"\"             # CloudQuery source plugin image\n")
		b.WriteString("    # destination: \"\"      # CloudQuery destination plugin image\n")
	}

	return b.String()
}
