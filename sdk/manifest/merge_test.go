package manifest

import (
	"reflect"
	"testing"
)

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]any
		override map[string]any
		want     map[string]any
	}{
		{
			name:     "empty base",
			base:     map[string]any{},
			override: map[string]any{"foo": "bar"},
			want:     map[string]any{"foo": "bar"},
		},
		{
			name:     "empty override",
			base:     map[string]any{"foo": "bar"},
			override: map[string]any{},
			want:     map[string]any{"foo": "bar"},
		},
		{
			name:     "scalar override",
			base:     map[string]any{"foo": "bar"},
			override: map[string]any{"foo": "baz"},
			want:     map[string]any{"foo": "baz"},
		},
		{
			name: "nested map merge",
			base: map[string]any{
				"spec": map[string]any{
					"name":    "my-dp",
					"version": "1.0",
				},
			},
			override: map[string]any{
				"spec": map[string]any{
					"version": "2.0",
					"author":  "test",
				},
			},
			want: map[string]any{
				"spec": map[string]any{
					"name":    "my-dp",
					"version": "2.0",
					"author":  "test",
				},
			},
		},
		{
			name: "array replacement",
			base: map[string]any{
				"items": []any{"a", "b", "c"},
			},
			override: map[string]any{
				"items": []any{"x", "y"},
			},
			want: map[string]any{
				"items": []any{"x", "y"},
			},
		},
		{
			name: "deep nested merge",
			base: map[string]any{
				"spec": map[string]any{
					"runtime": map[string]any{
						"image":   "myimage:v1",
						"timeout": "1h",
					},
				},
			},
			override: map[string]any{
				"spec": map[string]any{
					"runtime": map[string]any{
						"image": "myimage:v2",
					},
				},
			},
			want: map[string]any{
				"spec": map[string]any{
					"runtime": map[string]any{
						"image":   "myimage:v2",
						"timeout": "1h",
					},
				},
			},
		},
		{
			name: "type mismatch - scalar replaces map",
			base: map[string]any{
				"foo": map[string]any{"nested": "value"},
			},
			override: map[string]any{
				"foo": "simple",
			},
			want: map[string]any{
				"foo": "simple",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeepMerge(tt.base, tt.override, DefaultMergeOptions())
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeepMerge() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeepMerge_DoesNotModifyOriginal(t *testing.T) {
	base := map[string]any{
		"spec": map[string]any{
			"name": "original",
		},
	}
	override := map[string]any{
		"spec": map[string]any{
			"name": "modified",
		},
	}

	_ = DeepMerge(base, override, DefaultMergeOptions())

	// Verify base was not modified
	if base["spec"].(map[string]any)["name"] != "original" {
		t.Error("DeepMerge modified the original base map")
	}
}

func TestSetPath(t *testing.T) {
	tests := []struct {
		name    string
		initial map[string]any
		path    string
		value   any
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "simple path",
			initial: map[string]any{},
			path:    "foo",
			value:   "bar",
			want:    map[string]any{"foo": "bar"},
		},
		{
			name:    "nested path",
			initial: map[string]any{},
			path:    "spec.runtime.image",
			value:   "myimage:v1",
			want: map[string]any{
				"spec": map[string]any{
					"runtime": map[string]any{
						"image": "myimage:v1",
					},
				},
			},
		},
		{
			name: "override existing",
			initial: map[string]any{
				"spec": map[string]any{
					"runtime": map[string]any{
						"image":   "old:v1",
						"timeout": "1h",
					},
				},
			},
			path:  "spec.runtime.image",
			value: "new:v2",
			want: map[string]any{
				"spec": map[string]any{
					"runtime": map[string]any{
						"image":   "new:v2",
						"timeout": "1h",
					},
				},
			},
		},
		{
			name:    "array index",
			initial: map[string]any{},
			path:    "spec.env[0].name",
			value:   "LOG_LEVEL",
			want: map[string]any{
				"spec": map[string]any{
					"env": []any{
						map[string]any{"name": "LOG_LEVEL"},
					},
				},
			},
		},
		{
			name: "array index extend",
			initial: map[string]any{
				"items": []any{"a"},
			},
			path:  "items[2]",
			value: "c",
			want: map[string]any{
				"items": []any{"a", nil, "c"},
			},
		},
		{
			name:    "empty path",
			initial: map[string]any{},
			path:    "",
			value:   "bar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetPath(tt.initial, tt.path, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(tt.initial, tt.want) {
				t.Errorf("SetPath() result = %v, want %v", tt.initial, tt.want)
			}
		})
	}
}

