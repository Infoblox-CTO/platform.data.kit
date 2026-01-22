# Data Model: Documentation - Getting Started Guide

**Feature**: 003-docs-getting-started  
**Date**: 2025-01-22  
**Status**: Complete

## Overview

This document defines the data model for the documentation system. Since this is a static documentation site, the "data model" represents the structure and relationships between documentation entities.

## Entities

### 1. Documentation Page

A single markdown file that renders as a web page.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Page title (H1 heading or frontmatter) |
| `description` | string | No | Meta description for SEO |
| `nav_order` | integer | No | Order in navigation (implicit from nav config) |
| `section` | string | Yes | Parent section (getting-started, concepts, etc.) |
| `content` | markdown | Yes | Page body content |
| `last_updated` | date | Auto | Git commit date |

**Example Frontmatter**:
```yaml
---
title: Installation
description: Install the DP CLI on macOS and Linux
---
```

### 2. Documentation Section

A logical grouping of related pages with its own index.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Section identifier (kebab-case) |
| `title` | string | Yes | Display name |
| `index_page` | string | Yes | Section landing page (index.md) |
| `pages` | Page[] | Yes | Ordered list of pages in section |
| `icon` | string | No | Material icon for navigation |

**Sections Defined**:

| Name | Title | Icon | Page Count |
|------|-------|------|------------|
| `getting-started` | Getting Started | `rocket_launch` | 4 |
| `concepts` | Concepts | `school` | 7 |
| `tutorials` | Tutorials | `integration_instructions` | 4 |
| `reference` | Reference | `menu_book` | 4 |
| `troubleshooting` | Troubleshooting | `help` | 3 |

### 3. Code Example

Executable code snippet embedded in documentation.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `language` | string | Yes | Programming language (bash, yaml, go, python) |
| `code` | string | Yes | The code content |
| `title` | string | No | Code block title |
| `annotations` | object | No | Line-by-line annotations |
| `copyable` | boolean | Yes | Always true (Material theme default) |

**Supported Languages**:
- `bash` / `shell` - CLI commands
- `yaml` - Configuration files (dp.yaml, pipeline.yaml, mkdocs.yml)
- `go` - Go code examples
- `python` - Python processing examples
- `json` - API responses, data structures

### 4. Navigation Configuration

Hierarchical navigation structure defined in mkdocs.yml.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `nav` | NavItem[] | Yes | Ordered navigation tree |
| `tabs` | boolean | Yes | Enable top-level tabs |
| `sections` | boolean | Yes | Enable section grouping |

**Navigation Schema**:
```yaml
nav:
  - Home: index.md
  - Getting Started:
    - getting-started/index.md
    - Prerequisites: getting-started/prerequisites.md
    - Installation: getting-started/installation.md
    - Quickstart: getting-started/quickstart.md
  - Concepts:
    - concepts/index.md
    - Overview: concepts/overview.md
    - Data Packages: concepts/data-packages.md
    - Manifests: concepts/manifests.md
    - Lineage: concepts/lineage.md
    - Governance: concepts/governance.md
    - Environments: concepts/environments.md
  - Tutorials:
    - tutorials/index.md
    - Kafka to S3: tutorials/kafka-to-s3.md
    - Local Development: tutorials/local-development.md
    - Promoting Packages: tutorials/promoting-packages.md
  - Reference:
    - reference/index.md
    - CLI Commands: reference/cli.md
    - Manifest Schema: reference/manifest-schema.md
    - Configuration: reference/configuration.md
  - Troubleshooting:
    - troubleshooting/index.md
    - Common Issues: troubleshooting/common-issues.md
    - FAQ: troubleshooting/faq.md
  - Architecture: architecture.md
  - Testing: testing.md
  - Contributing: contributing.md
```

### 5. Admonition (Callout)

Highlighted content block for notes, warnings, tips.

| Type | Usage | Icon |
|------|-------|------|
| `note` | General information | `pencil` |
| `tip` | Helpful suggestions | `flame` |
| `warning` | Important cautions | `alert` |
| `danger` | Critical warnings | `zap` |
| `info` | Informational notes | `info` |
| `success` | Success confirmations | `check` |

