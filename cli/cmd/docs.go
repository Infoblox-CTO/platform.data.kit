package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var docsFormat string

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate CLI reference documentation",
	Long: `Generate complete CLI reference documentation in various formats.

The "llm" format produces a structured YAML document optimised for
consumption by large language models — every command, flag, manifest
schema, and error code in a single file.

Examples:
  # Print LLM-optimised reference to stdout
  dk docs -o llm

  # Write markdown reference to a file
  dk docs -o md > docs/cli-reference.md

  # Default plain-text output
  dk docs`,
	RunE: runDocs,
}

func init() {
	rootCmd.AddCommand(docsCmd)
	docsCmd.Flags().StringVarP(&docsFormat, "output", "o", "text", "Output format: text, md, llm")
}

// ---------------------------------------------------------------------------
// Data model
// ---------------------------------------------------------------------------

// CLIReference is the top-level structure for the full CLI reference.
type CLIReference struct {
	Version  string           `yaml:"version"`
	Workflow string           `yaml:"workflow"`
	Commands []CommandDoc     `yaml:"commands"`
	Schemas  map[string]any   `yaml:"schemas"`
	Errors   []ErrorDoc       `yaml:"errors"`
	Enums    map[string][]string `yaml:"enums"`
}

// CommandDoc describes a single CLI command.
type CommandDoc struct {
	Path    string    `yaml:"path"`
	Use     string    `yaml:"use"`
	Short   string    `yaml:"short"`
	Long    string    `yaml:"long,omitempty"`
	Example string    `yaml:"example,omitempty"`
	Flags   []FlagDoc `yaml:"flags,omitempty"`
}

// FlagDoc describes a single command flag.
type FlagDoc struct {
	Name      string `yaml:"name"`
	Shorthand string `yaml:"shorthand,omitempty"`
	Type      string `yaml:"type"`
	Default   string `yaml:"default,omitempty"`
	Usage     string `yaml:"usage"`
}

// ErrorDoc describes a validation error code.
type ErrorDoc struct {
	Code    string `yaml:"code"`
	Message string `yaml:"message"`
}

// ---------------------------------------------------------------------------
// Tree walker
// ---------------------------------------------------------------------------

func walkCommands(cmd *cobra.Command) []CommandDoc {
	var docs []CommandDoc
	if cmd.Hidden {
		return nil
	}

	doc := CommandDoc{
		Path:    cmd.CommandPath(),
		Use:     cmd.Use,
		Short:   cmd.Short,
		Long:    cmd.Long,
		Example: cmd.Example,
	}

	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		doc.Flags = append(doc.Flags, FlagDoc{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Type:      f.Value.Type(),
			Default:   f.DefValue,
			Usage:     f.Usage,
		})
	})

	// Include inherited (persistent) flags only on the root command.
	if !cmd.HasParent() {
		cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			doc.Flags = append(doc.Flags, FlagDoc{
				Name:      f.Name,
				Shorthand: f.Shorthand,
				Type:      f.Value.Type(),
				Default:   f.DefValue,
				Usage:     f.Usage,
			})
		})
	}

	docs = append(docs, doc)

	for _, sub := range cmd.Commands() {
		docs = append(docs, walkCommands(sub)...)
	}
	return docs
}

// ---------------------------------------------------------------------------
// Schema & enum builders
// ---------------------------------------------------------------------------

