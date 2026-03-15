package promotion

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValuesFilePath(t *testing.T) {
	tests := []struct {
		env  Environment
		cell string
		pkg  string
		want string
	}{
		{EnvDev, "c0", "kafka-s3-pipeline", "gitops/envs/dev/cells/c0/apps/kafka-s3-pipeline/values.yaml"},
		{EnvProd, "canary", "my-pkg", "gitops/envs/prod/cells/canary/apps/my-pkg/values.yaml"},
		{EnvInt, "group1", "etl", "gitops/envs/int/cells/group1/apps/etl/values.yaml"},
	}

	for _, tt := range tests {
		got := ValuesFilePath(tt.env, tt.cell, tt.pkg)
		if got != tt.want {
			t.Errorf("ValuesFilePath(%q, %q, %q) = %q, want %q", tt.env, tt.cell, tt.pkg, got, tt.want)
		}
	}
}

func TestGenerateValuesContent(t *testing.T) {
	content, err := GenerateValuesContent("v1.0.0")
	if err != nil {
		t.Fatalf("GenerateValuesContent() error: %v", err)
	}
	if content == "" {
		t.Fatal("GenerateValuesContent() returned empty content")
	}

	// Parse back and verify
	version := ParseAppVersion([]byte(content))
	if version != "v1.0.0" {
		t.Errorf("round-trip: ParseAppVersion() = %q, want %q", version, "v1.0.0")
	}
}

func TestParseAppVersion(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{"valid", []byte("appVersion: v1.0.0\n"), "v1.0.0"},
		{"empty", []byte(""), ""},
		{"nil", nil, ""},
		{"no appVersion", []byte("replicas: 3\n"), ""},
		{"with overrides", []byte("appVersion: v2.0.0\nreplicas: 5\nresources:\n  cpu: 200m\n"), "v2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAppVersion(tt.content)
			if got != tt.want {
				t.Errorf("ParseAppVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeAppVersion(t *testing.T) {
	t.Run("empty existing", func(t *testing.T) {
		result, err := MergeAppVersion(nil, "v1.0.0")
		if err != nil {
			t.Fatalf("MergeAppVersion() error: %v", err)
		}
		version := ParseAppVersion(result)
		if version != "v1.0.0" {
			t.Errorf("version = %q, want %q", version, "v1.0.0")
		}
	})

	t.Run("preserve overrides", func(t *testing.T) {
		existing := []byte("appVersion: v0.9.0\nreplicas: 5\nresources:\n  cpu: 200m\n")
		result, err := MergeAppVersion(existing, "v1.0.0")
		if err != nil {
			t.Fatalf("MergeAppVersion() error: %v", err)
		}

		// Verify version updated
		version := ParseAppVersion(result)
		if version != "v1.0.0" {
			t.Errorf("version = %q, want %q", version, "v1.0.0")
		}

		// Verify overrides preserved
		var values map[string]interface{}
		if err := yaml.Unmarshal(result, &values); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if replicas, ok := values["replicas"]; !ok || replicas != 5 {
			t.Errorf("replicas not preserved, got %v", values["replicas"])
		}
	})

	t.Run("update existing version", func(t *testing.T) {
		existing := []byte("appVersion: v1.0.0\n")
		result, err := MergeAppVersion(existing, "v2.0.0")
		if err != nil {
			t.Fatalf("MergeAppVersion() error: %v", err)
		}
		version := ParseAppVersion(result)
		if version != "v2.0.0" {
			t.Errorf("version = %q, want %q", version, "v2.0.0")
		}
	})
}

func TestResolveCell(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "c0"},
		{"canary", "canary"},
		{"group1", "group1"},
	}
	for _, tt := range tests {
		got := ResolveCell(tt.input)
		if got != tt.want {
			t.Errorf("ResolveCell(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
