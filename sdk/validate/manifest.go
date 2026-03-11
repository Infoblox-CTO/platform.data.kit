package validate

import (
	"context"
	"fmt"
	"os"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// ManifestValidator validates dk.yaml manifests for all supported kinds
// (Connector, Store, DataSet, DataSetGroup, Transform).
type ManifestValidator struct {
	manifest        manifest.Manifest
	kind            contracts.Kind
	pkgPath         string
	rawConnector    *contracts.Connector
	rawStore        *contracts.Store
	rawDataSet      *contracts.DataSetManifest
	rawDataSetGroup *contracts.DataSetGroupManifest
	rawTransform    *contracts.Transform
}

// NewManifestValidatorFromFile creates a validator from a dk.yaml file.
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
	case contracts.KindConnector:
		v.rawConnector = m.(*contracts.Connector)
	case contracts.KindStore:
		v.rawStore = m.(*contracts.Store)
	case contracts.KindDataSet:
		v.rawDataSet = m.(*contracts.DataSetManifest)
	case contracts.KindDataSetGroup:
		v.rawDataSetGroup = m.(*contracts.DataSetGroupManifest)
	case contracts.KindTransform:
		v.rawTransform = m.(*contracts.Transform)
	}

	return v, nil
}

// Name returns the validator name.
func (v *ManifestValidator) Name() string { return "manifest" }

// Kind returns the detected manifest kind.
func (v *ManifestValidator) Kind() contracts.Kind { return v.kind }

// Manifest returns the parsed manifest.
func (v *ManifestValidator) Manifest() manifest.Manifest { return v.manifest }

// Connector returns the parsed Connector (nil if kind is not Connector).
func (v *ManifestValidator) Connector() *contracts.Connector { return v.rawConnector }

// Store returns the parsed Store (nil if kind is not Store).
func (v *ManifestValidator) Store() *contracts.Store { return v.rawStore }

// DataSet returns the parsed DataSet (nil if kind is not DataSet).
func (v *ManifestValidator) DataSet() *contracts.DataSetManifest { return v.rawDataSet }

// DataSetGroup returns the parsed DataSetGroup (nil if kind is not DataSetGroup).
func (v *ManifestValidator) DataSetGroup() *contracts.DataSetGroupManifest { return v.rawDataSetGroup }

// Transform returns the parsed Transform (nil if kind is not Transform).
func (v *ManifestValidator) Transform() *contracts.Transform { return v.rawTransform }

// Validate validates the manifest.
func (v *ManifestValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if v.manifest == nil {
		errs.AddError(ErrMissingRequired, "", "manifest is nil")
		return errs
	}

	v.validateCommonFields(&errs)

	switch v.kind {
	case contracts.KindConnector:
		v.validateConnector(&errs)
	case contracts.KindStore:
		v.validateStore(&errs)
	case contracts.KindDataSet:
		v.validateDataSet(&errs)
	case contracts.KindDataSetGroup:
		v.validateDataSetGroup(&errs)
	case contracts.KindTransform:
		v.validateTransform(&errs)
	}

	return errs
}

// validateCommonFields checks fields common to all kinds.
func (v *ManifestValidator) validateCommonFields(errs *contracts.ValidationErrors) {
	m := v.manifest

	// Kind is already validated by the parser — but check it's valid.
	if !m.GetKind().IsValid() {
		errs.AddError(ErrInvalidFormat, "kind", "kind must be one of: Connector, Store, DataSet, DataSetGroup, Transform")
	}

	// Metadata — name is required for all kinds.
	if m.GetName() == "" {
		errs.AddError(ErrMissingRequired, "metadata.name", "metadata.name is required")
	} else if !isIdentifierValid(m.GetName()) {
		errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.name", "metadata.name must be DNS-safe")
	}

	// Namespace, version, description requirements depend on kind.
	switch v.kind {
	case contracts.KindConnector:
		if m.GetVersion() != "" && !isSemVerValid(m.GetVersion()) {
			errs.AddError(contracts.ErrCodeInvalidSemVer, "spec.version", "spec.version must be valid SemVer")
		}
	case contracts.KindStore, contracts.KindDataSet, contracts.KindDataSetGroup:
		if m.GetNamespace() != "" && !isIdentifierValid(m.GetNamespace()) {
			errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.namespace", "metadata.namespace must be DNS-safe")
		}
	case contracts.KindTransform:
		if m.GetNamespace() != "" && !isIdentifierValid(m.GetNamespace()) {
			errs.AddError(contracts.ErrCodeNameNotDNSSafe, "metadata.namespace", "metadata.namespace must be DNS-safe")
		}
		if m.GetVersion() != "" && !isSemVerValid(m.GetVersion()) {
			errs.AddError(contracts.ErrCodeInvalidSemVer, "metadata.version", "metadata.version must be valid SemVer")
		}
	}
}

