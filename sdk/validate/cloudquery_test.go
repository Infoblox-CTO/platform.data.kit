package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

func validCloudQueryPackage() *contracts.DataPackage {
	return &contracts.DataPackage{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "DataPackage",
		Metadata: contracts.PackageMetadata{
			Name:      "my-source",
			Namespace: "data-team",
			Version:   "1.0.0",
		},
		Spec: contracts.DataPackageSpec{
			Type:        contracts.PackageTypeCloudQuery,
			Description: "A test CloudQuery source plugin",
			Owner:       "data-team",
			Runtime: &contracts.RuntimeSpec{
				Image: "my-source:latest",
			},
			CloudQuery: &contracts.CloudQuerySpec{
				Role:        contracts.CloudQueryRoleSource,
				Tables:      []string{"users", "orders"},
				GRPCPort:    7777,
				Concurrency: 10000,
			},
		},
	}
}

func TestCloudQueryValidator_Name(t *testing.T) {
	v := NewCloudQueryValidator(nil)
	if got := v.Name(); got != "cloudquery" {
		t.Errorf("Name() = %s, want cloudquery", got)
	}
}

func TestCloudQueryValidator_NilPackage(t *testing.T) {
	v := NewCloudQueryValidator(nil)
	errs := v.Validate(context.Background())
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestCloudQueryValidator_NonCloudQueryType(t *testing.T) {
	pkg := &contracts.DataPackage{
		Spec: contracts.DataPackageSpec{
			Type: contracts.PackageTypePipeline,
		},
	}
	v := NewCloudQueryValidator(pkg)
	errs := v.Validate(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for non-cloudquery type, got %d", len(errs))
	}
}

func TestCloudQueryValidator_ValidPackage(t *testing.T) {
	pkg := validCloudQueryPackage()
	v := NewCloudQueryValidator(pkg)
	errs := v.Validate(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for valid package, got %d: %v", len(errs), errs)
	}
}

func TestCloudQueryValidator_E060_MissingCloudQuerySection(t *testing.T) {
	pkg := validCloudQueryPackage()
	pkg.Spec.CloudQuery = nil

	v := NewCloudQueryValidator(pkg)
	errs := v.Validate(context.Background())

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != ErrCloudQueryRequired {
		t.Errorf("expected error code %s, got %s", ErrCloudQueryRequired, errs[0].Code)
	}
	if errs[0].Field != "spec.cloudquery" {
		t.Errorf("expected field spec.cloudquery, got %s", errs[0].Field)
	}
}

func TestCloudQueryValidator_E061_MissingRole(t *testing.T) {
	pkg := validCloudQueryPackage()
	pkg.Spec.CloudQuery.Role = ""

	v := NewCloudQueryValidator(pkg)
	errs := v.Validate(context.Background())

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != ErrCloudQueryRoleInvalid {
		t.Errorf("expected error code %s, got %s", ErrCloudQueryRoleInvalid, errs[0].Code)
	}
}

func TestCloudQueryValidator_E061_InvalidRole(t *testing.T) {
	pkg := validCloudQueryPackage()
	pkg.Spec.CloudQuery.Role = "transform"

	v := NewCloudQueryValidator(pkg)
	errs := v.Validate(context.Background())

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != ErrCloudQueryRoleInvalid {
		t.Errorf("expected error code %s, got %s", ErrCloudQueryRoleInvalid, errs[0].Code)
	}
}

func TestCloudQueryValidator_W060_DestinationNotSupported(t *testing.T) {
	pkg := validCloudQueryPackage()
	pkg.Spec.CloudQuery.Role = contracts.CloudQueryRoleDestination

	v := NewCloudQueryValidator(pkg)
	errs := v.Validate(context.Background())

	if len(errs) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(errs), errs)
	}
	if errs[0].Code != WarnCloudQueryDestNotSupported {
		t.Errorf("expected warning code %s, got %s", WarnCloudQueryDestNotSupported, errs[0].Code)
	}
	if errs[0].Severity != contracts.SeverityWarning {
		t.Errorf("expected severity warning, got %s", errs[0].Severity)
	}
}

func TestCloudQueryValidator_E062_InvalidGRPCPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{name: "valid port 7777", port: 7777, wantErr: false},
		{name: "valid port 1", port: 1, wantErr: false},
		{name: "valid port 65535", port: 65535, wantErr: false},
		{name: "zero port (default)", port: 0, wantErr: false},
		{name: "negative port", port: -1, wantErr: true},
		{name: "port too high", port: 65536, wantErr: true},
		{name: "port way too high", port: 100000, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := validCloudQueryPackage()
			pkg.Spec.CloudQuery.GRPCPort = tt.port

			v := NewCloudQueryValidator(pkg)
			errs := v.Validate(context.Background())

			hasPortErr := false
			for _, e := range errs {
				if e.Code == ErrCloudQueryGRPCPort {
					hasPortErr = true
				}
			}

			if hasPortErr != tt.wantErr {
				t.Errorf("port %d: hasPortErr = %v, want %v. Errors: %v", tt.port, hasPortErr, tt.wantErr, errs)
			}
		})
	}
}

