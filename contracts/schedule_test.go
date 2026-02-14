package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestScheduleManifest_YAML(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Schedule
cron: "0 */6 * * *"
timezone: America/New_York
suspend: false
`

	var sm ScheduleManifest
	if err := yaml.Unmarshal([]byte(input), &sm); err != nil {
		t.Fatalf("failed to unmarshal schedule manifest: %v", err)
	}

	if sm.APIVersion != "data.infoblox.com/v1alpha1" {
		t.Errorf("APIVersion = %q, want %q", sm.APIVersion, "data.infoblox.com/v1alpha1")
	}
	if sm.Kind != "Schedule" {
		t.Errorf("Kind = %q, want %q", sm.Kind, "Schedule")
	}
	if sm.Cron != "0 */6 * * *" {
		t.Errorf("Cron = %q, want %q", sm.Cron, "0 */6 * * *")
	}
	if sm.Timezone != "America/New_York" {
		t.Errorf("Timezone = %q, want %q", sm.Timezone, "America/New_York")
	}
	if sm.Suspend {
		t.Error("Suspend = true, want false")
	}
}

func TestScheduleManifest_MinimalYAML(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Schedule
cron: "30 2 * * 1"
`

	var sm ScheduleManifest
	if err := yaml.Unmarshal([]byte(input), &sm); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if sm.Cron != "30 2 * * 1" {
		t.Errorf("Cron = %q, want %q", sm.Cron, "30 2 * * 1")
	}
	if sm.Timezone != "" {
		t.Errorf("Timezone = %q, want empty (default)", sm.Timezone)
	}
	if sm.Suspend {
		t.Error("Suspend = true, want false (default)")
	}
}

func TestScheduleManifest_SuspendedYAML(t *testing.T) {
	input := `apiVersion: data.infoblox.com/v1alpha1
kind: Schedule
cron: "0 0 * * *"
timezone: UTC
suspend: true
`

	var sm ScheduleManifest
	if err := yaml.Unmarshal([]byte(input), &sm); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !sm.Suspend {
		t.Error("Suspend = false, want true")
	}
}

func TestScheduleManifest_RoundTrip(t *testing.T) {
	original := ScheduleManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Schedule",
		Cron:       "0 */6 * * *",
		Timezone:   "Europe/London",
		Suspend:    true,
	}

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ScheduleManifest
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.APIVersion != original.APIVersion {
		t.Errorf("APIVersion = %q, want %q", decoded.APIVersion, original.APIVersion)
	}
	if decoded.Kind != original.Kind {
		t.Errorf("Kind = %q, want %q", decoded.Kind, original.Kind)
	}
	if decoded.Cron != original.Cron {
		t.Errorf("Cron = %q, want %q", decoded.Cron, original.Cron)
	}
	if decoded.Timezone != original.Timezone {
		t.Errorf("Timezone = %q, want %q", decoded.Timezone, original.Timezone)
	}
	if decoded.Suspend != original.Suspend {
		t.Errorf("Suspend = %v, want %v", decoded.Suspend, original.Suspend)
	}
}
