# Chart Definition Contract

**Feature**: 013-helm-dev-deps | **Date**: 2026-02-15

## ChartDefinition Interface

Every dev dependency chart MUST be representable by this contract. The deployment orchestration code operates exclusively on this interface — no service-specific code paths.

```go
// ChartDefinition describes a single dev dependency Helm chart.
type ChartDefinition struct {
    // Name is the unique identifier (matches chart directory name under sdk/localdev/charts/).
    Name string

    // ReleaseName is the Helm release name used with helm upgrade --install.
    // Convention: "dp-" + Name.
    ReleaseName string

    // Namespace is the Kubernetes namespace for deployment.
    Namespace string

    // PortForwards defines the port-forwarding rules for this chart's services.
    PortForwards []PortForward

    // HealthLabels are the pod label selectors used to check health.
    HealthLabels map[string]string

    // HealthTimeout is the maximum duration to wait for pods to become healthy.
    HealthTimeout time.Duration

    // DisplayEndpoints are the human-readable endpoints shown in status output.
    DisplayEndpoints []DisplayEndpoint
}

// PortForward defines a single port-forwarding rule.
type PortForward struct {
    // ServiceName is the Kubernetes service to port-forward to.
    ServiceName string

    // LocalPort is the port on localhost.
    LocalPort int

    // RemotePort is the port on the Kubernetes service.
    RemotePort int
}

// DisplayEndpoint is a human-readable endpoint for status display.
type DisplayEndpoint struct {
    // Label is the display name (e.g., "Kafka", "S3 API").
    Label string

    // URL is the localhost URL (e.g., "localhost:19092").
    URL string
}
```

## ChartOverride Config Contract

```go
// ChartOverride holds user-configurable overrides for a chart.
type ChartOverride struct {
    // Version overrides the embedded chart version.
    // Empty string means use the embedded default.
    Version string `yaml:"version,omitempty"`

    // Values are additional Helm --set values merged at deploy time.
    Values map[string]interface{} `yaml:"values,omitempty"`
}
```

## Config Extension Contract

```go
// DevConfig is the dev section of the hierarchical config.
type DevConfig struct {
    Runtime   string                    `yaml:"runtime"`
    Workspace string                   `yaml:"workspace"`
    K3d       K3dConfig                `yaml:"k3d"`
    Charts    map[string]ChartOverride `yaml:"charts,omitempty"` // NEW
}
```

Config paths:
- `dev.charts.redpanda.version` → `ChartOverride.Version` for redpanda
- `dev.charts.postgres.values.primary.persistence.enabled` → `ChartOverride.Values["primary.persistence.enabled"]`

## Chart Registry Contract

The default chart registry is defined as a Go variable:

```go
var DefaultCharts = []ChartDefinition{
    {
        Name:        "redpanda",
        ReleaseName: "dp-redpanda",
        Namespace:   "dp-local",
        PortForwards: []PortForward{
            {ServiceName: "dp-redpanda", LocalPort: 19092, RemotePort: 9092},
            {ServiceName: "dp-redpanda-console", LocalPort: 8080, RemotePort: 8080},
            {ServiceName: "dp-redpanda", LocalPort: 18081, RemotePort: 8081},
        },
        HealthLabels:  map[string]string{"app.kubernetes.io/name": "redpanda"},
        HealthTimeout: 120 * time.Second,
        DisplayEndpoints: []DisplayEndpoint{
            {Label: "Kafka", URL: "localhost:19092"},
            {Label: "Schema Registry", URL: "http://localhost:18081"},
            {Label: "Redpanda Console", URL: "http://localhost:8080"},
        },
    },
    {
        Name:        "localstack",
        ReleaseName: "dp-localstack",
        Namespace:   "dp-local",
        PortForwards: []PortForward{
            {ServiceName: "dp-localstack", LocalPort: 4566, RemotePort: 4566},
        },
        HealthLabels:  map[string]string{"app": "localstack"},
        HealthTimeout: 60 * time.Second,
        DisplayEndpoints: []DisplayEndpoint{
            {Label: "S3 API", URL: "http://localhost:4566"},
        },
    },
    {
        Name:        "postgres",
        ReleaseName: "dp-postgres",
        Namespace:   "dp-local",
        PortForwards: []PortForward{
            {ServiceName: "dp-postgres-postgresql", LocalPort: 5432, RemotePort: 5432},
        },
        HealthLabels:  map[string]string{"app.kubernetes.io/name": "postgresql"},
        HealthTimeout: 60 * time.Second,
        DisplayEndpoints: []DisplayEndpoint{
            {Label: "PostgreSQL", URL: "localhost:5432"},
        },
    },
    {
        Name:        "marquez",
        ReleaseName: "dp-marquez",
        Namespace:   "dp-local",
        PortForwards: []PortForward{
            {ServiceName: "dp-marquez", LocalPort: 5000, RemotePort: 5000},
            {ServiceName: "dp-marquez", LocalPort: 5001, RemotePort: 5001},
            {ServiceName: "dp-marquez-web", LocalPort: 3000, RemotePort: 3000},
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
```

## Deployment Function Contract

```go
// DeployCharts deploys all charts using a uniform mechanism.
// It extracts embedded charts, applies config overrides, and runs
// helm upgrade --install in parallel.
// Returns a DeployResult with per-chart success/failure status.
func DeployCharts(ctx context.Context, charts []ChartDefinition, overrides map[string]ChartOverride, kubeContext string) (*DeployResult, error)

// DeployResult reports per-chart deployment outcome.
type DeployResult struct {
    Succeeded []string  // chart names that deployed successfully
    Failed    []ChartError // chart names that failed with reasons
}

type ChartError struct {
    ChartName string
    Error     error
}
```

## Helm Chart Directory Contract

Each chart directory under `sdk/localdev/charts/<name>/` MUST contain:

```
<name>/
├── Chart.yaml          # REQUIRED: name, version, appVersion, dependencies (if subchart)
├── Chart.lock          # REQUIRED if Chart.yaml has dependencies
├── values.yaml         # REQUIRED: dev-appropriate defaults
├── charts/             # REQUIRED if subchart: contains .tgz archives
│   └── <dep>.tgz      # Pre-built subchart archive (committed to Git)
└── templates/          # REQUIRED: Kubernetes resource templates
    ├── deployment.yaml # OR delegated to subchart
    ├── service.yaml    # OR delegated to subchart
    └── init-job.yaml   # OPTIONAL: Helm post-install hook for resource seeding
```
