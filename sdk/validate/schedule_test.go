package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func TestScheduleValidator_Valid(t *testing.T) {
	sm := &contracts.ScheduleManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Schedule",
		Cron:       "0 */6 * * *",
		Timezone:   "UTC",
	}
	v := NewScheduleValidator(sm, "schedule.yaml")
	errs := v.Validate(context.Background())
	if errs.HasErrors() {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestScheduleValidator_ValidTimezones(t *testing.T) {
	timezones := []string{"UTC", "America/New_York", "Europe/London", "Asia/Tokyo", ""}
	for _, tz := range timezones {
		t.Run("timezone_"+tz, func(t *testing.T) {
			sm := &contracts.ScheduleManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Schedule",
				Cron:       "0 0 * * *",
				Timezone:   tz,
			}
			v := NewScheduleValidator(sm, "schedule.yaml")
			errs := v.Validate(context.Background())
			if errs.HasErrors() {
				t.Errorf("expected no errors for timezone %q, got %v", tz, errs)
			}
		})
	}
}

func TestScheduleValidator_NilSchedule(t *testing.T) {
	v := NewScheduleValidator(nil, "schedule.yaml")
	errs := v.Validate(context.Background())
	if !errs.HasErrors() {
		t.Error("expected error for nil schedule")
	}
}

func TestScheduleValidator_MissingCron(t *testing.T) {
	sm := &contracts.ScheduleManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Schedule",
	}
	v := NewScheduleValidator(sm, "schedule.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrScheduleMissingRequired) {
		t.Errorf("expected error code %s, got %v", ErrScheduleMissingRequired, errs)
	}
}

func TestScheduleValidator_InvalidCron(t *testing.T) {
	tests := []struct {
		name string
		cron string
	}{
		{name: "invalid expression", cron: "not-a-cron"},
		{name: "too few fields", cron: "0 *"},
		{name: "invalid minute", cron: "60 * * * *"},
		{name: "invalid hour", cron: "0 25 * * *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &contracts.ScheduleManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Schedule",
				Cron:       tt.cron,
			}
			v := NewScheduleValidator(sm, "schedule.yaml")
			errs := v.Validate(context.Background())
			if !hasErrorCode(errs, ErrScheduleInvalidCron) {
				t.Errorf("expected error code %s for cron %q, got %v", ErrScheduleInvalidCron, tt.cron, errs)
			}
		})
	}
}

func TestScheduleValidator_InvalidTimezone(t *testing.T) {
	sm := &contracts.ScheduleManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Schedule",
		Cron:       "0 0 * * *",
		Timezone:   "Invalid/Timezone",
	}
	v := NewScheduleValidator(sm, "schedule.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrScheduleInvalidTimezone) {
		t.Errorf("expected error code %s, got %v", ErrScheduleInvalidTimezone, errs)
	}
}

func TestScheduleValidator_InvalidAPIVersion(t *testing.T) {
	sm := &contracts.ScheduleManifest{
		APIVersion: "wrong/version",
		Kind:       "Schedule",
		Cron:       "0 0 * * *",
	}
	v := NewScheduleValidator(sm, "schedule.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrScheduleInvalidMeta) {
		t.Errorf("expected error code %s, got %v", ErrScheduleInvalidMeta, errs)
	}
}

func TestScheduleValidator_InvalidKind(t *testing.T) {
	sm := &contracts.ScheduleManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "WrongKind",
		Cron:       "0 0 * * *",
	}
	v := NewScheduleValidator(sm, "schedule.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrScheduleInvalidMeta) {
		t.Errorf("expected error code %s, got %v", ErrScheduleInvalidMeta, errs)
	}
}

func TestScheduleValidator_MissingAPIVersionAndKind(t *testing.T) {
	sm := &contracts.ScheduleManifest{
		Cron: "0 0 * * *",
	}
	v := NewScheduleValidator(sm, "schedule.yaml")
	errs := v.Validate(context.Background())
	if !hasErrorCode(errs, ErrScheduleInvalidMeta) {
		t.Errorf("expected error code %s, got %v", ErrScheduleInvalidMeta, errs)
	}
}

func TestScheduleValidator_Name(t *testing.T) {
	v := NewScheduleValidator(nil, "schedule.yaml")
	if v.Name() != "schedule" {
		t.Errorf("Name() = %q, want %q", v.Name(), "schedule")
	}
}

func TestScheduleValidator_ValidCronExpressions(t *testing.T) {
	crons := []string{
		"0 0 * * *",
		"*/15 * * * *",
		"0 */6 * * *",
		"30 2 * * 1",
		"0 0 1 * *",
		"0 0 * * 1-5",
	}
	for _, cron := range crons {
		t.Run(cron, func(t *testing.T) {
			sm := &contracts.ScheduleManifest{
				APIVersion: "data.infoblox.com/v1alpha1",
				Kind:       "Schedule",
				Cron:       cron,
			}
			v := NewScheduleValidator(sm, "schedule.yaml")
			errs := v.Validate(context.Background())
			if errs.HasErrors() {
				t.Errorf("expected no errors for cron %q, got %v", cron, errs)
			}
		})
	}
}