// --- Kind validation methods ---

// validateConnector validates Connector-specific fields.
func (v *ManifestValidator) validateConnector(errs *contracts.ValidationErrors) {
	c := v.rawConnector
	if c == nil {
		return
	}

	if c.Spec.Type == "" {
		errs.AddError(contracts.ErrCodeConnectorTypeRequired, "spec.type", "spec.type is required for Connector")
	}

	if len(c.Spec.Capabilities) == 0 {
		errs.AddError(contracts.ErrCodeConnectorCapabilitiesRequired, "spec.capabilities", "spec.capabilities must list at least one capability (source, destination)")
	} else {
		for i, cap := range c.Spec.Capabilities {
			if cap != "source" && cap != "destination" {
				errs.AddError(ErrInvalidFormat, fmt.Sprintf("spec.capabilities[%d]", i), "capability must be 'source' or 'destination'")
			}
		}
	}

	if c.Spec.Provider != "" && !isIdentifierValid(c.Spec.Provider) {
		errs.AddError(contracts.ErrCodeNameNotDNSSafe, "spec.provider", "spec.provider must be DNS-safe")
	}

	for i, tool := range c.Spec.Tools {
		if tool.Name == "" {
			errs.AddError(ErrMissingRequired, fmt.Sprintf("spec.tools[%d].name", i), "tool name is required")
		}
		if tool.Type != "exec" && tool.Type != "config" {
			errs.AddError(ErrInvalidFormat, fmt.Sprintf("spec.tools[%d].type", i), "tool type must be 'exec' or 'config'")
		}
	}
}

// validateStore validates Store-specific fields.
func (v *ManifestValidator) validateStore(errs *contracts.ValidationErrors) {
	s := v.rawStore
	if s == nil {
		return
	}

	if s.Spec.Connector == "" {
		errs.AddError(contracts.ErrCodeStoreConnectorRequired, "spec.connector", "spec.connector is required — must reference a Connector name")
	}

	if len(s.Spec.Connection) == 0 {
		errs.AddError(contracts.ErrCodeStoreConnectionRequired, "spec.connection", "spec.connection must have at least one entry")
	}

	if s.Spec.ConnectorVersion != "" {
		first := s.Spec.ConnectorVersion[0]
		if !(first >= '0' && first <= '9') && first != '^' && first != '~' && first != '>' && first != '<' && first != '=' {
			errs.AddError(ErrInvalidFormat, "spec.connectorVersion", "spec.connectorVersion must be a valid semver range (e.g., ^1.0.0, >=1.2.0)")
		}
	}

	for key, val := range s.Spec.Secrets {
		if val == "" {
			errs.AddError(contracts.ErrCodeStoreSecretsInvalid, "spec.secrets."+key, "secret value must not be empty")
		}
	}
}

