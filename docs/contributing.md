---
title: Contributing
description: Guidelines for contributing to the Data Platform documentation
---

# Contributing to Documentation

Thank you for your interest in improving the Data Platform documentation! This guide explains how to contribute.

## Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/Infoblox-CTO/platform.data.kit.git
   cd data-platform
   ```

2. **Install dependencies**
   ```bash
   pip install -r requirements.txt
   ```

3. **Start local preview**
   ```bash
   mkdocs serve
   ```
   Open http://127.0.0.1:8000 in your browser.

4. **Make your changes** in the `docs/` folder

5. **Submit a pull request**

## Documentation Structure

```
docs/
├── index.md                    # Homepage
├── getting-started/           # Onboarding content
│   ├── prerequisites.md
│   ├── installation.md
│   └── quickstart.md
├── concepts/                  # Conceptual guides
│   ├── overview.md
│   ├── data-packages.md
│   └── ...
├── tutorials/                 # Step-by-step tutorials
│   ├── kafka-to-s3.md
│   └── ...
├── reference/                 # Reference documentation
│   ├── cli.md
│   └── ...
├── troubleshooting/          # Help content
│   ├── common-issues.md
│   └── faq.md
└── stylesheets/              # Custom CSS
    └── extra.css
```

## Writing Guidelines

### Voice and Tone

- **Be direct**: Use active voice, imperative mood for instructions
- **Be helpful**: Anticipate questions, provide context
- **Be concise**: Favor short sentences and paragraphs
- **Be consistent**: Follow established patterns

### Good Examples

| ❌ Don't | ✅ Do |
|----------|-------|
| "It should be noted that..." | "Note:" |
| "In order to..." | "To..." |
| "You will need to run..." | "Run..." |
| "The command is executed by typing..." | "Run this command:" |

### Page Structure

Every documentation page should have:

1. **YAML frontmatter** with title and description
2. **Introduction** explaining what the page covers
3. **Content** organized with clear headings
4. **Next steps** linking to related content

```markdown
---
title: Page Title
description: Brief description for SEO
---

# Page Title

Introduction paragraph explaining what readers will learn.

## Section 1

Content...

## Section 2

Content...

## Next Steps

- [Related Page](../path/to/page.md)
```

## Content Types

### Getting Started

For onboarding new users:

- Focus on getting to "hello world" quickly
- Assume minimal prior knowledge
- Include all prerequisites
- Test instructions on a fresh environment

### Concepts

For explaining ideas:

- Start with "what" and "why"
- Use diagrams and examples
- Link to related concepts
- Avoid implementation details

### Tutorials

For teaching skills:

- One clear learning objective per tutorial
- Step-by-step with numbered steps
- Show expected output at each step
- Include complete, runnable examples
- Test the full tutorial end-to-end

### Reference

For looking up details:

- Organized for quick scanning (tables, lists)
- Complete and precise
- Every option/flag documented
- Examples for common use cases

### Troubleshooting

For solving problems:

- Symptom → Cause → Solution format
- Cover common issues first
- Include exact error messages
- Provide copy-paste solutions

## Formatting

### Headings

```markdown
# Page Title (H1) - One per page

## Section (H2)

### Subsection (H3)
```

### Code Blocks

Use fenced code blocks with language hints:

````markdown
```bash
dp init my-pipeline
```

```yaml title="dp.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
```

```python title="src/main.py"
def process():
    pass
```
````

### Admonitions

Use admonitions for callouts:

```markdown
!!! note
    Additional information.

!!! tip
    Helpful suggestion.

!!! warning
    Important caution.

!!! danger
    Critical warning.
```

### Tables

```markdown
| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Data     | Data     | Data     |
```

### Links

```markdown
# Internal links
[Installation](../getting-started/installation.md)

# External links
[OpenLineage](https://openlineage.io/)

# Anchor links
[See CLI Reference](#cli-reference)
```

## Testing Changes

### Local Preview

```bash
mkdocs serve
```

### Validate Links

```bash
mkdocs build --strict
```

This fails on broken links.

### Check Spelling

We recommend using a spell checker. Common technical terms are in the project dictionary.

## Submitting Changes

### Pull Request Process

1. Create a branch: `git checkout -b docs/my-improvement`
2. Make changes
3. Test locally: `mkdocs serve`
4. Validate: `mkdocs build --strict`
5. Commit with descriptive message
6. Push and open PR

### PR Checklist

- [ ] Changes preview correctly locally
- [ ] `mkdocs build --strict` passes
- [ ] New pages added to `mkdocs.yml` nav
- [ ] Links work correctly
- [ ] Code examples are tested
- [ ] Follows style guidelines

### Review Process

1. Automated checks run on PR
2. Documentation team reviews
3. Feedback addressed
4. Merged to main
5. Auto-deployed to GitHub Pages

## Style Reference

### Command Examples

Show the command, then the output:

```bash
dp version
```

```
dp version v0.1.0
  commit: abc1234
  built:  2025-01-22T10:00:00Z
```

### File Examples

Use title to show filename:

```yaml title="dp.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Model
```

### Placeholders

Use angle brackets for placeholders:

```bash
dp init <package-name> --kind <kind> --runtime <runtime>
```

### Variables

Use `$VARIABLE` for environment variables:

```bash
export DP_REGISTRY=$REGISTRY_URL
```

## Getting Help

- **Questions**: Open an issue with the `docs` label
- **Ideas**: Open an issue describing the improvement
- **Discussion**: Use GitHub Discussions

Thank you for contributing!
