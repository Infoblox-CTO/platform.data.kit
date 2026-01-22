# Quickstart: CDPP MVP

**Feature**: 001-cdpp-mvp  
**Date**: 2026-01-22  
**Time to Complete**: ~30 minutes

## Prerequisites

- Go 1.22+ installed
- Docker and Docker Compose installed
- kubectl configured (optional, for K8s deployment)
- Access to an OCI registry (e.g., GitHub Container Registry)

## Install the CLI

```bash
# Install dp CLI
go install github.com/Infoblox-CTO/data-platform/cli@latest

# Verify installation
dp version
# Output: dp version v0.1.0 (go1.22.0)
```

## 1. Bootstrap a New Pipeline Package

```bash
# Create a new pipeline package
dp init my-first-pipeline --type pipeline

# Output:
# ✓ Created my-first-pipeline/
# ✓ Generated dp.yaml (package manifest)
# ✓ Generated pipeline.yaml (runtime config)
# ✓ Generated src/main.go (example pipeline)
# ✓ Generated docker-compose.yaml (local dependencies)
# ✓ Generated Dockerfile
#
# Next steps:
#   cd my-first-pipeline
#   dp dev        # Start local dependencies
#   dp run        # Execute pipeline locally

cd my-first-pipeline
```

### Generated Structure

```
my-first-pipeline/
├── dp.yaml              # Package manifest
├── pipeline.yaml        # Runtime configuration
├── bindings.local.yaml  # Local bindings
├── src/
│   └── main.go          # Pipeline code
├── schemas/
│   └── output.avsc      # Output schema
├── Dockerfile
├── docker-compose.yaml  # Local dependencies
└── .dp/
    └── config.yaml      # CLI configuration
```

## 2. Explore the Package Manifest

```yaml
# dp.yaml
apiVersion: cdpp.io/v1alpha1
kind: DataPackage
metadata:
  name: my-first-pipeline
  namespace: examples
  labels:
    team: data-platform
spec:
  type: pipeline
  description: Example pipeline that reads from Kafka and writes to S3
  owner: your-team
  
  inputs:
    - name: events
      type: kafka-topic
      binding: input.events
      
  outputs:
    - name: processed-events
      type: s3-prefix
      binding: output.lake
      schema:
        type: parquet
        schemaRef: ./schemas/output.avsc
      classification:
        pii: false
        sensitivity: internal
        dataCategory: example
        
  schedule:
    cron: "0 * * * *"  # Every hour
```

## 3. Start Local Dependencies

```bash
# Start LocalStack (S3), Redpanda (Kafka), and PostgreSQL
dp dev

# Output:
# ✓ Starting local stack...
# ✓ LocalStack (S3) ready at localhost:4566
# ✓ Redpanda (Kafka) ready at localhost:9092
# ✓ PostgreSQL ready at localhost:5432
# ✓ Created input topic: raw-events
# ✓ Created output bucket: local-data-lake
#
# Local stack is running. Use 'dp dev stop' to shut down.
```

## 4. Run the Pipeline Locally

```bash
# Execute the pipeline against local dependencies
dp run

# Output:
# ✓ Validating package manifest...
# ✓ Resolving bindings from bindings.local.yaml...
# ✓ Building pipeline image...
# ✓ Starting pipeline run (run-id: abc123)...
#
# [2026-01-22T10:30:00Z] INFO  Starting pipeline my-first-pipeline
# [2026-01-22T10:30:01Z] INFO  Connected to Kafka at localhost:9092
# [2026-01-22T10:30:02Z] INFO  Reading from topic: raw-events
# [2026-01-22T10:30:05Z] INFO  Processed 100 events
# [2026-01-22T10:30:06Z] INFO  Writing to s3://local-data-lake/output/
# [2026-01-22T10:30:07Z] INFO  Pipeline completed successfully
#
# ✓ Run completed in 7.2s
# ✓ Output written to s3://local-data-lake/output/
# ✓ Lineage event emitted
#
# Next steps:
#   dp run --watch    # Watch mode for development
#   dp lint           # Validate package
#   dp build          # Build release artifact
```

