---
title: Local Development
description: Master the local development stack for efficient pipeline development
---

# Tutorial: Local Development

This tutorial covers the local development stack in detail, including how to use it effectively for pipeline development and testing.

**Prerequisites**: Complete the [Quickstart](../getting-started/quickstart.md) tutorial.

**Time**: ~20 minutes

## What You'll Learn

- Start and configure the development stack
- Use local Kafka for testing
- Work with MinIO (S3-compatible storage)
- View lineage in Marquez
- Debug pipeline issues locally

## The Development Stack

The `dp dev` command manages a Docker Compose stack with these services:

| Service | Port | Purpose |
|---------|------|---------|
| Kafka | 9092 | Message streaming |
| MinIO | 9000/9001 | S3-compatible storage |
| Marquez | 5000 | Lineage tracking |
| PostgreSQL | 5432 | Marquez database |

## Starting the Stack

### Basic Start

```bash
dp dev up
```

This starts all services in the foreground. Press `Ctrl+C` to stop.

### Background Start

For development sessions, run in background:

```bash
dp dev up --detach
```

### With Custom Timeout

If services are slow to start:

```bash
dp dev up --timeout 120s
```

## Checking Status

View the status of all services:

```bash
dp dev status
```

Example output:

```
Local Development Stack
━━━━━━━━━━━━━━━━━━━━━━━

Service     Status    Port     Health
───────     ──────    ────     ──────
kafka       running   9092     healthy
minio       running   9000     healthy
marquez     running   5000     healthy
postgres    running   5432     healthy

Services: 4/4 running
Stack: healthy

Endpoints:
  Kafka:      localhost:9092
  MinIO API:  http://localhost:9000
  MinIO UI:   http://localhost:9001
  Marquez:    http://localhost:5000
```

## Working with Kafka

### Accessing Kafka

The local Kafka is accessible at `localhost:9092`.

### Creating Topics

Topics are auto-created by default, but you can create them explicitly:

```bash
docker exec -it dp-kafka kafka-topics \
  --bootstrap-server localhost:9092 \
  --create \
  --topic user-events \
  --partitions 3 \
  --replication-factor 1
```

### Listing Topics

```bash
docker exec -it dp-kafka kafka-topics \
  --bootstrap-server localhost:9092 \
  --list
```

### Producing Test Messages

```bash
echo '{"id": "123", "event": "test"}' | docker exec -i dp-kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic user-events
```

### Consuming Messages

```bash
docker exec -it dp-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic user-events \
  --from-beginning
```

## Working with MinIO (S3)

### Accessing MinIO

- **API**: http://localhost:9000
- **Console**: http://localhost:9001
- **Credentials**: minioadmin/minioadmin

### Using the MinIO Console

1. Open http://localhost:9001 in your browser
2. Login with `minioadmin` / `minioadmin`
3. Browse buckets and objects

### Using MinIO CLI (mc)

Install the MinIO client:

```bash
# macOS
brew install minio/stable/mc

# Linux
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc
sudo mv mc /usr/local/bin/
```

Configure for local stack:

```bash
mc alias set local http://localhost:9000 minioadmin minioadmin
```

Common operations:

```bash
# List buckets
mc ls local

# Create bucket
mc mb local/my-bucket

# Upload file
mc cp myfile.txt local/my-bucket/

# List objects
mc ls local/my-bucket/

# Download file
mc cp local/my-bucket/myfile.txt ./downloaded.txt
```

### Using AWS CLI

Configure AWS CLI for MinIO:

```bash
# Create profile
aws configure --profile local
# Access Key: minioadmin
# Secret Key: minioadmin
# Region: us-east-1

# Use with endpoint override
aws --profile local --endpoint-url http://localhost:9000 s3 ls
```

## Working with Marquez

### Viewing Lineage

Open the Marquez UI at http://localhost:5000.

The UI shows:

- **Jobs**: Data packages and their runs
- **Datasets**: Inputs and outputs
- **Lineage Graph**: Visual data flow

### Using the Marquez API

```bash
# List namespaces
curl http://localhost:5000/api/v1/namespaces

# List jobs in a namespace
curl http://localhost:5000/api/v1/namespaces/default/jobs

# Get job details
curl http://localhost:5000/api/v1/namespaces/default/jobs/my-pipeline

# List datasets
curl http://localhost:5000/api/v1/namespaces/default/datasets
```

