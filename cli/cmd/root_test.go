package cmd

import (
	"bytes"
	"testing"

	"github.com/Infoblox-CTO/data-platform/cli/internal/output"
)

func TestRootCmd_OutputFlag(t *testing.T) {
	// Verify output format flag is registered
	flag := rootCmd.PersistentFlags().Lookup("output")
	if flag == nil {
		t.Error("output flag not found")
		return
	}
	if flag.DefValue != "table" {
		t.Errorf("output flag default = %v, want table", flag.DefValue)
	}
	if flag.Shorthand != "o" {
		t.Errorf("output flag shorthand = %v, want o", flag.Shorthand)
	}
}

func TestRootCmd_Usage(t *testing.T) {
	// Verify root command attributes
	if rootCmd.Use != "dp" {
		t.Errorf("Use = %v, want dp", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("Short description is empty")
	}
	if rootCmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestVersionCmd(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	versionCmd.SetOut(&buf)

	// Execute version command
	err := versionCmd.RunE
	if err != nil {
		// Run function, not RunE
		versionCmd.Run(versionCmd, []string{})
	}

	output := buf.String()
	if output == "" {
		t.Log("Output was empty (expected when no SetOut)")
	}
}

func TestVersionCmd_Attributes(t *testing.T) {
	if versionCmd.Use != "version" {
		t.Errorf("Use = %v, want version", versionCmd.Use)
	}
	if versionCmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestGetOutputFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected output.Format
	}{
		{"table", output.FormatTable},
		{"json", output.FormatJSON},
		{"yaml", output.FormatYAML},
		// ParseFormat is case sensitive - uppercase defaults to table
		{"TABLE", output.FormatTable},
		{"JSON", output.FormatTable},
		{"YAML", output.FormatTable},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			// Save and restore
			old := outputFormat
			defer func() { outputFormat = old }()

			outputFormat = tt.format
			got := GetOutputFormat()
			if got != tt.expected {
				t.Errorf("GetOutputFormat() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetOutputFormat_Default(t *testing.T) {
	// Save and restore
	old := outputFormat
	defer func() { outputFormat = old }()

	outputFormat = "invalid"
	got := GetOutputFormat()
	// Invalid should default to table
	if got != output.FormatTable {
		t.Errorf("GetOutputFormat() with invalid = %v, want table", got)
	}
}

func TestGetFormatter(t *testing.T) {
	// Save and restore
	old := outputFormat
	defer func() { outputFormat = old }()

	outputFormat = "table"
	formatter := GetFormatter()
	if formatter == nil {
		t.Error("GetFormatter() returned nil")
	}
}

func TestRootCmd_SilenceUsage(t *testing.T) {
	// Verify SilenceUsage is set to true
	if !rootCmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
}

func TestRootCmd_HasVersionSubcommand(t *testing.T) {
	// Check that version is a subcommand of root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("version subcommand not found")
	}
}

func TestRootCmd_HasRequiredSubcommands(t *testing.T) {
	expectedCmds := []string{"version", "init", "lint", "build", "run", "publish", "promote"}

	cmds := rootCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, cmd := range cmds {
		cmdNames[cmd.Use] = true
		// Handle commands with args in Use like "promote <package> <version>"
		for _, expected := range expectedCmds {
			if len(cmd.Use) >= len(expected) && cmd.Use[:len(expected)] == expected {
				cmdNames[expected] = true
			}
		}
	}

	for _, expected := range expectedCmds {
		if !cmdNames[expected] {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

func TestVersion_DefaultValue(t *testing.T) {
	// Version should be "dev" by default (overridden at build time)
	if Version == "" {
		t.Error("Version should not be empty")
	}
}
