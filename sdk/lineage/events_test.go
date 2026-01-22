package lineage

import (
	"testing"
	"time"
)

func TestEventType_Constants(t *testing.T) {
	tests := []struct {
		eventType EventType
		want      string
	}{
		{EventTypeStart, "START"},
		{EventTypeRunning, "RUNNING"},
		{EventTypeComplete, "COMPLETE"},
		{EventTypeFail, "FAIL"},
		{EventTypeAbort, "ABORT"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			if string(tt.eventType) != tt.want {
				t.Errorf("EventType = %s, want %s", tt.eventType, tt.want)
			}
		})
	}
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventTypeStart, "run-123", "data-team", "my-pipeline")

	if event == nil {
		t.Fatal("NewEvent should not return nil")
	}
	if event.EventType != EventTypeStart {
		t.Errorf("EventType = %s, want START", event.EventType)
	}
	if event.Run.RunID != "run-123" {
		t.Errorf("RunID = %s, want run-123", event.Run.RunID)
	}
	if event.Job.Namespace != "data-team" {
		t.Errorf("Job.Namespace = %s, want data-team", event.Job.Namespace)
	}
	if event.Job.Name != "my-pipeline" {
		t.Errorf("Job.Name = %s, want my-pipeline", event.Job.Name)
	}
	if event.SchemaURL != SchemaVersion {
		t.Errorf("SchemaURL = %s, want %s", event.SchemaURL, SchemaVersion)
	}
	if event.Producer != "dp" {
		t.Errorf("Producer = %s, want dp", event.Producer)
	}
}

func TestNewEvent_InitializesCollections(t *testing.T) {
	event := NewEvent(EventTypeStart, "run-123", "ns", "job")

	if event.Inputs == nil {
		t.Error("Inputs should be initialized")
	}
	if event.Outputs == nil {
		t.Error("Outputs should be initialized")
	}
	if event.Run.Facets == nil {
		t.Error("Run.Facets should be initialized")
	}
	if event.Job.Facets == nil {
		t.Error("Job.Facets should be initialized")
	}
}

func TestEvent_EventTime(t *testing.T) {
	before := time.Now().UTC()
	event := NewEvent(EventTypeStart, "run-123", "ns", "job")
	after := time.Now().UTC()

	if event.EventTime.Before(before) || event.EventTime.After(after) {
		t.Error("EventTime should be set to current time")
	}
}

func TestRun(t *testing.T) {
	run := Run{
		RunID: "run-abc123",
		Facets: map[string]interface{}{
			"nominalTime": "2024-01-01T00:00:00Z",
		},
	}

	if run.RunID != "run-abc123" {
		t.Errorf("RunID = %s, want run-abc123", run.RunID)
	}
	if run.Facets["nominalTime"] != "2024-01-01T00:00:00Z" {
		t.Error("Facets should contain nominalTime")
	}
}

func TestJob(t *testing.T) {
	job := Job{
		Namespace: "data-team",
		Name:      "etl-pipeline",
		Facets: map[string]interface{}{
			"documentation": "ETL pipeline for data processing",
		},
	}

	if job.Namespace != "data-team" {
		t.Errorf("Namespace = %s, want data-team", job.Namespace)
	}
	if job.Name != "etl-pipeline" {
		t.Errorf("Name = %s, want etl-pipeline", job.Name)
	}
}

func TestDataset(t *testing.T) {
	dataset := Dataset{
		Namespace:   "s3://my-bucket",
		Name:        "data/events",
		Facets:      map[string]interface{}{"schema": "test"},
		InputFacets: map[string]interface{}{"rowCount": 1000},
	}

	if dataset.Namespace != "s3://my-bucket" {
		t.Errorf("Namespace = %s, want s3://my-bucket", dataset.Namespace)
	}
	if dataset.Name != "data/events" {
		t.Errorf("Name = %s, want data/events", dataset.Name)
	}
}

func TestEvent_WithInputsAndOutputs(t *testing.T) {
	event := NewEvent(EventTypeComplete, "run-123", "ns", "job")

	event.Inputs = []Dataset{
		{Namespace: "kafka://cluster", Name: "events-topic"},
	}
	event.Outputs = []Dataset{
		{Namespace: "s3://bucket", Name: "processed/data"},
	}

	if len(event.Inputs) != 1 {
		t.Errorf("Inputs count = %d, want 1", len(event.Inputs))
	}
	if len(event.Outputs) != 1 {
		t.Errorf("Outputs count = %d, want 1", len(event.Outputs))
	}
}

func TestSchemaVersion(t *testing.T) {
	if SchemaVersion == "" {
		t.Error("SchemaVersion should not be empty")
	}
	if SchemaVersion != "https://openlineage.io/spec/2-0-2/OpenLineage.json" {
		t.Errorf("SchemaVersion = %s, unexpected value", SchemaVersion)
	}
}
