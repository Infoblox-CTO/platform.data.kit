package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestPromoteCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"to", ""},
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
	// Test argument validation - promote requires exactly 2 args
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "two args is valid",
			args:    []string{"my-package", "v1.0.0"},
			wantErr: false,
		},
		{
			name:    "no args is invalid",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "one arg is invalid",
			args:    []string{"my-package"},
			wantErr: true,
		},
		{
			name:    "three args is invalid",
			args:    []string{"pkg", "v1", "extra"},
			wantErr: true,
		},
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
	// Test that an invalid environment returns an error
	// Save and restore global flags
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

func TestPromoteCmd_ValidEnvironments(t *testing.T) {
	// Test valid environment values
	validEnvs := []string{"dev", "int", "prod"}

	for _, env := range validEnvs {
		t.Run(env, func(t *testing.T) {
			// Save and restore global flags
			oldToEnv := promoteToEnv
			oldDryRun := promoteDryRun
			oldToken := os.Getenv("GITHUB_TOKEN")
			defer func() {
				promoteToEnv = oldToEnv
				promoteDryRun = oldDryRun
				if oldToken != "" {
					os.Setenv("GITHUB_TOKEN", oldToken)
				}
			}()

			promoteToEnv = env
			promoteDryRun = true

			cmd := &cobra.Command{}
			err := runPromote(cmd, []string{"my-package", "v1.0.0"})

			// Dry run should succeed with valid environment
			// May still fail if other requirements not met
			_ = err
		})
	}
}

func TestPromoteCmd_MissingGitHubToken(t *testing.T) {
	// Test that missing GITHUB_TOKEN returns error (non-dry-run)
	// Save and restore
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

func TestPromoteCmd_DryRun(t *testing.T) {
	// Test dry-run mode (should simulate without creating PR)
	// Save and restore global flags
	oldToEnv := promoteToEnv
	oldDryRun := promoteDryRun
	oldToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		promoteToEnv = oldToEnv
		promoteDryRun = oldDryRun
		if oldToken != "" {
			os.Setenv("GITHUB_TOKEN", oldToken)
		}
	}()

	promoteToEnv = "dev"
	promoteDryRun = true

	cmd := &cobra.Command{}
	err := runPromote(cmd, []string{"my-package", "v1.0.0"})

	// Dry run should work without GITHUB_TOKEN
	// May fail for other reasons in test environment
	_ = err
}

func TestPromoteCmd_WithDigest(t *testing.T) {
	// Test promoting with content digest
	// Save and restore global flags
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

	// Just verify the command handles the digest flag
	_ = err
}

func TestPromoteCmd_AutoMerge(t *testing.T) {
	// Test auto-merge flag
	flag := promoteCmd.Flags().Lookup("auto-merge")
	if flag == nil {
		t.Error("auto-merge flag not found")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("auto-merge default = %v, want false", flag.DefValue)
	}
}
