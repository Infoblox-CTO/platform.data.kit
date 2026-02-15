// Package schemas provides embedded extension schemas for offline validation.
package schemas

import "embed"

// EmbeddedSchemas contains all embedded extension schemas.
//
//go:embed *.schema.json
var EmbeddedSchemas embed.FS

// KnownExtensions maps extension FQNs to their embedded schema filenames.
var KnownExtensions = map[string]string{
	"cloudquery.source.aws": "cloudquery.source.schema.json",
}
