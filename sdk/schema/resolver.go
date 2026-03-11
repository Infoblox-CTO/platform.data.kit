// Package schema provides schema resolution, lock file management, and
// breaking-change detection for DataSet schemas. It is designed to integrate
// with the APX schema management tool.
package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/Infoblox-CTO/platform.data.kit/contracts"
)

// SchemaResolver resolves APX schema references to concrete schemas.
type SchemaResolver interface {
	// Resolve resolves a schema reference (e.g., "users@^1.0.0") to a
	// concrete resolved schema with fields and checksum.
	Resolve(ctx context.Context, ref string) (*ResolvedSchema, error)

	// Search searches the schema catalog by query string.
	Search(ctx context.Context, query string) ([]SchemaModule, error)

	// CheckBreaking checks for breaking changes between two schema versions.
	CheckBreaking(ctx context.Context, oldRef, newRef string) ([]BreakingChange, error)
}

// ResolvedSchema is the result of resolving a schema reference.
type ResolvedSchema struct {
	// Module is the resolved schema module metadata.
	Module SchemaModule

	// Fields is the schema expressed as DK SchemaField entries.
	Fields []contracts.SchemaField

	// Checksum is the integrity hash of the schema content.
	Checksum string
}

// SchemaModule describes a schema module in the catalog.
// This mirrors the APX catalog.Module structure.
type SchemaModule struct {
	// ID is the module identifier (e.g., "users").
	ID string `json:"id" yaml:"id"`

	// Format is the schema format (parquet, avro, json, protobuf, openapi).
	Format string `json:"format" yaml:"format"`

	// Domain is the business domain (e.g., "identity", "billing").
	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`

	// Version is the semantic version (e.g., "1.2.3").
	Version string `json:"version" yaml:"version"`

	// Lifecycle is the module lifecycle stage (draft, active, deprecated).
	Lifecycle string `json:"lifecycle,omitempty" yaml:"lifecycle,omitempty"`

	// Repo is the canonical repository URL.
	Repo string `json:"repo,omitempty" yaml:"repo,omitempty"`

	// Tags are searchable metadata tags.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Owners are the module owners (email or team name).
	Owners []string `json:"owners,omitempty" yaml:"owners,omitempty"`
}

// BreakingChange describes a single breaking change between schema versions.
type BreakingChange struct {
	// Field is the affected field path.
	Field string `json:"field"`

	// Type describes the kind of breaking change (e.g., "removed", "type_changed").
	Type string `json:"type"`

	// Message is a human-readable description.
	Message string `json:"message"`
}

// ParseSchemaRef parses a schema reference string into module name and constraint.
// Examples: "users" → ("users", ""), "users@^1.0.0" → ("users", "^1.0.0").
func ParseSchemaRef(ref string) (module, constraint string) {
	if idx := strings.IndexByte(ref, '@'); idx >= 0 {
		return ref[:idx], ref[idx+1:]
	}
	return ref, ""
}

// FormatSchemaRef formats a module name and constraint into a schema reference.
func FormatSchemaRef(module, constraint string) string {
	if constraint == "" {
		return module
	}
	return fmt.Sprintf("%s@%s", module, constraint)
}
