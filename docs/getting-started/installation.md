---
title: Installation
description: Install the DP CLI on macOS and Linux
---

# Installation

This guide covers how to install the `dp` CLI on your machine.

## Quick Install

### From Source (Recommended)

Build the latest version from source:

```bash
# Clone the repository
git clone https://github.com/Infoblox-CTO/data-platform.git
cd data-platform

# Build the CLI
make build

# The binary is created in bin/dp
./bin/dp version
```

### Add to PATH

Add the `dp` binary to your PATH for easy access:

=== "Temporary (current session)"

    ```bash
    export PATH=$PATH:$(pwd)/bin
    ```

=== "Permanent (bash)"

    ```bash
    echo 'export PATH=$PATH:/path/to/data-platform/bin' >> ~/.bashrc
    source ~/.bashrc
    ```

=== "Permanent (zsh)"

    ```bash
    echo 'export PATH=$PATH:/path/to/data-platform/bin' >> ~/.zshrc
    source ~/.zshrc
    ```

## Verify Installation

Confirm the CLI is installed correctly:

```bash
dp version
```

Expected output:

```
dp version v0.1.0
  commit: abc1234
  built:  2025-01-22T10:00:00Z
  go:     go1.22.0
```

## Shell Completion

Enable tab completion for your shell:

=== "Bash"

    ```bash
    # Add to ~/.bashrc
    source <(dp completion bash)
    ```

=== "Zsh"

    ```bash
    # Add to ~/.zshrc
    source <(dp completion zsh)
    ```

=== "Fish"

    ```bash
    dp completion fish | source
    ```

## Configuration

### Default Settings

The CLI uses sensible defaults, but you can customize behavior with environment variables:

```bash
# Set default output format
export DP_OUTPUT_FORMAT=json

# Set log level
export DP_LOG_LEVEL=debug

# Set default registry
export DP_REGISTRY=ghcr.io/my-org
```

### Configuration File (Optional)

Create a configuration file at `~/.dp/config.yaml`:

```yaml
# ~/.dp/config.yaml
registry: ghcr.io/my-org
namespace: my-team
output: table
log_level: info
```

## Troubleshooting

### Command Not Found

If you get `command not found: dp`:

1. Verify the binary exists: `ls -la bin/dp`
2. Check your PATH: `echo $PATH`
3. Ensure the binary is executable: `chmod +x bin/dp`

### Build Errors

If `make build` fails:

1. Verify Go version: `go version` (requires 1.22+)
2. Update dependencies: `go mod download`
3. Check for Go environment issues: `go env`

### Permission Denied

If you get permission errors:

```bash
# Make the binary executable
chmod +x bin/dp

# Or run with explicit path
./bin/dp version
```

## Upgrading

To upgrade to the latest version:

```bash
cd data-platform

# Pull latest changes
git pull origin main

# Rebuild
make build

# Verify new version
dp version
```

## Uninstalling

To remove the DP CLI:

```bash
# Remove the binary
rm /path/to/data-platform/bin/dp

# Remove configuration (optional)
rm -rf ~/.dp

# Remove from PATH (edit ~/.bashrc or ~/.zshrc)
```

## Next Steps

Now that the CLI is installed, run through the quickstart:

[Start the Quickstart →](quickstart.md){ .md-button .md-button--primary }
