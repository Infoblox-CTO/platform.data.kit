package runner

import (
	"testing"
	"time"

	"github.com/Infoblox-CTO/data.platform.kit/contracts"
)

func TestDefaultRunOptions(t *testing.T) {
	opts := DefaultRunOptions("/path/to/package")

	if opts.PackageDir != "/path/to/package" {
		t.Errorf("PackageDir = %s, want /path/to/package", opts.PackageDir)
	}
	if opts.Env == nil {
		t.Error("Env should not be nil")
	}
	if opts.Network != "dp-network" {
		t.Errorf("Network = %s, want dp-network", opts.Network)
	}
	if opts.Timeout != 30*time.Minute {
		t.Errorf("Timeout = %v, want 30m", opts.Timeout)
	}
	if opts.DryRun {
		t.Error("DryRun should be false by default")
	}
	if opts.Detach {
		t.Error("Detach should be false by default")
	}
}

func TestRunOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    RunOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: RunOptions{
				PackageDir: "/path/to/package",
			},
			wantErr: false,
		},
		{
			name: "empty package dir",
			opts: RunOptions{
				PackageDir: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateRunID(t *testing.T) {
	runID := GenerateRunID("my-package")

	if runID == "" {
		t.Error("runID should not be empty")
	}
	if len(runID) < len("my-package-") {
		t.Error("runID should include package name")
	}
}

func TestRunResult(t *testing.T) {
	now := time.Now()
	endTime := now.Add(5 * time.Minute)

	result := &RunResult{
		RunID:            "my-package-20240101-120000",
		Status:           contracts.RunStatusCompleted,
		StartTime:        now,
		EndTime:          &endTime,
		Duration:         5 * time.Minute,
		ExitCode:         0,
		RecordsProcessed: 1000,
		ContainerID:      "abc123",
	}

	if result.RunID == "" {
		t.Error("RunID should not be empty")
	}
	if result.Status != contracts.RunStatusCompleted {
		t.Errorf("Status = %s, want completed", result.Status)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestRunResult_WithError(t *testing.T) {
	result := &RunResult{
		RunID:    "my-package-20240101-120000",
		Status:   contracts.RunStatusFailed,
		ExitCode: 1,
		Error:    "pipeline execution failed",
	}

	if result.Status != contracts.RunStatusFailed {
		t.Errorf("Status = %s, want failed", result.Status)
	}
	if result.Error == "" {
		t.Error("Error should not be empty for failed run")
	}
}

func TestRegisterAndGetRunner(t *testing.T) {
	RegisterRunner("test-runner-1", func() (Runner, error) {
		return nil, nil
	})

	_, err := GetRunner("test-runner-1")
	if err != nil {
		t.Errorf("GetRunner() error = %v", err)
	}
}

func TestGetRunner_Unknown(t *testing.T) {
	_, err := GetRunner("unknown-runner-xyz")
	if err == nil {
		t.Error("expected error for unknown runner")
	}
}

func TestListRunners(t *testing.T) {
	RegisterRunner("docker-test", func() (Runner, error) { return nil, nil })

	list := ListRunners()
	if len(list) == 0 {
		t.Error("ListRunners should return registered runners")
	}
}
