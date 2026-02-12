package validate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAggregateValidator(t *testing.T) {
	v := NewAggregateValidator("/path/to/pkg")

	if v == nil {
		t.Fatal("validator should not be nil")
	}
	if v.Name() != "aggregate" {
		t.Errorf("Name() = %s, want aggregate", v.Name())
	}
}

func TestAggregateValidator_WithContext(t *testing.T) {
	v := NewAggregateValidator("/path/to/pkg")

	ctx := &ValidationContext{
		PackageDir:  "/custom/path",
		StrictMode:  true,
		ValidatePII: false,
	}

	v2 := v.WithContext(ctx)

	if v2 != v {
		t.Error("WithContext should return same validator")
	}
	if v.vctx != ctx {
		t.Error("vctx should be updated")
	}
}

func TestAggregateValidator_Validate_DirectoryNotFound(t *testing.T) {
	v := NewAggregateValidator("/nonexistent/path")

	result := v.Validate(context.Background())

	if result.Valid {
		t.Error("expected invalid result for nonexistent directory")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestAggregateValidator_Validate_ValidPackage(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test package
  owner: data-team
  runtime:
    image: myimage:v1
  outputs:
    - name: output-data
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644)
	if err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	v := NewAggregateValidator(tmpDir)
	result := v.Validate(context.Background())

	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestAggregateValidator_Validate_MissingDpYaml(t *testing.T) {
	tmpDir := t.TempDir()

	v := NewAggregateValidator(tmpDir)
	result := v.Validate(context.Background())

	if result.Valid {
		t.Error("expected invalid when dp.yaml is missing")
	}

	hasError := false
	for _, err := range result.Errors {
		if err.Field == "dp.yaml" {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("expected error about missing dp.yaml")
	}
}

func TestAggregateValidator_Validate_InvalidDpYaml(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `not: valid: yaml: here
  - broken indentation
apiVersion: data.infoblox.com/v1alpha1
`
	err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644)
	if err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	v := NewAggregateValidator(tmpDir)
	result := v.Validate(context.Background())

	if result.Valid {
		t.Error("expected invalid for malformed dp.yaml")
	}
}

func TestAggregateValidator_Validate_WithPipeline(t *testing.T) {
	tmpDir := t.TempDir()

	dpContent := `apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pkg
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test pipeline package
  owner: data-team
  runtime:
    image: myorg/pipeline:latest
  outputs:
    - name: output-data
      type: s3-prefix
      binding: output-bucket
      classification:
        sensitivity: public
        pii: false
`
	err := os.WriteFile(filepath.Join(tmpDir, "dp.yaml"), []byte(dpContent), 0644)
	if err != nil {
		t.Fatalf("failed to write dp.yaml: %v", err)
	}

	v := NewAggregateValidator(tmpDir)
	result := v.Validate(context.Background())

	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}
