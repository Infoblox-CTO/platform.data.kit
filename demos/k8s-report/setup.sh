#!/usr/bin/env bash
# Setup script for the k8s-report demo pipeline.
# Prerequisites: dk dev up (k3d cluster running)
set -euo pipefail

CONTEXT="k3d-dk-local"
NAMESPACE="dk-local"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> Checking k3d cluster..."
if ! kubectl --context "$CONTEXT" get ns "$NAMESPACE" &>/dev/null; then
  echo "ERROR: k3d cluster not running. Run 'dk dev up' first."
  exit 1
fi
echo "    Cluster ready."

echo "==> Applying RBAC (ServiceAccount dk-k8s-reader)..."
kubectl --context "$CONTEXT" apply -f "$SCRIPT_DIR/k8s/rbac.yaml"

echo "==> Creating S3 bucket dk-k8s-report..."
aws --endpoint-url http://localhost:4566 s3 mb s3://dk-k8s-report 2>/dev/null || echo "    Bucket already exists."

echo "==> Creating PostgreSQL reporting schema..."
PG_POD=$(kubectl --context "$CONTEXT" -n "$NAMESPACE" get pod \
  -l app.kubernetes.io/name=postgresql -o jsonpath='{.items[0].metadata.name}')
kubectl --context "$CONTEXT" -n "$NAMESPACE" exec "$PG_POD" -- \
  psql -U dkuser -d datakit -c "CREATE SCHEMA IF NOT EXISTS reporting;"

echo ""
echo "Setup complete. Run the pipeline:"
echo "  dk run demos/k8s-report/transforms/k8s-collector"
echo "  dk run demos/k8s-report/transforms/s3-to-postgres"
