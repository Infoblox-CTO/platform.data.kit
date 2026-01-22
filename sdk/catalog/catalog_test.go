package catalog

import (
	"context"
	"testing"
	"time"
)

func TestClientInterface(t *testing.T) {
	var _ Client = (*NoopClient)(nil)
	var _ Client = (*InMemoryClient)(nil)
}

func TestRecordType_Constants(t *testing.T) {
	tests := []struct {
		recordType RecordType
		want       string
	}{
		{RecordTypeDataset, "dataset"},
		{RecordTypeJob, "job"},
		{RecordTypeSource, "source"},
	}

	for _, tt := range tests {
		t.Run(string(tt.recordType), func(t *testing.T) {
			if string(tt.recordType) != tt.want {
				t.Errorf("RecordType = %s, want %s", tt.recordType, tt.want)
			}
		})
	}
}

func TestNoopClient(t *testing.T) {
	client := NewNoopClient()
	if client == nil {
		t.Fatal("NewNoopClient should not return nil")
	}

	ctx := context.Background()
	record := &Record{ID: "test-id", Namespace: "ns", Name: "test"}

	if err := client.CreateRecord(ctx, record); err != nil {
		t.Errorf("CreateRecord() error = %v", err)
	}
	if err := client.UpdateRecord(ctx, record); err != nil {
		t.Errorf("UpdateRecord() error = %v", err)
	}
	if _, err := client.GetRecord(ctx, "ns", "test"); err != nil {
		t.Errorf("GetRecord() error = %v", err)
	}
	if err := client.DeleteRecord(ctx, "ns", "test"); err != nil {
		t.Errorf("DeleteRecord() error = %v", err)
	}
	if _, err := client.ListRecords(ctx, "ns", ListOptions{}); err != nil {
		t.Errorf("ListRecords() error = %v", err)
	}
	if _, err := client.SearchRecords(ctx, SearchQuery{}); err != nil {
		t.Errorf("SearchRecords() error = %v", err)
	}
	if _, err := client.GetLineage(ctx, "ns", "test", 1); err != nil {
		t.Errorf("GetLineage() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestInMemoryClient(t *testing.T) {
	client := NewInMemoryClient()
	if client == nil {
		t.Fatal("NewInMemoryClient should not return nil")
	}

	ctx := context.Background()
	record := &Record{
		ID:        "ns/test",
		Type:      RecordTypeDataset,
		Namespace: "ns",
		Name:      "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := client.CreateRecord(ctx, record); err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}

	record.Description = "updated"
	if err := client.UpdateRecord(ctx, record); err != nil {
		t.Errorf("UpdateRecord() error = %v", err)
	}

	if err := client.DeleteRecord(ctx, "ns", "test"); err != nil {
		t.Errorf("DeleteRecord() error = %v", err)
	}
}

func TestRecord(t *testing.T) {
	now := time.Now()
	record := &Record{
		ID:             "data-team/events",
		Type:           RecordTypeDataset,
		Namespace:      "data-team",
		Name:           "events",
		Description:    "Event data",
		Tags:           []string{"events", "streaming"},
		Owner:          "data-team",
		Classification: "internal",
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       map[string]string{"environment": "production"},
	}

	if record.ID == "" {
		t.Error("ID should not be empty")
	}
	if record.Type != RecordTypeDataset {
		t.Errorf("Type = %s, want dataset", record.Type)
	}
	if len(record.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(record.Tags))
	}
}

func TestSchema(t *testing.T) {
	schema := &Schema{
		Fields: []Field{
			{Name: "id", Type: "string", Required: true},
			{Name: "value", Type: "int64", Required: false},
		},
		Format:  "json",
		Version: "1.0",
	}

	if len(schema.Fields) != 2 {
		t.Errorf("Fields count = %d, want 2", len(schema.Fields))
	}
	if schema.Format != "json" {
		t.Errorf("Format = %s, want json", schema.Format)
	}
}

func TestField(t *testing.T) {
	field := Field{
		Name:           "email",
		Type:           "string",
		Description:    "User email address",
		Required:       true,
		Classification: "pii",
	}

	if field.Name != "email" {
		t.Errorf("Name = %s, want email", field.Name)
	}
	if field.Classification != "pii" {
		t.Errorf("Classification = %s, want pii", field.Classification)
	}
}

func TestSource(t *testing.T) {
	source := &Source{
		Name:       "Kafka Events",
		Type:       "kafka",
		Connection: "kafka://cluster:9092",
		Location:   "events-topic",
	}

	if source.Type != "kafka" {
		t.Errorf("Type = %s, want kafka", source.Type)
	}
}

func TestListOptions(t *testing.T) {
	opts := ListOptions{
		Type:   RecordTypeDataset,
		Limit:  100,
		Offset: 0,
		Tags:   []string{"production"},
	}

	if opts.Limit != 100 {
		t.Errorf("Limit = %d, want 100", opts.Limit)
	}
}

func TestSearchQuery(t *testing.T) {
	query := SearchQuery{
		Text:           "events",
		Namespace:      "data-team",
		Type:           RecordTypeDataset,
		Tags:           []string{"streaming"},
		Classification: "internal",
		Owner:          "data-team",
		Limit:          50,
	}

	if query.Text != "events" {
		t.Errorf("Text = %s, want events", query.Text)
	}
	if query.Limit != 50 {
		t.Errorf("Limit = %d, want 50", query.Limit)
	}
}

func TestLineageGraph(t *testing.T) {
	graph := &LineageGraph{
		Root: &Record{ID: "root", Name: "root-dataset"},
		Nodes: []*LineageNode{
			{ID: "node1", Depth: 1, Direction: LineageUpstream},
		},
		Edges: []*LineageEdge{
			{Source: "node1", Target: "root", Type: "produces"},
		},
	}

	if graph.Root == nil {
		t.Error("Root should not be nil")
	}
	if len(graph.Nodes) != 1 {
		t.Errorf("Nodes count = %d, want 1", len(graph.Nodes))
	}
}

func TestLineageDirection(t *testing.T) {
	if LineageUpstream != "upstream" {
		t.Errorf("LineageUpstream = %s, want upstream", LineageUpstream)
	}
	if LineageDownstream != "downstream" {
		t.Errorf("LineageDownstream = %s, want downstream", LineageDownstream)
	}
}
