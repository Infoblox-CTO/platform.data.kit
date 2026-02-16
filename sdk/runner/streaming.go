package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// RunStreaming executes a streaming pipeline that runs indefinitely.
// Streaming pipelines are expected to run continuously until stopped.
func (r *DockerRunner) RunStreaming(ctx context.Context, opts RunOptions, image string, result *RunResult, args []string) error {
	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Starting streaming pipeline: docker %s\n\n", strings.Join(args, " "))
	}

	// For streaming, we run in detached mode but attach to logs
	if opts.Detach {
		return r.runStreamingDetached(ctx, opts, image, result, args)
	}

	return r.runStreamingAttached(ctx, opts, image, result, args)
}

// runStreamingDetached runs the container in the background.
func (r *DockerRunner) runStreamingDetached(ctx context.Context, opts RunOptions, image string, result *RunResult, args []string) error {
	// Add -d flag for detached mode
	detachedArgs := make([]string, 0, len(args)+1)
	for i, arg := range args {
		detachedArgs = append(detachedArgs, arg)
		if arg == "run" && i < len(args)-1 {
			detachedArgs = append(detachedArgs, "-d")
		}
	}

	cmd := exec.CommandContext(ctx, "docker", detachedArgs...)
	output, err := cmd.Output()
	if err != nil {
		result.Status = contracts.RunStatusFailed
		result.Error = err.Error()
		return err
	}

	containerID := strings.TrimSpace(string(output))
	result.ContainerID = containerID
	result.Status = contracts.RunStatusRunning

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "Container started: %s\n", containerID[:12])
		fmt.Fprintf(opts.Output, "Use 'dp logs' to view output\n")
	}

	return nil
}

// runStreamingAttached runs the container and streams logs.
func (r *DockerRunner) runStreamingAttached(ctx context.Context, opts RunOptions, image string, result *RunResult, args []string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	result.Status = contracts.RunStatusRunning

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		result.Status = contracts.RunStatusFailed
		result.Error = err.Error()
		return err
	}

	// Stream output in goroutines
	go streamOutput(stdout, opts.Output)
	go streamOutput(stderr, opts.Output)

	// Wait for either signal or process exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case sig := <-sigChan:
		if opts.Output != nil {
			fmt.Fprintf(opts.Output, "\nReceived %s, shutting down gracefully...\n", sig)
		}
		// Send SIGTERM to container for graceful shutdown
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			cmd.Process.Kill()
		}
		// Wait for graceful shutdown (max 30s)
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			cmd.Process.Kill()
		}
		result.Status = contracts.RunStatusCompleted
		endTime := time.Now()
		result.EndTime = &endTime
		result.Duration = endTime.Sub(result.StartTime)
		return nil

	case err := <-done:
		endTime := time.Now()
		result.EndTime = &endTime
		result.Duration = endTime.Sub(result.StartTime)
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
			}
			result.Status = contracts.RunStatusFailed
			result.Error = err.Error()
			return err
		}
		result.Status = contracts.RunStatusCompleted
		result.ExitCode = 0
		return nil
	}
}

// streamOutput streams data from a reader to a writer.
func streamOutput(r io.Reader, w io.Writer) {
	if w == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintln(w, scanner.Text())
	}
}

// IsStreamingMode checks if a pipeline is configured for streaming mode.
func IsStreamingMode(mode contracts.Mode) bool {
	return mode == contracts.ModeStreaming
}
