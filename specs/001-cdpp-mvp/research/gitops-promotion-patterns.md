# Research: GitOps Patterns for Multi-Environment Deployment & Promotion

**Date**: 2026-01-22  
**Context**: CDPP MVP - Package promotion workflow design  
**Sources**: Codefresh GitOps guides, Argo CD docs, Flux docs, Kargo project, External Secrets Operator docs

---

## Executive Summary

This research informs the GitOps architecture for promoting data packages through environments (dev → integration → staging → production). Based on industry best practices and tooling analysis, we recommend:

| Area | Decision | Confidence |
|------|----------|------------|
| GitOps Tool | **ArgoCD with ApplicationSets** | High |
| Repository Structure | **Single repo, environment-per-folder** | High |
| Templating | **Kustomize overlays** | High |
| Promotion Automation | **Kargo** (or custom GitHub Actions) | Medium |
| Bindings/Secrets | **External Secrets Operator + ConfigMaps** | High |

---

## 1. Flux vs ArgoCD for Package Promotion

### Requirements Recap
- Each environment has pinned package versions
- Promotion = PR that changes the version reference
- Multiple independent packages per environment
- Rollback = promoting the previous version

### Analysis

| Criteria | ArgoCD | Flux | Winner |
|----------|--------|------|--------|
| **Multi-app management** | ApplicationSets generate apps from folders/files | Kustomizations per path | ArgoCD (declarative multi-app) |
| **UI for visibility** | Built-in web UI with sync status | No built-in UI (use Weave GitOps) | ArgoCD |
| **Promotion workflow** | Better ecosystem (Kargo, Codefresh) | Image reflector for auto-updates | ArgoCD |
| **Diff preview** | Built-in diff before sync | Requires `flux diff` CLI | ArgoCD |
| **Helm values from Git** | Multi-source apps (v2.6+) | Native | Tie |
| **Rollback UX** | One-click in UI or sync to revision | `flux reconcile --revision` | ArgoCD |
| **Enterprise adoption** | Higher (Akuity, Codefresh backing) | Strong (CNCF graduated) | ArgoCD |
| **Webhook triggers** | Supported | Supported | Tie |
| **Progressive delivery** | Argo Rollouts integration | Flagger integration | Tie |

### Recommendation: **ArgoCD**

**Rationale**:
1. **ApplicationSets** excel at generating Applications from folder structures—ideal for "one folder per package per environment"
2. **Kargo** (by Akuity) provides native promotion pipeline automation on top of ArgoCD
3. **Web UI** provides visibility into sync status across environments without additional tooling
4. **Diff preview** helps reviewers understand promotion impact before merging PRs
5. **Industry momentum**: More GitOps promotion tools target ArgoCD (Codefresh, Kargo, Telefonistka)

**Alternatives Considered**:
- **Flux**: Excellent for simpler setups; weaker ecosystem for promotion automation
- **Hybrid**: Not recommended due to operational complexity

---

## 2. Environment Structure: Repository Organization

### Options Evaluated

| Pattern | Description | Pros | Cons |
|---------|-------------|------|------|
| **Branch-per-env** | `main`=prod, `staging` branch, etc. | Simple mental model | Merge direction problems, commit order issues, hard to compare envs |
| **Repo-per-env** | Separate Git repos for prod vs non-prod | Strong access control | Promotion requires cross-repo PRs, harder to audit |
| **Folder-per-env (single repo)** | All envs in one repo, one branch, folders | Easy diffing, atomic multi-env changes, single source of truth | Requires folder-level access control (CODEOWNERS) |

### Recommendation: **Single Repo, Folder-per-Environment**

**Rationale** (from Codefresh best practices):
1. **Commit order irrelevant**: When promoting, you copy files—no merge conflicts or cherry-pick nightmares
2. **Easy rollback**: `git revert` works cleanly when each commit touches one environment
3. **Auditable diff**: `vimdiff envs/staging/version.yaml envs/prod/version.yaml` shows exactly what differs
4. **Atomic operations**: Promote multiple packages together in one commit if needed
5. **Static branch count**: Always 1 branch (`main`), regardless of environment count

### Recommended Structure

```
gitops-repo/
├── base/                           # Shared defaults (rarely changes)
│   └── package-template/
│       ├── kustomization.yaml
│       └── deployment.yaml
│
├── variants/                       # Cross-cutting environment traits
│   ├── prod/
│   │   └── production-settings.yaml
│   └── non-prod/
│       └── non-prod-settings.yaml
│
├── packages/                       # Package-specific configurations
│   ├── kafka-s3-pipeline/
│   │   ├── base/
│   │   │   ├── kustomization.yaml
│   │   │   └── pipeline-job.yaml
│   │   └── overlays/
│   │       ├── dev/
│   │       │   ├── kustomization.yaml
│   │       │   └── version.yaml       # ← PROMOTION CHANGES THIS FILE
│   │       ├── integration/
│   │       ├── staging/
│   │       └── prod/
│   │
│   └── clickstream-processor/
│       └── overlays/
│           ├── dev/
│           └── ...
│
└── environments/                   # ArgoCD ApplicationSets
    ├── dev/
    │   └── applicationset.yaml     # Generates apps for all packages in dev
    ├── integration/
    ├── staging/
    └── prod/
```

