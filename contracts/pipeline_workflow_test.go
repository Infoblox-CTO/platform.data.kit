package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStepType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		st   StepType
		want bool
	}{
		{name: "sync", st: StepTypeSync, want: true},
		{name: "transform", st: StepTypeTransform, want: true},
		{name: "test", st: StepTypeTest, want: true},
		{name: "publish", st: StepTypePublish, want: true},
		{name: "custom", st: StepTypeCustom, want: true},
		{name: "invalid", st: StepType("invalid"), want: false},
		{name: "empty", st: StepType(""), want: false},
		{name: "SYNC uppercase", st: StepType("SYNC"), want: false},
		{name: "Transform mixed case", st: StepType("Transform"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.st.IsValid(); got != tt.want {
				t.Errorf("StepType(%q).IsValid() = %v, want %v", tt.st, got, tt.want)
			}
		})
	}
}

func TestValidStepTypes(t *testing.T) {
	types := ValidStepTypes()
	if len(types) != 5 {
		t.Errorf("ValidStepTypes() returned %d types, want 5", len(types))
	}
	expected := map[StepType]bool{
		StepTypeSync:      true,
		StepTypeTransform: true,
		StepTypeTest:      true,
		StepTypePublish:   true,
		StepTypeCustom:    true,
	}
	for _, st := range types {
		if !expected[st] {
			t.Errorf("unexpected step type: %s", st)
		}
	}
}

