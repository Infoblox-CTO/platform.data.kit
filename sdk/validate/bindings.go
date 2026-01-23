package validate

import (
	"context"
	"os"
	"strconv"

	"github.com/Infoblox-CTO/data.platform.kit/contracts"
	"github.com/Infoblox-CTO/data.platform.kit/sdk/manifest"
)

// BindingsValidator validates bindings.yaml files.
type BindingsValidator struct {
	bindings     []contracts.Binding
	bindingsPath string
}

// NewBindingsValidator creates a validator for bindings.
func NewBindingsValidator(bindings []contracts.Binding, path string) *BindingsValidator {
	return &BindingsValidator{
		bindings:     bindings,
		bindingsPath: path,
	}
}

// NewBindingsValidatorFromFile creates a validator from a bindings.yaml file.
func NewBindingsValidatorFromFile(path string) (*BindingsValidator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := manifest.NewParser()
	bindings, err := parser.ParseBindings(data)
	if err != nil {
		return nil, err
	}

	return &BindingsValidator{
		bindings:     bindings,
		bindingsPath: path,
	}, nil
}

// Name returns the validator name.
func (v *BindingsValidator) Name() string {
	return "bindings"
}

// Validate validates the bindings.
func (v *BindingsValidator) Validate(ctx context.Context) contracts.ValidationErrors {
	var errs contracts.ValidationErrors

	if len(v.bindings) == 0 {
		return errs
	}

	seenNames := make(map[string]bool)

	for i, binding := range v.bindings {
		v.validateBinding(&errs, &binding, i, seenNames)
	}

	return errs
}

// validateBinding validates a single binding.
func (v *BindingsValidator) validateBinding(errs *contracts.ValidationErrors, binding *contracts.Binding, index int, seenNames map[string]bool) {
	basePath := "bindings[" + strconv.Itoa(index) + "]"

	if err := validateRequired(basePath+".name", binding.Name); err != nil {
		errs.Add(err)
	} else {
		if seenNames[binding.Name] {
			errs.AddError(ErrDuplicateName, basePath+".name", "duplicate binding name: "+binding.Name)
		}
		seenNames[binding.Name] = true
	}

	validTypes := []contracts.BindingType{
		contracts.BindingTypeS3Prefix,
		contracts.BindingTypeKafkaTopic,
		contracts.BindingTypePostgresTable,
	}
	if err := validateEnum(basePath+".type", binding.Type, validTypes); err != nil {
		errs.Add(err)
	}

	switch binding.Type {
	case contracts.BindingTypeS3Prefix:
		v.validateS3Binding(errs, binding, basePath)
	case contracts.BindingTypeKafkaTopic:
		v.validateKafkaBinding(errs, binding, basePath)
	case contracts.BindingTypePostgresTable:
		v.validatePostgresBinding(errs, binding, basePath)
	}
}

// validateS3Binding validates S3-specific binding configuration.
func (v *BindingsValidator) validateS3Binding(errs *contracts.ValidationErrors, binding *contracts.Binding, basePath string) {
	s3 := binding.S3

	if s3 == nil {
		errs.AddError(ErrMissingRequired, basePath+".s3", "s3 configuration required for s3-prefix binding type")
		return
	}

	if err := validateRequired(basePath+".s3.bucket", s3.Bucket); err != nil {
		errs.Add(err)
	}
}

// validateKafkaBinding validates Kafka-specific binding configuration.
func (v *BindingsValidator) validateKafkaBinding(errs *contracts.ValidationErrors, binding *contracts.Binding, basePath string) {
	kafka := binding.Kafka

	if kafka == nil {
		errs.AddError(ErrMissingRequired, basePath+".kafka", "kafka configuration required for kafka-topic binding type")
		return
	}

	if err := validateRequired(basePath+".kafka.topic", kafka.Topic); err != nil {
		errs.Add(err)
	}

	if len(kafka.Brokers) == 0 {
		errs.AddError(ErrMissingRequired, basePath+".kafka.brokers", "at least one broker required")
	}
}

// validatePostgresBinding validates PostgreSQL-specific binding configuration.
func (v *BindingsValidator) validatePostgresBinding(errs *contracts.ValidationErrors, binding *contracts.Binding, basePath string) {
	pg := binding.Postgres

	if pg == nil {
		errs.AddError(ErrMissingRequired, basePath+".postgres", "postgres configuration required for postgres-table binding type")
		return
	}

	if err := validateRequired(basePath+".postgres.table", pg.Table); err != nil {
		errs.Add(err)
	}

	if err := validateRequired(basePath+".postgres.database", pg.Database); err != nil {
		errs.Add(err)
	}
}
