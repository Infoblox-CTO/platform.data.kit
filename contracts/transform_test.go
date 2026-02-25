package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTransform_CloudQueryYAML(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  namespace: default
  version: 0.1.0
  labels:
    team: data-platform
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - asset: users
  outputs:
    - asset: users-parquet
  schedule:
    cron: "0 */6 * * *"
  timeout: 30m
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if tr.APIVersion != "data.infoblox.com/v1alpha1" {
		t.Errorf("APIVersion = %q", tr.APIVersion)
	}
	if tr.Kind != "Transform" {
		t.Errorf("Kind = %q", tr.Kind)
	}
	if tr.Metadata.Name != "pg-to-s3" {
		t.Errorf("Metadata.Name = %q", tr.Metadata.Name)
	}
	if tr.Metadata.Version != "0.1.0" {
		t.Errorf("Metadata.Version = %q", tr.Metadata.Version)
	}
	if tr.Spec.Runtime != RuntimeCloudQuery {
		t.Errorf("Spec.Runtime = %q, want %q", tr.Spec.Runtime, RuntimeCloudQuery)
	}
	if tr.Spec.Mode != ModeBatch {
		t.Errorf("Spec.Mode = %q, want %q", tr.Spec.Mode, ModeBatch)
	}
	if len(tr.Spec.Inputs) != 1 {
		t.Fatalf("Spec.Inputs len = %d, want 1", len(tr.Spec.Inputs))
	}
	if tr.Spec.Inputs[0].Asset != "users" {
		t.Errorf("Spec.Inputs[0].Asset = %q", tr.Spec.Inputs[0].Asset)
	}
	if len(tr.Spec.Outputs) != 1 {
		t.Fatalf("Spec.Outputs len = %d, want 1", len(tr.Spec.Outputs))
	}
	if tr.Spec.Outputs[0].Asset != "users-parquet" {
		t.Errorf("Spec.Outputs[0].Asset = %q", tr.Spec.Outputs[0].Asset)
	}
	if tr.Spec.Schedule == nil {
		t.Fatal("Spec.Schedule is nil")
	}
	if tr.Spec.Schedule.Cron != "0 */6 * * *" {
		t.Errorf("Spec.Schedule.Cron = %q", tr.Spec.Schedule.Cron)
	}
	if tr.Spec.Timeout != "30m" {
		t.Errorf("Spec.Timeout = %q", tr.Spec.Timeout)
	}

	// Round-trip
	out, err := yaml.Marshal(&tr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	var tr2 Transform
	if err := yaml.Unmarshal(out, &tr2); err != nil {
		t.Fatalf("Unmarshal round-trip failed: %v", err)
	}
	if tr2.Metadata.Name != tr.Metadata.Name {
		t.Errorf("round-trip Name mismatch")
	}
	if tr2.Spec.Inputs[0].Asset != tr.Spec.Inputs[0].Asset {
		t.Errorf("round-trip Inputs mismatch")
	}
}

func TestTransform_GenericPythonYAML(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: enrich-users
  version: 0.2.0
spec:
  runtime: generic-python
  mode: batch
  inputs:
    - asset: users-parquet
  outputs:
    - asset: users-enriched
  image: my-team/enrich-users:latest
  command:
    - python
    - main.py
  env:
    - name: LOG_LEVEL
      value: debug
  timeout: 15m
  resources:
    requests:
      cpu: 500m
      memory: 256Mi
    limits:
      cpu: "1"
      memory: 512Mi
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if tr.Spec.Runtime != RuntimeGenericPython {
		t.Errorf("Spec.Runtime = %q", tr.Spec.Runtime)
	}
	if tr.Spec.Image != "my-team/enrich-users:latest" {
		t.Errorf("Spec.Image = %q", tr.Spec.Image)
	}
	if len(tr.Spec.Command) != 2 || tr.Spec.Command[0] != "python" {
		t.Errorf("Spec.Command = %v", tr.Spec.Command)
	}
	if len(tr.Spec.Env) != 1 || tr.Spec.Env[0].Name != "LOG_LEVEL" {
		t.Errorf("Spec.Env = %v", tr.Spec.Env)
	}
	if tr.Spec.Resources == nil {
		t.Fatal("Spec.Resources is nil")
	}
}

func TestTransform_ManifestInterface(t *testing.T) {
	tr := &Transform{
		Metadata: TransformMetadata{
			Name:      "pg-to-s3",
			Namespace: "analytics",
			Version:   "1.0.0",
		},
	}
	if tr.GetKind() != KindTransform {
		t.Errorf("GetKind() = %v, want %v", tr.GetKind(), KindTransform)
	}
	if tr.GetName() != "pg-to-s3" {
		t.Errorf("GetName() = %q", tr.GetName())
	}
	if tr.GetNamespace() != "analytics" {
		t.Errorf("GetNamespace() = %q", tr.GetNamespace())
	}
	if tr.GetVersion() != "1.0.0" {
		t.Errorf("GetVersion() = %q", tr.GetVersion())
	}
}

func TestTransform_MultipleInputsOutputs(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Transform
metadata:
  name: join-data
spec:
  runtime: generic-go
  inputs:
    - asset: users
    - asset: orders
    - asset: products
  outputs:
    - asset: user-order-summary
    - asset: product-stats
  image: my-team/join-data:latest
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if len(tr.Spec.Inputs) != 3 {
		t.Errorf("Spec.Inputs len = %d, want 3", len(tr.Spec.Inputs))
	}
	if len(tr.Spec.Outputs) != 2 {
		t.Errorf("Spec.Outputs len = %d, want 2", len(tr.Spec.Outputs))
	}
}

func TestAssetRef_YAML(t *testing.T) {
	input := `asset: users-parquet`
	var ref AssetRef
	if err := yaml.Unmarshal([]byte(input), &ref); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if ref.Asset != "users-parquet" {
		t.Errorf("Asset = %q, want %q", ref.Asset, "users-parquet")
	}
}
