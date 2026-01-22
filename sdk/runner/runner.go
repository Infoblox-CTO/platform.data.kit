// Package runner provides local execution capabilities for DP pipelines.
package runner

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Infoblox-CTO/data-platform/contracts"
	"github.com/Infoblox-CTO/data-platform/sdk/lineage"
)

// Runner defines the interface for executing pipelines locally.
type Runner interface {
	Run(ctx context.Context, opts RunOptions) (*RunResult, error)
	Stop(ctx context.Context, runID string) error
	Logs(ctx context.Context, runID string, follow bool, output io.Writer) error
	Status(ctx context.Context, runID string) (*RunResult, error)
}

// RunOptions contains options for running a pipeline.
type RunOptions struct {
	PackageDir     string
	Env            map[string]string
	BindingsFile   string
	Network        string
	Timeout        time.Duration
	DryRun         bool
	Detach         bool
	Output         io.Writer
	LineageEmitter lineage.Emitter // Optional lineage emitter for tracking
}

// RunResult contains the result of a pipeline run.
type RunResult struct {
	RunID            string
	Status           contracts.RunStatus
	StartTime        time.Time
	EndTime          *time.Time
	Duration         time.Duration
	ExitCode         int
	RecordsProcessed int64
	Error            string
	ContainerID      string
}

// DefaultRunOptions returns RunOptions with sensible defaults.
func DefaultRunOptions(packageDir string) RunOptions {
	return RunOptions{
		PackageDir: packageDir,
		Env:        make(map[string]string),
		Network:    "dp-network",
		Timeout:    30 * time.Minute,
		DryRun:     false,
		Detach:     false,
	}
}

// Validate validates the run options.
func (o *RunOptions) Validate() error {
	if o.PackageDir == "" {
		return fmt.Errorf("package directory is required")
	}
	return nil
}

// GenerateRunID generates a unique run ID.
func GenerateRunID(packageName string) string {
	timestamp := time.Now().UTC().Format("20060102-150405")
	return fmt.Sprintf("%s-%s", packageName, timestamp)
}

// RunnerFactory creates runners of a specific type.
type RunnerFactory func() (Runner, error)

var runners = make(map[string]RunnerFactory)

// RegisterRunner registers a runner factory.
func RegisterRunner(name string, factory RunnerFactory) {
	runners[name] = factory
}

// GetRunner returns a runner by name.
func GetRunner(name string) (Runner, error) {
	factory, ok := runners[name]
	if !ok {
		return nil, fmt.Errorf("unknown runner: %s", name)
	}
	return factory()
}

// ListRunners returns the names of all registered runners.
func ListRunners() []string {
	names := make([]string, 0, len(runners))
	for name := range runners {
		names = append(names, name)
	}
	return names
}
