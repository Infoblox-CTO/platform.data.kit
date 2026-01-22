#!/bin/bash
# Marquez initialization script for DP local development
# This script initializes the Marquez lineage service with default namespaces and sources.

set -e

MARQUEZ_URL="${MARQUEZ_URL:-http://localhost:5000}"
MAX_RETRIES=30
RETRY_INTERVAL=2

echo "================================================"
echo "  DP Marquez Initialization"
echo "================================================"

# Wait for Marquez to be ready
echo ""
echo "Waiting for Marquez to be ready..."
for i in $(seq 1 $MAX_RETRIES); do
    if curl -s "${MARQUEZ_URL}/api/v1/namespaces" > /dev/null 2>&1; then
        echo "✓ Marquez is ready"
        break
    fi
    if [ $i -eq $MAX_RETRIES ]; then
        echo "✗ Marquez failed to start after ${MAX_RETRIES} attempts"
        exit 1
    fi
    echo "  Waiting... (attempt $i/$MAX_RETRIES)"
    sleep $RETRY_INTERVAL
done

echo ""
echo "Creating default namespace 'dp'..."
curl -s -X PUT "${MARQUEZ_URL}/api/v1/namespaces/dp" \
    -H "Content-Type: application/json" \
    -d '{
        "ownerName": "data-platform",
        "description": "Default namespace for DP data packages"
    }' | jq -r '.name // "failed"' | xargs -I {} echo "  Created namespace: {}"

echo ""
echo "Creating development namespace 'dp-dev'..."
curl -s -X PUT "${MARQUEZ_URL}/api/v1/namespaces/dp-dev" \
    -H "Content-Type: application/json" \
    -d '{
        "ownerName": "data-platform",
        "description": "Development namespace for local testing"
    }' | jq -r '.name // "failed"' | xargs -I {} echo "  Created namespace: {}"

echo ""
echo "Creating analytics namespace 'analytics'..."
curl -s -X PUT "${MARQUEZ_URL}/api/v1/namespaces/analytics" \
    -H "Content-Type: application/json" \
    -d '{
        "ownerName": "analytics-team",
        "description": "Analytics team data packages"
    }' | jq -r '.name // "failed"' | xargs -I {} echo "  Created namespace: {}"

echo ""
echo "Creating source 'kafka-cluster'..."
curl -s -X PUT "${MARQUEZ_URL}/api/v1/sources/kafka-cluster" \
    -H "Content-Type: application/json" \
    -d '{
        "type": "KAFKA",
        "connectionUrl": "kafka://redpanda:9092",
        "description": "Local Redpanda Kafka cluster"
    }' | jq -r '.name // "failed"' | xargs -I {} echo "  Created source: {}"

echo ""
echo "Creating source 's3-lake'..."
curl -s -X PUT "${MARQUEZ_URL}/api/v1/sources/s3-lake" \
    -H "Content-Type: application/json" \
    -d '{
        "type": "S3",
        "connectionUrl": "s3://localstack:4566/dp-lake",
        "description": "Local S3 data lake (LocalStack)"
    }' | jq -r '.name // "failed"' | xargs -I {} echo "  Created source: {}"

echo ""
echo "Creating source 'postgres-warehouse'..."
curl -s -X PUT "${MARQUEZ_URL}/api/v1/sources/postgres-warehouse" \
    -H "Content-Type: application/json" \
    -d '{
        "type": "POSTGRESQL",
        "connectionUrl": "postgresql://postgres:5432/dp_warehouse",
        "description": "Local PostgreSQL data warehouse"
    }' | jq -r '.name // "failed"' | xargs -I {} echo "  Created source: {}"

echo ""
echo "================================================"
echo "  Marquez initialization complete!"
echo "================================================"
echo ""
echo "Marquez UI:  ${MARQUEZ_URL}"
echo "Lineage API: ${MARQUEZ_URL}/api/v1/lineage"
echo ""
echo "Available namespaces:"
curl -s "${MARQUEZ_URL}/api/v1/namespaces" | jq -r '.namespaces[].name' | while read ns; do
    echo "  • $ns"
done
echo ""
