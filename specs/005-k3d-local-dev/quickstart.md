# Quickstart: k3d Local Development Environment

**Feature**: 005-k3d-local-dev  
**Date**: January 25, 2026

## Prerequisites

Before using k3d runtime, ensure you have:

1. **Docker** (v24.0+) - Running and accessible
   ```bash
   docker info
   ```

2. **k3d** (v5.0+) - Installed and in PATH
   ```bash
   # macOS
   brew install k3d
   
   # Linux
   curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
   ```

3. **kubectl** (v1.28+) - Installed and in PATH
   ```bash
   # macOS
   brew install kubectl
   
   # Linux
   curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
   chmod +x kubectl && sudo mv kubectl /usr/local/bin/
   ```

## Quick Start

### Start the Development Environment

```bash
# Start k3d cluster with all services
dp dev up --runtime=k3d
```

Expected output:
```
Starting k3d local development stack...
✓ k3d cluster 'dp-local' created
✓ Deploying Redpanda...
✓ Deploying LocalStack...
✓ Deploying PostgreSQL...
✓ Waiting for services to become healthy...
✓ Setting up port forwards...

Local development stack is ready!

Services available at:
  • Redpanda (Kafka): localhost:19092
  • LocalStack (S3):  localhost:4566
  • PostgreSQL:       localhost:5432

kubectl context: k3d-dp-local
```

### Check Status

```bash
dp dev status --runtime=k3d
```

Expected output:
```
k3d Local Development Stack: RUNNING

Cluster: dp-local
Context: k3d-dp-local

Services:
  ✓ redpanda    Running  Healthy  localhost:19092
  ✓ localstack  Running  Healthy  localhost:4566
  ✓ postgres    Running  Healthy  localhost:5432

Port Forwards: 3 active
```

### Stop the Environment

```bash
# Stop but preserve data
dp dev down --runtime=k3d

# Stop and delete all data
dp dev down --runtime=k3d --volumes
```

## Configuration

### Set k3d as Default Runtime

Create or edit `~/.config/dp/config.yaml`:

```yaml
dev:
  runtime: k3d
```

Now `dp dev up` will use k3d by default.

### Using from Any Directory

With k3d runtime, you can run `dp dev up` from any directory - no need to be in the DP workspace:

```bash
cd /tmp/my-pipeline
dp dev up --runtime=k3d  # Works!
```

## Connecting Your Data Package

### Kafka (Redpanda)

```go
// Go example
brokers := []string{"localhost:19092"}
```

```python
# Python example
bootstrap_servers = "localhost:19092"
```

### S3 (LocalStack)

```go
// Go example
cfg := aws.Config{
    Region: "us-east-1",
    EndpointResolver: aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
        return aws.Endpoint{URL: "http://localhost:4566"}, nil
    }),
}
```

```python
# Python example
import boto3
s3 = boto3.client('s3', endpoint_url='http://localhost:4566')
```

### PostgreSQL

```go
// Go example
connStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
```

```python
# Python example
conn = psycopg2.connect(
    host="localhost",
    port=5432,
    user="postgres",
    password="postgres",
    database="postgres"
)
```

## Troubleshooting

### "k3d: command not found"

Install k3d:
```bash
brew install k3d  # macOS
# or
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
```

### "Port already in use"

Check what's using the port:
```bash
lsof -i :19092
lsof -i :4566
lsof -i :5432
```

Stop conflicting services or use Docker Compose runtime which may use different ports.

### "Cluster creation timed out"

1. Check Docker is running: `docker info`
2. Check available resources: `docker system df`
3. Try deleting and recreating: `dp dev down --runtime=k3d --volumes && dp dev up --runtime=k3d`

### "Services not healthy"

Check pod status:
```bash
kubectl --context k3d-dp-local get pods
kubectl --context k3d-dp-local describe pod <pod-name>
kubectl --context k3d-dp-local logs <pod-name>
```

## Comparison: Docker Compose vs k3d

| Feature | Docker Compose | k3d |
|---------|---------------|-----|
| Startup time | ~30s | ~90s (first), ~30s (subsequent) |
| Resource usage | Lower | Higher (runs k3s) |
| Kubernetes-native | No | Yes |
| Works from any directory | No (needs compose file) | Yes |
| Production parity | Lower | Higher |

**Recommendation**: Use k3d if you want a Kubernetes-native development experience. Use Docker Compose for faster iteration on simple pipelines.
