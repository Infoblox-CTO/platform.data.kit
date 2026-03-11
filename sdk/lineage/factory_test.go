package lineage

import "testing"

func TestNewEmitterFromConfig_Noop(t *testing.T) {
	emitter, err := NewEmitterFromConfig(EmitterConfig{Type: "noop"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := emitter.(*NoopEmitter); !ok {
		t.Errorf("expected *NoopEmitter, got %T", emitter)
	}
}

func TestNewEmitterFromConfig_EmptyType(t *testing.T) {
	emitter, err := NewEmitterFromConfig(EmitterConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := emitter.(*NoopEmitter); !ok {
		t.Errorf("expected *NoopEmitter for empty type, got %T", emitter)
	}
}

func TestNewEmitterFromConfig_Console(t *testing.T) {
	emitter, err := NewEmitterFromConfig(EmitterConfig{Type: "console"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := emitter.(*ConsoleEmitter); !ok {
		t.Errorf("expected *ConsoleEmitter, got %T", emitter)
	}
}

func TestNewEmitterFromConfig_Marquez(t *testing.T) {
	emitter, err := NewEmitterFromConfig(EmitterConfig{
		Type:      "marquez",
		Namespace: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := emitter.(*MarquezEmitter); !ok {
		t.Errorf("expected *MarquezEmitter, got %T", emitter)
	}
}

func TestNewEmitterFromConfig_MarquezDefaultEndpoint(t *testing.T) {
	emitter, err := NewEmitterFromConfig(EmitterConfig{
		Type: "marquez",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	me, ok := emitter.(*MarquezEmitter)
	if !ok {
		t.Fatalf("expected *MarquezEmitter, got %T", emitter)
	}
	if me.endpoint != "http://localhost:5000" {
		t.Errorf("expected default endpoint %q, got %q", "http://localhost:5000", me.endpoint)
	}
}

func TestNewEmitterFromConfig_DataHub(t *testing.T) {
	emitter, err := NewEmitterFromConfig(EmitterConfig{
		Type:     "datahub",
		Endpoint: "http://datahub-gms:8080",
		APIKey:   "my-token",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := emitter.(*DataHubEmitter); !ok {
		t.Errorf("expected *DataHubEmitter, got %T", emitter)
	}
}

func TestNewEmitterFromConfig_DataHubNoEndpoint(t *testing.T) {
	_, err := NewEmitterFromConfig(EmitterConfig{
		Type: "datahub",
	})
	if err == nil {
		t.Fatal("expected error for datahub without endpoint")
	}
}

func TestNewEmitterFromConfig_HTTP(t *testing.T) {
	emitter, err := NewEmitterFromConfig(EmitterConfig{
		Type:     "http",
		Endpoint: "http://lineage-server:5000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// HTTP transport reuses MarquezEmitter (standard OpenLineage HTTP)
	if _, ok := emitter.(*MarquezEmitter); !ok {
		t.Errorf("expected *MarquezEmitter for http type, got %T", emitter)
	}
}

func TestNewEmitterFromConfig_HTTPNoEndpoint(t *testing.T) {
	_, err := NewEmitterFromConfig(EmitterConfig{
		Type: "http",
	})
	if err == nil {
		t.Fatal("expected error for http without endpoint")
	}
}

func TestNewEmitterFromConfig_Unknown(t *testing.T) {
	_, err := NewEmitterFromConfig(EmitterConfig{Type: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}