### Key Design Principles

1. **`version.yaml` is the promotion target**: Each overlay has a single file that contains only the package version/image tag. Promotion = copying this file.

2. **Separation of concerns**:
   - `base/`: Kubernetes resource templates (Deployment, Job, etc.)
   - `variants/`: Environment-class settings (prod vs non-prod)
   - `packages/<name>/overlays/<env>/`: Per-package, per-environment config

3. **ApplicationSet per environment**: One ApplicationSet discovers all packages in that environment's folder, generating ArgoCD Applications automatically.

---

## 3. Kustomize Overlays vs Helm Values

### Analysis

| Criteria | Kustomize Overlays | Helm Values Files |
|----------|-------------------|-------------------|
| **Version pinning** | Patch file with image/tag | values.yaml with `image.tag` |
| **Composability** | Components, patches, strategic merge | Chart dependencies, subcharts |
| **Learning curve** | Lower (plain YAML) | Higher (Go templates) |
| **Debugging** | `kustomize build` shows final YAML | `helm template` shows final YAML |
| **Multi-environment** | Native overlay structure | Multiple values files, override order matters |
| **ArgoCD support** | Native | Native |
| **Chart versioning** | N/A | Requires placing Chart.yaml in each env folder for version differences |

### Recommendation: **Kustomize Overlays**

**Rationale**:
1. **Simpler promotion**: Copy `version.yaml` between folders—no template understanding needed
2. **Transparent diffing**: Reviewers see exact YAML differences, not values abstraction
3. **Fewer moving parts**: No Helm chart versioning complexity
4. **ArgoCD best practice**: Helm for third-party charts, Kustomize for in-house apps

**File: `packages/kafka-s3-pipeline/overlays/dev/version.yaml`**
```yaml
apiVersion: apps/v1
kind: Job
metadata:
  name: kafka-s3-pipeline
spec:
  template:
    spec:
      containers:
      - name: pipeline
        image: registry.example.com/cdpp/kafka-s3-pipeline:v1.2.3  # ← Only this changes
```

**File: `packages/kafka-s3-pipeline/overlays/dev/kustomization.yaml`**
```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

components:
  - ../../../variants/non-prod

patchesStrategicMerge:
  - version.yaml
  - bindings.yaml
```

---

## 4. Promotion Workflow Automation

### The Promotion Model

```
┌─────────────────────────────────────────────────────────────────────┐
│                         PROMOTION FLOW                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌─────────┐      ┌─────────────┐      ┌─────────┐      ┌───────┐  │
│   │   DEV   │ ───► │ INTEGRATION │ ───► │ STAGING │ ───► │ PROD  │  │
│   │  v1.2.3 │      │    v1.2.2   │      │  v1.2.1 │      │ v1.2.0│  │
│   └─────────┘      └─────────────┘      └─────────┘      └───────┘  │
│                                                                      │
│   Promotion Action:                                                  │
│   cp packages/<pkg>/overlays/dev/version.yaml \                     │
│      packages/<pkg>/overlays/integration/version.yaml               │
│                                                                      │
│   Rollback Action:                                                   │
│   Edit version.yaml to previous version (same as promotion)         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Tools for Automation

| Tool | Description | Fit for CDPP |
|------|-------------|--------------|
| **Kargo** | Purpose-built for ArgoCD promotions; tracks "Freight" through Stages | ⭐ Excellent |
| **GitHub Actions** | Custom workflow: cp file, create PR | Good (more DIY) |
| **Telefonistka** | Auto-promote on merge; good for simple flows | Medium |
| **Codefresh GitOps** | Commercial platform with promotion UI | Medium (cost) |
| **Flux Image Automation** | Auto-updates image tags | Poor (auto-promote not desired) |

### Recommendation: **Kargo** (primary) or **GitHub Actions** (fallback)

#### Kargo Approach

Kargo introduces:
- **Warehouse**: Watches for new artifact versions (OCI registry)
- **Freight**: An immutable collection of artifacts at specific versions
- **Stage**: Environment where Freight can be promoted
- **Promotion**: Moving Freight from one Stage to the next

**Kargo Stage Example**:
```yaml
apiVersion: kargo.akuity.io/v1alpha1
kind: Stage
metadata:
  name: integration
  namespace: cdpp
