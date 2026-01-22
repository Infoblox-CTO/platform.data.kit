// Package logging provides structured logging for DP.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Logger is the structured logger interface.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
}

// SlogLogger implements Logger using slog.
type SlogLogger struct {
	logger *slog.Logger
}

// Config contains logger configuration.
type Config struct {
	// Level is the minimum log level.
	Level string
	// Format is the output format (json, text).
	Format string
	// Output is the output writer.
	Output io.Writer
	// AddSource adds source file info to logs.
	AddSource bool
}

// DefaultConfig returns the default logger configuration.
func DefaultConfig() Config {
	return Config{
		Level:     "info",
		Format:    "json",
		Output:    os.Stdout,
		AddSource: false,
	}
}

// New creates a new Logger with the given configuration.
func New(cfg Config) Logger {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(cfg.Output, opts)
	} else {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	return &SlogLogger{
		logger: slog.New(handler),
	}
}

// NewDefault creates a new Logger with default configuration.
func NewDefault() Logger {
	return New(DefaultConfig())
}

// parseLevel parses a log level string.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug implements Logger.
func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info implements Logger.
func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn implements Logger.
func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error implements Logger.
func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// With returns a new Logger with the given attributes.
func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}

// WithContext returns a new Logger with context.
func (l *SlogLogger) WithContext(ctx context.Context) Logger {
	return l
}

// Ensure SlogLogger implements Logger.
var _ Logger = (*SlogLogger)(nil)

// Standard logger fields.
const (
	FieldPackage     = "package"
	FieldNamespace   = "namespace"
	FieldVersion     = "version"
	FieldEnvironment = "environment"
	FieldRunID       = "run_id"
	FieldDuration    = "duration_ms"
	FieldError       = "error"
	FieldRecords     = "records"
	FieldBytes       = "bytes"
)

// ForRun returns a logger configured for a specific run.
func ForRun(base Logger, runID, pkg, namespace, version, env string) Logger {
	return base.With(
		FieldRunID, runID,
		FieldPackage, pkg,
		FieldNamespace, namespace,
		FieldVersion, version,
		FieldEnvironment, env,
	)
}

// NopLogger is a no-op logger.
type NopLogger struct{}

// NewNopLogger creates a new no-op logger.
func NewNopLogger() Logger {
	return &NopLogger{}
}

// Debug implements Logger.
func (l *NopLogger) Debug(msg string, args ...any) {}

// Info implements Logger.
func (l *NopLogger) Info(msg string, args ...any) {}

// Warn implements Logger.
func (l *NopLogger) Warn(msg string, args ...any) {}

// Error implements Logger.
func (l *NopLogger) Error(msg string, args ...any) {}

// With implements Logger.
func (l *NopLogger) With(args ...any) Logger { return l }

// WithContext implements Logger.
func (l *NopLogger) WithContext(ctx context.Context) Logger { return l }

// Ensure NopLogger implements Logger.
var _ Logger = (*NopLogger)(nil)
