package table

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Formatter converts Rows into an ASCII table format with space separated
// columns
type Formatter struct {
	out       io.Writer
	tabWriter *tabwriter.Writer
}

// New returns a new tabwriter, if headers is not empty it's written as first
// row to the output
func New(headers []string, out io.Writer) *Formatter {
	f := Formatter{
		out:       out,
		tabWriter: tabwriter.NewWriter(out, 0, 0, 8, ' ', 0),
	}

	if len(headers) > 0 {
		_ = f.writeHeader(headers)
	}

	return &f
}

func (f *Formatter) writeHeader(headers []string) error {
	header := strings.Join(headers, "\t")

	_, err := fmt.Fprintln(f.tabWriter, header)

	return err
}

// WriteRow writes a row to the tabwriter buffer
func (f *Formatter) WriteRow(row ...any) error {
	var rowStr string

	for i, col := range row {
		if col != nil {
			rowStr += fmt.Sprintf("%v", col)
		}

		if i+1 < len(row) {
			rowStr += "\t"
		}
	}

	_, err := fmt.Fprintln(f.tabWriter, rowStr)
	return err
}

// Flush flushes the tabwriter buffer, should be called after all rows were
// written, otherwise the column width might be incorrect. See tabwriter.Flush()
// documentation.
func (f *Formatter) Flush() error {
	return f.tabWriter.Flush()
}