func TestGetPath(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		path string
		want any
	}{
		{
			name: "simple",
			m:    map[string]any{"foo": "bar"},
			path: "foo",
			want: "bar",
		},
		{
			name: "nested",
			m: map[string]any{
				"spec": map[string]any{
					"runtime": map[string]any{
						"image": "myimage:v1",
					},
				},
			},
			path: "spec.runtime.image",
			want: "myimage:v1",
		},
		{
			name: "array index",
			m: map[string]any{
				"items": []any{"a", "b", "c"},
			},
			path: "items[1]",
			want: "b",
		},
		{
			name: "missing path",
			m:    map[string]any{"foo": "bar"},
			path: "missing.path",
			want: nil,
		},
		{
			name: "array out of bounds",
			m: map[string]any{
				"items": []any{"a"},
			},
			path: "items[5]",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPath(tt.m, tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSetFlag(t *testing.T) {
	tests := []struct {
		name      string
		flag      string
		wantPath  string
		wantValue any
		wantErr   bool
	}{
		{
			name:      "string value",
			flag:      "spec.runtime.image=myimage:v1",
			wantPath:  "spec.runtime.image",
			wantValue: "myimage:v1",
		},
		{
			name:      "integer value",
			flag:      "spec.replicas=3",
			wantPath:  "spec.replicas",
			wantValue: int64(3),
		},
		{
			name:      "bool true",
			flag:      "spec.enabled=true",
			wantPath:  "spec.enabled",
			wantValue: true,
		},
		{
			name:      "bool false",
			flag:      "spec.enabled=false",
			wantPath:  "spec.enabled",
			wantValue: false,
		},
		{
			name:      "float value",
			flag:      "spec.threshold=0.95",
			wantPath:  "spec.threshold",
			wantValue: 0.95,
		},
		{
			name:      "value with spaces",
			flag:      "spec.name = value with spaces",
			wantPath:  "spec.name",
			wantValue: "value with spaces",
		},
		{
			name:    "missing equals",
			flag:    "spec.runtime.image",
			wantErr: true,
		},
		{
			name:    "empty key",
			flag:    "=value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, value, err := ParseSetFlag(tt.flag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSetFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if path != tt.wantPath {
					t.Errorf("ParseSetFlag() path = %v, want %v", path, tt.wantPath)
				}
				if !reflect.DeepEqual(value, tt.wantValue) {
					t.Errorf("ParseSetFlag() value = %v (%T), want %v (%T)", value, value, tt.wantValue, tt.wantValue)
				}
			}
		})
	}
}

func TestValidateOverridePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{name: "spec.runtime.image", path: "spec.runtime.image", wantErr: false},
		{name: "spec.runtime.timeout", path: "spec.runtime.timeout", wantErr: false},
		{name: "spec.runtime.retries", path: "spec.runtime.retries", wantErr: false},
		{name: "spec.runtime.env", path: "spec.runtime.env", wantErr: false},
		{name: "spec.runtime.env with index", path: "spec.runtime.env[0].value", wantErr: false},
		{name: "spec.runtime.resources.memory", path: "spec.runtime.resources.memory", wantErr: false},
		{name: "spec.runtime.resources.cpu", path: "spec.runtime.resources.cpu", wantErr: false},
		{name: "metadata.name", path: "metadata.name", wantErr: false},
		{name: "metadata.version", path: "metadata.version", wantErr: false},
		{name: "metadata.labels", path: "metadata.labels", wantErr: false},
		{name: "metadata.labels.custom", path: "metadata.labels.custom-label", wantErr: false},
		{name: "spec.inputs", path: "spec.inputs", wantErr: false},
		{name: "spec.outputs", path: "spec.outputs", wantErr: false},

		// Invalid paths
		{name: "typo runtime", path: "runtime.image", wantErr: true},
		{name: "completely invalid", path: "invalid.path.here", wantErr: true},
		{name: "wrong prefix", path: "foo.bar.baz", wantErr: true},
		{name: "close but wrong", path: "spec.runtimes.image", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOverridePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOverridePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateOverridePath_Suggestions(t *testing.T) {
	// Test that suggestions are provided for common typos
	tests := []struct {
		path       string
		suggestion string
	}{
		{path: "runtime", suggestion: "spec.runtime"},
		{path: "image", suggestion: "spec.runtime.image"},
		{path: "timeout", suggestion: "spec.runtime.timeout"},
		{path: "name", suggestion: "metadata.name"},
		{path: "version", suggestion: "metadata.version"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := ValidateOverridePath(tt.path)
			if err == nil {
				t.Error("expected error for invalid path")
				return
			}

			errStr := err.Error()
			if !containsString(errStr, tt.suggestion) {
				t.Errorf("error should suggest %q, got: %s", tt.suggestion, errStr)
			}
		})
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