func buildSchemas() map[string]any {
	return map[string]any{
		"Transform": map[string]any{
			"apiVersion": "datakit.infoblox.dev/v1alpha1",
			"kind":       "Transform",
			"metadata": map[string]any{
				"name":        "(string, required) DNS-safe transform name",
				"namespace":   "(string) team namespace",
				"version":     "(string) semantic version e.g. 0.1.0",
				"labels":      "(map[string]string) key-value labels",
				"annotations": "(map[string]string) arbitrary annotations",
			},
			"spec": map[string]any{
				"runtime":            "(string, required) cloudquery | generic-go | generic-python | dbt",
				"mode":               "(string) batch | streaming (default: batch)",
				"inputs":             "([]DataSetRef, required) input dataset references",
				"outputs":            "([]DataSetRef, required) output dataset references",
				"image":              "(string) container image (required for generic-go, generic-python, dbt)",
				"command":            "([]string) override container entrypoint",
				"env":                "([]EnvVar) environment variables",
				"trigger":            "(TriggerSpec) when this transform executes",
				"timeout":            "(string) max execution duration e.g. 30m, 1h",
				"resources":          "(ResourceSpec) cpu, memory, ephemeralStorage",
				"replicas":           "(int) parallel workers (streaming only)",
				"lineage":            "(LineageSpec) lineage event emission config",
				"serviceAccountName": "(string) k8s ServiceAccount for the job pod",
			},
		},
		"DataSet": map[string]any{
			"apiVersion": "datakit.infoblox.dev/v1alpha1",
			"kind":       "DataSet",
			"metadata": map[string]any{
				"name":      "(string, required) DNS-safe dataset name",
				"namespace": "(string) team namespace",
				"version":   "(string) semantic version of the data contract",
				"labels":    "(map[string]string) key-value labels",
			},
			"spec": map[string]any{
				"store":          "(string, required) name of the Store where this data lives",
				"table":          "(string) fully-qualified table name (relational stores)",
				"prefix":         "(string) object prefix (object stores like S3)",
				"topic":          "(string) topic name (streaming stores like Kafka)",
				"format":         "(string) data format: parquet | json | csv | avro",
				"classification": "(string) data classification: public | internal | confidential | restricted",
				"schema":         "([]SchemaField) inline field definitions (name, type, pii, from)",
				"schemaRef":      "(string) APX module reference e.g. users@^1.0.0 (mutually exclusive with schema)",
				"dev":            "(DataSetDevSpec) development-only config: seed data, profiles",
			},
		},
		"Store": map[string]any{
			"apiVersion": "datakit.infoblox.dev/v1alpha1",
			"kind":       "Store",
			"metadata": map[string]any{
				"name":      "(string, required) logical store name",
				"namespace": "(string) team namespace",
				"labels":    "(map[string]string) key-value labels",
			},
			"spec": map[string]any{
				"connector":        "(string, required) provider name e.g. postgres, s3, kafka",
				"connectorVersion": "(string) semver range constraining connector version",
				"connection":       "(map[string]any, required) technology-specific connection parameters",
				"secrets":          "(map[string]string) credential references using ${VAR} interpolation",
			},
		},
		"Connector": map[string]any{
			"apiVersion": "datakit.infoblox.dev/v1alpha1",
			"kind":       "Connector",
			"metadata": map[string]any{
				"name":   "(string, required) unique CR instance name",
				"labels": "(map[string]string) key-value labels",
			},
			"spec": map[string]any{
				"type":             "(string, required) technology identifier e.g. postgres, s3, kafka",
				"provider":         "(string) logical connector identity (defaults to type)",
				"version":          "(string) semantic version of this connector release",
				"protocol":         "(string) wire protocol e.g. postgresql, s3, kafka",
				"capabilities":     "([]string, required) source | destination (or both)",
				"plugin":           "(ConnectorPlugin) CloudQuery plugin image references (source, destination)",
				"tools":            "([]ConnectorTool) technology-specific actions",
				"connectionSchema": "(map) declares connection fields the connector expects",
			},
		},
		"DataSetGroup": map[string]any{
			"apiVersion": "datakit.infoblox.dev/v1alpha1",
			"kind":       "DataSetGroup",
			"metadata": map[string]any{
				"name":      "(string, required) group name",
				"namespace": "(string) team namespace",
				"labels":    "(map[string]string) key-value labels",
			},
			"spec": map[string]any{
				"store":    "(string, required) common Store for all datasets",
				"datasets": "([]string, required) list of DataSet names in this group",
			},
		},
		"DataSetRef": map[string]any{
			"dataset": "(string) exact DataSet name (mutually exclusive with tags)",
			"tags":    "(map[string]string) label selector (mutually exclusive with dataset)",
			"version": "(string) semver range constraint e.g. >=1.0.0 <2.0.0",
			"cell":    "(string) resolve Store from a specific cell's namespace",
			"schema":  "(string) APX module ID for consumer-side schema validation",
		},
		"TriggerSpec": map[string]any{
			"policy":   "(string, required) schedule | on-change | manual | composite",
			"schedule": "(ScheduleSpec) cron, timezone, suspend",
			"policies": "([]string) sub-policies for composite triggers",
		},
	}
}

