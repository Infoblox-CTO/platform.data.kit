# DK - DataKit CLI

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
cd platform.data.kit

# Build the CLI
make build

# Add to PATH
export PATH=$PATH:$(pwd)/bin

# Verify installation
dk version
```

### Your First Pipeline (30 minutes)

```bash
# 1. Create a new data package
dk init my-pipeline --runtime generic-python

# 2. Start the local development stack
dk dev up

# 3. Validate your manifest
dk lint ./my-pipeline

# 4. Run locally
dk run ./my-pipeline

# 5. Build and publish
dk build ./my-pipeline
dk publish ./my-pipeline

# 6. Promote to an environment
dk promote my-pipeline v0.1.0 --to dev
```

## 📦 What is a Data Package?

A data package is a self-contained unit of data processing that includes:

- **Manifest** (`dk.yaml`): Transform definition with runtime, inputs, outputs, and schedule
- **Connectors & Stores**: Infrastructure connection definitions
- **Assets**: Data contracts with schema and lineage
- **Code**: Your data processing logic (Python, Go, etc.)

```yaml
# dk.yaml example
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: kafka-s3-pipeline
  namespace: analytics
  version: 1.0.0
spec:
  runtime: generic-go
  mode: batch
  image: "myimage:v1"
  inputs:
    - asset: kafka-events
  outputs:
    - asset: processed-events
```

## 🛠️ CLI Commands

| Command                   | Description                             |
| ------------------------- | --------------------------------------- |
| `dk init`               | Create a new data package               |
| `dk dev up/down/status` | Manage local development stack          |
| `dk lint`               | Validate package manifests              |
| `dk run`                | Execute pipeline locally                |
| `dk test`               | Run tests with sample data              |
| `dk build`              | Build OCI artifact                      |
| `dk publish`            | Publish to OCI registry                 |
| `dk promote`            | Promote to environment via GitOps       |
| `dk status`             | Show package status across environments |
| `dk logs`               | Stream logs from running pipeline       |
| `dk rollback`           | Rollback to previous version            |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Developer                            │
│                            │                                 │
│                            ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   DK CLI (dk)                        │   │
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
├── cli/                # DK CLI implementation
├── demos/              # Scripted terminal demos (see demos/README.md)
├── platform/
│   └── controller/     # Kubernetes controller
├── gitops/             # Environment definitions
├── examples/           # Reference packages
├── hack/               # Development utilities
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

Validate with: `dk lint --strict`

### Data Lineage

DK automatically tracks lineage via OpenLineage:

- **START** event when pipeline begins
- **COMPLETE** event on success
- **FAIL** event with error details on failure

View lineage in Marquez UI: http://localhost:5000

## 🧪 Local Development

Start the full local stack:

```bash
# Start all services (Kafka, S3, PostgreSQL, Marquez)
dk dev up

# Check status
dk dev status

# View Marquez lineage UI
open http://localhost:5000

# Stop when done
dk dev down
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
