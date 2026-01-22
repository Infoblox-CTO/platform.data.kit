// Package catalog provides Marquez-based catalog client implementation.
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MarquezClient implements Client using Marquez as the backend.
type MarquezClient struct {
	client    *http.Client
	endpoint  string
	namespace string
}

// MarquezConfig contains configuration for the Marquez client.
type MarquezConfig struct {
	// Endpoint is the Marquez API endpoint.
	Endpoint string
	// Namespace is the default namespace.
	Namespace string
	// TimeoutSeconds is the HTTP timeout.
	TimeoutSeconds int
}

// DefaultMarquezConfig returns default configuration.
func DefaultMarquezConfig() MarquezConfig {
	return MarquezConfig{
		Endpoint:       "http://localhost:5000",
		Namespace:      "dp",
		TimeoutSeconds: 30,
	}
}

// NewMarquezClient creates a new Marquez-based catalog client.
func NewMarquezClient(config MarquezConfig) *MarquezClient {
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &MarquezClient{
		client: &http.Client{
			Timeout: timeout,
		},
		endpoint:  config.Endpoint,
		namespace: config.Namespace,
	}
}

// CreateRecord creates a record by creating a dataset in Marquez.
func (m *MarquezClient) CreateRecord(ctx context.Context, record *Record) error {
	// Marquez uses PUT for create/update
	return m.UpdateRecord(ctx, record)
}

// UpdateRecord updates a record in Marquez.
func (m *MarquezClient) UpdateRecord(ctx context.Context, record *Record) error {
	// For datasets, use the datasets API
	if record.Type == RecordTypeDataset {
		return m.createDataset(ctx, record)
	}
	// For jobs, use the jobs API
	if record.Type == RecordTypeJob {
		return m.createJob(ctx, record)
	}
	return fmt.Errorf("unsupported record type: %s", record.Type)
}

func (m *MarquezClient) createDataset(ctx context.Context, record *Record) error {
	namespace := record.Namespace
	if namespace == "" {
		namespace = m.namespace
	}

	// Build schema fields
	var fields []map[string]interface{}
	if record.Schema != nil {
		for _, f := range record.Schema.Fields {
			fields = append(fields, map[string]interface{}{
				"name":        f.Name,
				"type":        f.Type,
				"description": f.Description,
			})
		}
	}

	payload := map[string]interface{}{
		"type":         "DB_TABLE",
		"physicalName": record.Name,
		"description":  record.Description,
		"fields":       fields,
	}

	if record.Source != nil {
		payload["sourceName"] = record.Source.Name
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/datasets/%s",
		m.endpoint, namespace, record.Name)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(jsonReader(data))

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create dataset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (m *MarquezClient) createJob(ctx context.Context, record *Record) error {
	namespace := record.Namespace
	if namespace == "" {
		namespace = m.namespace
	}

	// Build inputs and outputs
	var inputs []map[string]string
	var outputs []map[string]string

	if record.Lineage != nil {
		for _, ref := range record.Lineage.Upstream {
			inputs = append(inputs, map[string]string{
				"namespace": ref.Namespace,
				"name":      ref.Name,
			})
		}
		for _, ref := range record.Lineage.Downstream {
			outputs = append(outputs, map[string]string{
				"namespace": ref.Namespace,
				"name":      ref.Name,
			})
		}
	}

	payload := map[string]interface{}{
		"type":        "BATCH",
		"description": record.Description,
		"inputs":      inputs,
		"outputs":     outputs,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/jobs/%s",
		m.endpoint, namespace, record.Name)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(jsonReader(data))

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetRecord retrieves a record from Marquez.
func (m *MarquezClient) GetRecord(ctx context.Context, namespace, name string) (*Record, error) {
	if namespace == "" {
		namespace = m.namespace
	}

	// Try datasets first
	record, err := m.getDataset(ctx, namespace, name)
	if err == nil && record != nil {
		return record, nil
	}

	// Try jobs
	return m.getJob(ctx, namespace, name)
}

func (m *MarquezClient) getDataset(ctx context.Context, namespace, name string) (*Record, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/datasets/%s",
		m.endpoint, namespace, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("marquez returned error %d", resp.StatusCode)
	}

	var dataset marquezDataset
	if err := json.NewDecoder(resp.Body).Decode(&dataset); err != nil {
		return nil, err
	}

	return datasetToRecord(&dataset), nil
}

func (m *MarquezClient) getJob(ctx context.Context, namespace, name string) (*Record, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/jobs/%s",
		m.endpoint, namespace, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("marquez returned error %d", resp.StatusCode)
	}

	var job marquezJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return jobToRecord(&job), nil
}

// DeleteRecord deletes a record from Marquez.
func (m *MarquezClient) DeleteRecord(ctx context.Context, namespace, name string) error {
	// Marquez doesn't support direct deletion, return nil
	return nil
}

// ListRecords lists records in a namespace.
func (m *MarquezClient) ListRecords(ctx context.Context, namespace string, opts ListOptions) ([]*Record, error) {
	if namespace == "" {
		namespace = m.namespace
	}

	var records []*Record

	// List datasets
	if opts.Type == "" || opts.Type == RecordTypeDataset {
		datasets, err := m.listDatasets(ctx, namespace, opts.Limit)
		if err != nil {
			return nil, err
		}
		records = append(records, datasets...)
	}

	// List jobs
	if opts.Type == "" || opts.Type == RecordTypeJob {
		jobs, err := m.listJobs(ctx, namespace, opts.Limit)
		if err != nil {
			return nil, err
		}
		records = append(records, jobs...)
	}

	return records, nil
}

