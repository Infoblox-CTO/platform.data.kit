#!/bin/bash
# LocalStack initialization script for CDPP local development
# Creates S3 buckets used by data pipelines

set -e

echo "Initializing LocalStack S3 buckets for CDPP..."

# Wait for LocalStack to be ready
until aws --endpoint-url=http://localhost:4566 s3 ls 2>/dev/null; do
  echo "Waiting for LocalStack S3..."
  sleep 2
done

# Create standard CDPP buckets
BUCKETS=(
  "cdpp-raw"          # Raw ingested data
  "cdpp-staging"      # Intermediate processing
  "cdpp-curated"      # Curated/processed data
  "cdpp-artifacts"    # Published package artifacts
  "cdpp-test"         # Test data for development
)

for bucket in "${BUCKETS[@]}"; do
  if aws --endpoint-url=http://localhost:4566 s3 ls "s3://${bucket}" 2>/dev/null; then
    echo "Bucket ${bucket} already exists"
  else
    aws --endpoint-url=http://localhost:4566 s3 mb "s3://${bucket}"
    echo "Created bucket: ${bucket}"
  fi
done

# Create sample directory structure in test bucket
aws --endpoint-url=http://localhost:4566 s3api put-object \
  --bucket cdpp-test \
  --key sample-data/.keep \
  --body /dev/null 2>/dev/null || true

echo "LocalStack S3 initialization complete!"
echo "Buckets available at: http://localhost:4566"
