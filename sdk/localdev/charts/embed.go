package charts

import "embed"

// FS contains the embedded Helm charts for local development services.
//
//go:embed redpanda localstack postgres
var FS embed.FS

// ChartNames is the list of available charts.
var ChartNames = []string{"redpanda", "localstack", "postgres"}
