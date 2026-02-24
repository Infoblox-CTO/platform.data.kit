---
title: Home
description: Data Platform (DP) - A Kubernetes-native data pipeline platform
hide:
  - navigation
  - toc
---

<div class="hero" markdown>

# Data Platform Documentation

A Kubernetes-native data pipeline platform enabling teams to contribute reusable, versioned "data packages" with a complete developer workflow.

**bootstrap → local run → validate → publish → promote**

[Get Started](getting-started/index.md){ .md-button .md-button--primary }
[View on GitHub](https://github.com/Infoblox-CTO/platform.data.kit){ .md-button }

</div>

---

## What is DP?

DP (Data Platform) is a developer-first platform for building, testing, and deploying data pipelines. It provides:

- **📦 Data Packages**: Self-contained units of data processing with manifests, pipelines, and bindings
- **🔄 GitOps Workflow**: PR-based promotion through dev → int → prod environments
- **📊 Data Lineage**: Automatic OpenLineage tracking with Marquez integration
- **🔒 Governance**: Built-in PII classification and compliance metadata

---

## Quick Links

<div class="grid" markdown>

<div class="card" markdown>
### :rocket: Getting Started
New to DP? Start here to install the CLI and run your first pipeline in under 30 minutes.

[Get Started →](getting-started/index.md)
</div>

<div class="card" markdown>
### :books: Concepts
Understand the core concepts: data packages, manifests, lineage, and governance.

[Learn Concepts →](concepts/index.md)
</div>

<div class="card" markdown>
### :hammer_and_wrench: Tutorials
Step-by-step guides for building real-world pipelines and workflows.

[View Tutorials →](tutorials/index.md)
</div>

<div class="card" markdown>
### :book: Reference
Complete CLI reference, manifest schemas, and configuration options.

[Browse Reference →](reference/index.md)
</div>

</div>

---

## The DP Workflow

```bash
# 1. Create a new data package
dp init my-pipeline --kind model --runtime cloudquery

# 2. Start local development environment
dp dev up

# 3. Validate your package
dp lint ./my-pipeline

# 4. Run locally
dp run ./my-pipeline

# 5. Build and publish
dp build ./my-pipeline
dp publish ./my-pipeline

# 6. Promote to an environment
dp promote my-pipeline v0.1.0 --to dev
```

---

## Architecture Overview

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
│  └─────────────────────────────────────────────────────┘   │
│                            │                                 │
│                            ▼                                 │
│  ┌─────────────────────────────────────────────────────┐   │
│  │            Kubernetes Platform Controller            │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

[View Full Architecture →](architecture.md)

---

## Need Help?

- **[Troubleshooting](troubleshooting/index.md)** - Common issues and solutions
- **[FAQ](troubleshooting/faq.md)** - Frequently asked questions
- **[Contributing](contributing.md)** - How to contribute to DP
- **[GitHub Issues](https://github.com/Infoblox-CTO/platform.data.kit/issues)** - Report bugs or request features
