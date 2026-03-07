package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
)

func TestDoctorCmd_Attributes(t *testing.T) {
	if doctorCmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", doctorCmd.Use, "doctor")
	}
	if doctorCmd.Short == "" {
		t.Error("Short description is empty")
	}
	if doctorCmd.Long == "" {
		t.Error("Long description is empty")
	}
}

func TestDoctorCmd_Flags(t *testing.T) {
	flag := doctorCmd.Flags().Lookup("verbose")
	if flag == nil {
		t.Fatal("flag --verbose not found")
	}
	if flag.Shorthand != "v" {
		t.Errorf("verbose shorthand = %q, want %q", flag.Shorthand, "v")
	}
	if flag.DefValue != "false" {
		t.Errorf("verbose default = %q, want %q", flag.DefValue, "false")
	}
}

func TestDoctorCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "doctor" {
			found = true
			break
		}
	}
	if !found {
		t.Error("doctor command is not registered on root command")
	}
}

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantMajor int
		wantMinor int
		wantOK    bool
	}{
		{
			name:      "standard linux",
			input:     "go version go1.25.1 linux/amd64",
			wantMajor: 1,
			wantMinor: 25,
			wantOK:    true,
		},
		{
			name:      "standard darwin",
			input:     "go version go1.25.0 darwin/arm64",
			wantMajor: 1,
			wantMinor: 25,
			wantOK:    true,
		},
		{
			name:      "older version",
			input:     "go version go1.21.6 linux/amd64",
			wantMajor: 1,
			wantMinor: 21,
			wantOK:    true,
		},
		{
			name:      "no patch version",
			input:     "go version go1.25 linux/amd64",
			wantMajor: 1,
			wantMinor: 25,
			wantOK:    true,
		},
		{
			name:      "garbage input",
			input:     "not a go version",
			wantMajor: 0,
			wantMinor: 0,
			wantOK:    false,
		},
		{
			name:      "empty input",
			input:     "",
			wantMajor: 0,
			wantMinor: 0,
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, ok := parseGoVersion(tt.input)
			if ok != tt.wantOK {
				t.Errorf("parseGoVersion(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if major != tt.wantMajor {
				t.Errorf("parseGoVersion(%q) major = %d, want %d", tt.input, major, tt.wantMajor)
			}
			if minor != tt.wantMinor {
				t.Errorf("parseGoVersion(%q) minor = %d, want %d", tt.input, minor, tt.wantMinor)
			}
		})
	}
}

func TestRegistryHost(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"plain host", "ghcr.io", "ghcr.io:443"},
		{"host with port", "registry.io:5000", "registry.io:5000"},
		{"https scheme", "https://ghcr.io", "ghcr.io:443"},
		{"http scheme", "http://localhost:5000", "localhost:5000"},
		{"host with path", "ghcr.io/myteam/images", "ghcr.io:443"},
		{"full url", "https://registry.example.com/v2/", "registry.example.com:443"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registryHost(tt.raw)
			if got != tt.want {
				t.Errorf("registryHost(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status CheckStatus
		want   string
	}{
		{CheckOK, "✓"},
		{CheckWarn, "⚠"},
		{CheckFail, "✗"},
	}

	for _, tt := range tests {
		got := statusIcon(tt.status)
		if got != tt.want {
			t.Errorf("statusIcon(%d) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestCheckGoVersion(t *testing.T) {
	// This test runs against the actual local Go installation.
	// It verifies the check produces a non-empty result.
	ctx := context.Background()
	result := checkGoVersion(ctx)

	if result.Name != "Go" {
		t.Errorf("Name = %q, want %q", result.Name, "Go")
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
	// On a CI/dev machine Go should be installed, so expect OK or at least not empty.
	if result.Status == CheckFail {
		t.Logf("Go check failed (may be expected if Go is not in PATH): %s", result.Message)
	}
}

func TestGatherChecks(t *testing.T) {
	ctx := context.Background()
	results := gatherChecks(ctx)

	expectedNames := []string{
		"Go",
		"Container Runtime",
		"k3d",
		"kubectl",
		"helm",
		"Registry Connectivity",
		"Dev Stack",
	}

	if len(results) != len(expectedNames) {
		t.Fatalf("gatherChecks returned %d results, want %d", len(results), len(expectedNames))
	}

	for i, name := range expectedNames {
		if results[i].Name != name {
			t.Errorf("results[%d].Name = %q, want %q", i, results[i].Name, name)
		}
	}
}

func TestPrintResults(t *testing.T) {
	checks := []CheckResult{
		{Name: "Go", Status: CheckOK, Message: "go1.25"},
		{Name: "Docker", Status: CheckFail, Message: "not found", Fix: "install docker"},
		{Name: "Dev Stack", Status: CheckWarn, Message: "not running", Detail: "k3d cluster off"},
	}

	// Non-verbose: no detail lines
	oldVerbose := doctorVerbose
	defer func() { doctorVerbose = oldVerbose }()
	doctorVerbose = false

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	printResults(cmd, checks)
	output := buf.String()

	if !containsSubstring(output, "✓ Go: go1.25") {
		t.Errorf("expected OK line for Go, got:\n%s", output)
	}
	if !containsSubstring(output, "✗ Docker: not found") {
		t.Errorf("expected FAIL line for Docker, got:\n%s", output)
	}
	if !containsSubstring(output, "Fix: install docker") {
		t.Errorf("expected fix suggestion for Docker, got:\n%s", output)
	}
	if !containsSubstring(output, "⚠ Dev Stack: not running") {
		t.Errorf("expected WARN line for Dev Stack, got:\n%s", output)
	}
	// Detail should NOT appear in non-verbose mode
	if containsSubstring(output, "k3d cluster off") {
		t.Errorf("detail should not appear in non-verbose mode, got:\n%s", output)
	}
}

func TestPrintResults_Verbose(t *testing.T) {
	checks := []CheckResult{
		{Name: "Dev Stack", Status: CheckWarn, Message: "not running", Detail: "k3d cluster off"},
	}

	oldVerbose := doctorVerbose
	defer func() { doctorVerbose = oldVerbose }()
	doctorVerbose = true

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	printResults(cmd, checks)
	output := buf.String()

	if !containsSubstring(output, "k3d cluster off") {
		t.Errorf("expected detail in verbose mode, got:\n%s", output)
	}
}

func TestRunDoctor_OutputContainsSections(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	// runDoctor may return error (if checks fail), that's fine
	_ = runDoctor(cmd, []string{})

	output := buf.String()
	if !containsSubstring(output, "DK Doctor") {
		t.Errorf("output should contain header, got:\n%s", output)
	}
	if !containsSubstring(output, "Result:") {
		t.Errorf("output should contain result summary, got:\n%s", output)
	}
}

func containsSubstring(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}
