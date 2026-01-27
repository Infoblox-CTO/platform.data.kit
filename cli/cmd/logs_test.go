package cmd

import (
	"testing"
)

func TestLogsCmd_Flags(t *testing.T) {
	// Verify flags are registered correctly
	tests := []struct {
		flag     string
		defValue string
	}{
		{"follow", "false"},
		{"run", ""},
		{"environment", "dev"},
		{"tail", "100"},
		{"since", ""},
		{"timestamps", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := logsCmd.Flags().Lookup(tt.flag)
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

func TestLogsCmd_Args(t *testing.T) {
	// Test argument validation
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "one arg is valid",
			args:    []string{"my-package"},
			wantErr: false,
		},
		{
			name:    "no args is invalid",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "two args is invalid",
			args:    []string{"pkg1", "pkg2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := logsCmd.Args(logsCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogsCmd_InvalidEnvironment(t *testing.T) {
	// Test that invalid environment is rejected
	oldEnv := logsEnvironment
	defer func() {
		logsEnvironment = oldEnv
	}()

	logsEnvironment = "invalid-env"

	err := runLogs(logsCmd, []string{"test-package"})
	if err == nil {
		t.Error("runLogs() expected error for invalid environment, got nil")
	}
}

func TestLogsCmd_ValidEnvironments(t *testing.T) {
	// Test that valid environments are accepted
	environments := []string{"dev", "int", "prod"}

	for _, env := range environments {
		t.Run(env, func(t *testing.T) {
			oldEnv := logsEnvironment
			defer func() {
				logsEnvironment = oldEnv
			}()

			logsEnvironment = env
			// This will fail because no container exists, but should not fail on environment validation
			err := runLogs(logsCmd, []string{"nonexistent-package"})
			if err != nil && err.Error() == "invalid environment: "+env+" (must be dev, int, or prod)" {
				t.Errorf("runLogs() rejected valid environment: %s", env)
			}
		})
	}
}

func TestLogsCmd_FlagShortcuts(t *testing.T) {
	// Verify shortcut flags exist
	shortcuts := []struct {
		short    string
		longFlag string
	}{
		{"f", "follow"},
		{"e", "environment"},
		{"t", "timestamps"},
	}

	for _, s := range shortcuts {
		t.Run(s.short, func(t *testing.T) {
			flag := logsCmd.Flags().ShorthandLookup(s.short)
			if flag == nil {
				t.Errorf("shortcut -%s not found", s.short)
				return
			}
			if flag.Name != s.longFlag {
				t.Errorf("shortcut -%s maps to %s, want %s", s.short, flag.Name, s.longFlag)
			}
		})
	}
}
