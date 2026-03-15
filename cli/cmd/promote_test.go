package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestPromoteCmd_Flags(t *testing.T) {
	tests := []struct {
		flag     string
		defValue string
	}{
		{"to", ""},
		{"cell", ""},
		{"digest", ""},
		{"registry", ""},
		{"dry-run", "false"},
		{"auto-merge", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := promoteCmd.Flags().Lookup(tt.flag)
			if flag == nil {
				t.Errorf("flag --%s not found", tt.flag)
				return
			}
			if flag.DefValue != tt.defValue {
				t.Errorf("flag --%s default = %v, want %v", tt.flag, flag.DefValue, tt.defValue)
			}
		})
	}
}

func TestPromoteCmd_Args(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"two args is valid", []string{"my-package", "v1.0.0"}, false},
		{"no args is invalid", []string{}, true},
		{"one arg is invalid", []string{"my-package"}, true},
		{"three args is invalid", []string{"pkg", "v1", "extra"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := promoteCmd.Args(promoteCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPromoteCmd_InvalidEnvironment(t *testing.T) {
	oldToEnv := promoteToEnv
	oldDryRun := promoteDryRun
	defer func() {
		promoteToEnv = oldToEnv
		promoteDryRun = oldDryRun
	}()

	promoteToEnv = "invalid-env"
	promoteDryRun = true

	cmd := &cobra.Command{}
	err := runPromote(cmd, []string{"my-package", "v1.0.0"})
	if err == nil {
		t.Error("expected error for invalid environment")
	}
}

func TestPromoteCmd_DryRunDefaultCell(t *testing.T) {
	oldToEnv := promoteToEnv
	oldCell := promoteCell
	oldDryRun := promoteDryRun
	defer func() {
		promoteToEnv = oldToEnv
		promoteCell = oldCell
		promoteDryRun = oldDryRun
	}()

	promoteToEnv = "dev"
	promoteCell = "" // defaults to c0
	promoteDryRun = true

	cmd := &cobra.Command{}
	err := runPromote(cmd, []string{"my-package", "v1.0.0"})
	if err != nil {
		t.Errorf("dry-run with default cell should succeed, got error: %v", err)
	}
}

func TestPromoteCmd_DryRunNamedCell(t *testing.T) {
	oldToEnv := promoteToEnv
	oldCell := promoteCell
	oldDryRun := promoteDryRun
	defer func() {
		promoteToEnv = oldToEnv
		promoteCell = oldCell
		promoteDryRun = oldDryRun
	}()

	promoteToEnv = "prod"
	promoteCell = "canary"
	promoteDryRun = true

	cmd := &cobra.Command{}
	err := runPromote(cmd, []string{"my-package", "v1.0.0"})
	if err != nil {
		t.Errorf("dry-run with named cell should succeed, got error: %v", err)
	}
}

func TestPromoteCmd_MissingGitHubToken(t *testing.T) {
	oldToEnv := promoteToEnv
	oldDryRun := promoteDryRun
	oldToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		promoteToEnv = oldToEnv
		promoteDryRun = oldDryRun
		if oldToken != "" {
			os.Setenv("GITHUB_TOKEN", oldToken)
		} else {
			os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	os.Unsetenv("GITHUB_TOKEN")
	promoteToEnv = "dev"
	promoteDryRun = false

	cmd := &cobra.Command{}
	err := runPromote(cmd, []string{"my-package", "v1.0.0"})
	if err == nil {
		t.Error("expected error when GITHUB_TOKEN is missing")
	}
}

func TestPromoteCmd_WithDigest(t *testing.T) {
	oldToEnv := promoteToEnv
	oldDryRun := promoteDryRun
	oldDigest := promoteDigest
	defer func() {
		promoteToEnv = oldToEnv
		promoteDryRun = oldDryRun
		promoteDigest = oldDigest
	}()

	promoteToEnv = "dev"
	promoteDryRun = true
	promoteDigest = "sha256:abc123def456"

	cmd := &cobra.Command{}
	err := runPromote(cmd, []string{"my-package", "v1.0.0"})
	if err != nil {
		t.Errorf("dry-run with --digest should succeed, got error: %v", err)
	}
}

func TestPromoteCmd_AutoMerge(t *testing.T) {
	flag := promoteCmd.Flags().Lookup("auto-merge")
	if flag == nil {
		t.Error("auto-merge flag not found")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("auto-merge default = %v, want false", flag.DefValue)
	}
}

func TestPromoteCmd_CellFlag(t *testing.T) {
	flag := promoteCmd.Flags().Lookup("cell")
	if flag == nil {
		t.Error("cell flag not found")
		return
	}
	if flag.DefValue != "" {
		t.Errorf("cell default = %v, want empty", flag.DefValue)
	}
}
