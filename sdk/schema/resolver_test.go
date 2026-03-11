package schema

import "testing"

func TestParseSchemaRef(t *testing.T) {
	tests := []struct {
		ref        string
		wantModule string
		wantConst  string
	}{
		{"users", "users", ""},
		{"users@^1.0.0", "users", "^1.0.0"},
		{"my-dataset@>=2.0.0", "my-dataset", ">=2.0.0"},
		{"orders@1.2.3", "orders", "1.2.3"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			module, constraint := ParseSchemaRef(tt.ref)
			if module != tt.wantModule {
				t.Errorf("ParseSchemaRef(%q) module = %q, want %q", tt.ref, module, tt.wantModule)
			}
			if constraint != tt.wantConst {
				t.Errorf("ParseSchemaRef(%q) constraint = %q, want %q", tt.ref, constraint, tt.wantConst)
			}
		})
	}
}

func TestFormatSchemaRef(t *testing.T) {
	tests := []struct {
		module     string
		constraint string
		want       string
	}{
		{"users", "", "users"},
		{"users", "^1.0.0", "users@^1.0.0"},
		{"orders", "1.2.3", "orders@1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatSchemaRef(tt.module, tt.constraint)
			if got != tt.want {
				t.Errorf("FormatSchemaRef(%q, %q) = %q, want %q", tt.module, tt.constraint, got, tt.want)
			}
		})
	}
}
