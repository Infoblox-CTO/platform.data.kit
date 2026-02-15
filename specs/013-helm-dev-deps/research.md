# Research: Helm-Based Dev Dependencies

**Feature**: 013-helm-dev-deps | **Date**: 2026-02-15

## 1. Upstream Helm Chart Evaluation

### Redpanda — Use as Subchart ✅

| Property | Value |
|----------|-------|
| Chart | `redpanda` from `https://charts.redpanda.com` |
| Version | `25.3.2` (appVersion v25.3.6) |
| K8s minimum | `>= 1.25.0-0` |
| Helm minimum | `>= 3.10.0` |
| Console included | Yes — built-in subchart, `console.enabled: true` |

**Decision**: Use as subchart. Actively maintained, includes Console natively.

**Dev-mode overrides**:
- `statefulset.replicas: 1` (default 3)
- `storage.persistentVolume.enabled: false`
- `tls.enabled: false` (default true)
- `external.enabled: false` (prevents NodePort creation)
- `tuning.tune_aio_events: false` (avoids privileged initContainer)
- `resources.cpu.cores: 1`, `resources.memory.container.max: 1.5Gi`
- `console.enabled: true` (Console deployed as second pod in same release)

**Gotchas**: Default TLS=true, 3 replicas, privileged tuning initContainer. Must explicitly override all for dev.

### PostgreSQL (Bitnami) — Use as Subchart ✅

| Property | Value |
|----------|-------|
| Chart | `postgresql` from `oci://registry-1.docker.io/bitnamicharts/postgresql` |
| Version | `18.3.0` (appVersion 18.2.0 / PostgreSQL 17.x) |
| K8s minimum | `>= 1.23` |
| Dependencies | `common@2.36.0` from `oci://registry-1.docker.io/bitnamicharts` |

**Decision**: Use as subchart. Very mature, well-documented, supports initdb scripts natively.

**Dev-mode overrides**:
- `architecture: standalone`
- `primary.persistence.enabled: false`
- `primary.resourcesPreset: nano`
- `auth.postgresPassword: devpassword`, `auth.username: dpuser`, `auth.password: dppassword`, `auth.database: dataplatform`
- `primary.initdb.scripts` — SQL scripts for schema/table creation

**Gotchas**: Requires `common` subchart `.tgz` bundled. `auth.postgresPassword` is required.

### LocalStack — Custom Chart ✅

| Property | Value |
|----------|-------|
| Upstream chart | Exists at `https://localstack.github.io/helm-charts` v0.6.27 |
| Decision | **Keep custom chart** |

**Rationale**: Upstream chart has unpinned `latest` tag, 493 vulnerabilities, low activity, and overkill for S3-only dev use. Our custom chart is ~50 lines of YAML targeting only `SERVICES=s3`.

**Alternatives considered**:
- Upstream subchart: Rejected — security issues, unpinned image, unnecessary complexity (Docker-in-Docker, lambda support)
- Custom chart: Selected — minimal, pinned image tag, init Job for bucket creation

### Marquez — Custom Chart ✅

| Property | Value |
|----------|-------|
| Upstream chart | Exists in GitHub repo only (not published to any Helm repo) |
| Images | `marquezproject/marquez:0.51.1`, `marquezproject/marquez-web:0.51.1` |
| Decision | **Build custom chart** |

**Rationale**: No Helm repo URL means cannot declare as subchart dependency. The upstream chart is simple (~3 templates) and bundles its own PostgreSQL subchart (which conflicts with our shared instance).

**Design**:
- Marquez API server deployment (ports 5000/5001)
- Marquez Web UI deployment (port 3000)
- Points to shared PostgreSQL instance (no embedded DB)
- `migrateOnStartup: true` for schema creation via Flyway
- Init Job to create Marquez database in shared PostgreSQL and seed namespaces

## 2. Embedding .tgz Archives

**Decision**: Go `embed.FS` handles `.tgz` files natively — no special handling needed.

**Rationale**: `embed.FS` treats all files as opaque byte blobs. The existing `extractChartToDir()` function in `k3d.go` writes files byte-by-byte, preserving binary content.

**Build-time workflow**:
1. Run `helm dependency build` for redpanda and postgres charts (Makefile target)
2. Commit `Chart.lock` and `charts/*.tgz` files to Git
3. `//go:embed` directive automatically includes them
4. No runtime `helm dependency build` needed

**Alternatives considered**:
- Runtime `helm dependency build`: Rejected — requires network access, breaks offline operation (SC-004)
- Separate archive storage: Rejected — unnecessary complexity, embed handles it natively

## 3. Chart Deployment Abstraction

**Decision**: Introduce a `ChartDefinition` struct that replaces hardcoded service lists.

**Rationale**: Currently, service names, ports, health checks, and chart names are hardcoded across 5+ files. A single registry of chart definitions eliminates this coupling.

**Design**:
```
ChartDefinition {
  Name           string            // e.g. "redpanda"
  ReleaseName    string            // e.g. "dp-redpanda"
  Namespace      string            // e.g. "dp-local"
  PortForwards   []PortForward     // [{LocalPort: 19092, RemotePort: 9092, ServiceName: "dp-redpanda"}]
  HealthCheck    HealthCheckConfig // {ServiceLabel: "app.kubernetes.io/name=redpanda", TimeoutSeconds: 120}
}
```

All deployment, port-forwarding, health-checking, status, and teardown code operates on `[]ChartDefinition` — no service-specific code paths.

**Alternatives considered**:
- Keep hardcoded lists: Rejected — violates SC-003 (adding a dependency should not require orchestration changes)
- YAML config file for chart registry: Rejected — over-engineered for 4 charts; Go struct is sufficient

## 4. Config Extension for Chart Overrides

**Decision**: Add `Charts` field to `DevConfig` struct.

**Rationale**: Currently `DevConfig` has no chart-related configuration. FR-010 and FR-011 require version and value overrides via `dp config set`.

**Design**:
```
DevConfig {
  Runtime   string
  Workspace string
  K3d       K3dConfig
  Charts    map[string]ChartOverride   // NEW
}
ChartOverride {
  Version string                       // overrides embedded chart version
  Values  map[string]interface{}       // extra helm --set values
}
```

Config path: `dev.charts.<name>.version` and `dev.charts.<name>.values.<path>`.

## 5. Init Job Strategy

**Decision**: Use Helm post-install hooks for initialization tasks.

**Rationale**: Helm hooks run as Kubernetes Jobs after chart installation, provide completion tracking, and are deleted/recreated on subsequent `helm upgrade --install` runs. This is idempotent and fits the Helm lifecycle.

**Per-chart init**:
| Chart | Init Job | Tasks |
|-------|----------|-------|
| Redpanda | Post-install hook Job | Create topics: `dp.raw.events`, `dp.processed.events`, `dp.errors.dlq`, `dp.audit.logs`, `dp.test.input`, `dp.test.output` |
| LocalStack | Post-install hook Job | Create buckets: `cdpp-raw`, `cdpp-staging`, `cdpp-curated`, `cdpp-artifacts`, `cdpp-test` |
| PostgreSQL | `initdb.scripts` (built-in) | Create schemas, tables, indexes, sample data |
| Marquez | Init container + post-install hook | Create `marquez` database in PostgreSQL, seed namespaces (`dp`, `dp-dev`, `analytics`) and sources |

**Alternatives considered**:
- Init containers: Partial use — good for DB creation but can't wait for external services
- External scripts from Go code: Rejected — breaks uniform chart mechanism (FR-005)
