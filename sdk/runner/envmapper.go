// Package runner provides local execution capabilities for DP pipelines.
package runner

import (
	"github.com/Infoblox-CTO/platform.data.kit/contracts"
	"github.com/Infoblox-CTO/platform.data.kit/sdk/manifest"
)

// EnvVarsFromManifest extracts explicit environment variables from a manifest.
// Only Transform manifests have user-defined env vars; other kinds return empty.
func EnvVarsFromManifest(m manifest.Manifest, kind contracts.Kind) map[string]string {
	result := make(map[string]string)

	if kind != contracts.KindTransform {
		return result
	}

	t, ok := m.(*contracts.Transform)
	if !ok {
		return result
	}

	for _, env := range t.Spec.Env {
		if env.Value != "" {
			result[env.Name] = env.Value
		}
	}

	return result
}
