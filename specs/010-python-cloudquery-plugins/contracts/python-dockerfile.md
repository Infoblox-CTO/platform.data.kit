# Contract: Python Dockerfile Template

**Feature**: 010-python-cloudquery-plugins
**Type**: Generated Dockerfile specification

## Overview

The CLI generates a Dockerfile for Python CloudQuery plugins at build time. This contract defines the exact Dockerfile content that `cloudQueryDockerfile("python", grpcPort)` must produce.

## Dockerfile Specification

```dockerfile
# syntax=docker/dockerfile:1
# Build stage — cached pip installs via buildx cache mount
FROM python:3.11-slim AS builder
WORKDIR /app
COPY requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --target=/deps -r requirements.txt
COPY . .

# Runtime stage — distroless for minimal attack surface
FROM gcr.io/distroless/python3-debian12:nonroot
WORKDIR /app
COPY --from=builder /deps /usr/local/lib/python3.11/site-packages
COPY --from=builder /app /app
ENV PYTHONPATH=/usr/local/lib/python3.11/site-packages
EXPOSE {grpcPort}
ENTRYPOINT ["python3", "main.py", "serve", "--address", "[::]:{grpcPort}"]
```

## Key Properties

| Property | Value | Rationale |
|----------|-------|-----------|
| Build base | `python:3.11-slim` | Must match distroless runtime Python 3.11 (grpcio ABI requirement) |
| Runtime base | `gcr.io/distroless/python3-debian12:nonroot` | Minimal attack surface, Python 3.11 |
| Pip target | `/deps` | Standalone deps dir, copied to runtime |
| Runtime site-packages | `/usr/local/lib/python3.11/site-packages` | Matches distroless Python 3.11 |
| Cache mount | `/root/.cache/pip` | Fast rebuilds |
| gRPC address | `[::]:7777` | IPv4+IPv6, matches SDK `--address` flag |
| User | nonroot | distroless security default |

## Notes

- The build and runtime stages both use Python 3.11 to match `gcr.io/distroless/python3-debian12`. This is required because packages like `grpcio` contain Cython-compiled extensions that are ABI-specific per Python minor version — building with a different minor version than the runtime causes `ImportError`.
- The `.dockerignore` patterns from `ensureDockerignore()` exclude `cq-sync-output/`, `*.log`, `.env`, `.env.*` from the build context.
