# DP - Data Platform CLI

[![Build Status](https://github.com/Infoblox-CTO/platform.data.kit/actions/workflows/ci.yaml/badge.svg)](https://github.com/Infoblox-CTO/platform.data.kit/actions)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go)](https://go.dev)
[![Coverage](https://img.shields.io/badge/coverage-70%25+-green)](https://github.com/Infoblox-CTO/platform.data.kit/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-datakit-blue)](https://datakit.internal.infoblox.dev/)

A Kubernetes-native data pipeline platform enabling teams to contribute reusable, versioned "data packages" with a complete developer workflow: **bootstrap → local run → validate → publish → promote**.

> 📚 **[View Full Documentation](https://datakit.internal.infoblox.dev/)** for detailed guides, tutorials, and reference.

## 🚀 Quick Start

### Prerequisites

- Go 1.22+
- Docker Desktop or Docker Engine
- kubectl (for Kubernetes operations)

**For k3d runtime (optional, for Kubernetes-native local development):**

- k3d (v5.0+) - Install with `brew install k3d` (macOS) or [install script](https://k3d.io/v5.6.0/#installation)

### Installation

```bash
# Clone the repository
git clone https://github.com/Infoblox-CTO/platform.data.kit.git
cd data-platform

# Build the CLI
make build

# Add to PATH
export PATH=$PATH:$(pwd)/bin

# Verify installation
dp version
```

### Your First Pipeline (30 minutes)

```bash
# 1. Create a new data package
dp init my-pipeline --kind model --runtime generic-python

# 2. Start the local development stack
dp dev up

# 3. Validate your manifest
dp lint ./my-pipeline

# 4. Run locally
dp run ./my-pipeline

# 5. Build and publish
dp build ./my-pipeline
dp publish ./my-pipeline

# 6. Promote to an environment
dp promote my-pipeline v0.1.0 --to dev
```

## 📦 What is a Data Package?

A data package is a self-contained unit of data processing that includes:

- **Manifest** (`dp.yaml`): Metadata, inputs, outputs, and classification
- **Pipeline** (`pipeline.yaml`): Runtime configuration and execution details
- **Bindings** (`bindings.yaml`): Environment-specific infrastructure mappings
- **Code**: Your data processing logic (Python, Spark, etc.)

```yaml
# dp.yaml example
apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: kafka-s3-pipeline
  namespace: analytics
spec:
  type: pipeline
  description: Processes events from Kafka to S3
  owner: data-team
  
  inputs:
    - name: events
      type: kafka-topic
      binding: input.events
      
  outputs:
    - name: processed-events
      type: s3-prefix
      binding: output.lake
      classification:
        pii: true
        sensitivity: confidential
```

## 🛠️ CLI Commands

| Command | Description |
|---------|-------------|
| `dp init` | Create a new data package |
| `dp dev up/down/status` | Manage local development stack |
| `dp lint` | Validate package manifests |
| `dp run` | Execute pipeline locally |
| `dp test` | Run tests with sample data |
| `dp build` | Build OCI artifact |
| `dp publish` | Publish to OCI registry |
| `dp promote` | Promote to environment via GitOps |
| `dp status` | Show package status across environments |
| `dp logs` | Stream logs from running pipeline |
| `dp rollback` | Rollback to previous version |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Developer                            │
│                            │                                 │
│                            ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   DP CLI (dp)                        │   │
│  │  init, dev, run, lint, build, publish, promote      │   │
│  └─────────────────────────────────────────────────────┘   │
│                            │                                 │
│              ┌─────────────┴──────────────┐                 │
│              ▼                            ▼                 │
│  ┌────────────────────┐      ┌────────────────────┐        │
│  │       SDK          │      │    OCI Registry    │        │
│  │  Validation        │      │  Immutable Pkgs    │        │
│  │  Lineage Emit      │      │                    │        │
│  └────────────────────┘      └────────────────────┘        │
│                                          │                  │
│                                          ▼                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │               GitOps (Kustomize + ArgoCD)            │   │
│  │   environments/dev  │  int  │  prod                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                            │                                 │
│                            ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │            Kubernetes Platform Controller            │   │
│  │          PackageDeployment CRD Reconciler           │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 📁 Repository Structure

```
data-platform/
├── contracts/          # Shared types and schemas
├── sdk/                # Core SDK functionality
│   ├── validate/       # Manifest validation
│   ├── lineage/        # OpenLineage integration
│   ├── registry/       # OCI registry client
│   ├── runner/         # Local execution runtime
│   └── catalog/        # Data catalog client
├── cli/                # DP CLI implementation
├── demos/              # Scripted terminal demos (see demos/README.md)
├── platform/
│   └── controller/     # Kubernetes controller
├── gitops/             # Environment definitions
├── examples/           # Reference packages
├── hack/               # Development utilities
│   └── compose/        # Local dev stack
└── dashboards/         # Grafana dashboards
```

## 🔒 Data Governance

### PII Classification

All data package outputs must declare their classification:

```yaml
outputs:
  - name: user-profiles
    classification:
      pii: true
      sensitivity: confidential
      dataCategory: customer-data
      retentionDays: 365
```

Validate with: `dp lint --strict`

### Data Lineage

DP automatically tracks lineage via OpenLineage:

- **START** event when pipeline begins
- **COMPLETE** event on success
- **FAIL** event with error details on failure

View lineage in Marquez UI: http://localhost:5000

## 🧪 Local Development

Start the full local stack:

```bash
# Start all services (Kafka, S3, PostgreSQL, Marquez)
dp dev up

# Check status
dp dev status

# View Marquez lineage UI
open http://localhost:5000

# Stop when done
dp dev down
```

## 📊 Observability

- **Metrics**: Prometheus-compatible metrics exposed by CLI and controller
- **Logs**: Structured JSON logging with correlation IDs
- **Dashboards**: Pre-built Grafana dashboards in `dashboards/`

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

Apache 2.0 - See [LICENSE](LICENSE) for details.

---

Built with ❤️ by the Data Platform Team
