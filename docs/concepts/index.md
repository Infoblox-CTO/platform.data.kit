---
title: Concepts
description: Core concepts and architecture of the Data Platform
---

# Concepts

Understand the core concepts that power the Data Platform. This section explains the fundamental building blocks you'll use when building data pipelines.

## Core Concepts

<div class="grid" markdown>

<div class="card" markdown>
### :building_construction: Overview
High-level architecture and how the components fit together.

[Architecture Overview →](overview.md)
</div>

<div class="card" markdown>
### :package: Data Packages
Self-contained units of data processing with metadata and code.

[Learn about Data Packages →](data-packages.md)
</div>

<div class="card" markdown>
### :page_facing_up: Manifests
Configuration files that define your package: dp.yaml, pipeline.yaml, bindings.yaml.

[Understand Manifests →](manifests.md)
</div>

<div class="card" markdown>
### :link: Lineage
Track data flow and dependencies with OpenLineage and Marquez.

[Explore Lineage →](lineage.md)
</div>

<div class="card" markdown>
### :shield: Governance
PII classification, compliance metadata, and data protection.

[Data Governance →](governance.md)
</div>

<div class="card" markdown>
### :earth_americas: Environments
Development, integration, and production workflows.

[Environment Workflow →](environments.md)
</div>

</div>

## Learning Path

We recommend reading the concepts in this order:

1. **[Overview](overview.md)** - Start with the big picture
2. **[Data Packages](data-packages.md)** - Understand the core unit of work
3. **[Manifests](manifests.md)** - Learn how to configure packages
4. **[Lineage](lineage.md)** - Track data flow
5. **[Governance](governance.md)** - Classify and protect data
6. **[Environments](environments.md)** - Deploy across stages

## Key Principles

The Data Platform is built on these principles:

| Principle | Description |
|-----------|-------------|
| **Developer Experience First** | Simple happy path: bootstrap → run → validate → publish → promote |
| **Immutability** | Released artifacts cannot be modified; versions are permanent |
| **Separation of Concerns** | Infrastructure bindings are separate from pipeline logic |
| **Security by Default** | PII classification is required, not optional |
| **Observability** | Every operation emits metrics and lineage events |
