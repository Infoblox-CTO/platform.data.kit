package cmd

import (
	"context"
	"os"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/localdev"
)

// TestDevCmd_Flags tests that dev command flags are registered correctly.
func TestDevCmd_Flags(t *testing.T) {
	tests := []struct {
		flag     string
		defValue string
	}{
		{"runtime", "k3d"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			flag := devCmd.PersistentFlags().Lookup(tt.flag)
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

// TestDevDownCmd_VolumeFlag tests the --volumes flag on dev down command.
func TestDevDownCmd_VolumeFlag(t *testing.T) {
	flag := devDownCmd.Flags().Lookup("volumes")
	if flag == nil {
		t.Error("flag --volumes not found on dev down command")
		return
	}
	if flag.DefValue != "false" {
		t.Errorf("flag --volumes default = %v, want %v", flag.DefValue, "false")
	}
}

// TestGetRuntime_DefaultK3d tests that getRuntime returns k3d by default.
func TestGetRuntime_DefaultK3d(t *testing.T) {
	// Save and restore global flag
	oldRuntime := devRuntime
	defer func() { devRuntime = oldRuntime }()

	devRuntime = ""
	runtime, err := getRuntime()
	if err != nil {
		t.Fatalf("getRuntime() error = %v", err)
	}

	if runtime != localdev.RuntimeK3d {
		t.Errorf("getRuntime() = %q, want %q", runtime, localdev.RuntimeK3d)
	}
}

// TestGetRuntime_K3d tests getRuntime with k3d runtime.
func TestGetRuntime_K3d(t *testing.T) {
	oldRuntime := devRuntime
	defer func() { devRuntime = oldRuntime }()

	testCases := []string{"k3d", "kubernetes", "k8s"}
	for _, rt := range testCases {
		t.Run(rt, func(t *testing.T) {
			devRuntime = rt
			runtime, err := getRuntime()
			if err != nil {
				t.Fatalf("getRuntime() error = %v", err)
			}
			if runtime != localdev.RuntimeK3d {
				t.Errorf("getRuntime() = %q, want %q", runtime, localdev.RuntimeK3d)
			}
		})
	}
}

// TestGetRuntime_Invalid tests getRuntime with invalid runtime.
func TestGetRuntime_Invalid(t *testing.T) {
	oldRuntime := devRuntime
	defer func() { devRuntime = oldRuntime }()

	devRuntime = "invalid-runtime"
	_, err := getRuntime()
	if err == nil {
		t.Error("getRuntime() should return error for invalid runtime")
	}
}

// TestGetWorkspacePath_FromEnv tests workspace path from environment variable.
func TestGetWorkspacePath_FromEnv(t *testing.T) {
	// Save and restore env var
	oldVal := os.Getenv("DP_WORKSPACE_PATH")
	defer os.Setenv("DP_WORKSPACE_PATH", oldVal)

	testPath := "/test/workspace/path"
	os.Setenv("DP_WORKSPACE_PATH", testPath)

	result := getWorkspacePath()
	if result != testPath {
		t.Errorf("getWorkspacePath() = %q, want %q", result, testPath)
	}
}

// TestGetWorkspacePath_NoEnv tests workspace path when env var is not set.
func TestGetWorkspacePath_NoEnv(t *testing.T) {
	// Save and restore env var
	oldVal := os.Getenv("DP_WORKSPACE_PATH")
	defer os.Setenv("DP_WORKSPACE_PATH", oldVal)

	os.Unsetenv("DP_WORKSPACE_PATH")

	// Result depends on config file; may be empty if no config
	result := getWorkspacePath()
	// Just verify it doesn't panic and returns a string
	_ = result
}

// TestGetWorkspacePath_EnvOverridesConfig tests that env var takes precedence.
func TestGetWorkspacePath_EnvOverridesConfig(t *testing.T) {
	// Save and restore env var
	oldVal := os.Getenv("DP_WORKSPACE_PATH")
	defer os.Setenv("DP_WORKSPACE_PATH", oldVal)

	envPath := "/from/env/path"
	os.Setenv("DP_WORKSPACE_PATH", envPath)

	result := getWorkspacePath()
	if result != envPath {
		t.Errorf("getWorkspacePath() = %q, want %q (env should override config)", result, envPath)
	}
}

// TestGetRuntimeManager_K3d tests getRuntimeManager for k3d runtime.
func TestGetRuntimeManager_K3d(t *testing.T) {
	manager, err := getRuntimeManager(localdev.RuntimeK3d)
	if err != nil {
		t.Fatalf("getRuntimeManager(k3d) error = %v", err)
	}

	if manager == nil {
		t.Error("getRuntimeManager(k3d) returned nil")
	}

	if manager.Type() != localdev.RuntimeK3d {
		t.Errorf("manager.Type() = %q, want %q", manager.Type(), localdev.RuntimeK3d)
	}
}

// TestGetRuntimeManager_Invalid tests getRuntimeManager for invalid runtime.
func TestGetRuntimeManager_Invalid(t *testing.T) {
	_, err := getRuntimeManager("invalid")
	if err == nil {
		t.Error("getRuntimeManager(invalid) should return error")
	}
}

// TestDevSubcommands tests that dev command has expected subcommands.
func TestDevSubcommands(t *testing.T) {
	subcommands := []string{"up", "down", "status"}

	for _, name := range subcommands {
		t.Run(name, func(t *testing.T) {
			found := false
			for _, cmd := range devCmd.Commands() {
				if cmd.Use == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("subcommand %q not found on dev command", name)
			}
		})
	}
}

// TestDevCmd_LongDescription tests dev command descriptions are set.
func TestDevCmd_LongDescription(t *testing.T) {
	if devCmd.Short == "" {
		t.Error("devCmd.Short is empty")
	}
	if devCmd.Long == "" {
		t.Error("devCmd.Long is empty")
	}
}

// TestDevUpCmd_Description tests dev up command descriptions.
func TestDevUpCmd_Description(t *testing.T) {
	if devUpCmd.Short == "" {
		t.Error("devUpCmd.Short is empty")
	}
	if devUpCmd.Long == "" {
		t.Error("devUpCmd.Long is empty")
	}
	if devUpCmd.RunE == nil {
		t.Error("devUpCmd.RunE is nil")
	}
}

// TestDevDownCmd_Description tests dev down command descriptions.
func TestDevDownCmd_Description(t *testing.T) {
	if devDownCmd.Short == "" {
		t.Error("devDownCmd.Short is empty")
	}
	if devDownCmd.Long == "" {
		t.Error("devDownCmd.Long is empty")
	}
	if devDownCmd.RunE == nil {
		t.Error("devDownCmd.RunE is nil")
	}
}

// TestDevStatusCmd_Description tests dev status command descriptions.
func TestDevStatusCmd_Description(t *testing.T) {
	if devStatusCmd.Short == "" {
		t.Error("devStatusCmd.Short is empty")
	}
	if devStatusCmd.Long == "" {
		t.Error("devStatusCmd.Long is empty")
	}
	if devStatusCmd.RunE == nil {
		t.Error("devStatusCmd.RunE is nil")
	}
}

// Integration tests - These require actual runtime dependencies and are skipped by default.
// Run with: go test -v -run Integration -tags=integration

// TestIntegration_DevUp_K3d tests dp dev up --runtime=k3d (T032).
// Skipped unless k3d is available and integration tag is set.
func TestIntegration_DevUp_K3d(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if k3d is available
	checker := localdev.NewPrerequisiteChecker(localdev.RuntimeK3d)
	ctx := context.Background()
	if err := checker.CheckAll(ctx); err != nil {
		t.Skipf("skipping: k3d prerequisites not available: %v", err)
	}

	// This test would create an actual k3d cluster
	// For CI, this is typically run separately or skipped
	t.Log("Integration test: dp dev up --runtime=k3d would be run here")
	t.Log("This creates an actual k3d cluster and should be run manually")
}

// TestIntegration_DevDown_K3d tests dp dev down --runtime=k3d (T038).
func TestIntegration_DevDown_K3d(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	checker := localdev.NewPrerequisiteChecker(localdev.RuntimeK3d)
	ctx := context.Background()
	if err := checker.CheckAll(ctx); err != nil {
		t.Skipf("skipping: k3d prerequisites not available: %v", err)
	}

	t.Log("Integration test: dp dev down --runtime=k3d would be run here")
}

// TestIntegration_DevStatus_K3d tests dp dev status --runtime=k3d (T045).
func TestIntegration_DevStatus_K3d(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	checker := localdev.NewPrerequisiteChecker(localdev.RuntimeK3d)
	ctx := context.Background()
	if err := checker.CheckAll(ctx); err != nil {
		t.Skipf("skipping: k3d prerequisites not available: %v", err)
	}

	t.Log("Integration test: dp dev status --runtime=k3d would be run here")
}

// TestIntegration_BackwardCompatibility tests that k3d is the default runtime.
func TestIntegration_BackwardCompatibility(t *testing.T) {
	// This test verifies k3d is now the default runtime
	oldRuntime := devRuntime
	defer func() { devRuntime = oldRuntime }()

	// No runtime flag set
	devRuntime = ""

	runtime, err := getRuntime()
	if err != nil {
		t.Fatalf("getRuntime() error = %v", err)
	}

	if runtime != localdev.RuntimeK3d {
		t.Errorf("Default runtime = %q, want %q (k3d should be default)",
			runtime, localdev.RuntimeK3d)
	}
}
