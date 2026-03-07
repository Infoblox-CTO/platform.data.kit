// Package manifest provides parsing and manipulation for DataKit manifests.
package manifest

import (
	"fmt"
	"strconv"
	"strings"
)

// MergeOptions configures the merge behavior.
type MergeOptions struct {
	// ReplaceArrays when true replaces arrays entirely instead of merging.
	// Default is true (following Helm convention).
	ReplaceArrays bool
}

// DefaultMergeOptions returns the default merge options.
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		ReplaceArrays: true, // Helm-style: arrays are replaced, not merged
	}
}

// DeepMerge merges override into base following Helm-style merge rules:
// - Scalars: override replaces base
// - Maps: recursive merge
// - Arrays: override replaces base (no array merge)
//
// Returns a new map; original maps are not modified.
func DeepMerge(base, override map[string]any, opts MergeOptions) map[string]any {
	result := make(map[string]any)

	// Copy base values
	for k, v := range base {
		result[k] = deepCopy(v)
	}

	// Merge override values
	for k, overrideVal := range override {
		baseVal, exists := result[k]
		if !exists {
			// Key doesn't exist in base, just set it
			result[k] = deepCopy(overrideVal)
			continue
		}

		// Both exist - determine merge strategy based on types
		baseMap, baseIsMap := baseVal.(map[string]any)
		overrideMap, overrideIsMap := overrideVal.(map[string]any)

		if baseIsMap && overrideIsMap {
			// Both are maps - recursive merge
			result[k] = DeepMerge(baseMap, overrideMap, opts)
		} else {
			// Scalar, array, or type mismatch - override replaces
			result[k] = deepCopy(overrideVal)
		}
	}

	return result
}

// deepCopy creates a deep copy of a value.
func deepCopy(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = deepCopy(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = deepCopy(v)
		}
		return result
	default:
		// Scalars are immutable, return as-is
		return v
	}
}

// SetPath sets a value at the given dot-notation path.
// Example: SetPath(m, "spec.runtime.image", "myimage:v1")
// Creates intermediate maps as needed.
//
// Supports:
// - Dot notation: "spec.runtime.image"
// - Array index: "spec.runtime.env[0].value"
//
// Returns error if path traversal fails (e.g., setting child of scalar).
func SetPath(m map[string]any, path string, value any) error {
	parts := parsePath(path)
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	return setPathRecursive(m, parts, value)
}

// pathPart represents a single segment of a path.
type pathPart struct {
	key   string
	index int // -1 if not an array access
	isArr bool
}

// parsePath parses a dot-notation path into parts.
// "spec.runtime.env[0].value" -> [{key:"spec"}, {key:"runtime"}, {key:"env", index:0, isArr:true}, {key:"value"}]
func parsePath(path string) []pathPart {
	var parts []pathPart
	segments := strings.Split(path, ".")

	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// Check for array index: key[0]
		if idx := strings.Index(seg, "["); idx != -1 {
			key := seg[:idx]
			idxStr := strings.TrimSuffix(seg[idx+1:], "]")
			if n, err := strconv.Atoi(idxStr); err == nil {
				parts = append(parts, pathPart{key: key, index: n, isArr: true})
				continue
			}
		}

		parts = append(parts, pathPart{key: seg, index: -1, isArr: false})
	}

	return parts
}

// setPathRecursive sets the value at the given path parts.
func setPathRecursive(m map[string]any, parts []pathPart, value any) error {
	if len(parts) == 0 {
		return fmt.Errorf("empty path parts")
	}

	part := parts[0]
	isLast := len(parts) == 1

	if part.isArr {
		// Array access
		arr, ok := m[part.key].([]any)
		if !ok {
			// Create array if doesn't exist
			arr = make([]any, part.index+1)
			m[part.key] = arr
		}

		// Extend array if needed
		for len(arr) <= part.index {
			arr = append(arr, nil)
		}
		m[part.key] = arr

		if isLast {
			arr[part.index] = value
		} else {
			// Need to recurse into array element
			elem, ok := arr[part.index].(map[string]any)
			if !ok {
				elem = make(map[string]any)
				arr[part.index] = elem
			}
			return setPathRecursive(elem, parts[1:], value)
		}
	} else {
		// Map access
		if isLast {
			m[part.key] = value
		} else {
			// Need to recurse
			child, ok := m[part.key].(map[string]any)
			if !ok {
				child = make(map[string]any)
				m[part.key] = child
			}
			return setPathRecursive(child, parts[1:], value)
		}
	}

	return nil
}

// GetPath gets a value at the given dot-notation path.
// Returns nil if path doesn't exist.
func GetPath(m map[string]any, path string) any {
	parts := parsePath(path)
	if len(parts) == 0 {
		return nil
	}

	return getPathRecursive(m, parts)
}

