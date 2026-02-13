package validate

import (
	"context"
	"fmt"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// CloudQuery validation error codes.
const (
	ErrCloudQueryRequired          = "E060"
	ErrCloudQueryRoleInvalid       = "E061"
	ErrCloudQueryGRPCPort          = "E062"
	ErrCloudQueryConcurrency       = "E063"
	WarnCloudQueryDestNotSupported = "W060"
)

// CloudQueryValidator validates CloudQuery-specific fields in a DataPackage manifest.
type CloudQueryValidator struct {
	pkg *contracts.DataPackage
}

// NewCloudQueryValidator creates a validator for CloudQuery packages.
func NewCloudQueryValidator(pkg *contracts.DataPackage) *CloudQueryValidator {
	return &CloudQueryValidator{pkg: pkg}
}

// Name returns the validator name.
func (v *CloudQueryValidator) Name() string {
	return "cloudquery"
}

// Validate validates CloudQuery-specific fields.
func (v *CloudQueryValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.pkg == nil {
		errs.AddError(ErrMissingRequired, "", "datapackage is nil")
		return errs
	}

	spec := v.pkg.Spec

	// Only validate cloudquery fields if the type is cloudquery
	if spec.Type != contracts.PackageTypeCloudQuery {
		return errs
	}

	// E060: cloudquery section is required
	if spec.CloudQuery == nil {
		errs.AddError(ErrCloudQueryRequired, "spec.cloudquery", "spec.cloudquery is required when type is cloudquery")
		return errs
	}

	cq := spec.CloudQuery

	// E061: role is required and must be valid
	if cq.Role == "" {
		errs.AddError(ErrCloudQueryRoleInvalid, "spec.cloudquery.role", "spec.cloudquery.role is required")
	} else if !cq.Role.IsValid() {
		errs.AddError(ErrCloudQueryRoleInvalid, "spec.cloudquery.role",
			fmt.Sprintf("spec.cloudquery.role must be 'source' or 'destination', got '%s'", cq.Role))
	} else if !cq.Role.IsSupported() {
		// W060: destination is recognized but not yet supported
		errs.AddWarning(WarnCloudQueryDestNotSupported, "spec.cloudquery.role",
			"destination plugins are not yet supported; only 'source' is currently available")
	}

	// E062: grpcPort must be in valid range if specified
	if cq.GRPCPort != 0 && (cq.GRPCPort < 1 || cq.GRPCPort > 65535) {
		errs.AddError(ErrCloudQueryGRPCPort, "spec.cloudquery.grpcPort",
			fmt.Sprintf("spec.cloudquery.grpcPort must be between 1 and 65535, got %d", cq.GRPCPort))
	}

	// E063: concurrency must be > 0 if specified
	if cq.Concurrency < 0 {
		errs.AddError(ErrCloudQueryConcurrency, "spec.cloudquery.concurrency",
			fmt.Sprintf("spec.cloudquery.concurrency must be greater than 0, got %d", cq.Concurrency))
	}

	return errs
}