func (m *MarquezClient) listDatasets(ctx context.Context, namespace string, limit int) ([]*Record, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/datasets", m.endpoint, namespace)
	if limit > 0 {
		url = fmt.Sprintf("%s?limit=%d", url, limit)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("marquez returned error %d", resp.StatusCode)
	}

	var result struct {
		Datasets []marquezDataset `json:"datasets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var records []*Record
	for _, d := range result.Datasets {
		records = append(records, datasetToRecord(&d))
	}
	return records, nil
}

func (m *MarquezClient) listJobs(ctx context.Context, namespace string, limit int) ([]*Record, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/jobs", m.endpoint, namespace)
	if limit > 0 {
		url = fmt.Sprintf("%s?limit=%d", url, limit)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("marquez returned error %d", resp.StatusCode)
	}

	var result struct {
		Jobs []marquezJob `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var records []*Record
	for _, j := range result.Jobs {
		records = append(records, jobToRecord(&j))
	}
	return records, nil
}

// SearchRecords searches for records.
func (m *MarquezClient) SearchRecords(ctx context.Context, query SearchQuery) ([]*Record, error) {
	url := fmt.Sprintf("%s/api/v1/search?q=%s", m.endpoint, query.Text)
	if query.Limit > 0 {
		url = fmt.Sprintf("%s&limit=%d", url, query.Limit)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Search may not be available, return empty
		return []*Record{}, nil
	}

	var result struct {
		Results []searchResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var records []*Record
	for _, r := range result.Results {
		records = append(records, &Record{
			ID:        r.Namespace + "/" + r.Name,
			Name:      r.Name,
			Namespace: r.Namespace,
			Type:      RecordType(r.Type),
		})
	}
	return records, nil
}

// GetLineage retrieves lineage from Marquez.
func (m *MarquezClient) GetLineage(ctx context.Context, namespace, name string, depth int) (*LineageGraph, error) {
	if namespace == "" {
		namespace = m.namespace
	}

	url := fmt.Sprintf("%s/api/v1/lineage?nodeId=dataset:%s:%s&depth=%d",
		m.endpoint, namespace, name, depth)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("marquez returned error %d", resp.StatusCode)
	}

	var result marquezLineage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return convertLineageGraph(&result), nil
}

// Close releases resources.
func (m *MarquezClient) Close() error {
	return nil
}

// Internal types for Marquez API

type marquezDataset struct {
	ID          datasetID      `json:"id"`
	Name        string         `json:"name"`
	Namespace   string         `json:"namespace"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	Description string         `json:"description"`
	Fields      []marquezField `json:"fields"`
	SourceName  string         `json:"sourceName"`
}

type datasetID struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type marquezField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type marquezJob struct {
	ID          jobID       `json:"id"`
	Name        string      `json:"name"`
	Namespace   string      `json:"namespace"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Description string      `json:"description"`
	Inputs      []datasetID `json:"inputs"`
	Outputs     []datasetID `json:"outputs"`
}

type jobID struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type searchResult struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
}

type marquezLineage struct {
	Graph []marquezNode `json:"graph"`
}

type marquezNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data"`
	InEdges  []marquezEdge          `json:"inEdges"`
	OutEdges []marquezEdge          `json:"outEdges"`
}

type marquezEdge struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
}

func datasetToRecord(d *marquezDataset) *Record {
	record := &Record{
		ID:          d.Namespace + "/" + d.Name,
		Name:        d.Name,
		Namespace:   d.Namespace,
		Type:        RecordTypeDataset,
		Description: d.Description,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}

	if len(d.Fields) > 0 {
		record.Schema = &Schema{
			Fields: make([]Field, len(d.Fields)),
		}
		for i, f := range d.Fields {
			record.Schema.Fields[i] = Field{
				Name:        f.Name,
				Type:        f.Type,
				Description: f.Description,
			}
		}
	}

	if d.SourceName != "" {
		record.Source = &Source{Name: d.SourceName}
	}

	return record
}

func jobToRecord(j *marquezJob) *Record {
	record := &Record{
		ID:          j.Namespace + "/" + j.Name,
		Name:        j.Name,
		Namespace:   j.Namespace,
		Type:        RecordTypeJob,
		Description: j.Description,
		CreatedAt:   j.CreatedAt,
		UpdatedAt:   j.UpdatedAt,
	}

	if len(j.Inputs) > 0 || len(j.Outputs) > 0 {
		record.Lineage = &Lineage{}
		for _, input := range j.Inputs {
			record.Lineage.Upstream = append(record.Lineage.Upstream, Reference{
				Namespace: input.Namespace,
				Name:      input.Name,
			})
		}
		for _, output := range j.Outputs {
			record.Lineage.Downstream = append(record.Lineage.Downstream, Reference{
				Namespace: output.Namespace,
				Name:      output.Name,
			})
		}
	}

	return record
}

func convertLineageGraph(ml *marquezLineage) *LineageGraph {
	graph := &LineageGraph{
		Nodes: make([]*LineageNode, 0, len(ml.Graph)),
		Edges: make([]*LineageEdge, 0),
	}

	for i, node := range ml.Graph {
		graph.Nodes = append(graph.Nodes, &LineageNode{
			ID:    node.ID,
			Depth: i,
		})

		for _, edge := range node.OutEdges {
			graph.Edges = append(graph.Edges, &LineageEdge{
				Source: edge.Origin,
				Target: edge.Destination,
			})
		}
	}

	return graph
}

// jsonReader creates an io.Reader from JSON bytes
type jsonBytesReader struct {
	data   []byte
	offset int
}

func jsonReader(data []byte) *jsonBytesReader {
	return &jsonBytesReader{data: data}
}

func (r *jsonBytesReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}
