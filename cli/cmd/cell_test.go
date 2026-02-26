package cmd

import (
	"strings"
	"testing"
)

func TestCellCmd_HasSubcommands(t *testing.T) {
	// Verify cell command is registered with expected subcommands.
	if cellCmd == nil {
		t.Fatal("cellCmd is nil")
	}

	subcommands := map[string]bool{
		"list":   false,
		"show":   false,
		"stores": false,
	}

	for _, cmd := range cellCmd.Commands() {
		if _, ok := subcommands[cmd.Name()]; ok {
			subcommands[cmd.Name()] = true
		}
	}

	for name, found := range subcommands {
		if !found {
			t.Errorf("expected subcommand %q not found on cellCmd", name)
		}
	}
}

func TestCellCmd_Usage(t *testing.T) {
	output := cellCmd.UsageString()
	if !strings.Contains(output, "cell") {
		t.Errorf("cell usage output missing 'cell' reference: %s", output)
	}
	if !strings.Contains(output, "list") {
		t.Errorf("cell usage output missing 'list' subcommand: %s", output)
	}
	if !strings.Contains(output, "stores") {
		t.Errorf("cell usage output missing 'stores' subcommand: %s", output)
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		labels map[string]string
		empty  bool // whether the result should be empty
	}{
		{nil, true},
		{map[string]string{}, true},
		{map[string]string{"tier": "canary"}, false},
		{map[string]string{"tier": "canary", "region": "us-east-1"}, false},
	}

	for _, tt := range tests {
		result := formatLabels(tt.labels)
		if tt.empty && result != "" {
			t.Errorf("formatLabels(%v) = %q, want empty", tt.labels, result)
		}
		if !tt.empty && result == "" {
			t.Errorf("formatLabels(%v) = empty, want non-empty", tt.labels)
		}
		// Verify key=value format.
		if !tt.empty {
			for k, v := range tt.labels {
				expected := k + "=" + v
				if !strings.Contains(result, expected) {
					t.Errorf("formatLabels(%v) = %q, missing %q", tt.labels, result, expected)
				}
			}
		}
	}
}

func TestRunCmd_HasCellFlag(t *testing.T) {
	flag := runCmd.Flags().Lookup("cell")
	if flag == nil {
		t.Fatal("runCmd missing --cell flag")
	}
	if flag.DefValue != "" {
		t.Errorf("--cell default = %q, want empty", flag.DefValue)
	}
}

func TestRunCmd_HasContextFlag(t *testing.T) {
	flag := runCmd.Flags().Lookup("context")
	if flag == nil {
		t.Fatal("runCmd missing --context flag")
	}
	if flag.DefValue != "" {
		t.Errorf("--context default = %q, want empty", flag.DefValue)
	}
}
