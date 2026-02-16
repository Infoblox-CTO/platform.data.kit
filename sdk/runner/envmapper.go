// Package runner provides local execution capabilities for DP pipelines.
package runner

import (
	"fmt"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// BindingToEnvVar converts a binding path to an environment variable name.
// The conversion follows the convention: dots become underscores, all uppercase.
// Example: "input.events.brokers" → "INPUT_EVENTS_BROKERS"
func BindingToEnvVar(bindingPath string) string {
	return strings.ToUpper(strings.ReplaceAll(bindingPath, ".", "_"))
}

// BindingProperty represents a resolved binding property with its env var name.
type BindingProperty struct {
	// EnvVar is the environment variable name
	EnvVar string
	// BindingPath is the original binding path (e.g., "input.events.brokers")
	BindingPath string
	// Value is the resolved value from the bindings
	Value string
}

// MapBindingsToEnvVars takes a parsed manifest and bindings,
// and returns a list of environment variable mappings.
// Only Model manifests have inputs/outputs; other kinds return empty.
func MapBindingsToEnvVars(m manifest.Manifest, kind contracts.Kind, bindings []contracts.Binding) ([]BindingProperty, []string) {
	var result []BindingProperty
	var warnings []string
	seen := make(map[string]string) // envVar -> bindingPath (for collision detection)

	if kind != contracts.KindModel {
		return result, warnings
	}

	model, ok := m.(*contracts.Model)
	if !ok {
		return result, warnings
	}

	// Process inputs
	for _, input := range model.Spec.Inputs {
		if input.Binding == "" {
			continue
		}
		mappings, warn := mapBindingProperties(input.Binding, bindings, seen)
		result = append(result, mappings...)
		warnings = append(warnings, warn...)
	}

	// Process outputs
	for _, output := range model.Spec.Outputs {
		if output.Binding == "" {
			continue
		}
		mappings, warn := mapBindingProperties(output.Binding, bindings, seen)
		result = append(result, mappings...)
		warnings = append(warnings, warn...)
	}

	return result, warnings
}

// mapBindingProperties maps all properties from a binding to environment variables.
func mapBindingProperties(bindingName string, bindings []contracts.Binding, seen map[string]string) ([]BindingProperty, []string) {
	var result []BindingProperty
	var warnings []string

	// Find the binding by name
	var binding *contracts.Binding
	for i := range bindings {
		if bindings[i].Name == bindingName {
			binding = &bindings[i]
			break
		}
	}

	if binding == nil {
		return nil, []string{fmt.Sprintf("binding '%s' not found", bindingName)}
	}

	// Map properties based on binding type
	props := extractBindingProperties(binding)
	for key, value := range props {
		bindingPath := fmt.Sprintf("%s.%s", bindingName, key)
		envVar := BindingToEnvVar(bindingPath)

		// Check for collision
		if existingPath, exists := seen[envVar]; exists {
			warnings = append(warnings, fmt.Sprintf(
				"env var collision: %s mapped from both '%s' and '%s', using latest",
				envVar, existingPath, bindingPath,
			))
		}
		seen[envVar] = bindingPath

		result = append(result, BindingProperty{
			EnvVar:      envVar,
			BindingPath: bindingPath,
			Value:       value,
		})
	}

	return result, warnings
}

// extractBindingProperties extracts all properties from a typed binding as a map.
func extractBindingProperties(binding *contracts.Binding) map[string]string {
	props := make(map[string]string)

	switch binding.Type {
	case contracts.BindingTypeS3Prefix:
		if binding.S3 != nil {
			if binding.S3.Bucket != "" {
				props["bucket"] = binding.S3.Bucket
			}
			if binding.S3.Prefix != "" {
				props["prefix"] = binding.S3.Prefix
			}
			if binding.S3.Region != "" {
				props["region"] = binding.S3.Region
			}
			if binding.S3.Endpoint != "" {
				props["endpoint"] = binding.S3.Endpoint
			}
			if binding.S3.Format != "" {
				props["format"] = binding.S3.Format
			}
		}

	case contracts.BindingTypeKafkaTopic:
		if binding.Kafka != nil {
			if binding.Kafka.Topic != "" {
				props["topic"] = binding.Kafka.Topic
			}
			if len(binding.Kafka.Brokers) > 0 {
				props["brokers"] = strings.Join(binding.Kafka.Brokers, ",")
			}
			if binding.Kafka.ConsumerGroup != "" {
				props["consumerGroup"] = binding.Kafka.ConsumerGroup
			}
			if binding.Kafka.SchemaRegistry != "" {
				props["schemaRegistry"] = binding.Kafka.SchemaRegistry
			}
			if binding.Kafka.SecurityProtocol != "" {
				props["securityProtocol"] = binding.Kafka.SecurityProtocol
			}
		}

	case contracts.BindingTypePostgresTable:
		if binding.Postgres != nil {
			if binding.Postgres.Host != "" {
				props["host"] = binding.Postgres.Host
			}
			if binding.Postgres.Port > 0 {
				props["port"] = fmt.Sprintf("%d", binding.Postgres.Port)
			}
			if binding.Postgres.Database != "" {
				props["database"] = binding.Postgres.Database
			}
			if binding.Postgres.Schema != "" {
				props["schema"] = binding.Postgres.Schema
			}
			if binding.Postgres.Table != "" {
				props["table"] = binding.Postgres.Table
			}
			if binding.Postgres.SSLMode != "" {
				props["sslMode"] = binding.Postgres.SSLMode
			}
		}
	}

	return props
}

// EnvVarsFromManifest extracts explicit environment variables from a manifest.
// Only Model manifests have user-defined env vars; other kinds return empty.
func EnvVarsFromManifest(m manifest.Manifest, kind contracts.Kind) map[string]string {
	result := make(map[string]string)

	if kind != contracts.KindModel {
		return result
	}

	model, ok := m.(*contracts.Model)
	if !ok {
		return result
	}

	for _, env := range model.Spec.Env {
		if env.Value != "" {
			result[env.Name] = env.Value
		}
		// Note: valueFrom handling would require additional resolution
	}

	return result
}

// MergeEnvVars merges binding-derived env vars with explicit env vars.
// Explicit env vars (from runtime.env) take precedence over auto-mapped bindings.
func MergeEnvVars(bindingEnvs []BindingProperty, explicitEnvs map[string]string) map[string]string {
	result := make(map[string]string)

	// First, add binding-derived env vars
	for _, bp := range bindingEnvs {
		result[bp.EnvVar] = bp.Value
	}

	// Then, add explicit env vars (these override bindings)
	for k, v := range explicitEnvs {
		result[k] = v
	}

	return result
}
