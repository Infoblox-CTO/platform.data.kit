package output

import (
	"io"
)

// Format represents the output format.
type Format string

// Output format constants.
const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// Formatter is the interface for formatting output.
type Formatter interface {
	Format(w io.Writer, data any) error
	FormatTable(w io.Writer, headers []string, rows [][]string) error
}

// NewFormatter creates a formatter for the given format.
func NewFormatter(format Format) Formatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatYAML:
		return &YAMLFormatter{}
	default:
		return &TableFormatter{}
	}
}

// ParseFormat parses a format string into a Format.
func ParseFormat(s string) Format {
	switch s {
	case "json":
		return FormatJSON
	case "yaml":
		return FormatYAML
	default:
		return FormatTable
	}
}
