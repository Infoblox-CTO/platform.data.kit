package contracts

import (
	"testing"
)

func TestCloudQueryRole_IsValid(t *testing.T) {
	tests := []struct {
		name string
		role CloudQueryRole
		want bool
	}{
		{name: "source is valid", role: CloudQueryRoleSource, want: true},
		{name: "destination is valid", role: CloudQueryRoleDestination, want: true},
		{name: "empty is invalid", role: "", want: false},
		{name: "unknown is invalid", role: "transform", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.want {
				t.Errorf("CloudQueryRole(%q).IsValid() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestCloudQueryRole_IsSupported(t *testing.T) {
	tests := []struct {
		name string
		role CloudQueryRole
		want bool
	}{
		{name: "source is supported", role: CloudQueryRoleSource, want: true},
		{name: "destination is not supported", role: CloudQueryRoleDestination, want: false},
		{name: "empty is not supported", role: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsSupported(); got != tt.want {
				t.Errorf("CloudQueryRole(%q).IsSupported() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestCloudQuerySpec_Default(t *testing.T) {
	tests := []struct {
		name            string
		spec            CloudQuerySpec
		wantGRPCPort    int
		wantConcurrency int
	}{
		{
			name:            "all zeros get defaults",
			spec:            CloudQuerySpec{Role: CloudQueryRoleSource},
			wantGRPCPort:    7777,
			wantConcurrency: 10000,
		},
		{
			name:            "explicit values preserved",
			spec:            CloudQuerySpec{Role: CloudQueryRoleSource, GRPCPort: 8888, Concurrency: 500},
			wantGRPCPort:    8888,
			wantConcurrency: 500,
		},
		{
			name:            "partial defaults - only port",
			spec:            CloudQuerySpec{Role: CloudQueryRoleSource, Concurrency: 200},
			wantGRPCPort:    7777,
			wantConcurrency: 200,
		},
		{
			name:            "partial defaults - only concurrency",
			spec:            CloudQuerySpec{Role: CloudQueryRoleSource, GRPCPort: 9999},
			wantGRPCPort:    9999,
			wantConcurrency: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.spec.Default()
			if tt.spec.GRPCPort != tt.wantGRPCPort {
				t.Errorf("GRPCPort = %v, want %v", tt.spec.GRPCPort, tt.wantGRPCPort)
			}
			if tt.spec.Concurrency != tt.wantConcurrency {
				t.Errorf("Concurrency = %v, want %v", tt.spec.Concurrency, tt.wantConcurrency)
			}
		})
	}
}

func TestCloudQueryRole_Constants(t *testing.T) {
	if got := string(CloudQueryRoleSource); got != "source" {
		t.Errorf("CloudQueryRoleSource = %v, want %v", got, "source")
	}
	if got := string(CloudQueryRoleDestination); got != "destination" {
		t.Errorf("CloudQueryRoleDestination = %v, want %v", got, "destination")
	}
}

func TestCloudQuerySpec_Tables(t *testing.T) {
	spec := CloudQuerySpec{
		Role:   CloudQueryRoleSource,
		Tables: []string{"users", "orders", "products"},
	}
	if got := len(spec.Tables); got != 3 {
		t.Errorf("len(Tables) = %v, want 3", got)
	}
	if spec.Tables[0] != "users" {
		t.Errorf("Tables[0] = %v, want %v", spec.Tables[0], "users")
	}
}

func TestDataPackageSpec_CloudQueryField(t *testing.T) {
	// Verify the CloudQuery field is accessible on DataPackageSpec
	spec := DataPackageSpec{
		Type: PackageTypeCloudQuery,
		CloudQuery: &CloudQuerySpec{
			Role:     CloudQueryRoleSource,
			GRPCPort: 7777,
		},
	}
	if spec.CloudQuery == nil {
		t.Fatal("CloudQuery field should not be nil")
	}
	if spec.CloudQuery.Role != CloudQueryRoleSource {
		t.Errorf("CloudQuery.Role = %v, want %v", spec.CloudQuery.Role, CloudQueryRoleSource)
	}

	// Pipeline type should have nil CloudQuery
	pipelineSpec := DataPackageSpec{
		Type: PackageTypePipeline,
	}
	if pipelineSpec.CloudQuery != nil {
		t.Error("Pipeline type should have nil CloudQuery")
	}
}
