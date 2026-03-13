// Package cmd contains all CLI commands for dk.
package cmd

import (
	"os"

	"github.com/Infoblox-CTO/platform.data.kit/cli/internal/output"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// outputFormat is the global output format flag
	outputFormat string

	// Version is set at build time
	Version = "dev"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dk",
	Short: "DK - DataKit CLI",
	Long: `DK (DataKit) is a Kubernetes-native data pipeline platform
enabling teams to contribute reusable, versioned "data packages" with
a complete developer workflow.

Workflow: init -> dev -> run -> lint -> test -> build -> publish -> promote

Example:
  # Create a new transform package
  dk init my-pipeline --runtime cloudquery

  # Start local development environment
  dk dev up

  # Validate manifest files
  dk lint

  # Run pipeline locally
  dk run

  # Build and publish package
  dk build
  dk publish

  # Promote to next environment
  dk promote my-pipeline v1.0.0 --to int`,
	SilenceUsage: true,
	// Show banner + description before usage when invoked with no args.
	Run: func(cmd *cobra.Command, args []string) {
		ShowBanner()
		if isTTYOut() {
			desc := lipgloss.NewStyle().Faint(true).Render(cmd.Long)
			cmd.Println(desc)
			cmd.Println()
		} else {
			cmd.Println(cmd.Long)
			cmd.Println()
		}
		cmd.Usage()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table",
		"Output format: table, json, yaml")
	rootCmd.AddCommand(versionCmd)

	// Colorize help output when running in a TTY.
	cobra.AddTemplateFunc("heading", styleHeading)
	cobra.AddTemplateFunc("cyan", styleCyan)
	cobra.AddTemplateFunc("dim", styleDim)
	cobra.AddTemplateFunc("bold", styleBold)
	rootCmd.SetUsageTemplate(colorUsageTemplate)
}

// isTTYOut reports whether stdout is an interactive terminal.
func isTTYOut() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func styleHeading(s string) string {
	if !isTTYOut() {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(s)
}

func styleCyan(s string) string {
	if !isTTYOut() {
		return s
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(s)
}

func styleDim(s string) string {
	if !isTTYOut() {
		return s
	}
	return lipgloss.NewStyle().Faint(true).Render(s)
}

func styleBold(s string) string {
	if !isTTYOut() {
		return s
	}
	return lipgloss.NewStyle().Bold(true).Render(s)
}

// colorUsageTemplate is a Cobra usage template with color functions injected.
var colorUsageTemplate = `{{ heading "Usage:" }}
  {{.UseLine}}{{if .HasAvailableSubCommands}} [command]{{end}}

{{- if gt (len .Aliases) 0}}

{{ heading "Aliases:" }}
  {{.NameAndAliases}}
{{- end}}

{{- if .HasAvailableSubCommands}}

{{ heading "Available Commands:" }}
{{- range .Commands}}
{{- if (or .IsAvailableCommand (eq .Name "help"))}}
  {{ cyan (rpad .Name .NamePadding) }}  {{ .Short }}
{{- end}}
{{- end}}
{{- end}}

{{- if .HasAvailableLocalFlags}}

{{ heading "Flags:" }}
{{ dim (.LocalFlags.FlagUsages | trimTrailingWhitespaces) }}
{{- end}}

{{- if .HasAvailableInheritedFlags}}

{{ heading "Global Flags:" }}
{{ dim (.InheritedFlags.FlagUsages | trimTrailingWhitespaces) }}
{{- end}}

{{- if .HasHelpSubCommands}}

{{ heading "Additional help topics:" }}
{{- range .Commands}}
{{- if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}
{{- end}}
{{- end}}
{{- end}}

{{- if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
{{- end}}
`

// GetOutputFormat returns the current output format.
func GetOutputFormat() output.Format {
	return output.ParseFormat(outputFormat)
}

// GetFormatter returns a formatter for the current output format.
func GetFormatter() output.Formatter {
	return output.NewFormatter(GetOutputFormat())
}

// versionCmd prints the CLI version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("dk version %s\n", Version)
	},
}
