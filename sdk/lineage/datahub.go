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

// DataHubEmitter sends OpenLineage events to a DataHub GMS endpoint.
type DataHubEmitter struct {
	client    *http.Client
	endpoint  string
	namespace string
	apiToken  string
}

// DataHubConfig contains configuration for the DataHub emitter.
type DataHubConfig struct {
	// Endpoint is the DataHub GMS API endpoint (e.g., "http://datahub-gms:8080").
	Endpoint string
	// Namespace is the default namespace for jobs and datasets.
	Namespace string
	// APIToken is an optional API token for authentication.
	APIToken string
	// TimeoutSeconds is the HTTP request timeout.
	TimeoutSeconds int
}

// DefaultDataHubConfig returns default DataHub configuration.
func DefaultDataHubConfig() DataHubConfig {
	return DataHubConfig{
		Namespace:      "dk",
		TimeoutSeconds: 30,
	}
}

// NewDataHubEmitter creates a new DataHub emitter.
func NewDataHubEmitter(config DataHubConfig) (*DataHubEmitter, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("datahub endpoint is required")
	}

	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &DataHubEmitter{
		client: &http.Client{
			Timeout: timeout,
		},
		endpoint:  config.Endpoint,
		namespace: config.Namespace,
		apiToken:  config.APIToken,
	}, nil
}

// Emit sends a lineage event to DataHub's OpenLineage endpoint.
func (d *DataHubEmitter) Emit(ctx context.Context, event *Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Ensure namespace is set
	if event.Job.Namespace == "" {
		event.Job.Namespace = d.namespace
	}

	// Serialize event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// DataHub accepts OpenLineage events at /openapi/v1/lineage
	url := fmt.Sprintf("%s/openapi/v1/lineage", d.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if d.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+d.apiToken)
	}

	// Send request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event to datahub: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("datahub returned error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close releases resources (no-op for HTTP client).
func (d *DataHubEmitter) Close() error {
	return nil
}
