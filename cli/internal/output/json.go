package output

import (
	"encoding/json"
	"io"
)

// JSONFormatter formats output as JSON.
type JSONFormatter struct {
	Indent bool
}

// Format formats data as JSON.
func (f *JSONFormatter) Format(w io.Writer, data any) error {
	encoder := json.NewEncoder(w)
	if f.Indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

// FormatTable formats tabular data as a JSON array of objects.
func (f *JSONFormatter) FormatTable(w io.Writer, headers []string, rows [][]string) error {
	var result []map[string]string
	for _, row := range rows {
		obj := make(map[string]string)
		for i, h := range headers {
			if i < len(row) {
				obj[h] = row[i]
			}
		}
		result = append(result, obj)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
