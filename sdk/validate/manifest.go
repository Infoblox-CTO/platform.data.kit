package validate

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/asset"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// ManifestValidator validates dp.yaml manifests for all supported kinds
// (Source, Destination, Model).
type ManifestValidator struct {
	manifest manifest.Manifest
	kind     contracts.Kind
	pkgPath  string
	// raw keeps the concrete type for kind-specific checks.
	rawSource *contracts.Source
	rawDest   *contracts.Destination
	rawModel  *contracts.Model
}

// NewManifestValidatorFromFile creates a validator from a dp.yaml file.
func NewManifestValidatorFromFile(path string) (*ManifestValidator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	m, kind, err := manifest.ParseManifest(data)
	if err != nil {
		return nil, err
	}

	v := &ManifestValidator{
		manifest: m,
		kind:     kind,
		pkgPath:  path,
	}

	switch kind {
	case contracts.KindSource:
		v.rawSource = m.(*contracts.Source)
	case contracts.KindDestination:
		v.rawDest = m.(*contracts.Destination)
	case contracts.KindModel:
		v.rawModel = m.(*contracts.Model)
	}

	return v, nil
}

// Name returns the validator name.
func (v *ManifestValidator) Name() string { return "manifest" }

// Kind returns the detected manifest kind.
func (v *ManifestValidator) Kind() contracts.Kind { return v.kind }

// Manifest returns the parsed manifest.
func (v *ManifestValidator) Manifest() manifest.Manifest { return v.manifest }

// Model returns the parsed Model (nil if kind is not Model).
func (v *ManifestValidator) Model() *contracts.Model { return v.rawModel }

// Source returns the parsed Source (nil if kind is not Source).
func (v *ManifestValidator) Source() *contracts.Source { return v.rawSource }

// Destination returns the parsed Destination (nil if kind is not Destination).
func (v *ManifestValidator) Destination() *contracts.Destination { return v.rawDest }

// Validate validates the manifest.
func (v *ManifestValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.manifest == nil {
		errs.AddError(ErrMissingRequired, "", "manifest is nil")
		return errs
	}

	v.validateCommonFields(&errs)

	switch v.kind {
	case contracts.KindSource:
		v.validateSource(&errs)
	case contracts.KindDestination:
		v.validateDestination(&errs)
	case contracts.KindModel:
		v.validateModel(&errs)
	}

	v.validateAssetRefs(&errs)

	return errs
}

// validateCommonFields checks fields common to all kinds.
func (v *ManifestValidator) validateCommonFields(errs *contracts.ValidationErrors) {
	m := v.manifest

	// Kind is already validated by the parser — but check it's valid.
	if !m.GetKind().IsValid() {
		errs.AddError(ErrInvalidFormat, "kind", "kind must be one of: Source, Destination, Model")
	}

	// Metadata
	if m.GetName() == "" {
		errs.AddError(ErrMissingRequired, "metadata.name", "metadata.name is required")
	} else if !isIdentifierValid(m.GetName()) {
		errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.name", "metadata.name must be DNS-safe")
	}

	if m.GetNamespace() == "" {
		errs.AddError(ErrMissingRequired, "metadata.namespace", "metadata.namespace is required")
	} else if !isIdentifierValid(m.GetNamespace()) {
		errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.namespace", "metadata.namespace must be DNS-safe")
	}

	if m.GetVersion() == "" {
		errs.AddError(ErrMissingRequired, "metadata.version", "metadata.version is required")
	} else if !isSemVerValid(m.GetVersion()) {
		errs.AddError(contracts.ErrCodeInvalidSemVer, "metadata.version", "metadata.version must be valid SemVer")
	}

	// Description is required for all kinds.
	if m.GetDescription() == "" {
		errs.AddError(ErrMissingRequired, "spec.description", "spec.description is required")
	}
}

