package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// OutputFormat is the format for command output.
type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
	FormatPlain OutputFormat = "plain"
)

// ParseOutputFormat validates and returns an OutputFormat.
func ParseOutputFormat(s string) (OutputFormat, error) {
	switch OutputFormat(s) {
	case FormatTable, FormatJSON, FormatCSV, FormatPlain:
		return OutputFormat(s), nil
	default:
		return "", fmt.Errorf("unknown output format %q: must be table, json, csv, or plain", s)
	}
}

// TableWriter wraps tabwriter for aligned table output.
func NewTableWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// WriteJSON writes v as pretty-printed JSON to w.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteCSVRow writes a single CSV row to w.
func WriteCSVRow(w io.Writer, fields []string) {
	escaped := make([]string, len(fields))
	for i, f := range fields {
		if strings.ContainsAny(f, ",\"\n") {
			escaped[i] = `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
		} else {
			escaped[i] = f
		}
	}
	fmt.Fprintln(w, strings.Join(escaped, ","))
}
