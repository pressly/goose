package goosecli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

const (
	// defaultSeparator is the default separator for table data.
	defaultSeparator = '─'
)

type printer struct {
	tabWriter *tabwriter.Writer
	jsonEnc   *json.Encoder
	w         io.Writer
	separator rune
}

type tableData struct {
	Headers []string
	Rows    [][]string
}

func newPrinter(w io.Writer, separator rune) *printer {
	return &printer{
		tabWriter: tabwriter.NewWriter(w, 0, 0, 3, ' ', tabwriter.TabIndent),
		jsonEnc:   json.NewEncoder(w),
		w:         w,
		separator: separator,
	}
}

// func (p *printer) plain(format string, a ...any) {
// 	fmt.Fprintf(p.w, format, a...)
// }

// Table prints data in tabular format.
func (p *printer) Table(data tableData) error {
	if err := validateTableData(data); err != nil {
		return err
	}
	defer p.tabWriter.Flush()

	// Create format pattern based on number of columns
	fmtPattern := strings.Repeat("%v\t", len(data.Headers)-1) + "%v\n"

	// Print headers
	fmt.Fprintf(p.tabWriter, fmtPattern, toAnySlice(data.Headers)...)

	// Print separator line
	separators := make([]string, len(data.Headers))
	for i := range separators {
		separators[i] = strings.Repeat(string(p.separator), len(data.Headers[i]))
	}
	fmt.Fprintf(p.tabWriter, fmtPattern, toAnySlice(separators)...)

	// Print rows
	for _, row := range data.Rows {
		fmt.Fprintf(p.tabWriter, fmtPattern, toAnySlice(row)...)
	}

	return nil
}

// JSON prints any struct with json tags as JSON.
func (p *printer) JSON(v any) error {
	return p.jsonEnc.Encode(v)
}

func validateTableData(data tableData) error {
	if len(data.Headers) == 0 {
		return fmt.Errorf("headers slice cannot be empty")
	}

	for _, row := range data.Rows {
		if len(row) != len(data.Headers) {
			return fmt.Errorf("each row must have the same number of columns as headers")
		}
	}
	return nil
}

func toAnySlice(s []string) []any {
	interfaces := make([]any, 0, len(s))
	for _, v := range s {
		interfaces = append(interfaces, v)
	}
	return interfaces
}