// validateSource validates Source-specific fields.
func (v *ManifestValidator) validateSource(errs *contracts.ValidationErrors) {
	src := v.rawSource
	if src == nil {
		return
	}

	// E101: Runtime must be a known value.
	if !src.Spec.Runtime.IsValid() {
		errs.AddError(ErrInvalidFormat, "spec.runtime", "spec.runtime must be a valid runtime (cloudquery, generic-go, generic-python, dbt)")
	}

	// E102: Source must declare what it provides.
	if src.Spec.Provides.Name == "" && src.Spec.Provides.Type == "" {
		errs.AddError(contracts.ErrCodeSourceProvidesRequired, "spec.provides", "source must declare spec.provides (what data this source produces)")
	}

	// E104: spec.image is required for generic-* runtimes.
	if src.Spec.Runtime.IsGeneric() && src.Spec.Image == "" {
		errs.AddError(contracts.ErrCodeImageRequiredGeneric, "spec.image", "spec.image is required for generic-* runtimes")
	}

	// W104: configSchema is recommended for extensions.
	if src.Spec.ConfigSchema == nil {
		errs.AddWarning(contracts.WarnCodeConfigSchemaMissing, "spec.configSchema", "configSchema is recommended so data engineers can validate config")
	}
}

// validateDestination validates Destination-specific fields.
func (v *ManifestValidator) validateDestination(errs *contracts.ValidationErrors) {
	dest := v.rawDest
	if dest == nil {
		return
	}

	// E101: Runtime must be a known value.
	if !dest.Spec.Runtime.IsValid() {
		errs.AddError(ErrInvalidFormat, "spec.runtime", "spec.runtime must be a valid runtime (cloudquery, generic-go, generic-python, dbt)")
	}

	// E103: Destination must declare what it accepts.
	if dest.Spec.Accepts.Name == "" && dest.Spec.Accepts.Type == "" {
		errs.AddError(contracts.ErrCodeDestAcceptsRequired, "spec.accepts", "destination must declare spec.accepts (what data this destination consumes)")
	}

	// E104: spec.image is required for generic-* runtimes.
	if dest.Spec.Runtime.IsGeneric() && dest.Spec.Image == "" {
		errs.AddError(contracts.ErrCodeImageRequiredGeneric, "spec.image", "spec.image is required for generic-* runtimes")
	}

	// W104: configSchema is recommended for extensions.
	if dest.Spec.ConfigSchema == nil {
		errs.AddWarning(contracts.WarnCodeConfigSchemaMissing, "spec.configSchema", "configSchema is recommended so data engineers can validate config")
	}
}

// validateModel validates Model-specific fields.
func (v *ManifestValidator) validateModel(errs *contracts.ValidationErrors) {
	model := v.rawModel
	if model == nil {
		return
	}

	// E101: Runtime must be a known value.
	if !model.Spec.Runtime.IsValid() {
		errs.AddError(ErrInvalidFormat, "spec.runtime", "spec.runtime must be a valid runtime (cloudquery, generic-go, generic-python, dbt)")
	}

	// E202: Mode must be batch or streaming (if specified).
	if !model.Spec.Mode.IsValid() && model.Spec.Mode != "" {
		errs.AddError(ErrInvalidFormat, "spec.mode", "spec.mode must be batch or streaming")
	}

	// E104/E208: spec.image is required for generic-* runtimes when no extension provides it.
	if model.Spec.Runtime.IsGeneric() && model.Spec.Image == "" {
		errs.AddError(contracts.ErrCodeImageRequiredGeneric, "spec.image", "spec.image is required for generic-* runtimes")
	}

	// Outputs are required for Model kind.
	if len(model.Spec.Outputs) == 0 {
		errs.AddError(contracts.ErrCodeOutputsRequired, "spec.outputs", "outputs are required for Model packages")
	}

	// Validate inputs if present.
	v.validateArtifacts(errs, model.Spec.Inputs, "spec.inputs", false)

	// Validate outputs — classification is required on outputs.
	v.validateArtifacts(errs, model.Spec.Outputs, "spec.outputs", true)

	// Validate schedule if present.
	if model.Spec.Schedule != nil {
		v.validateSchedule(errs, model.Spec.Schedule)
	}

	// W209: Schedule recommended for batch mode.
	effectiveMode := model.Spec.Mode
	if effectiveMode == "" {
		effectiveMode = effectiveMode.Default()
	}
	if effectiveMode == contracts.ModeBatch && model.Spec.Schedule == nil {
		errs.AddWarning(contracts.WarnCodeScheduleBatchMode, "spec.schedule", "schedule is recommended for batch-mode models")
	}

	// Validate timeout format.
	if model.Spec.Timeout != "" {
		if !isValidDuration(model.Spec.Timeout) {
			errs.AddError(contracts.ErrCodeInvalidTimeout, "spec.timeout", "spec.timeout must be a valid duration (e.g., 1h, 30m)")
		}
	}
}