// validateDataSet validates DataSet-specific fields.
func (v *ManifestValidator) validateDataSet(errs *contracts.ValidationErrors) {
	a := v.rawDataSet
	if a == nil {
		return
	}

	if a.Spec.Store == "" {
		errs.AddError(contracts.ErrCodeDataSetStoreRequired, "spec.store", "spec.store is required — must reference a Store name")
	}

	if a.Metadata.Version != "" && !isSemVerValid(a.Metadata.Version) {
		errs.AddError(contracts.ErrCodeInvalidSemVer, "metadata.version", "metadata.version must be valid SemVer")
	}

	if a.Spec.Table == "" && a.Spec.Prefix == "" && a.Spec.Topic == "" {
		errs.AddError(contracts.ErrCodeDataSetLocationRequired, "spec", "at least one of spec.table, spec.prefix, or spec.topic is required")
	}

	// SchemaRef and inline Schema are mutually exclusive.
	if a.Spec.SchemaRef != "" && len(a.Spec.Schema) > 0 {
		errs.AddError(contracts.ErrCodeSchemaRefMutualExclusive, "spec.schemaRef",
			"spec.schemaRef and spec.schema are mutually exclusive — use one or the other")
	}

	// Validate schemaRef format if present.
	if a.Spec.SchemaRef != "" {
		if !isSchemaRefValid(a.Spec.SchemaRef) {
			errs.AddError(contracts.ErrCodeSchemaRefInvalidFormat, "spec.schemaRef",
				"spec.schemaRef must be in the format \"module\" or \"module@constraint\" (e.g., \"users@^1.0.0\")")
		}
	}

	seen := make(map[string]bool)
	for i, field := range a.Spec.Schema {
		path := fmt.Sprintf("spec.schema[%d]", i)
		if field.Name == "" {
			errs.AddError(contracts.ErrCodeDataSetSchemaInvalid, path+".name", "schema field name is required")
		} else if seen[field.Name] {
			errs.AddError(ErrDuplicateName, path+".name", "duplicate schema field name: "+field.Name)
		} else {
			seen[field.Name] = true
		}
		if field.Type == "" {
			errs.AddError(contracts.ErrCodeDataSetSchemaInvalid, path+".type", "schema field type is required")
		}
	}

	if a.Spec.Classification != "" {
		valid := map[string]bool{"public": true, "internal": true, "confidential": true, "restricted": true}
		if !valid[a.Spec.Classification] {
			errs.AddError(ErrInvalidFormat, "spec.classification", "classification must be public, internal, confidential, or restricted")
		}
	}
}

// validateDataSetGroup validates DataSetGroup-specific fields.
func (v *ManifestValidator) validateDataSetGroup(errs *contracts.ValidationErrors) {
	ag := v.rawDataSetGroup
	if ag == nil {
		return
	}

	if ag.Spec.Store == "" {
		errs.AddError(contracts.ErrCodeDataSetGroupStoreRequired, "spec.store", "spec.store is required for DataSetGroup")
	}

	if len(ag.Spec.DataSets) == 0 {
		errs.AddError(contracts.ErrCodeDataSetGroupDataSetsRequired, "spec.datasets", "spec.datasets must list at least one dataset name")
	}
}

// validateTransform validates Transform-specific fields.
func (v *ManifestValidator) validateTransform(errs *contracts.ValidationErrors) {
	tr := v.rawTransform
	if tr == nil {
		return
	}

	if !tr.Spec.Runtime.IsValid() {
		errs.AddError(ErrInvalidFormat, "spec.runtime", "spec.runtime must be a valid runtime (cloudquery, generic-go, generic-python, dbt)")
	}

	if tr.Spec.Mode != "" && !tr.Spec.Mode.IsValid() {
		errs.AddError(ErrInvalidFormat, "spec.mode", "spec.mode must be batch or streaming")
	}

	if len(tr.Spec.Inputs) == 0 {
		errs.AddError(contracts.ErrCodeTransformInputsRequired, "spec.inputs", "at least one input dataset is required")
	}

	if len(tr.Spec.Outputs) == 0 {
		errs.AddError(contracts.ErrCodeTransformOutputsRequired, "spec.outputs", "at least one output dataset is required")
	}

	if tr.Spec.Runtime.IsGeneric() && tr.Spec.Image == "" {
		errs.AddError(contracts.ErrCodeTransformImageRequired, "spec.image", "spec.image is required for generic-* runtimes")
	}
	if tr.Spec.Runtime == contracts.RuntimeDBT && tr.Spec.Image == "" {
		errs.AddError(contracts.ErrCodeTransformImageRequired, "spec.image", "spec.image is required for dbt runtime")
	}

	effectiveMode := tr.Spec.Mode
	if effectiveMode == "" {
		effectiveMode = effectiveMode.Default()
	}
	if effectiveMode == contracts.ModeBatch && tr.Spec.Trigger == nil {
		errs.AddWarning(contracts.WarnCodeTriggerBatchMode, "spec.trigger", "trigger is recommended for batch-mode transforms")
	}

	if tr.Spec.Trigger != nil {
		if !tr.Spec.Trigger.Policy.IsValid() {
			errs.AddError(ErrInvalidFormat, "spec.trigger.policy", "spec.trigger.policy must be schedule, on-change, manual, or composite")
		}
		if tr.Spec.Trigger.Policy == contracts.TriggerPolicySchedule && tr.Spec.Trigger.Schedule == nil {
			errs.AddError(ErrMissingRequired, "spec.trigger.schedule", "spec.trigger.schedule is required when policy is schedule")
		}
		if tr.Spec.Trigger.Policy == contracts.TriggerPolicyComposite {
			if len(tr.Spec.Trigger.Policies) == 0 {
				errs.AddError(ErrMissingRequired, "spec.trigger.policies", "spec.trigger.policies is required when policy is composite")
			}
			for i, p := range tr.Spec.Trigger.Policies {
				if !p.IsValid() || p == contracts.TriggerPolicyComposite {
					errs.AddError(ErrInvalidFormat, fmt.Sprintf("spec.trigger.policies[%d]", i), "sub-policy must be schedule, on-change, or manual")
				}
			}
		}
	}

	if tr.Spec.Timeout != "" {
		if !isValidDuration(tr.Spec.Timeout) {
			errs.AddError(contracts.ErrCodeInvalidTimeout, "spec.timeout", "spec.timeout must be a valid duration (e.g., 1h, 30m)")
		}
	}

	// Validate input/output dataset refs.
	for i, ref := range tr.Spec.Inputs {
		validateDataSetRef(errs, ref, fmt.Sprintf("spec.inputs[%d]", i))
	}
	for i, ref := range tr.Spec.Outputs {
		validateDataSetRef(errs, ref, fmt.Sprintf("spec.outputs[%d]", i))
	}
}

