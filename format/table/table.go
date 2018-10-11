package table

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/simplesurance/baur/format"
)

// Formatter converts Rows into an ASCII table format with space separated
// columns
type Formatter struct {
	out       io.Writer
	tabWriter *tabwriter.Writer
}

// New returns a new tabwriter
func New(headers []string, out io.Writer, writeHeader bool) *Formatter {
	f := Formatter{
		out:       out,
		tabWriter: tabwriter.NewWriter(out, 0, 0, 8, ' ', 0),
	}

	if writeHeader {
		_ = f.writeHeader(headers)
	}

	return &f
}

func (f *Formatter) writeHeader(headers []string) error {
	var header string

	for i, h := range headers {
		header += h

		if i+1 < len(headers) {
			header += "\t"
		}
	}

	_, err := fmt.Fprintln(f.tabWriter, header)
	return err
}

// WriteRow writes a row to the tabwriter buffer
func (f *Formatter) WriteRow(row *format.Row) error {
	var rowStr string

	for i, col := range row.Data {
		rowStr += fmt.Sprintf("%s", col)

		if i+1 < len(row.Data) {
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