spec:
  requestedFreight:
    - origin:
        kind: Warehouse
        name: kafka-s3-pipeline-warehouse
      sources:
        stages:
          - dev  # Only accept freight that's been in dev
  promotionTemplate:
    spec:
      steps:
        - uses: git-clone
        - uses: kustomize-set-image
          config:
            path: packages/kafka-s3-pipeline/overlays/integration
            images:
              - image: registry.example.com/cdpp/kafka-s3-pipeline
        - uses: git-commit
        - uses: git-push
        - uses: argocd-update
```

**Benefits**:
- Declarative promotion pipelines
- Built-in approval gates
- Visual promotion dashboard
- Automatic PR creation
- Freight tracking (what's deployed where)

#### GitHub Actions Fallback

If Kargo is too heavy for MVP:

```yaml
# .github/workflows/promote.yaml
name: Promote Package
on:
  workflow_dispatch:
    inputs:
      package:
        description: 'Package name'
        required: true
      source_env:
        description: 'Source environment'
        required: true
      target_env:
        description: 'Target environment'
        required: true

jobs:
  promote:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Copy version file
        run: |
          cp packages/${{ inputs.package }}/overlays/${{ inputs.source_env }}/version.yaml \
             packages/${{ inputs.package }}/overlays/${{ inputs.target_env }}/version.yaml
      
      - name: Create PR
        uses: peter-evans/create-pull-request@v6
        with:
          title: "Promote ${{ inputs.package }} from ${{ inputs.source_env }} to ${{ inputs.target_env }}"
          branch: promote/${{ inputs.package }}/${{ inputs.target_env }}
          body: |
            ## Promotion Request
            - **Package**: ${{ inputs.package }}
            - **From**: ${{ inputs.source_env }}
            - **To**: ${{ inputs.target_env }}
            
            ### Changes
            This PR updates the package version in the target environment.
```

### Rollback Pattern

**Rollback = Promotion of Previous Version**

There is no special rollback mechanism. To rollback:
1. Identify the previous working version
2. Edit `version.yaml` to that version (or copy from a Git revision)
3. Create PR, merge, ArgoCD syncs

This is intentional: rollback follows the same auditable, reviewable process as any promotion.

---

## 5. Bindings Pattern: Environment-Specific Configuration

### Problem Statement

Packages need environment-specific configuration that is NOT part of the package artifact:
- Kafka bootstrap servers
- S3 bucket names
- Database connection strings
- API endpoints
- Feature flags

These "bindings" must:
1. Be injected at deploy time, not baked into artifacts
2. Differ between environments
3. Be validated against a contract

### Pattern: ConfigMaps + External Secrets Operator

```
┌─────────────────────────────────────────────────────────────────────┐
│                       BINDINGS ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌──────────────────┐        ┌──────────────────┐                  │
│   │ bindings.yaml    │        │  ExternalSecret  │                  │
│   │ (ConfigMap)      │        │  (ESO CRD)       │                  │
│   ├──────────────────┤        ├──────────────────┤                  │
│   │ KAFKA_BROKERS    │        │ DB_PASSWORD      │                  │
│   │ S3_BUCKET        │        │ API_KEY          │                  │
│   │ LOG_LEVEL        │        │ (from Vault/AWS) │                  │
│   └────────┬─────────┘        └────────┬─────────┘                  │
│            │                           │                             │
│            └───────────┬───────────────┘                            │
│                        ▼                                             │
│              ┌──────────────────┐                                   │
│              │  Pipeline Pod    │                                   │
│              │  (envFrom both)  │                                   │
│              └──────────────────┘                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Implementation

**1. Bindings ConfigMap (per environment)**

```yaml
# packages/kafka-s3-pipeline/overlays/dev/bindings.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kafka-s3-pipeline-bindings
data:
  KAFKA_BROKERS: "kafka.dev.internal:9092"
  S3_BUCKET: "cdpp-dev-data"
  S3_PREFIX: "kafka-s3-pipeline/output"
  LOG_LEVEL: "debug"
```

**2. External Secrets for Sensitive Data**

```yaml
# packages/kafka-s3-pipeline/overlays/dev/secrets.yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: kafka-s3-pipeline-secrets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend  # Cluster-wide SecretStore
    kind: ClusterSecretStore
  target:
    name: kafka-s3-pipeline-secrets
  data:
    - secretKey: KAFKA_SASL_PASSWORD
      remoteRef:
        key: secret/cdpp/dev/kafka
        property: password
    - secretKey: AWS_SECRET_ACCESS_KEY
      remoteRef:
        key: secret/cdpp/dev/aws
        property: secret_key
```

**3. Pipeline Job Consumes Both**

