package validate

import (
	"context"
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/stretchr/testify/assert"
)

func validDataSet() *contracts.DataSetManifest {
	return &contracts.DataSetManifest{
		APIVersion: "datakit.infoblox.dev/v1alpha1",
		Kind:       "DataSet",
		Metadata: contracts.DataSetMetadata{
			Name: "aws-security",
		},
		Spec: contracts.DataSetSpec{
			Store: "my-s3",
			Table: "security_findings",
		},
	}
}

func TestValidateDataSet_Valid(t *testing.T) {
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), validDataSet())
	assert.Empty(t, []*contracts.ValidationError(errs), "expected no errors for a valid dataset")
}

func TestValidateDataSet_Nil(t *testing.T) {
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), nil)
	assert.True(t, hasErrorCode(errs, ErrDataSetRequired))
}

func TestValidateDataSet_MissingAPIVersion(t *testing.T) {
	a := validDataSet()
	a.APIVersion = ""
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrDataSetRequired))
}

func TestValidateDataSet_MissingKind(t *testing.T) {
	a := validDataSet()
	a.Kind = ""
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrDataSetRequired))
}

func TestValidateDataSet_WrongKind(t *testing.T) {
	a := validDataSet()
	a.Kind = "Store"
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateDataSet_MissingName(t *testing.T) {
	a := validDataSet()
	a.Metadata.Name = ""
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrDataSetRequired))
}

func TestValidateDataSet_InvalidName(t *testing.T) {
	a := validDataSet()
	a.Metadata.Name = "AB"
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateDataSet_MissingStore(t *testing.T) {
	a := validDataSet()
	a.Spec.Store = ""
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrDataSetRequired))
}

func TestValidateDataSet_NoLocator_Warning(t *testing.T) {
	a := validDataSet()
	a.Spec.Table = ""
	a.Spec.Prefix = ""
	a.Spec.Topic = ""
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	// Should produce a warning, not an error
	found := false
	for _, e := range errs {
		if e.Code == ErrDataSetRequired && e.Severity == contracts.SeverityWarning {
			found = true
		}
	}
	assert.True(t, found, "expected a warning about missing locator")
}

func TestValidateDataSet_InvalidClassification(t *testing.T) {
	a := validDataSet()
	a.Spec.Classification = "top-secret"
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateDataSet_ValidClassification(t *testing.T) {
	for _, c := range []string{"public", "internal", "confidential", "restricted"} {
		a := validDataSet()
		a.Spec.Classification = c
		v := NewOfflineDataSetValidator()
		errs := v.ValidateDataSet(context.Background(), a)
		assert.Empty(t, []*contracts.ValidationError(errs), "expected no errors for classification=%s", c)
	}
}

func TestValidateDataSet_DuplicateSchemaFieldName(t *testing.T) {
	a := validDataSet()
	a.Spec.Schema = []contracts.SchemaField{
		{Name: "id", Type: "integer"},
		{Name: "id", Type: "string"},
	}
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrInvalidFormat))
}

func TestValidateDataSet_SchemaFieldMissingName(t *testing.T) {
	a := validDataSet()
	a.Spec.Schema = []contracts.SchemaField{
		{Name: "", Type: "integer"},
	}
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrDataSetRequired))
}

func TestValidateDataSet_SchemaFieldMissingType(t *testing.T) {
	a := validDataSet()
	a.Spec.Schema = []contracts.SchemaField{
		{Name: "id", Type: ""},
	}
	v := NewOfflineDataSetValidator()
	errs := v.ValidateDataSet(context.Background(), a)
	assert.True(t, hasErrorCode(errs, ErrDataSetRequired))
}
