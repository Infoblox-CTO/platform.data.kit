---
title: Kafka to S3 Pipeline
description: Build a complete data pipeline that reads from Kafka and writes to S3
---

# Tutorial: Kafka to S3 Pipeline

In this tutorial, you'll build a production-ready data pipeline that reads events from Kafka, transforms them, and writes them to S3 in Parquet format.

**Prerequisites**: Complete the [Quickstart](../getting-started/quickstart.md) tutorial.

**Time**: ~30 minutes

## What You'll Learn

- Configure Kafka inputs with consumer groups
- Set up S3 outputs with partitioning
- Add data transformation logic
- Configure lineage tracking
- Deploy to an environment

## Step 1: Create the Package

Initialize a new Transform package:

```bash
dp init kafka-to-s3-pipeline --runtime generic-python
cd kafka-to-s3-pipeline
```

## Step 2: Define the Transform Manifest

Edit `dp.yaml` with your Transform configuration:

```yaml title="dp.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: kafka-to-s3-pipeline
  namespace: tutorials
  version: 1.0.0
  labels:
    team: data-engineering
    domain: events

spec:
  runtime: generic-python
  mode: batch
  image: myorg/kafka-to-s3-pipeline:v1.0.0
  timeout: 1h

  inputs:
    - asset: user-events

  outputs:
    - asset: processed-events

  env:
    - name: LOG_LEVEL
      value: info
  resources:
    cpu: "500m"
    memory: "2Gi"
```

## Step 3: Define Assets and Stores

Create the input Asset referencing a Kafka Store:

```yaml title="asset/user-events.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: user-events
  namespace: tutorials
spec:
  store: local-kafka
  topic: user-events
  format: json
```

Create the output Asset referencing an S3 Store:

```yaml title="asset/processed-events.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Asset
metadata:
  name: processed-events
  namespace: tutorials
spec:
  store: local-s3
  prefix: processed/events/
  format: parquet
  classification: internal
```

Create the Stores with connection details:

```yaml title="store/local-kafka.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: local-kafka
spec:
  connector: kafka
  connection:
    bootstrap-servers: localhost:9092
    consumer-group: kafka-to-s3-consumer
    auto-offset-reset: earliest
```

```yaml title="store/local-s3.yaml"
apiVersion: data.infoblox.com/v1alpha1
kind: Store
metadata:
  name: local-s3
spec:
  connector: s3
  connection:
    endpoint: http://localhost:9000
    region: us-east-1
  secrets:
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
```

## Step 4: Write the Pipeline Code

Create the transformation logic in `src/main.py`:

```python title="src/main.py"
#!/usr/bin/env python3
"""
Kafka to S3 Pipeline

Reads user events from Kafka, transforms them, and writes to S3.
"""

import json
import os
from datetime import datetime
from typing import Iterator

import pyarrow as pa
import pyarrow.parquet as pq
from kafka import KafkaConsumer
import boto3


def get_kafka_consumer() -> KafkaConsumer:
    """Create Kafka consumer from environment."""
    return KafkaConsumer(
        os.environ["INPUT_TOPIC"],
        bootstrap_servers=os.environ["KAFKA_BOOTSTRAP_SERVERS"],
        group_id=os.environ.get("CONSUMER_GROUP", "kafka-to-s3-consumer"),
        auto_offset_reset="earliest",
        value_deserializer=lambda x: json.loads(x.decode("utf-8")),
        consumer_timeout_ms=30000,  # 30 second timeout
    )


def get_s3_client():
    """Create S3 client from environment."""
    return boto3.client(
        "s3",
        endpoint_url=os.environ.get("S3_ENDPOINT"),
        aws_access_key_id=os.environ.get("AWS_ACCESS_KEY_ID", "minioadmin"),
        aws_secret_access_key=os.environ.get("AWS_SECRET_ACCESS_KEY", "minioadmin"),
    )


def transform_event(event: dict) -> dict:
    """Transform a single event."""
    return {
        "event_id": event.get("id"),
        "user_id": event.get("user_id"),
        "event_type": event.get("type"),
        "timestamp": event.get("timestamp"),
        "processed_at": datetime.utcnow().isoformat(),
        "properties": json.dumps(event.get("properties", {})),
    }


def batch_events(consumer: KafkaConsumer, batch_size: int) -> Iterator[list]:
    """Yield batches of events from Kafka."""
    batch = []
    for message in consumer:
        batch.append(transform_event(message.value))
        if len(batch) >= batch_size:
            yield batch
            batch = []
    if batch:
        yield batch


def write_to_s3(s3_client, bucket: str, prefix: str, events: list):
    """Write events to S3 as Parquet."""
    if not events:
        return
        
    # Create Arrow table
    table = pa.Table.from_pylist(events)
    
    # Generate partition path
    date = datetime.utcnow().strftime("%Y-%m-%d")
    timestamp = datetime.utcnow().strftime("%H%M%S")
    key = f"{prefix}date={date}/events_{timestamp}.parquet"
    
    # Write to S3
    with pa.BufferOutputStream() as buf:
        pq.write_table(table, buf, compression="snappy")
        s3_client.put_object(
            Bucket=bucket,
            Key=key,
            Body=buf.getvalue().to_pybytes(),
        )
    
    print(f"Wrote {len(events)} events to s3://{bucket}/{key}")


def main():
    """Main pipeline entry point."""
    print("Starting Kafka to S3 pipeline...")
    
    # Configuration
    batch_size = int(os.environ.get("BATCH_SIZE", "1000"))
    bucket = os.environ["OUTPUT_BUCKET"]
    prefix = os.environ.get("OUTPUT_PREFIX", "processed/events/")
    
    # Initialize clients
    consumer = get_kafka_consumer()
    s3_client = get_s3_client()
    
    # Process batches
    total_events = 0
    for batch in batch_events(consumer, batch_size):
        write_to_s3(s3_client, bucket, prefix, batch)
        total_events += len(batch)
    
    print(f"Pipeline complete. Processed {total_events} events.")


if __name__ == "__main__":
    main()
```

