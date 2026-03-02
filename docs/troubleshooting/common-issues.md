---
title: Common Issues
description: Solutions to common issues with the Data Platform
---

# Common Issues

This page covers common issues you may encounter and how to resolve them.

## Installation Issues

### Command Not Found: dk

**Symptom**: Running `dk` returns "command not found"

**Cause**: The dk binary is not in your PATH

**Solution**:

```bash
# Check if binary exists
ls -la bin/dk

# Add to PATH temporarily
export PATH=$PATH:$(pwd)/bin

# Or add to shell config permanently
echo 'export PATH=$PATH:/path/to/data-platform/bin' >> ~/.zshrc
source ~/.zshrc
```

### Build Fails with Go Errors

**Symptom**: `make build` fails with Go compilation errors

**Cause**: Wrong Go version or missing dependencies

**Solution**:

```bash
# Verify Go version (needs 1.22+)
go version

# Download dependencies
go mod download

# Clear module cache if needed
go clean -modcache
go mod download
```

### Permission Denied on Binary

**Symptom**: Running dk gives "permission denied"

**Solution**:

```bash
chmod +x bin/dk
./bin/dk version
```

---

## Development Stack Issues

### dk dev up Fails

**Symptom**: `dk dev up` fails to start services

**Common Causes**:

1. **Docker not running**
   ```bash
   # Check Docker
   docker info
   
   # Start Docker Desktop (macOS)
   open -a Docker
   ```

2. **Port conflicts**
   ```bash
   # Check ports in use
   lsof -i :9092  # Kafka
   lsof -i :9000  # MinIO
   lsof -i :5000  # Marquez
   
   # Kill conflicting process
   kill -9 <PID>
   ```

3. **Previous containers not cleaned up**
   ```bash
   # Force cleanup
   dk dev down --volumes
   dk dev up
   ```

### Kafka Connection Refused

**Symptom**: Pipeline can't connect to Kafka at localhost:9092

**Solutions**:

1. **Wait for Kafka to be ready**
   ```bash
   # Check health
   dk dev status
   
   # Wait and retry
   sleep 30
   dk run ./my-pipeline
   ```

2. **Check Kafka logs**
   ```bash
   kubectl --context k3d-dk-local logs -l app=redpanda
   ```

3. **Verify Kafka is accepting connections**
   ```bash
   docker exec dk-kafka kafka-broker-api-versions \
     --bootstrap-server localhost:9092
   ```

### MinIO Access Denied

**Symptom**: S3 operations fail with access denied

**Solution**:

```bash
# Verify MinIO is running
dk dev status

# Check credentials (default: minioadmin/minioadmin)
mc alias set local http://localhost:9000 minioadmin minioadmin

# Create bucket if it doesn't exist
mc mb local/my-bucket
```

### Marquez Not Showing Lineage

**Symptom**: Lineage events don't appear in Marquez UI

**Solutions**:

1. **Verify Marquez is healthy**
   ```bash
   curl http://localhost:5000/api/v1/namespaces
   ```

2. **Check environment variables**
   ```bash
   # Ensure this is set
   export OPENLINEAGE_URL=http://localhost:5000/api/v1/lineage
   ```

3. **Run pipeline and check events**
   ```bash
   # Run with debug
   DK_LOG_LEVEL=debug dk run ./my-pipeline
   ```

---

## Pipeline Issues

### dk lint Fails

**Symptom**: `dk lint` returns validation errors

**Common Errors**:

| Error | Cause | Fix |
|-------|-------|-----|
| `E001: metadata.name is required` | Missing name | Add `metadata.name` to dk.yaml |
| `E004: invalid name format` | Uppercase/special chars | Use lowercase and hyphens only |
| `E010: store not found` | Missing store reference | Add a Store manifest with the referenced name |
| `E025: pii=true requires sensitivity` | Missing classification | Add sensitivity level |

**Example fixes**:

```yaml
# Fix E001/E004 - invalid name
metadata:
  name: my-pipeline  # lowercase, hyphens only

# Fix E010 - add missing store
# store.yaml
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: local-events
spec:
  type: kafka-topic
  connection:
    brokers: localhost:9092
    topic: events

# Fix E025 - add sensitivity
outputs:
  - name: data
    classification:
      pii: true
      sensitivity: confidential  # Add this
```

### dk run Fails Immediately

**Symptom**: Pipeline starts but exits immediately

**Solutions**:

1. **Check logs**
   ```bash
   dk logs <run-id>
   ```

2. **Run with debug**
   ```bash
   DK_LOG_LEVEL=debug dk run ./my-pipeline
   ```

3. **Common causes**:
   - Missing environment variables
   - Can't connect to input sources
   - Syntax errors in pipeline code

### dk run Timeout

**Symptom**: Pipeline times out before completing

**Solution**:

```bash
# Increase timeout
dk run ./my-pipeline --timeout 60m

# Or set default in config
# ~/.dk/config.yaml
defaults:
  timeout: 60m
```

### No Data Flowing

**Symptom**: Pipeline runs but processes no records

**Debugging steps**:

1. **Check input has data**
   ```bash
   # For Kafka
   docker exec dk-kafka kafka-console-consumer \
     --bootstrap-server localhost:9092 \
     --topic user-events \
     --from-beginning \
     --max-messages 5
   ```

2. **Check consumer group offset**
   ```bash
   docker exec dk-kafka kafka-consumer-groups \
     --bootstrap-server localhost:9092 \
     --group my-consumer-group \
     --describe
   ```

3. **Check bindings**
   - Verify topic name matches
   - Check consumer group setting
   - Verify offset reset policy

---

## Publishing Issues

### Authentication Failed

**Symptom**: `dk publish` fails with authentication error

**Solution**:

```bash
# For GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin

# For Docker Hub
docker login

# Set in config
export DK_REGISTRY_USER=myuser
export DK_REGISTRY_TOKEN=mytoken
```

### Push Denied

**Symptom**: `dk publish` says push denied

**Common Causes**:

1. **No write access to registry**
   - Verify you have push permissions
   - Check repository/package visibility settings

2. **Wrong registry URL**
   ```bash
   dk publish --registry ghcr.io/correct-org
   ```

3. **Token expired**
   ```bash
   docker logout ghcr.io
   docker login ghcr.io
   ```

### Package Already Exists

**Symptom**: Can't push because version exists

**Cause**: OCI artifacts are immutable

**Solution**:

```bash
# Use a new version
dk build --tag v1.0.1
dk publish
```

---

## Promotion Issues

### PR Not Created

**Symptom**: `dk promote` doesn't create a PR

**Solutions**:

1. **Check GitHub token**
   ```bash
   # Ensure token has repo access
   export GITHUB_TOKEN=ghp_xxx
   ```

2. **Check GitOps repository**
   ```yaml
   # ~/.dk/config.yaml
   environments:
     dev:
       gitops: https://github.com/org/gitops.git  # Verify URL
   ```

3. **Check network connectivity**
   ```bash
   curl -I https://github.com
   ```

### PR Failed CI

**Symptom**: Promotion PR fails CI checks

**Solutions**:

1. Check the PR for failure details
2. Run lint locally first:
   ```bash
   dk lint --strict
   ```
3. Verify package exists in registry:
   ```bash
   dk versions my-pipeline
   ```

### Sync Failed in ArgoCD

**Symptom**: PR merged but deployment not synced

**Check**:

1. ArgoCD application status
2. Kubernetes cluster connectivity
3. Resource quotas/limits

```bash
dk status my-pipeline --env dev
dk logs my-pipeline --env dev --sync
```

---

## Performance Issues

### Pipeline Running Slowly

**Optimization strategies**:

1. **Increase resources**
   ```yaml
   # dk.yaml
   spec:
     runtime:
       resources:
         memory: "4Gi"
         cpu: "4"
   ```

2. **Increase batch size**
   ```bash
   dk run --env BATCH_SIZE=5000
   ```

3. **Check I/O bottlenecks**
   - Use local SSDs
   - Increase network throughput
   - Use compression

### High Memory Usage

**Solution**:

1. Process in smaller batches
2. Use streaming instead of loading all data
3. Increase memory limits:
   ```yaml
   resources:
     limits:
       memory: "8Gi"
   ```

---

## Registry Cache Issues (k3d Runtime)

### Cache Container Not Starting

**Symptom**: `dev-registry-cache` container fails to start

**Solutions**:

1. **Check Docker resources**
   ```bash
   docker system df
   docker system prune -f  # Clean up if low on space
   ```

2. **Check for port conflicts**
   ```bash
   lsof -i :5000
   # Kill conflicting process if found
   ```

3. **Inspect container logs**
   ```bash
   docker logs dev-registry-cache
   ```

4. **Force cache rebuild**
   ```bash
   docker rm -f dev-registry-cache
   docker volume rm dev_registry_cache
   dk dev up --runtime=k3d
   ```

### Image Pulls Still Slow

**Symptom**: Cache is running but images aren't being cached

**Solutions**:

1. **Verify cache is being used**
   ```bash
   # Check the registries.yaml was created
   cat .cache/registries.yaml
   ```

2. **Check cache hit rate**
   ```bash
   docker logs dev-registry-cache 2>&1 | grep -E "manifest|blob"
   ```

3. **Ensure cluster uses the registry config**
   ```bash
   # Delete cluster and recreate
   dk dev down --runtime=k3d
   dk dev up --runtime=k3d
   ```

### Cache Not Created in CI

**Symptom**: Cache doesn't start in CI environment

**Cause**: This is expected behavior. The registry cache is automatically skipped in CI to avoid complications.

**Detection**: The following environment variables trigger CI mode:
- `CI=true`
- `GITHUB_ACTIONS=true`
- `JENKINS_URL` (any value)

**Solution**: If you need the cache in CI, unset these variables (not recommended).

### "Network not found" Error

**Symptom**: Container fails to start with network error

**Solution**:
```bash
# Create the network manually
docker network create devcache

# Or remove and let it recreate
docker network rm devcache
dk dev up --runtime=k3d
```

---

## Lineage Issues

### Missing Upstream/Downstream

**Symptom**: Lineage graph shows orphan nodes

**Causes**:

1. **Different namespaces**
   - Packages should use the same namespace for linked lineage

2. **Binding name mismatch**
   ```yaml
   # Both packages should reference same binding
   binding: shared/user-events  # Use consistent naming
   ```

3. **Never ran successfully**
   - Only successful runs emit complete lineage

### Stale Lineage

**Symptom**: Lineage shows old data

**Solution**:

```bash
# Planned: dk lineage my-pipeline --refresh
# For now, query Marquez directly:
curl http://localhost:5000/api/v1/namespaces/default/jobs/my-pipeline/runs
```

---

## Getting More Help

If you can't resolve your issue:

1. **Check debug logs**
   ```bash
   DK_LOG_LEVEL=debug dk <command>
   ```

2. **Search existing issues**
   - [GitHub Issues](https://github.com/Infoblox-CTO/platform.data.kit/issues)

3. **Open a new issue**
   - Include: command, error message, environment details

4. **Contact the team**
   - Slack: #data-platform-support
   - Email: data-platform@example.com

---

## See Also

- [FAQ](faq.md) - Frequently asked questions
- [CLI Reference](../reference/cli.md) - Command documentation
- [Configuration](../reference/configuration.md) - Config options