func TestCloudQueryValidator_E063_InvalidConcurrency(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
	}{
		{name: "valid 10000", concurrency: 10000, wantErr: false},
		{name: "valid 1", concurrency: 1, wantErr: false},
		{name: "zero (default)", concurrency: 0, wantErr: false},
		{name: "negative", concurrency: -1, wantErr: true},
		{name: "negative large", concurrency: -100, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := validCloudQueryPackage()
			pkg.Spec.CloudQuery.Concurrency = tt.concurrency

			v := NewCloudQueryValidator(pkg)
			errs := v.Validate(context.Background())

			hasConcErr := false
			for _, e := range errs {
				if e.Code == ErrCloudQueryConcurrency {
					hasConcErr = true
				}
			}

			if hasConcErr != tt.wantErr {
				t.Errorf("concurrency %d: hasConcErr = %v, want %v. Errors: %v", tt.concurrency, hasConcErr, tt.wantErr, errs)
			}
		})
	}
}

func TestCloudQueryValidator_MultipleErrors(t *testing.T) {
	pkg := validCloudQueryPackage()
	pkg.Spec.CloudQuery.Role = ""
	pkg.Spec.CloudQuery.GRPCPort = -1
	pkg.Spec.CloudQuery.Concurrency = -5

	v := NewCloudQueryValidator(pkg)
	errs := v.Validate(context.Background())

	// Should have: E061 (missing role), E062 (invalid port), E063 (invalid concurrency)
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors, got %d: %v", len(errs), errs)
	}

	codes := make(map[string]bool)
	for _, e := range errs {
		codes[e.Code] = true
	}

	expected := []string{ErrCloudQueryRoleInvalid, ErrCloudQueryGRPCPort, ErrCloudQueryConcurrency}
	for _, code := range expected {
		if !codes[code] {
			t.Errorf("expected error code %s not found in errors", code)
		}
	}
}

func TestDataPackageValidator_CloudQueryType_OutputsNotRequired(t *testing.T) {
	// CloudQuery packages should NOT require outputs
	pkg := &contracts.DataPackage{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "DataPackage",
		Metadata: contracts.PackageMetadata{
			Name:      "my-source",
			Namespace: "data-team",
			Version:   "1.0.0",
		},
		Spec: contracts.DataPackageSpec{
			Type:        contracts.PackageTypeCloudQuery,
			Description: "A CloudQuery source plugin",
			Owner:       "data-team",
			Runtime: &contracts.RuntimeSpec{
				Image: "my-source:latest",
			},
			CloudQuery: &contracts.CloudQuerySpec{
				Role: contracts.CloudQueryRoleSource,
			},
		},
	}

	v := NewDataPackageValidator(pkg, "/path/to/pkg")
	errs := v.Validate(context.Background())

	for _, e := range errs {
		if e.Code == contracts.ErrCodeOutputsRequired {
			t.Error("cloudquery type should not require outputs, but got E003 error")
		}
	}
}

func TestDataPackageValidator_CloudQueryType_RuntimeRequired(t *testing.T) {
	// CloudQuery packages should require runtime
	pkg := &contracts.DataPackage{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "DataPackage",
		Metadata: contracts.PackageMetadata{
			Name:      "my-source",
			Namespace: "data-team",
			Version:   "1.0.0",
		},
		Spec: contracts.DataPackageSpec{
			Type:        contracts.PackageTypeCloudQuery,
			Description: "A CloudQuery source plugin",
			Owner:       "data-team",
			CloudQuery: &contracts.CloudQuerySpec{
				Role: contracts.CloudQueryRoleSource,
			},
			// No Runtime — should fail
		},
	}

	v := NewDataPackageValidator(pkg, "/path/to/pkg")
	errs := v.Validate(context.Background())

	hasRuntimeErr := false
	for _, e := range errs {
		if e.Code == contracts.ErrCodeRuntimeRequired {
			hasRuntimeErr = true
		}
	}

	if !hasRuntimeErr {
		t.Error("cloudquery type should require runtime, but E040 not found")
	}
}

func TestDataPackageValidator_CloudQueryType_Accepted(t *testing.T) {
	// CloudQuery should be accepted as a valid type (no E002 error)
	pkg := validCloudQueryPackage()
	v := NewDataPackageValidator(pkg, "/path/to/pkg")
	errs := v.Validate(context.Background())

	for _, e := range errs {
		if e.Code == contracts.ErrCodeInvalidPackageType {
			t.Error("cloudquery should be a valid package type, but got E002 error")
		}
	}
}
