---
title: Concepts
description: Core concepts and architecture of DataKit
---

# Concepts

Understand the core concepts that power DataKit. This section explains the fundamental building blocks you'll use when building data pipelines.

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
Configuration files that define your package: Transform, DataSet, Connector, and Store manifests.

[Understand Manifests →](manifests.md)
</div>

<div class="card" markdown>
### :jigsaw: DataSets
Data contracts — tables, S3 prefixes, topics — with schema and column-level lineage.

[Learn about DataSets →](datasets.md)
</div>

<div class="card" markdown>
### :gear: Pipeline Workflows
Multi-step pipeline execution with sync, transform, test, publish, and custom steps.

[Pipeline Workflows →](pipelines.md)
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

<div class="card" markdown>
### :diamond_shape_with_a_dot_inside: Cells & Stores
Infrastructure contexts that separate what runs from where it runs.

[Learn about Cells →](cells.md)
</div>

</div>

## Learning Path

We recommend reading the concepts in this order:

1. **[Overview](overview.md)** - Start with the big picture
2. **[Data Packages](data-packages.md)** - Understand the core unit of work
3. **[Manifests](manifests.md)** - Learn how to configure packages
4. **[DataSets](datasets.md)** - Data contracts with schema and classification
5. **[Pipeline Workflows](pipelines.md)** - Define multi-step execution
6. **[Lineage](lineage.md)** - Track data flow
7. **[Governance](governance.md)** - Classify and protect data
8. **[Environments](environments.md)** - Deploy across stages
9. **[Cells & Stores](cells.md)** - Understand the Package × Cell model

## Key Principles

DataKit is built on these principles:

| Principle | Description |
|-----------|-------------|
| **Developer Experience First** | Simple happy path: bootstrap → run → validate → publish → promote |
| **Immutability** | Released artifacts cannot be modified; versions are permanent |
| **Separation of Concerns** | Connectors, Stores, DataSets, and Transforms have distinct ownership |
| **Security by Default** | PII classification is required, not optional |
| **Observability** | Every operation emits metrics and lineage events |