func buildErrors() []ErrorDoc {
	// Use the authoritative error messages from contracts.
	codes := map[string]string{
		"E001": "name must be DNS-safe (lowercase, alphanumeric, hyphens)",
		"E002": "kind must be one of: Connector, Store, DataSet, DataSetGroup, Transform",
		"E003": "outputs are required for Transform kind packages",
		"E004": "classification is required for output artifacts",
		"E005": "schema type must be one of: parquet, avro, json, csv",
		"E020": "version must be a valid SemVer string",
		"E021": "version already exists and cannot be overwritten",
		"E030": "image must be a valid container image reference",
		"E031": "timeout must be a positive duration",
		"E040": "spec.runtime is required",
		"E041": "spec.image is required for generic-* runtimes",
		"E070": "required field missing (dataset validation)",
		"E076": "dataset reference not found",
		"E200": "spec.type is required for Connector",
		"E201": "spec.capabilities must list at least one capability (source, destination)",
		"E210": "spec.connector is required for Store",
		"E211": "spec.connection must contain at least one connection parameter",
		"E212": "spec.secrets values must use ${VAR} interpolation syntax",
		"E220": "spec.store is required for DataSet",
		"E221": "at least one of spec.table, spec.prefix, or spec.topic is required",
		"E222": "spec.schema contains invalid field definitions (name and type are required)",
		"E230": "spec.inputs must contain at least one dataset reference",
		"E231": "spec.outputs must contain at least one dataset reference",
		"E232": "spec.image is required for generic-go, generic-python, and dbt runtimes",
		"E240": "spec.store is required for DataSetGroup",
		"E241": "spec.datasets must contain at least one dataset name",
		"E310": "spec.schemaRef and spec.schema are mutually exclusive",
		"E311": "spec.schemaRef must be in format module@constraint",
		"E312": "dk.lock is missing an entry for this schemaRef — run dk lock",
		"E313": "dk.lock checksum does not match resolved schema — run dk lock --upgrade",
		"E314": "breaking schema change detected between locked and current version",
		"W209": "trigger is recommended for batch mode transforms",
	}

	var docs []ErrorDoc
	for code, msg := range codes {
		docs = append(docs, ErrorDoc{Code: code, Message: msg})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Code < docs[j].Code })
	return docs
}

func buildEnums() map[string][]string {
	// Pull valid values from the contracts package where possible.
	var kinds []string
	for _, k := range contracts.AllKinds() {
		kinds = append(kinds, string(k))
	}

	return map[string][]string{
		"kind":            kinds,
		"runtime":         {"cloudquery", "generic-go", "generic-python", "dbt"},
		"mode":            {"batch", "streaming"},
		"classification":  {"public", "internal", "confidential", "restricted"},
		"triggerPolicy":   {"schedule", "on-change", "manual", "composite"},
		"dataFormat":      {"parquet", "json", "csv", "avro"},
		"outputFormat":    {"table", "json", "yaml"},
		"schemaFieldType": {"integer", "string", "timestamp", "boolean", "float"},
	}
}

// ---------------------------------------------------------------------------
// Reference builder
// ---------------------------------------------------------------------------

func buildReference() *CLIReference {
	return &CLIReference{
		Version:  Version,
		Workflow: "init -> dev up -> run -> lint -> test -> build -> publish -> promote",
		Commands: walkCommands(rootCmd),
		Schemas:  buildSchemas(),
		Errors:   buildErrors(),
		Enums:    buildEnums(),
	}
}

// ---------------------------------------------------------------------------
// Output renderers
// ---------------------------------------------------------------------------

func runDocs(cmd *cobra.Command, _ []string) error {
	ref := buildReference()

	switch docsFormat {
	case "llm":
		return renderLLM(os.Stdout, ref)
	case "md":
		return renderMarkdown(os.Stdout, ref)
	default:
		return renderText(os.Stdout, ref)
	}
}

// renderLLM emits structured YAML — optimised for machine consumption.
func renderLLM(w io.Writer, ref *CLIReference) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(ref)
}

