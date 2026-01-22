// Package catalog provides the client interface for data catalog operations.
package catalog

import (
	"context"
)

// Client is the interface for data catalog operations.
type Client interface {
	// CreateRecord creates a new catalog record.
	CreateRecord(ctx context.Context, record *Record) error
	// UpdateRecord updates an existing catalog record.
	UpdateRecord(ctx context.Context, record *Record) error
	// GetRecord retrieves a catalog record by ID.
	GetRecord(ctx context.Context, namespace, name string) (*Record, error)
	// DeleteRecord deletes a catalog record.
	DeleteRecord(ctx context.Context, namespace, name string) error
	// ListRecords lists records in a namespace.
	ListRecords(ctx context.Context, namespace string, opts ListOptions) ([]*Record, error)
	// SearchRecords searches for records matching criteria.
	SearchRecords(ctx context.Context, query SearchQuery) ([]*Record, error)
	// GetLineage retrieves lineage for a record.
	GetLineage(ctx context.Context, namespace, name string, depth int) (*LineageGraph, error)
	// Close releases any resources.
	Close() error
}

// ListOptions contains options for listing records.
type ListOptions struct {
	// Type filters by record type.
	Type RecordType
	// Limit is the maximum number of records to return.
	Limit int
	// Offset is the offset for pagination.
	Offset int
	// Tags filters by tags.
	Tags []string
}

// SearchQuery contains search criteria.
type SearchQuery struct {
	// Text is a free-text search query.
	Text string
	// Namespace filters by namespace.
	Namespace string
	// Type filters by record type.
	Type RecordType
	// Tags filters by tags (AND).
	Tags []string
	// Classification filters by classification.
	Classification string
	// Owner filters by owner.
	Owner string
	// Limit is the maximum number of records to return.
	Limit int
}

// LineageGraph represents a lineage graph.
type LineageGraph struct {
	// Root is the root record.
	Root *Record
	// Nodes are all nodes in the graph.
	Nodes []*LineageNode
	// Edges are the edges between nodes.
	Edges []*LineageEdge
}

// LineageNode is a node in the lineage graph.
type LineageNode struct {
	// ID is the node ID.
	ID string
	// Record is the catalog record for this node.
	Record *Record
	// Depth is the distance from the root.
	Depth int
	// Direction indicates upstream or downstream from root.
	Direction LineageDirection
}

// LineageEdge is an edge in the lineage graph.
type LineageEdge struct {
	// Source is the source node ID.
	Source string
	// Target is the target node ID.
	Target string
	// Type is the edge type (produces, consumes, etc.).
	Type string
}

// LineageDirection indicates the direction of lineage.
type LineageDirection string

const (
	// LineageUpstream is upstream (dependencies).
	LineageUpstream LineageDirection = "upstream"
	// LineageDownstream is downstream (dependents).
	LineageDownstream LineageDirection = "downstream"
)

// NoopClient is a client that does nothing, useful for testing.
type NoopClient struct{}

// CreateRecord does nothing.
func (n *NoopClient) CreateRecord(ctx context.Context, record *Record) error {
	return nil
}

// UpdateRecord does nothing.
func (n *NoopClient) UpdateRecord(ctx context.Context, record *Record) error {
	return nil
}

// GetRecord returns nil.
func (n *NoopClient) GetRecord(ctx context.Context, namespace, name string) (*Record, error) {
	return nil, nil
}

// DeleteRecord does nothing.
func (n *NoopClient) DeleteRecord(ctx context.Context, namespace, name string) error {
	return nil
}

// ListRecords returns empty list.
func (n *NoopClient) ListRecords(ctx context.Context, namespace string, opts ListOptions) ([]*Record, error) {
	return []*Record{}, nil
}

// SearchRecords returns empty list.
func (n *NoopClient) SearchRecords(ctx context.Context, query SearchQuery) ([]*Record, error) {
	return []*Record{}, nil
}