// validateDataSetRef validates that exactly one of dataset or tags is set.
func validateDataSetRef(errs *contracts.ValidationErrors, ref contracts.DataSetRef, path string) {
	hasDataSet := ref.DataSet != ""
	hasTags := len(ref.Tags) > 0
	if !hasDataSet && !hasTags {
		errs.AddError(ErrMissingRequired, path, "either dataset name or tags is required")
	}
	if hasDataSet && hasTags {
		errs.AddError(ErrInvalidFormat, path, "dataset and tags are mutually exclusive — specify one or the other")
	}
	// Version only makes sense with tags.
	if ref.Version != "" && !hasTags {
		errs.AddWarning("W210", path+".version", "version constraint is only used with tag-based resolution")
	}
	// Schema ref on DataSetRef — validate APX module ID format if present.
	if ref.Schema != "" && !isSchemaRefValid(ref.Schema) {
		errs.AddError(contracts.ErrCodeSchemaRefInvalidFormat, path+".schema",
			"schema must be a valid APX module reference (e.g., \"users\" or \"users@^1.0.0\")")
	}
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

// isSchemaRefValid checks if a string is a valid APX schema reference.
// Valid formats: "module-name" or "module-name@constraint" (e.g., "users@^1.0.0").
func isSchemaRefValid(s string) bool {
	if s == "" {
		return false
	}
	// Split on @ — module part must be a valid identifier.
	atIdx := -1
	for i, c := range s {
		if c == '@' {
			atIdx = i
			break
		}
	}
	module := s
	if atIdx >= 0 {
		module = s[:atIdx]
		constraint := s[atIdx+1:]
		if constraint == "" {
			return false // trailing @ with no constraint
		}
	}
	// Module must be non-empty and DNS-safe-like (lowercase, alphanumeric, hyphens).
	if len(module) == 0 {
		return false
	}
	for i, c := range module {
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' && i > 0 && i < len(module)-1 {
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

// --- Convenience constructors ---

// NewManifestValidator creates a ManifestValidator from a concrete manifest.
func NewManifestValidator(m manifest.Manifest, kind contracts.Kind, pkgPath string) *ManifestValidator {
	v := &ManifestValidator{
		manifest: m,
		kind:     kind,
		pkgPath:  pkgPath,
	}
	switch kind {
	case contracts.KindConnector:
		if c, ok := m.(*contracts.Connector); ok {
			v.rawConnector = c
		}
	case contracts.KindStore:
		if s, ok := m.(*contracts.Store); ok {
			v.rawStore = s
		}
	case contracts.KindDataSet:
		if a, ok := m.(*contracts.DataSetManifest); ok {
			v.rawDataSet = a
		}
	case contracts.KindDataSetGroup:
		if ag, ok := m.(*contracts.DataSetGroupManifest); ok {
			v.rawDataSetGroup = ag
		}
	case contracts.KindTransform:
		if tr, ok := m.(*contracts.Transform); ok {
			v.rawTransform = tr
		}
	}
	return v
}