### Lineage Events

When you run `dp run`, these events are emitted:

```json
{
  "eventType": "START",
  "eventTime": "2025-01-22T10:00:00.000Z",
  "job": {
    "namespace": "default",
    "name": "my-pipeline"
  },
  "run": {
    "runId": "run-abc123"
  },
  "inputs": [...],
  "outputs": [...]
}
```

## Running Pipelines Locally

### Basic Run

```bash
dp run ./my-pipeline
```

### With Local Store Overrides

Create environment-specific Store manifests for local development:

```yaml title="store/local-events.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: local-events
spec:
  connector: kafka
  connection:
    bootstrapServers: localhost:9092

  secrets:
    groupId: my-pipeline-dev
```

```yaml title="store/local-output.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: local-output
spec:
  connector: s3
  connection:
    bucket: test-bucket
    endpoint: http://localhost:9000
    region: us-east-1
  secrets:
    accessKeyId: minioadmin
    secretAccessKey: minioadmin
```

```bash
dp run ./my-pipeline
```

### With Environment Variables

```bash
dp run ./my-pipeline \
  --env DEBUG=true \
  --env BATCH_SIZE=100
```

### Dry Run

See what would run without executing:

```bash
dp run ./my-pipeline --dry-run
```

## Debugging

### View Container Logs

```bash
# All services
docker compose -p dp logs -f

# Specific service
docker compose -p dp logs -f kafka
docker compose -p dp logs -f marquez
```

### Access Container Shell

```bash
docker exec -it dp-kafka /bin/bash
```

### Check Resource Usage

```bash
docker stats
```

### Common Issues

#### Kafka Not Connecting

```bash
# Check if Kafka is healthy
docker exec dp-kafka kafka-broker-api-versions \
  --bootstrap-server localhost:9092

# Check Kafka logs
docker compose -p dp logs kafka
```

#### MinIO Access Denied

```bash
# Verify credentials
mc admin info local

# Check bucket policy
mc anonymous get local/my-bucket
```

#### Marquez Events Missing

```bash
# Check Marquez health
curl http://localhost:5000/api/v1/namespaces

# Check recent runs
curl http://localhost:5000/api/v1/namespaces/default/jobs
```

## Customizing the Stack

### Override Configuration

Create `.dp/docker-compose.override.yaml`:

```yaml
version: '3.8'

services:
  kafka:
    environment:
      KAFKA_HEAP_OPTS: "-Xmx2G -Xms2G"
      
  # Add custom service
  redis:
    image: redis:7
    ports:
      - "6379:6379"
```

### Using Different Ports

```yaml
version: '3.8'

services:
  kafka:
    ports:
      - "19092:9092"  # Use port 19092 on host
```

## Stopping the Stack

### Stop and Keep Data

```bash
dp dev down
```

Data persists in Docker volumes.

### Stop and Remove Data

```bash
dp dev down --volumes
```

Removes all data (topics, objects, lineage).

## Best Practices

### 1. Use Local Store Configurations

Create environment-specific Store manifests for development:

```bash
# Development — uses local store manifests
dp run

# Production builds use environment-specific stores
dp build && dp publish
```

### 2. Clean Up Regularly

Remove old data when testing:

```bash
dp dev down --volumes
dp dev up
```

### 3. Separate Test Data

Create separate topics/buckets for testing:

```bash
mc mb local/test-data
mc mb local/prod-data
```

### 4. Use Lineage for Debugging

Check Marquez to verify:

- Pipeline ran successfully
- Correct inputs were consumed
- Outputs were produced

## Summary

You've learned how to:

- [x] Start and manage the development stack
- [x] Work with local Kafka
- [x] Use MinIO for S3 storage
- [x] View lineage in Marquez
- [x] Debug common issues
- [x] Customize the stack

## Next Steps

- [Kafka to S3](kafka-to-s3.md) - Build a complete pipeline
- [Promoting Packages](promoting-packages.md) - Deploy to environments
- [Troubleshooting](../troubleshooting/common-issues.md) - Common issues
