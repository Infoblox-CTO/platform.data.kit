package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/stretchr/testify/assert"
)

func validAsset() *contracts.AssetManifest {
	return &contracts.AssetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "Asset",
		Metadata: contracts.AssetMetadata{
			Name: "aws-security",
		},
		Spec: contracts.AssetSpec{
			Store: "my-s3",
			Table: "security_findings",
		},
	}
}

func TestValidateAsset_Valid(t *testing.T) {
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), validAsset())
	assert.Empty(t, []*contracts.ValidationError(errs), "expected no errors for a valid asset")
}

func TestValidateAsset_Nil(t *testing.T) {
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), nil)
	assert.True(t, hasErrorCode(errs, ErrAssetRequired))
}

func TestValidateAsset_MissingAPIVersion(t *testing.T) {
	a := validAsset()
	a.APIVersion = ""
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrAssetRequired))
}

func TestValidateAsset_MissingKind(t *testing.T) {
	a := validAsset()
	a.Kind = ""
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrAssetRequired))
}

func TestValidateAsset_WrongKind(t *testing.T) {
	a := validAsset()
	a.Kind = "Store"
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateAsset_MissingName(t *testing.T) {
	a := validAsset()
	a.Metadata.Name = ""
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrAssetRequired))
}

func TestValidateAsset_InvalidName(t *testing.T) {
	a := validAsset()
	a.Metadata.Name = "AB"
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateAsset_MissingStore(t *testing.T) {
	a := validAsset()
	a.Spec.Store = ""
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrAssetRequired))
}

func TestValidateAsset_NoLocator_Warning(t *testing.T) {
	a := validAsset()
	a.Spec.Table = ""
	a.Spec.Prefix = ""
	a.Spec.Topic = ""
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	// Should produce a warning, not an error
	found := false
	for _, e := range errs {
		if e.Code == ErrAssetRequired && e.Severity == contracts.SeverityWarning {
			found = true
		}
	}
	assert.True(t, found, "expected a warning about missing locator")
}

func TestValidateAsset_InvalidClassification(t *testing.T) {
	a := validAsset()
	a.Spec.Classification = "top-secret"
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateAsset_ValidClassification(t *testing.T) {
	for _, c := range []string{"public", "internal", "confidential", "restricted"} {
		a := validAsset()
		a.Spec.Classification = c
		v := NewOfflineAssetValidator()
		errs := v.ValidateAsset(context.Background(), a)
		assert.Empty(t, []*contracts.ValidationError(errs), "expected no errors for classification=%s", c)
	}
}

func TestValidateAsset_DuplicateSchemaFieldName(t *testing.T) {
	a := validAsset()
	a.Spec.Schema = []contracts.SchemaField{
		{Name: "id", Type: "integer"},
		{Name: "id", Type: "string"},
	}
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateAsset_SchemaFieldMissingName(t *testing.T) {
	a := validAsset()
	a.Spec.Schema = []contracts.SchemaField{
		{Name: "", Type: "integer"},
	}
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrAssetRequired))
}

func TestValidateAsset_SchemaFieldMissingType(t *testing.T) {
	a := validAsset()
	a.Spec.Schema = []contracts.SchemaField{
		{Name: "id", Type: ""},
	}
	v := NewOfflineAssetValidator()
	errs := v.ValidateAsset(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrAssetRequired))
}
