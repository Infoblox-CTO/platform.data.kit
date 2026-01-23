package manifest

import (
	"testing"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
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

func TestDataPackageFromBytes_WithRuntime(t *testing.T) {
	data := []byte(`apiVersion: data.infoblox.com/v1alpha1
kind: DataPackage
metadata:
  name: test-with-runtime
  namespace: data-team
  version: 1.0.0
spec:
  type: pipeline
  description: Test with runtime
  owner: data-team
  runtime:
    image: myregistry.io/myapp:v1.2.3
    timeout: 2h
    retries: 3
    replicas: 2
    env:
      - name: LOG_LEVEL
        value: debug
      - name: DEBUG
        value: "true"
`)

	pkg, err := DataPackageFromBytes(data)
	if err != nil {
		t.Fatalf("DataPackageFromBytes() error = %v", err)
	}

	if pkg.Spec.Runtime == nil {
		t.Fatal("runtime should not be nil")
	}

	if pkg.Spec.Runtime.Image != "myregistry.io/myapp:v1.2.3" {
		t.Errorf("runtime.image = %v, want myregistry.io/myapp:v1.2.3", pkg.Spec.Runtime.Image)
	}

	if pkg.Spec.Runtime.Timeout != "2h" {
		t.Errorf("runtime.timeout = %v, want 2h", pkg.Spec.Runtime.Timeout)
	}

	if pkg.Spec.Runtime.Retries != 3 {
		t.Errorf("runtime.retries = %v, want 3", pkg.Spec.Runtime.Retries)
	}

	if pkg.Spec.Runtime.Replicas != 2 {
		t.Errorf("runtime.replicas = %v, want 2", pkg.Spec.Runtime.Replicas)
	}

	if len(pkg.Spec.Runtime.Env) != 2 {
		t.Errorf("runtime.env count = %v, want 2", len(pkg.Spec.Runtime.Env))
	}

	// Check env var values
	if pkg.Spec.Runtime.Env[0].Name != "LOG_LEVEL" {
		t.Errorf("runtime.env[0].name = %v, want LOG_LEVEL", pkg.Spec.Runtime.Env[0].Name)
	}
	if pkg.Spec.Runtime.Env[0].Value != "debug" {
		t.Errorf("runtime.env[0].value = %v, want debug", pkg.Spec.Runtime.Env[0].Value)
	}
}

func TestValidateDataPackageRuntime(t *testing.T) {
	tests := []struct {
		name    string
		pkg     *contracts.DataPackage
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil package",
			pkg:     nil,
			wantErr: true,
			errMsg:  "DataPackage is nil",
		},
		{
			name: "missing runtime",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Type: contracts.PackageTypePipeline,
				},
			},
			wantErr: true,
			errMsg:  "spec.runtime is required",
		},
		{
			name: "missing image",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Type:    contracts.PackageTypePipeline,
					Runtime: &contracts.RuntimeSpec{},
				},
			},
			wantErr: true,
			errMsg:  "spec.runtime.image is required",
		},
		{
			name: "valid runtime",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Type: contracts.PackageTypePipeline,
					Runtime: &contracts.RuntimeSpec{
						Image: "myimage:v1",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDataPackageRuntime(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDataPackageRuntime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestHasRuntimeSection(t *testing.T) {
	tests := []struct {
		name string
		pkg  *contracts.DataPackage
		want bool
	}{
		{
			name: "nil package",
			pkg:  nil,
			want: false,
		},
		{
			name: "no runtime",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{},
			},
			want: false,
		},
		{
			name: "with runtime",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Runtime: &contracts.RuntimeSpec{Image: "test"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasRuntimeSection(tt.pkg); got != tt.want {
				t.Errorf("HasRuntimeSection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRuntimeImage(t *testing.T) {
	tests := []struct {
		name string
		pkg  *contracts.DataPackage
		want string
	}{
		{
			name: "nil package",
			pkg:  nil,
			want: "",
		},
		{
			name: "no runtime",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{},
			},
			want: "",
		},
		{
			name: "with image",
			pkg: &contracts.DataPackage{
				Spec: contracts.DataPackageSpec{
					Runtime: &contracts.RuntimeSpec{Image: "myimage:v1"},
				},
			},
			want: "myimage:v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRuntimeImage(tt.pkg); got != tt.want {
				t.Errorf("GetRuntimeImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
