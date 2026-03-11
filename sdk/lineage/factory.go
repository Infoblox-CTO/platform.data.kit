package lineage

import "fmt"

// NewEmitterFromConfig creates an Emitter based on the provided configuration.
// Supported types: "", "noop", "console", "marquez", "datahub", "http".
func NewEmitterFromConfig(config EmitterConfig) (Emitter, error) {
	switch config.Type {
	case "", "noop":
		return NewNoopEmitter(), nil

	case "console":
		return NewConsoleEmitter(nil), nil

	case "marquez":
		endpoint := config.Endpoint
		if endpoint == "" {
			endpoint = "http://localhost:5000"
		}
		return NewMarquezEmitter(MarquezConfig{
			Endpoint:       endpoint,
			Namespace:      config.Namespace,
			APIKey:         config.APIKey,
			TimeoutSeconds: config.TimeoutSeconds,
		}), nil

	case "datahub":
		if config.Endpoint == "" {
			return nil, fmt.Errorf("datahub emitter requires an endpoint")
		}
		emitter, err := NewDataHubEmitter(DataHubConfig{
			Endpoint:       config.Endpoint,
			Namespace:      config.Namespace,
			APIToken:       config.APIKey,
			TimeoutSeconds: config.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create datahub emitter: %w", err)
		}
		return emitter, nil

	case "http":
		// Generic OpenLineage HTTP transport — uses MarquezEmitter since
		// Marquez speaks standard OpenLineage at /api/v1/lineage.
		endpoint := config.Endpoint
		if endpoint == "" {
			return nil, fmt.Errorf("http emitter requires an endpoint")
		}
		return NewMarquezEmitter(MarquezConfig{
			Endpoint:       endpoint,
			Namespace:      config.Namespace,
			APIKey:         config.APIKey,
			TimeoutSeconds: config.TimeoutSeconds,
		}), nil

	default:
		return nil, fmt.Errorf("unknown emitter type: %q", config.Type)
	}
}
