package output

import (
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats output as YAML.
type YAMLFormatter struct{}

// Format formats data as YAML.
func (f *YAMLFormatter) Format(w io.Writer, data any) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// FormatTable formats tabular data as a YAML array of objects.
func (f *YAMLFormatter) FormatTable(w io.Writer, headers []string, rows [][]string) error {
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

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(result)
}