**Syntax**:
```markdown
!!! tip "Pro Tip"
    This is a helpful tip for developers.

!!! warning
    This is an important warning.
```

## Relationships

```
┌─────────────────────────────────────────────────────────────┐
│                    mkdocs.yml                               │
│                  (Configuration)                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 Navigation                           │   │
│  │  nav:                                               │   │
│  │    - Section A                                      │   │
│  │        - Page 1                                     │   │
│  │        - Page 2                                     │   │
│  │    - Section B                                      │   │
│  │        - Page 3                                     │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                       docs/                                  │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐   │
│  │   Section A   │  │   Section B   │  │   Root Pages  │   │
│  │   (folder)    │  │   (folder)    │  │               │   │
│  │ ┌───────────┐ │  │ ┌───────────┐ │  │ index.md      │   │
│  │ │ index.md  │ │  │ │ index.md  │ │  │ architecture  │   │
│  │ │ page1.md  │ │  │ │ page3.md  │ │  │ testing.md    │   │
│  │ │ page2.md  │ │  │ └───────────┘ │  │ contributing  │   │
│  │ └───────────┘ │  └───────────────┘  └───────────────┘   │
│  └───────────────┘                                          │
└─────────────────────────────────────────────────────────────┘
```

## File Inventory

### New Files to Create

| Path | Description | Priority |
|------|-------------|----------|
| `mkdocs.yml` | MkDocs configuration | P1 |
| `requirements-docs.txt` | Python dependencies | P1 |
| `docs/index.md` | Homepage | P1 |
| `docs/getting-started/index.md` | Section landing | P1 |
| `docs/getting-started/prerequisites.md` | Prerequisites | P1 |
| `docs/getting-started/installation.md` | Installation guide | P1 |
| `docs/getting-started/quickstart.md` | Quickstart tutorial | P1 |
| `docs/concepts/index.md` | Section landing | P2 |
| `docs/concepts/overview.md` | Architecture overview | P2 |
| `docs/concepts/data-packages.md` | Data packages concept | P2 |
| `docs/concepts/manifests.md` | Manifest files | P2 |
| `docs/concepts/lineage.md` | OpenLineage | P2 |
| `docs/concepts/governance.md` | PII/Governance | P2 |
| `docs/concepts/environments.md` | Env workflow | P2 |
| `docs/tutorials/index.md` | Section landing | P3 |
| `docs/tutorials/kafka-to-s3.md` | Kafka pipeline | P3 |
| `docs/tutorials/local-development.md` | Local dev | P3 |
| `docs/tutorials/promoting-packages.md` | GitOps promotion | P3 |
| `docs/reference/index.md` | Section landing | P2 |
| `docs/reference/cli.md` | CLI reference | P2 |
| `docs/reference/manifest-schema.md` | Schema reference | P2 |
| `docs/reference/configuration.md` | Config reference | P2 |
| `docs/troubleshooting/index.md` | Section landing | P3 |
| `docs/troubleshooting/common-issues.md` | Common issues | P3 |
| `docs/troubleshooting/faq.md` | FAQ | P3 |
| `docs/contributing.md` | Contribution guide | P2 |
| `docs/stylesheets/extra.css` | Custom styles | P2 |
| `.github/workflows/docs.yaml` | Deployment workflow | P1 |

### Existing Files to Update

| Path | Changes | Priority |
|------|---------|----------|
| `docs/architecture.md` | Add frontmatter, update internal links | P2 |
| `docs/cli-reference.md` | Migrate content to `reference/cli.md` | P2 |
| `docs/testing.md` | Add frontmatter, update links | P2 |
| `.gitignore` | Add `site/` directory | P1 |
| `README.md` | Add link to documentation site | P1 |

## Validation Rules

1. **All pages must have valid frontmatter** (title at minimum)
2. **All internal links must be relative** (not absolute URLs)
3. **All code blocks must specify a language** for syntax highlighting
4. **Navigation must include all pages** in mkdocs.yml nav section
5. **Images must be in `docs/assets/images/`** directory
6. **Build must pass with `mkdocs build --strict`** (no warnings)
