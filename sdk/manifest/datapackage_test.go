package manifest

import (
	"testing"

	"github.com/Infoblox-CTO/data-platform/contracts"
)

func TestDataPackageFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		wantName string
		wantType contracts.PackageType
	}{
		{
			name: "valid datapackage",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-dataset
  namespace: data-team
  version: 1.0.0
spec:
  type: dataset
  description: A test dataset
  owner: data-team
`),
			wantErr:  false,
			wantName: "test-dataset",
			wantType: contracts.PackageTypeDataset,
		},
		{
			name: "pipeline type",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-pipeline
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: A test pipeline
  owner: data-team
`),
			wantErr:  false,
			wantName: "test-pipeline",
			wantType: contracts.PackageTypePipeline,
		},
		{
			name: "model type",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-model
  namespace: ml-team
  version: 1.0.0
spec:
  type: model
  description: A test model
  owner: ml-team
`),
			wantErr:  false,
			wantName: "test-model",
			wantType: contracts.PackageTypeModel,
		},
		{
			name: "wrong kind returns error",
			data: []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: Pipeline
metadata:
  name: test
`),
			wantErr: true,
		},
		{
			name:    "empty bytes returns error",
			data:    []byte(""),
			wantErr: true,
		},
		{
			name:    "invalid YAML returns error",
			data:    []byte("invalid: yaml: content: ["),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := DataPackageFromBytes(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataPackageFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if pkg.Metadata.Name != tt.wantName {
					t.Errorf("name = %v, want %v", pkg.Metadata.Name, tt.wantName)
				}
				if pkg.Spec.Type != tt.wantType {
					t.Errorf("type = %v, want %v", pkg.Spec.Type, tt.wantType)
				}
			}
		})
	}
}

func TestDataPackageToBytes(t *testing.T) {
	tests := []struct {
		name    string
		pkg     *contracts.DataPackage
		wantErr bool
	}{
		{
			name: "valid datapackage",
			pkg: &contracts.DataPackage{
				APIVersion: string(contracts.APIVersionV1Alpha1),
				Kind:       "DataPackage",
				Metadata: contracts.PackageMetadata{
					Name:      "test-pkg",
					Namespace: "data-team",
					Version:   "1.0.0",
				},
				Spec: contracts.DataPackageSpec{
					Type:        contracts.PackageTypeDataset,
					Description: "Test package",
					Owner:       "data-team",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty package",
			pkg:     &contracts.DataPackage{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := DataPackageToBytes(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataPackageToBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("DataPackageToBytes() returned empty data")
			}
		})
	}
}

func TestDataPackage_RoundTrip(t *testing.T) {
	original := &contracts.DataPackage{
		APIVersion: string(contracts.APIVersionV1Alpha1),
		Kind:       "DataPackage",
		Metadata: contracts.PackageMetadata{
			Name:      "roundtrip-test",
			Namespace: "data-team",
			Version:   "2.0.0",
			Labels:    map[string]string{"env": "test"},
		},
		Spec: contracts.DataPackageSpec{
			Type:        contracts.PackageTypeDataset,
			Description: "Round trip test",
			Owner:       "data-team",
		},
	}

	// Serialize to YAML
	data, err := DataPackageToBytes(original)
	if err != nil {
		t.Fatalf("DataPackageToBytes() error = %v", err)
	}

	// Parse back
	parsed, err := DataPackageFromBytes(data)
	if err != nil {
		t.Fatalf("DataPackageFromBytes() error = %v", err)
	}

	// Verify fields
	if parsed.Metadata.Name != original.Metadata.Name {
		t.Errorf("name = %v, want %v", parsed.Metadata.Name, original.Metadata.Name)
	}
	if parsed.Metadata.Version != original.Metadata.Version {
		t.Errorf("version = %v, want %v", parsed.Metadata.Version, original.Metadata.Version)
	}
	if parsed.Spec.Type != original.Spec.Type {
		t.Errorf("type = %v, want %v", parsed.Spec.Type, original.Spec.Type)
	}
	if parsed.Metadata.Labels["env"] != "test" {
		t.Errorf("label env = %v, want test", parsed.Metadata.Labels["env"])
	}
}

func TestValidateDataPackageVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{
			name:    "v1alpha1 is valid",
			version: string(contracts.APIVersionV1Alpha1),
			wantErr: false,
		},
		{
			name:    "v1beta1 is valid",
			version: string(contracts.APIVersionV1Beta1),
			wantErr: false,
		},
		{
			name:    "v1 is valid",
			version: string(contracts.APIVersionV1),
			wantErr: false,
		},
		{
			name:    "unsupported version",
			version: "data.infoblox.com/v999",
			wantErr: true,
		},
		{
			name:    "empty version",
			version: "",
			wantErr: true,
		},
		{
			name:    "random string",
			version: "not-a-version",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataPackageVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDataPackageVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDataPackageFromBytes_WithOutputs(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-with-outputs
  namespace: data-team
  version: 1.0.0
spec:
  type: dataset
  description: Test with outputs
  owner: data-team
  outputs:
    - name: customer-data
      type: s3-prefix
      binding: customer-bucket
      classification:
        sensitivity: internal
        pii: true
    - name: events
      type: kafka-topic
      binding: events-topic
      classification:
        sensitivity: public
        pii: false
`)

	pkg, err := DataPackageFromBytes(data)
	if err != nil {
		t.Fatalf("DataPackageFromBytes() error = %v", err)
	}

	if len(pkg.Spec.Outputs) != 2 {
		t.Errorf("outputs count = %v, want 2", len(pkg.Spec.Outputs))
	}

	// Check first output
	if pkg.Spec.Outputs[0].Name != "customer-data" {
		t.Errorf("output[0].name = %v, want customer-data", pkg.Spec.Outputs[0].Name)
	}
	if pkg.Spec.Outputs[0].Type != contracts.ArtifactTypeS3Prefix {
		t.Errorf("output[0].type = %v, want s3-prefix", pkg.Spec.Outputs[0].Type)
	}
	if pkg.Spec.Outputs[0].Classification == nil {
		t.Error("output[0].classification should not be nil")
	} else if !pkg.Spec.Outputs[0].Classification.PII {
		t.Error("output[0].classification.pii should be true")
	}

	// Check second output
	if pkg.Spec.Outputs[1].Name != "events" {
		t.Errorf("output[1].name = %v, want events", pkg.Spec.Outputs[1].Name)
	}
	if pkg.Spec.Outputs[1].Type != contracts.ArtifactTypeKafkaTopic {
		t.Errorf("output[1].type = %v, want kafka-topic", pkg.Spec.Outputs[1].Type)
	}
}

func TestDataPackageFromBytes_WithInputs(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-with-inputs
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test with inputs
  owner: data-team
  inputs:
    - name: source-data
      package: upstream-dataset
      version: 1.0.0
`)

	pkg, err := DataPackageFromBytes(data)
	if err != nil {
		t.Fatalf("DataPackageFromBytes() error = %v", err)
	}

	if len(pkg.Spec.Inputs) != 1 {
		t.Errorf("inputs count = %v, want 1", len(pkg.Spec.Inputs))
	}

	if pkg.Spec.Inputs[0].Name != "source-data" {
		t.Errorf("input[0].name = %v, want source-data", pkg.Spec.Inputs[0].Name)
	}
}
