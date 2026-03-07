package lineage

import (
	"context"
	"testing"
)

func TestEmitterInterface(t *testing.T) {
	var _ Emitter = (*NoopEmitter)(nil)
	var _ Emitter = (*ConsoleEmitter)(nil)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Type != "noop" {
		t.Errorf("Type = %s, want noop", config.Type)
	}
	if config.Namespace != "default" {
		t.Errorf("Namespace = %s, want default", config.Namespace)
	}
	if config.TimeoutSeconds != 30 {
		t.Errorf("TimeoutSeconds = %d, want 30", config.TimeoutSeconds)
	}
	if config.BatchSize != 0 {
		t.Errorf("BatchSize = %d, want 0", config.BatchSize)
	}
	if config.FlushIntervalSeconds != 10 {
		t.Errorf("FlushIntervalSeconds = %d, want 10", config.FlushIntervalSeconds)
	}
}

func TestEmitterConfig(t *testing.T) {
	config := EmitterConfig{
		Type:                 "marquez",
		Endpoint:             "http://localhost:5000",
		Namespace:            "datakit",
		APIKey:               "secret-key",
		TimeoutSeconds:       60,
		BatchSize:            10,
		FlushIntervalSeconds: 5,
	}

	if config.Type != "marquez" {
		t.Errorf("Type = %s, want marquez", config.Type)
	}
	if config.Endpoint != "http://localhost:5000" {
		t.Errorf("Endpoint = %s, want http://localhost:5000", config.Endpoint)
	}
}

func TestNoopEmitter(t *testing.T) {
	emitter := &NoopEmitter{}

	event := NewEvent(EventTypeStart, "run-123", "ns", "job")
	err := emitter.Emit(context.Background(), event)
	if err != nil {
		t.Errorf("Emit() error = %v", err)
	}

	err = emitter.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewNoopEmitter(t *testing.T) {
	emitter := NewNoopEmitter()
	if emitter == nil {
		t.Error("NewNoopEmitter should not return nil")
	}

	// Verify it's a NoopEmitter
	_, ok := emitter.(*NoopEmitter)
	if !ok {
		t.Error("NewNoopEmitter should return *NoopEmitter")
	}
}

func TestConsoleEmitter(t *testing.T) {
	var loggedMessages []string
	logger := func(format string, args ...interface{}) {
		loggedMessages = append(loggedMessages, format)
	}

	emitter := NewConsoleEmitter(logger)
	if emitter == nil {
		t.Fatal("NewConsoleEmitter should not return nil")
	}

	event := NewEvent(EventTypeComplete, "run-123", "ns", "job")
	event.Inputs = []Dataset{{Namespace: "kafka", Name: "topic"}}
	event.Outputs = []Dataset{{Namespace: "s3", Name: "bucket"}}

	err := emitter.Emit(context.Background(), event)
	if err != nil {
		t.Errorf("Emit() error = %v", err)
	}

	err = emitter.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewConsoleEmitter_NilLogger(t *testing.T) {
	emitter := NewConsoleEmitter(nil)
	if emitter == nil {
		t.Fatal("NewConsoleEmitter with nil logger should not return nil")
	}

	// Should not panic
	event := NewEvent(EventTypeStart, "run-123", "ns", "job")
	err := emitter.Emit(context.Background(), event)
	if err != nil {
		t.Errorf("Emit() error = %v", err)
	}
}

func TestEmitter_EmitAllEventTypes(t *testing.T) {
	emitter := NewNoopEmitter()
	ctx := context.Background()

	eventTypes := []EventType{
		EventTypeStart,
		EventTypeRunning,
		EventTypeComplete,
		EventTypeFail,
		EventTypeAbort,
	}

	for _, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			event := NewEvent(eventType, "run-123", "ns", "job")
			err := emitter.Emit(ctx, event)
			if err != nil {
				t.Errorf("Emit(%s) error = %v", eventType, err)
			}
		})
	}
}

func TestEmitter_EmitWithContext(t *testing.T) {
	emitter := NewNoopEmitter()

	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := NewEvent(EventTypeStart, "run-123", "ns", "job")
	err := emitter.Emit(ctx, event)
	// NoopEmitter ignores context, so no error expected
	if err != nil {
		t.Errorf("Emit() error = %v", err)
	}
}