## 5. Validate the Package

```bash
# Validate manifest and contracts
dp lint

# Output:
# ✓ Checking dp.yaml syntax...
# ✓ Checking pipeline.yaml syntax...
# ✓ Validating artifact contracts...
# ✓ Checking schema references...
# ✓ Validating classification metadata...
#
# All checks passed!
```

## 6. Build and Publish

```bash
# Build the package artifact
dp build --version v0.1.0

# Output:
# ✓ Building container image...
# ✓ Packaging manifests...
# ✓ Creating OCI artifact...
# ✓ Artifact built: my-first-pipeline:v0.1.0
#
# Artifact ready for publishing.

# Publish to registry
dp publish --version v0.1.0

# Output:
# ✓ Authenticating to registry.example.com...
# ✓ Pushing artifact...
# ✓ Published: registry.example.com/cdpp/my-first-pipeline:v0.1.0
# ✓ Digest: sha256:abc123...
#
# Next steps:
#   dp promote dev --version v0.1.0   # Deploy to dev environment
```

## 7. Deploy to Dev Environment

```bash
# Deploy to dev environment (creates promotion PR)
dp promote dev --version v0.1.0

# Output:
# ✓ Validating version v0.1.0 exists...
# ✓ Checking bindings for dev environment...
# ✓ Generating deployment manifest...
# ✓ Creating promotion PR...
#
# Promotion PR created:
#   https://github.com/org/gitops-repo/pull/42
#
# The package will deploy after PR is approved and merged.
```

## 8. Monitor Pipeline Runs

Once deployed, view pipeline status in Grafana:

```bash
# Open Grafana dashboard
dp dashboard

# Or view run status via CLI
dp status my-first-pipeline --env dev

# Output:
# Package: my-first-pipeline
# Environment: dev
# Version: v0.1.0
# Status: Running
#
# Recent Runs:
#   run-xyz789  2026-01-22T11:00:00Z  Succeeded  Duration: 6.8s
#   run-xyz788  2026-01-22T10:00:00Z  Succeeded  Duration: 7.1s
#   run-xyz787  2026-01-22T09:00:00Z  Succeeded  Duration: 6.5s
```

## Common Commands Reference

| Command | Description |
|---------|-------------|
| `dp init <name>` | Bootstrap a new package |
| `dp dev` | Start local development stack |
| `dp dev stop` | Stop local development stack |
| `dp run` | Execute pipeline locally |
| `dp run --watch` | Execute with file watching |
| `dp lint` | Validate package manifests |
| `dp test` | Run package tests |
| `dp build --version <v>` | Build release artifact |
| `dp publish --version <v>` | Publish to registry |
| `dp promote <env> --version <v>` | Promote to environment |
| `dp status <pkg> --env <env>` | View deployment status |
| `dp logs <run-id>` | View run logs |
| `dp rollback <env> --version <v>` | Rollback to previous version |

## Output Formats

All commands support structured output:

```bash
# Human-readable (default)
dp status my-first-pipeline --env dev

# JSON for automation
dp status my-first-pipeline --env dev -o json

# YAML
dp status my-first-pipeline --env dev -o yaml
```

## Next Steps

1. **Customize your pipeline**: Edit `src/main.go` with your transformation logic
2. **Add real dependencies**: Update `dp.yaml` inputs/outputs for your data sources
3. **Set up CI/CD**: Use `dp lint` and `dp test` in your CI pipeline
4. **Deploy to production**: Promote through integration → staging → prod

## Troubleshooting

### Local stack won't start

```bash
# Check Docker is running
docker info

# Clean up and restart
dp dev stop
docker compose down -v
dp dev
```

### Pipeline fails with binding error

```bash
# Validate bindings match your manifest
dp lint --verbose

# Check bindings.local.yaml matches dp.yaml inputs/outputs
```

### Publish fails with auth error

```bash
# Configure registry credentials
dp config set registry.url registry.example.com
dp config set registry.username $USER
docker login registry.example.com
```
