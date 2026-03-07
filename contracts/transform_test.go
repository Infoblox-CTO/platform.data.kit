package contracts

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTransform_CloudQueryYAML(t *testing.T) {
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: pg-to-s3
  namespace: default
  version: 0.1.0
  labels:
    team: datakit
spec:
  runtime: cloudquery
  mode: batch
  inputs:
    - asset: users
  outputs:
    - asset: users-parquet
  trigger:
    policy: schedule
    schedule:
      cron: "0 */6 * * *"
  timeout: 30m
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if tr.APIVersion != "datakit.infoblox.dev/v1alpha1" {
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
	if tr.Spec.Trigger == nil {
		t.Fatal("Spec.Trigger is nil")
	}
	if tr.Spec.Trigger.Policy != TriggerPolicySchedule {
		t.Errorf("Spec.Trigger.Policy = %q, want %q", tr.Spec.Trigger.Policy, TriggerPolicySchedule)
	}
	if tr.Spec.Trigger.Schedule == nil {
		t.Fatal("Spec.Trigger.Schedule is nil")
	}
	if tr.Spec.Trigger.Schedule.Cron != "0 */6 * * *" {
		t.Errorf("Spec.Trigger.Schedule.Cron = %q", tr.Spec.Trigger.Schedule.Cron)
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
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
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
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
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

func TestTransform_CrossCellOutputs(t *testing.T) {
	// Integration test: cell-qualified AssetRef inside a full Transform YAML,
	// matching the cross-cell routing pattern from partitioning.md.
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: tenant-router
  version: 0.1.0
spec:
  runtime: generic-go
  mode: streaming
  inputs:
    - asset: raw-events
  outputs:
    - asset: tenant-a-events
      cell: cell-us-east
    - asset: tenant-b-events
      cell: cell-us-east
    - asset: tenant-c-events
      cell: cell-eu-west
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if tr.Metadata.Name != "tenant-router" {
		t.Errorf("Name = %q, want %q", tr.Metadata.Name, "tenant-router")
	}
	if tr.Spec.Runtime != RuntimeGenericGo {
		t.Errorf("Runtime = %q, want %q", tr.Spec.Runtime, RuntimeGenericGo)
	}
	if tr.Spec.Mode != ModeStreaming {
		t.Errorf("Mode = %q, want %q", tr.Spec.Mode, ModeStreaming)
	}

	// Input has no cell qualifier.
	if len(tr.Spec.Inputs) != 1 {
		t.Fatalf("Inputs len = %d, want 1", len(tr.Spec.Inputs))
	}
	if tr.Spec.Inputs[0].Cell != "" {
		t.Errorf("Input[0].Cell = %q, want empty (resolved from deployment cell)", tr.Spec.Inputs[0].Cell)
	}

	// Outputs have cell qualifiers for cross-cell routing.
	if len(tr.Spec.Outputs) != 3 {
		t.Fatalf("Outputs len = %d, want 3", len(tr.Spec.Outputs))
	}

	wantOutputs := []struct {
		asset string
		cell  string
	}{
		{"tenant-a-events", "cell-us-east"},
		{"tenant-b-events", "cell-us-east"},
		{"tenant-c-events", "cell-eu-west"},
	}
	for i, want := range wantOutputs {
		got := tr.Spec.Outputs[i]
		if got.Asset != want.asset {
			t.Errorf("Output[%d].Asset = %q, want %q", i, got.Asset, want.asset)
		}
		if got.Cell != want.cell {
			t.Errorf("Output[%d].Cell = %q, want %q", i, got.Cell, want.cell)
		}
	}

	// Round-trip: marshal and unmarshal, ensure cell fields survive.
	out, err := yaml.Marshal(&tr)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var tr2 Transform
	if err := yaml.Unmarshal(out, &tr2); err != nil {
		t.Fatalf("Round-trip unmarshal failed: %v", err)
	}

	for i, want := range wantOutputs {
		got := tr2.Spec.Outputs[i]
		if got.Cell != want.cell {
			t.Errorf("Round-trip Output[%d].Cell = %q, want %q", i, got.Cell, want.cell)
		}
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
	if ref.Cell != "" {
		t.Errorf("Cell = %q, want empty", ref.Cell)
	}
}

func TestAssetRef_YAML_WithCell(t *testing.T) {
	input := `
asset: tenant-a-events
cell: cell-us-east`
	var ref AssetRef
	if err := yaml.Unmarshal([]byte(input), &ref); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if ref.Asset != "tenant-a-events" {
		t.Errorf("Asset = %q, want %q", ref.Asset, "tenant-a-events")
	}
	if ref.Cell != "cell-us-east" {
		t.Errorf("Cell = %q, want %q", ref.Cell, "cell-us-east")
	}
}

func TestAssetRef_YAML_CellOmitted(t *testing.T) {
	// When cell is not set, it should be omitted from YAML output.
	ref := AssetRef{Asset: "users"}
	data, err := yaml.Marshal(&ref)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if strings.Contains(string(data), "cell") {
		t.Errorf("Expected cell to be omitted from YAML, got: %s", string(data))
	}
}

func TestAssetRef_YAML_WithTags(t *testing.T) {
	input := `
tags:
  domain: identity
  tier: raw
version: ">=1.0.0 <2.0.0"`
	var ref AssetRef
	if err := yaml.Unmarshal([]byte(input), &ref); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if ref.Asset != "" {
		t.Errorf("Asset = %q, want empty (tag-based ref)", ref.Asset)
	}
	if len(ref.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(ref.Tags))
	}
	if ref.Tags["domain"] != "identity" {
		t.Errorf("Tags[domain] = %q", ref.Tags["domain"])
	}
	if ref.Tags["tier"] != "raw" {
		t.Errorf("Tags[tier] = %q", ref.Tags["tier"])
	}
	if ref.Version != ">=1.0.0 <2.0.0" {
		t.Errorf("Version = %q", ref.Version)
	}
}

func TestAssetRef_YAML_TagsOmitted(t *testing.T) {
	ref := AssetRef{Asset: "users"}
	data, err := yaml.Marshal(&ref)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if strings.Contains(string(data), "tags") {
		t.Errorf("Expected tags to be omitted from YAML, got: %s", string(data))
	}
	if strings.Contains(string(data), "version") {
		t.Errorf("Expected version to be omitted from YAML, got: %s", string(data))
	}
}

func TestTransform_TriggerOnChange(t *testing.T) {
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: enrich
  version: 0.1.0
spec:
  runtime: generic-python
  mode: batch
  inputs:
    - asset: raw-events-parquet
  outputs:
    - asset: enriched-events
  image: my-team/enrich:latest
  trigger:
    policy: on-change
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if tr.Spec.Trigger == nil {
		t.Fatal("Spec.Trigger is nil")
	}
	if tr.Spec.Trigger.Policy != TriggerPolicyOnChange {
		t.Errorf("Trigger.Policy = %q, want %q", tr.Spec.Trigger.Policy, TriggerPolicyOnChange)
	}
	// Schedule should be nil for on-change.
	if tr.Spec.Trigger.Schedule != nil {
		t.Errorf("Trigger.Schedule should be nil for on-change")
	}
}

func TestTransform_TriggerSchedule(t *testing.T) {
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: aggregate
  version: 0.1.0
spec:
  runtime: dbt
  mode: batch
  inputs:
    - asset: enriched-events
  outputs:
    - asset: event-summary
  image: my-team/dbt:latest
  trigger:
    policy: schedule
    schedule:
      cron: "0 */6 * * *"
      timezone: UTC
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if tr.Spec.Trigger == nil {
		t.Fatal("Spec.Trigger is nil")
	}
	if tr.Spec.Trigger.Policy != TriggerPolicySchedule {
		t.Errorf("Trigger.Policy = %q, want %q", tr.Spec.Trigger.Policy, TriggerPolicySchedule)
	}
	if tr.Spec.Trigger.Schedule == nil {
		t.Fatal("Trigger.Schedule is nil")
	}
	if tr.Spec.Trigger.Schedule.Cron != "0 */6 * * *" {
		t.Errorf("Trigger.Schedule.Cron = %q", tr.Spec.Trigger.Schedule.Cron)
	}
	if tr.Spec.Trigger.Schedule.Timezone != "UTC" {
		t.Errorf("Trigger.Schedule.Timezone = %q", tr.Spec.Trigger.Schedule.Timezone)
	}
}

func TestTransform_TriggerComposite(t *testing.T) {
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: hybrid
spec:
  runtime: generic-go
  inputs:
    - asset: source
  outputs:
    - asset: dest
  image: my-team/hybrid:latest
  trigger:
    policy: composite
    schedule:
      cron: "0 0 * * *"
    policies:
      - on-change
      - schedule
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if tr.Spec.Trigger.Policy != TriggerPolicyComposite {
		t.Errorf("Trigger.Policy = %q, want %q", tr.Spec.Trigger.Policy, TriggerPolicyComposite)
	}
	if len(tr.Spec.Trigger.Policies) != 2 {
		t.Fatalf("Trigger.Policies len = %d, want 2", len(tr.Spec.Trigger.Policies))
	}
	if tr.Spec.Trigger.Policies[0] != TriggerPolicyOnChange {
		t.Errorf("Trigger.Policies[0] = %q", tr.Spec.Trigger.Policies[0])
	}
	if tr.Spec.Trigger.Policies[1] != TriggerPolicySchedule {
		t.Errorf("Trigger.Policies[1] = %q", tr.Spec.Trigger.Policies[1])
	}
}

func TestTriggerPolicy_IsValid(t *testing.T) {
	tests := []struct {
		policy TriggerPolicy
		want   bool
	}{
		{TriggerPolicySchedule, true},
		{TriggerPolicyOnChange, true},
		{TriggerPolicyManual, true},
		{TriggerPolicyComposite, true},
		{TriggerPolicy("invalid"), false},
		{TriggerPolicy(""), false},
	}
	for _, tt := range tests {
		if got := tt.policy.IsValid(); got != tt.want {
			t.Errorf("TriggerPolicy(%q).IsValid() = %v, want %v", tt.policy, got, tt.want)
		}
	}
}

func TestTransform_TagBasedInputs(t *testing.T) {
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: Transform
metadata:
  name: loose-coupled
spec:
  runtime: generic-go
  inputs:
    - tags:
        domain: identity
        tier: raw
      version: ">=1.0.0"
  outputs:
    - asset: processed-data
  image: my-team/processor:latest
`

	var tr Transform
	if err := yaml.Unmarshal([]byte(input), &tr); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(tr.Spec.Inputs) != 1 {
		t.Fatalf("Inputs len = %d, want 1", len(tr.Spec.Inputs))
	}
	in := tr.Spec.Inputs[0]
	if in.Asset != "" {
		t.Errorf("Input Asset = %q, want empty", in.Asset)
	}
	if len(in.Tags) != 2 {
		t.Fatalf("Input Tags len = %d, want 2", len(in.Tags))
	}
	if in.Version != ">=1.0.0" {
		t.Errorf("Input Version = %q", in.Version)
	}

	// Output is still name-based.
	if tr.Spec.Outputs[0].Asset != "processed-data" {
		t.Errorf("Output Asset = %q", tr.Spec.Outputs[0].Asset)
	}
}