func TestPipelineWorkflow_YAML(t *testing.T) {
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: PipelineWorkflow
metadata:
  name: security-pipeline
  description: "Sync, transform, test, and publish security data"
steps:
  - name: sync-data
    type: sync
    input: aws-security
    output: raw-output
  - name: transform-data
    type: transform
    asset: dbt-transform
  - name: test-output
    type: test
    asset: dbt-transform
    command: ["dbt", "test"]
  - name: publish-results
    type: publish
    notify:
      channels: ["#data-alerts"]
    promote: false
`

	var pw PipelineWorkflow
	if err := yaml.Unmarshal([]byte(input), &pw); err != nil {
		t.Fatalf("failed to unmarshal pipeline workflow: %v", err)
	}

	if pw.APIVersion != "datakit.infoblox.dev/v1alpha1" {
		t.Errorf("APIVersion = %q, want %q", pw.APIVersion, "datakit.infoblox.dev/v1alpha1")
	}
	if pw.Kind != "PipelineWorkflow" {
		t.Errorf("Kind = %q, want %q", pw.Kind, "PipelineWorkflow")
	}
	if pw.Metadata.Name != "security-pipeline" {
		t.Errorf("Metadata.Name = %q, want %q", pw.Metadata.Name, "security-pipeline")
	}
	if pw.Metadata.Description != "Sync, transform, test, and publish security data" {
		t.Errorf("Metadata.Description = %q, want %q", pw.Metadata.Description, "Sync, transform, test, and publish security data")
	}
	if len(pw.Steps) != 4 {
		t.Fatalf("len(Steps) = %d, want 4", len(pw.Steps))
	}

	// Verify sync step
	s := pw.Steps[0]
	if s.Name != "sync-data" {
		t.Errorf("step[0].Name = %q, want %q", s.Name, "sync-data")
	}
	if s.Type != StepTypeSync {
		t.Errorf("step[0].Type = %q, want %q", s.Type, StepTypeSync)
	}
	if s.Input != "aws-security" {
		t.Errorf("step[0].Input = %q, want %q", s.Input, "aws-security")
	}
	if s.Output != "raw-output" {
		t.Errorf("step[0].Output = %q, want %q", s.Output, "raw-output")
	}

	// Verify transform step
	s = pw.Steps[1]
	if s.Type != StepTypeTransform {
		t.Errorf("step[1].Type = %q, want %q", s.Type, StepTypeTransform)
	}
	if s.Asset != "dbt-transform" {
		t.Errorf("step[1].Asset = %q, want %q", s.Asset, "dbt-transform")
	}

	// Verify test step
	s = pw.Steps[2]
	if s.Type != StepTypeTest {
		t.Errorf("step[2].Type = %q, want %q", s.Type, StepTypeTest)
	}
	if s.Asset != "dbt-transform" {
		t.Errorf("step[2].Asset = %q, want %q", s.Asset, "dbt-transform")
	}
	if len(s.Command) != 2 || s.Command[0] != "dbt" || s.Command[1] != "test" {
		t.Errorf("step[2].Command = %v, want [dbt test]", s.Command)
	}

	// Verify publish step
	s = pw.Steps[3]
	if s.Type != StepTypePublish {
		t.Errorf("step[3].Type = %q, want %q", s.Type, StepTypePublish)
	}
	if s.Notify == nil {
		t.Fatal("step[3].Notify is nil, want non-nil")
	}
	if len(s.Notify.Channels) != 1 || s.Notify.Channels[0] != "#data-alerts" {
		t.Errorf("step[3].Notify.Channels = %v, want [#data-alerts]", s.Notify.Channels)
	}
	if s.Promote {
		t.Error("step[3].Promote = true, want false")
	}
}

func TestPipelineWorkflow_CustomStep_YAML(t *testing.T) {
	input := `apiVersion: datakit.infoblox.dev/v1alpha1
kind: PipelineWorkflow
metadata:
  name: custom-pipeline
steps:
  - name: run-container
    type: custom
    image: my-app:latest
    command: ["/bin/sh", "-c"]
    args: ["echo hello"]
    env:
      - name: MY_VAR
        value: "my-value"
`

	var pw PipelineWorkflow
	if err := yaml.Unmarshal([]byte(input), &pw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(pw.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(pw.Steps))
	}
	s := pw.Steps[0]
	if s.Type != StepTypeCustom {
		t.Errorf("Type = %q, want %q", s.Type, StepTypeCustom)
	}
	if s.Image != "my-app:latest" {
		t.Errorf("Image = %q, want %q", s.Image, "my-app:latest")
	}
	if len(s.Command) != 2 || s.Command[0] != "/bin/sh" {
		t.Errorf("Command = %v, want [/bin/sh -c]", s.Command)
	}
	if len(s.Args) != 1 || s.Args[0] != "echo hello" {
		t.Errorf("Args = %v, want [echo hello]", s.Args)
	}
	if len(s.Env) != 1 {
		t.Fatalf("len(Env) = %d, want 1", len(s.Env))
	}
	if s.Env[0].Name != "MY_VAR" || s.Env[0].Value != "my-value" {
		t.Errorf("Env[0] = {%s, %s}, want {MY_VAR, my-value}", s.Env[0].Name, s.Env[0].Value)
	}
}

func TestStepResult_Fields(t *testing.T) {
	r := StepResult{
		Name:     "sync-data",
		Type:     StepTypeSync,
		Status:   StepStatusCompleted,
		Duration: "2.5s",
	}
	if r.Name != "sync-data" {
		t.Errorf("Name = %q, want %q", r.Name, "sync-data")
	}
	if r.Status != StepStatusCompleted {
		t.Errorf("Status = %q, want %q", r.Status, StepStatusCompleted)
	}
}

func TestPipelineRunResult_Fields(t *testing.T) {
	result := PipelineRunResult{
		PipelineName: "my-pipeline",
		Status:       StepStatusFailed,
		Steps: []StepResult{
			{Name: "step-one", Type: StepTypeSync, Status: StepStatusCompleted, Duration: "1s"},
			{Name: "step-two", Type: StepTypeTransform, Status: StepStatusFailed, Duration: "0.5s", Error: "exit code 1"},
			{Name: "step-three", Type: StepTypeTest, Status: StepStatusSkipped},
		},
		Duration:   "1.5s",
		FailedStep: "step-two",
	}
	if result.PipelineName != "my-pipeline" {
		t.Errorf("PipelineName = %q, want %q", result.PipelineName, "my-pipeline")
	}
	if result.Status != StepStatusFailed {
		t.Errorf("Status = %q, want %q", result.Status, StepStatusFailed)
	}
	if len(result.Steps) != 3 {
		t.Errorf("len(Steps) = %d, want 3", len(result.Steps))
	}
	if result.FailedStep != "step-two" {
		t.Errorf("FailedStep = %q, want %q", result.FailedStep, "step-two")
	}
}