## Step 5: Add Dependencies

Create `requirements.txt` for Python dependencies:

```text title="src/requirements.txt"
kafka-python>=2.0.2
boto3>=1.26.0
pyarrow>=14.0.0
```

## Step 6: Start Local Development

Start the local development stack:

```bash
dp dev up
```

Verify all services are running:

```bash
dp dev status
```

## Step 7: Produce Test Data

Create a test data producer script:

```python title="scripts/produce_test_data.py"
#!/usr/bin/env python3
"""Generate test events for the pipeline."""

import json
import time
import uuid
from kafka import KafkaProducer

producer = KafkaProducer(
    bootstrap_servers="localhost:9092",
    value_serializer=lambda x: json.dumps(x).encode("utf-8"),
)

for i in range(100):
    event = {
        "id": str(uuid.uuid4()),
        "user_id": f"user_{i % 10}",
        "type": "page_view",
        "timestamp": time.time(),
        "properties": {
            "page": f"/page/{i}",
            "referrer": "https://example.com",
        },
    }
    producer.send("user-events", event)
    print(f"Sent event {i + 1}")

producer.flush()
print("Done!")
```

Run it:

```bash
python scripts/produce_test_data.py
```

## Step 8: Validate and Run

Validate your package:

```bash
dp lint
```

Run the pipeline:

```bash
dp run
```

## Step 9: Check Results

### View Lineage

<!-- dp lineage is not yet implemented -->
Open the Marquez UI at http://localhost:3000 to view the lineage graph.

### Check S3 Output

```bash
# Using MinIO CLI (mc)
mc ls local/local-bucket/processed/events/
```

Or open the MinIO console at http://localhost:9001.

### View in Marquez

Open http://localhost:5000 to see the lineage graph.

## Step 10: Build and Publish

When ready for deployment:

```bash
# Build OCI artifact
dp build --tag v1.0.0

# Publish to registry
dp publish
```

## Step 11: Promote to Environment

Deploy to the development environment:

```bash
dp promote kafka-to-s3-pipeline v1.0.0 --to dev
```

## Summary

You've built a complete Kafka to S3 pipeline with:

- [x] Kafka consumer with consumer groups
- [x] Data transformation logic
- [x] S3 output with Parquet format
- [x] Date-based partitioning
- [x] Automatic lineage tracking
- [x] Environment promotion

## Next Steps

- [Local Development](local-development.md) - Deep dive into the dev stack
- [Promoting Packages](promoting-packages.md) - Advanced promotion workflows
- [Troubleshooting](../troubleshooting/common-issues.md) - Common issues
