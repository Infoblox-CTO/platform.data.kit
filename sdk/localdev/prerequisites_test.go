package localdev

import (
	"context"
	"testing"
	"time"
)

func TestNewPrerequisiteChecker(t *testing.T) {
	tests := []struct {
		name              string
		runtime           RuntimeType
		expectedAnyOfKeys [][]string // At least one key from each group must be present
	}{
		{
			name:    "compose runtime requires container runtime and docker-compose",
			runtime: RuntimeCompose,
			// Either docker OR rancher must be present, plus docker-compose
			expectedAnyOfKeys: [][]string{{"docker", "rancher"}, {"docker-compose"}},
		},
		{
			name:    "k3d runtime requires container runtime, k3d, kubectl, and helm",
			runtime: RuntimeK3d,
			// Either docker OR rancher must be present, plus k3d, kubectl, and helm
			expectedAnyOfKeys: [][]string{{"docker", "rancher"}, {"k3d"}, {"kubectl"}, {"helm"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewPrerequisiteChecker(tt.runtime)

			if checker == nil {
				t.Fatal("NewPrerequisiteChecker returned nil")
			}

			for _, group := range tt.expectedAnyOfKeys {
				found := false
				for _, key := range group {
					if _, ok := checker.requiredTools[key]; ok {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("requiredTools missing one of %v", group)
				}
			}
		})
	}
}

func TestPrerequisiteChecker_Check(t *testing.T) {
	// This test verifies that Check returns results for all tools
	checker := NewPrerequisiteChecker(RuntimeCompose)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results := checker.Check(ctx)

	if len(results) == 0 {
		t.Error("Check returned no results")
	}

	// Verify each result has a tool name
	for _, r := range results {
		if r.Tool == "" {
			t.Error("result has empty Tool name")
		}
	}
}

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "docker version output",
			input:  "Docker version 24.0.5, build ced0996",
			expect: "24.0.5,",
		},
		{
			name:   "kubectl version output",
			input:  "kubectl version v1.28.0",
			expect: "v1.28.0",
		},
		{
			name:   "k3d version output",
			input:  "k3d version v5.6.0\nk3s version v1.27.4-k3s1 (default)",
			expect: "v5.6.0",
		},
		{
			name:   "simple version",
			input:  "1.2.3",
			expect: "1.2.3",
		},
		{
			name:   "empty input",
			input:  "",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersion(tt.input)
			if result != tt.expect {
				t.Errorf("extractVersion(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestPrerequisiteResult_Struct(t *testing.T) {
	result := PrerequisiteResult{
		Tool:      "docker",
		Available: true,
		Version:   "24.0.5",
		Error:     "",
	}

	if result.Tool != "docker" {
		t.Errorf("Tool = %q, want 'docker'", result.Tool)
	}
	if !result.Available {
		t.Error("Available = false, want true")
	}
	if result.Version != "24.0.5" {
		t.Errorf("Version = %q, want '24.0.5'", result.Version)
	}
}

func TestContainerRuntime_Constants(t *testing.T) {
	// Verify container runtime constants are defined correctly
	if ContainerRuntimeDocker != "docker" {
		t.Errorf("ContainerRuntimeDocker = %q, want 'docker'", ContainerRuntimeDocker)
	}
	if ContainerRuntimeRancher != "rancher" {
		t.Errorf("ContainerRuntimeRancher = %q, want 'rancher'", ContainerRuntimeRancher)
	}
	if ContainerRuntimeNone != "none" {
		t.Errorf("ContainerRuntimeNone = %q, want 'none'", ContainerRuntimeNone)
	}
}

func TestPrerequisiteChecker_GetContainerRuntime(t *testing.T) {
	checker := NewPrerequisiteChecker(RuntimeK3d)

	runtime := checker.GetContainerRuntime()

	// Should return one of the valid runtime types
	switch runtime {
	case ContainerRuntimeDocker, ContainerRuntimeRancher, ContainerRuntimeNone:
		// Valid
	default:
		t.Errorf("GetContainerRuntime() = %q, want one of docker/rancher/none", runtime)
	}
}

func TestGetContainerRuntimeName(t *testing.T) {
	name := GetContainerRuntimeName()

	// Should return a non-empty user-friendly name
	if name == "" {
		t.Error("GetContainerRuntimeName() returned empty string")
	}

	// Should be one of the expected names
	switch name {
	case "Docker", "Rancher Desktop":
		// Valid
	default:
		t.Errorf("GetContainerRuntimeName() = %q, want 'Docker' or 'Rancher Desktop'", name)
	}
}
