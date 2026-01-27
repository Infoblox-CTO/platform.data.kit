package lineage

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestHeartbeat_Start(t *testing.T) {
	emitter := &mockEmitter{}
	config := HeartbeatConfig{
		Interval: 50 * time.Millisecond,
		RunID:    "test-run-123",
		Job: Job{
			Namespace: "test",
			Name:      "test-job",
		},
		Producer: "test-producer",
	}

	heartbeat := NewHeartbeat(emitter, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeat.Start(ctx)

	// Wait for a couple heartbeats
	time.Sleep(150 * time.Millisecond)

	heartbeat.Stop()

	// Should have emitted at least 2 events (initial + 2 intervals)
	if emitter.count() < 2 {
		t.Errorf("Expected at least 2 heartbeat events, got %d", emitter.count())
	}

	// Check event type
	events := emitter.events()
	for _, event := range events {
		if event.EventType != EventTypeRunning {
			t.Errorf("Expected RUNNING event type, got %v", event.EventType)
		}
		if event.Run.RunID != "test-run-123" {
			t.Errorf("Expected runID test-run-123, got %v", event.Run.RunID)
		}
	}
}

func TestHeartbeat_Stop(t *testing.T) {
	emitter := &mockEmitter{}
	config := HeartbeatConfig{
		Interval: 10 * time.Millisecond,
		RunID:    "test-run",
		Job: Job{
			Namespace: "test",
			Name:      "test-job",
		},
	}

	heartbeat := NewHeartbeat(emitter, config)
	ctx := context.Background()

	heartbeat.Start(ctx)
	time.Sleep(25 * time.Millisecond)
	heartbeat.Stop()

	countAfterStop := emitter.count()

	// Wait a bit more - no new events should be emitted
	time.Sleep(50 * time.Millisecond)

	if emitter.count() != countAfterStop {
		t.Errorf("Expected no new events after stop, got %d more", emitter.count()-countAfterStop)
	}
}

func TestHeartbeat_UpdateRecordsProcessed(t *testing.T) {
	emitter := &mockEmitter{}
	config := HeartbeatConfig{
		Interval: 50 * time.Millisecond,
		RunID:    "test-run",
		Job: Job{
			Namespace: "test",
			Name:      "test-job",
		},
	}

	heartbeat := NewHeartbeat(emitter, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeat.Start(ctx)
	heartbeat.UpdateRecordsProcessed(1000)

	time.Sleep(100 * time.Millisecond)
	heartbeat.Stop()

	// Find an event with records processed
	events := emitter.events()
	found := false
	for _, event := range events {
		if facets, ok := event.Run.Facets["streaming"].(map[string]interface{}); ok {
			if records, ok := facets["records_processed"].(int64); ok && records == 1000 {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Expected to find event with records_processed = 1000")
	}
}

func TestHeartbeat_DefaultConfig(t *testing.T) {
	config := DefaultHeartbeatConfig()

	if config.Interval != 30*time.Second {
		t.Errorf("Expected default interval 30s, got %v", config.Interval)
	}
	if config.Producer != "dp-runner" {
		t.Errorf("Expected default producer dp-runner, got %v", config.Producer)
	}
}

func TestNewHeartbeat_DefaultValues(t *testing.T) {
	emitter := &mockEmitter{}
	config := HeartbeatConfig{
		RunID: "test",
		Job:   Job{Namespace: "test", Name: "test"},
		// Interval and Producer not set
	}

	heartbeat := NewHeartbeat(emitter, config)

	if heartbeat.config.Interval != 30*time.Second {
		t.Errorf("Expected default interval 30s, got %v", heartbeat.config.Interval)
	}
	if heartbeat.config.Producer != "dp-runner" {
		t.Errorf("Expected default producer dp-runner, got %v", heartbeat.config.Producer)
	}
}

func TestHeartbeat_DoubleStart(t *testing.T) {
	emitter := &mockEmitter{}
	config := HeartbeatConfig{
		Interval: 50 * time.Millisecond,
		RunID:    "test-run",
		Job:      Job{Namespace: "test", Name: "test"},
	}

	heartbeat := NewHeartbeat(emitter, config)
	ctx := context.Background()

	// Start twice should not panic or start twice
	heartbeat.Start(ctx)
	heartbeat.Start(ctx)

	time.Sleep(100 * time.Millisecond)
	heartbeat.Stop()

	// Should still work normally
	if emitter.count() < 1 {
		t.Error("Expected at least 1 heartbeat event")
	}
}

func TestHeartbeat_DoubleStop(t *testing.T) {
	emitter := &mockEmitter{}
	config := HeartbeatConfig{
		Interval: 50 * time.Millisecond,
		RunID:    "test-run",
		Job:      Job{Namespace: "test", Name: "test"},
	}

	heartbeat := NewHeartbeat(emitter, config)
	ctx := context.Background()

	heartbeat.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Stop twice should not panic
	heartbeat.Stop()
	heartbeat.Stop() // Should be a no-op
}

func TestParseHeartbeatInterval(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30s", 30 * time.Second, false},
		{"1m", time.Minute, false},
		{"5m30s", 5*time.Minute + 30*time.Second, false},
		{"", 30 * time.Second, false}, // Default
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseHeartbeatInterval(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHeartbeatInterval(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("ParseHeartbeatInterval(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// mockEmitter is a test emitter that records events
type mockEmitter struct {
	mu      sync.Mutex
	emitted []*Event
}

func (m *mockEmitter) Emit(ctx context.Context, event *Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emitted = append(m.emitted, event)
	return nil
}

func (m *mockEmitter) Close() error {
	return nil
}

func (m *mockEmitter) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.emitted)
}

func (m *mockEmitter) events() []*Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*Event, len(m.emitted))
	copy(result, m.emitted)
	return result
}
