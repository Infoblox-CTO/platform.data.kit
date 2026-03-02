---
title: Installation
description: Install the DK CLI on macOS and Linux
---

# Installation

This guide covers how to install the `dk` CLI on your machine.

## Quick Install

### From Source (Recommended)

Build the latest version from source:

```bash
# Clone the repository
git clone https://github.com/Infoblox-CTO/platform.data.kit.git
cd data-platform

# Build and install the CLI (installs to ~/go/bin by default)
make build-cli && make install

# Verify dk is on your PATH
which dk
dk version
```

### Add to PATH

By default `make install` places the binary in `~/go/bin`. Ensure that
directory is on your PATH:

=== "bash"

    ```bash
    echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
    source ~/.bashrc
    ```

=== "zsh"

    ```bash
    echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc
    source ~/.zshrc
    ```

To install elsewhere, override `DESTDIR`:

```bash
make install DESTDIR=/usr/local/bin
```

## Verify Installation

Confirm the CLI is installed correctly:

```bash
dk version
```

Expected output:

```
dk version v0.1.0
  commit: abc1234
  built:  2025-01-22T10:00:00Z
  go:     go1.22.0
```

## Shell Completion

Enable tab completion for your shell:

=== "Bash"

    ```bash
    # Add to ~/.bashrc
    source <(dk completion bash)
    ```

=== "Zsh"

    ```bash
    # Add to ~/.zshrc
    source <(dk completion zsh)
    ```

=== "Fish"

    ```bash
    dk completion fish | source
    ```

## Configuration

### Default Settings

The CLI uses sensible defaults, but you can customize behavior with environment variables:

```bash
# Set default output format
export DK_OUTPUT_FORMAT=json

# Set log level
export DK_LOG_LEVEL=debug

# Set default registry
export DK_REGISTRY=ghcr.io/my-org
```

### Configuration File (Optional)

Create a configuration file at `~/.dk/config.yaml`:

```yaml
# ~/.dk/config.yaml
registry: ghcr.io/my-org
namespace: my-team
output: table
log_level: info
```

## Troubleshooting

### Command Not Found

If you get `command not found: dk`:

1. Re-run the install: `make install`
2. Verify it was installed: `ls ~/go/bin/dk`
3. Check your PATH includes `~/go/bin`: `echo $PATH`
4. Or install to a directory already on your PATH: `make install DESTDIR=/usr/local/bin`

### Build Errors

If `make build` fails:

1. Verify Go version: `go version` (requires 1.22+)
2. Update dependencies: `go mod download`
3. Check for Go environment issues: `go env`

### Permission Denied

If you get permission errors:

```bash
# Install to a user-writable location (default)
make install

# Or fix permissions on the installed binary
chmod +x ~/go/bin/dk
```

## Upgrading

To upgrade to the latest version:

```bash
cd data-platform

# Pull latest changes
git pull origin main

# Rebuild and reinstall
make build && make install

# Verify new version
dk version
```

## Uninstalling

To remove the DK CLI:

```bash
# Remove the binary
rm "$(which dk)"

# Remove configuration (optional)
rm -rf ~/.dk
```

## Next Steps

Now that the CLI is installed, run through the quickstart:

[Start the Quickstart →](quickstart.md){ .md-button .md-button--primary }
