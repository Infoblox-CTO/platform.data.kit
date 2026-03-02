// Package lineage provides the Marquez HTTP emitter implementation.
package lineage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MarquezEmitter sends lineage events to a Marquez server.
type MarquezEmitter struct {
	client    *http.Client
	endpoint  string
	namespace string
	apiKey    string
}

// MarquezConfig contains configuration for the Marquez emitter.
type MarquezConfig struct {
	// Endpoint is the Marquez API endpoint (e.g., "http://localhost:5000").
	Endpoint string
	// Namespace is the default namespace for jobs and datasets.
	Namespace string
	// APIKey is an optional API key for authentication.
	APIKey string
	// TimeoutSeconds is the HTTP request timeout.
	TimeoutSeconds int
}

// DefaultMarquezConfig returns default Marquez configuration for local development.
func DefaultMarquezConfig() MarquezConfig {
	return MarquezConfig{
		Endpoint:       "http://localhost:5000",
		Namespace:      "dk",
		TimeoutSeconds: 30,
	}
}

// NewMarquezEmitter creates a new Marquez emitter.
func NewMarquezEmitter(config MarquezConfig) *MarquezEmitter {
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &MarquezEmitter{
		client: &http.Client{
			Timeout: timeout,
		},
		endpoint:  config.Endpoint,
		namespace: config.Namespace,
		apiKey:    config.APIKey,
	}
}

// Emit sends a lineage event to Marquez.
func (m *MarquezEmitter) Emit(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Ensure namespace is set
	if event.Job.Namespace == "" {
		event.Job.Namespace = m.namespace
	}

	// Serialize event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/api/v1/lineage", m.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	// Send request
	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close releases resources (no-op for HTTP client).
func (m *MarquezEmitter) Close() error {
	return nil
}

// GetNamespaces retrieves all namespaces from Marquez.
func (m *MarquezEmitter) GetNamespaces(ctx context.Context) ([]Namespace, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces", m.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Namespaces []Namespace `json:"namespaces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Namespaces, nil
}

// Namespace represents a Marquez namespace.
type Namespace struct {
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	OwnerName   string    `json:"ownerName"`
	Description string    `json:"description,omitempty"`
}

// GetJobs retrieves jobs from a namespace.
func (m *MarquezEmitter) GetJobs(ctx context.Context, namespace string) ([]JobInfo, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/jobs", m.endpoint, namespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Jobs []JobInfo `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Jobs, nil
}

// JobInfo represents job information from Marquez.
type JobInfo struct {
	ID          JobID     `json:"id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Namespace   string    `json:"namespace"`
	Description string    `json:"description,omitempty"`
	LatestRun   *RunInfo  `json:"latestRun,omitempty"`
}

// JobID represents a job identifier.
type JobID struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// RunInfo represents run information from Marquez.
type RunInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	State     string    `json:"state"`
	StartedAt time.Time `json:"startedAt,omitempty"`
	EndedAt   time.Time `json:"endedAt,omitempty"`
}

// GetDatasets retrieves datasets from a namespace.
func (m *MarquezEmitter) GetDatasets(ctx context.Context, namespace string) ([]DatasetInfo, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/datasets", m.endpoint, namespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get datasets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Datasets []DatasetInfo `json:"datasets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Datasets, nil
}

// DatasetInfo represents dataset information from Marquez.
type DatasetInfo struct {
	ID          DatasetID `json:"id"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Namespace   string    `json:"namespace"`
	SourceName  string    `json:"sourceName,omitempty"`
	Description string    `json:"description,omitempty"`
}

// DatasetID represents a dataset identifier.
type DatasetID struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// GetLineageGraph retrieves the lineage graph for a node.
func (m *MarquezEmitter) GetLineageGraph(ctx context.Context, nodeType, namespace, name string, depth int) (*LineageGraph, error) {
	url := fmt.Sprintf("%s/api/v1/lineage?nodeId=%s:%s:%s&depth=%d",
		m.endpoint, nodeType, namespace, name, depth)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get lineage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	var graph LineageGraph
	if err := json.NewDecoder(resp.Body).Decode(&graph); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &graph, nil
}

// LineageGraph represents a lineage graph from Marquez.
type LineageGraph struct {
	Graph []LineageNode `json:"graph"`
}

// LineageNode represents a node in the lineage graph.
type LineageNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Data     map[string]interface{} `json:"data"`
	InEdges  []LineageEdge          `json:"inEdges"`
	OutEdges []LineageEdge          `json:"outEdges"`
}

// LineageEdge represents an edge in the lineage graph.
type LineageEdge struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
}

// CreateNamespace creates a namespace in Marquez.
func (m *MarquezEmitter) CreateNamespace(ctx context.Context, name, ownerName, description string) error {
	payload := map[string]string{
		"ownerName":   ownerName,
		"description": description,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s", m.endpoint, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marquez returned error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Health checks if Marquez is healthy.
func (m *MarquezEmitter) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/namespaces", m.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("marquez is not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("marquez returned status %d", resp.StatusCode)
	}

	return nil
}
