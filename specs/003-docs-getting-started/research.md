# Research: Documentation - Getting Started Guide

**Feature**: 003-docs-getting-started  
**Date**: 2025-01-22  
**Status**: Complete

## Research Questions

### 1. Static Site Generator Selection

**Question**: Which static site generator should be used for GitHub Pages documentation?

**Options Evaluated**:

| Generator | Pros | Cons |
|-----------|------|------|
| **Jekyll** | Native GitHub Pages support, Ruby-based, built-in themes | Ruby dependency, slower builds, less modern UI |
| **MkDocs** | Python-based, excellent Material theme, fast builds, built-in search | Requires Python, needs deployment action |
| **Docusaurus** | React-based, versioning support, modern UI | Heavier (Node.js), overkill for project size |
| **Hugo** | Extremely fast, Go-based | Less documentation-focused themes |

**Decision**: **MkDocs with Material theme**

**Rationale**:
1. Python is commonly available in development environments
2. Material theme provides excellent UX out of the box (search, navigation, mobile-responsive)
3. Wide adoption in similar projects (FastAPI, Kubernetes components, Poetry)
4. Simple YAML configuration
5. Native GitHub Pages deployment via `mkdocs gh-deploy` or Actions
6. Built-in syntax highlighting for code blocks
7. No JavaScript framework complexity

**Alternatives Rejected**:
- Jekyll: Requires Ruby, theme customization more complex
- Docusaurus: Overkill for ~20 pages, Node.js dependency
- Hugo: Theme selection less documentation-focused

---

### 2. Documentation Structure Best Practices

**Question**: What structure do popular software products use for documentation?

**Research Findings**:

| Project | Structure | Key Patterns |
|---------|-----------|--------------|
| **Kubernetes** | Getting Started → Concepts → Tasks → Reference | Task-based tutorials, concept-first learning |
| **Docker** | Get Started → Guides → Reference → Manuals | Progressive complexity, clear CLI reference |
| **Stripe** | Quickstart → Guides → API Reference | Developer-first, copy-paste examples |
| **Terraform** | Intro → Install → Use → Language → CLI | Workflow-based progression |
| **FastAPI** | Tutorial → Advanced → Reference | Single progressive tutorial path |

**Decision**: Hybrid structure following Kubernetes/Docker patterns

**Structure**:
```
1. Getting Started (P1 - Onboarding)
   - Prerequisites
   - Installation
   - Quickstart (end-to-end in 10 min)

2. Concepts (P2 - Understanding)
   - Architecture Overview
   - Data Packages
   - Manifests
   - Lineage
   - Governance
   - Environments

3. Tutorials (P3 - Real-world usage)
   - Kafka to S3 Pipeline
   - Local Development
   - Promoting Packages

4. Reference (P2 - Daily usage)
   - CLI Commands
   - Manifest Schema
   - Configuration

5. Troubleshooting (P3 - Self-service support)
   - Common Issues
   - FAQ
```

**Rationale**:
1. Getting Started is the entry point for all new users
2. Concepts provide foundational knowledge before deep-dive tutorials
3. Reference is for daily lookup by experienced users
4. Troubleshooting reduces support burden

---

### 3. GitHub Pages Deployment

**Question**: How to deploy MkDocs to GitHub Pages?

**Options Evaluated**:

| Method | Pros | Cons |
|--------|------|------|
| `mkdocs gh-deploy` | Simple, one command | Manual deployment |
| **GitHub Actions** | Automated on push, CI/CD integrated | Requires workflow file |
| GitHub Pages (Jekyll) | Native integration | Doesn't support MkDocs |

**Decision**: **GitHub Actions workflow**

**Rationale**:
1. Automatic deployment on push to main branch
2. Can validate documentation builds before deployment
3. Integrates with existing CI/CD workflow
4. Allows build validation (`mkdocs build --strict`)

**Implementation**:
```yaml
# .github/workflows/docs.yaml
name: Deploy Docs
on:
  push:
    branches: [main]
    paths: ['docs/**', 'mkdocs.yml']
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.11'
      - run: pip install -r requirements-docs.txt
      - run: mkdocs build --strict
      - run: mkdocs gh-deploy --force
```

---

### 4. MkDocs Material Theme Configuration

**Question**: What configuration provides best developer experience?

**Research Findings**:

Key Material theme features to enable:
- **Instant loading**: Faster page transitions
- **Search**: Full-text search across all pages
- **Code blocks**: Syntax highlighting, copy button, line numbers
- **Navigation**: Tabs, sections, table of contents
- **Dark mode**: Toggle for user preference
- **Social cards**: Auto-generated for sharing

**Decision**: Full-featured Material configuration

**Configuration**:
```yaml
# mkdocs.yml
site_name: Data Platform Documentation
site_url: https://infoblox-cto.github.io/data-platform/
repo_url: https://github.com/Infoblox-CTO/data.platform.kit
repo_name: Infoblox-CTO/data-platform

theme:
  name: material
  features:
    - navigation.instant
    - navigation.tracking
    - navigation.tabs
    - navigation.sections
    - navigation.expand
    - navigation.top
    - search.suggest
    - search.highlight
    - content.code.copy
    - content.code.annotate
  palette:
    - scheme: default
      primary: indigo
      accent: indigo
      toggle:
        icon: material/brightness-7
        name: Switch to dark mode
    - scheme: slate
      primary: indigo
      accent: indigo
      toggle:
        icon: material/brightness-4
        name: Switch to light mode

plugins:
  - search
  - minify:
      minify_html: true

markdown_extensions:
  - pymdownx.highlight:
      anchor_linenums: true
  - pymdownx.superfences
  - pymdownx.tabbed:
      alternate_style: true
  - admonition
  - pymdownx.details
  - attr_list
  - md_in_html
  - toc:
      permalink: true
```

---

### 5. Existing Content Migration

**Question**: How to incorporate existing documentation?

**Current Files**:
- `docs/architecture.md` - 264 lines, platform architecture
- `docs/cli-reference.md` - 462 lines, CLI command reference
- `docs/testing.md` - 279 lines, testing guide

**Decision**: Reorganize with minimal content changes

**Migration Plan**:

| Current File | New Location | Changes |
|--------------|--------------|---------|
| `architecture.md` | `concepts/overview.md` | Add MkDocs frontmatter, update links |
| `cli-reference.md` | `reference/cli.md` | Add frontmatter, verify completeness |
| `testing.md` | `testing.md` (root) | Add frontmatter, link from contributing |

**Rationale**:
1. Preserve existing valuable content
2. Fit into new hierarchical structure
3. Add consistent frontmatter for navigation
4. Minimal rewriting needed

---

## Summary

| Topic | Decision | Key Rationale |
|-------|----------|---------------|
| Static Site Generator | MkDocs + Material | Best DX, wide adoption, Python-based |
| Structure | Kubernetes/Docker hybrid | Industry standard, progressive complexity |
| Deployment | GitHub Actions | Automated, CI-integrated |
| Theme Config | Full Material features | Search, navigation, code blocks, dark mode |
| Migration | Reorganize existing files | Preserve content, fit new structure |

## Dependencies

```text
# requirements-docs.txt
mkdocs>=1.5.0
mkdocs-material>=9.5.0
mkdocs-minify-plugin>=0.8.0
```

## Next Steps

1. Create `mkdocs.yml` configuration
2. Create documentation directory structure
3. Create index pages for each section
4. Migrate existing documentation
5. Write new content (getting-started, concepts, tutorials)
6. Set up GitHub Actions workflow
7. Configure GitHub Pages settings