```yaml
# packages/kafka-s3-pipeline/base/pipeline-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: kafka-s3-pipeline
spec:
  template:
    spec:
      containers:
        - name: pipeline
          image: registry.example.com/cdpp/kafka-s3-pipeline:latest
          envFrom:
            - configMapRef:
                name: kafka-s3-pipeline-bindings
            - secretRef:
                name: kafka-s3-pipeline-secrets
```

### Contract Validation for Bindings

Packages declare required bindings in their manifest:

```yaml
# dp.yaml (package manifest)
apiVersion: cdpp.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: kafka-s3-pipeline
spec:
  bindings:
    required:
      - name: KAFKA_BROKERS
        description: "Comma-separated Kafka broker addresses"
        pattern: "^[a-z0-9.-]+(:[0-9]+)?(,[a-z0-9.-]+(:[0-9]+)?)*$"
      - name: S3_BUCKET
        description: "Target S3 bucket name"
      - name: S3_PREFIX
        description: "Object prefix within bucket"
    optional:
      - name: LOG_LEVEL
        description: "Logging verbosity"
        default: "info"
        enum: ["debug", "info", "warn", "error"]
```

**Validation at Deploy Time** (in ArgoCD PreSync hook or admission webhook):

```go
// sdk/gitops/validate_bindings.go
func ValidateBindings(manifest DataPackage, configMap ConfigMap) error {
    for _, req := range manifest.Spec.Bindings.Required {
        value, exists := configMap.Data[req.Name]
        if !exists {
            return fmt.Errorf("missing required binding: %s", req.Name)
        }
        if req.Pattern != "" {
            re := regexp.MustCompile(req.Pattern)
            if !re.MatchString(value) {
                return fmt.Errorf("binding %s value %q does not match pattern %s", 
                    req.Name, value, req.Pattern)
            }
        }
    }
    return nil
}
```

---

## 6. Decision Summary

### Final Recommendations

| Decision Area | Choice | Rationale |
|---------------|--------|-----------|
| **GitOps Controller** | ArgoCD + ApplicationSets | Best ecosystem for promotion workflows; Kargo integration; built-in UI |
| **Repository Structure** | Single repo, folder-per-env | Atomic commits, easy diffing, no merge conflicts on promotion |
| **Templating** | Kustomize overlays | Simpler than Helm for in-house apps; transparent promotion via file copy |
| **Promotion Tool** | Kargo (MVP+) or GitHub Actions (MVP) | Kargo is ideal but adds complexity; GH Actions sufficient for MVP |
| **Config Injection** | ConfigMaps for bindings, ESO for secrets | Clean separation; well-supported pattern |
| **Rollback** | Same as promotion (pin previous version) | No special mechanism needed; maintains audit trail |

### Alternatives Rejected

| Alternative | Why Rejected |
|-------------|--------------|
| **Flux** | Weaker promotion ecosystem; no built-in UI |
| **Branch-per-environment** | Merge conflicts, commit order issues, hard to audit |
| **Separate repos per env** | Cross-repo PRs, harder to maintain atomic promotions |
| **Helm for everything** | Overkill for in-house apps; chart versioning adds complexity |
| **Baked-in config** | Violates immutability; can't reuse same artifact across envs |
| **Kubernetes Secrets directly** | Committed secrets are a security anti-pattern |

---

## 7. Implementation Roadmap for CDPP MVP

### Phase 1: Basic Structure (Week 1-2)
- [ ] Create gitops repo with folder structure
- [ ] Set up ArgoCD with ApplicationSets for dev environment
- [ ] Create sample package Kustomize overlay
- [ ] Implement `version.yaml` pattern

### Phase 2: Promotion Workflow (Week 3-4)
- [ ] Create GitHub Actions workflow for manual promotion
- [ ] Add PR template with promotion checklist
- [ ] Implement bindings ConfigMap pattern
- [ ] Integrate External Secrets Operator

### Phase 3: Automation (Post-MVP)
- [ ] Evaluate Kargo for declarative promotion pipelines
- [ ] Add bindings validation webhook
- [ ] Build promotion dashboard in CDPP UI

---

## References

1. [How to Model Your GitOps Environments and Promote Releases (Codefresh)](https://codefresh.io/blog/how-to-model-your-gitops-environments-and-promote-releases-between-them/)
2. [ArgoCD Best Practices](https://argo-cd.readthedocs.io/en/stable/user-guide/best_practices/)
3. [ArgoCD ApplicationSets](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/)
4. [Flux for Helm Users](https://fluxcd.io/flux/use-cases/helm/)
5. [External Secrets Operator](https://external-secrets.io/latest/)
6. [Kargo - Application Lifecycle Orchestration](https://github.com/akuity/kargo)
7. [Stop Using Branches for GitOps Environments (Codefresh)](https://codefresh.io/blog/stop-using-branches-deploying-different-gitops-environments/)
