# Quickstart: Helm-Based Dev Dependencies

**Feature**: 013-helm-dev-deps

## Developer Workflow

### Start the dev environment

```bash
# Start all dev dependencies (Redpanda, LocalStack, PostgreSQL, Marquez)
dp dev up

# Output shows:
# ✓ Deploying dp-redpanda... done
# ✓ Deploying dp-localstack... done
# ✓ Deploying dp-postgres... done
# ✓ Deploying dp-marquez... done
#
# Dev environment ready:
#   Kafka:            localhost:19092
#   Schema Registry:  http://localhost:18081
#   Redpanda Console: http://localhost:8080
#   S3 API:           http://localhost:4566
#   PostgreSQL:       localhost:5432
#   Marquez API:      http://localhost:5000
#   Marquez Web:      http://localhost:3000
```

### Check status

```bash
dp dev status

# Output shows per-chart status:
# NAME         CHART VERSION  APP VERSION  STATUS    HEALTH
# redpanda     25.3.2         v25.3.6      deployed  healthy
# localstack   0.2.0          3.8.1        deployed  healthy
# postgres     18.3.0         17.x         deployed  healthy
# marquez      0.2.0          0.51.1       deployed  healthy
```

### Stop the dev environment

```bash
# Stop and remove all dev services
dp dev down

# Stop and remove volumes too
dp dev down --volumes
```

### Override a chart version (advanced)

```bash
# Test with a different Redpanda version
dp config set dev.charts.redpanda.version 25.2.0

# Next dp dev up will use the overridden version
dp dev up

# Revert to embedded default
dp config unset dev.charts.redpanda.version
```

### Override Helm values (advanced)

```bash
# Give PostgreSQL more memory
dp config set dev.charts.postgres.values.primary.resources.limits.memory 1Gi

# Apply changes
dp dev up
```

## For Maintainers: Adding a New Dev Dependency

1. Create a chart directory under `sdk/localdev/charts/<name>/` with:
   - `Chart.yaml` — chart metadata
   - `values.yaml` — dev-appropriate defaults
   - `templates/` — Kubernetes resource templates

2. If using an upstream chart as subchart:
   - Add `dependencies:` to `Chart.yaml`
   - Run `helm dependency build` in the chart directory
   - Commit the `Chart.lock` and `charts/*.tgz` files

3. Register the chart in `sdk/localdev/charts/embed.go`:
   - Add the directory name to the `//go:embed` directive
   - Add a `ChartDefinition` entry to `DefaultCharts`

4. Run `make build` — the chart is now embedded in the CLI binary.

No changes to deployment, health-checking, port-forwarding, or status-reporting code are needed.