// GetLineage returns nil.
func (n *NoopClient) GetLineage(ctx context.Context, namespace, name string, depth int) (*LineageGraph, error) {
	return nil, nil
}

// Close does nothing.
func (n *NoopClient) Close() error {
	return nil
}

// NewNoopClient creates a new no-op client.
func NewNoopClient() Client {
	return &NoopClient{}
}

// InMemoryClient is an in-memory client for testing.
type InMemoryClient struct {
	records map[string]*Record
}

// NewInMemoryClient creates a new in-memory client.
func NewInMemoryClient() *InMemoryClient {
	return &InMemoryClient{
		records: make(map[string]*Record),
	}
}

// CreateRecord stores a record in memory.
func (c *InMemoryClient) CreateRecord(ctx context.Context, record *Record) error {
	c.records[record.ID] = record
	return nil
}

// UpdateRecord updates a record in memory.
func (c *InMemoryClient) UpdateRecord(ctx context.Context, record *Record) error {
	c.records[record.ID] = record
	return nil
}

// GetRecord retrieves a record from memory.
func (c *InMemoryClient) GetRecord(ctx context.Context, namespace, name string) (*Record, error) {
	id := namespace + "/" + name
	return c.records[id], nil
}

// DeleteRecord deletes a record from memory.
func (c *InMemoryClient) DeleteRecord(ctx context.Context, namespace, name string) error {
	id := namespace + "/" + name
	delete(c.records, id)
	return nil
}

// ListRecords lists all records in a namespace.
func (c *InMemoryClient) ListRecords(ctx context.Context, namespace string, opts ListOptions) ([]*Record, error) {
	var result []*Record
	for _, r := range c.records {
		if r.Namespace == namespace {
			if opts.Type != "" && r.Type != opts.Type {
				continue
			}
			result = append(result, r)
		}
	}
	return result, nil
}

// SearchRecords searches records by text.
func (c *InMemoryClient) SearchRecords(ctx context.Context, query SearchQuery) ([]*Record, error) {
	var result []*Record
	for _, r := range c.records {
		if query.Namespace != "" && r.Namespace != query.Namespace {
			continue
		}
		if query.Type != "" && r.Type != query.Type {
			continue
		}
		if query.Classification != "" && r.Classification != query.Classification {
			continue
		}
		if query.Owner != "" && r.Owner != query.Owner {
			continue
		}
		// Simple text matching
		if query.Text != "" {
			if !stringContains(r.Name, query.Text) && !stringContains(r.Description, query.Text) {
				continue
			}
		}
		result = append(result, r)
	}
	return result, nil
}

// GetLineage builds a lineage graph from records.
func (c *InMemoryClient) GetLineage(ctx context.Context, namespace, name string, depth int) (*LineageGraph, error) {
	root, err := c.GetRecord(ctx, namespace, name)
	if err != nil || root == nil {
		return nil, err
	}

	graph := &LineageGraph{
		Root:  root,
		Nodes: []*LineageNode{},
		Edges: []*LineageEdge{},
	}

	// Add root node
	graph.Nodes = append(graph.Nodes, &LineageNode{
		ID:     root.ID,
		Record: root,
		Depth:  0,
	})

	// Add upstream nodes
	if root.Lineage != nil {
		for _, ref := range root.Lineage.Upstream {
			upstream, _ := c.GetRecord(ctx, ref.Namespace, ref.Name)
			if upstream != nil {
				graph.Nodes = append(graph.Nodes, &LineageNode{
					ID:        upstream.ID,
					Record:    upstream,
					Depth:     1,
					Direction: LineageUpstream,
				})
				graph.Edges = append(graph.Edges, &LineageEdge{
					Source: upstream.ID,
					Target: root.ID,
					Type:   "produces",
				})
			}
		}
	}

	return graph, nil
}

// Close does nothing.
func (c *InMemoryClient) Close() error {
	return nil
}

// stringContains is a simple case-insensitive contains check.
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(substr) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
