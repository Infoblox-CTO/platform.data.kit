// Package lineage provides heartbeat functionality for streaming pipelines.
package lineage

import (
	"context"
	"sync"
	"time"
)

// HeartbeatConfig contains configuration for lineage heartbeats.
type HeartbeatConfig struct {
	// Interval is how often to emit RUNNING events.
	Interval time.Duration
	// Job identifies the job to emit heartbeats for.
	Job Job
	// RunID is the unique identifier for this run.
	RunID string
	// Producer identifies the producer of heartbeat events.
	Producer string
}

// DefaultHeartbeatConfig returns a default heartbeat configuration.
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Interval: 30 * time.Second,
		Producer: "dk-runner",
	}
}

// Heartbeat emits periodic RUNNING events for streaming pipelines.
type Heartbeat struct {
	config           HeartbeatConfig
	emitter          Emitter
	stopCh           chan struct{}
	stoppedCh        chan struct{}
	mu               sync.Mutex
	running          bool
	startTime        time.Time
	recordsProcessed int64
}

// NewHeartbeat creates a new heartbeat emitter.
func NewHeartbeat(emitter Emitter, config HeartbeatConfig) *Heartbeat {
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.Producer == "" {
		config.Producer = "dk-runner"
	}

	return &Heartbeat{
		config:    config,
		emitter:   emitter,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

// Start begins emitting heartbeat events at the configured interval.
func (h *Heartbeat) Start(ctx context.Context) {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.startTime = time.Now()
	h.mu.Unlock()

	go h.run(ctx)
}

// Stop stops emitting heartbeat events.
func (h *Heartbeat) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.mu.Unlock()

	close(h.stopCh)
	<-h.stoppedCh
}

// UpdateRecordsProcessed updates the records processed count for heartbeat facets.
func (h *Heartbeat) UpdateRecordsProcessed(count int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recordsProcessed = count
}

func (h *Heartbeat) run(ctx context.Context) {
	defer close(h.stoppedCh)

	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()

	// Emit initial heartbeat
	h.emitHeartbeat(ctx)

	for {
		select {
		case <-ticker.C:
			h.emitHeartbeat(ctx)
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (h *Heartbeat) emitHeartbeat(ctx context.Context) {
	h.mu.Lock()
	uptime := time.Since(h.startTime)
	recordsProcessed := h.recordsProcessed
	h.mu.Unlock()

	event := &Event{
		EventTime: time.Now().UTC(),
		Producer:  h.config.Producer,
		SchemaURL: SchemaVersion,
		EventType: EventTypeRunning,
		Run: Run{
			RunID: h.config.RunID,
			Facets: map[string]interface{}{
				"streaming": map[string]interface{}{
					"uptime_seconds":    uptime.Seconds(),
					"records_processed": recordsProcessed,
					"last_heartbeat":    time.Now().UTC().Format(time.RFC3339),
				},
			},
		},
		Job: h.config.Job,
	}

	// Emit the heartbeat event, ignoring errors (heartbeats are best-effort)
	_ = h.emitter.Emit(ctx, event)
}

// ParseHeartbeatInterval parses a heartbeat interval string (e.g., "30s", "1m").
func ParseHeartbeatInterval(s string) (time.Duration, error) {
	if s == "" {
		return 30 * time.Second, nil
	}
	return time.ParseDuration(s)
}
