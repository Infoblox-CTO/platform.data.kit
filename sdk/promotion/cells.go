// Package promotion provides services for promoting data packages between environments.
package promotion

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// DefaultCell is the default cell name when none is specified.
const DefaultCell = "c0"

// ValuesFilePath returns the path to the per-app values.yaml.
// Layout: envs/{env}/cells/{cell}/apps/{pkg}/values.yaml
func ValuesFilePath(env Environment, cell, pkg string) string {
	return fmt.Sprintf("envs/%s/cells/%s/apps/%s/values.yaml", env, cell, pkg)
}

// GenerateValuesContent produces a minimal values.yaml with appVersion set.
func GenerateValuesContent(appVersion string) (string, error) {
	content := map[string]interface{}{
		"appVersion": appVersion,
	}
	data, err := yaml.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("marshaling values content: %w", err)
	}
	return string(data), nil
}

// ParseAppVersion extracts the appVersion from an existing values.yaml.
// Returns "" if the field is not present or the content is empty.
func ParseAppVersion(content []byte) string {
	if len(content) == 0 {
		return ""
	}
	var values map[string]interface{}
	if err := yaml.Unmarshal(content, &values); err != nil {
		return ""
	}
	v, ok := values["appVersion"]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// MergeAppVersion updates appVersion in an existing values.yaml while preserving
// all other user overrides (resources, replicas, schedule, etc.).
// If existing is empty, a fresh values.yaml with only appVersion is returned.
func MergeAppVersion(existing []byte, newVersion string) ([]byte, error) {
	values := make(map[string]interface{})
	if len(existing) > 0 {
		if err := yaml.Unmarshal(existing, &values); err != nil {
			return nil, fmt.Errorf("parsing existing values.yaml: %w", err)
		}
	}
	values["appVersion"] = newVersion

	data, err := yaml.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshaling updated values: %w", err)
	}
	return data, nil
}

// ResolveCell returns the cell name to use, defaulting to DefaultCell.
func ResolveCell(cell string) string {
	if cell == "" {
		return DefaultCell
	}
	return cell
}
