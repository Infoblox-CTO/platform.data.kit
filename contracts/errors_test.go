package contracts

import (
	"strings"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *ValidationError
		want    string
		wantSub string
	}{
		{
			name: "with field",
			err: &ValidationError{
				Code:    "E001",
				Field:   "metadata.name",
				Message: "name is required",
			},
			want:    "[E001] metadata.name: name is required",
			wantSub: "metadata.name",
		},
		{
			name: "without field",
			err: &ValidationError{
				Code:    "E002",
				Message: "general validation error",
			},
			want:    "[E002] general validation error",
			wantSub: "E002",
		},
		{
			name: "with value",
			err: &ValidationError{
				Code:    "E003",
				Field:   "spec.type",
				Message: "invalid type",
				Value:   "unknown",
			},
			want:    "[E003] spec.type: invalid type",
			wantSub: "spec.type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("Error() should contain %q", tt.wantSub)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name    string
		errs    ValidationErrors
		wantLen int
		wantSub string
	}{
		{
			name:    "empty errors",
			errs:    ValidationErrors{},
			wantLen: 0,
			wantSub: "no validation errors",
		},
		{
			name: "single error",
			errs: ValidationErrors{
				{Code: "E001", Field: "name", Message: "required"},
			},
			wantLen: 1,
			wantSub: "E001",
		},
		{
			name: "multiple errors",
			errs: ValidationErrors{
				{Code: "E001", Field: "name", Message: "required"},
				{Code: "E002", Field: "type", Message: "invalid"},
			},
			wantLen: 2,
			wantSub: "2 validation errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.Error()
			if len(tt.errs) != tt.wantLen {
				t.Errorf("len(errs) = %v, want %v", len(tt.errs), tt.wantLen)
			}
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("Error() = %q, should contain %q", got, tt.wantSub)
			}
		})
	}
}

func TestValidationErrors_HasErrors(t *testing.T) {
	tests := []struct {
		name string
		errs ValidationErrors
		want bool
	}{
		{
			name: "empty",
			errs: ValidationErrors{},
			want: false,
		},
		{
			name: "with errors",
			errs: ValidationErrors{
				{Code: "E001", Message: "error"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errs.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationErrors_Add(t *testing.T) {
	var errs ValidationErrors

	errs.Add(&ValidationError{Code: "E001", Message: "first"})
	if len(errs) != 1 {
		t.Errorf("len(errs) = %v, want 1", len(errs))
	}

	errs.Add(&ValidationError{Code: "E002", Message: "second"})
	if len(errs) != 2 {
		t.Errorf("len(errs) = %v, want 2", len(errs))
	}
}

func TestValidationErrors_AddError(t *testing.T) {
	var errs ValidationErrors

	errs.AddError("E001", "field1", "message1")
	if len(errs) != 1 {
		t.Errorf("len(errs) = %v, want 1", len(errs))
	}
	if errs[0].Code != "E001" {
		t.Errorf("Code = %v, want E001", errs[0].Code)
	}
	if errs[0].Field != "field1" {
		t.Errorf("Field = %v, want field1", errs[0].Field)
	}
}

func TestValidationErrors_AddErrorWithValue(t *testing.T) {
	var errs ValidationErrors

	errs.AddErrorWithValue("E003", "spec.type", "invalid type", "unknown")
	if len(errs) != 1 {
		t.Errorf("len(errs) = %v, want 1", len(errs))
	}
	if errs[0].Value != "unknown" {
		t.Errorf("Value = %v, want unknown", errs[0].Value)
	}
}

func TestErrorCodes(t *testing.T) {
	// Test that error code constants are defined
	codes := map[string]string{
		"ErrCodeNameNotDNSSafe":         ErrCodeNameNotDNSSafe,
		"ErrCodeInvalidPackageType":     ErrCodeInvalidPackageType,
		"ErrCodeOutputsRequired":        ErrCodeOutputsRequired,
		"ErrCodeClassificationRequired": ErrCodeClassificationRequired,
		"ErrCodeInvalidSchemaType":      ErrCodeInvalidSchemaType,
		"ErrCodeBindingNotFound":        ErrCodeBindingNotFound,
		"ErrCodeBindingTypeMismatch":    ErrCodeBindingTypeMismatch,
		"ErrCodeInvalidSemVer":          ErrCodeInvalidSemVer,
		"ErrCodeVersionAlreadyExists":   ErrCodeVersionAlreadyExists,
		"ErrCodeInvalidImageRef":        ErrCodeInvalidImageRef,
		"ErrCodeInvalidTimeout":         ErrCodeInvalidTimeout,
	}

	for name, code := range codes {
		if code == "" {
			t.Errorf("%s is empty", name)
		}
		// Error codes should start with 'E'
		if !strings.HasPrefix(code, "E") {
			t.Errorf("%s = %q should start with 'E'", name, code)
		}
	}
}
