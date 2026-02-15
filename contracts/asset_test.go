package contracts

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAssetType_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		atype AssetType
		want  bool
	}{
		{name: "source", atype: AssetTypeSource, want: true},
		{name: "sink", atype: AssetTypeSink, want: true},
		{name: "model-engine", atype: AssetTypeModelEngine, want: true},
		{name: "invalid", atype: AssetType("invalid"), want: false},
		{name: "empty", atype: AssetType(""), want: false},
		{name: "SOURCE uppercase", atype: AssetType("SOURCE"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.atype.IsValid(); got != tt.want {
				t.Errorf("AssetType(%q).IsValid() = %v, want %v", tt.atype, got, tt.want)
			}
		})
	}
}

func TestValidAssetTypes(t *testing.T) {
	types := ValidAssetTypes()
	if len(types) != 3 {
		t.Errorf("ValidAssetTypes() returned %d types, want 3", len(types))
	}
	expected := map[AssetType]bool{
		AssetTypeSource:      true,
		AssetTypeSink:        true,
		AssetTypeModelEngine: true,
	}
	for _, at := range types {
		if !expected[at] {
			t.Errorf("unexpected asset type: %s", at)
		}
	}
}

func TestParseExtensionFQN(t *testing.T) {
	tests := []struct {
		name       string
		fqn        string
		wantVendor string
		wantKind   string
		wantName   string
		wantErr    bool
	}{
		{
			name:       "valid source",
			fqn:        "cloudquery.source.aws",
			wantVendor: "cloudquery",
			wantKind:   "source",
			wantName:   "aws",
		},
		{
			name:       "valid sink",
			fqn:        "infoblox.sink.snowflake",
			wantVendor: "infoblox",
			wantKind:   "sink",
			wantName:   "snowflake",
		},
		{
			name:       "valid model-engine",
			fqn:        "dbt.model-engine.transform",
			wantVendor: "dbt",
			wantKind:   "model-engine",
			wantName:   "transform",
		},
		{
			name:    "too few segments",
			fqn:     "cloudquery.source",
			wantErr: true,
		},
		{
			name:    "single segment",
			fqn:     "cloudquery",
			wantErr: true,
		},
		{
			name:    "empty string",
			fqn:     "",
			wantErr: true,
		},
		{
			name:    "invalid kind",
			fqn:     "cloudquery.database.postgres",
			wantErr: true,
		},
		{
			name:    "empty kind",
			fqn:     "cloudquery..aws",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vendor, kind, name, err := ParseExtensionFQN(tt.fqn)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExtensionFQN(%q) error = %v, wantErr %v", tt.fqn, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if vendor != tt.wantVendor {
				t.Errorf("vendor = %q, want %q", vendor, tt.wantVendor)
			}
			if kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", kind, tt.wantKind)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
		})
	}
}

func TestAssetTypeFromFQN(t *testing.T) {
	tests := []struct {
		name    string
		fqn     string
		want    AssetType
		wantErr bool
	}{
		{name: "source", fqn: "cloudquery.source.aws", want: AssetTypeSource},
		{name: "sink", fqn: "infoblox.sink.snowflake", want: AssetTypeSink},
		{name: "model-engine", fqn: "dbt.model-engine.transform", want: AssetTypeModelEngine},
		{name: "invalid", fqn: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AssetTypeFromFQN(tt.fqn)
			if (err != nil) != tt.wantErr {
				t.Errorf("AssetTypeFromFQN(%q) error = %v, wantErr %v", tt.fqn, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("AssetTypeFromFQN(%q) = %v, want %v", tt.fqn, got, tt.want)
			}
		})
	}
}

func TestAssetTypeDirName(t *testing.T) {
	tests := []struct {
		name  string
		atype AssetType
		want  string
	}{
		{name: "source", atype: AssetTypeSource, want: "sources"},
		{name: "sink", atype: AssetTypeSink, want: "sinks"},
		{name: "model-engine", atype: AssetTypeModelEngine, want: "models"},
		{name: "invalid", atype: AssetType("invalid"), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AssetTypeDirName(tt.atype); got != tt.want {
				t.Errorf("AssetTypeDirName(%q) = %q, want %q", tt.atype, got, tt.want)
			}
		})
	}
}

func TestAssetManifest_YAML(t *testing.T) {
	asset := AssetManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Asset",
		Name:       "aws-security",
		Type:       AssetTypeSource,
		Extension:  "cloudquery.source.aws",
		Version:    "v24.0.2",
		OwnerTeam:  "security-data",
		Binding:    "aws-raw-output",
		Config: map[string]any{
			"accounts": []any{"123456789012"},
			"regions":  []any{"us-east-1"},
			"tables":   []any{"aws_s3_buckets"},
		},
		Labels: map[string]string{
			"domain": "security",
		},
	}

	data, err := yaml.Marshal(&asset)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var roundtrip AssetManifest
	if err := yaml.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if roundtrip.APIVersion != asset.APIVersion {
		t.Errorf("APIVersion = %q, want %q", roundtrip.APIVersion, asset.APIVersion)
	}
	if roundtrip.Kind != asset.Kind {
		t.Errorf("Kind = %q, want %q", roundtrip.Kind, asset.Kind)
	}
	if roundtrip.Name != asset.Name {
		t.Errorf("Name = %q, want %q", roundtrip.Name, asset.Name)
	}
	if roundtrip.Type != asset.Type {
		t.Errorf("Type = %q, want %q", roundtrip.Type, asset.Type)
	}
	if roundtrip.Extension != asset.Extension {
		t.Errorf("Extension = %q, want %q", roundtrip.Extension, asset.Extension)
	}
	if roundtrip.Version != asset.Version {
		t.Errorf("Version = %q, want %q", roundtrip.Version, asset.Version)
	}
	if roundtrip.OwnerTeam != asset.OwnerTeam {
		t.Errorf("OwnerTeam = %q, want %q", roundtrip.OwnerTeam, asset.OwnerTeam)
	}
	if roundtrip.Binding != asset.Binding {
		t.Errorf("Binding = %q, want %q", roundtrip.Binding, asset.Binding)
	}
	if len(roundtrip.Config) == 0 {
		t.Error("Config is empty after roundtrip")
	}
	if roundtrip.Labels["domain"] != "security" {
		t.Errorf("Labels[domain] = %q, want %q", roundtrip.Labels["domain"], "security")
	}
}

func TestAssetManifest_Omitempty(t *testing.T) {
	// Minimal asset — optional fields should NOT appear in YAML output
	asset := AssetManifest{
		APIVersion: "data.infoblox.com/v1alpha1",
		Kind:       "Asset",
		Name:       "minimal",
		Type:       AssetTypeSource,
		Extension:  "cloudquery.source.aws",
		Version:    "v1.0.0",
		OwnerTeam:  "team",
		Config:     map[string]any{"tables": []any{"t1"}},
	}

	data, err := yaml.Marshal(&asset)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	yamlStr := string(data)

	// These optional fields should be omitted
	if contains(yamlStr, "description:") {
		t.Error("description should be omitted when empty")
	}
	if contains(yamlStr, "binding:") {
		t.Error("binding should be omitted when empty")
	}
	if contains(yamlStr, "labels:") {
		t.Error("labels should be omitted when empty")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