// renderMarkdown emits a full Markdown reference document.
func renderMarkdown(w io.Writer, ref *CLIReference) error {
	fmt.Fprintf(w, "# dk CLI Reference\n\n")
	fmt.Fprintf(w, "**Version:** %s\n\n", ref.Version)
	fmt.Fprintf(w, "**Workflow:** `%s`\n\n", ref.Workflow)

	// Commands
	fmt.Fprintf(w, "## Commands\n\n")
	for _, c := range ref.Commands {
		fmt.Fprintf(w, "### `%s`\n\n", c.Path)
		fmt.Fprintf(w, "%s\n\n", c.Short)
		if c.Long != "" {
			fmt.Fprintf(w, "%s\n\n", c.Long)
		}
		if c.Example != "" {
			fmt.Fprintf(w, "**Examples:**\n\n```\n%s\n```\n\n", c.Example)
		}
		if len(c.Flags) > 0 {
			fmt.Fprintf(w, "| Flag | Short | Type | Default | Description |\n")
			fmt.Fprintf(w, "|------|-------|------|---------|-------------|\n")
			for _, f := range c.Flags {
				def := f.Default
				if def == "" {
					def = "-"
				}
				sh := f.Shorthand
				if sh == "" {
					sh = "-"
				}
				fmt.Fprintf(w, "| `--%s` | `%s` | %s | %s | %s |\n", f.Name, sh, f.Type, def, f.Usage)
			}
			fmt.Fprintln(w)
		}
	}

	// Schemas
	fmt.Fprintf(w, "## Manifest Schemas\n\n")
	// Sort schema keys for deterministic output.
	schemaKeys := make([]string, 0, len(ref.Schemas))
	for k := range ref.Schemas {
		schemaKeys = append(schemaKeys, k)
	}
	sort.Strings(schemaKeys)
	for _, name := range schemaKeys {
		schema := ref.Schemas[name]
		fmt.Fprintf(w, "### %s\n\n```yaml\n", name)
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		enc.Encode(schema)
		fmt.Fprintf(w, "```\n\n")
	}

	// Errors
	fmt.Fprintf(w, "## Validation Error Codes\n\n")
	fmt.Fprintf(w, "| Code | Message |\n|------|--------|\n")
	for _, e := range ref.Errors {
		fmt.Fprintf(w, "| %s | %s |\n", e.Code, e.Message)
	}
	fmt.Fprintln(w)

	// Enums
	fmt.Fprintf(w, "## Enum Values\n\n")
	enumKeys := make([]string, 0, len(ref.Enums))
	for k := range ref.Enums {
		enumKeys = append(enumKeys, k)
	}
	sort.Strings(enumKeys)
	for _, name := range enumKeys {
		fmt.Fprintf(w, "- **%s**: %s\n", name, strings.Join(ref.Enums[name], ", "))
	}
	fmt.Fprintln(w)

	return nil
}

// renderText emits a human-readable plain-text reference.
func renderText(w io.Writer, ref *CLIReference) error {
	fmt.Fprintf(w, "dk CLI Reference (version %s)\n", ref.Version)
	fmt.Fprintf(w, "Workflow: %s\n\n", ref.Workflow)

	fmt.Fprintln(w, "COMMANDS")
	fmt.Fprintln(w, strings.Repeat("=", 60))
	for _, c := range ref.Commands {
		fmt.Fprintf(w, "\n  %s\n", c.Path)
		fmt.Fprintf(w, "    %s\n", c.Short)
		if len(c.Flags) > 0 {
			fmt.Fprintln(w, "    Flags:")
			for _, f := range c.Flags {
				sh := ""
				if f.Shorthand != "" {
					sh = fmt.Sprintf("-%s, ", f.Shorthand)
				}
				def := ""
				if f.Default != "" {
					def = fmt.Sprintf(" (default: %s)", f.Default)
				}
				fmt.Fprintf(w, "      %s--%s <%s>%s  %s\n", sh, f.Name, f.Type, def, f.Usage)
			}
		}
	}

	fmt.Fprintf(w, "\nMANIFEST KINDS\n")
	fmt.Fprintln(w, strings.Repeat("=", 60))
	kindKeys := make([]string, 0, len(ref.Schemas))
	for k := range ref.Schemas {
		kindKeys = append(kindKeys, k)
	}
	sort.Strings(kindKeys)
	for _, name := range kindKeys {
		fmt.Fprintf(w, "\n  %s\n", name)
	}

	fmt.Fprintf(w, "\nVALIDATION ERRORS\n")
	fmt.Fprintln(w, strings.Repeat("=", 60))
	for _, e := range ref.Errors {
		fmt.Fprintf(w, "  %s  %s\n", e.Code, e.Message)
	}

	fmt.Fprintf(w, "\nENUM VALUES\n")
	fmt.Fprintln(w, strings.Repeat("=", 60))
	enumKeys := make([]string, 0, len(ref.Enums))
	for k := range ref.Enums {
		enumKeys = append(enumKeys, k)
	}
	sort.Strings(enumKeys)
	for _, name := range enumKeys {
		fmt.Fprintf(w, "  %-18s %s\n", name+":", strings.Join(ref.Enums[name], ", "))
	}
	fmt.Fprintln(w)

	return nil
}
