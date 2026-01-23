---
title: Common Issues
description: Solutions to common issues with the Data Platform
---

# Common Issues

This page covers common issues you may encounter and how to resolve them.

## Installation Issues

### Command Not Found: dp

**Symptom**: Running `dp` returns "command not found"

**Cause**: The dp binary is not in your PATH

**Solution**:

```bash
# Check if binary exists
ls -la bin/dp

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

**Symptom**: Running dp gives "permission denied"

**Solution**:

```bash
chmod +x bin/dp
./bin/dp version
```

---

## Development Stack Issues

### dp dev up Fails

**Symptom**: `dp dev up` fails to start services

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
   dp dev down --volumes
   docker compose -p dp down --volumes --remove-orphans
   dp dev up
   ```

### Kafka Connection Refused

**Symptom**: Pipeline can't connect to Kafka at localhost:9092

**Solutions**:

1. **Wait for Kafka to be ready**
   ```bash
   # Check health
   dp dev status
   
   # Wait and retry
   sleep 30
   dp run ./my-pipeline
   ```

2. **Check Kafka logs**
   ```bash
   docker compose -p dp logs kafka
   ```

3. **Verify Kafka is accepting connections**
   ```bash
   docker exec dp-kafka kafka-broker-api-versions \
     --bootstrap-server localhost:9092
   ```

### MinIO Access Denied

**Symptom**: S3 operations fail with access denied

**Solution**:

```bash
# Verify MinIO is running
dp dev status

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
   DP_LOG_LEVEL=debug dp run ./my-pipeline
   ```

---

## Pipeline Issues

### dp lint Fails

**Symptom**: `dp lint` returns validation errors

**Common Errors**:

| Error | Cause | Fix |
|-------|-------|-----|
| `E001: metadata.name is required` | Missing name | Add `metadata.name` to dp.yaml |
| `E004: invalid name format` | Uppercase/special chars | Use lowercase and hyphens only |
| `E010: binding not found` | Missing binding | Add binding to bindings.yaml |
| `E025: pii=true requires sensitivity` | Missing classification | Add sensitivity level |

**Example fixes**:

```yaml
# Fix E001/E004 - invalid name
metadata:
  name: my-pipeline  # lowercase, hyphens only

# Fix E010 - add missing binding
# bindings.yaml
spec:
  bindings:
    input.events:
      type: kafka-topic
      ref: local/events

# Fix E025 - add sensitivity
outputs:
  - name: data
    classification:
      pii: true
      sensitivity: confidential  # Add this
```

### dp run Fails Immediately

**Symptom**: Pipeline starts but exits immediately

**Solutions**:

1. **Check logs**
   ```bash
   dp logs <run-id>
   ```

2. **Run with debug**
   ```bash
   DP_LOG_LEVEL=debug dp run ./my-pipeline
   ```

3. **Common causes**:
   - Missing environment variables
   - Can't connect to input sources
   - Syntax errors in pipeline code

### dp run Timeout

**Symptom**: Pipeline times out before completing

**Solution**:

```bash
# Increase timeout
dp run ./my-pipeline --timeout 60m

# Or set default in config
# ~/.dp/config.yaml
defaults:
  timeout: 60m
```

### No Data Flowing

**Symptom**: Pipeline runs but processes no records

**Debugging steps**:

1. **Check input has data**
   ```bash
   # For Kafka
   docker exec dp-kafka kafka-console-consumer \
     --bootstrap-server localhost:9092 \
     --topic user-events \
     --from-beginning \
     --max-messages 5
   ```

2. **Check consumer group offset**
   ```bash
   docker exec dp-kafka kafka-consumer-groups \
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

**Symptom**: `dp publish` fails with authentication error

**Solution**:

```bash
# For GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin

# For Docker Hub
docker login

# Set in config
export DP_REGISTRY_USER=myuser
export DP_REGISTRY_TOKEN=mytoken
```

### Push Denied

**Symptom**: `dp publish` says push denied

**Common Causes**:

1. **No write access to registry**
   - Verify you have push permissions
   - Check repository/package visibility settings

2. **Wrong registry URL**
   ```bash
   dp publish --registry ghcr.io/correct-org
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
dp build --tag v1.0.1
dp publish
```

---

## Promotion Issues

### PR Not Created

**Symptom**: `dp promote` doesn't create a PR

**Solutions**:

1. **Check GitHub token**
   ```bash
   # Ensure token has repo access
   export GITHUB_TOKEN=ghp_xxx
   ```

2. **Check GitOps repository**
   ```yaml
   # ~/.dp/config.yaml
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
   dp lint --strict
   ```
3. Verify package exists in registry:
   ```bash
   dp versions my-pipeline
   ```

### Sync Failed in ArgoCD

**Symptom**: PR merged but deployment not synced

**Check**:

1. ArgoCD application status
2. Kubernetes cluster connectivity
3. Resource quotas/limits

```bash
dp status my-pipeline --env dev
dp logs my-pipeline --env dev --sync
```

---

## Performance Issues

### Pipeline Running Slowly

**Optimization strategies**:

1. **Increase resources**
   ```yaml
   # dp.yaml
   spec:
     runtime:
       resources:
         memory: "4Gi"
         cpu: "4"
   ```

2. **Increase batch size**
   ```bash
   dp run --env BATCH_SIZE=5000
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
# Force refresh
dp lineage my-pipeline --refresh

# Check recent runs
curl http://localhost:5000/api/v1/namespaces/default/jobs/my-pipeline/runs
```

---

## Getting More Help

If you can't resolve your issue:

1. **Check debug logs**
   ```bash
   DP_LOG_LEVEL=debug dp <command>
   ```

2. **Search existing issues**
   - [GitHub Issues](https://github.com/Infoblox-CTO/data-platform/issues)

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
