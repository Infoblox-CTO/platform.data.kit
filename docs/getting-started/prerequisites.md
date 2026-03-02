---
title: Prerequisites
description: Required tools and setup before installing the DK CLI
---

# Prerequisites

Before installing the DK CLI, make sure you have the following tools installed and configured.

## Required Tools

### Go 1.22+

The DK CLI is built with Go. You'll need Go 1.22 or later to build from source.

=== "macOS (Homebrew)"

    ```bash
    brew install go
    ```

=== "Linux (apt)"

    ```bash
    # Download and install Go
    wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
    
    # Add to PATH (add to ~/.bashrc or ~/.zshrc)
    export PATH=$PATH:/usr/local/go/bin
    ```

=== "Manual Installation"

    Download from [go.dev/dl](https://go.dev/dl/) and follow the installation instructions.

Verify your installation:

```bash
go version
# Expected: go version go1.22.x ...
```

### Docker

Docker is required for local development and running pipelines.

=== "macOS"

    Install [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/).

=== "Linux"

    Install [Docker Engine](https://docs.docker.com/engine/install/) for your distribution.
    
    ```bash
    # Ubuntu example
    sudo apt-get update
    sudo apt-get install docker-ce docker-ce-cli containerd.io
    
    # Add your user to the docker group
    sudo usermod -aG docker $USER
    ```

=== "Podman (Alternative)"

    Podman can be used as a Docker alternative:
    
    ```bash
    # macOS
    brew install podman
    podman machine init
    podman machine start
    
    # Set Docker compatibility
    alias docker=podman
    ```

Verify Docker is running:

```bash
docker info
# Should show Docker version and system info
```

### kubectl (Optional)

Required only for Kubernetes operations like `dk promote` to production.

=== "macOS (Homebrew)"

    ```bash
    brew install kubectl
    ```

=== "Linux"

    ```bash
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
    ```

Verify kubectl:

```bash
kubectl version --client
```

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 4 GB | 8+ GB |
| Disk | 10 GB free | 20+ GB free |
| OS | macOS 12+, Linux (kernel 4.15+) | Latest macOS or Ubuntu LTS |

## Network Requirements

The DK CLI requires network access to:

| Destination | Purpose |
|-------------|---------|
| `github.com` | Source code, releases |
| `ghcr.io` | OCI registry for packages |
| `registry.hub.docker.com` | Base container images |

!!! tip "Behind a Proxy?"
    Configure Docker and Go to use your proxy:
    
    ```bash
    # Docker proxy
    export HTTP_PROXY=http://proxy.example.com:8080
    export HTTPS_PROXY=http://proxy.example.com:8080
    
    # Go proxy
    export GOPROXY=https://proxy.golang.org,direct
    ```

## Next Steps

Once you have all prerequisites installed, proceed to:

[Install the DK CLI →](installation.md){ .md-button .md-button--primary }
