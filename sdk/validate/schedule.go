package validate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/adhocore/gronx"
	"gopkg.in/yaml.v3"
)

// Error codes for schedule validation.
const (
	ErrScheduleMissingRequired = "E100"
	ErrScheduleInvalidCron     = "E101"
	ErrScheduleInvalidTimezone = "E102"
	ErrScheduleInvalidMeta     = "E103"
)

const (
	scheduleAPIVersion = "data.infoblox.com/v1alpha1"
	scheduleKind       = "Schedule"
)

// ScheduleFileName is the default filename for schedule definitions.
const ScheduleFileName = "schedule.yaml"

// ScheduleValidator validates schedule.yaml manifests.
type ScheduleValidator struct {
	schedule     *contracts.ScheduleManifest
	schedulePath string
}

// NewScheduleValidator creates a validator for a ScheduleManifest.
func NewScheduleValidator(sm *contracts.ScheduleManifest, path string) *ScheduleValidator {
	return &ScheduleValidator{
		schedule:     sm,
		schedulePath: path,
	}
}

// NewScheduleValidatorFromFile creates a validator by loading from a file.
func NewScheduleValidatorFromFile(path string) (*ScheduleValidator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schedule file %s: %w", path, err)
	}

	var sm contracts.ScheduleManifest
	if err := yaml.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("failed to parse schedule file %s: %w", path, err)
	}

	return &ScheduleValidator{
		schedule:     &sm,
		schedulePath: path,
	}, nil
}

// Name returns the validator name.
func (v *ScheduleValidator) Name() string {
	return "schedule"
}

// Schedule returns the parsed ScheduleManifest.
func (v *ScheduleValidator) Schedule() *contracts.ScheduleManifest {
	return v.schedule
}

// Validate validates the ScheduleManifest.
func (v *ScheduleValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.schedule == nil {
		errs.AddError(ErrScheduleMissingRequired, "", "schedule manifest is nil")
		return errs
	}

	v.validateMeta(&errs)
	v.validateCron(&errs)
	v.validateTimezone(&errs)

	return errs
}

// validateMeta checks apiVersion and kind.
func (v *ScheduleValidator) validateMeta(errs *contracts.ValidationErrors) {
	if v.schedule.APIVersion == "" {
		errs.AddError(ErrScheduleInvalidMeta, "apiVersion", "apiVersion is required")
	} else if v.schedule.APIVersion != scheduleAPIVersion {
		errs.AddError(ErrScheduleInvalidMeta, "apiVersion", "apiVersion must be "+scheduleAPIVersion)
	}

	if v.schedule.Kind == "" {
		errs.AddError(ErrScheduleInvalidMeta, "kind", "kind is required")
	} else if v.schedule.Kind != scheduleKind {
		errs.AddError(ErrScheduleInvalidMeta, "kind", "kind must be "+scheduleKind)
	}
}

// validateCron checks the cron expression.
func (v *ScheduleValidator) validateCron(errs *contracts.ValidationErrors) {
	if v.schedule.Cron == "" {
		errs.AddError(ErrScheduleMissingRequired, "cron", "cron expression is required")
		return
	}

	g := gronx.New()
	if !g.IsValid(v.schedule.Cron) {
		errs.AddError(ErrScheduleInvalidCron, "cron", "invalid cron expression: "+v.schedule.Cron)
	}
}

// validateTimezone checks the IANA timezone identifier.
func (v *ScheduleValidator) validateTimezone(errs *contracts.ValidationErrors) {
	if v.schedule.Timezone == "" {
		return // Empty timezone defaults to UTC
	}

	if _, err := time.LoadLocation(v.schedule.Timezone); err != nil {
		errs.AddError(ErrScheduleInvalidTimezone, "timezone", "invalid timezone: "+v.schedule.Timezone)
	}
}
