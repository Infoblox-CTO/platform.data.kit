---
title: Troubleshooting
description: Common issues and solutions for DataKit
---

# Troubleshooting

Having problems? This section helps you diagnose and resolve common issues with DataKit.

## Quick Help

<div class="grid" markdown>

<div class="card" markdown>
### :wrench: Common Issues
Solutions for frequently encountered problems.

[View Common Issues →](common-issues.md)
</div>

<div class="card" markdown>
### :question: FAQ
Answers to frequently asked questions.

[Browse FAQ →](faq.md)
</div>

</div>

## Getting Help

If you can't find a solution here:

1. **Search existing issues**: [GitHub Issues](https://github.com/Infoblox-CTO/platform.data.kit/issues)
2. **Check the logs**: Run commands with `--log-level debug`
3. **Validate your setup**: Run `dk version` and `dk dev status`
4. **Open a new issue**: Include error messages and environment details

## Quick Diagnostics

Run these commands to gather diagnostic information:

```bash
# Check DK CLI version
dk version

# Verify Docker is running
docker info

# Check local dev stack status
dk dev status

# Validate a package
dk lint ./my-package

# Run with debug logging
dk --log-level debug run ./my-package
```

## Common Error Categories

| Category | Description | Where to Look |
|----------|-------------|---------------|
| Installation | CLI won't install or run | [Common Issues](common-issues.md#installation-issues) |
| Docker | Container problems | [Common Issues](common-issues.md#development-stack-issues) |
| Validation | Lint or schema errors | [Common Issues](common-issues.md#pipeline-issues) |
| Publishing | Registry or auth issues | [Common Issues](common-issues.md#publishing-issues) |
| Runtime | Pipeline execution failures | [Common Issues](common-issues.md#pipeline-issues) |

## Reporting Bugs

When reporting a bug, please include:

- DK CLI version (`dk version`)
- Operating system and version
- Docker version (`docker version`)
- Complete error message
- Steps to reproduce
- Relevant configuration files (with secrets redacted)
