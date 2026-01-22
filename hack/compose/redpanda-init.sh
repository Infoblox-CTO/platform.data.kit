#!/bin/bash
# Redpanda initialization script for DP local development
# Creates Kafka topics used by data pipelines

set -e

echo "Initializing Redpanda topics for DP..."

# Redpanda broker address (internal network)
BROKER="redpanda:9092"

# Wait for Redpanda to be fully ready
until rpk cluster health --brokers ${BROKER} 2>/dev/null | grep -q "Healthy:.*true"; do
  echo "Waiting for Redpanda cluster..."
  sleep 2
done

# Standard DP topics with appropriate configurations
declare -A TOPICS=(
  ["dp.raw.events"]="partitions=3,retention.ms=604800000"
  ["dp.processed.events"]="partitions=3,retention.ms=2592000000"
  ["dp.errors.dlq"]="partitions=1,retention.ms=2592000000"
  ["dp.audit.logs"]="partitions=1,retention.ms=7776000000"
  ["dp.test.input"]="partitions=1,retention.ms=86400000"
  ["dp.test.output"]="partitions=1,retention.ms=86400000"
)

for topic in "${!TOPICS[@]}"; do
  config="${TOPICS[$topic]}"
  partitions=$(echo "$config" | grep -oP 'partitions=\K\d+')
  retention=$(echo "$config" | grep -oP 'retention.ms=\K\d+')
  
  if rpk topic describe "$topic" --brokers ${BROKER} 2>/dev/null; then
    echo "Topic ${topic} already exists"
  else
    rpk topic create "$topic" \
      --brokers ${BROKER} \
      --partitions "${partitions}" \
      --config "retention.ms=${retention}"
    echo "Created topic: ${topic}"
  fi
done

echo "Redpanda topic initialization complete!"
echo "Console available at: http://localhost:8080"
echo "Kafka broker at: localhost:19092"
