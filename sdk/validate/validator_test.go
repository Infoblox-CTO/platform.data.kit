package validate

import (
	"testing"
)

func TestNewValidationResult(t *testing.T) {
	result := NewValidationResult()

	if !result.Valid {
		t.Errorf("Valid = false, want true")
	}
	if len(result.Errors) != 0 {
		t.Errorf("len(Errors) = %d, want 0", len(result.Errors))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("len(Warnings) = %d, want 0", len(result.Warnings))
	}
}

func TestValidationResult_AddError(t *testing.T) {
	result := NewValidationResult()

	result.AddError("E001", "field1", "error message")

	if result.Valid {
		t.Error("Valid should be false after AddError")
	}
	if len(result.Errors) != 1 {
		t.Errorf("len(Errors) = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Code != "E001" {
		t.Errorf("Error Code = %s, want E001", result.Errors[0].Code)
	}
}

func TestValidationResult_AddWarning(t *testing.T) {
	result := NewValidationResult()

	result.AddWarning("warning message")

	if !result.Valid {
		t.Error("Valid should still be true after AddWarning")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("len(Warnings) = %d, want 1", len(result.Warnings))
	}
	if result.Warnings[0] != "warning message" {
		t.Errorf("Warning = %s, want 'warning message'", result.Warnings[0])
	}
}

func TestValidationResult_Merge(t *testing.T) {
	tests := []struct {
		name       string
		base       *ValidationResult
		other      *ValidationResult
		wantValid  bool
		wantErrors int
	}{
		{
			name:       "merge nil result",
			base:       NewValidationResult(),
			other:      nil,
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "merge valid into valid",
			base: NewValidationResult(),
			other: &ValidationResult{
				Valid:    true,
				Errors:   nil,
				Warnings: []string{"warning"},
			},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "merge invalid into valid",
			base: NewValidationResult(),
			other: func() *ValidationResult {
				r := NewValidationResult()
				r.AddError("E001", "field", "error")
				return r
			}(),
			wantValid:  false,
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.base.Merge(tt.other)

			if tt.base.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", tt.base.Valid, tt.wantValid)
			}
			if len(tt.base.Errors) != tt.wantErrors {
				t.Errorf("len(Errors) = %d, want %d", len(tt.base.Errors), tt.wantErrors)
			}
		})
	}
}

func TestDefaultValidationContext(t *testing.T) {
	ctx := DefaultValidationContext("/path/to/pkg")

	if ctx.PackageDir != "/path/to/pkg" {
		t.Errorf("PackageDir = %s, want /path/to/pkg", ctx.PackageDir)
	}
	if ctx.StrictMode {
		t.Error("StrictMode should be false by default")
	}
	if ctx.SkipSchemaValidation {
		t.Error("SkipSchemaValidation should be false by default")
	}
	if !ctx.ValidatePII {
		t.Error("ValidatePII should be true by default")
	}
}

func TestErrorCodes(t *testing.T) {
	codes := map[string]string{
		"ErrMissingRequired": ErrMissingRequired,
		"ErrInvalidFormat":   ErrInvalidFormat,
		"ErrInvalidVersion":  ErrInvalidVersion,
		"ErrFileNotFound":    ErrFileNotFound,
		"ErrParseError":      ErrParseError,
		"ErrSchemaError":     ErrSchemaError,
		"ErrDuplicateName":   ErrDuplicateName,
	}

	for name, code := range codes {
		if code == "" {
			t.Errorf("%s is empty", name)
		}
		if code[0] != 'E' {
			t.Errorf("%s = %q should start with 'E'", name, code)
		}
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    string
		wantErr  bool
		wantCode string
	}{
		{
			name:    "valid value",
			field:   "name",
			value:   "myvalue",
			wantErr: false,
		},
		{
			name:     "empty value",
			field:    "name",
			value:    "",
			wantErr:  true,
			wantCode: ErrMissingRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequired(tt.field, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if err.Code != tt.wantCode {
					t.Errorf("Code = %s, want %s", err.Code, tt.wantCode)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
