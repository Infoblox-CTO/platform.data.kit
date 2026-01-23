# Quickstart: Contributing to Documentation

**Feature**: 003-docs-getting-started  
**Date**: 2025-01-22

## Overview

This guide helps contributors get started with the documentation system quickly.

## Prerequisites

- Python 3.11+
- pip (Python package manager)
- Git

## Setup (2 minutes)

```bash
# Clone the repository (if not already done)
git clone https://github.com/Infoblox-CTO/data.platform.kit.git
cd data-platform

# Install documentation dependencies
pip install -r requirements-docs.txt

# Start the development server
mkdocs serve
```

Open http://127.0.0.1:8000 in your browser.

## Making Changes

### 1. Edit Existing Page

1. Find the page in `docs/` directory
2. Edit the markdown file
3. Save - browser auto-reloads

### 2. Add New Page

1. Create markdown file in appropriate section:
   ```bash
   touch docs/concepts/my-new-concept.md
   ```

2. Add frontmatter:
   ```markdown
   ---
   title: My New Concept
   description: A brief description for SEO
   ---

   # My New Concept

   Content here...
   ```

3. Add to navigation in `mkdocs.yml`:
   ```yaml
   nav:
     - Concepts:
       - My New Concept: concepts/my-new-concept.md
   ```

### 3. Add Code Examples

Use fenced code blocks with language specification:

````markdown
```bash
dp init my-pipeline --type pipeline
```

```yaml
apiVersion: dp.io/v1alpha1
kind: DataPackage
metadata:
  name: example
```
````

### 4. Add Admonitions (Callouts)

```markdown
!!! note
    This is a note.

!!! tip "Pro Tip"
    This is a helpful tip.

!!! warning
    This is a warning.
```

## Validation

Before committing, validate the documentation:

```bash
# Build with strict mode (catches broken links)
mkdocs build --strict

# Preview the built site
mkdocs serve
```

## File Structure

```
docs/
├── index.md                 # Homepage
├── getting-started/         # Onboarding section
├── concepts/                # Core concepts
├── tutorials/               # Step-by-step guides
├── reference/               # CLI and API reference
├── troubleshooting/         # FAQ and common issues
├── stylesheets/extra.css    # Custom styling
└── assets/images/           # Images and diagrams

mkdocs.yml                   # Configuration
requirements-docs.txt        # Python dependencies
```

## Style Guidelines

1. **Use sentence case for headings** (not Title Case)
2. **Keep paragraphs short** (3-5 sentences)
3. **Use code blocks for all commands** (with language specified)
4. **Use relative links** for internal pages (`../concepts/data-packages.md`)
5. **Add alt text to images** for accessibility
6. **Use admonitions sparingly** for important callouts

## Deployment

Documentation deploys automatically when merged to `main` branch via GitHub Actions.

Manual deployment (maintainers only):
```bash
mkdocs gh-deploy --force
```

## Common Tasks

| Task | Command |
|------|---------|
| Start dev server | `mkdocs serve` |
| Build site | `mkdocs build` |
| Validate links | `mkdocs build --strict` |
| Deploy | `mkdocs gh-deploy --force` |
| Check Python deps | `pip list \| grep mkdocs` |

## Troubleshooting

### Port already in use

```bash
mkdocs serve -a 127.0.0.1:8001
```

### Module not found

```bash
pip install -r requirements-docs.txt
```

### Build warnings

Run `mkdocs build --strict` to see all warnings. Common issues:
- Missing pages in navigation
- Broken internal links
- Missing language in code blocks
