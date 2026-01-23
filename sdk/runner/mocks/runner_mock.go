package mocks

import (
	"context"
	"io"

	"github.com/Infoblox-CTO/platform.data.kit/sdk/runner"
)

// MockRunner is a mock implementation of runner.Runner for testing.
type MockRunner struct {
	RunFunc    func(ctx context.Context, opts runner.RunOptions) (*runner.RunResult, error)
	StopFunc   func(ctx context.Context, runID string) error
	LogsFunc   func(ctx context.Context, runID string, follow bool, output io.Writer) error
	StatusFunc func(ctx context.Context, runID string) (*runner.RunResult, error)
}

func (m *MockRunner) Run(ctx context.Context, opts runner.RunOptions) (*runner.RunResult, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, opts)
	}
	return &runner.RunResult{
		RunID:  runner.GenerateRunID("mock"),
		Status: "completed",
	}, nil
}

func (m *MockRunner) Stop(ctx context.Context, runID string) error {
	if m.StopFunc != nil {
		return m.StopFunc(ctx, runID)
	}
	return nil
}

func (m *MockRunner) Logs(ctx context.Context, runID string, follow bool, output io.Writer) error {
	if m.LogsFunc != nil {
		return m.LogsFunc(ctx, runID, follow, output)
	}
	return nil
}

func (m *MockRunner) Status(ctx context.Context, runID string) (*runner.RunResult, error) {
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx, runID)
	}
	return &runner.RunResult{
		RunID:  runID,
		Status: "running",
	}, nil
}

var _ runner.Runner = (*MockRunner)(nil)