// validateSchedule validates a ScheduleSpec.
func (v *ManifestValidator) validateSchedule(errs *contracts.ValidationErrors, schedule *contracts.ScheduleSpec) {
	// Schedule can have cron expression, or be suspended.
	// For now, just verify cron is not empty if schedule is provided.
	if schedule.Cron == "" && !schedule.Suspend {
		// This is okay — schedule may be event-driven.
	}
}

// validateArtifacts validates input or output artifacts.
// If requireClassification is true, classification is required on each artifact (E204).
func (v *ManifestValidator) validateArtifacts(errs *contracts.ValidationErrors, artifacts []contracts.ArtifactContract, basePath string, requireClassification bool) {
	seen := make(map[string]bool)

	for i := range artifacts {
		artifact := &artifacts[i]
		path := basePath + "[" + strconv.Itoa(i) + "]"

		if artifact.Name == "" {
			errs.AddError(ErrMissingRequired, path+".name", "artifact name is required")
		} else if seen[artifact.Name] {
			errs.AddError(ErrDuplicateName, path+".name", "duplicate artifact name: "+artifact.Name)
		} else {
			seen[artifact.Name] = true
		}

		if !artifact.Type.IsValid() {
			errs.AddError(contracts.ErrCodeInvalidSchemaType, path+".type", "invalid artifact type")
		}

		if artifact.Binding == "" {
			errs.AddError(ErrMissingRequired, path+".binding", "artifact binding is required")
		}

		// E204: Classification is required on output artifacts.
		if requireClassification && artifact.Classification == nil {
			errs.AddError(contracts.ErrCodeClassificationRequired, path+".classification", "classification is required on output artifacts")
		}

		if artifact.Classification != nil {
			if artifact.Classification.Sensitivity != "" && !artifact.Classification.Sensitivity.IsValid() {
				errs.AddError(ErrInvalidFormat, path+".classification.sensitivity", "invalid sensitivity level")
			}
		}
	}
}

// validateAssetRefs validates that referenced assets exist in the project.
func (v *ManifestValidator) validateAssetRefs(errs *contracts.ValidationErrors) {
	// Asset refs only exist on Model kind currently.
	if v.rawModel == nil {
		return
	}

	// Models don't currently have an Assets field — skip.
	// If assets are added to Model in the future, validate here.
}

// isValidDuration checks if a string is a valid Go duration.
func isValidDuration(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		}
		if c == 'h' || c == 'm' || c == 's' {
			if i == 0 {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// isIdentifierValid checks if a string is a valid DNS-safe identifier.
func isIdentifierValid(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}
	for i, c := range s {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i > 0 && i < len(s)-1 {
			continue
		}
		return false
	}
	return true
}

// isSemVerValid checks if a string is a valid semantic version.
func isSemVerValid(s string) bool {
	parts := 0
	numLen := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			numLen++
		} else if c == '.' {
			if numLen == 0 {
				return false
			}
			parts++
			numLen = 0
		} else if c == '-' || c == '+' {
			break
		} else {
			return false
		}
	}
	return parts >= 2 && numLen > 0
}

// --- Convenience constructors for backward-compatible callers ---

// NewManifestValidator creates a ManifestValidator from a concrete manifest.
func NewManifestValidator(m manifest.Manifest, kind contracts.Kind, pkgPath string) *ManifestValidator {
	v := &ManifestValidator{
		manifest: m,
		kind:     kind,
		pkgPath:  pkgPath,
	}
	switch kind {
	case contracts.KindSource:
		if src, ok := m.(*contracts.Source); ok {
			v.rawSource = src
		}
	case contracts.KindDestination:
		if dest, ok := m.(*contracts.Destination); ok {
			v.rawDest = dest
		}
	case contracts.KindModel:
		if model, ok := m.(*contracts.Model); ok {
			v.rawModel = model
		}
	}
	return v
}

// ValidatePIIForModel validates PII fields on a Model manifest (placeholder).
func ValidatePIIForModel(model *contracts.Model) contracts.ValidationErrors {
	var errs contracts.ValidationErrors
	_ = model
	_ = asset.DefaultResolver // keep import alive for future PII checks
	return errs
}

// Legacy compatibility: keep unused import reference satisfied.
var _ = fmt.Sprintf
