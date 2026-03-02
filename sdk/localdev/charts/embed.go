package charts

import (
	"embed"
	"time"
)

// FS contains the embedded Helm charts for local development services.
//
//go:embed all:redpanda all:localstack all:postgres all:marquez
var FS embed.FS

// ChartNames is the list of available charts.
var ChartNames = []string{"redpanda", "localstack", "postgres", "marquez"}

// DefaultCharts is the canonical registry of all dev dependency chart definitions.
// All deployment, port-forwarding, health-checking, and status code operates on this slice.
var DefaultCharts = []ChartDefinition{
	{
		Name:        "redpanda",
		ReleaseName: "dk-redpanda",
		Namespace:   "dk-local",
		PortForwards: []PortForwardRule{
			{ServiceName: "dk-redpanda", LocalPort: 19092, RemotePort: 9092},
			{ServiceName: "dk-redpanda", LocalPort: 18081, RemotePort: 8081},
		},
		HealthLabels:  map[string]string{"app.kubernetes.io/name": "redpanda"},
		HealthTimeout: 120 * time.Second,
		DisplayEndpoints: []DisplayEndpoint{
			{Label: "Kafka", URL: "localhost:19092"},
			{Label: "Schema Registry", URL: "http://localhost:18081"},
		},
	},
	{
		Name:        "localstack",
		ReleaseName: "dk-localstack",
		Namespace:   "dk-local",
		PortForwards: []PortForwardRule{
			{ServiceName: "dk-localstack", LocalPort: 4566, RemotePort: 4566},
		},
		HealthLabels:  map[string]string{"app": "localstack"},
		HealthTimeout: 60 * time.Second,
		DisplayEndpoints: []DisplayEndpoint{
			{Label: "S3 API", URL: "http://localhost:4566"},
		},
	},
	{
		Name:        "postgres",
		ReleaseName: "dk-postgres",
		Namespace:   "dk-local",
		PortForwards: []PortForwardRule{
			{ServiceName: "dk-postgres-postgresql", LocalPort: 5432, RemotePort: 5432},
		},
		HealthLabels:  map[string]string{"app.kubernetes.io/name": "postgresql"},
		HealthTimeout: 60 * time.Second,
		DisplayEndpoints: []DisplayEndpoint{
			{Label: "PostgreSQL", URL: "localhost:5432"},
		},
	},
	{
		Name:        "marquez",
		ReleaseName: "dk-marquez",
		Namespace:   "dk-local",
		PortForwards: []PortForwardRule{
			{ServiceName: "dk-marquez", LocalPort: 5000, RemotePort: 5000},
			{ServiceName: "dk-marquez", LocalPort: 5001, RemotePort: 5001},
			{ServiceName: "dk-marquez-web", LocalPort: 3000, RemotePort: 3000},
		},
		HealthLabels:  map[string]string{"app": "marquez"},
		HealthTimeout: 90 * time.Second,
		DisplayEndpoints: []DisplayEndpoint{
			{Label: "Marquez API", URL: "http://localhost:5000"},
			{Label: "Marquez Admin", URL: "http://localhost:5001"},
			{Label: "Marquez Web", URL: "http://localhost:3000"},
		},
	},
}