// getPathRecursive gets the value at the given path parts.
func getPathRecursive(m map[string]any, parts []pathPart) any {
	if len(parts) == 0 || m == nil {
		return nil
	}

	part := parts[0]
	isLast := len(parts) == 1

	if part.isArr {
		arr, ok := m[part.key].([]any)
		if !ok || part.index >= len(arr) {
			return nil
		}
		if isLast {
			return arr[part.index]
		}
		child, ok := arr[part.index].(map[string]any)
		if !ok {
			return nil
		}
		return getPathRecursive(child, parts[1:])
	}

	val, exists := m[part.key]
	if !exists {
		return nil
	}
	if isLast {
		return val
	}
	child, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	return getPathRecursive(child, parts[1:])
}

// ParseSetFlag parses a --set flag value (key=value format).
// Returns the key path and value.
func ParseSetFlag(flag string) (path string, value any, err error) {
	idx := strings.Index(flag, "=")
	if idx == -1 {
		return "", nil, fmt.Errorf("invalid --set format, expected key=value: %s", flag)
	}

	path = strings.TrimSpace(flag[:idx])
	rawValue := strings.TrimSpace(flag[idx+1:])

	if path == "" {
		return "", nil, fmt.Errorf("empty key in --set: %s", flag)
	}

	// Try to parse value as number or bool
	value = parseValue(rawValue)

	return path, value, nil
}

// parseValue attempts to parse a string as a typed value.
func parseValue(s string) any {
	// Try bool
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	// Try int
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}

	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Return as string
	return s
}

// ValidOverridePaths defines the allowed top-level paths for --set overrides.
// This provides a safety net against typos and invalid paths.
var ValidOverridePaths = []string{
	"apiVersion",
	"kind",
	"metadata",
	"metadata.name",
	"metadata.namespace",
	"metadata.version",
	"metadata.labels",
	"metadata.annotations",
	"spec.runtime",
	"spec.mode",
	"spec.description",
	"spec.owner",
	"spec.image",
	"spec.command",
	"spec.timeout",
	"spec.retries",
	"spec.replicas",
	"spec.env",
	"spec.inputs",
	"spec.outputs",
	"spec.config",
	"spec.source",
	"spec.destination",
	"spec.provides",
	"spec.accepts",
	"spec.schedule",
	"spec.resources",
	"spec.resources.cpu",
	"spec.resources.memory",
}

// ValidPrefixes defines the allowed prefixes for nested path extensions.
// Only these prefixes allow arbitrary nested paths.
var ValidPrefixes = []string{
	"metadata.labels.",
	"metadata.annotations.",
	"spec.inputs.",
	"spec.inputs[",
	"spec.outputs.",
	"spec.outputs[",
	"spec.env.",
	"spec.env[",
	"spec.config.",
	"spec.provides.",
	"spec.accepts.",
	"spec.source.",
	"spec.destination.",
	"spec.schedule.",
	"spec.resources.",
}

// ValidateOverridePath checks if a path is valid for overriding.
// Returns an error if the path is not recognized.
func ValidateOverridePath(path string) error {
	// Strip array indices for validation of base path
	cleanPath := stripArrayIndices(path)

	// Check for exact match first
	for _, valid := range ValidOverridePaths {
		if cleanPath == valid {
			return nil
		}
	}

	// Check if path starts with a valid prefix that allows nesting
	for _, prefix := range ValidPrefixes {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}

	// Build suggestion list for common typos
	suggestion := suggestPath(cleanPath)
	if suggestion != "" {
		return fmt.Errorf("invalid override path: %s (did you mean: %s?)", path, suggestion)
	}

	return fmt.Errorf("invalid override path: %s (valid prefixes: spec.runtime, spec.mode, spec.inputs, spec.outputs, spec.config, metadata)", path)
}

// stripArrayIndices removes array indices from a path.
// Example: "spec.env[0].value" -> "spec.env.value"
func stripArrayIndices(path string) string {
	result := strings.Builder{}
	inBracket := false

	for _, c := range path {
		if c == '[' {
			inBracket = true
			continue
		}
		if c == ']' {
			inBracket = false
			continue
		}
		if !inBracket {
			result.WriteRune(c)
		}
	}

	return result.String()
}

// suggestPath suggests a correct path for common typos.
func suggestPath(path string) string {
	commonTypos := map[string]string{
		"runtime":          "spec.runtime",
		"image":            "spec.image",
		"timeout":          "spec.timeout",
		"retries":          "spec.retries",
		"env":              "spec.env",
		"resources":        "spec.resources",
		"resources.memory": "spec.resources.memory",
		"resources.cpu":    "spec.resources.cpu",
		"name":             "metadata.name",
		"version":          "metadata.version",
		"labels":           "metadata.labels",
		"mode":             "spec.mode",
		"config":           "spec.config",
	}

	if suggestion, ok := commonTypos[path]; ok {
		return suggestion
	}

	// Check for partial match at end
	for typo, correct := range commonTypos {
		if strings.HasSuffix(path, "."+typo) || path == typo {
			return correct
		}
	}

	return ""
}
