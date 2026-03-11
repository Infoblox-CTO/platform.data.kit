package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	storeCreateConnector string
	storeCreateForce     bool
)

// storeCreateCmd scaffolds a new store.
var storeCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Scaffold a new store",
	Long: `Scaffold a new store.yaml in stores/<name>/store.yaml.

A store is a named instance of a connector with connection details and
secret references.

Examples:
  # Create a PostgreSQL store
  dk store create pg-warehouse --connector postgres

  # Create an S3 store
  dk store create s3-raw --connector s3

  # Create a Kafka store
  dk store create events --connector kafka

  # Overwrite existing store
  dk store create pg-warehouse --connector postgres --force`,
	Args: cobra.ExactArgs(1),
	RunE: runStoreCreate,
}

func init() {
	storeCmd.AddCommand(storeCreateCmd)

	storeCreateCmd.Flags().StringVar(&storeCreateConnector, "connector", "",
		"Connector type (e.g., postgres, s3, kafka)")
	storeCreateCmd.Flags().BoolVar(&storeCreateForce, "force", false,
		"Overwrite existing store")
}

func runStoreCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name using the same pattern as datasets
	if !isValidPackageName(name) {
		return fmt.Errorf("invalid store name %q: must be DNS-safe (lowercase, alphanumeric, hyphens, 3-63 chars)", name)
	}

	// Determine project directory
	projectDir, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("failed to resolve project directory: %w", err)
	}

	// Create store directory and file
	storeDir := filepath.Join(projectDir, "stores", name)
	storePath := filepath.Join(storeDir, "store.yaml")

	if _, err := os.Stat(storePath); err == nil && !storeCreateForce {
		return fmt.Errorf("store %q already exists at %s (use --force to overwrite)",
			name, filepath.Join("stores", name, "store.yaml"))
	}

	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return fmt.Errorf("creating store directory: %w", err)
	}

	// Generate store YAML based on connector type
	yaml := generateStoreYAML(name, storeCreateConnector)
	if err := os.WriteFile(storePath, []byte(yaml), 0644); err != nil {
		return fmt.Errorf("writing store.yaml: %w", err)
	}

	relPath := filepath.Join("stores", name, "store.yaml")
	cmd.Printf("Created store %q at %s\n", name, relPath)
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  1. Edit %s to configure connection details\n", relPath)
	cmd.Printf("  2. Set secret references (${VAR} syntax)\n")
	cmd.Printf("  3. Run 'dk lint' to validate\n")

	return nil
}

func generateStoreYAML(name, connector string) string {
	var b strings.Builder

	b.WriteString("apiVersion: datakit.infoblox.dev/v1alpha1\n")
	b.WriteString("kind: Store\n")
	b.WriteString("metadata:\n")
	b.WriteString(fmt.Sprintf("  name: %s\n", name))
	b.WriteString("spec:\n")

	if connector != "" {
		b.WriteString(fmt.Sprintf("  connector: %s\n", connector))
	} else {
		b.WriteString("  connector: \"\"            # REQUIRED: connector type (postgres, s3, kafka, etc.)\n")
	}

	// Generate connection template based on connector type
	switch connector {
	case "postgres", "postgresql":
		b.WriteString("  connection:\n")
		b.WriteString("    host: localhost\n")
		b.WriteString("    port: 5432\n")
		b.WriteString("    database: mydb\n")
		b.WriteString("  secrets:\n")
		b.WriteString("    username: ${PG_USER}\n")
		b.WriteString("    password: ${PG_PASS}\n")

	case "s3":
		b.WriteString("  connection:\n")
		b.WriteString("    bucket: my-bucket\n")
		b.WriteString("    region: us-east-1\n")
		b.WriteString("  secrets:\n")
		b.WriteString("    accessKeyId: ${AWS_ACCESS_KEY_ID}\n")
		b.WriteString("    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}\n")

	case "kafka":
		b.WriteString("  connection:\n")
		b.WriteString("    bootstrapServers: localhost:9092\n")
		b.WriteString("  secrets:\n")
		b.WriteString("    # saslUsername: ${KAFKA_USER}\n")
		b.WriteString("    # saslPassword: ${KAFKA_PASS}\n")

	default:
		b.WriteString("  connection:\n")
		b.WriteString("    # Add connection parameters for your connector\n")
		b.WriteString("    # host: localhost\n")
		b.WriteString("  secrets:\n")
		b.WriteString("    # Add secret references using ${VAR} syntax\n")
		b.WriteString("    # username: ${DB_USER}\n")
	}

	return b.String()
}
