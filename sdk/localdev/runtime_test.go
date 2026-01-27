package localdev

import (
	"testing"
)

func TestRuntimeType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		runtime  RuntimeType
		expected string
	}{
		{
			name:     "compose runtime type",
			runtime:  RuntimeCompose,
			expected: "compose",
		},
		{
			name:     "k3d runtime type",
			runtime:  RuntimeK3d,
			expected: "k3d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.runtime) != tt.expected {
				t.Errorf("RuntimeType = %q, want %q", tt.runtime, tt.expected)
			}
		})
	}
}

func TestRuntimeType_String(t *testing.T) {
	compose := RuntimeCompose
	k3d := RuntimeK3d

	if string(compose) != "compose" {
		t.Errorf("RuntimeCompose string = %q, want compose", compose)
	}
	if string(k3d) != "k3d" {
		t.Errorf("RuntimeK3d string = %q, want k3d", k3d)
	}
}

func TestPortForwardStatus_Struct(t *testing.T) {
	status := PortForwardStatus{
		LocalPort:     19092,
		TargetService: "redpanda",
		Active:        true,
		Error:         "",
	}

	if status.LocalPort != 19092 {
		t.Errorf("LocalPort = %d, want 19092", status.LocalPort)
	}
	if status.TargetService != "redpanda" {
		t.Errorf("TargetService = %q, want redpanda", status.TargetService)
	}
	if !status.Active {
		t.Error("Active = false, want true")
	}
}
