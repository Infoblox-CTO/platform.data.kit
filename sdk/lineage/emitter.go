// Package lineage provides emitter interfaces and implementations
// for sending OpenLineage events to lineage collection backends.
package lineage

import (
	"context"
)

// Emitter is the interface for sending lineage events.
type Emitter interface {
	// Emit sends a lineage event to the backend.
	Emit(ctx context.Context, event *Event) error
	// Close releases any resources held by the emitter.
	Close() error
}

// EmitterConfig contains configuration for lineage emitters.
type EmitterConfig struct {
	// Type is the emitter type (e.g., "marquez", "http", "console", "noop").
	Type string
	// Endpoint is the backend URL for HTTP-based emitters.
	Endpoint string
	// Namespace is the default namespace for events.
	Namespace string
	// APIKey is an optional API key for authentication.
	APIKey string
	// Timeout is the HTTP request timeout.
	TimeoutSeconds int
	// BatchSize is the number of events to batch before sending (0 = no batching).
	BatchSize int
	// FlushIntervalSeconds is the interval to flush batched events.
	FlushIntervalSeconds int
}

// DefaultConfig returns a default emitter configuration.
func DefaultConfig() EmitterConfig {
	return EmitterConfig{
		Type:                 "noop",
		Namespace:            "default",
		TimeoutSeconds:       30,
		BatchSize:            0,
		FlushIntervalSeconds: 10,
	}
}

// NoopEmitter is an emitter that does nothing, useful for testing.
type NoopEmitter struct{}

// Emit does nothing and returns nil.
func (n *NoopEmitter) Emit(ctx context.Context, event *Event) error {
	return nil
}

// Close does nothing and returns nil.
func (n *NoopEmitter) Close() error {
	return nil
}

// NewNoopEmitter creates a new no-op emitter.
func NewNoopEmitter() Emitter {
	return &NoopEmitter{}
}

// ConsoleEmitter logs events to standard output for debugging.
type ConsoleEmitter struct {
	// Logger is the logger to use for output.
	logger func(format string, args ...interface{})
}

// NewConsoleEmitter creates a new console emitter.
func NewConsoleEmitter(logger func(format string, args ...interface{})) Emitter {
	if logger == nil {
		logger = func(format string, args ...interface{}) {}
	}
	return &ConsoleEmitter{logger: logger}
}

// Emit logs the event to the console.
func (c *ConsoleEmitter) Emit(ctx context.Context, event *Event) error {
	c.logger("[LINEAGE] %s - Job: %s/%s, Run: %s, Inputs: %d, Outputs: %d\n",
		event.EventType,
		event.Job.Namespace,
		event.Job.Name,
		event.Run.RunID,
		len(event.Inputs),
		len(event.Outputs),
	)
	return nil
}

// Close does nothing for console emitter.
func (c *ConsoleEmitter) Close() error {
	return nil
}

// MultiEmitter sends events to multiple emitters.
type MultiEmitter struct {
	emitters []Emitter
}

// NewMultiEmitter creates an emitter that sends to multiple backends.
func NewMultiEmitter(emitters ...Emitter) Emitter {
	return &MultiEmitter{emitters: emitters}
}

// Emit sends the event to all configured emitters.
func (m *MultiEmitter) Emit(ctx context.Context, event *Event) error {
	var lastErr error
	for _, e := range m.emitters {
		if err := e.Emit(ctx, event); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Close closes all configured emitters.
func (m *MultiEmitter) Close() error {
	var lastErr error
	for _, e := range m.emitters {
		if err := e.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// EventBuilder provides a fluent API for building lineage events.
type EventBuilder struct {
	event   *Event
	emitter Emitter
	jobNS   string
	jobName string
}

// NewEventBuilder creates a new event builder.
func NewEventBuilder(emitter Emitter, jobNamespace, jobName string) *EventBuilder {
	return &EventBuilder{
		emitter: emitter,
		jobNS:   jobNamespace,
		jobName: jobName,
	}
}

// Start creates and emits a START event.
func (b *EventBuilder) Start(ctx context.Context, runID string) error {
	b.event = NewEvent(EventTypeStart, runID, b.jobNS, b.jobName)
	return b.emitter.Emit(ctx, b.event)
}

// Running creates and emits a RUNNING event.
func (b *EventBuilder) Running(ctx context.Context, runID string) error {
	b.event = NewEvent(EventTypeRunning, runID, b.jobNS, b.jobName)
	return b.emitter.Emit(ctx, b.event)
}

// Complete creates and emits a COMPLETE event.
func (b *EventBuilder) Complete(ctx context.Context, runID string) error {
	b.event = NewEvent(EventTypeComplete, runID, b.jobNS, b.jobName)
	return b.emitter.Emit(ctx, b.event)
}

// Fail creates and emits a FAIL event with error information.
func (b *EventBuilder) Fail(ctx context.Context, runID string, err error, stackTrace string) error {
	b.event = NewEvent(EventTypeFail, runID, b.jobNS, b.jobName)
	b.event.WithErrorFacet(err.Error(), stackTrace)
	return b.emitter.Emit(ctx, b.event)
}

// Abort creates and emits an ABORT event.
func (b *EventBuilder) Abort(ctx context.Context, runID string) error {
	b.event = NewEvent(EventTypeAbort, runID, b.jobNS, b.jobName)
	return b.emitter.Emit(ctx, b.event)
}

// WithInputs sets the input datasets for the current event.
func (b *EventBuilder) WithInputs(datasets ...Dataset) *EventBuilder {
	if b.event != nil {
		b.event.Inputs = datasets
	}
	return b
}

// WithOutputs sets the output datasets for the current event.
func (b *EventBuilder) WithOutputs(datasets ...Dataset) *EventBuilder {
	if b.event != nil {
		b.event.Outputs = datasets
	}
	return b
}

// Event returns the current event.
func (b *EventBuilder) Event() *Event {
	return b.event
}
