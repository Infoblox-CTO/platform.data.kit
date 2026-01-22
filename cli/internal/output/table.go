package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// TableFormatter formats output as a human-readable table.
type TableFormatter struct{}

// Format formats data as a table.
func (f *TableFormatter) Format(w io.Writer, data any) error {
	_, err := fmt.Fprintf(w, "%v\n", data)
	return err
}

// FormatTable formats tabular data with headers and rows.
func (f *TableFormatter) FormatTable(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	if len(headers) > 0 {
		for i, h := range headers {
			headers[i] = strings.ToUpper(h)
		}
		fmt.Fprintln(tw, strings.Join(headers, "\t"))
	}

	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}

	return tw.Flush()
}

// PrintSuccess prints a success message.
func PrintSuccess(w io.Writer, message string) {
	fmt.Fprintf(w, "OK %s\n", message)
}

// PrintError prints an error message.
func PrintError(w io.Writer, message string) {
	fmt.Fprintf(w, "ERR %s\n", message)
}
